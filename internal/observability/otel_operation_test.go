package observability

import (
	"context"
	"encoding/hex"
	"errors"
	"testing"

	"go.opentelemetry.io/otel/attribute"
	otellog "go.opentelemetry.io/otel/log"
	sdklog "go.opentelemetry.io/otel/sdk/log"
)

type operationLogExporter struct {
	records []sdklog.Record
	err     error
}

func (e *operationLogExporter) Export(_ context.Context, records []sdklog.Record) error {
	for _, record := range records {
		e.records = append(e.records, record.Clone())
	}
	return e.err
}

func (*operationLogExporter) Shutdown(context.Context) error   { return nil }
func (*operationLogExporter) ForceFlush(context.Context) error { return nil }

func operationLogRuntime(t *testing.T, exportError error) (*Runtime, *operationLogExporter) {
	t.Helper()
	exporter := &operationLogExporter{err: exportError}
	health := &exportHealth{}
	// The synchronous SDK processor makes exporter output observable without a
	// collector, global provider, timer, or sleep. Production batching is separate.
	provider := sdklog.NewLoggerProvider(sdklog.WithProcessor(sdklog.NewSimpleProcessor(
		safeLogExporter{next: exporter, health: health},
	)))
	runtime := &Runtime{backend: &otelBackend{
		logger: provider.Logger("github.com/maldous/CircularQuay"),
		logs:   provider, exportState: health,
	}}
	t.Cleanup(func() {
		if err := runtime.Shutdown(context.Background()); err != nil {
			t.Errorf("shutdown operation log runtime: %v", err)
		}
	})
	return runtime, exporter
}

func assertOperationLog(t *testing.T, record sdklog.Record, operation Operation, outcome Outcome, severity otellog.Severity) string {
	t.Helper()
	assertOperationLogEnvelope(t, record, outcome, severity)
	attributes := assertOperationLogAttributes(t, record, operation, outcome)
	return assertOperationLogCorrelation(t, attributes["cq.correlation_id"])
}

func assertOperationLogEnvelope(t *testing.T, record sdklog.Record, outcome Outcome, severity otellog.Severity) {
	t.Helper()
	if record.EventName() != "cq.operation" || record.Body().AsString() != "cq.operation" {
		t.Fatalf("unexpected operation log event or body: %q / %q", record.EventName(), record.Body().AsString())
	}
	if record.Severity() != severity || record.SeverityText() != string(outcome) || record.Timestamp().IsZero() {
		t.Fatalf("operation log envelope: severity=%v text=%q timestamp=%v", record.Severity(), record.SeverityText(), record.Timestamp())
	}
}

func assertOperationLogAttributes(t *testing.T, record sdklog.Record, operation Operation, outcome Outcome) map[string]string {
	t.Helper()
	attributes := map[string]string{}
	record.WalkAttributes(func(value attribute.KeyValue) bool {
		attributes[string(value.Key)] = value.Value.AsString()
		return true
	})
	// The closed wire vocabulary, not a denylist of known secrets, is the
	// privacy oracle. Unknown caller labels and raw errors cannot add fields.
	if record.AttributesLen() != 4 || len(attributes) != 4 ||
		attributes["cq.phase"] != string(operation.Phase) ||
		attributes["cq.provider_kind"] != operation.ProviderKind ||
		attributes["cq.outcome"] != string(outcome) {
		t.Fatalf("unexpected operation log attributes: %#v", attributes)
	}
	return attributes
}

func assertOperationLogCorrelation(t *testing.T, value string) string {
	t.Helper()
	correlation, err := hex.DecodeString(value)
	if err != nil || len(correlation) != 16 {
		t.Fatalf("operation log correlation is not an opaque 128-bit identifier: %q", value)
	}
	return value
}

func TestOTelOperationLogsKeepOnlyBoundedMetadata(t *testing.T) {
	runtime, exporter := operationLogRuntime(t, nil)
	ctx := WithRuntime(context.Background(), runtime)
	operationCtx, finish := Start(ctx, Operation{
		Phase:        Phase("SYNTHETIC_PRIVATE_WORKLOAD"),
		ProviderKind: "https://provider.example/private?token=SYNTHETIC_CREDENTIAL",
	})
	Record(operationCtx, Operation{Phase: PhaseRecovery, ProviderKind: "openbao"}, OutcomeFallback)
	finish(OutcomeForError(errors.New("SYNTHETIC_PROVIDER_RESPONSE_WITH_SECRET")))
	Record(ctx, Operation{Phase: PhaseSource, ProviderKind: "env"}, OutcomeSuccess)
	if len(exporter.records) != 3 {
		t.Fatalf("exported operation records = %d, want 3", len(exporter.records))
	}
	correlation := assertOperationLog(t, exporter.records[0], Operation{Phase: PhaseRecovery, ProviderKind: "openbao"}, OutcomeFallback, otellog.SeverityInfo)
	completionCorrelation := assertOperationLog(t, exporter.records[1], Operation{Phase: PhaseCLI, ProviderKind: "none"}, OutcomeFailed, otellog.SeverityError)
	if completionCorrelation != correlation {
		t.Fatal("operation completion lost the nested lifecycle event's opaque correlation")
	}
	assertOperationLog(t, exporter.records[2], Operation{Phase: PhaseSource, ProviderKind: "env"}, OutcomeSuccess, otellog.SeverityInfo)
	if got := runtime.Health(); got != (Health{}) {
		t.Fatalf("successful operation log export health = %#v", got)
	}
}

func TestOTelOperationLogExportFailureRemainsAdvisory(t *testing.T) {
	runtime, exporter := operationLogRuntime(t, errors.New("SYNTHETIC_COLLECTOR_RESPONSE_WITH_SECRET"))
	ctx := WithRuntime(context.Background(), runtime)
	operation := Operation{Phase: PhaseWorkload, ProviderKind: "native"}
	operationCtx, finish := Start(ctx, operation)
	Record(operationCtx, operation, OutcomeSuccess)
	finish(OutcomeSuccess)
	if len(exporter.records) != 2 {
		t.Fatalf("failed exporter received %d operation records, want 2", len(exporter.records))
	}
	for _, record := range exporter.records {
		// A collector outage must not rewrite a successful workload outcome.
		assertOperationLog(t, record, operation, OutcomeSuccess, otellog.SeverityInfo)
	}
	if got := runtime.Health(); got != (Health{ExportFailures: 2}) {
		t.Fatalf("collector outage must be reported separately from workload outcome: %#v", got)
	}
}
