package policy

import (
	"strings"
	"testing"
	"time"

	"github.com/maldous/CircularQuay/internal/model"
)

func namedIdentityPolicy() model.Policy {
	p := validPolicy()
	p.MCP = model.MCP{
		Resource: "https://cquay.example.test/mcp",
		Authorities: map[string]model.IdentityAuthority{"workforce": {
			Issuer: "https://identity.example.test", JWKSURL: "https://identity.example.test/keys",
			Audience: "registered-api", Profile: "rfc9068", Clients: []string{"approved-client"},
			Subjects:       []string{"https://identity.example.test|person"},
			RequiredScopes: []string{"session:approved"}, RequiredRoles: []string{"Operator"},
			Permissions: map[string]model.TokenPermission{"read": {Scopes: []string{"secrets:use"}, Roles: []string{"SecretReader"}}},
		}},
	}
	p.Grants = []model.Grant{{Authority: "workforce", Subject: "https://identity.example.test|person", Client: "approved-client", Workload: "demo", Operations: []string{"read"}, Keys: []string{"DEMO_MESSAGE"}}}
	return p
}

func TestNamedIdentityContractAndExactGrant(t *testing.T) {
	p := namedIdentityPolicy()
	if err := Validate(p); err != nil {
		t.Fatal(err)
	}
	if err := resolvedPolicySchema(t).Validate(policyJSONValue(t, p)); err != nil {
		t.Fatal(err)
	}
	grant := p.Grants[0]
	now := time.Date(2026, 11, 4, 1, 0, 0, 0, time.UTC)
	if !AuthorizeIdentityAt(p, Identity{Authority: "workforce", Subject: grant.Subject, Client: grant.Client}, "demo", "read", "DEMO_MESSAGE", now) {
		t.Fatal("exact named authority grant denied")
	}
	for _, authority := range []string{"", "other"} {
		if AuthorizeIdentityAt(p, Identity{Authority: authority, Subject: grant.Subject, Client: grant.Client}, "demo", "read", "DEMO_MESSAGE", now) {
			t.Fatal("authority boundary crossed")
		}
	}
	if AuthorizeAt(p, grant.Subject, grant.Client, "demo", "read", "DEMO_MESSAGE", now) {
		t.Fatal("legacy authorization consumed a named authority grant")
	}
}

func TestEntraProfileSchemaAndValidatorRequireGUIDAudience(t *testing.T) {
	p := namedIdentityPolicy()
	authority := p.MCP.Authorities["workforce"]
	tenant := "11111111-1111-1111-1111-111111111111"
	authority.Profile = "microsoft-entra-v2"
	authority.Issuer += "/" + tenant + "/v2.0"
	authority.Audience = "22222222-2222-2222-2222-222222222222"
	authority.Clients = []string{"33333333-3333-3333-3333-333333333333"}
	authority.Subjects = []string{authority.Issuer + "|" + tenant + "/44444444-4444-4444-4444-444444444444"}
	p.MCP.Authorities["workforce"] = authority
	p.Grants[0].Subject, p.Grants[0].Client = authority.Subjects[0], authority.Clients[0]
	assertSchemaAndParseAccept(t, policyJSONValue(t, p))
	authority.Audience = "interactive-login-or-other-api"
	p.MCP.Authorities["workforce"] = authority
	assertSchemaAndParseReject(t, policyJSONValue(t, p))
}

