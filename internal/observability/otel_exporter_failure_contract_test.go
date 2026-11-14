package observability

import (
	"context"
	"errors"
	"testing"

	sdklog "go.opentelemetry.io/otel/sdk/log"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

func TestSafeMetricExporterFailureAndRecoveryContract(t *testing.T) {
	for _, method := range []string{"Export", "ForceFlush", "Shutdown"} {
		t.Run(method, func(t *testing.T) {
			state := &exporterContractDelegate{failures: map[string]error{}}
			delegate := &metricContractDelegate{exporterContractDelegate: state}
			health := &exportHealth{}
			exporter := safeMetricExporter{next: delegate, health: health}
			metrics := &metricdata.ResourceMetrics{}
			calls := map[string]func(context.Context) error{
				"Export":     func(ctx context.Context) error { return exporter.Export(ctx, metrics) },
				"ForceFlush": exporter.ForceFlush,
				"Shutdown":   exporter.Shutdown,
			}
			assertExporterFailureAndRecovery(t, state, health, method, calls[method])
			if method == "Export" && delegate.metrics != metrics {
				t.Fatal("metric export did not preserve the supplied SDK payload")
			}
		})
	}
}

func TestSafeLogExporterLifecycleFailureAndRecoveryContract(t *testing.T) {
	// Log Export success/failure is already covered through the actual SDK by
	// TestOTelOperationLogsKeepOnlyBoundedMetadata and its advisory-failure test.
	for _, method := range []string{"ForceFlush", "Shutdown"} {
		t.Run(method, func(t *testing.T) {
			state := &exporterContractDelegate{failures: map[string]error{}}
			health := &exportHealth{}
			exporter := safeLogExporter{next: &logContractDelegate{state}, health: health}
			calls := map[string]func(context.Context) error{"ForceFlush": exporter.ForceFlush, "Shutdown": exporter.Shutdown}
			assertExporterFailureAndRecovery(t, state, health, method, calls[method])
		})
	}
}

func assertExporterFailureAndRecovery(t *testing.T, delegate *exporterContractDelegate, health *exportHealth, method string, call func(context.Context) error) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.WithValue(context.Background(), exporterContractContextKey{}, "synthetic-request"))
	cancel()
	private := errors.New("synthetic collector private response and token")
	delegate.failures[method] = private
	err := call(ctx)
	if err == nil || err.Error() != "telemetry export" || errors.Is(err, private) || errors.Unwrap(err) != nil {
		t.Fatal("exporter failure exposed delegate details or lost its safe failure")
	}
	assertExporterContextAndHealth(t, delegate.context, health)
	delegate.failures[method] = nil
	delegate.context = nil
	if err = call(ctx); err != nil {
		t.Fatal("successful delegate result became an exporter failure")
	}
	assertExporterContextAndHealth(t, delegate.context, health)
}

func assertExporterContextAndHealth(t *testing.T, received context.Context, health *exportHealth) {
	t.Helper()
	if received == nil || received.Err() != context.Canceled || received.Value(exporterContractContextKey{}) != "synthetic-request" {
		t.Fatal("exporter dropped caller cancellation or request context values")
	}
	runtime := &Runtime{backend: &otelBackend{exportState: health}}
	if got := runtime.Health(); got != (Health{ExportFailures: 1}) {
		t.Fatalf("export failure health was lost, reset or inflated: %#v", got)
	}
}

type exporterContractContextKey struct{}

// Each operation has an independent result. Calling the wrong lifecycle
// operation cannot accidentally reproduce the selected failure.
type exporterContractDelegate struct {
	failures map[string]error
	context  context.Context
}

func (d *exporterContractDelegate) result(method string, ctx context.Context) error {
	d.context = ctx
	return d.failures[method]
}

func (d *exporterContractDelegate) ForceFlush(ctx context.Context) error {
	return d.result("ForceFlush", ctx)
}

func (d *exporterContractDelegate) Shutdown(ctx context.Context) error {
	return d.result("Shutdown", ctx)
}

type metricContractDelegate struct {
	*exporterContractDelegate
	metrics *metricdata.ResourceMetrics
}

func (*metricContractDelegate) Temporality(sdkmetric.InstrumentKind) metricdata.Temporality {
	return metricdata.DeltaTemporality
}

func (*metricContractDelegate) Aggregation(sdkmetric.InstrumentKind) sdkmetric.Aggregation {
	return sdkmetric.AggregationSum{}
}

func (d *metricContractDelegate) Export(ctx context.Context, metrics *metricdata.ResourceMetrics) error {
	d.metrics = metrics
	return d.result("Export", ctx)
}

type logContractDelegate struct{ *exporterContractDelegate }

func (d *logContractDelegate) Export(ctx context.Context, _ []sdklog.Record) error {
	return d.result("Export", ctx)
}
