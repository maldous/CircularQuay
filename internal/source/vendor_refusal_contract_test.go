package source

import (
	"encoding/json"
	"errors"
	"reflect"
	"testing"

	"github.com/maldous/CircularQuay/internal/fault"
	"github.com/maldous/CircularQuay/internal/model"
)

// These contracts exercise only pure response parsing and command construction.
// They never select an executable, read a credential or invoke a vendor process.
func requireVendorContractFault(t *testing.T, err error, kind fault.Kind, operation string) {
	t.Helper()
	var typed *fault.Error
	if !errors.As(err, &typed) {
		t.Fatal("expected a typed vendor refusal")
	}
	if typed.Kind != kind || typed.Operation != operation {
		t.Fatalf("vendor refusal mismatch: got %s/%s, want %s/%s", typed.Kind, typed.Operation, kind, operation)
	}
	if err.Error() != string(kind)+": "+operation {
		t.Fatal("vendor refusal included information outside the fixed category and operation")
	}
}

func vendorContractResponse(t *testing.T, provider, value string) []byte {
	t.Helper()
	if provider == "onepassword" {
		return []byte(value)
	}
	encoded, err := json.Marshal(map[string]string{
		"projectId":    "synthetic-project",
		"value":        value,
		"revisionDate": "synthetic-revision",
	})
	if err != nil {
		t.Fatal("cannot construct the synthetic vendor envelope")
	}
	return encoded
}

func TestVendorRefusalContractMalformedResponse(t *testing.T) {
	responses := []struct {
		name string
		body string
	}{
		{"empty", ""},
		{"truncated object", `{"projectId":"synthetic-project","value":`},
		{"array instead of object", `[]`},
		{"wrong project type", `{"projectId":1,"value":"synthetic-value"}`},
		{"wrong value type", `{"projectId":"synthetic-project","value":{}}`},
		{"wrong revision type", `{"projectId":"synthetic-project","value":"synthetic-value","revisionDate":false}`},
		{"second document", `{"projectId":"synthetic-project","value":"synthetic-value"} {}`},
	}
	for _, response := range responses {
		t.Run(response.name, func(t *testing.T) {
			source := model.Source{Type: "bitwarden", Project: "synthetic-project", AuditVerified: true}
			value, err := parseVendorOutput(source, model.Binding{}, []byte(response.body))
			requireVendorContractFault(t, err, fault.Trust, "vendor_response")
			if value != (model.Value{}) {
				t.Fatal("invalid vendor envelope exposed value or lifecycle metadata")
			}
		})
	}
}

func TestVendorRefusalContractProjectBindingPrecedesValueSelection(t *testing.T) {
	responses := []struct {
		name string
		body string
	}{
		{"foreign project", `{"projectId":"foreign-project","value":"synthetic-value","revisionDate":"synthetic-revision"}`},
		{"foreign project without value", `{"projectId":"foreign-project"}`},
		{"missing project", `{"value":"synthetic-value"}`},
		{"null envelope", `null`},
	}
	for _, response := range responses {
		t.Run(response.name, func(t *testing.T) {
			source := model.Source{Type: "bitwarden", Project: "synthetic-project", AuditVerified: true}
			value, err := parseVendorOutput(source, model.Binding{Field: "requested"}, []byte(response.body))
			requireVendorContractFault(t, err, fault.Denied, "vendor_project")
			if value != (model.Value{}) {
				t.Fatal("foreign or absent project released a partial result")
			}
		})
	}
}

func TestVendorRefusalContractRequiredValueDistinguishesEmptyFromAbsent(t *testing.T) {
	source := model.Source{Type: "bitwarden", Project: "synthetic-project", AuditVerified: true}
	for _, body := range []string{
		`{"projectId":"synthetic-project","revisionDate":"synthetic-revision"}`,
		`{"projectId":"synthetic-project","value":null,"revisionDate":"synthetic-revision"}`,
	} {
		value, err := parseVendorOutput(source, model.Binding{}, []byte(body))
		requireVendorContractFault(t, err, fault.Missing, "vendor_value")
		if value != (model.Value{}) {
			t.Fatal("missing required value released metadata from an incomplete envelope")
		}
	}
	// A present empty string is a value, not a missing pointer. The revision is
	// only the source's returned metadata, not proof of an honored version pin.
	value, err := parseVendorOutput(source, model.Binding{}, vendorContractResponse(t, "bitwarden", ""))
	if err != nil || value != (model.Value{Version: "synthetic-revision", AuditVerified: true}) {
		t.Fatal("present empty vendor value lost its distinct success contract")
	}
}

