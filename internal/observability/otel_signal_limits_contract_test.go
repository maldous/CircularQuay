package observability

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"testing"

	"github.com/maldous/CircularQuay/internal/model"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
	logspb "go.opentelemetry.io/proto/otlp/collector/logs/v1"
	tracepb "go.opentelemetry.io/proto/otlp/collector/trace/v1"
	commonpb "go.opentelemetry.io/proto/otlp/common/v1"
	"google.golang.org/protobuf/proto"
)

type signalLimitsTransport struct{ bodies chan []byte }

func (c signalLimitsTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	body, err := io.ReadAll(request.Body)
	if err != nil {
		return nil, err
	}
	c.bodies <- body
	return &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": {"application/x-protobuf"}}, Body: io.NopCloser(bytes.NewReader(nil))}, nil
}

func TestConfiguredSignalsKeepLabelsDespiteAmbientAttributeLimits(t *testing.T) {
	for _, key := range []string{
		"OTEL_SPAN_ATTRIBUTE_COUNT_LIMIT", "OTEL_SPAN_ATTRIBUTE_VALUE_LENGTH_LIMIT",
		"OTEL_LOGRECORD_ATTRIBUTE_COUNT_LIMIT", "OTEL_LOGRECORD_ATTRIBUTE_VALUE_LENGTH_LIMIT",
	} {
		t.Setenv(key, "1")
	}
	ctx := context.Background()
	backend := &otelBackend{exportState: &exportHealth{}}
	t.Cleanup(func() {
		if err := backend.shutdown(ctx); err != nil {
			t.Errorf("shutdown synthetic signals: %v", err)
		}
	})
	ratio := 1.0
	config := model.Observability{TimeoutSeconds: 5, ExportIntervalSeconds: 60, BatchSize: 4, QueueSize: 4,
		Traces: &model.TracesSignal{SampleRatio: &ratio}}
	res := resource.NewSchemaless(attribute.String("service.name", "cq"))
	traces := signalLimitsTransport{bodies: make(chan []byte, 4)}
	logs := signalLimitsTransport{bodies: make(chan []byte, 4)}
	if err := backend.configureTraces(ctx, config, res, &http.Client{Transport: traces}, nil, "https://synthetic.invalid/v1/traces"); err != nil {
		t.Fatalf("configure synthetic traces: %v", err)
	}
	if err := backend.configureLogs(ctx, config, res, &http.Client{Transport: logs}, nil, "https://synthetic.invalid/v1/logs"); err != nil {
		t.Fatalf("configure synthetic logs: %v", err)
	}
	_, finish := Start(WithRuntime(ctx, &Runtime{backend: backend}), Operation{Phase: PhaseRecovery, ProviderKind: "openbao"})
	finish(OutcomeFallback)
	if err := backend.traces.ForceFlush(ctx); err != nil {
		t.Fatalf("flush synthetic traces: %v", err)
	}
	if err := backend.logs.ForceFlush(ctx); err != nil {
		t.Fatalf("flush synthetic logs: %v", err)
	}
	t.Run("traces", func(t *testing.T) { assertSignalTraceLabels(t, capturedSignalBody(t, traces)) })
	t.Run("logs", func(t *testing.T) { assertSignalLogLabels(t, capturedSignalBody(t, logs)) })
}

func capturedSignalBody(t *testing.T, capture signalLimitsTransport) []byte {
	t.Helper()
	select {
	case body := <-capture.bodies:
		return body
	default:
		t.Fatal("flushed signal did not reach the in-memory HTTP transport")
		return nil
	}
}

func assertSignalTraceLabels(t *testing.T, body []byte) {
	t.Helper()
	var request tracepb.ExportTraceServiceRequest
	if err := proto.Unmarshal(body, &request); err != nil {
		t.Fatalf("decode trace protobuf: %v", err)
	}
	if len(request.ResourceSpans) != 1 || len(request.ResourceSpans[0].ScopeSpans) != 1 || len(request.ResourceSpans[0].ScopeSpans[0].Spans) != 1 {
		t.Fatal("unexpected trace export shape")
	}
	attrs := request.ResourceSpans[0].ScopeSpans[0].Spans[0].Attributes
	assertSignalLabels(t, attrs, false)
}

func assertSignalLogLabels(t *testing.T, body []byte) {
	t.Helper()
	var request logspb.ExportLogsServiceRequest
	if err := proto.Unmarshal(body, &request); err != nil {
		t.Fatalf("decode log protobuf: %v", err)
	}
	if len(request.ResourceLogs) != 1 || len(request.ResourceLogs[0].ScopeLogs) != 1 || len(request.ResourceLogs[0].ScopeLogs[0].LogRecords) != 1 {
		t.Fatal("unexpected log export shape")
	}
	attrs := request.ResourceLogs[0].ScopeLogs[0].LogRecords[0].Attributes
	assertSignalLabels(t, attrs, true)
}

func assertSignalLabels(t *testing.T, attrs []*commonpb.KeyValue, withCorrelation bool) {
	t.Helper()
	want := map[string]string{"cq.phase": "recovery", "cq.provider_kind": "openbao", "cq.outcome": "fallback"}
	if withCorrelation {
		want["cq.correlation_id"] = ""
	}
	if len(attrs) != len(want) {
		t.Fatalf("signal attribute count = %d, want %d", len(attrs), len(want))
	}
	for _, attr := range attrs {
		expected, ok := want[attr.Key]
		value := attr.Value.GetStringValue()
		if attr.Key == "cq.correlation_id" {
			assertOperationLogCorrelation(t, value)
		} else if !ok || value != expected {
			t.Fatal("signal lost a complete closed-vocabulary label")
		}
		delete(want, attr.Key)
	}
	if len(want) != 0 {
		t.Fatal("signal omitted a required label")
	}
}
