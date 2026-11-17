package source

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/maldous/CircularQuay/internal/fault"
	"github.com/maldous/CircularQuay/internal/model"
)

func TestVendorAdmissionContractDeclaredScopeBeforeIdentity(t *testing.T) {
	cases := []struct {
		name        string
		provider    string
		project     string
		environment string
		resource    string
		kind        fault.Kind
		operation   string
	}{
		{"cross-vault", "onepassword", "permitted", "", "op://other/item/field", fault.Denied, "vendor_scope"},
		{"missing-vault", "onepassword", "", "", "op://permitted/item/field", fault.Denied, "vendor_scope"},
		{"wrong-scheme", "onepassword", "permitted", "", "https://permitted/item/field", fault.Denied, "vendor_scope"},
		{"incomplete-reference", "onepassword", "permitted", "", "op://permitted/item", fault.Denied, "vendor_scope"},
		{"query-reference", "onepassword", "permitted", "", "op://permitted/item/field?query", fault.Denied, "vendor_scope"},
		{"fragment-reference", "onepassword", "permitted", "", "op://permitted/item/field#fragment", fault.Denied, "vendor_scope"},
		{"unsupported-environment", "onepassword", "permitted", "unsupported", "op://permitted/item/field", fault.Unsupported, "onepassword_environments"},
		{"missing-project", "bitwarden", "", "", "synthetic-id", fault.Invalid, "vendor_scope"},
		{"option-resource", "bitwarden", "permitted", "", "--synthetic-option", fault.Invalid, "vendor_scope"},
		{"nul-resource", "bitwarden", "permitted", "", "synthetic\x00id", fault.Invalid, "vendor_resource"},
		{"cr-resource", "bitwarden", "permitted", "", "synthetic\rid", fault.Invalid, "vendor_resource"},
		{"lf-resource", "onepassword", "permitted", "", "op://permitted/item/field\n", fault.Invalid, "vendor_resource"},
	}
	registry := NewRegistry()
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			source := vendorAdmissionMissingIdentity(t, tc.provider)
			source.Project = tc.project
			source.Environment = tc.environment
			// Exact declaration passes Registry's resource gate, not the adapter's
			// additional vault/project/selector restrictions.
			source.Resources = []string{tc.resource}
			value, err := registry.Read(context.Background(), source, model.Binding{Resource: tc.resource})
			requireVendorAdmissionRefusal(t, value, err, tc.kind, tc.operation)
		})
	}
}

func TestVendorAdmissionContractLeafRefusalBeforeIdentity(t *testing.T) {
	cases := []struct {
		name      string
		provider  string
		resource  string
		kind      fault.Kind
		operation string
	}{
		{"empty-bitwarden-resource", "bitwarden", "", fault.Invalid, "vendor_scope"},
		{"unregistered-provider", "unsupported-synthetic", "synthetic-id", fault.Unsupported, "vendor_type"},
		{"control-before-provider", "unsupported-synthetic", "synthetic\x00id", fault.Invalid, "vendor_resource"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			source := vendorAdmissionMissingIdentity(t, tc.provider)
			source.Project = "permitted"
			// These are leaf-adapter defenses; the default Registry refuses an
			// empty resource or an unregistered provider before dispatch.
			value, err := readVendor(context.Background(), source, model.Binding{Resource: tc.resource})
			requireVendorAdmissionRefusal(t, value, err, tc.kind, tc.operation)
		})
	}
}

func TestVendorAdmissionContractVersionRemainsFirst(t *testing.T) {
	registry := NewRegistry()
	for _, provider := range []string{"bitwarden", "onepassword"} {
		t.Run(provider, func(t *testing.T) {
			source := vendorAdmissionMissingIdentity(t, provider)
			source.Resources = []string{"synthetic-invalid-selector"}
			binding := model.Binding{Resource: "synthetic-invalid-selector", Version: "unsupported-version"}
			value, err := readVendor(context.Background(), source, binding)
			requireVendorAdmissionRefusal(t, value, err, fault.Invalid, "source_version_unsupported")
			value, err = registry.Read(context.Background(), source, binding)
			requireVendorAdmissionRefusal(t, value, err, fault.Invalid, "source_version_unsupported")
		})
	}
}

func TestVendorAdmissionContractRegistryScopeRemainsFirst(t *testing.T) {
	registry := NewRegistry()
	for _, provider := range []string{"bitwarden", "onepassword"} {
		t.Run(provider, func(t *testing.T) {
			source := vendorAdmissionMissingIdentity(t, provider)
			source.Resources = []string{"only-declared-resource"}
			binding := model.Binding{Resource: "undeclared-resource", Version: "unsupported-version"}
			value, err := registry.Read(context.Background(), source, binding)
			requireVendorAdmissionRefusal(t, value, err, fault.Denied, "source_scope")
		})
	}
}

func TestVendorAdmissionContractAdmittedSelectorStillRequiresExecutable(t *testing.T) {
	registry := NewRegistry()
	for _, tc := range []struct{ provider, resource string }{
		{"bitwarden", "synthetic-id"},
		{"onepassword", "op://permitted/item/field"},
	} {
		t.Run(tc.provider, func(t *testing.T) {
			source := vendorAdmissionMissingIdentity(t, tc.provider)
			source.Project = "permitted"
			source.Resources = []string{tc.resource}
			binding := model.Binding{Resource: tc.resource}
			value, err := registry.Read(context.Background(), source, binding)
			requireVendorAdmissionRefusal(t, value, err, fault.Trust, "vendor_executable")
			// A relative path must still fail its pure executable-shape check,
			// without consulting the missing credential or launching anything.
			source.Path = "relative-synthetic-vendor"
			value, err = registry.Read(context.Background(), source, binding)
			requireVendorAdmissionRefusal(t, value, err, fault.Invalid, "vendor_executable")
		})
	}
}

func vendorAdmissionMissingIdentity(t *testing.T, provider string) model.Source {
	t.Helper()
	// Both children are deliberately absent from a new private synthetic fixture.
	// On the old ordering even a denied selector yields vendor_executable here.
	directory := t.TempDir()
	return model.Source{
		Type:          provider,
		Path:          filepath.Join(directory, "missing-vendor"),
		Identity:      "token_file",
		TokenFile:     filepath.Join(directory, "missing-bootstrap-token"),
		AuditVerified: true,
	}
}

func requireVendorAdmissionRefusal(t *testing.T, value model.Value, err error, kind fault.Kind, operation string) {
	t.Helper()
	var typed *fault.Error
	if !errors.As(err, &typed) || typed.Kind != kind || typed.Operation != operation {
		t.Fatalf("admission error = %v, want %s: %s", err, kind, operation)
	}
	if err.Error() != string(kind)+": "+operation {
		t.Fatalf("admission error contains unexpected detail: %v", err)
	}
	if value != (model.Value{}) {
		t.Fatal("denied admission returned a value or metadata")
	}
}
