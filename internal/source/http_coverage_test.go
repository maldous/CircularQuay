package source

import (
	"net/url"
	"testing"
	"time"

	"github.com/maldous/CircularQuay/internal/fault"
	"github.com/maldous/CircularQuay/internal/model"
)

func TestCoverageVaultURLRejectsUnsafeSelectors(t *testing.T) {
	source := model.Source{Endpoint: "https://vault.example.test", Mount: "team"}
	binding := model.Binding{Resource: "service/key"}
	for _, version := range []string{"0", "-1", "one"} {
		binding.Version = version
		if _, err := vaultURL(source, binding); !fault.Is(err, fault.Invalid) {
			t.Fatalf("version %q error = %v, want invalid", version, err)
		}
	}
	binding.Version = "7"
	got, err := vaultURL(source, binding)
	if err != nil || got.Path != "/v1/team/data/service/key" || got.Query().Get("version") != "7" {
		t.Fatalf("versioned URL = %v, %v", got, err)
	}
	if _, err := vaultURL(model.Source{Endpoint: "https://vault.example.test"}, binding); !fault.Is(err, fault.Invalid) {
		t.Fatalf("empty mount error = %v, want invalid", err)
	}
}

func TestCoverageVaultValuePreservesLeaseOnlyForPositiveDuration(t *testing.T) {
	result := vaultResult{}
	result.Data.Data = map[string]any{"token": "synthetic"}
	result.Data.Metadata.Version = 4
	result.LeaseID = "lease"
	result.LeaseDuration = 60
	result.Renewable = true
	before := time.Now().UTC()
	value, err := vaultValue(result, model.Source{AuditVerified: true}, model.Binding{Name: "token", Version: "4"})
	if err != nil || value.Data != "synthetic" || !value.Leased || !value.Renewable || !value.AuditVerified || value.ExpiresAt.Before(before) {
		t.Fatalf("leased vault value = %#v, %v", value, err)
	}
	result.LeaseDuration = 0
	value, err = vaultValue(result, model.Source{}, model.Binding{Name: "token"})
	if err != nil || !value.ExpiresAt.IsZero() {
		t.Fatalf("zero-duration lease = %#v, %v", value, err)
	}
	result.Data.Metadata.Destroyed = true
	if _, err = vaultValue(result, model.Source{}, model.Binding{Name: "token"}); !fault.Is(err, fault.Missing) {
		t.Fatalf("destroyed version error = %v, want missing", err)
	}
}

func TestCoverageInfisicalURLAndValueBoundaries(t *testing.T) {
	source := model.Source{Endpoint: "https://infisical.example.test", Project: "project", Environment: "prod", Path: "/app"}
	binding := model.Binding{Resource: "TOKEN", Version: "9"}
	got, err := infisicalURL(source, binding)
	if err != nil {
		t.Fatal(err)
	}
	want := url.Values{"projectId": {"project"}, "environment": {"prod"}, "secretPath": {"/app"}, "expandSecretReferences": {"false"}, "includeImports": {"false"}, "version": {"9"}}
	if got.Path != "/api/v4/secrets/TOKEN" || got.Query().Encode() != want.Encode() {
		t.Fatalf("Infisical URL = %s", got)
	}
	binding.Version = "not-a-version"
	if _, err := infisicalURL(source, binding); !fault.Is(err, fault.Invalid) {
		t.Fatalf("invalid version error = %v, want invalid", err)
	}
	secret := `{"nested":"value"}`
	for _, test := range []struct {
		name string
		raw  *string
		hide bool
		want fault.Kind
	}{
		{"hidden", &secret, true, fault.Denied},
		{"missing", nil, false, fault.Missing},
		{"version mismatch", &secret, false, fault.Trust},
	} {
		t.Run(test.name, func(t *testing.T) {
			binding := model.Binding{Resource: "TOKEN"}
			if test.name == "version mismatch" {
				binding.Version = "8"
			}
			_, err := infisicalValue(test.raw, 9, test.hide, model.Source{}, binding)
			if !fault.Is(err, test.want) {
				t.Fatalf("error = %v, want %s", err, test.want)
			}
		})
	}
}
