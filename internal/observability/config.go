package observability

import (
	"context"
	"strings"
	"time"
	"unicode"

	"github.com/maldous/CircularQuay/internal/fault"
	"github.com/maldous/CircularQuay/internal/model"
	"github.com/maldous/CircularQuay/internal/platform"
)

const maximumTokenBytes = 8192

// Options supplies process-local details that never enter policy. The
// diagnostic directory must be a private state directory owned by cq.
type Options struct {
	DiagnosticDirectory string
	Version             string
}

// New constructs an explicitly configured OpenTelemetry pipeline. It does not
// inspect OTEL_*, proxy, credential, or resource environment variables. An
// absent or disabled declaration returns a no-op runtime without I/O.
func New(ctx context.Context, config *model.Observability, options Options) (*Runtime, error) {
	if config == nil || config.Enabled == nil || !*config.Enabled {
		return Disabled(), nil
	}
	if err := validateRuntimeConfig(config); err != nil {
		return nil, err
	}
	token, err := readToken(config.TokenFile)
	if err != nil {
		return nil, err
	}
	diagnostic, err := newDiagnostic(config.DiagnosticLevel, options.DiagnosticDirectory)
	if err != nil {
		return nil, err
	}
	configured, err := newOTelBackend(ctx, *config, token, options.Version, diagnostic)
	if err != nil {
		return nil, err
	}
	return &Runtime{backend: configured}, nil
}

func validateRuntimeConfig(config *model.Observability) error {
	if err := validatePipelineConfig(config); err != nil {
		return err
	}
	if err := validateResourceConfig(config); err != nil {
		return err
	}
	return validateSignalConfig(config)
}

func validatePipelineConfig(config *model.Observability) error {
	if config.ServiceName == "" || config.Endpoint == "" || !validRuntimePipelineBounds(config) {
		return fault.New(fault.Invalid, "observability configuration")
	}
	if _, err := parseCollectorURLs(config.Endpoint); err != nil || (config.TrustStoreFile == "") != (config.TrustStoreSHA256 == "") {
		return fault.New(fault.Invalid, "observability transport")
	}
	return nil
}

func validRuntimePipelineBounds(config *model.Observability) bool {
	return runtimeIntegerInRange(config.TimeoutSeconds, 1, 30) &&
		runtimeIntegerInRange(config.ExportIntervalSeconds, 1, 60) &&
		runtimeIntegerInRange(config.BatchSize, 1, 512) &&
		runtimeIntegerInRange(config.QueueSize, 1, 8192) &&
		config.BatchSize <= config.QueueSize
}

func runtimeIntegerInRange(value, minimum, maximum int) bool {
	return value >= minimum && value <= maximum
}

func validateResourceConfig(config *model.Observability) error {
	if !oneOf(config.DiagnosticLevel, "off", "error", "info", "debug") || !safeResourceText(config.ServiceName) || !safeResourceText(config.Environment) {
		return fault.New(fault.Invalid, "observability identity")
	}
	if config.ResourceAttributes != nil && (!safeResourceText(config.ResourceAttributes.ServiceNamespace) || !safeResourceText(config.ResourceAttributes.ServiceInstanceID)) {
		return fault.New(fault.Invalid, "observability resource")
	}
	return nil
}

func validateSignalConfig(config *model.Observability) error {
	if enabled(config.Traces) && !validRuntimeTraceSampleRatio(config.Traces.SampleRatio) {
		return fault.New(fault.Invalid, "observability traces")
	}
	if enabled(config.Metrics) && !runtimeIntegerInRange(config.Metrics.IntervalSeconds, 1, 3600) {
		return fault.New(fault.Invalid, "observability metrics")
	}
	if !enabled(config.Metrics) && !enabled(config.Traces) && !enabled(config.Logs) {
		return fault.New(fault.Invalid, "observability signals")
	}
	return nil
}

func validRuntimeTraceSampleRatio(value *float64) bool {
	// Positive bounds also reject NaN, whose less/greater comparisons are false.
	return value != nil && *value >= 0 && *value <= 1
}

func enabled(signal any) bool {
	switch value := signal.(type) {
	case *model.MetricsSignal:
		return value != nil && value.Enabled != nil && *value.Enabled
	case *model.TracesSignal:
		return value != nil && value.Enabled != nil && *value.Enabled
	case *model.LogsSignal:
		return value != nil && value.Enabled != nil && *value.Enabled
	default:
		return false
	}
}

func readToken(path string) (string, error) {
	if path == "" {
		return "", nil
	}
	data, err := platform.ReadProtected(path, false)
	if err != nil || len(data) == 0 || len(data) > maximumTokenBytes {
		return "", fault.New(fault.Trust, "observability token")
	}
	token := strings.TrimSpace(string(data))
	if token == "" || len(token) > maximumTokenBytes || strings.ContainsFunc(token, unicode.IsControl) {
		return "", fault.New(fault.Invalid, "observability token")
	}
	return token, nil
}

func safeResourceText(value string) bool {
	if value == "" {
		return true
	}
	if len(value) > 128 || strings.ContainsFunc(value, unicode.IsControl) {
		return false
	}
	return true
}

func oneOf(value string, values ...string) bool {
	for _, candidate := range values {
		if value == candidate {
			return true
		}
	}
	return false
}

func signalTimeout(config model.Observability) time.Duration {
	return time.Duration(config.TimeoutSeconds) * time.Second
}
