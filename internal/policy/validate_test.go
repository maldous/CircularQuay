package policy

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/maldous/CircularQuay/internal/fault"
	"github.com/maldous/CircularQuay/internal/model"
)

func validPolicy() model.Policy {
	return model.Policy{
		Version: 1,
		Sources: map[string]model.Source{"fixture": {Type: "json", Path: "/fixtures/values.json"}},
		Workloads: map[string]model.Workload{"demo": {
			Mode: "offline", Bindings: []model.Binding{{Name: "DEMO_MESSAGE", Source: "fixture", Field: "message", AllowOffline: true}},
		}},
	}
}

func changeWorkload(policy *model.Policy, edit func(*model.Workload)) {
	workload := policy.Workloads["demo"]
	edit(&workload)
	policy.Workloads["demo"] = workload
}

func changeSource(policy *model.Policy, edit func(*model.Source)) {
	source := policy.Sources["fixture"]
	edit(&source)
	policy.Sources["fixture"] = source
}

func TestPolicyRejectsUnsafeDeclarations(t *testing.T) {
	tests := []struct {
		name string
		edit func(*model.Policy)
	}{
		{"version", func(p *model.Policy) { p.Version = 2 }},
		{"empty_sources", func(p *model.Policy) { p.Sources = nil }},
		{"empty_workloads", func(p *model.Policy) { p.Workloads = nil }},
		{"source_name", func(p *model.Policy) { p.Sources["bad/name"] = p.Sources["fixture"] }},
		{"source_type", func(p *model.Policy) { changeSource(p, func(s *model.Source) { s.Type = "exec" }) }},
		{"source_path", func(p *model.Policy) { changeSource(p, func(s *model.Source) { s.Path = "" }) }},
		{"source_control", func(p *model.Policy) { changeSource(p, func(s *model.Source) { s.Path = "a\x00b" }) }},
		{"source_http", func(p *model.Policy) { changeSource(p, func(s *model.Source) { s.Endpoint = "http://example.test" }) }},
		{"source_url_credentials", func(p *model.Policy) {
			changeSource(p, func(s *model.Source) { s.Endpoint = "https://user:password@example.test" })
		}},
		{"source_query", func(p *model.Policy) {
			changeSource(p, func(s *model.Source) { s.Endpoint = "https://example.test?token=example" })
		}},
		{"source_fragment", func(p *model.Policy) {
			changeSource(p, func(s *model.Source) { s.Endpoint = "https://example.test#part" })
		}},
		{"source_missing_endpoint", func(p *model.Policy) {
			changeSource(p, func(s *model.Source) { s.Type = "openbao"; s.Resources = []string{"demo"} })
		}},
		{"source_missing_scope", func(p *model.Policy) { changeSource(p, func(s *model.Source) { s.Type = "doppler" }) }},
		{"source_wildcard", func(p *model.Policy) { changeSource(p, func(s *model.Source) { s.Resources = []string{"*"} }) }},
		{"source_duplicate", func(p *model.Policy) {
			changeSource(p, func(s *model.Source) { s.Resources = []string{"demo", "demo"} })
		}},
		{"source_traversal", func(p *model.Policy) {
			changeSource(p, func(s *model.Source) { s.Resources = []string{"../bootstrap"} })
		}},
		{"source_identity", func(p *model.Policy) { changeSource(p, func(s *model.Source) { s.Identity = "guess" }) }},
		{"source_profile", func(p *model.Policy) { changeSource(p, func(s *model.Source) { s.Identity = "profile" }) }},
		{"source_token", func(p *model.Policy) { changeSource(p, func(s *model.Source) { s.Identity = "token_file" }) }},
		{"source_vault_on_json", func(p *model.Policy) {
			changeSource(p, func(s *model.Source) { s.VaultAllowlist = []string{"https://example.test"} })
		}},
		{"workload_name", func(p *model.Policy) { p.Workloads["bad/name"] = p.Workloads["demo"] }},
		{"workload_mode", func(p *model.Policy) { changeWorkload(p, func(w *model.Workload) { w.Mode = "fallback-anything" }) }},
		{"workload_empty", func(p *model.Policy) { changeWorkload(p, func(w *model.Workload) { w.Bindings = nil }) }},
		{"workload_directory", func(p *model.Policy) { changeWorkload(p, func(w *model.Workload) { w.Directory = "a\nb" }) }},
		{"workload_digest", func(p *model.Policy) { changeWorkload(p, func(w *model.Workload) { w.CommandSHA256 = "unverified" }) }},
		{"workload_command", func(p *model.Policy) { changeWorkload(p, func(w *model.Workload) { w.Command = []string{""} }) }},
		{"binding_invalid_name", func(p *model.Policy) { changeWorkload(p, func(w *model.Workload) { w.Bindings[0].Name = "bad=name" }) }},
		{"binding_bootstrap", func(p *model.Policy) {
			changeWorkload(p, func(w *model.Workload) { w.Bindings[0].Name = "VAULT_TOKEN" })
		}},
		{"binding_loader", func(p *model.Policy) {
			changeWorkload(p, func(w *model.Workload) { w.Bindings[0].Name = "LD_PRELOAD" })
		}},
		{"binding_missing_source", func(p *model.Policy) { changeWorkload(p, func(w *model.Workload) { w.Bindings[0].Source = "missing" }) }},
		{"binding_missing_scope", func(p *model.Policy) {
			changeWorkload(p, func(w *model.Workload) { w.Bindings[0].Resource = "unlisted" })
		}},
		{"binding_selector", func(p *model.Policy) { changeWorkload(p, func(w *model.Workload) { w.Bindings[0].Field = "*" }) }},
		{"binding_version_unsupported_source", func(p *model.Policy) { changeWorkload(p, func(w *model.Workload) { w.Bindings[0].Version = "1" }) }},
		{"binding_duplicate", func(p *model.Policy) {
			changeWorkload(p, func(w *model.Workload) { w.Bindings = append(w.Bindings, w.Bindings[0]) })
		}},
		{"binding_case_collision", func(p *model.Policy) {
			changeWorkload(p, func(w *model.Workload) {
				b := w.Bindings[0]
				b.Name = "demo_message"
				w.Bindings = append(w.Bindings, b)
			})
		}},
		{"binding_environment_collision", func(p *model.Policy) {
			changeWorkload(p, func(w *model.Workload) { w.Environment = map[string]string{"demo_message": "literal"} })
		}},
		{"binding_offline_denied", func(p *model.Policy) {
			changeWorkload(p, func(w *model.Workload) { w.Bindings[0].AllowOffline = false })
		}},
		{"binding_audit_unverified", func(p *model.Policy) { changeWorkload(p, func(w *model.Workload) { w.AuditRequired = true }) }},
		{"online_literal", func(p *model.Policy) { changeWorkload(p, func(w *model.Workload) { w.Mode = "online" }) }},
		{"target_type", func(p *model.Policy) {
			changeWorkload(p, func(w *model.Workload) {
				w.Targets = map[string]model.Target{"sink": {Type: "exec", AcceptSnapshot: true}}
			})
		}},
		{"target_snapshot_semantics", func(p *model.Policy) {
			changeWorkload(p, func(w *model.Workload) { w.Targets = map[string]model.Target{"sink": {Type: "file"}} })
		}},
		{"target_export", func(p *model.Policy) {
			changeWorkload(p, func(w *model.Workload) {
				w.Targets = map[string]model.Target{"sink": {Type: "file", AcceptSnapshot: true}}
			})
		}},
		{"bootstrap_source", func(p *model.Policy) { changeSource(p, func(s *model.Source) { s.TokenFile = s.Path }) }},
		{"bootstrap_snapshot", func(p *model.Policy) {
			changeWorkload(p, func(w *model.Workload) { w.Snapshot.IdentityFile = "/fixtures/values.json" })
		}},
		{"mcp_authority", func(p *model.Policy) { p.MCP.Issuer = "https://issuer.example.test" }},
		{"mcp_origin", func(p *model.Policy) { p.MCP.Origins = []string{"https://user:password@example.test"} }},
		{"mcp_host", func(p *model.Policy) { p.MCP.Hosts = []string{"*"} }},
		{"mcp_address", func(p *model.Policy) { p.MCP.Address = "bad-address" }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			policy := validPolicy()
			test.edit(&policy)
			if err := Validate(policy); !fault.Is(err, fault.Invalid) {
				t.Fatalf("expected sanitized invalid policy, got %v", err)
			}
		})
	}
}

