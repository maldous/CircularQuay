package mcp

import (
	"context"
	"testing"
	"time"

	"github.com/maldous/CircularQuay/internal/fault"
	"github.com/maldous/CircularQuay/internal/model"
	"github.com/maldous/CircularQuay/internal/policy"
)

func namedAuthorityServicePolicy(t *testing.T) model.Policy {
	t.Helper()
	config := testPolicy(t)
	config.MCP = model.MCP{
		Resource: "https://cquay.example.test/mcp",
		Authorities: map[string]model.IdentityAuthority{
			"workforce": {
				Issuer:         "https://identity.example.test",
				JWKSURL:        "https://identity.example.test/keys",
				Audience:       "registered-api",
				Profile:        "rfc9068",
				Clients:        []string{"approved-client"},
				Subjects:       []string{"https://identity.example.test|person"},
				RequiredScopes: []string{"session:approved"},
				RequiredRoles:  []string{"Operator"},
				Permissions: map[string]model.TokenPermission{
					"read": {Scopes: []string{"secrets:use"}, Roles: []string{"SecretReader"}},
				},
			},
		},
	}
	config.Grants = []model.Grant{{
		Authority: "workforce", Subject: "https://identity.example.test|person", Client: "approved-client",
		Workload: "alpha", Operations: []string{"read"}, Keys: []string{"SERVICE_KEY"},
	}}
	return config
}

func namedAuthorityPrincipal(config model.Policy, expires time.Time) Principal {
	return Principal{
		Subject:              "https://identity.example.test|person",
		Client:               "approved-client",
		Authority:            "workforce",
		AuthorityFingerprint: policy.MCPAuthorityFingerprint(config.MCP, "workforce"),
		HTTP:                 true,
		Scopes:               []string{"session:approved", "secrets:use"},
		Roles:                []string{"Operator", "SecretReader"},
		ExpiresAt:            expires,
	}
}

func TestNamedAuthorityPrincipalRequiresExactCurrentContract(t *testing.T) {
	now := time.Date(2026, time.November, 4, 1, 0, 0, 0, time.UTC)
	config := namedAuthorityServicePolicy(t)
	principal := namedAuthorityPrincipal(config, now.Add(time.Minute))
	if !principal.Authorize(config, "alpha", "read", "SERVICE_KEY", now) {
		t.Fatal("exact named authority denied")
	}
	for name, edit := range map[string]func(*Principal){
		"missing authority":    func(p *Principal) { p.Authority = "" },
		"wrong fingerprint":    func(p *Principal) { p.AuthorityFingerprint = "other" },
		"unapproved client":    func(p *Principal) { p.Client = "other" },
		"unapproved subject":   func(p *Principal) { p.Subject = "https://identity.example.test|other" },
		"missing global scope": func(p *Principal) { p.Scopes = []string{"secrets:use"} },
		"missing global role":  func(p *Principal) { p.Roles = []string{"SecretReader"} },
		"missing action scope": func(p *Principal) { p.Scopes = []string{"session:approved"} },
		"missing action role":  func(p *Principal) { p.Roles = []string{"Operator"} },
	} {
		t.Run(name, func(t *testing.T) {
			candidate := principal
			edit(&candidate)
			if candidate.Authorize(config, "alpha", "read", "SERVICE_KEY", now) {
				t.Fatal("authority claim boundary bypassed")
			}
		})
	}
	local := Principal{Subject: principal.Subject, Client: principal.Client}
	if local.Authorize(config, "alpha", "read", "SERVICE_KEY", now) {
		t.Fatal("local principal consumed named authority grant")
	}
	emptyPermission := config
	emptyAuthority := emptyPermission.MCP.Authorities["workforce"]
	emptyAuthority.Permissions = map[string]model.TokenPermission{"read": {}}
	emptyPermission.MCP.Authorities["workforce"] = emptyAuthority
	principal.AuthorityFingerprint = policy.MCPAuthorityFingerprint(emptyPermission.MCP, "workforce")
	if principal.Authorize(emptyPermission, "alpha", "read", "SERVICE_KEY", now) {
		t.Fatal("empty action permission granted access")
	}
}

