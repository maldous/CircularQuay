package observability

import (
	"bytes"
	"io"
	"mime"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/maldous/CircularQuay/internal/fault"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
	collectorlog "go.opentelemetry.io/proto/otlp/collector/logs/v1"
	collectormetric "go.opentelemetry.io/proto/otlp/collector/metrics/v1"
	collectortrace "go.opentelemetry.io/proto/otlp/collector/trace/v1"
	commonpb "go.opentelemetry.io/proto/otlp/common/v1"
	resourcepb "go.opentelemetry.io/proto/otlp/resource/v1"
	"google.golang.org/protobuf/proto"
)

type collectorSignal uint8

const (
	collectorTrace collectorSignal = iota
	collectorMetrics
	collectorLogs
)

// The pinned SDK merges ambient resources even when supplied an explicit
// resource. Enforce CQ's declaration at the last shared transmission boundary,
// without changing process environment, global handlers or caller transports.
func collectorResourceClient(client *http.Client, declared *resource.Resource, signal collectorSignal) (*http.Client, error) {
	if err := validateCollectorEnvironment(); err != nil {
		return nil, err
	}
	if client == nil || client.Transport == nil || signal > collectorLogs {
		return nil, fault.New(fault.Invalid, "observability transport")
	}
	approved, err := collectorWireResource(declared)
	if err != nil {
		return nil, err
	}
	isolated := *client
	isolated.Transport = collectorResourceTransport{next: client.Transport, approved: approved, signal: signal}
	return &isolated, nil
}

// Malformed resource syntax makes the SDK send the original input to its global
// error handler before export. Reject those forms with a fixed error first.
// Like the surrounding SDK construction, this assumes the embedding process
// does not concurrently replace its environment with hostile input.
func validateCollectorEnvironment() error {
	value := strings.TrimSpace(os.Getenv("OTEL_RESOURCE_ATTRIBUTES"))
	if value == "" {
		return nil
	}
	for _, pair := range strings.Split(value, ",") {
		_, encoded, found := strings.Cut(pair, "=")
		if !found {
			return fault.New(fault.Invalid, "observability resource environment")
		}
		if _, err := url.PathUnescape(strings.TrimSpace(encoded)); err != nil {
			return fault.New(fault.Invalid, "observability resource environment")
		}
	}
	return nil
}

func collectorWireResource(declared *resource.Resource) (*resourcepb.Resource, error) {
	if declared == nil {
		return nil, fault.New(fault.Invalid, "observability resource")
	}
	result := &resourcepb.Resource{}
	for _, item := range declared.Attributes() {
		if !collectorResourceKey(item.Key) || item.Value.Type() != attribute.STRING {
			return nil, fault.New(fault.Invalid, "observability resource")
		}
		result.Attributes = append(result.Attributes, &commonpb.KeyValue{
			Key: string(item.Key), Value: &commonpb.AnyValue{Value: &commonpb.AnyValue_StringValue{StringValue: item.Value.AsString()}},
		})
	}
	return result, nil
}

func collectorResourceKey(key attribute.Key) bool {
	switch key {
	case "service.name", "service.version", "deployment.environment.name", "service.namespace", "service.instance.id":
		return true
	default:
		return false
	}
}

type collectorResourceTransport struct {
	next     http.RoundTripper
	approved *resourcepb.Resource
	signal   collectorSignal
}

func (t collectorResourceTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	if request == nil || request.Body == nil {
		return nil, telemetryExportError()
	}
	if err := request.Context().Err(); err != nil {
		_ = request.Body.Close()
		return nil, err
	}
	if !collectorProtobufRequest(request) {
		_ = request.Body.Close()
		return nil, telemetryExportError()
	}
	body, err := io.ReadAll(io.LimitReader(request.Body, maxOTLPRequestBytes+1))
	closed := request.Body.Close()
	if err != nil || closed != nil || len(body) > maxOTLPRequestBytes {
		return nil, telemetryExportError()
	}
	encoded, err := rewriteCollectorResource(t.signal, body, t.approved)
	if err != nil || len(encoded) > maxOTLPRequestBytes {
		return nil, telemetryExportError()
	}
	return t.next.RoundTrip(collectorRequestBody(request, encoded))
}

func collectorProtobufRequest(request *http.Request) bool {
	mediaType, _, err := mime.ParseMediaType(request.Header.Get("Content-Type"))
	encoding := request.Header.Get("Content-Encoding")
	return err == nil && mediaType == "application/x-protobuf" && (encoding == "" || encoding == "identity")
}

func collectorRequestBody(original *http.Request, encoded []byte) *http.Request {
	request := original.Clone(original.Context())
	request.Body = io.NopCloser(bytes.NewReader(encoded))
	// SDK replay closures otherwise still contain the unsanitized original.
	request.GetBody = func() (io.ReadCloser, error) { return io.NopCloser(bytes.NewReader(encoded)), nil }
	request.ContentLength = int64(len(encoded))
	request.TransferEncoding = nil
	request.Header.Del("Content-Length")
	return request
}

func rewriteCollectorResource(signal collectorSignal, body []byte, approved *resourcepb.Resource) ([]byte, error) {
	if approved == nil || proto.Size(approved) > maxOTLPRequestBytes {
		return nil, telemetryExportError()
	}
	// Share one immutable copy within this request. A small input can contain
	// many resource groups; copying the declaration for each would amplify it.
	canonical := proto.Clone(approved).(*resourcepb.Resource)
	switch signal {
	case collectorTrace:
		return rewriteTraceResource(body, canonical)
	case collectorMetrics:
		return rewriteMetricResource(body, canonical)
	case collectorLogs:
		return rewriteLogResource(body, canonical)
	default:
		return nil, telemetryExportError()
	}
}

func rewriteTraceResource(body []byte, approved *resourcepb.Resource) ([]byte, error) {
	var request collectortrace.ExportTraceServiceRequest
	if err := proto.Unmarshal(body, &request); err != nil {
		return nil, telemetryExportError()
	}
	for _, item := range request.ResourceSpans {
		item.Resource = approved
		item.SchemaUrl = ""
	}
	return marshalCollectorRequest(&request)
}

func rewriteMetricResource(body []byte, approved *resourcepb.Resource) ([]byte, error) {
	var request collectormetric.ExportMetricsServiceRequest
	if err := proto.Unmarshal(body, &request); err != nil {
		return nil, telemetryExportError()
	}
	for _, item := range request.ResourceMetrics {
		item.Resource = approved
		item.SchemaUrl = ""
	}
	return marshalCollectorRequest(&request)
}

func rewriteLogResource(body []byte, approved *resourcepb.Resource) ([]byte, error) {
	var request collectorlog.ExportLogsServiceRequest
	if err := proto.Unmarshal(body, &request); err != nil {
		return nil, telemetryExportError()
	}
	for _, item := range request.ResourceLogs {
		item.Resource = approved
		item.SchemaUrl = ""
	}
	return marshalCollectorRequest(&request)
}

func marshalCollectorRequest(request proto.Message) ([]byte, error) {
	// Check before allocating encoded output, retaining the existing wire cap.
	if proto.Size(request) > maxOTLPRequestBytes {
		return nil, telemetryExportError()
	}
	return proto.Marshal(request)
}
