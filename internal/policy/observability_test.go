package policy

import (
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/maldous/CircularQuay/internal/model"
	"gopkg.in/yaml.v3"
)

func boolValue(value bool) *bool { return &value }

func floatValue(value float64) *float64 { return &value }

func validObservability() *model.Observability {
	return &model.Observability{
		Enabled: boolValue(true), ServiceName: "cquay", Environment: "production", DiagnosticLevel: "info",
		ResourceAttributes: &model.ResourceAttributes{ServiceNamespace: "platform", ServiceInstanceID: "instance-01"},
		Metrics:            &model.MetricsSignal{Enabled: boolValue(true), IntervalSeconds: 60},
		Traces:             &model.TracesSignal{Enabled: boolValue(true), SampleRatio: floatValue(0.1)},
		Logs:               &model.LogsSignal{Enabled: boolValue(true)},
		Endpoint:           "https://collector.example.test", TimeoutSeconds: 10, ExportIntervalSeconds: 10, BatchSize: 128, QueueSize: 2048,
	}
}

func validUpdates() *model.Updates {
	return &model.Updates{Check: "interactive", Channel: "stable", IntervalHours: 24, TimeoutSeconds: 10, MinimumVersion: "v1.0.0"}
}

func TestObservabilityPolicyContractRejectsInactiveOrIncompleteDeclarations(t *testing.T) {
	tests := []struct {
		name string
		edit func(*model.Observability)
	}{
		{"missing enabled", func(value *model.Observability) { value.Enabled = nil }},
		{"disabled options", func(value *model.Observability) { value.Enabled = boolValue(false); value.ServiceName = "cquay" }},
		{"service name", func(value *model.Observability) { value.ServiceName = "" }},
		{"missing signals", func(value *model.Observability) { value.Metrics = nil }},
		{"all signals disabled", func(value *model.Observability) {
			value.Metrics.Enabled = boolValue(false)
			value.Metrics.IntervalSeconds = 0
			value.Traces.Enabled = boolValue(false)
			value.Traces.SampleRatio = nil
			value.Logs.Enabled = boolValue(false)
		}},
		{"disabled metric interval", func(value *model.Observability) { value.Metrics.Enabled = boolValue(false) }},
		{"trace sample omitted", func(value *model.Observability) { value.Traces.SampleRatio = nil }},
		{"collector path", func(value *model.Observability) { value.Endpoint = "https://collector.example.test/v1/metrics" }},
		{"trust pairing", func(value *model.Observability) { value.TrustStoreFile = "/private/roots.pem" }},
		{"relative token", func(value *model.Observability) { value.TokenFile = "token" }},
		{"empty resource", func(value *model.Observability) { value.ResourceAttributes = &model.ResourceAttributes{} }},
		{"queue range", func(value *model.Observability) { value.QueueSize = 8193 }},
		{"batch exceeds queue", func(value *model.Observability) { value.BatchSize = 512; value.QueueSize = 128 }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			policy := validPolicy()
			policy.Observability = validObservability()
			test.edit(policy.Observability)
			if Validate(policy) == nil {
				t.Fatal("invalid observability declaration accepted")
			}
		})
	}
}

func TestUpdatePolicyContractSupportsAirGappedMinimumVersion(t *testing.T) {
	policy := validPolicy()
	policy.Updates = &model.Updates{Check: "never", MinimumVersion: "v1.0.0"}
	if err := Validate(policy); err != nil {
		t.Fatalf("air-gapped minimum version rejected: %v", err)
	}
	for _, test := range []struct {
		name string
		edit func(*model.Updates)
	}{
		{"unknown check", func(value *model.Updates) { value.Check = "background" }},
		{"network when never", func(value *model.Updates) { value.FeedEndpoint = "https://updates.example.test" }},
		{"channel when never", func(value *model.Updates) { value.Channel = "stable" }},
		{"invalid minimum", func(value *model.Updates) { value.MinimumVersion = "1.0" }},
		{"missing interactive channel", func(value *model.Updates) { value.Check = "interactive" }},
		{"trust pairing", func(value *model.Updates) {
			value.Check = "interactive"
			value.Channel = "stable"
			value.IntervalHours = 24
			value.TimeoutSeconds = 10
			value.TrustStoreSHA256 = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
		}},
		{"relative token", func(value *model.Updates) {
			value.Check = "interactive"
			value.Channel = "stable"
			value.IntervalHours = 24
			value.TimeoutSeconds = 10
			value.TokenFile = "token"
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			policy := validPolicy()
			policy.Updates = &model.Updates{Check: "never"}
			test.edit(policy.Updates)
			if Validate(policy) == nil {
				t.Fatal("invalid updates declaration accepted")
			}
		})
	}
}

