package observability

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"testing"

	"go.opentelemetry.io/otel/sdk/resource"
	logspb "go.opentelemetry.io/proto/otlp/collector/logs/v1"
	metricspb "go.opentelemetry.io/proto/otlp/collector/metrics/v1"
	tracepb "go.opentelemetry.io/proto/otlp/collector/trace/v1"
	commonpb "go.opentelemetry.io/proto/otlp/common/v1"
	logv1 "go.opentelemetry.io/proto/otlp/logs/v1"
	metricv1 "go.opentelemetry.io/proto/otlp/metrics/v1"
	resourcepb "go.opentelemetry.io/proto/otlp/resource/v1"
	tracev1 "go.opentelemetry.io/proto/otlp/trace/v1"
	"google.golang.org/protobuf/proto"
)

type resourceRequestInspector struct {
	calls   int
	inspect func(*http.Request)
}

func (c *resourceRequestInspector) RoundTrip(r *http.Request) (*http.Response, error) {
	c.calls++
	if c.inspect != nil {
		c.inspect(r)
	}
	return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(bytes.NewReader(nil))}, nil
}
func canonicalTransportResource(t *testing.T) *resource.Resource {
	t.Helper()
	r, err := telemetryResource(resourcePipelineConfig(), "synthetic-version")
	if err != nil {
		t.Fatal(err)
	}
	return r
}
func syntheticUncontrolledResource() *resourcepb.Resource {
	r := &resourcepb.Resource{Attributes: []*commonpb.KeyValue{{Key: "synthetic.identity", Value: &commonpb.AnyValue{Value: &commonpb.AnyValue_StringValue{StringValue: "SYNTHETIC_PRIVATE"}}}}, DroppedAttributesCount: 7, EntityRefs: []*commonpb.EntityRef{{}}}
	r.ProtoReflect().SetUnknown([]byte{0x98, 0x06, 0x01})
	return r
}
func syntheticResourceScope() *commonpb.InstrumentationScope {
	return &commonpb.InstrumentationScope{Name: "synthetic.scope", Version: "1", Attributes: []*commonpb.KeyValue{{Key: "synthetic.scope.marker", Value: &commonpb.AnyValue{Value: &commonpb.AnyValue_StringValue{StringValue: "keep-scope"}}}}}
}
func resourceTransportMessage(signal collectorSignal) proto.Message {
	switch signal {
	case collectorTrace:
		r := &tracev1.ResourceSpans{Resource: syntheticUncontrolledResource(), SchemaUrl: "synthetic.private.schema", ScopeSpans: []*tracev1.ScopeSpans{{Scope: syntheticResourceScope(), SchemaUrl: "synthetic.scope.schema", Spans: []*tracev1.Span{{Name: "cq.recovery", TraceId: []byte{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}, SpanId: []byte{2, 0, 0, 0, 0, 0, 0, 0}}}}}}
		return &tracepb.ExportTraceServiceRequest{ResourceSpans: []*tracev1.ResourceSpans{r, proto.Clone(r).(*tracev1.ResourceSpans)}}
	case collectorMetrics:
		r := &metricv1.ResourceMetrics{Resource: syntheticUncontrolledResource(), SchemaUrl: "synthetic.private.schema", ScopeMetrics: []*metricv1.ScopeMetrics{{Scope: syntheticResourceScope(), SchemaUrl: "synthetic.scope.schema", Metrics: []*metricv1.Metric{{Name: "cq.operation.total", Description: "keep-metric", Unit: "1", Data: &metricv1.Metric_Sum{Sum: &metricv1.Sum{IsMonotonic: true, AggregationTemporality: metricv1.AggregationTemporality_AGGREGATION_TEMPORALITY_CUMULATIVE, DataPoints: []*metricv1.NumberDataPoint{{Value: &metricv1.NumberDataPoint_AsInt{AsInt: 3}}}}}}}}}}
		return &metricspb.ExportMetricsServiceRequest{ResourceMetrics: []*metricv1.ResourceMetrics{r, proto.Clone(r).(*metricv1.ResourceMetrics)}}
	default:
		r := &logv1.ResourceLogs{Resource: syntheticUncontrolledResource(), SchemaUrl: "synthetic.private.schema", ScopeLogs: []*logv1.ScopeLogs{{Scope: syntheticResourceScope(), SchemaUrl: "synthetic.scope.schema", LogRecords: []*logv1.LogRecord{{EventName: "cq.operation", Body: &commonpb.AnyValue{Value: &commonpb.AnyValue_StringValue{StringValue: "keep-log"}}}}}}}
		return &logspb.ExportLogsServiceRequest{ResourceLogs: []*logv1.ResourceLogs{r, proto.Clone(r).(*logv1.ResourceLogs)}}
	}
}
func resourceProtoBytes(t *testing.T, p proto.Message) []byte {
	t.Helper()
	body, err := proto.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	return body
}
func TestResourceTransportReplacesWholeResourceAndPreservesScope(t *testing.T) {
	for name, signal := range map[string]collectorSignal{"traces": collectorTrace, "metrics": collectorMetrics, "logs": collectorLogs} {
		t.Run(name, func(t *testing.T) {
			original := resourceTransportMessage(signal)
			input := resourceProtoBytes(t, original)
			downstream := &resourceRequestInspector{inspect: func(r *http.Request) { assertResourceTransportBody(t, r, original) }}
			client, err := collectorResourceClient(&http.Client{Transport: downstream}, canonicalTransportResource(t), signal)
			if err != nil {
				t.Fatal(err)
			}
			request := newResourceTransportRequest(t, input)
			response, err := client.Transport.RoundTrip(request)
			if err != nil {
				t.Fatal(err)
			}
			response.Body.Close()
			if downstream.calls != 1 {
				t.Fatal("canonical export was not forwarded exactly once")
			}
		})
	}
}
func newResourceTransportRequest(t *testing.T, body []byte) *http.Request {
	t.Helper()
	r, err := http.NewRequestWithContext(resourcePipelineContext(t), http.MethodPost, "https://synthetic.invalid/v1/signal", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	r.Header.Set("Content-Type", "application/x-protobuf")
	return r
}
func assertResourceTransportBody(t *testing.T, r *http.Request, original proto.Message) {
	t.Helper()
	body, err := io.ReadAll(r.Body)
	if err != nil {
		t.Fatal(err)
	}
	if r.ContentLength != int64(len(body)) || r.GetBody == nil {
		t.Fatal("rewritten request length/replay body inconsistent")
	}
	replay, err := r.GetBody()
	if err != nil {
		t.Fatal(err)
	}
	defer replay.Close()
	other, err := io.ReadAll(replay)
	if err != nil || !bytes.Equal(body, other) {
		t.Fatal("rewritten replay differs from body")
	}
	actual := original.ProtoReflect().New().Interface()
	decodeResourceProto(t, body, actual)
	assertWholeResourceReplacement(t, actual)
	expected := proto.Clone(original)
	copyResourceEnvelopes(expected, actual)
	if !proto.Equal(expected, actual) {
		t.Fatal("resource rewrite changed scope or signal data")
	}
}
func assertWholeResourceReplacement(t *testing.T, p proto.Message) {
	t.Helper()
	switch r := p.(type) {
	case *tracepb.ExportTraceServiceRequest:
		assertTraceResourceContainers(t, r.ResourceSpans)
	case *metricspb.ExportMetricsServiceRequest:
		assertMetricResourceContainers(t, r.ResourceMetrics)
	case *logspb.ExportLogsServiceRequest:
		assertLogResourceContainers(t, r.ResourceLogs)
	}
}
func assertTraceResourceContainers(t *testing.T, rs []*tracev1.ResourceSpans) {
	t.Helper()
	if len(rs) != 2 {
		t.Fatal("trace containers lost")
	}
	for _, v := range rs {
		assertCanonicalWireResource(t, v.Resource, v.SchemaUrl)
		assertNoResourceUnknown(t, v.Resource)
	}
}
func assertMetricResourceContainers(t *testing.T, rs []*metricv1.ResourceMetrics) {
	t.Helper()
	if len(rs) != 2 {
		t.Fatal("metric containers lost")
	}
	for _, v := range rs {
		assertCanonicalWireResource(t, v.Resource, v.SchemaUrl)
		assertNoResourceUnknown(t, v.Resource)
	}
}
func assertLogResourceContainers(t *testing.T, rs []*logv1.ResourceLogs) {
	t.Helper()
	if len(rs) != 2 {
		t.Fatal("log containers lost")
	}
	for _, v := range rs {
		assertCanonicalWireResource(t, v.Resource, v.SchemaUrl)
		assertNoResourceUnknown(t, v.Resource)
	}
}
func assertNoResourceUnknown(t *testing.T, r *resourcepb.Resource) {
	t.Helper()
	if len(r.ProtoReflect().GetUnknown()) != 0 {
		t.Fatal("unknown resource fields survived replacement")
	}
}
func copyResourceEnvelopes(expected, actual proto.Message) {
	switch e := expected.(type) {
	case *tracepb.ExportTraceServiceRequest:
		for i, v := range e.ResourceSpans {
			a := actual.(*tracepb.ExportTraceServiceRequest).ResourceSpans[i]
			v.Resource = a.Resource
			v.SchemaUrl = a.SchemaUrl
		}
	case *metricspb.ExportMetricsServiceRequest:
		for i, v := range e.ResourceMetrics {
			a := actual.(*metricspb.ExportMetricsServiceRequest).ResourceMetrics[i]
			v.Resource = a.Resource
			v.SchemaUrl = a.SchemaUrl
		}
	case *logspb.ExportLogsServiceRequest:
		for i, v := range e.ResourceLogs {
			a := actual.(*logspb.ExportLogsServiceRequest).ResourceLogs[i]
			v.Resource = a.Resource
			v.SchemaUrl = a.SchemaUrl
		}
	}
}

type resourceBrokenBody struct{}

func (resourceBrokenBody) Read([]byte) (int, error) {
	return 0, errors.New("SYNTHETIC_PRIVATE_READ_ERROR")
}
func (resourceBrokenBody) Close() error { return nil }
func TestResourceTransportRejectsInvalidRequestsBeforeForwarding(t *testing.T) {
	for _, name := range []string{"protobuf", "content_type", "encoding", "read", "input_size", "output_size", "canceled"} {
		t.Run(name, func(t *testing.T) {
			downstream := &resourceRequestInspector{}
			client, err := collectorResourceClient(&http.Client{Transport: downstream}, canonicalTransportResource(t), collectorTrace)
			if err != nil {
				t.Fatal(err)
			}
			request := newResourceTransportRequest(t, resourceProtoBytes(t, resourceTransportMessage(collectorTrace)))
			corruptResourceRequest(request, name)
			body := &resourceClosingBody{ReadCloser: request.Body}
			request.Body = body
			_, err = client.Transport.RoundTrip(request)
			assertResourceTransportFailure(t, err, downstream.calls, name == "canceled")
			if !body.closed {
				t.Fatal("rejected input body was not closed")
			}
		})
	}
}
func corruptResourceRequest(r *http.Request, name string) {
	switch name {
	case "protobuf":
		r.Body = io.NopCloser(bytes.NewReader([]byte{0xff}))
	case "content_type":
		r.Header.Set("Content-Type", "application/json")
	case "encoding":
		r.Header.Set("Content-Encoding", "gzip")
	case "read":
		r.Body = resourceBrokenBody{}
	case "input_size":
		r.Body = io.NopCloser(bytes.NewReader(make([]byte, (1<<20)+1)))
		r.ContentLength = (1 << 20) + 1
	case "output_size":
		p := &tracepb.ExportTraceServiceRequest{ResourceSpans: make([]*tracev1.ResourceSpans, 10000)}
		for i := range p.ResourceSpans {
			p.ResourceSpans[i] = &tracev1.ResourceSpans{}
		}
		body, _ := proto.Marshal(p)
		r.Body = io.NopCloser(bytes.NewReader(body))
		r.ContentLength = int64(len(body))
	case "canceled":
		ctx, cancel := context.WithCancel(r.Context())
		cancel()
		*r = *r.WithContext(ctx)
	}
}
func assertResourceTransportFailure(t *testing.T, err error, calls int, canceled bool) {
	t.Helper()
	want := telemetryExportError().Error()
	if canceled {
		want = context.Canceled.Error()
	}
	if err == nil || err.Error() != want || errors.Unwrap(err) != nil || calls != 0 {
		t.Fatalf("invalid request was not safely stopped: err=%v calls=%d", err, calls)
	}
}

type resourceClosingBody struct {
	io.ReadCloser
	closed bool
}

func (b *resourceClosingBody) Close() error { b.closed = true; return b.ReadCloser.Close() }