func TestValidNativeScopes(t *testing.T) {
	native := filepath.Join(t.TempDir(), "native")
	policy := validPolicy()
	changeSource(&policy, func(s *model.Source) {
		s.Type = "openbao"
		s.Path = ""
		s.Endpoint = "https://bao.example.test"
		s.Mount = "kv"
		s.Resources = []string{"demo"}
		s.Identity = "token_file"
		s.TokenFile = filepath.Join(native, "bootstrap")
		s.AuditVerified = true
	})
	changeWorkload(&policy, func(w *model.Workload) {
		w.Mode = "online"
		w.Bindings[0].Resource = "demo"
		w.Bindings[0].Version = "2"
		w.Bindings[0].AllowPublish = true
		w.AuditRequired = true
		w.Command = []string{"/opt/worker", "--once"}
		w.CommandSHA256 = strings.Repeat("a", 64)
		w.Environment = map[string]string{"PATH": "/usr/bin", "SystemRoot": `C:\Windows`}
		w.Targets = map[string]model.Target{"sink": {Type: "file", Path: filepath.Join(native, "output"), AcceptSnapshot: true}}
	})
	policy.MCP = model.MCP{
		Issuer: "https://issuer.example.test", Audience: "http://127.0.0.1:8787/mcp",
		JWKSURL: "https://issuer.example.test/keys", Address: "127.0.0.1:8787",
		Origins: []string{"http://localhost:8787"}, Hosts: []string{"localhost", "127.0.0.1:8787"},
	}
	if err := Validate(policy); err != nil {
		t.Fatal(err)
	}
}

