package source

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/maldous/CircularQuay/internal/fault"
	"github.com/maldous/CircularQuay/internal/model"
	"github.com/maldous/CircularQuay/internal/platform"
	"github.com/maldous/CircularQuay/internal/transport"
)

func TestProviderInputBoundaryContractAzureRefusesBeforeCredentialConstruction(t *testing.T) {
	missing := filepath.Join(privateSourceDirectory(t, "azure-input"), "missing-assertion")
	credential, err := azureCredential(model.Source{Account: "synthetic-tenant", Profile: "synthetic-client", Identity: "workload", TokenFile: missing})
	if credential != nil {
		t.Fatal("missing assertion returned a credential")
	}
	requireProviderInputError(t, err, fault.Trust, "protected path permissions")
	credential, err = azureCredential(model.Source{Account: "synthetic-tenant", Identity: "managed"})
	if credential != nil {
		t.Fatal("missing explicit managed identity returned a credential")
	}
	requireProviderInputError(t, err, fault.Invalid, "azure_managed_identity")
	assertProviderInputAbsent(t, missing)
}

func TestProviderInputBoundaryContractSOPSVersionPrecedesSourceRead(t *testing.T) {
	missing := filepath.Join(privateSourceDirectory(t, "sops-version"), "missing.json")
	config := model.Source{Type: "sops", Path: missing, SOPS: "unused-helper"}
	value, err := (SOPSReader{}).Read(context.Background(), config, model.Binding{Name: "KEY", Version: "unsupported"})
	requireProviderInputValueRefusal(t, value, err, fault.Invalid, "source_version_unsupported")
	assertProviderInputAbsent(t, missing)
}

func TestProviderInputBoundaryContractSOPSResourceFallbackDoesNotChooseAnotherPath(t *testing.T) {
	missing := filepath.Join(privateSourceDirectory(t, "sops-resource"), "missing.json")
	value, err := (SOPSReader{}).Read(context.Background(), model.Source{Type: "sops", SOPS: "unused-helper"}, model.Binding{Name: "KEY", Resource: missing})
	requireProviderInputValueRefusal(t, value, err, fault.Missing, "read protected file")
	assertProviderInputAbsent(t, missing)
}

func TestProviderInputBoundaryContractSnapshotRejectsNULBeforeHelperOrWrite(t *testing.T) {
	directory := privateSourceDirectory(t, "snapshot-input")
	config := model.Snapshot{Path: filepath.Join(directory, "snapshot.json"), SOPS: filepath.Join(directory, "missing-helper"), Recipients: []string{"age1synthetic"}}
	values := map[string]string{"KEY": "SYNTHETIC\x00VALUE"}
	err := EncryptSnapshot(context.Background(), config, values)
	requireProviderInputError(t, err, fault.Invalid, "invalid snapshot string")
	if !reflect.DeepEqual(values, map[string]string{"KEY": "SYNTHETIC\x00VALUE"}) {
		t.Fatal("refused snapshot mutated caller input")
	}
	assertProviderInputAbsent(t, config.Path)
	assertProviderInputAbsent(t, config.SOPS)
}

func TestProviderInputBoundaryContractTransitRecipientMustBeExplicitlyListed(t *testing.T) {
	directory := privateSourceDirectory(t, "transit-recipient")
	file := filepath.Join(directory, "encrypted.json")
	data := []byte(`{"sops":{"mac":"synthetic","hc_vault":[{"vault_address":"https://vault.example.test","engine_path":"transit","key_name":"unlisted"}]}}`)
	writeProviderInputFixture(t, file, data)
	config := model.Source{
		Type: "sops", Path: file, Identity: "transit", SOPS: filepath.Join(directory, "missing-helper"),
		TokenFile: filepath.Join(directory, "missing-token"), VaultAllowlist: []string{"https://vault.example.test"}, VaultKeys: []string{"transit/listed"},
	}
	value, err := (SOPSReader{}).Read(context.Background(), config, model.Binding{Name: "KEY"})
	requireProviderInputValueRefusal(t, value, err, fault.Trust, "unapproved Transit recipient")
	assertProviderInputBytes(t, file, data)
	assertProviderInputAbsent(t, config.TokenFile)
	assertProviderInputAbsent(t, config.SOPS)
}

func TestProviderInputBoundaryContractTransitTokenReadFailureReturnsNoToken(t *testing.T) {
	missing := filepath.Join(privateSourceDirectory(t, "transit-token"), "missing-token")
	token, err := transitToken(missing, []string{"https://vault.example.test"})
	if token != "" {
		t.Fatal("failed protected token read returned token bytes")
	}
	requireProviderInputError(t, err, fault.Missing, "read protected file")
	assertProviderInputAbsent(t, missing)
}

func TestProviderInputBoundaryContractVendorCredentialFailurePrecedesLaunch(t *testing.T) {
	directory := privateSourceDirectory(t, "vendor-input")
	executable := filepath.Join(directory, "inert-vendor")
	want := []byte("SYNTHETIC_INERT_NOT_AN_EXECUTABLE")
	writeProviderInputFixture(t, executable, want)
	if err := validateVendorExecutable(executable); err != nil {
		t.Fatal("owned regular-file metadata control refused", err)
	}
	config := model.Source{Type: "bitwarden", Path: executable, Project: "synthetic-project", TokenFile: filepath.Join(directory, "missing-token")}
	value, err := readVendor(context.Background(), config, model.Binding{Resource: "synthetic-id"})
	requireProviderInputValueRefusal(t, value, err, fault.Denied, "credential_file")
	assertProviderInputBytes(t, executable, want)
	assertProviderInputAbsent(t, config.TokenFile)
}

