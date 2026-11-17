package observability

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

func TestDiagnosticRefusalContractDisabledAndNilDoNotOpenSink(t *testing.T) {
	directory := filepath.Join(t.TempDir(), "not-created")
	disabled, err := newDiagnostic("off", directory)
	if err != nil || disabled == nil || disabled.path != "" {
		t.Fatal("disabled diagnostics required a sink")
	}
	disabled.record(context.Background(), diagnosticRecord{Event: "operation", Outcome: OutcomeFailed})
	var absent *diagnostic
	absent.record(context.Background(), diagnosticRecord{Event: "operation", Outcome: OutcomeFailed})
	if disabled.health() != 0 || absent.health() != 0 {
		t.Fatal("suppressed diagnostics reported a write failure")
	}
	if _, err := os.Stat(directory); !os.IsNotExist(err) {
		t.Fatal("disabled diagnostics created the selected sink directory")
	}
}

func TestDiagnosticRefusalContractDirectoryErrorsAreBounded(t *testing.T) {
	file := filepath.Join(t.TempDir(), "synthetic-private-directory-marker")
	original := []byte("SYNTHETIC_PRIVATE_DIAGNOSTIC_FILE")
	if err := os.WriteFile(file, original, 0o600); err != nil {
		t.Fatal(err)
	}
	for _, directory := range []string{"", "synthetic-relative-directory", file} {
		diagnostic, err := newDiagnostic("info", directory)
		if diagnostic != nil || err != errDiagnosticDirectory || err.Error() != "protected diagnostic directory required" {
			t.Fatal("invalid diagnostic directory lost its fixed refusal")
		}
	}
	retained, err := os.ReadFile(file)
	if err != nil || !bytes.Equal(retained, original) {
		t.Fatal("directory refusal changed the existing synthetic file")
	}
}

func TestDiagnosticRefusalContractPublicRecordsAreBoundedJSONLines(t *testing.T) {
	sink, runtime, ctx := diagnosticRefusalRuntime(t, "info")
	private := "SYNTHETIC_PRIVATE_DIAGNOSTIC_INPUT"
	Record(ctx, Operation{Phase: Phase(private), ProviderKind: private}, Outcome(private))
	Record(ctx, Operation{Phase: PhaseResolve, ProviderKind: "native"}, OutcomeSuccess)
	records := diagnosticRefusalRecords(t, sink.path)
	if len(records) != 2 {
		t.Fatal("public diagnostic events did not append separate JSONL records")
	}
	assertDiagnosticRefusalNormalizedOperation(t, records[0])
	if records[1].Phase != PhaseResolve || records[1].Provider != "native" || records[1].Outcome != OutcomeSuccess {
		t.Fatal("valid bounded event fields changed during append")
	}
	if runtime.Health() != (Health{}) {
		t.Fatal("successful local diagnostic writes became pipeline failures")
	}
}

func assertDiagnosticRefusalNormalizedOperation(t *testing.T, first diagnosticRecord) {
	t.Helper()
	if first.Event != "operation" || first.Phase != PhaseCLI || first.Provider != "none" || first.Outcome != OutcomeFailed {
		t.Fatal("public event values were not normalized before diagnostic disclosure")
	}
	if decoded, err := hex.DecodeString(first.Correlation); err != nil || len(decoded) != 16 {
		t.Fatal("public diagnostic record lacks bounded opaque correlation")
	}
}

func TestDiagnosticRefusalContractErrorLevelKeepsOnlyFailures(t *testing.T) {
	sink, _, ctx := diagnosticRefusalRuntime(t, "error")
	for _, outcome := range []Outcome{OutcomeSuccess, OutcomeFallback} {
		Record(ctx, Operation{Phase: PhaseResolve, ProviderKind: "native"}, outcome)
	}
	if _, err := os.Stat(sink.path); !os.IsNotExist(err) {
		t.Fatal("error-only diagnostics opened a file for a nonfailure")
	}
	want := []Outcome{OutcomeFailed, OutcomeUnavailable, OutcomeDenied, OutcomeExpired, OutcomeCancelled}
	for _, outcome := range want {
		Record(ctx, Operation{Phase: PhaseResolve, ProviderKind: "native"}, outcome)
	}
	records := diagnosticRefusalRecords(t, sink.path)
	if len(records) != len(want) {
		t.Fatal("error-level diagnostic filtering lost or added a failure record")
	}
	for index, outcome := range want {
		if records[index].Outcome != outcome {
			t.Fatal("error-level diagnostic append order or outcome changed")
		}
	}
}

func TestDiagnosticRefusalContractCanceledRequestsDoNotWrite(t *testing.T) {
	sink, runtime, ctx := diagnosticRefusalRuntime(t, "info")
	canceled, cancel := context.WithCancel(ctx)
	cancel()
	expired, expire := context.WithDeadline(ctx, time.Unix(1, 0))
	defer expire()
	for _, stopped := range []context.Context{canceled, expired} {
		if stopped.Err() == nil {
			t.Fatal("synthetic stopped context was not canceled")
		}
		Record(stopped, Operation{Phase: PhaseResolve, ProviderKind: "native"}, OutcomeFailed)
	}
	if _, err := os.Stat(sink.path); !os.IsNotExist(err) {
		t.Fatal("a canceled request produced a diagnostic file")
	}
	if runtime.Health() != (Health{}) {
		t.Fatal("request cancellation was misreported as diagnostic write failure")
	}
}

