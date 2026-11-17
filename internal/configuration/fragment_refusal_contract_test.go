package configuration

import (
	"bytes"
	"errors"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/maldous/CircularQuay/internal/fault"
	"github.com/maldous/CircularQuay/internal/model"
	"github.com/maldous/CircularQuay/internal/policy"
)

func TestFragmentRefusalContractTopLevelAuthorityPresence(t *testing.T) {
	assertFragmentRefusalContract(t, fragmentRefusalDocument(), true)
	cases := []struct {
		name  string
		value any
	}{
		{"grants", []model.Grant{}},
		{"mcp", map[string]any{}},
		{"repository", map[string]any{}},
		{"observability", map[string]any{"enabled": false}},
		{"updates", map[string]any{"check": "never"}},
		{"composition", map[string]any{"includes": []model.Include{}, "conflicts": "reject"}},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			document := fragmentRefusalDocument()
			document[test.name] = test.value
			assertFragmentRefusalContract(t, document, false)
		})
	}
}

func TestFragmentRefusalContractSourcesRemainDataOnly(t *testing.T) {
	cases := []struct {
		name    string
		sources map[string]any
		allowed bool
	}{
		{"json", map[string]any{"data": map[string]any{"type": "json", "path": "values.json"}}, true},
		{"environment", map[string]any{"data": map[string]any{"type": "env"}}, true},
		{"sops_data", map[string]any{"data": map[string]any{"type": "sops", "path": "values.enc.json"}}, true},
		{"empty_sources", map[string]any{}, false},
		{"reserved_namespace_separator", map[string]any{"team__data": map[string]any{"type": "env"}}, false},
		{"invalid_identifier_start", map[string]any{"_data": map[string]any{"type": "env"}}, false},
		{"bitwarden_executable", map[string]any{"data": map[string]any{"type": "bitwarden", "path": "not-executed"}}, false},
		{"onepassword_executable", map[string]any{"data": map[string]any{"type": "onepassword", "path": "not-executed"}}, false},
		{"explicit_empty_sops_command", map[string]any{"data": map[string]any{"type": "json", "path": "values.json", "sops": ""}}, false},
		{"repository_relative_path", map[string]any{"data": map[string]any{"type": "json", "path": "repo:values.json"}}, false},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			document := fragmentRefusalDocument()
			document["sources"] = test.sources
			assertFragmentRefusalContract(t, document, test.allowed)
		})
	}
}

func TestFragmentRefusalContractWorkloadAuthorityPresence(t *testing.T) {
	control := fragmentRefusalDocument()
	control["workloads"] = map[string]any{"worker": fragmentRefusalWorkload()}
	assertFragmentRefusalContract(t, control, true)
	cases := []struct {
		name  string
		value any
	}{
		{"command", []string{}},
		{"command_sha256", ""},
		{"untrusted_code", false},
		{"directory", ""},
		{"snapshot", map[string]any{}},
		{"targets", map[string]any{}},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			workload := fragmentRefusalWorkload()
			workload[test.name] = test.value
			document := fragmentRefusalDocument()
			document["workloads"] = map[string]any{"worker": workload}
			assertFragmentRefusalContract(t, document, false)
		})
	}
}

func TestFragmentRefusalContractWorkloadIdentifiers(t *testing.T) {
	control := fragmentRefusalDocument()
	control["workloads"] = map[string]any{"worker-1": fragmentRefusalWorkload()}
	assertFragmentRefusalContract(t, control, true)
	for _, name := range []string{"", "_worker", "team__worker", "Worker"} {
		t.Run("reject_"+name, func(t *testing.T) {
			document := fragmentRefusalDocument()
			document["workloads"] = map[string]any{name: fragmentRefusalWorkload()}
			assertFragmentRefusalContract(t, document, false)
		})
	}
}

func fragmentRefusalDocument() map[string]any {
	return map[string]any{
		"version":   1,
		"sources":   map[string]any{"data": map[string]any{"type": "json", "path": "values.json"}},
		"workloads": map[string]any{},
	}
}

func fragmentRefusalWorkload() map[string]any {
	return map[string]any{
		"mode":     "online",
		"bindings": []model.Binding{{Name: "VALUE", Source: "data", Field: "value"}},
	}
}

func assertFragmentRefusalContract(t *testing.T, document map[string]any, allowed bool) {
	t.Helper()
	data := marshalValue(t, document)
	original := append([]byte(nil), data...)
	include := localInclude(t, data)
	// The guard receives the real typed decode of these exact bytes, never a
	// manufactured raw/decoded mismatch that the include loader cannot produce.
	child, err := policy.DecodeDocument(data, filepath.Join(include.BaseDirectory, "fragment.json"))
	if err != nil {
		t.Fatalf("fixture failed typed decoding before fragment admission: %v", err)
	}
	before := marshalPolicy(t, child)
	if got := validFragmentDocument(data, child); got != allowed {
		t.Fatalf("data-only admission = %v, want %v", got, allowed)
	}
	if !bytes.Equal(before, marshalPolicy(t, child)) || child.ConfigurationGuard != nil {
		t.Fatal("fragment admission mutated the decoded declaration")
	}
	decoded, err := decodeFragment(data, include)
	if allowed {
		if err != nil {
			t.Fatalf("data-only fragment refused: %v", err)
		}
		if !reflect.DeepEqual(decoded, child) {
			t.Fatal("accepted fragment changed its already-bound declaration")
		}
	} else {
		assertFragmentRefusalResult(t, decoded, err)
	}
	if !bytes.Equal(original, data) {
		t.Fatal("fragment admission mutated the caller's serialized input")
	}
}

func assertFragmentRefusalResult(t *testing.T, decoded model.Policy, err error) {
	t.Helper()
	var refusal *fault.Error
	if !errors.As(err, &refusal) {
		t.Fatalf("expected typed fragment refusal, got %v", err)
	}
	if refusal.Kind != fault.Invalid {
		t.Fatalf("fragment refusal kind = %q", refusal.Kind)
	}
	if refusal.Operation != "configuration_include_fragment" {
		t.Fatalf("refusal did not reach the fragment guard: %q", refusal.Operation)
	}
	if !reflect.DeepEqual(decoded, model.Policy{}) {
		t.Fatal("rejected fragment exposed a partial policy")
	}
}