func TestProviderInputBoundaryContractVendorRejectsShellSuffixes(t *testing.T) {
	for _, name := range []string{"inert.cmd", "inert.BAT"} {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(privateSourceDirectory(t, "vendor-suffix"), name)
			want := []byte("SYNTHETIC_INERT_NOT_A_SCRIPT")
			writeProviderInputFixture(t, path, want)
			requireProviderInputError(t, validateVendorExecutable(path), fault.Trust, "vendor_executable")
			assertProviderInputBytes(t, path, want)
		})
	}
}

func TestProviderInputBoundaryContractExplicitAbsentDotenvNeverUsesEnvironment(t *testing.T) {
	missing := filepath.Join(privateSourceDirectory(t, "dotenv-input"), "missing.env")
	value, err := readEnvironment(context.Background(), model.Source{Type: "env", Path: missing, AuditVerified: true}, model.Binding{Resource: "SYNTHETIC_PROVIDER_INPUT"})
	requireProviderInputValueRefusal(t, value, err, fault.Missing, "read protected file")
	assertProviderInputAbsent(t, missing)
}

func TestProviderInputBoundaryContractRawSelectionRefusesNUL(t *testing.T) {
	value, err := selectValue("SYNTHETIC_RAW_VALUE", "")
	if err != nil || value != "SYNTHETIC_RAW_VALUE" {
		t.Fatal("unselected raw string control changed")
	}
	value, err = selectValue("SYNTHETIC\x00RAW_VALUE", "")
	if value != "" {
		t.Fatal("invalid raw string escaped selection")
	}
	requireProviderInputError(t, err, fault.Trust, "value_type")
}

func TestProviderInputBoundaryContractDopplerEndpointRefusalPrecedesHTTP(t *testing.T) {
	calls := 0
	client := transport.Client{HTTP: &http.Client{Transport: cloudRoundTripper(func(*http.Request) (*http.Response, error) {
		calls++
		t.Fatal("invalid endpoint selected HTTP")
		return nil, errors.New("SYNTHETIC_UNEXPECTED_HTTP")
	})}}
	for _, endpoint := range []string{"http://doppler.example.test", "https://user:SYNTHETIC_PRIVATE@doppler.example.test", "https://doppler.example.test?token=SYNTHETIC_PRIVATE"} {
		config := model.Source{Type: "doppler", Endpoint: endpoint, Project: "synthetic-project", Environment: "synthetic-environment"}
		value, err := readDoppler(context.Background(), client, config, model.Binding{Resource: "KEY"}, "SYNTHETIC_PRIVATE_TOKEN")
		requireProviderInputValueRefusal(t, value, err, fault.Trust, "provider_endpoint")
	}
	if calls != 0 {
		t.Fatal("endpoint refusal reached HTTP")
	}
}

func TestProviderInputBoundaryContractCanonicalEndpointNeverAcceptsURLSecrets(t *testing.T) {
	const origin = "https://provider.example.test/base"
	endpoint, err := CanonicalEndpoint(origin)
	if err != nil || endpoint == nil || endpoint.String() != origin {
		t.Fatal("valid explicit HTTPS endpoint was not preserved")
	}
	for _, raw := range []string{"http://provider.example.test", "https://user:SYNTHETIC_PRIVATE@provider.example.test", origin + "?token=SYNTHETIC_PRIVATE", origin + "#SYNTHETIC_PRIVATE"} {
		endpoint, err := CanonicalEndpoint(raw)
		if endpoint != nil {
			t.Fatal("invalid endpoint returned a partial URL")
		}
		requireProviderInputError(t, err, fault.Trust, "provider_endpoint")
	}
}

func writeProviderInputFixture(t *testing.T, path string, data []byte) {
	t.Helper()
	if err := platform.WriteProtected(path, data); err != nil {
		t.Fatal(err)
	}
}

func assertProviderInputBytes(t *testing.T, path string, want []byte) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(data, want) {
		t.Fatal("refusal changed owned synthetic input bytes")
	}
}

func assertProviderInputAbsent(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Lstat(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("pre-effect refusal created an owned fixture path")
	}
}

func requireProviderInputValueRefusal(t *testing.T, value model.Value, err error, kind fault.Kind, operation string) {
	t.Helper()
	requireProviderInputError(t, err, kind, operation)
	if value != (model.Value{}) {
		t.Fatal("provider input refusal returned a partial value or metadata")
	}
}

func requireProviderInputError(t *testing.T, err error, kind fault.Kind, operation string) {
	t.Helper()
	var typed *fault.Error
	if !errors.As(err, &typed) {
		t.Fatalf("missing typed provider input refusal: %v", err)
	}
	if err != typed || typed.Kind != kind || typed.Operation != operation || err.Error() != string(kind)+": "+operation {
		t.Fatalf("provider input refusal changed category or disclosure boundary: %v", err)
	}
}