func TestNamedIdentityRejectsIncompleteAuthority(t *testing.T) {
	for name, edit := range map[string]func(*model.Policy){
		"resource missing":                       func(p *model.Policy) { p.MCP.Resource = "" },
		"resource root":                          func(p *model.Policy) { p.MCP.Resource = "https://cquay.example.test/" },
		"mixed contracts":                        func(p *model.Policy) { p.MCP.Issuer = "https://identity.example.test" },
		"missing grant authority":                func(p *model.Policy) { p.Grants[0].Authority = "unknown" },
		"unqualified subject":                    func(p *model.Policy) { p.Grants[0].Subject = "person" },
		"missing client":                         func(p *model.Policy) { p.Grants[0].Client = "" },
		"client outside authority":               func(p *model.Policy) { p.Grants[0].Client = "other-client" },
		"subject outside authority":              func(p *model.Policy) { p.Grants[0].Subject = "https://identity.example.test|other-person" },
		"operation without authority permission": func(p *model.Policy) { p.Grants[0].Operations = []string{"status"}; p.Grants[0].Keys = nil },
		"ambiguous issuer audience":              func(p *model.Policy) { p.MCP.Authorities["alias"] = p.MCP.Authorities["workforce"] },
		"empty authorities":                      func(p *model.Policy) { p.MCP.Authorities = map[string]model.IdentityAuthority{} },
	} {
		t.Run(name, func(t *testing.T) {
			p := namedIdentityPolicy()
			edit(&p)
			if Validate(p) == nil {
				t.Fatal("invalid identity authority accepted")
			}
		})
	}
}

func TestNamedIdentityRejectsClaimAndPermissionDrift(t *testing.T) {
	for name, edit := range map[string]func(*model.IdentityAuthority){
		"profile":             func(a *model.IdentityAuthority) { a.Profile = "any-jwt" },
		"key origin":          func(a *model.IdentityAuthority) { a.JWKSURL = "https://other.example.test/keys" },
		"audience whitespace": func(a *model.IdentityAuthority) { a.Audience = "api other" },
		"duplicate clients":   func(a *model.IdentityAuthority) { a.Clients = []string{"client", "client"} },
		"wildcard subjects":   func(a *model.IdentityAuthority) { a.Subjects = []string{"*"} },
		"oversize roles":      func(a *model.IdentityAuthority) { a.RequiredRoles = []string{strings.Repeat("r", 513)} },
		"scope whitespace":    func(a *model.IdentityAuthority) { a.RequiredScopes = []string{"one two"} },
		"permissions missing": func(a *model.IdentityAuthority) { a.Permissions = nil },
		"permission empty":    func(a *model.IdentityAuthority) { a.Permissions["run"] = model.TokenPermission{} },
		"permission unknown": func(a *model.IdentityAuthority) {
			a.Permissions["admin"] = model.TokenPermission{Roles: []string{"Owner"}}
		},
		"permission invalid role": func(a *model.IdentityAuthority) { a.Permissions["read"] = model.TokenPermission{Roles: []string{"*"}} },
	} {
		t.Run(name, func(t *testing.T) {
			p := namedIdentityPolicy()
			a := p.MCP.Authorities["workforce"]
			edit(&a)
			p.MCP.Authorities["workforce"] = a
			if Validate(p) == nil {
				t.Fatal("invalid claims/permissions accepted")
			}
		})
	}
}

func TestAuthorityFingerprintBindsOnlySelectedTrust(t *testing.T) {
	config := namedIdentityPolicy().MCP
	first := MCPAuthorityFingerprint(config, "workforce")
	if len(first) != 64 || MCPAuthorityFingerprint(config, "missing") != "" {
		t.Fatal("missing or malformed authority fingerprint")
	}
	config.Authorities["other"] = config.Authorities["workforce"]
	config.AuditPath = "/changed/audit.json"
	if MCPAuthorityFingerprint(config, "workforce") != first {
		t.Fatal("unrelated authority or audit changed selected fingerprint")
	}
	authority := config.Authorities["workforce"]
	authority.Audience = "other-api"
	config.Authorities["workforce"] = authority
	if MCPAuthorityFingerprint(config, "workforce") == first {
		t.Fatal("audience change retained authority")
	}
	config.Resource += "/changed"
	if MCPAuthorityFingerprint(config, "workforce") == first {
		t.Fatal("resource change retained authority")
	}
}
