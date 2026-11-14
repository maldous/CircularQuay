package observability

import (
	"context"
	"errors"
	"math"
	"path/filepath"
	"testing"

	"github.com/maldous/CircularQuay/internal/fault"
	"github.com/maldous/CircularQuay/internal/model"
)

func TestNewRejectsNaNSamplingBeforeReadingCredentials(t *testing.T) {
	enabled := true
	ratio := math.NaN()
	config := &model.Observability{
		Enabled: &enabled, ServiceName: "cq", DiagnosticLevel: "off",
		Endpoint: "https://collector.invalid", TokenFile: filepath.Join(t.TempDir(), "missing-token"),
		TimeoutSeconds: 5, ExportIntervalSeconds: 5, BatchSize: 4, QueueSize: 4,
		Traces: &model.TracesSignal{Enabled: &enabled, SampleRatio: &ratio},
	}
	// A credential read would fail with Trust, so the exact public Invalid
	// category establishes that sampling is rejected before that boundary.
	runtime, err := New(context.Background(), config, Options{})
	if runtime != nil {
		t.Fatal("invalid sampling returned an active telemetry runtime")
	}
	var safe *fault.Error
	if !errors.As(err, &safe) || safe.Kind != fault.Invalid || safe.Operation != "observability traces" {
		t.Fatal("invalid sampling was not rejected before credential access")
	}
}