func TestNamedAuthorityFingerprintRechecksLoadedPolicy(t *testing.T) {
	now := time.Date(2026, time.November, 4, 1, 0, 0, 0, time.UTC)
	config := namedAuthorityServicePolicy(t)
	principal := namedAuthorityPrincipal(config, now.Add(time.Minute))
	current := config
	backend := &fakeBackend{value: model.Value{Data: "MUST-NOT-DISCLOSE"}}
	service := makeService(t, config, backend, Options{
		Audit:          &memoryAudit{},
		PolicyProvider: func() (model.Policy, error) { return current, nil },
	})
	service.now = func() time.Time { return now }
	authority := current.MCP.Authorities["workforce"]
	authority.Audience = "rotated-api"
	current.MCP.Authorities["workforce"] = authority
	if _, err := service.ReadSecret(context.Background(), principal, "alpha", "SERVICE_KEY"); !fault.Is(err, fault.Denied) || backend.reads != 0 {
		t.Fatalf("reconfigured authority result = %v, reads=%d", err, backend.reads)
	}
}

func TestLegacyHTTPPrincipalRequiresDefaultOperationScope(t *testing.T) {
	now := time.Date(2026, time.November, 4, 1, 0, 0, 0, time.UTC)
	config := testPolicy(t)
	principal := Principal{
		Subject: "local:alice", HTTP: true, Scopes: []string{"cquay:read"}, ExpiresAt: now.Add(time.Minute),
	}
	if !principal.Authorize(config, "alpha", "read", "SERVICE_KEY", now) {
		t.Fatal("legacy HTTP authorization denied")
	}
	principal.Scopes = []string{"cquay:status"}
	if principal.Authorize(config, "alpha", "read", "SERVICE_KEY", now) {
		t.Fatal("legacy HTTP scope granted another operation")
	}
}

func TestLegacyAuthorityFingerprintRechecksLoadedPolicyBeforeProvider(t *testing.T) {
	now := time.Date(2026, time.November, 4, 1, 0, 0, 0, time.UTC)
	config := legacyAuthorityServicePolicy(t)
	principal := Principal{
		Subject:              "local:alice",
		HTTP:                 true,
		Scopes:               []string{"cquay:read"},
		ExpiresAt:            now.Add(time.Minute),
		AuthorityFingerprint: policy.MCPLegacyFingerprint(config.MCP),
	}
	for name, edit := range map[string]func(*Principal, *model.MCP){
		"wrong fingerprint": func(principal *Principal, _ *model.MCP) { principal.AuthorityFingerprint = "other" },
		"issuer": func(_ *Principal, mcp *model.MCP) {
			mcp.Issuer = "https://rotated-identity.example.test"
			mcp.JWKSURL = "https://rotated-identity.example.test/keys"
		},
		"jwks":     func(_ *Principal, mcp *model.MCP) { mcp.JWKSURL = "https://identity.example.test/rotated-keys" },
		"audience": func(_ *Principal, mcp *model.MCP) { mcp.Audience = "https://cquay.example.test/rotated-mcp" },
	} {
		t.Run(name, func(t *testing.T) {
			current := config
			candidate := principal
			edit(&candidate, &current.MCP)
			backend := &fakeBackend{value: model.Value{Data: "MUST-NOT-DISCLOSE"}}
			service := makeService(t, config, backend, Options{
				Audit:          &memoryAudit{},
				PolicyProvider: func() (model.Policy, error) { return current, nil },
			})
			service.now = func() time.Time { return now }
			if _, err := service.ReadSecret(context.Background(), candidate, "alpha", "SERVICE_KEY"); !fault.Is(err, fault.Denied) || backend.reads != 0 {
				t.Fatalf("legacy authority change result = %v, reads=%d", err, backend.reads)
			}
		})
	}
}

func legacyAuthorityServicePolicy(t *testing.T) model.Policy {
	t.Helper()
	config := testPolicy(t)
	config.MCP = model.MCP{
		Issuer: "https://identity.example.test", JWKSURL: "https://identity.example.test/keys", Audience: "https://cquay.example.test/mcp",
	}
	return config
}
