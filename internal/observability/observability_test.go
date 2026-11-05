package observability

import (
	"context"
	"errors"
	"testing"

	"github.com/maldous/CircularQuay/internal/fault"
	"github.com/maldous/CircularQuay/internal/model"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

type captureBackend struct {
	operations []Operation
	outcomes   []Outcome
	events     []Event
}

func (c *captureBackend) begin(ctx context.Context, operation Operation) (context.Context, span) {
	c.operations = append(c.operations, operation)
	return ctx, captureSpan{capture: c}
}

func (c *captureBackend) event(_ context.Context, event Event) { c.events = append(c.events, event) }
func (*captureBackend) health() Health                         { return Health{} }
func (*captureBackend) shutdown(context.Context) error {
	return errors.New("synthetic exporter unavailable")
}

type captureSpan struct{ capture *captureBackend }

func (s captureSpan) end(outcome Outcome) { s.capture.outcomes = append(s.capture.outcomes, outcome) }

func TestOperationCatalogNormalizesUnboundedInputs(t *testing.T) {
	capture := &captureBackend{}
	ctx := WithRuntime(context.Background(), &Runtime{backend: capture})
	_, finish := Start(ctx, Operation{Phase: Phase("workload-alpha-secret"), ProviderKind: "https://private.example/token"})
	finish(Outcome("raw provider error"))
	if len(capture.operations) != 1 || capture.operations[0] != (Operation{Phase: PhaseCLI, ProviderKind: "none"}) {
		t.Fatalf("unbounded operation escaped catalog: %#v", capture.operations)
	}
	if len(capture.outcomes) != 1 || capture.outcomes[0] != OutcomeFailed {
		t.Fatalf("unbounded outcome escaped catalog: %#v", capture.outcomes)
	}
}

func TestRecordCarriesOnlyOpaqueCorrelation(t *testing.T) {
	capture := &captureBackend{}
	ctx := WithRuntime(context.Background(), &Runtime{backend: capture})
	Record(ctx, Operation{Phase: PhaseRecovery, ProviderKind: "openbao"}, OutcomeFallback)
	if len(capture.events) != 1 {
		t.Fatal("event was not recorded")
	}
	event := capture.events[0]
	if len(event.Correlation) != 32 || event.Operation != (Operation{Phase: PhaseRecovery, ProviderKind: "openbao"}) || event.Outcome != OutcomeFallback {
		t.Fatalf("unsafe or incomplete event: %#v", event)
	}
}

func TestOutcomeForErrorDoesNotExposeErrorText(t *testing.T) {
	for err, want := range map[error]Outcome{
		context.Canceled:         OutcomeCancelled,
		context.DeadlineExceeded: OutcomeCancelled,
		errors.New("SYNTHETIC_PROVIDER_SECRET_MUST_NOT_ESCAPE"): OutcomeFailed,
	} {
		if got := OutcomeForError(err); got != want {
			t.Fatalf("OutcomeForError(%v) = %q, want %q", err, got, want)
		}
	}
}

func TestRuntimeConfigurationRejectsNonExecutableSignals(t *testing.T) {
	enabled := true
	disabled := false
	ratio := 0.5
	base := &model.Observability{
		Enabled: &enabled, ServiceName: "cq", DiagnosticLevel: "off", Endpoint: "https://collector.example",
		TimeoutSeconds: 5, ExportIntervalSeconds: 5, BatchSize: 4, QueueSize: 4,
		Metrics: &model.MetricsSignal{Enabled: &enabled, IntervalSeconds: 5},
		Traces:  &model.TracesSignal{Enabled: &disabled},
		Logs:    &model.LogsSignal{Enabled: &disabled},
	}
	if err := validateRuntimeConfig(base); err != nil {
		t.Fatalf("valid metrics configuration rejected: %v", err)
	}
	base.QueueSize = 3
	if err := validateRuntimeConfig(base); err == nil {
		t.Fatal("batch larger than queue accepted")
	}
	base.QueueSize = 4
	base.Metrics.Enabled = &disabled
	base.Traces = &model.TracesSignal{Enabled: &enabled, SampleRatio: &ratio}
	if err := validateRuntimeConfig(base); err != nil {
		t.Fatalf("valid trace configuration rejected: %v", err)
	}
}

type failingSpanExporter struct {
	err      error
	canceled bool
}

func (e *failingSpanExporter) ExportSpans(ctx context.Context, _ []sdktrace.ReadOnlySpan) error {
	e.canceled = errors.Is(ctx.Err(), context.Canceled)
	return e.err
}

func (*failingSpanExporter) Shutdown(context.Context) error { return nil }

func TestTraceExporterFailureIsBoundedAndDoesNotExposeDelegateError(t *testing.T) {
	delegate := &failingSpanExporter{err: errors.New("SYNTHETIC_COLLECTOR_RESPONSE_WITH_TOKEN")}
	health := &exportHealth{}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := (safeTraceExporter{next: delegate, health: health}).ExportSpans(ctx, nil)
	if err == nil || err.Error() != "telemetry export" || health.count() != 1 || !delegate.canceled {
		t.Fatalf("safe exporter error=%v health=%d canceled=%v", err, health.count(), delegate.canceled)
	}
}

func TestDisabledRuntimeDoesNotRequireConfiguration(t *testing.T) {
	runtime, err := New(context.Background(), nil, Options{})
	if err != nil || runtime.Health() != (Health{}) {
		t.Fatalf("disabled runtime = %#v, %v", runtime, err)
	}
}

func TestRuntimeRejectsInvalidEndpointBeforeTokenRead(t *testing.T) {
	enabled := true
	config := &model.Observability{
		Enabled: &enabled, ServiceName: "cq", DiagnosticLevel: "off", Endpoint: "http://collector.example",
		TokenFile: "/synthetic/missing-token", TimeoutSeconds: 5, ExportIntervalSeconds: 5, BatchSize: 4, QueueSize: 4,
		Metrics: &model.MetricsSignal{Enabled: &enabled, IntervalSeconds: 5},
	}
	if _, err := New(context.Background(), config, Options{}); err == nil || !fault.Is(err, fault.Invalid) {
		t.Fatalf("invalid endpoint result = %v", err)
	}
}
