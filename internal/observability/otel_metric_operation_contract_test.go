package observability

import (
	"context"
	"math"
	"testing"

	"go.opentelemetry.io/otel/attribute"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

func TestOperationMetricsKeepBoundedLabelsAndCompletionCounts(t *testing.T) {
	reader := sdkmetric.NewManualReader()
	provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	t.Cleanup(func() {
		if err := provider.Shutdown(context.Background()); err != nil {
			t.Errorf("shutdown in-memory metric provider: %v", err)
		}
	})
	backend := &otelBackend{exportState: &exportHealth{}}
	if err := backend.configureOperationMetrics(provider.Meter(instrumentationScope)); err != nil {
		t.Fatalf("configure operation instruments: %v", err)
	}
	runtime := &Runtime{backend: backend}
	ctx := WithRuntime(context.Background(), runtime)
	operationCtx, finish := Start(ctx, Operation{
		Phase:        Phase("synthetic private workload"),
		ProviderKind: "https://private.invalid/?token=synthetic-private-token",
	})
	Record(operationCtx, Operation{Phase: PhaseRecovery, ProviderKind: "openbao"}, OutcomeFallback)
	finish(Outcome("synthetic private error response"))
	var result metricdata.ResourceMetrics
	if err := reader.Collect(context.Background(), &result); err != nil {
		t.Fatalf("collect in-memory operation metrics: %v", err)
	}
	metrics := operationMetricResults(t, result)
	assertOperationCounter(t, metrics["cq.operation.total"])
	assertOperationDuration(t, metrics["cq.operation.duration"])
	if got := runtime.Health(); got != (Health{}) {
		t.Fatalf("in-memory measurement changed export health: %#v", got)
	}
}

func operationMetricResults(t *testing.T, result metricdata.ResourceMetrics) map[string]metricdata.Metrics {
	t.Helper()
	if len(result.ScopeMetrics) != 1 || result.ScopeMetrics[0].Scope.Name != instrumentationScope {
		t.Fatal("operation metrics lost their instrumentation scope")
	}
	metrics := map[string]metricdata.Metrics{}
	for _, metric := range result.ScopeMetrics[0].Metrics {
		metrics[metric.Name] = metric
	}
	if len(metrics) != 2 || len(result.ScopeMetrics[0].Metrics) != 2 {
		t.Fatal("operation measurement emitted unexpected instruments")
	}
	return metrics
}

func operationMetricLabels(t *testing.T, attributes attribute.Set) string {
	t.Helper()
	labels := map[string]string{}
	for _, item := range attributes.ToSlice() {
		labels[string(item.Key)] = item.Value.AsString()
	}
	// A closed vocabulary prevents payloads, endpoints and correlation IDs from
	// becoming metric dimensions, rather than checking a token denylist.
	if len(labels) != 3 {
		t.Fatal("operation metrics exposed undeclared label dimensions")
	}
	return labels["cq.phase"] + "/" + labels["cq.provider_kind"] + "/" + labels["cq.outcome"]
}

func assertOperationCounter(t *testing.T, metric metricdata.Metrics) {
	t.Helper()
	sum, ok := metric.Data.(metricdata.Sum[int64])
	if !ok || !sum.IsMonotonic || sum.Temporality != metricdata.CumulativeTemporality || len(sum.DataPoints) != 2 {
		t.Fatal("operation total lost its cumulative counter contract")
	}
	want := map[string]int64{"recovery/openbao/fallback": 1, "cli/none/failed": 1}
	for _, point := range sum.DataPoints {
		labels := operationMetricLabels(t, point.Attributes)
		if expected, exists := want[labels]; !exists || point.Value != expected {
			t.Fatal("operation total leaked labels or miscounted lifecycle events")
		}
		delete(want, labels)
	}
	if len(want) != 0 {
		t.Fatal("operation total omitted a recorded lifecycle event")
	}
}

func assertOperationDuration(t *testing.T, metric metricdata.Metrics) {
	t.Helper()
	histogram, ok := metric.Data.(metricdata.Histogram[float64])
	if !ok || metric.Unit != "s" || histogram.Temporality != metricdata.CumulativeTemporality || len(histogram.DataPoints) != 1 {
		t.Fatal("operation duration lost its seconds histogram contract")
	}
	point := histogram.DataPoints[0]
	if operationMetricLabels(t, point.Attributes) != "cli/none/failed" || point.Count != 1 || math.IsNaN(point.Sum) || point.Sum < 0 {
		t.Fatal("operation duration leaked labels or counted standalone events")
	}
}
