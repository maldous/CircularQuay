package policy

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/maldous/CircularQuay/internal/model"
)

func namedIdentityPolicy() model.Policy {
	p := validPolicy()
	p.MCP = model.MCP{
		Resource: "https://cq.example.test/mcp",
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
		"resource root":                          func(p *model.Policy) { p.MCP.Resource = "https://cq.example.test/" },
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

func TestNamedIdentityTrustStoreDeclarationIsPairedAndPinned(t *testing.T) {
	trustStorePath := filepath.Join(privatePolicyDirectory(t), "identity-roots.pem")
	p := namedIdentityPolicy()
	authority := p.MCP.Authorities["workforce"]
	authority.TrustStoreFile = trustStorePath
	authority.TrustStoreSHA256 = strings.Repeat("a", 64)
	p.MCP.Authorities["workforce"] = authority
	assertSchemaAndParseAccept(t, policyJSONValue(t, p))
	for name, edit := range map[string]func(*model.IdentityAuthority){
		"file only":     func(a *model.IdentityAuthority) { a.TrustStoreSHA256 = "" },
		"digest only":   func(a *model.IdentityAuthority) { a.TrustStoreFile = "" },
		"relative path": func(a *model.IdentityAuthority) { a.TrustStoreFile = "roots.pem" },
		"noncanonical path": func(a *model.IdentityAuthority) {
			a.TrustStoreFile = filepath.Dir(trustStorePath) + string(filepath.Separator) + "." + string(filepath.Separator) + filepath.Base(trustStorePath)
		},
		"upper digest": func(a *model.IdentityAuthority) { a.TrustStoreSHA256 = strings.Repeat("A", 64) },
		"short digest": func(a *model.IdentityAuthority) { a.TrustStoreSHA256 = "abc" },
	} {
		t.Run(name, func(t *testing.T) {
			candidate := namedIdentityPolicy()
			value := candidate.MCP.Authorities["workforce"]
			value.TrustStoreFile = trustStorePath
			value.TrustStoreSHA256 = strings.Repeat("a", 64)
			edit(&value)
			candidate.MCP.Authorities["workforce"] = value
			assertSchemaAndParseReject(t, policyJSONValue(t, candidate))
		})
	}
}

func TestIdentityTrustStoreSchemaCanonicalWindowsPaths(t *testing.T) {
	for path, wantValid := range map[string]bool{
		`C:\ProgramData\CircularQuay\identity-roots.pem`:    true,
		`C:/ProgramData/CircularQuay/identity-roots.pem`:    false,
		`\\server\share\identity-roots.pem`:                 true,
		`C:\ProgramData\\CircularQuay\identity-roots.pem`:   false,
		`C:\ProgramData\CircularQuay\..\identity-roots.pem`: false,
		`C:\ProgramData/CircularQuay\identity-roots.pem`:    false,
		`C:identity-roots.pem`:                              false,
	} {
		t.Run(path, func(t *testing.T) {
			policy := namedIdentityPolicy()
			authority := policy.MCP.Authorities["workforce"]
			authority.TrustStoreFile = path
			authority.TrustStoreSHA256 = strings.Repeat("a", 64)
			policy.MCP.Authorities["workforce"] = authority
			valid := resolvedPolicySchema(t).Validate(policyJSONValue(t, policy)) == nil
			if valid != wantValid {
				t.Fatalf("schema accepted %q = %t, want %t", path, valid, wantValid)
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
	trustConfig := namedIdentityPolicy().MCP
	trustFirst := MCPAuthorityFingerprint(trustConfig, "workforce")
	authority = trustConfig.Authorities["workforce"]
	authority.TrustStoreFile = "/private/identity-roots.pem"
	authority.TrustStoreSHA256 = strings.Repeat("a", 64)
	trustConfig.Authorities["workforce"] = authority
	withTrustStore := MCPAuthorityFingerprint(trustConfig, "workforce")
	if withTrustStore == trustFirst {
		t.Fatal("trust-store declaration retained authority fingerprint")
	}
	authority.TrustStoreFile = "/private/rotated-roots.pem"
	trustConfig.Authorities["workforce"] = authority
	withRotatedPath := MCPAuthorityFingerprint(trustConfig, "workforce")
	if withRotatedPath == withTrustStore {
		t.Fatal("trust-store path retained authority fingerprint")
	}
	authority.TrustStoreSHA256 = strings.Repeat("b", 64)
	trustConfig.Authorities["workforce"] = authority
	if MCPAuthorityFingerprint(trustConfig, "workforce") == withRotatedPath {
		t.Fatal("trust-store digest retained authority fingerprint")
	}
}