func TestDiagnosticRefusalContractWriteFailureStaysAdvisory(t *testing.T) {
	sink, runtime, ctx := diagnosticRefusalRuntime(t, "info")
	// A real filesystem refusal, independent of host UID and permission bypass:
	// the protected log path is an existing directory, never a writable file.
	if err := os.Mkdir(sink.path, 0o700); err != nil {
		t.Fatal(err)
	}
	_, finish := Start(ctx, Operation{Phase: PhaseWorkload, ProviderKind: "native"})
	finish(OutcomeSuccess)
	Record(ctx, Operation{Phase: PhaseRecovery, ProviderKind: "native"}, OutcomeDenied)
	if runtime.Health() != (Health{DiagnosticWriteFailures: 2}) {
		t.Fatal("diagnostic append failures were lost or affected unrelated health")
	}
	if entries, err := os.ReadDir(sink.path); err != nil || len(entries) != 0 {
		t.Fatal("failed log append replaced or wrote inside the blocking directory")
	}
	if err := runtime.Shutdown(context.Background()); err != nil {
		t.Fatal("diagnostic write failure became a pipeline shutdown failure")
	}
}

func TestDiagnosticRefusalContractExporterErrorProducesOneSafeRecord(t *testing.T) {
	sink, _, _ := diagnosticRefusalRuntime(t, "error")
	health := &exportHealth{diagnostic: sink}
	delegate := &diagnosticRefusalTraceExporter{failure: errors.New("SYNTHETIC_PRIVATE_DIAGNOSTIC_EXPORTER_BODY")}
	exporter := safeTraceExporter{next: delegate, health: health}
	for range 2 {
		err := exporter.ExportSpans(context.Background(), nil)
		assertDiagnosticRefusalExporterError(t, err, delegate.failure)
	}
	if health.count() != 2 || delegate.calls != 2 || sink.health() != 0 {
		t.Fatal("export failure accounting was lost or became a diagnostic write error")
	}
	records := diagnosticRefusalRecords(t, sink.path)
	if len(records) != 1 || records[0].Event != "telemetry_export" || records[0].Outcome != OutcomeUnavailable {
		t.Fatal("repeated exporter failure did not retain one bounded diagnostic event")
	}
}

func assertDiagnosticRefusalExporterError(t *testing.T, err, privateCause error) {
	t.Helper()
	if err == nil || err.Error() != "telemetry export" || errors.Is(err, privateCause) || errors.Unwrap(err) != nil {
		t.Fatal("exporter failure exposed its private cause")
	}
}

func diagnosticRefusalRuntime(t *testing.T, level string) (*diagnostic, *Runtime, context.Context) {
	t.Helper()
	// Darwin may place test data below a symlinked temporary root. Resolve only
	// this synthetic fixture; the production private-path checks stay strict.
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal("could not resolve the synthetic private diagnostic root")
	}
	sink, err := newDiagnostic(level, filepath.Join(root, "diagnostics"))
	if err != nil {
		t.Fatal("could not create the synthetic private diagnostic sink")
	}
	runtime := &Runtime{backend: &otelBackend{diag: sink, exportState: &exportHealth{}}}
	return sink, runtime, WithRuntime(context.Background(), runtime)
}

func diagnosticRefusalRecords(t *testing.T, path string) []diagnosticRecord {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil || len(data) == 0 || data[len(data)-1] != '\n' {
		t.Fatal("diagnostic sink is not a nonempty newline-terminated JSONL file")
	}
	if bytes.Contains(data, []byte("SYNTHETIC_PRIVATE_DIAGNOSTIC")) {
		t.Fatal("private synthetic input or exporter text escaped to diagnostics")
	}
	allowed := map[string]bool{"time": true, "event": true, "phase": true, "provider": true, "outcome": true, "correlation": true}
	lines := bytes.Split(bytes.TrimSuffix(data, []byte("\n")), []byte("\n"))
	records := make([]diagnosticRecord, 0, len(lines))
	for _, line := range lines {
		records = append(records, diagnosticRefusalRecord(t, line, allowed))
	}
	return records
}

func diagnosticRefusalRecord(t *testing.T, line []byte, allowed map[string]bool) diagnosticRecord {
	t.Helper()
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(line, &fields); err != nil {
		t.Fatal("diagnostic append did not preserve individual JSON objects")
	}
	for name := range fields {
		if !allowed[name] {
			t.Fatal("diagnostic record disclosed an undeclared field")
		}
	}
	var record diagnosticRecord
	if err := json.Unmarshal(line, &record); err != nil || record.Time.IsZero() || record.Event == "" || record.Outcome == "" {
		t.Fatal("diagnostic record lost its required bounded fields")
	}
	return record
}

type diagnosticRefusalTraceExporter struct {
	failure error
	calls   int
}

func (e *diagnosticRefusalTraceExporter) ExportSpans(context.Context, []sdktrace.ReadOnlySpan) error {
	e.calls++
	return e.failure
}

func (*diagnosticRefusalTraceExporter) Shutdown(context.Context) error { return nil }
