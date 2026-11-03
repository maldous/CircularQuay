package identity

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	tenantID   = "11111111-1111-1111-1111-111111111111"
	objectID   = "22222222-2222-2222-2222-222222222222"
	clientID   = "33333333-3333-3333-3333-333333333333"
	audienceID = "44444444-4444-4444-4444-444444444444"
)

type authorityFixture struct {
	server *httptest.Server
	key    *ecdsa.PrivateKey
	now    time.Time
}

func newAuthorityFixture(t *testing.T) *authorityFixture {
	return newAuthorityFixtureWithServer(t, httptest.NewTLSServer)
}

func newLoopbackAuthorityFixture(t *testing.T) *authorityFixture {
	return newAuthorityFixtureWithServer(t, httptest.NewServer)
}

func newAuthorityFixtureWithServer(t *testing.T, newServer func(http.Handler) *httptest.Server) *authorityFixture {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	fixture := &authorityFixture{key: key, now: time.Date(2026, 11, 4, 0, 0, 0, 0, time.UTC)}
	fixture.server = newServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/keys" {
			http.NotFound(w, r)
			return
		}
		if err := json.NewEncoder(w).Encode(map[string]any{"keys": []any{fixture.jwk()}}); err != nil {
			t.Error(err)
		}
	}))
	t.Cleanup(fixture.server.Close)
	return fixture
}

func (fixture *authorityFixture) jwk() map[string]any {
	return map[string]any{
		"kty": "EC", "crv": "P-256", "alg": "ES256", "use": "sig", "kid": "fixture",
		"x": base64.RawURLEncoding.EncodeToString(fixture.key.X.FillBytes(make([]byte, 32))),
		"y": base64.RawURLEncoding.EncodeToString(fixture.key.Y.FillBytes(make([]byte, 32))),
	}
}

func (fixture *authorityFixture) authority(profile Profile) Authority {
	issuer := fixture.server.URL
	audience := "api://cquay"
	if profile == ProfileMicrosoftEntraV2 {
		issuer += "/" + tenantID + "/v2.0"
		audience = audienceID
	}
	return Authority{Name: "synthetic", Issuer: issuer, JWKSURL: fixture.server.URL + "/keys", Audience: audience, Profile: profile}
}

func (fixture *authorityFixture) token(t *testing.T, authority Authority, changes map[string]any) string {
	t.Helper()
	typ := "JWT"
	if authority.Profile == ProfileRFC9068 {
		typ = "at+jwt"
	}
	return fixture.tokenWithType(t, authority, changes, typ)
}

