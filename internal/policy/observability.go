package policy

import (
	"math"
	"net/url"
	"path/filepath"
	"reflect"
	"regexp"

	"github.com/maldous/CircularQuay/internal/model"
	"github.com/maldous/CircularQuay/internal/platform"
	"github.com/maldous/CircularQuay/internal/transport"
)

var strictSemVer = regexp.MustCompile(`^v?(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(?:-(?:0|[1-9][0-9]*|[0-9A-Za-z-]*[A-Za-z-][0-9A-Za-z-]*)(?:\.(?:0|[1-9][0-9]*|[0-9A-Za-z-]*[A-Za-z-][0-9A-Za-z-]*))*)?(?:\+[0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*)?$`)

var observabilityFields = fieldSet("enabled", "service_name", "environment", "diagnostic_level", "resource_attributes", "metrics", "traces", "logs", "endpoint", "token_file", "trust_store_file", "trust_store_sha256", "timeout_seconds", "export_interval_seconds", "batch_size", "queue_size")
var updateFields = fieldSet("check", "channel", "interval_hours", "timeout_seconds", "feed_endpoint", "token_file", "trust_store_file", "trust_store_sha256", "minimum_version")

// validateObservability accepts only an executable OTLP contract. It does not
// open files or contact the collector; Load validates declared local inputs.
func validateObservability(config *model.Observability) error {
	if config == nil {
		return nil
	}
	if config.Enabled == nil {
		return invalid("observability_enabled")
	}
	if !*config.Enabled {
		return validateDisabledObservability(*config)
	}
	return validateEnabledObservability(config)
}

func validateDisabledObservability(config model.Observability) error {
	if observabilityHasOptions(config) {
		return invalid("observability_disabled_options")
	}
	return nil
}

func validateEnabledObservability(config *model.Observability) error {
	if !validObservabilityIdentity(config) {
		return invalid("observability_identity")
	}
	if !validResourceAttributes(config.ResourceAttributes) || !validSignals(config) {
		return invalid("observability_signals")
	}
	if !validObservabilityTransport(config) {
		return invalid("observability_transport")
	}
	if !validObservabilityBounds(config) {
		return invalid("observability_bounds")
	}
	return nil
}

func validObservabilityIdentity(config *model.Observability) bool {
	return config.ServiceName != "" && validTelemetryText(config.ServiceName) && validTelemetryText(config.Environment) && validDiagnosticLevel(config.DiagnosticLevel)
}

func validObservabilityTransport(config *model.Observability) bool {
	return validOTLPOrigin(config.Endpoint) && validProtectedReference(config.TokenFile) && validTrustReferences(config.TrustStoreFile, config.TrustStoreSHA256)
}

func validObservabilityBounds(config *model.Observability) bool {
	return inRange(config.TimeoutSeconds, 1, 30) && inRange(config.ExportIntervalSeconds, 1, 60) && inRange(config.BatchSize, 1, 512) && inRange(config.QueueSize, 1, 8192) && config.BatchSize <= config.QueueSize
}

func inRange(value, minimum, maximum int) bool { return value >= minimum && value <= maximum }

func observabilityHasOptions(config model.Observability) bool {
	return anyTrue(config.ServiceName != "", config.Environment != "", config.DiagnosticLevel != "", config.ResourceAttributes != nil, config.Metrics != nil, config.Traces != nil, config.Logs != nil, config.Endpoint != "", config.TokenFile != "", config.TrustStoreFile != "", config.TrustStoreSHA256 != "", config.TimeoutSeconds != 0, config.ExportIntervalSeconds != 0, config.BatchSize != 0, config.QueueSize != 0)
}

func anyTrue(values ...bool) bool {
	for _, value := range values {
		if value {
			return true
		}
	}
	return false
}

func validTelemetryText(value string) bool {
	return value == "" || safeText(value) && len(value) <= 128
}

func validDiagnosticLevel(value string) bool {
	return value == "off" || value == "error" || value == "info" || value == "debug"
}

func validResourceAttributes(value *model.ResourceAttributes) bool {
	return value == nil || (value.ServiceNamespace != "" || value.ServiceInstanceID != "") && validTelemetryText(value.ServiceNamespace) && validTelemetryText(value.ServiceInstanceID)
}

func validSignals(config *model.Observability) bool {
	if !signalsPresent(config) || !validMetricsSignal(*config.Metrics) || !validTracesSignal(*config.Traces) {
		return false
	}
	return *config.Metrics.Enabled || *config.Traces.Enabled || *config.Logs.Enabled
}

func signalsPresent(config *model.Observability) bool {
	return config.Metrics != nil && config.Traces != nil && config.Logs != nil && config.Metrics.Enabled != nil && config.Traces.Enabled != nil && config.Logs.Enabled != nil
}

