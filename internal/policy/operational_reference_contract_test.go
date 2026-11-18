package policy

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/maldous/CircularQuay/internal/model"
	"gopkg.in/yaml.v3"
)

func TestOperationalReferenceContractDisabledAndYAMLScalars(t *testing.T) {
	candidate := declarationPolicy(t)
	candidate.Observability = &model.Observability{Enabled: boolValue(false)}
	assertDeclarationPolicy(t, candidate, "")
	candidate.Observability = validObservability()
	candidate.Observability.Traces.SampleRatio = floatValue(1)
	data, err := yaml.Marshal(candidate)
	if err != nil {
		t.Fatal(err)
	}
	before := append([]byte(nil), data...)
	loaded, err := Parse(data, filepath.Join(t.TempDir(), "policy.yaml"))
	if err != nil || !reflect.DeepEqual(loaded, candidate) || !bytes.Equal(data, before) {
		t.Fatalf("integer YAML sample ratio did not preserve valid policy: %v", err)
	}
}

func TestOperationalReferenceContractYAMLFieldRefusals(t *testing.T) {
	for _, fields := range []map[string]any{{"enabled": false, "unexpected": "synthetic"}, {"service_name": "cq"}} {
		document := policyJSONValue(t, declarationPolicy(t))
		document["observability"] = fields
		data, err := yaml.Marshal(document)
		if err != nil {
			t.Fatal(err)
		}
		before := append([]byte(nil), data...)
		result, err := Parse(data, filepath.Join(t.TempDir(), "policy.yaml"))
		assertDeclarationError(t, err, "policy_yaml")
		if !reflect.DeepEqual(result, model.Policy{}) || !bytes.Equal(data, before) {
			t.Fatal("YAML refusal mutated or exposed policy")
		}
	}
}

func TestOperationalReferenceContractOwnedFiles(t *testing.T) {
	for _, updates := range []bool{false, true} {
		for _, state := range []string{"valid_token", "missing_token", "invalid_ca"} {
			t.Run(state+map[bool]string{false: "_telemetry", true: "_updates"}[updates], func(t *testing.T) {
				assertOperationalOwnedFiles(t, updates, state)
			})
		}
	}
}

func assertOperationalOwnedFiles(t *testing.T, updates bool, state string) {
	t.Helper()
	root := privatePolicyDirectory(t)
	token := filepath.Join(root, "token")
	ca := filepath.Join(root, "ca.pem")
	tokenBytes := []byte("synthetic-owned-token-not-a-credential")
	caBytes := []byte("synthetic-invalid-public-CA")
	if err := os.WriteFile(token, tokenBytes, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(ca, caBytes, 0o600); err != nil {
		t.Fatal(err)
	}
	candidate := operationalReferencePolicy(t, updates, token, ca, state, caBytes)
	file := filepath.Join(root, "policy.json")
	data := declarationBytes(t, candidate)
	if _, err := Parse(data, file); err != nil {
		t.Fatalf("fixture failed pure admission: %v", err)
	}
	if err := os.WriteFile(file, data, 0o600); err != nil {
		t.Fatal(err)
	}
	operation := operationalExpectedOperation(updates, state)
	assertDeclarationError(t, ValidateLoadedOperationalReferences(candidate), operation)
	loaded, err := Load(file)
	assertDeclarationError(t, err, operation)
	assertOperationalLoadedResult(t, loaded, candidate, operation)
	for path, want := range map[string][]byte{file: data, token: tokenBytes, ca: caBytes} {
		actual, err := os.ReadFile(path)
		if err != nil || !bytes.Equal(actual, want) {
			t.Fatalf("owned reference changed: %v", err)
		}
	}
}

func operationalReferencePolicy(t *testing.T, updates bool, token, ca, state string, caBytes []byte) model.Policy {
	t.Helper()
	candidate := declarationPolicy(t)
	trustPath, trustDigest := "", ""
	if state == "missing_token" {
		token += ".absent"
	}
	if state == "invalid_ca" {
		trustPath = ca
		digest := sha256.Sum256(caBytes)
		trustDigest = hex.EncodeToString(digest[:])
	}
	if updates {
		candidate.Updates = validUpdates()
		candidate.Updates.TokenFile, candidate.Updates.TrustStoreFile, candidate.Updates.TrustStoreSHA256 = token, trustPath, trustDigest
	} else {
		candidate.Observability = validObservability()
		candidate.Observability.TokenFile, candidate.Observability.TrustStoreFile, candidate.Observability.TrustStoreSHA256 = token, trustPath, trustDigest
	}
	return candidate
}

func operationalExpectedOperation(updates bool, state string) string {
	if state == "valid_token" {
		return ""
	}
	prefix := "observability_"
	if updates {
		prefix = "updates_"
	}
	if state == "missing_token" {
		return prefix + "token_file"
	}
	return prefix + "trust_store"
}

func assertOperationalLoadedResult(t *testing.T, loaded, expected model.Policy, operation string) {
	t.Helper()
	if operation != "" {
		expected = model.Policy{}
	}
	if !reflect.DeepEqual(loaded, expected) {
		t.Fatal("loaded reference result was partial or changed declaration")
	}
}

func TestOperationalReferenceContractMutableRootAlias(t *testing.T) {
	candidate := declarationPolicy(t)
	candidate.Observability = validObservability()
	file := filepath.Join(t.TempDir(), "policy.json")
	candidate.Observability.TokenFile = file
	before := declarationBytes(t, candidate)
	assertDeclarationError(t, ValidateResolvedReferences(candidate, file), "operational_protected_path")
	if !bytes.Equal(before, declarationBytes(t, candidate)) {
		t.Fatal("ownership check mutated declaration")
	}
}

func TestOperationalReferenceContractIdentityTrustRefusal(t *testing.T) {
	candidate := namedIdentityPolicy()
	candidate.Grants = nil
	root := privatePolicyDirectory(t)
	ca := filepath.Join(root, "invalid-ca.pem")
	data := []byte("synthetic-invalid-public-CA")
	if err := os.WriteFile(ca, data, 0o600); err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(data)
	authority := candidate.MCP.Authorities["workforce"]
	authority.TrustStoreFile, authority.TrustStoreSHA256 = ca, hex.EncodeToString(digest[:])
	candidate.MCP.Authorities["workforce"] = authority
	assertDeclarationPolicy(t, candidate, "")
	assertDeclarationError(t, ValidateResolvedReferences(candidate, filepath.Join(root, "policy.json")), "identity_trust_store")
	actual, err := os.ReadFile(ca)
	if err != nil || !bytes.Equal(actual, data) {
		t.Fatal("trust refusal changed owned public CA bytes")
	}
}
