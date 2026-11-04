package observability

import (
	"context"
	"errors"
	"math"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/maldous/CircularQuay/internal/model"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

func TestCollectorURLsRequireRootHTTPSAndUseSignalPaths(t *testing.T) {
	urls, err := parseCollectorURLs("https://collector.example/")
	if err != nil {
		t.Fatalf("parse root collector URL: %v", err)
	}
	if urls.traces != "https://collector.example/v1/traces" || urls.metrics != "https://collector.example/v1/metrics" || urls.logs != "https://collector.example/v1/logs" {
		t.Fatalf("collector URLs = %#v", urls)
	}
	for _, endpoint := range []string{
		"http://collector.example",
		"https://collector.example/custom-path",
		"https://collector.example?endpoint=other",
	} {
		if _, err := parseCollectorURLs(endpoint); err == nil {
			t.Fatalf("accepted untrusted collector URL %q", endpoint)
		}
	}
}

func TestCollectorClientIsBoundedAndRejectsRedirects(t *testing.T) {
	client, health, err := collectorClient(model.Observability{TimeoutSeconds: 7}, nil)
	if err != nil {
		t.Fatalf("collector client: %v", err)
	}
	if client.Timeout != 7*time.Second || client.CheckRedirect == nil || health.count() != 0 {
		t.Fatalf("collector client bounds = timeout %v redirect %t failures %d", client.Timeout, client.CheckRedirect != nil, health.count())
	}
	request, err := http.NewRequest(http.MethodGet, "https://redirect.example", nil)
	if err != nil {
		t.Fatalf("synthetic request: %v", err)
	}
	if client.CheckRedirect(request, nil) == nil {
		t.Fatal("collector client accepted redirect")
	}
}

func TestTelemetryResourceIsClosedToDeclaredStaticFields(t *testing.T) {
	resource, err := telemetryResource(model.Observability{
		ServiceName: "cquay",
		Environment: "test",
		ResourceAttributes: &model.ResourceAttributes{
			ServiceNamespace:  "checkout",
			ServiceInstanceID: "worker-a",
		},
	}, "1.2.3")
	if err != nil {
		t.Fatalf("telemetry resource: %v", err)
	}
	got := map[string]string{}
	for _, value := range resource.Attributes() {
		got[string(value.Key)] = value.Value.AsString()
	}
	want := map[string]string{
		"service.name":                "cquay",
		"service.version":             "1.2.3",
		"deployment.environment.name": "test",
		"service.namespace":           "checkout",
		"service.instance.id":         "worker-a",
	}
	if len(got) != len(want) {
		t.Fatalf("resource attributes = %#v, want %#v", got, want)
	}
	for key, value := range want {
		if got[key] != value {
			t.Fatalf("resource attribute %q = %q, want %q", key, got[key], value)
		}
	}
}

type shutdownFailingSpanExporter struct{ calls int }

func (*shutdownFailingSpanExporter) ExportSpans(context.Context, []sdktrace.ReadOnlySpan) error {
	return nil
}

func (e *shutdownFailingSpanExporter) Shutdown(context.Context) error {
	e.calls++
	return errors.New("synthetic collector private detail")
}

func TestBackendShutdownReportsBoundedFailureOnce(t *testing.T) {
	health := &exportHealth{}
	delegate := &shutdownFailingSpanExporter{}
	backend := &otelBackend{
		traces:      sdktrace.NewTracerProvider(sdktrace.WithSyncer(safeTraceExporter{next: delegate, health: health})),
		diag:        &diagnostic{level: "off"},
		exportState: health,
	}
	first := backend.shutdown(context.Background())
	if first == nil || strings.Contains(first.Error(), "private detail") || !backend.shutdownFail.Load() || health.count() != 1 || delegate.calls != 1 {
		t.Fatalf("first shutdown = %v, health=%d, calls=%d, failed=%t", first, health.count(), delegate.calls, backend.shutdownFail.Load())
	}
	if second := backend.shutdown(context.Background()); second == nil || delegate.calls != 1 {
		t.Fatalf("second shutdown = %v, calls=%d", second, delegate.calls)
	}
}

func TestBackendLocalOperationRecordsWithoutProvider(t *testing.T) {
	backend := &otelBackend{diag: &diagnostic{level: "off"}, exportState: &exportHealth{}}
	_, current := backend.begin(context.Background(), Operation{Phase: PhaseResolve, ProviderKind: "native"})
	if _, ok := current.(localSpan); !ok {
		t.Fatalf("local backend span = %T, want localSpan", current)
	}
	current.end(OutcomeSuccess)
	if got := backend.health(); got != (Health{}) {
		t.Fatalf("local operation health = %#v", got)
	}
}

func TestSaturatingInt64BoundsRuntimeGauge(t *testing.T) {
	for input, want := range map[uint64]int64{
		0:                     0,
		uint64(math.MaxInt64): math.MaxInt64,
		math.MaxUint64:        math.MaxInt64,
	} {
		if got := saturatingInt64(input); got != want {
			t.Fatalf("saturatingInt64(%d) = %d, want %d", input, got, want)
		}
	}
}