func TestOperationalSchemaAndParsePreserveRequiredFieldParity(t *testing.T) {
	policy := validPolicy()
	policy.Observability = validObservability()
	policy.Updates = validUpdates()
	document := policyJSONValue(t, policy)
	assertSchemaAndParseAccept(t, document)
	for _, edit := range []func(map[string]any){
		func(value map[string]any) {
			value["observability"] = map[string]any{"enabled": false, "service_name": ""}
		},
		func(value map[string]any) { value["observability"].(map[string]any)["environment"] = "" },
		func(value map[string]any) {
			value["observability"].(map[string]any)["resource_attributes"].(map[string]any)["service_namespace"] = ""
		},
		func(value map[string]any) {
			value["observability"].(map[string]any)["metrics"] = map[string]any{"enabled": false, "interval_seconds": 0}
		},
		func(value map[string]any) {
			value["observability"].(map[string]any)["metrics"].(map[string]any)["enabled"] = false
		},
		func(value map[string]any) { delete(value["observability"].(map[string]any), "traces") },
		func(value map[string]any) {
			value["observability"].(map[string]any)["endpoint"] = "https://collector.example.test/v1/logs"
		},
		func(value map[string]any) { value["updates"].(map[string]any)["feed_endpoint"] = "" },
		func(value map[string]any) { value["updates"] = map[string]any{"check": "never", "feed_endpoint": ""} },
		func(value map[string]any) { delete(value["updates"].(map[string]any), "channel") },
	} {
		candidate := policyJSONValue(t, policy)
		edit(candidate)
		assertSchemaAndParseReject(t, candidate)
	}
}

func TestOperationalSchemaAndYAMLParsePreserveTypeAndNullParity(t *testing.T) {
	policy := validPolicy()
	policy.Observability = validObservability()
	policy.Updates = validUpdates()
	valid := policyJSONValue(t, policy)
	data, err := yaml.Marshal(valid)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = Parse(data, "/private/policy.yaml"); err != nil {
		t.Fatalf("valid YAML policy rejected: %v", err)
	}
	for _, edit := range []func(map[string]any){
		func(value map[string]any) { value["observability"] = nil },
		func(value map[string]any) { value["updates"] = nil },
		func(value map[string]any) { value["observability"].(map[string]any)["environment"] = nil },
		func(value map[string]any) {
			value["observability"].(map[string]any)["resource_attributes"].(map[string]any)["service_namespace"] = nil
		},
		func(value map[string]any) {
			value["observability"].(map[string]any)["metrics"].(map[string]any)["interval_seconds"] = nil
		},
		func(value map[string]any) { value["observability"].(map[string]any)["service_name"] = 123 },
		func(value map[string]any) { value["observability"].(map[string]any)["environment"] = true },
		func(value map[string]any) {
			value["observability"].(map[string]any)["resource_attributes"].(map[string]any)["service_namespace"] = 1
		},
		func(value map[string]any) {
			value["observability"].(map[string]any)["metrics"].(map[string]any)["enabled"] = "true"
		},
		func(value map[string]any) {
			value["observability"].(map[string]any)["metrics"].(map[string]any)["interval_seconds"] = "60"
		},
		func(value map[string]any) {
			value["observability"].(map[string]any)["traces"].(map[string]any)["sample_ratio"] = "0.1"
		},
		func(value map[string]any) { value["updates"].(map[string]any)["feed_endpoint"] = 123 },
		func(value map[string]any) { value["updates"].(map[string]any)["feed_endpoint"] = nil },
		func(value map[string]any) { value["updates"].(map[string]any)["minimum_version"] = nil },
		func(value map[string]any) { value["updates"].(map[string]any)["interval_hours"] = "24" },
	} {
		candidate := policyJSONValue(t, policy)
		edit(candidate)
		assertSchemaAndYAMLParseReject(t, candidate)
	}
}

func assertSchemaAndYAMLParseReject(t *testing.T, candidate map[string]any) {
	t.Helper()
	if err := resolvedPolicySchema(t).Validate(candidate); err == nil {
		t.Fatal("schema accepted invalid operational YAML shape")
	}
	data, err := yaml.Marshal(candidate)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = Parse(data, "/private/policy.yaml"); err == nil {
		t.Fatal("Parse accepted YAML invalid under the schema")
	}
}

func TestOperationalReferencesAreProtectedInputs(t *testing.T) {
	base := t.TempDir()
	policy := validPolicy()
	policy.Observability = validObservability()
	policy.Observability.TokenFile = filepath.Join(base, "token")
	policy.Updates = validUpdates()
	policy.Updates.TrustStoreFile = filepath.Join(base, "roots.pem")
	policy.Updates.TrustStoreSHA256 = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	setFileTarget(&policy, "sink", policy.Observability.TokenFile)
	if Validate(policy) == nil {
		t.Fatal("writer collision with observability token accepted")
	}

	policy = validPolicy()
	policy.Updates = validUpdates()
	policy.Updates.FeedEndpoint = "https://updates.example.test/releases"
	data, err := json.Marshal(policy)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = Parse(data, filepath.Join(base, "policy.json")); err != nil {
		t.Fatalf("valid update policy rejected by Parse: %v", err)
	}
}
