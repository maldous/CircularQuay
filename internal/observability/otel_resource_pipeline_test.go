package observability

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/maldous/CircularQuay/internal/fault"
	"github.com/maldous/CircularQuay/internal/model"
	"go.opentelemetry.io/otel/sdk/resource"
	logspb "go.opentelemetry.io/proto/otlp/collector/logs/v1"
	metricspb "go.opentelemetry.io/proto/otlp/collector/metrics/v1"
	tracepb "go.opentelemetry.io/proto/otlp/collector/trace/v1"
	commonpb "go.opentelemetry.io/proto/otlp/common/v1"
	metricv1 "go.opentelemetry.io/proto/otlp/metrics/v1"
	resourcepb "go.opentelemetry.io/proto/otlp/resource/v1"
	"google.golang.org/protobuf/proto"
)

type resourceWireRequest struct {
	path        string
	body        []byte
	contentType string
}
type resourceWireTransport struct{ requests chan resourceWireRequest }

func (c resourceWireTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}
	c.requests <- resourceWireRequest{r.URL.Path, body, r.Header.Get("Content-Type")}
	return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": {"application/x-protobuf"}}, Body: io.NopCloser(bytes.NewReader(nil))}, nil
}
func resourcePipelineConfig() model.Observability {
	enabled, ratio := true, 1.0
	return model.Observability{Enabled: &enabled, ServiceName: "cq", Environment: "test", DiagnosticLevel: "off", Endpoint: "https://synthetic.invalid", TimeoutSeconds: 5, ExportIntervalSeconds: 60, BatchSize: 512, QueueSize: 512,
		ResourceAttributes: &model.ResourceAttributes{ServiceNamespace: "synthetic-suite", ServiceInstanceID: "synthetic-instance"},
		Traces:             &model.TracesSignal{Enabled: &enabled, SampleRatio: &ratio}, Metrics: &model.MetricsSignal{Enabled: &enabled, IntervalSeconds: 3600}, Logs: &model.LogsSignal{Enabled: &enabled}}
}
func resourcePipelineContext(t *testing.T) context.Context {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)
	return ctx
}
func cleanupResourceBackend(t *testing.T, b *otelBackend) {
	t.Helper()
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := b.shutdown(ctx); err != nil {
			t.Errorf("cleanup synthetic backend: %v", err)
		}
	})
}
func configureResourceSignal(ctx context.Context, b *otelBackend, c model.Observability, res *resource.Resource, client *http.Client, signal string) error {
	endpoint := "https://synthetic.invalid/v1/" + signal
	switch signal {
	case "traces":
		return b.configureTraces(ctx, c, res, client, nil, endpoint)
	case "metrics":
		return b.configureMetrics(ctx, c, res, client, nil, endpoint)
	default:
		return b.configureLogs(ctx, c, res, client, nil, endpoint)
	}
}
func TestResourcePipelinesExportOnlyCanonicalResource(t *testing.T) {
	for name, ambient := range map[string]string{"clean": "", "ambient": "synthetic.identity=SYNTHETIC%20PRIVATE,service.name=SYNTHETIC_COLLISION,synthetic.path=%2Fsynthetic%2Fprivate"} {
		t.Run(name, func(t *testing.T) { runCanonicalResourcePipeline(t, ambient) })
	}
}
func runCanonicalResourcePipeline(t *testing.T, ambient string) {
	t.Helper()
	t.Setenv("OTEL_RESOURCE_ATTRIBUTES", ambient)
	service, protocol := "", "http/protobuf"
	if ambient != "" {
		service = "SYNTHETIC_AMBIENT_SERVICE"
		protocol = "http/json"
	}
	t.Setenv("OTEL_SERVICE_NAME", service)
	t.Setenv("OTEL_EXPORTER_OTLP_TRACES_PROTOCOL", protocol)
	ctx := resourcePipelineContext(t)
	config := resourcePipelineConfig()
	res, err := telemetryResource(config, "synthetic-version")
	if err != nil {
		t.Fatal(err)
	}
	capture := resourceWireTransport{requests: make(chan resourceWireRequest, 16)}
	client := &http.Client{Transport: capture}
	b := &otelBackend{exportState: &exportHealth{}}
	cleanupResourceBackend(t, b)
	for _, signal := range []string{"traces", "metrics", "logs"} {
		if err := configureResourceSignal(ctx, b, config, res, client, signal); err != nil {
			t.Fatalf("configure %s: %v", signal, err)
		}
	}
	_, finish := Start(WithRuntime(ctx, &Runtime{backend: b}), Operation{Phase: PhaseRecovery, ProviderKind: "openbao"})
	finish(OutcomeFallback)
	flushResourceSignals(t, ctx, b)
	if len(capture.requests) != 3 {
		t.Fatalf("signal requests=%d, want 3", len(capture.requests))
	}
	assertThreeResourceRequests(t, capture)
}
func flushResourceSignals(t *testing.T, ctx context.Context, b *otelBackend) {
	t.Helper()
	for _, flush := range []func(context.Context) error{b.traces.ForceFlush, b.metrics.ForceFlush, b.logs.ForceFlush} {
		if err := flush(ctx); err != nil {
			t.Fatal(err)
		}
	}
}
func assertThreeResourceRequests(t *testing.T, capture resourceWireTransport) {
	t.Helper()
	seen := map[string]bool{}
	for range 3 {
		request := <-capture.requests
		if seen[request.path] {
			t.Fatal("duplicate signal request")
		}
		seen[request.path] = true
		t.Run(request.path, func(t *testing.T) { assertCanonicalResourceWire(t, request) })
	}
}
func assertCanonicalResourceWire(t *testing.T, request resourceWireRequest) {
	t.Helper()
	if request.contentType != "application/x-protobuf" {
		t.Fatal("configured signal did not use protobuf")
	}
	switch request.path {
	case "/v1/traces":
		assertCanonicalTraceWire(t, request.body)
	case "/v1/logs":
		assertCanonicalLogWire(t, request.body)
	case "/v1/metrics":
		assertCanonicalMetricWire(t, request.body)
	default:
		t.Fatal("unexpected signal path")
	}
}
func assertCanonicalTraceWire(t *testing.T, body []byte) {
	t.Helper()
	var p tracepb.ExportTraceServiceRequest
	decodeResourceProto(t, body, &p)
	if len(p.ResourceSpans) != 1 || len(p.ResourceSpans[0].ScopeSpans) != 1 {
		t.Fatal("trace scope lost")
	}
	r := p.ResourceSpans[0]
	assertCanonicalWireResource(t, r.Resource, r.SchemaUrl)
	s := r.ScopeSpans[0]
	if s.Scope.GetName() != instrumentationScope || len(s.Spans) != 1 || s.Spans[0].Name != "cq.recovery" {
		t.Fatal("trace operation lost")
	}
	assertSignalLabels(t, s.Spans[0].Attributes, false)
}
func assertCanonicalLogWire(t *testing.T, body []byte) {
	t.Helper()
	var p logspb.ExportLogsServiceRequest
	decodeResourceProto(t, body, &p)
	if len(p.ResourceLogs) != 1 || len(p.ResourceLogs[0].ScopeLogs) != 1 {
		t.Fatal("log scope lost")
	}
	r := p.ResourceLogs[0]
	assertCanonicalWireResource(t, r.Resource, r.SchemaUrl)
	s := r.ScopeLogs[0]
	if s.Scope.GetName() != instrumentationScope || len(s.LogRecords) != 1 || s.LogRecords[0].Body.GetStringValue() != "cq.operation" {
		t.Fatal("log operation lost")
	}
	assertSignalLabels(t, s.LogRecords[0].Attributes, true)
}
func assertCanonicalMetricWire(t *testing.T, body []byte) {
	t.Helper()
	var p metricspb.ExportMetricsServiceRequest
	decodeResourceProto(t, body, &p)
	if len(p.ResourceMetrics) != 1 || len(p.ResourceMetrics[0].ScopeMetrics) != 1 {
		t.Fatal("metric scope lost")
	}
	r := p.ResourceMetrics[0]
	assertCanonicalWireResource(t, r.Resource, r.SchemaUrl)
	s := r.ScopeMetrics[0]
	if s.Scope.GetName() != instrumentationScope {
		t.Fatal("metric instrumentation lost")
	}
	assertResourceOperationCounter(t, s.Metrics)
}
func assertResourceOperationCounter(t *testing.T, metrics []*metricv1.Metric) {
	t.Helper()
	for _, m := range metrics {
		if m.Name != "cq.operation.total" {
			continue
		}
		points := m.GetSum().GetDataPoints()
		if len(points) != 1 || points[0].GetAsInt() != 1 {
			t.Fatal("metric count lost")
		}
		assertSignalLabels(t, points[0].Attributes, false)
		return
	}
	t.Fatal("operation metric lost")
}
func decodeResourceProto(t *testing.T, body []byte, p proto.Message) {
	t.Helper()
	if err := proto.Unmarshal(body, p); err != nil {
		t.Fatalf("decode signal protobuf: %v", err)
	}
}
func assertCanonicalWireResource(t *testing.T, r *resourcepb.Resource, schema string) {
	t.Helper()
	want := map[string]string{"service.name": "cq", "service.version": "synthetic-version", "deployment.environment.name": "test", "service.namespace": "synthetic-suite", "service.instance.id": "synthetic-instance"}
	if r == nil || len(r.Attributes) != len(want) || r.DroppedAttributesCount != 0 || len(r.EntityRefs) != 0 || schema != "" {
		t.Fatal("resource envelope is not canonical")
	}
	assertResourceAttributes(t, r.Attributes, want)
}
func assertResourceAttributes(t *testing.T, attrs []*commonpb.KeyValue, want map[string]string) {
	t.Helper()
	for _, a := range attrs {
		v, ok := want[a.Key]
		if !ok || a.Value.GetStringValue() != v {
			t.Fatal("noncanonical resource attribute")
		}
		delete(want, a.Key)
	}
	if len(want) != 0 {
		t.Fatal("missing canonical resource attribute")
	}
}
func TestMalformedResourceEnvironmentRejectedBeforeInitialization(t *testing.T) {
	for _, value := range []string{"SYNTHETIC_MISSING_EQUALS", "synthetic.identity=%ZZ"} {
		t.Run(value, func(t *testing.T) {
			t.Setenv("OTEL_RESOURCE_ATTRIBUTES", value)
			for _, signal := range []string{"traces", "metrics", "logs"} {
				t.Run(signal, func(t *testing.T) { assertMalformedSignalRejected(t, signal) })
			}
			t.Run("public_new", assertMalformedNewRejected)
			t.Run("disabled", assertMalformedDisabledUnchanged)
		})
	}
}
func assertMalformedSignalRejected(t *testing.T, signal string) {
	t.Helper()
	b := &otelBackend{exportState: &exportHealth{}}
	cleanupResourceBackend(t, b)
	c := resourcePipelineConfig()
	res, _ := telemetryResource(c, "synthetic-version")
	err := configureResourceSignal(resourcePipelineContext(t), b, c, res, &http.Client{Transport: resourceWireTransport{make(chan resourceWireRequest, 16)}}, signal)
	assertBoundedResourceInvalid(t, err)
	if b.traces != nil || b.metrics != nil || b.logs != nil {
		t.Fatal("malformed resource initialized provider")
	}
}
func assertMalformedNewRejected(t *testing.T) {
	t.Helper()
	c := resourcePipelineConfig()
	c.TokenFile = filepath.Join(t.TempDir(), "missing-token")
	_, err := New(context.Background(), &c, Options{})
	assertBoundedResourceInvalid(t, err)
}
func assertMalformedDisabledUnchanged(t *testing.T) {
	t.Helper()
	c := resourcePipelineConfig()
	disabled := false
	c.Enabled = &disabled
	r, err := New(context.Background(), &c, Options{})
	if err != nil || r.Health() != (Health{}) {
		t.Fatal("disabled telemetry changed behavior")
	}
}
func assertBoundedResourceInvalid(t *testing.T, err error) {
	t.Helper()
	if !fault.Is(err, fault.Invalid) || strings.Contains(err.Error(), "SYNTHETIC") || strings.Contains(err.Error(), "%ZZ") {
		t.Fatalf("expected bounded resource Invalid, got %v", err)
	}
}
