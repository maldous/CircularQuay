package observability

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/maldous/CircularQuay/internal/fault"
	"github.com/maldous/CircularQuay/internal/model"
)

func TestNewEnforcesTelemetryResourceBoundsBeforeCredentials(t *testing.T) {
	for _, test := range []struct {
		name      string
		configure func(*model.Observability)
		kind      fault.Kind
		operation string
	}{
		{"timeout above policy maximum", func(c *model.Observability) { c.TimeoutSeconds = 31 }, fault.Invalid, "observability configuration"},
		{"export interval above policy maximum", func(c *model.Observability) { c.ExportIntervalSeconds = 61 }, fault.Invalid, "observability configuration"},
		{"batch above policy maximum", func(c *model.Observability) { c.BatchSize = 513 }, fault.Invalid, "observability configuration"},
		{"queue above policy maximum", func(c *model.Observability) { c.QueueSize = 8193 }, fault.Invalid, "observability configuration"},
		{"metrics interval above policy maximum", func(c *model.Observability) { c.Metrics.IntervalSeconds = 3601 }, fault.Invalid, "observability metrics"},
		{"policy maxima reach credential boundary", func(*model.Observability) {}, fault.Trust, "observability token"},
	} {
		t.Run(test.name, func(t *testing.T) {
			config := telemetryUpperBoundsConfig(t)
			test.configure(config)
			// No token exists. A Trust/token result proves valid bounds reached
			// the next boundary, without starting an exporter or reading secrets.
			runtime, err := New(context.Background(), config, Options{})
			assertTelemetryBoundsFailure(t, runtime, err, test.kind, test.operation)
		})
	}
}

func assertTelemetryBoundsFailure(t *testing.T, runtime *Runtime, err error, kind fault.Kind, operation string) {
	t.Helper()
	if runtime != nil {
		t.Fatal("resource-boundary check returned a telemetry pipeline")
	}
	var safe *fault.Error
	if !errors.As(err, &safe) {
		t.Fatal("resource-boundary check returned no typed failure")
	}
	if safe.Kind != kind || safe.Operation != operation {
		t.Fatalf("got %s/%s; want %s/%s", safe.Kind, safe.Operation, kind, operation)
	}
}

func telemetryUpperBoundsConfig(t *testing.T) *model.Observability {
	t.Helper()
	enabled := true
	return &model.Observability{
		Enabled: &enabled, ServiceName: "cq", DiagnosticLevel: "off",
		Endpoint: "https://collector.invalid", TokenFile: filepath.Join(t.TempDir(), "missing-token"),
		TimeoutSeconds: 30, ExportIntervalSeconds: 60, BatchSize: 512, QueueSize: 8192,
		Metrics: &model.MetricsSignal{Enabled: &enabled, IntervalSeconds: 3600},
	}
}