func validMetricsSignal(signal model.MetricsSignal) bool {
	if signal.Enabled == nil {
		return false
	}
	return !*signal.Enabled && signal.IntervalSeconds == 0 || *signal.Enabled && inRange(signal.IntervalSeconds, 1, 3600)
}

func validTracesSignal(signal model.TracesSignal) bool {
	if signal.Enabled == nil {
		return false
	}
	return !*signal.Enabled && signal.SampleRatio == nil || *signal.Enabled && validSampleRatio(signal.SampleRatio)
}

func validSampleRatio(value *float64) bool {
	return value != nil && !math.IsNaN(*value) && !math.IsInf(*value, 0) && *value >= 0 && *value <= 1
}

// validOperationalDocument checks the new policy objects before typed YAML
// decoding can coerce scalar values. JSON uses the same decoded tree, keeping
// YAML, JSON and the published schema aligned.
func validOperationalDocument(document any) bool {
	policy, ok := objectValue(document)
	return !ok || validRawOptionalObject(policy, "observability", validRawObservability) && validRawOptionalObject(policy, "updates", validRawUpdates)
}

func validRawObservability(value map[string]any) bool {
	return onlyFields(value, observabilityFields) && validRawObservabilityBasics(value) && validRawObservabilityChildren(value) && validRawObservabilityDisabled(value)
}

func validRawObservabilityBasics(value map[string]any) bool {
	return validRawBool(value, "enabled", true) && validRawStrings(value, "service_name", "environment", "diagnostic_level", "endpoint", "token_file", "trust_store_file", "trust_store_sha256") && validRawIntegers(value, "timeout_seconds", "export_interval_seconds", "batch_size", "queue_size")
}

func validRawObservabilityChildren(value map[string]any) bool {
	return validRawOptionalObject(value, "resource_attributes", validRawResourceAttributes) && validRawOptionalObject(value, "metrics", validRawMetrics) && validRawOptionalObject(value, "traces", validRawTraces) && validRawOptionalObject(value, "logs", validRawLogs)
}

func validRawObservabilityDisabled(value map[string]any) bool {
	if rawFalse(value, "enabled") {
		return len(value) == 1
	}
	return validRawDisabledSignal(value, "metrics") && validRawDisabledSignal(value, "traces")
}

func validRawResourceAttributes(value map[string]any) bool {
	return onlyFields(value, fieldSet("service_namespace", "service_instance_id")) && validRawStrings(value, "service_namespace", "service_instance_id")
}

func validRawMetrics(value map[string]any) bool {
	return onlyFields(value, fieldSet("enabled", "interval_seconds")) && validRawBool(value, "enabled", true) && validRawIntegers(value, "interval_seconds")
}

func validRawTraces(value map[string]any) bool {
	return onlyFields(value, fieldSet("enabled", "sample_ratio")) && validRawBool(value, "enabled", true) && validRawNumbers(value, "sample_ratio")
}

func validRawLogs(value map[string]any) bool {
	return onlyFields(value, fieldSet("enabled")) && validRawBool(value, "enabled", true)
}

func validRawDisabledSignal(value map[string]any, name string) bool {
	signal, found := value[name]
	config, isObject := objectValue(signal)
	return !found || !isObject || !rawFalse(config, "enabled") || len(config) == 1
}

func validRawUpdates(value map[string]any) bool {
	if !onlyFields(value, updateFields) || !validRawStrings(value, "check", "channel", "feed_endpoint", "token_file", "trust_store_file", "trust_store_sha256", "minimum_version") || !validRawIntegers(value, "interval_hours", "timeout_seconds") {
		return false
	}
	if value["check"] == "never" {
		return onlyFields(value, fieldSet("check", "minimum_version"))
	}
	return true
}

func validRawOptionalObject(value map[string]any, name string, validate func(map[string]any) bool) bool {
	child, found := value[name]
	if !found {
		return true
	}
	object, ok := objectValue(child)
	return ok && validate(object)
}

func onlyFields(value map[string]any, allowed map[string]bool) bool {
	for name := range value {
		if !allowed[name] {
			return false
		}
	}
	return true
}

func fieldSet(names ...string) map[string]bool {
	result := make(map[string]bool, len(names))
	for _, name := range names {
		result[name] = true
	}
	return result
}

func validRawStrings(value map[string]any, names ...string) bool {
	for _, name := range names {
		if entry, found := value[name]; found && !rawString(entry) {
			return false
		}
	}
	return true
}

func rawString(value any) bool {
	text, ok := value.(string)
	return ok && text != ""
}

func validRawBool(value map[string]any, name string, required bool) bool {
	entry, found := value[name]
	if !found {
		return !required
	}
	_, ok := entry.(bool)
	return ok
}

func rawFalse(value map[string]any, name string) bool {
	entry, found := value[name]
	boolean, ok := entry.(bool)
	return found && ok && !boolean
}

func validRawIntegers(value map[string]any, names ...string) bool {
	for _, name := range names {
		if entry, found := value[name]; found && !rawInteger(entry) {
			return false
		}
	}
	return true
}