func TestSOPSAuthority(t *testing.T) {
	native := filepath.Join(t.TempDir(), "native")
	policy := validPolicy()
	changeSource(&policy, func(s *model.Source) {
		s.Type = "sops"
		s.Identity = "token_file"
		s.TokenFile = filepath.Join(native, "transit")
		s.SOPS = filepath.Join(native, "sops")
	})
	changeWorkload(&policy, func(w *model.Workload) { w.Mode = "online" })
	if Validate(policy) == nil {
		t.Fatal("online must reject absent Transit authority")
	}
	changeSource(&policy, func(s *model.Source) { s.VaultAllowlist = []string{"https://bao.example.test"} })
	if Validate(policy) == nil {
		t.Fatal("Transit key must be pinned")
	}
	changeSource(&policy, func(s *model.Source) { s.VaultKeys = []string{"transit/demo"} })
	if err := Validate(policy); err != nil {
		t.Fatal(err)
	}
	changeSource(&policy, func(s *model.Source) { s.AgeIdentityFile = "/protected/age" })
	if Validate(policy) == nil {
		t.Fatal("mixed online/offline authority must fail")
	}
	changeSource(&policy, func(s *model.Source) { s.AgeIdentityFile = ""; s.VaultAllowlist = []string{"all"} })
	if Validate(policy) == nil {
		t.Fatal("unrestricted Transit must fail")
	}
	changeSource(&policy, func(s *model.Source) { s.VaultAllowlist = []string{"https://bao.example.test"} })
	changeWorkload(&policy, func(w *model.Workload) { w.Snapshot.Path = "/protected/offline.sops.json" })
	if Validate(policy) == nil {
		t.Fatal("online cannot persist a fallback")
	}
}

func TestEnvironmentIsLiteralAndDistinct(t *testing.T) {
	if err := ValidateEnvironment(map[string]string{"MESSAGE": "$(not-executed)\nUnicode: 🦆"}); err != nil {
		t.Fatal(err)
	}
	for _, values := range []map[string]string{
		{"Name": "one", "NAME": "two"}, {"BAD=NAME": "x"}, {"X": "a\x00b"}, {"SOPS_AGE_KEY": "never"},
	} {
		if ValidateEnvironment(values) == nil {
			t.Fatal("invalid environment was accepted")
		}
	}
}

func TestErrorsDoNotEchoPolicyValues(t *testing.T) {
	policy := validPolicy()
	marker := "SYNTHETIC_NEVER_ECHO_MARKER"
	changeSource(&policy, func(s *model.Source) { s.Endpoint = "https://user:" + marker + "@example.test" })
	err := Validate(policy)
	if err == nil || strings.Contains(err.Error(), marker) {
		t.Fatal("diagnostic failed redaction contract")
	}
}

func TestSchemaIsJSONAndContainsEveryContractField(t *testing.T) {
	var schema map[string]any
	if err := json.Unmarshal(Schema, &schema); err != nil {
		t.Fatal(err)
	}
	for _, required := range []string{"vault_keys", "command_sha256", "untrusted_code", "audit_verified", "allow_offline", "jwks_url"} {
		if !strings.Contains(string(Schema), `"`+required+`"`) {
			t.Fatalf("schema missing %s", required)
		}
	}
}

func FuzzPolicyEnvironmentNames(f *testing.F) {
	f.Add("DEMO_MESSAGE")
	f.Add("demo_message")
	f.Add("VAULT_TOKEN")
	f.Add("PATH")
	f.Fuzz(func(t *testing.T, name string) {
		policy := validPolicy()
		changeWorkload(&policy, func(w *model.Workload) { w.Bindings[0].Name = name })
		err := Validate(policy)
		if err == nil && (!environmentName.MatchString(name) || forbiddenBindingName(strings.ToUpper(name))) {
			t.Fatal("unsafe environment name accepted")
		}
	})
}

func TestAdditionalAuthorityFailures(t *testing.T) {
	changes := []func(*model.Policy){
		func(p *model.Policy) {
			changeSource(p, func(s *model.Source) { s.Type = "sops"; s.VaultKeys = []string{"*"} })
		},
		func(p *model.Policy) {
			changeWorkload(p, func(w *model.Workload) { w.CommandSHA256 = strings.Repeat("g", 64) })
		},
		func(p *model.Policy) {
			changeWorkload(p, func(w *model.Workload) { w.Environment = map[string]string{"SOPS_AGE_KEY": "never"} })
		},
		func(p *model.Policy) {
			changeWorkload(p, func(w *model.Workload) { w.Mode = "hybrid"; w.AuditRequired = true })
		},
		func(p *model.Policy) {
			changeWorkload(p, func(w *model.Workload) {
				w.Targets = map[string]model.Target{"sink": {Type: "file", Endpoint: "http://example.test", AcceptSnapshot: true}}
			})
		},
	}
	for _, change := range changes {
		policy := validPolicy()
		change(&policy)
		if Validate(policy) == nil {
			t.Fatal("authority failure accepted")
		}
	}
}