func (fixture *authorityFixture) tokenWithType(t *testing.T, authority Authority, changes map[string]any, typ string) string {
	t.Helper()
	claims := jwt.MapClaims{
		"iss": authority.Issuer, "sub": "pairwise-subject", "aud": authority.Audience,
		"exp": fixture.now.Add(time.Hour).Unix(), "iat": fixture.now.Unix(), "jti": "synthetic-token-id",
		"client_id": "client-a", "scope": "cquay:mcp cquay:read",
	}
	for name, value := range changes {
		claims[name] = value
	}
	token := jwt.NewWithClaims(jwt.SigningMethodES256, claims)
	token.Header["kid"] = "fixture"
	token.Header["typ"] = typ
	raw, err := token.SignedString(fixture.key)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func newFixtureVerifier(t *testing.T, fixture *authorityFixture, authority Authority) Verifier {
	t.Helper()
	verifier, err := newVerifier(context.Background(), authority, fixture.server.Client(), func() time.Time { return fixture.now })
	if err != nil {
		t.Fatal(err)
	}
	return verifier
}

func TestRFC9068VerifierBindsSignedIdentityAndFilters(t *testing.T) {
	fixture := newAuthorityFixture(t)
	authority := fixture.authority(ProfileRFC9068)
	authority.Clients = []string{"client-b", "client-a"}
	authority.Subjects = []string{authority.Issuer + "|other-subject", authority.Issuer + "|pairwise-subject"}
	authority.RequiredScopes = []string{"cquay:read"}
	authority.RequiredRoles = []string{"Cquay.Reader"}
	verifier := newFixtureVerifier(t, fixture, authority)
	raw := fixture.token(t, authority, map[string]any{"roles": []string{"Cquay.Reader"}})
	principal, err := verifier(context.Background(), raw)
	if err != nil || principal.Subject != authority.Issuer+"|pairwise-subject" || principal.Client != "client-a" || principal.Authority != authority.Name {
		t.Fatalf("verified principal = %+v, %v", principal, err)
	}
}

func TestRFC9068VerifierRejectsProfileAndPermissionViolations(t *testing.T) {
	fixture := newAuthorityFixture(t)
	authority := fixture.authority(ProfileRFC9068)
	authority.RequiredScopes = []string{"cquay:read"}
	verifier := newFixtureVerifier(t, fixture, authority)
	for _, changes := range []map[string]any{
		{"scope": "cquay:mcp"}, {"client_id": ""}, {"azp": "other-client"},
		{"cnf": map[string]any{"jkt": "unimplemented"}}, {"nonce": "id-token-only"},
		{"iat": fixture.now.Add(time.Hour).Unix()}, {"nbf": fixture.now.Add(time.Hour).Unix()}, {"exp": fixture.now.Unix()},
		{"scope": "cquay:read cquay:read"}, {"scope": "cquay:mcp\tcquay:read"}, {"scope": "cquay:readé"}, {"scope": strings.Repeat("x", maximumClaimText+1)}, {"roles": nil},
		{"scope": testScopes(maximumClaimItems + 1)},
	} {
		if _, err := verifier(context.Background(), fixture.token(t, authority, changes)); err != ErrInvalidToken {
			t.Fatalf("accepted invalid RFC 9068 claims %v: %v", changes, err)
		}
	}
	idToken := fixture.tokenWithType(t, authority, nil, "JWT")
	if _, err := verifier(context.Background(), idToken); err != ErrInvalidToken {
		t.Fatalf("accepted altered access header: %v", err)
	}
}

func TestVerifierRejectsMalformedTokensAndDisallowedIdentity(t *testing.T) {
	fixture := newAuthorityFixture(t)
	authority := fixture.authority(ProfileRFC9068)
	authority.Clients = []string{"client-b"}
	verifier := newFixtureVerifier(t, fixture, authority)
	if _, err := verifier(context.Background(), fixture.token(t, authority, nil)); err != ErrInvalidToken {
		t.Fatalf("accepted disallowed client: %v", err)
	}
	for _, raw := range []string{"", "not.a", "not.a.token", "%%%..", strings.Repeat("x", maximumTokenBytes+1)} {
		if _, err := verifier(context.Background(), raw); err != ErrInvalidToken {
			t.Fatalf("accepted malformed token %q: %v", raw, err)
		}
	}
}

func TestNewVerifierUsesDefaultHardenedKeyClientOnLoopback(t *testing.T) {
	fixture := newLoopbackAuthorityFixture(t)
	fixture.now = time.Now()
	authority := fixture.authority(ProfileRFC9068)
	verifier, err := NewVerifier(context.Background(), authority, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := verifier(context.Background(), fixture.token(t, authority, nil)); err != nil {
		t.Fatalf("default key client rejected local signed token: %v", err)
	}
}

func testScopes(count int) string {
	return strings.TrimSpace(strings.Repeat("cquay:read ", count))
}

func TestVerifierRejectsAmbiguousJWTJSON(t *testing.T) {
	fixture := newAuthorityFixture(t)
	authority := fixture.authority(ProfileRFC9068)
	verifier := newFixtureVerifier(t, fixture, authority)
	valid := fixture.token(t, authority, nil)
	header, payload := tokenDocumentsForTest(t, valid)
	cases := []struct {
		name    string
		header  []byte
		payload []byte
	}{
		{"duplicate header", []byte(`{"alg":"ES256","kid":"fixture","typ":"at+jwt","typ":"at+jwt"}`), payload},
		{"duplicate claims", header, append(append([]byte{}, payload[:len(payload)-1]...), []byte(`,"scope":"cquay:mcp cquay:read"}`)...)},
		{"trailing payload", header, append(append([]byte{}, payload...), []byte(`null`)...)},
		{"invalid UTF-8 payload", header, []byte{'{', '"', 'i', 's', 's', '"', ':', '"', 0xff, '"', '}'}},
	}
	for _, test := range cases {
		if _, err := verifier(context.Background(), fixture.signedDocuments(t, test.header, test.payload)); err != ErrInvalidToken {
			t.Fatalf("accepted %s: %v", test.name, err)
		}
	}
}

func TestMicrosoftEntraV2VerifierBindsTenantObjectAndPermissions(t *testing.T) {
	fixture := newAuthorityFixture(t)
	authority := fixture.authority(ProfileMicrosoftEntraV2)
	authority.Clients = []string{clientID}
	authority.Subjects = []string{authority.Issuer + "|" + tenantID + "/" + objectID}
	authority.RequiredScopes = []string{"cquay.read"}
	authority.RequiredRoles = []string{"Cquay.Reader"}
	verifier := newFixtureVerifier(t, fixture, authority)
	principal, err := verifier(context.Background(), fixture.token(t, authority, map[string]any{
		"ver": "2.0", "tid": tenantID, "oid": objectID, "azp": clientID, "scp": "cquay.read", "roles": []string{"Cquay.Reader"},
	}))
	if err != nil || principal.Subject != authority.Issuer+"|"+tenantID+"/"+objectID || principal.Client != clientID {
		t.Fatalf("verified Entra principal = %+v, %v", principal, err)
	}
}

func TestMicrosoftEntraV2RejectsIDAndUnqualifiedTokenProfiles(t *testing.T) {
	fixture := newAuthorityFixture(t)
	authority := fixture.authority(ProfileMicrosoftEntraV2)
	verifier := newFixtureVerifier(t, fixture, authority)
	for _, changes := range []map[string]any{
		{"ver": "1.0", "tid": tenantID, "oid": objectID, "azp": clientID, "scp": "cquay.read"},
		{"ver": "2.0", "tid": "44444444-4444-4444-4444-444444444444", "oid": objectID, "azp": clientID, "scp": "cquay.read"},
		{"ver": "2.0", "tid": tenantID, "oid": objectID, "azp": clientID},
		{"ver": "2.0", "tid": tenantID, "azp": clientID, "scp": "cquay.read"},
		{"ver": "2.0", "tid": tenantID, "oid": objectID, "azp": clientID, "roles": []string{""}},
		{"ver": "2.0", "tid": tenantID, "oid": "not-a-guid", "azp": clientID, "scp": "cquay.read"},
		{"ver": "2.0", "tid": tenantID, "oid": "22222222_2222-2222-2222-222222222222", "azp": clientID, "scp": "cquay.read"},
		{"ver": "2.0", "tid": tenantID, "oid": "22222222-2222-2222-2222-22222222222g", "azp": clientID, "scp": "cquay.read"},
		{"ver": "2.0", "tid": tenantID, "oid": objectID, "azp": "not-a-guid", "scp": "cquay.read"},
	} {
		if _, err := verifier(context.Background(), fixture.token(t, authority, changes)); err != ErrInvalidToken {
			t.Fatalf("accepted invalid Entra claims %v: %v", changes, err)
		}
	}
}

func TestMicrosoftEntraV2AcceptsApplicationRoleWithoutDelegatedScope(t *testing.T) {
	fixture := newAuthorityFixture(t)
	authority := fixture.authority(ProfileMicrosoftEntraV2)
	authority.RequiredRoles = []string{"Cquay.Application"}
	verifier := newFixtureVerifier(t, fixture, authority)
	_, err := verifier(context.Background(), fixture.token(t, authority, map[string]any{
		"ver": "2.0", "tid": tenantID, "oid": objectID, "azp": clientID, "roles": []string{"Cquay.Application"},
	}))
	if err != nil {
		t.Fatalf("application role token rejected: %v", err)
	}
}

func tokenDocumentsForTest(t *testing.T, raw string) ([]byte, []byte) {
	t.Helper()
	header, payload, err := tokenDocuments(raw)
	if err != nil {
		t.Fatal(err)
	}
	return header, payload
}

func (fixture *authorityFixture) signedDocuments(t *testing.T, header, payload []byte) string {
	t.Helper()
	encodedHeader := base64.RawURLEncoding.EncodeToString(header)
	encodedPayload := base64.RawURLEncoding.EncodeToString(payload)
	signingInput := encodedHeader + "." + encodedPayload
	signature, err := jwt.SigningMethodES256.Sign(signingInput, fixture.key)
	if err != nil {
		t.Fatal(err)
	}
	return signingInput + "." + base64.RawURLEncoding.EncodeToString(signature)
}

func TestAuthorityValidationRejectsUnsafeKeysAndUnsupportedProfiles(t *testing.T) {
	fixture := newAuthorityFixture(t)
	valid := fixture.authority(ProfileRFC9068)
	tooManyClients := testValues(maximumClaimItems + 1)
	for _, authority := range append([]Authority{
		{},
		{Name: "bad", Issuer: "https://issuer.example", JWKSURL: "https://other.example/keys", Audience: "api://cquay", Profile: ProfileRFC9068},
		{Name: "bad", Issuer: "http://issuer.example", JWKSURL: "http://issuer.example/keys", Audience: "api://cquay", Profile: ProfileRFC9068},
		{Name: "bad/name", Issuer: "https://issuer.example", JWKSURL: "https://issuer.example/keys", Audience: "api://cquay", Profile: ProfileRFC9068},
		{Name: "bad", Issuer: "https://%", JWKSURL: "https://issuer.example/keys", Audience: "api://cquay", Profile: ProfileRFC9068},
		{Name: "bad", Issuer: fixture.server.URL, JWKSURL: fixture.server.URL + "/keys", Audience: "api://cquay", Profile: "unrecognized"},
		{Name: "bad", Issuer: fixture.server.URL + "/tenant", JWKSURL: fixture.server.URL + "/keys", Audience: "api://cquay", Profile: ProfileMicrosoftEntraV2},
		{Name: "bad", Issuer: fixture.server.URL + "/" + tenantID + "/v2.0", JWKSURL: fixture.server.URL + "/keys", Audience: "api://not-a-guid", Profile: ProfileMicrosoftEntraV2},
	}, []Authority{
		withClients(valid, []string{"client-a", "client-a"}),
		withClients(valid, tooManyClients),
	}...) {
		if err := ValidateAuthority(authority); err == nil {
			t.Fatalf("unsafe authority accepted: %+v", authority)
		}
	}
	if err := ValidateAuthority(fixture.authority(ProfileRFC9068)); err != nil {
		t.Fatalf("valid authority rejected: %v", err)
	}
	if _, err := NewVerifier(context.Background(), Authority{}, nil); err == nil {
		t.Fatal("invalid authority constructed a verifier")
	}
}

func withClients(authority Authority, clients []string) Authority {
	authority.Clients = clients
	return authority
}

func testValues(count int) []string {
	values := make([]string, count)
	for index := range values {
		values[index] = fmt.Sprintf("client-%d", index)
	}
	return values
}
