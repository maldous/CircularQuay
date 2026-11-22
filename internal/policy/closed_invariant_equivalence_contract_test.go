package policy

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/maldous/CircularQuay/internal/fault"
	"github.com/maldous/CircularQuay/internal/model"
)

func TestClosedInvariantPolicyNamedFingerprintKeepsExactProjection(t *testing.T) {
	config := namedIdentityPolicy().MCP
	authority := config.Authorities["workforce"]
	authority.Clients = append(authority.Clients, "λ\x00synthetic")
	config.Authorities["workforce"] = authority
	encoded, err := json.Marshal(struct {
		Resource  string
		Name      string
		Authority model.IdentityAuthority
	}{config.Resource, "workforce", authority})
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(encoded)
	if MCPAuthorityFingerprint(config, "workforce") != hex.EncodeToString(digest[:]) || MCPAuthorityFingerprint(config, "absent") != "" {
		t.Fatal("named identity hash or absent-authority guard changed")
	}
}

func TestClosedInvariantPolicyLegacyFingerprintKeepsExactProjection(t *testing.T) {
	config := model.MCP{Issuer: "https://identity.example.invalid", JWKSURL: "https://identity.example.invalid/keys", Audience: "λ\x00synthetic"}
	encoded, err := json.Marshal(struct {
		Issuer   string
		JWKSURL  string
		Audience string
	}{config.Issuer, config.JWKSURL, config.Audience})
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(encoded)
	if MCPLegacyFingerprint(config) != hex.EncodeToString(digest[:]) {
		t.Fatal("legacy identity hash encoding changed")
	}
	config.Audience = ""
	if MCPLegacyFingerprint(config) != "" {
		t.Fatal("incomplete legacy contract received a fingerprint")
	}
}

func TestClosedInvariantPolicyJSONAdmissionRetainsOriginalRefusals(t *testing.T) {
	for _, data := range []string{`{"version":1,"version":1}`, `{"version":1} {}`, `{"version":1,"sources":null,"workloads":{}}`, `{"version":1,"sources":{},"workloads":{},"observability":{"enabled":true,"traces":{"enabled":true,"sample_ratio":1e400}}}`} {
		result, err := DecodeDocument([]byte(data), filepath.Join(t.TempDir(), "policy.json"))
		if !fault.Is(err, fault.Invalid) || !reflect.DeepEqual(result, model.Policy{}) {
			t.Fatal("strict document refusal returned a partial policy")
		}
	}
	candidate := validPolicy()
	candidate.Sources["fixture"] = model.Source{Type: "json", Path: filepath.Join(t.TempDir(), "values.json")}
	encoded, err := json.Marshal(candidate)
	if err != nil {
		t.Fatal(err)
	}
	got, err := DecodeDocument(encoded, filepath.Join(t.TempDir(), "policy.json"))
	if err != nil || !reflect.DeepEqual(got, candidate) {
		t.Fatal("single generic projection changed typed policy admission")
	}
}

// The following functions have their own defensive typed-input contracts. The
// public validator normally refuses the same invalid declaration earlier.
func TestClosedInvariantPolicySignalHelpersKeepNilContracts(t *testing.T) {
	if validMetricsSignal(model.MetricsSignal{}) || validTracesSignal(model.TracesSignal{}) {
		t.Fatal("missing typed signal enablement was accepted")
	}
	enabled := false
	if !validMetricsSignal(model.MetricsSignal{Enabled: &enabled}) || !validTracesSignal(model.TracesSignal{Enabled: &enabled}) {
		t.Fatal("explicit disabled-signal positive control refused")
	}
}

func TestClosedInvariantPolicyTrustStoreKeepsCanonicalDigestContract(t *testing.T) {
	path := filepath.Join(t.TempDir(), "roots.pem")
	valid := model.IdentityAuthority{TrustStoreFile: path, TrustStoreSHA256: strings.Repeat("a", 64)}
	if !validTrustStoreDeclaration(valid) {
		t.Fatal("paired canonical declaration refused without I/O")
	}
	for _, digest := range []string{strings.Repeat("A", 64), strings.Repeat("g", 64), "short"} {
		valid.TrustStoreSHA256 = digest
		if validTrustStoreDeclaration(valid) || lowerHexDigest(digest) {
			t.Fatal("noncanonical digest accepted")
		}
	}
}

func TestClosedInvariantPolicySourceHelpersKeepIdentityAndAuthorityRefusals(t *testing.T) {
	assertDeclarationError(t, requireSourcePath(model.Source{Type: "json"}), "source_contract")
	assertDeclarationError(t, validateSourceIdentity(model.Source{Identity: "unsupported"}), "source_identity")
	assertDeclarationError(t, validateSourceIdentity(model.Source{Identity: "token_file"}), "source_identity")
	assertDeclarationError(t, validateVaultAuthority(model.Source{Type: "env", VaultAllowlist: []string{"https://vault.example.invalid"}}), "vault_allowlist")
	if err := validateSourceIdentity(model.Source{Identity: "none"}); err != nil {
		t.Fatal("no-authority positive control refused")
	}
}

func TestClosedInvariantPolicyTargetHelpersKeepTypedRefusals(t *testing.T) {
	assertDeclarationError(t, validateTargetContract(model.Target{Type: "unsupported"}), "target_type")
	assertDeclarationError(t, validateActivation(model.Target{Type: "systemd", Activation: "unsupported"}), "activation_action")
	assertDeclarationError(t, validateActivation(model.Target{Type: "file", Service: "worker", Activation: "restart"}), "activation_scope")
	assertDeclarationError(t, validateTargets(model.Workload{Mode: "online", Snapshot: model.Snapshot{Path: "owned-snapshot"}}), "online_snapshot")
	if err := validateActivation(model.Target{Type: "systemd", Service: "worker", Activation: "restart"}); err != nil {
		t.Fatal("typed native declaration positive control refused")
	}
}

func TestClosedInvariantPolicyPrivateHelperKeepsPublicationRefusal(t *testing.T) {
	workload := model.Workload{Targets: map[string]model.Target{"destination": {Type: "file"}}}
	before := workload.Targets["destination"]
	err := validatePrivateWorkload(model.Policy{}, "worker", workload, "owned-root")
	assertDeclarationError(t, err, "configuration_private_target")
	if workload.Targets["destination"] != before {
		t.Fatal("private publication refusal mutated declaration")
	}
}

func TestClosedInvariantPolicyRemoteKeysKeepUnsupportedAndURLRefusals(t *testing.T) {
	if remoteOwnershipKey(model.Target{Type: "unsupported"}) != "" || endpointKey("https://[invalid", "") != "" {
		t.Fatal("unsupported remote key or malformed URL acquired an identity")
	}
	if endpointKey("HTTPS://EXAMPLE.INVALID/path/", "") != "https://example.invalid/path" {
		t.Fatal("canonical remote identity changed")
	}
}
