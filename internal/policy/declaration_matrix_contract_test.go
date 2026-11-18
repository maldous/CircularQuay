package policy

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/maldous/CircularQuay/internal/fault"
	"github.com/maldous/CircularQuay/internal/model"
)

func TestDeclarationMatrixContractRepositoryCustody(t *testing.T) {
	base := t.TempDir()
	control := model.Repository{URL: "https://example.test/repository.git", Directory: filepath.Join(base, "repository"), Ref: "refs/heads/main", AllowedSignersFile: filepath.Join(base, "signers"), InitialCommit: strings.Repeat("a", 40)}
	assertDeclarationError(t, ValidateRepository(control), "")
	cases := []struct {
		name string
		op   string
		edit func(*model.Repository)
	}{
		{"relative_directory", "repository_paths", func(r *model.Repository) { r.Directory = "relative" }},
		{"relative_token", "repository_token", func(r *model.Repository) { r.TokenFile = "relative" }},
		{"signers_in_repository", "repository_bootstrap_cycle", func(r *model.Repository) { r.AllowedSignersFile = filepath.Join(r.Directory, "signers") }},
		{"token_in_repository", "repository_bootstrap_cycle", func(r *model.Repository) { r.TokenFile = filepath.Join(r.Directory, "token") }},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			candidate := control
			test.edit(&candidate)
			before := candidate
			assertDeclarationError(t, ValidateRepository(candidate), test.op)
			if candidate != before {
				t.Fatal("repository declaration mutated")
			}
		})
	}
}

func TestDeclarationMatrixContractPortableRepositoryPaths(t *testing.T) {
	if !ValidRepositoryPath("data/configuration.json") {
		t.Fatal("portable data path refused")
	}
	for _, path := range []string{"data/name.", "data/.gitmodules", "data/CON.json"} {
		if ValidRepositoryPath(path) {
			t.Fatalf("nonportable path accepted: %q", path)
		}
	}
}

func TestDeclarationMatrixContractRepositoryExecutionReferences(t *testing.T) {
	cases := []struct {
		name string
		op   string
		edit func(*model.Policy)
	}{
		{"source", "repository_source", func(p *model.Policy) { changeSource(p, func(s *model.Source) { s.Path = "repo:values.json" }) }},
		{"bootstrap", "repository_bootstrap", func(p *model.Policy) { changeSource(p, func(s *model.Source) { s.TokenFile = "repo:token" }) }},
		{"directory", "repository_execution_or_cache", func(p *model.Policy) { changeWorkload(p, func(w *model.Workload) { w.Directory = "repo:work" }) }},
		{"command", "repository_execution", func(p *model.Policy) {
			changeWorkload(p, func(w *model.Workload) { w.Command = []string{"repo:program"} })
		}},
		{"destination", "repository_destination", func(p *model.Policy) {
			changeWorkload(p, func(w *model.Workload) {
				w.Targets = map[string]model.Target{"sink": {Type: "file", Path: "repo:destination"}}
			})
		}},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			candidate := declarationPolicy(t)
			test.edit(&candidate)
			assertDeclarationPolicy(t, candidate, test.op)
		})
	}
}

func TestDeclarationMatrixContractProviderDeclarations(t *testing.T) {
	base := t.TempDir()
	token := filepath.Join(base, "token")
	cases := []struct {
		name   string
		source model.Source
		op     string
	}{
		{"env_identity", model.Source{Type: "env", Resources: []string{"VALUE"}, Identity: "managed"}, "source_contract"},
		{"sops_missing_identity", model.Source{Type: "sops", Path: filepath.Join(base, "data"), SOPS: filepath.Join(base, "sops")}, "source_contract"},
		{"doppler", model.Source{Type: "doppler", Endpoint: "https://api.example.test", Project: "project", Environment: "test", Resources: []string{"VALUE"}, Identity: "token_file", TokenFile: token}, ""},
		{"akeyless", model.Source{Type: "akeyless", Endpoint: "https://api.example.test", Resources: []string{"VALUE"}, Identity: "token_file", TokenFile: token}, ""},
		{"aws_ecs", model.Source{Type: "aws-sm", Account: "123456789012", Region: "ap-southeast-2", Resources: []string{"arn:aws:secretsmanager:ap-southeast-2:123456789012:secret:value"}, Identity: "ecs"}, ""},
		{"azure_workload", model.Source{Type: "azure", Endpoint: "https://vault.example.test", Account: "tenant", Profile: "client", Resources: []string{"value"}, Identity: "workload", TokenFile: token}, ""},
		{"azure_missing_token", model.Source{Type: "azure", Endpoint: "https://vault.example.test", Account: "tenant", Profile: "client", Resources: []string{"value"}, Identity: "workload"}, "source_contract"},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			assertDeclarationSource(t, test.source, test.op)
		})
	}
}

