package mcp

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"

	"github.com/maldous/CircularQuay/internal/identity"
	"github.com/maldous/CircularQuay/internal/model"
	"github.com/maldous/CircularQuay/internal/policy"
	"github.com/modelcontextprotocol/go-sdk/auth"
)

func configuredNamedIssuer(issuer *issuerFixture) model.MCP {
	return model.MCP{
		Resource: "https://cq.example/mcp", Hosts: []string{"cq.example"},
		Authorities: map[string]model.IdentityAuthority{"workforce": {
			Issuer: issuer.server.URL, JWKSURL: issuer.server.URL + "/keys", Audience: "registered-api", Profile: "rfc9068",
			Clients: []string{"agent-a"}, Subjects: []string{issuer.server.URL + "|alice"},
			RequiredScopes: []string{"session:approved"}, RequiredRoles: []string{"Operator"},
			Permissions: map[string]model.TokenPermission{"read": {Scopes: []string{"secret:use"}, Roles: []string{"SecretReader"}}},
		}},
	}
}

func TestNamedTokenAudienceIsIndependentOfResourceURL(t *testing.T) {
	issuer := newIssuer(t)
	config := configuredNamedIssuer(issuer)
	verify, err := newAccessTokenVerifier(context.Background(), config, issuer.server.Client())
	if err != nil {
		t.Fatal(err)
	}
	raw := issuer.token(t, map[string]any{"aud": "registered-api", "scope": "session:approved secret:use", "roles": []string{"Operator", "SecretReader"}})
	token, err := verify(context.Background(), raw, nil)
	if err != nil {
		t.Fatal("valid resource API token denied", err)
	}
	principal := principalFromToken(token, token.Extra["client"].(string))
	p := testPolicy(t)
	p.MCP = config
	p.Grants = []model.Grant{{Authority: "workforce", Subject: issuer.server.URL + "|alice", Client: "agent-a", Workload: "alpha", Operations: []string{"read"}, Keys: []string{"SERVICE_KEY"}}}
	if !principal.Authorize(p, "alpha", "read", "SERVICE_KEY", time.Now()) {
		t.Fatal("verified scope/role/authority did not reach exact local grant")
	}
	if _, err = verify(context.Background(), issuer.token(t, map[string]any{"scope": "session:approved secret:use", "roles": []string{"Operator", "SecretReader"}}), nil); err != auth.ErrInvalidToken {
		t.Fatal("public resource URL incorrectly substituted for registered token audience")
	}
}

func TestNamedMetadataPublishesOnlyConfiguredScopesAndIssuers(t *testing.T) {
	issuer := newIssuer(t)
	p := testPolicy(t)
	p.MCP = configuredNamedIssuer(issuer)
	service := makeService(t, p, &fakeBackend{}, Options{})
	handler, err := service.httpHandler(func(context.Context, string, *http.Request) (*auth.TokenInfo, error) {
		return nil, auth.ErrInvalidToken
	})
	if err != nil {
		t.Fatal(err)
	}
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "https://cq.example/.well-known/oauth-protected-resource/mcp", nil))
	var metadata struct {
		Resource string   `json:"resource"`
		Issuers  []string `json:"authorization_servers"`
		Scopes   []string `json:"scopes_supported"`
	}
	if recorder.Code != http.StatusOK || json.Unmarshal(recorder.Body.Bytes(), &metadata) != nil {
		t.Fatal("protected resource metadata unavailable")
	}
	if metadata.Resource != p.MCP.Resource || !reflect.DeepEqual(metadata.Issuers, []string{issuer.server.URL}) || !reflect.DeepEqual(metadata.Scopes, []string{"secret:use", "session:approved"}) {
		t.Fatal("metadata drifted from configured resource/issuer/scopes")
	}
	if len(requiredTransportScopes(p.MCP)) != 0 {
		t.Fatal("named role-aware authority inherited a hard-coded transport scope")
	}
}

func TestNamedTokenRoutingDeniesMissingAndAmbiguousAuthority(t *testing.T) {
	principal := identity.Principal{Subject: "https://issuer.example|alice", Client: "approved", ExpiresAt: time.Now().Add(time.Minute)}
	accepted := namedVerifier{name: "one", fingerprint: "verified", verify: func(context.Context, string) (identity.Principal, error) { return principal, nil }}
	rejected := namedVerifier{name: "two", fingerprint: "verified", verify: func(context.Context, string) (identity.Principal, error) {
		return identity.Principal{}, identity.ErrInvalidToken
	}}
	for _, verifiers := range [][]namedVerifier{nil, {rejected}, {accepted, accepted}, {{name: "one", verify: accepted.verify}}} {
		if token, err := verifyNamedToken(context.Background(), "synthetic-token", verifiers); err != auth.ErrInvalidToken || token != nil {
			t.Fatal("unbound or ambiguous token became an authoritative principal")
		}
	}
	token, err := verifyNamedToken(context.Background(), "synthetic-token", []namedVerifier{rejected, accepted})
	if err != nil || token.Extra["authority"] != "one" {
		t.Fatal("single configured authority did not verify")
	}
}

func TestNamedVerifierRejectsInvalidPolicyBeforeKeyRetrieval(t *testing.T) {
	issuer := newIssuer(t)
	for _, edit := range []func(*model.MCP){
		func(config *model.MCP) { config.Resource = "" },
		func(config *model.MCP) { config.Issuer = issuer.server.URL },
		func(config *model.MCP) { config.Authorities["alias"] = config.Authorities["workforce"] },
	} {
		config := configuredNamedIssuer(issuer)
		edit(&config)
		if _, err := newAccessTokenVerifier(context.Background(), config, issuer.server.Client()); err == nil {
			t.Fatal("invalid named authority contract accepted")
		}
	}
	config := configuredNamedIssuer(issuer)
	if policy.MCPAuthorityFingerprint(config, "workforce") == "" {
		t.Fatal("authority fingerprint missing")
	}
}