func rawInteger(value any) bool {
	reflected := reflect.ValueOf(value)
	switch reflected.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return true
	case reflect.Float32, reflect.Float64:
		floatValue := reflected.Float()
		return !math.IsNaN(floatValue) && !math.IsInf(floatValue, 0) && math.Trunc(floatValue) == floatValue
	default:
		return false
	}
}

func validRawNumbers(value map[string]any, names ...string) bool {
	for _, name := range names {
		if entry, found := value[name]; found && !rawNumber(entry) {
			return false
		}
	}
	return true
}

func rawNumber(value any) bool {
	reflected := reflect.ValueOf(value)
	switch reflected.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return true
	case reflect.Float32, reflect.Float64:
		floatValue := reflected.Float()
		return !math.IsNaN(floatValue) && !math.IsInf(floatValue, 0)
	default:
		return false
	}
}

func objectValue(value any) (map[string]any, bool) {
	result, ok := value.(map[string]any)
	return result, ok
}

func validOTLPOrigin(raw string) bool {
	parsed, err := url.Parse(raw)
	return err == nil && secureEndpoint(raw, false) && (parsed.Path == "" || parsed.Path == "/") && parsed.RawPath == ""
}

func validProtectedReference(path string) bool {
	return path == "" || filepath.IsAbs(path) && filepath.Clean(path) == path
}

func validTrustReferences(path, digest string) bool {
	if path == "" && digest == "" {
		return true
	}
	return path != "" && validProtectedReference(path) && lowerHexDigest(digest)
}

// validateUpdates accepts an air-gapped minimum-version policy without a
// release feed. Interactive checks have bounded timing and an optional HTTPS
// enterprise mirror. It never makes a remote request.
func validateUpdates(config *model.Updates) error {
	if config == nil {
		return nil
	}
	if config.MinimumVersion != "" && !strictSemVer.MatchString(config.MinimumVersion) {
		return invalid("updates_minimum_version")
	}
	if config.Check == "never" {
		return validateDisabledUpdates(*config)
	}
	if config.Check != "interactive" || !validInteractiveUpdates(*config) {
		return invalid("updates_contract")
	}
	return nil
}

func validateDisabledUpdates(config model.Updates) error {
	if updatesHaveNetworkOptions(config) {
		return invalid("updates_disabled_options")
	}
	return nil
}

func validInteractiveUpdates(config model.Updates) bool {
	return validUpdateChannel(config.Channel) && inRange(config.IntervalHours, 1, 720) && inRange(config.TimeoutSeconds, 1, 30) && validUpdatesTransport(config)
}

func validUpdateChannel(value string) bool { return value == "stable" || value == "release-candidate" }

func updatesHaveNetworkOptions(config model.Updates) bool {
	return anyTrue(config.Channel != "", config.IntervalHours != 0, config.TimeoutSeconds != 0, config.FeedEndpoint != "", config.TokenFile != "", config.TrustStoreFile != "", config.TrustStoreSHA256 != "")
}

func validUpdatesTransport(config model.Updates) bool {
	return (config.FeedEndpoint == "" || secureEndpoint(config.FeedEndpoint, false)) && validProtectedReference(config.TokenFile) && validTrustReferences(config.TrustStoreFile, config.TrustStoreSHA256)
}

// ValidateLoadedOperationalReferences verifies file metadata for secret tokens
// without reading their contents. Public CA bundles are decoded and pinned so
// a collector cannot silently inherit system trust.
func ValidateLoadedOperationalReferences(policy model.Policy) error {
	if err := validateLoadedObservability(policy.Observability); err != nil {
		return err
	}
	return validateLoadedUpdates(policy.Updates)
}

func validateLoadedObservability(config *model.Observability) error {
	if config == nil || config.Enabled == nil || !*config.Enabled {
		return nil
	}
	if err := validateLoadedToken(config.TokenFile); err != nil {
		return invalid("observability_token_file")
	}
	if err := validateLoadedTrustStore(config.TrustStoreFile, config.TrustStoreSHA256); err != nil {
		return invalid("observability_trust_store")
	}
	return nil
}

func validateLoadedUpdates(config *model.Updates) error {
	if config == nil || config.Check != "interactive" {
		return nil
	}
	if err := validateLoadedToken(config.TokenFile); err != nil {
		return invalid("updates_token_file")
	}
	if err := validateLoadedTrustStore(config.TrustStoreFile, config.TrustStoreSHA256); err != nil {
		return invalid("updates_trust_store")
	}
	return nil
}

func validateLoadedToken(path string) error {
	if path == "" {
		return nil
	}
	return platform.ValidateFileMetadata(path, true)
}

func validateLoadedTrustStore(path, digest string) error {
	if path == "" {
		return nil
	}
	return transport.ValidateTrustStore(path, digest)
}