func TestDeclarationMatrixContractSOPSAuthority(t *testing.T) {
	base := t.TempDir()
	source := model.Source{Type: "sops", Path: filepath.Join(base, "values"), SOPS: filepath.Join(base, "sops"), AgeIdentityFile: filepath.Join(base, "age")}
	assertDeclarationPolicy(t, policyWithSource(source), "")
	for _, name := range []string{"missing_keys", "duplicate_keys", "mixed_authority", "online_age"} {
		t.Run(name, func(t *testing.T) { assertDeclarationSOPS(t, source, name) })
	}
}

func assertDeclarationSOPS(t *testing.T, source model.Source, name string) {
	t.Helper()
	operation := "vault_key_scope"
	source.VaultAllowlist = []string{"https://vault.example.test"}
	switch name {
	case "duplicate_keys":
		source.VaultKeys = []string{"key", "key"}
		operation = "resource_allowlist"
	case "mixed_authority":
		source.VaultKeys = []string{"key"}
		operation = "mixed_decryption_authority"
	case "online_age":
		source.VaultAllowlist = nil
		operation = "online_decryption_authority"
	}
	candidate := policyWithSource(source)
	if name == "online_age" {
		workload := candidate.Workloads["workload"]
		workload.Mode = "online"
		candidate.Workloads["workload"] = workload
	}
	if name == "online_age" {
		assertDeclarationWorkload(t, candidate, "workload", operation)
	} else {
		assertDeclarationSource(t, source, operation)
	}
}

func TestDeclarationMatrixContractTargetAuthority(t *testing.T) {
	base := t.TempDir()
	systemd := model.Target{Type: "systemd", Path: filepath.Join(base, "target"), AcceptSnapshot: true}
	cases := []struct {
		name    string
		target  model.Target
		publish bool
		op      string
	}{
		{"none", model.Target{Type: "systemd", Path: systemd.Path, Activation: "none", AcceptSnapshot: true}, true, ""},
		{"none_service", model.Target{Type: "systemd", Path: systemd.Path, Activation: "none", Service: "demo", AcceptSnapshot: true}, true, "systemd_target"},
		{"unknown_action", model.Target{Type: "systemd", Path: systemd.Path, Activation: "bogus", Service: "demo", AcceptSnapshot: true}, true, "systemd_target"},
		{"bad_service", model.Target{Type: "systemd", Path: systemd.Path, Activation: "restart", Service: "bad:name", AcceptSnapshot: true}, true, "activation_service"},
		{"snapshot_semantics", model.Target{Type: "file", Path: systemd.Path}, true, "target_snapshot_semantics"},
		{"export", systemd, false, "target_export_permission"},
		{"http", model.Target{Type: "windmill", Endpoint: "http://api.example.test", Project: "workspace", Resource: "f/config", TokenFile: filepath.Join(base, "token"), AcceptSnapshot: true}, true, "target_endpoint"},
		{"default_remote_name", model.Target{Type: "cloudflare", Account: "account", Resource: "worker", TokenFile: filepath.Join(base, "token"), AcceptSnapshot: true}, true, ""},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			candidate := declarationPolicy(t)
			changeWorkload(&candidate, func(w *model.Workload) {
				w.Bindings[0].AllowPublish = test.publish
				w.Targets = map[string]model.Target{"sink": test.target}
			})
			assertDeclarationWorkload(t, candidate, "demo", test.op)
		})
	}
}

func TestDeclarationMatrixContractPermissionBounds(t *testing.T) {
	for _, count := range []int{0, 64, 65} {
		t.Run(fmt.Sprint(count), func(t *testing.T) { assertDeclarationPermissions(t, count) })
	}
}

func assertDeclarationPermissions(t *testing.T, count int) {
	t.Helper()
	candidate := namedIdentityPolicy()
	candidate.Grants = nil
	authority := candidate.MCP.Authorities["workforce"]
	authority.Permissions = nil
	operation := "identity_permissions"
	if count > 0 {
		roles := make([]string, count)
		for index := range roles {
			roles[index] = fmt.Sprintf("Role%d", index)
		}
		authority.Permissions = map[string]model.TokenPermission{"read": {Roles: roles}}
		operation = ""
		if count > 64 {
			operation = "identity_permission_claims"
		}
	}
	candidate.MCP.Authorities["workforce"] = authority
	assertDeclarationError(t, validateIdentityAuthority("workforce", authority), operation)
	if operation != "" {
		operation = "mcp_named_authority"
	}
	assertDeclarationPolicy(t, candidate, operation)
}