func TestVendorRefusalContractSelectedFieldWithholdsDataButRetainsHints(t *testing.T) {
	cases := []struct {
		name      string
		body      string
		kind      fault.Kind
		operation string
	}{
		{"malformed selected document", `{"requested":`, fault.Trust, "json_source"},
		{"missing selected field", `{"unrelated":"synthetic-unselected"}`, fault.Missing, "source_field"},
		{"non-string field", `{"requested":7}`, fault.Trust, "value_type"},
		{"nul in selected field", `{"requested":"synthetic\u0000value"}`, fault.Trust, "value_type"},
		{"duplicate selected field", `{"requested":"first","requested":"second"}`, fault.Trust, "json_duplicate"},
		{"non-object selected document", `[]`, fault.Trust, "json_source"},
	}
	for _, provider := range []string{"bitwarden", "onepassword"} {
		t.Run(provider, func(t *testing.T) {
			for _, item := range cases {
				t.Run(item.name, func(t *testing.T) {
					source := model.Source{Type: provider, Project: "synthetic-project", AuditVerified: true}
					value, err := parseVendorOutput(source, model.Binding{Field: "requested"}, vendorContractResponse(t, provider, item.body))
					requireVendorContractFault(t, err, item.kind, item.operation)
					version := ""
					if provider == "bitwarden" {
						version = "synthetic-revision"
					}
					if value != (model.Value{Version: version, AuditVerified: true}) {
						t.Fatal("field refusal leaked data or changed the leaf adapter's metadata contract")
					}
				})
			}
		})
	}
}

func TestVendorRefusalContractCommandResourceRefusesBeforeSelection(t *testing.T) {
	cases := []struct {
		name      string
		provider  string
		project   string
		resource  string
		kind      fault.Kind
		operation string
	}{
		{"nul before unknown provider", "synthetic-unregistered", "synthetic-project", "id\x00tail", fault.Invalid, "vendor_resource"},
		{"carriage return", "bitwarden", "synthetic-project", "id\rtail", fault.Invalid, "vendor_resource"},
		{"line feed", "onepassword", "synthetic-project", "op://synthetic-project/item/field\n", fault.Invalid, "vendor_resource"},
		{"unknown provider", "synthetic-unregistered", "synthetic-project", "synthetic-id", fault.Unsupported, "vendor_type"},
		{"missing bitwarden project", "bitwarden", "", "synthetic-id", fault.Invalid, "vendor_scope"},
		{"missing bitwarden resource", "bitwarden", "synthetic-project", "", fault.Invalid, "vendor_scope"},
		{"option-looking bitwarden resource", "bitwarden", "synthetic-project", "--synthetic-option", fault.Invalid, "vendor_scope"},
	}
	for _, item := range cases {
		t.Run(item.name, func(t *testing.T) {
			args, key, err := vendorCommand(model.Source{Type: item.provider, Project: item.project}, model.Binding{Resource: item.resource})
			requireVendorContractFault(t, err, item.kind, item.operation)
			if args != nil || key != "" {
				t.Fatal("refused selector retained an executable argument list or credential environment key")
			}
		})
	}
	args, key, err := vendorCommand(model.Source{Type: "bitwarden", Project: "synthetic-project"}, model.Binding{Resource: "synthetic-id", Field: "local-selection"})
	want := []string{"secret", "get", "synthetic-id", "--output", "json"}
	if err != nil || key != "BWS_ACCESS_TOKEN" || !reflect.DeepEqual(args, want) {
		t.Fatal("valid selector did not retain the fixed minimal Bitwarden argv and environment-key contract")
	}
}

func TestVendorRefusalContractOnePasswordRequiresLiteralCompleteSelector(t *testing.T) {
	resources := []struct {
		name     string
		resource string
	}{
		{"missing scheme", "synthetic-project/item/field"},
		{"empty item", "op://synthetic-project//field"},
		{"empty field", "op://synthetic-project/item/"},
		{"additional path segment", "op://synthetic-project/item/field/extra"},
		{"query suffix", "op://synthetic-project/item/field?view=synthetic"},
		{"fragment suffix", "op://synthetic-project/item/field#synthetic"},
		{"scheme case substitution", "OP://synthetic-project/item/field"},
		{"project case substitution", "op://Synthetic-project/item/field"},
	}
	for _, resource := range resources {
		t.Run(resource.name, func(t *testing.T) {
			args, key, err := vendorCommand(model.Source{Type: "onepassword", Project: "synthetic-project"}, model.Binding{Resource: resource.resource})
			requireVendorContractFault(t, err, fault.Denied, "vendor_scope")
			if args != nil || key != "" {
				t.Fatal("ambiguous selector retained a partial vendor command")
			}
		})
	}
	resource := "op://synthetic-project/synthetic-item/synthetic-field"
	args, key, err := vendorCommand(model.Source{Type: "onepassword", Project: "synthetic-project"}, model.Binding{Resource: resource})
	if err != nil || key != "OP_SERVICE_ACCOUNT_TOKEN" || !reflect.DeepEqual(args, []string{"read", resource, "--no-newline"}) {
		t.Fatal("complete selector did not retain the fixed minimal 1Password argv and environment-key contract")
	}
}