func TestDeclarationMatrixContractPublicMetadataRefusals(t *testing.T) {
	assertDeclarationError(t, ValidateMCP(model.MCP{Resource: "https://cq.example.test/mcp"}), "mcp_resource_without_authority")
	legacy := model.MCP{Issuer: "https://issuer.example.test", Audience: "registered", JWKSURL: "https://issuer.example.test/keys"}
	if len(MCPLegacyFingerprint(legacy)) != 64 || MCPLegacyFingerprint(model.MCP{Issuer: legacy.Issuer}) != "" {
		t.Fatal("incomplete legacy authority gained fingerprint")
	}
	data := declarationBytes(t, declarationPolicy(t))
	result, err := DecodeDocument(data, "")
	assertDeclarationError(t, err, "policy_location")
	if !reflect.DeepEqual(result, model.Policy{}) {
		t.Fatal("empty location exposed partial policy")
	}
	candidate := declarationPolicy(t)
	candidate.Version = 0
	assertDeclarationError(t, ValidateResolvedReferences(candidate, filepath.Join(t.TempDir(), "policy.json")), "policy_structure")
}

func TestDeclarationMatrixContractGrantWindow(t *testing.T) {
	candidate := namedIdentityPolicy()
	grant := candidate.Grants[0]
	now := time.Unix(100, 0).UTC()
	principal := Identity{Authority: grant.Authority, Subject: grant.Subject, Client: grant.Client}
	if !AuthorizeIdentityAt(candidate, principal, grant.Workload, "read", grant.Keys[0], now) {
		t.Fatal("valid explicit metadata grant refused")
	}
	candidate.Grants[0].NotBefore = &now
	candidate.Grants[0].ExpiresAt = &now
	if AuthorizeIdentityAt(candidate, principal, grant.Workload, "read", grant.Keys[0], now) {
		t.Fatal("zero-length grant window admitted")
	}
}

func TestDeclarationMatrixContractSnapshotAlphabet(t *testing.T) {
	candidate := declarationPolicy(t)
	base := t.TempDir()
	changeWorkload(&candidate, func(w *model.Workload) {
		w.Snapshot = model.Snapshot{Path: filepath.Join(base, "snapshot"), IdentityFile: filepath.Join(base, "age"), SOPS: filepath.Join(base, "sops"), Recipients: []string{"age1" + strings.Repeat("q", 57) + "!"}}
	})
	assertDeclarationWorkload(t, candidate, "demo", "snapshot_declaration")
}

func declarationPolicy(t *testing.T) model.Policy {
	t.Helper()
	candidate := validPolicy()
	changeSource(&candidate, func(s *model.Source) { s.Path = filepath.Join(t.TempDir(), "values.json") })
	return candidate
}

func declarationBytes(t *testing.T, value any) []byte {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func assertDeclarationPolicy(t *testing.T, candidate model.Policy, operation string) {
	t.Helper()
	before := declarationBytes(t, candidate)
	assertDeclarationError(t, Validate(candidate), operation)
	if !bytes.Equal(before, declarationBytes(t, candidate)) {
		t.Fatal("pure validation mutated declaration")
	}
}

func assertDeclarationError(t *testing.T, err error, operation string) {
	t.Helper()
	if operation == "" {
		if err != nil {
			t.Fatalf("valid declaration refused: %v", err)
		}
		return
	}
	actual, ok := err.(*fault.Error)
	if !ok || actual.Kind != fault.Invalid || actual.Operation != operation || err.Error() != "invalid_policy: "+operation || errors.Unwrap(err) != nil {
		t.Fatalf("unexpected declaration refusal: %v", err)
	}
}

// Pair the caller's deliberately generalized refusal with its actual leaf
// contract, so the oracle distinguishes the intended guard from an earlier one.
func assertDeclarationSource(t *testing.T, source model.Source, operation string) {
	t.Helper()
	assertDeclarationError(t, validateSource(source), operation)
	if operation != "" {
		operation = "source_declaration"
	}
	assertDeclarationPolicy(t, policyWithSource(source), operation)
}

func assertDeclarationWorkload(t *testing.T, candidate model.Policy, name, operation string) {
	t.Helper()
	assertDeclarationError(t, validateWorkload(candidate, candidate.Workloads[name]), operation)
	if operation != "" {
		operation = "workload_declaration"
	}
	assertDeclarationPolicy(t, candidate, operation)
}
