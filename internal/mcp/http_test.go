package mcp

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/maldous/CircularQuay/internal/fault"
	"github.com/maldous/CircularQuay/internal/model"
	"github.com/modelcontextprotocol/go-sdk/auth"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

type issuerFixture struct {
	server *httptest.Server
	key    *ecdsa.PrivateKey
}

func newIssuer(t *testing.T) *issuerFixture {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	fixture := &issuerFixture{key: key}
	fixture.server = httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/keys" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		jwk := map[string]any{"kty": "EC", "crv": "P-256", "alg": "ES256", "use": "sig", "kid": "fixture",
			"x": base64.RawURLEncoding.EncodeToString(key.X.FillBytes(make([]byte, 32))),
			"y": base64.RawURLEncoding.EncodeToString(key.Y.FillBytes(make([]byte, 32)))}
		if err := json.NewEncoder(w).Encode(map[string]any{"keys": []any{jwk}}); err != nil {
			t.Error(err)
		}
	}))
	t.Cleanup(fixture.server.Close)
	return fixture
}

func (i *issuerFixture) token(t *testing.T, changes map[string]any) string {
	t.Helper()
	claims := jwt.MapClaims{"iss": i.server.URL, "sub": "alice", "aud": "https://cq.example/mcp", "exp": time.Now().Add(time.Minute).Unix(), "iat": time.Now().Unix(), "jti": "synthetic-token-id", "client_id": "agent-a", "scope": "cq:mcp cq:list cq:status cq:read"}
	for k, v := range changes {
		claims[k] = v
	}
	token := jwt.NewWithClaims(jwt.SigningMethodES256, claims)
	token.Header["kid"] = "fixture"
	token.Header["typ"] = "at+jwt"
	raw, err := token.SignedString(i.key)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func (i *issuerFixture) config() model.MCP {
	return model.MCP{Issuer: i.server.URL, JWKSURL: i.server.URL + "/keys", Audience: "https://cq.example/mcp", Hosts: []string{"cq.example"}}
}

func TestJWTProfileAndIdentity(t *testing.T) {
	issuer := newIssuer(t)
	verify, err := newAccessTokenVerifier(context.Background(), issuer.config(), issuer.server.Client())
	if err != nil {
		t.Fatal(err)
	}
	assertVerifiedTokenIdentity(t, verify, issuer)
	assertInvalidClaimTokens(t, verify, issuer)
	assertInvalidTokenEncodings(t, verify)
	assertIDTokenRejected(t, verify, issuer)
}

func assertVerifiedTokenIdentity(t *testing.T, verify auth.TokenVerifier, issuer *issuerFixture) {
	t.Helper()
	ti, err := verify(context.Background(), issuer.token(t, nil), nil)
	if err != nil || ti.UserID != issuer.server.URL+"|alice" || ti.Extra["client"] != "agent-a" {
		t.Fatal("verified identity not bound", err)
	}
}

func assertInvalidClaimTokens(t *testing.T, verify auth.TokenVerifier, issuer *issuerFixture) {
	t.Helper()
	for _, changes := range []map[string]any{
		{"iss": "https://other.example"}, {"aud": "https://other.example/mcp"},
		{"exp": time.Now().Add(-time.Hour).Unix()}, {"sub": ""},
		{"client_id": ""}, {"client_id": "agent-a", "azp": "agent-b"},
		{"nbf": time.Now().Add(time.Hour).Unix()}, {"sub": "alice\x00other"},
		{"client_id": "agent\nother"}, {"sub": strings.Repeat("a", 513)},
		{"iat": nil}, {"iat": time.Now().Add(time.Hour).Unix()}, {"jti": ""},
		{"client_id": "", "azp": "agent-a"}, {"iat": "malformed"}, {"jti": strings.Repeat("x", 513)},
	} {
		if _, err := verify(context.Background(), issuer.token(t, changes), nil); err != auth.ErrInvalidToken {
			t.Fatal("invalid claims accepted", changes, err)
		}
	}
}

func assertInvalidTokenEncodings(t *testing.T, verify auth.TokenVerifier) {
	t.Helper()
	for _, raw := range []string{"", "not.a.jwt", strings.Repeat("x", 16385)} {
		if _, err := verify(context.Background(), raw, nil); err != auth.ErrInvalidToken {
			t.Fatal("bad encoding accepted", err)
		}
	}
}

func assertIDTokenRejected(t *testing.T, verify auth.TokenVerifier, issuer *issuerFixture) {
	t.Helper()
	// A correctly signed ID token is not an MCP access token.
	idToken := jwt.NewWithClaims(jwt.SigningMethodES256, jwt.MapClaims{"iss": issuer.server.URL, "sub": "alice", "aud": "https://cq.example/mcp", "exp": time.Now().Add(time.Minute).Unix()})
	raw, err := idToken.SignedString(issuer.key)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := verify(context.Background(), raw, nil); err != auth.ErrInvalidToken {
		t.Fatal("ID token accepted", err)
	}
}

func TestIssuerConfiguration(t *testing.T) {
	for _, cfg := range []model.MCP{
		{},
		{Issuer: "https://issuer", JWKSURL: "https://other/keys", Audience: "https://cq/mcp"},
		{Issuer: "http://public.example", JWKSURL: "http://public.example/keys", Audience: "https://cq/mcp"},
		{Issuer: "https://issuer?token=x", JWKSURL: "https://issuer/keys", Audience: "https://cq/mcp"},
		{Issuer: "https://issuer", JWKSURL: "https://issuer/keys", Audience: "https://user:password@cq/mcp"},
	} {
		if _, err := NewAccessTokenVerifier(context.Background(), cfg); err == nil {
			t.Fatal("unsafe issuer config accepted")
		}
	}
	if !isLoopback("::1") || !isLoopback("127.0.0.1") || !isLoopback("localhost") || isLoopback("elsewhere") {
		t.Fatal("loopback classification")
	}
}

func TestHTTPHandlerBuildsVerifierFromTrustedConfiguration(t *testing.T) {
	issuer := newIssuer(t)
	p := testPolicy(t)
	p.MCP = issuer.config()
	s := makeService(t, p, &fakeBackend{}, Options{})
	if _, err := s.HTTPHandler(); err != nil {
		t.Fatal("trusted verifier configuration rejected", err)
	}
	p.MCP.Issuer = "http://public.example"
	s.configured = p
	if _, err := s.HTTPHandler(); !fault.Is(err, fault.Invalid) {
		t.Fatal("unsafe public plaintext issuer accepted", err)
	}
}

func TestHTTPGuardsAndMetadata(t *testing.T) {
	p := testPolicy(t)
	p.MCP = model.MCP{Issuer: "https://issuer.example", JWKSURL: "https://issuer.example/keys", Audience: "https://cq.example/mcp", Hosts: []string{"cq.example"}, Origins: []string{"https://client.example"}}
	s := makeService(t, p, &fakeBackend{}, Options{})
	verify := func(context.Context, string, *http.Request) (*auth.TokenInfo, error) {
		return nil, auth.ErrInvalidToken
	}
	h, err := s.httpHandler(verify)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		path, host, origin, token string
		want                      int
	}{
		{"/mcp", "attacker.example", "", "", 403},
		{"/mcp", "cq.example", "null", "", 403},
		{"/mcp", "cq.example", "https://attacker.example", "", 403},
		{"/mcp", "cq.example", "", "", 401},
		{"/mcp", "cq.example", "", "invalid", 401},
		{"/.well-known/oauth-protected-resource/mcp", "cq.example", "", "", 200},
	} {
		assertHTTPGuard(t, h, tc.path, tc.host, tc.origin, tc.token, tc.want)
	}
	request := httptest.NewRequest(http.MethodPost, "https://cq.example/mcp", strings.NewReader(strings.Repeat("x", maximumRequestBytes+1)))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, request)
	if w.Code != 413 {
		t.Fatal("oversized request accepted", w.Code)
	}
	p.MCP.Audience = "https://cq.example/"
	s.configured = p
	if _, err := s.httpHandler(verify); !fault.Is(err, fault.Invalid) {
		t.Fatal("invalid resource path", err)
	}
	if permittedHost("cq.example/other", []string{"cq.example/other"}) || permittedOrigin("null", []string{"null"}) {
		t.Fatal("origin parsing")
	}
}

func assertHTTPGuard(t *testing.T, handler http.Handler, path, host, origin, token string, want int) {
	t.Helper()
	request := httptest.NewRequest(http.MethodGet, "https://"+host+path, nil)
	request.Header.Set("Origin", origin)
	if token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != want || recorder.Header().Get("Cache-Control") != "private, no-store" {
		t.Fatal(want, recorder.Code, recorder.Body.String())
	}
	assertHTTPResponseContract(t, recorder, want)
}

func assertHTTPResponseContract(t *testing.T, recorder *httptest.ResponseRecorder, want int) {
	t.Helper()
	if want == http.StatusUnauthorized && !strings.Contains(recorder.Header().Get("WWW-Authenticate"), "resource_metadata") {
		t.Fatal("missing OAuth discovery challenge")
	}
	if want == http.StatusOK && !strings.Contains(recorder.Body.String(), "cq:read") {
		t.Fatal("missing scope contract")
	}
}

type bearerTransport struct {
	base  http.RoundTripper
	token string
}

func (b bearerTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	authorizedRequest := r.Clone(r.Context())
	authorizedRequest.Header.Set("Authorization", "Bearer "+b.token)
	return b.base.RoundTrip(authorizedRequest)
}

func TestAuthenticatedHTTPClient(t *testing.T) {
	client, b, audit, issuer, expires, ctx := authenticatedHTTPClient(t)
	tools, err := client.ListTools(ctx, nil)
	if err != nil || len(tools.Tools) != 2 || tools.CacheScope != "private" {
		t.Fatal("bad discovery", tools, err)
	}
	result, err := client.CallTool(ctx, &sdk.CallToolParams{Name: "read_secret", Arguments: map[string]any{"workload": "alpha", "key": "SERVICE_KEY"}})
	if err != nil || result.IsError {
		t.Fatal(result, err)
	}
	assertAuthenticatedHTTPRead(t, result, b, audit, issuer, expires)
}

func authenticatedHTTPClient(t *testing.T) (*sdk.ClientSession, *fakeBackend, *memoryAudit, *issuerFixture, time.Time, context.Context) {
	t.Helper()
	issuer := newIssuer(t)
	p := testPolicy(t)
	p.MCP = issuer.config()
	p.Grants = []model.Grant{{Subject: issuer.server.URL + "|alice", Client: "agent-a", Workload: "alpha", Operations: []string{"list", "read"}, Keys: []string{"SERVICE_KEY"}}}
	b := &fakeBackend{value: model.Value{Data: "HTTP_SYNTHETIC_MARKER"}}
	audit := &memoryAudit{}
	s := makeService(t, p, b, Options{Audit: audit})
	verify, err := newAccessTokenVerifier(context.Background(), p.MCP, issuer.server.Client())
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewUnstartedServer(http.NotFoundHandler())
	s.configured.MCP.Hosts = []string{server.Listener.Addr().String()}
	handler, err := s.httpHandler(verify)
	if err != nil {
		t.Fatal(err)
	}
	server.Config.Handler = handler
	server.Start()
	t.Cleanup(server.Close)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	t.Cleanup(cancel)
	expires := time.Now().Add(20 * time.Second).Truncate(time.Second)
	httpClient := &http.Client{Transport: bearerTransport{base: http.DefaultTransport, token: issuer.token(t, map[string]any{"exp": expires.Unix()})}}
	client, err := sdk.NewClient(&sdk.Implementation{Name: "http-fixture", Version: "1"}, nil).Connect(ctx, &sdk.StreamableClientTransport{Endpoint: server.URL + "/mcp", HTTPClient: httpClient}, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { client.Close() })
	return client, b, audit, issuer, expires, ctx
}

func assertAuthenticatedHTTPRead(t *testing.T, result *sdk.CallToolResult, backend *fakeBackend, audit *memoryAudit, issuer *issuerFixture, expires time.Time) {
	t.Helper()
	if !strings.Contains(result.Content[0].(*sdk.TextContent).Text, "HTTP_SYNTHETIC_MARKER") {
		t.Fatal("authorized HTTP read failed")
	}
	if len(audit.events) != 1 || audit.events[0].Subject != issuer.server.URL+"|alice" || audit.events[0].Client != "agent-a" {
		t.Fatal("actor receipt mismatch")
	}
	// Self-reported MCP client name never changes the token-bound principal.
	if backend.principal.Subject != issuer.server.URL+"|alice" {
		t.Fatal("untrusted client label became identity")
	}
	if !backend.principal.ExpiresAt.Equal(expires) {
		t.Fatalf("backend token expiry = %s, want %s", backend.principal.ExpiresAt, expires)
	}
	if !backend.hasDeadline || !backend.deadline.Equal(expires) {
		t.Fatalf("backend deadline = %s, want %s", backend.deadline, expires)
	}
}

func TestHTTPRejectsMissingOrExpiredTokenDeadline(t *testing.T) {
	p := testPolicy(t)
	p.MCP = model.MCP{Issuer: "https://issuer.example", JWKSURL: "https://issuer.example/keys", Audience: "https://cq.example/mcp", Hosts: []string{"cq.example"}}
	for _, expires := range []time.Time{{}, time.Unix(0, 0)} {
		s := makeService(t, p, &fakeBackend{}, Options{})
		h, err := s.httpHandler(func(context.Context, string, *http.Request) (*auth.TokenInfo, error) {
			return &auth.TokenInfo{UserID: "fixture|alice", Scopes: []string{"cq:mcp"}, Expiration: expires}, nil
		})
		if err != nil {
			t.Fatal(err)
		}
		r := httptest.NewRequest(http.MethodPost, "https://cq.example/mcp", strings.NewReader(`{}`))
		r.Header.Set("Authorization", "Bearer synthetic")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("token deadline %s status = %d, want %d", expires, w.Code, http.StatusUnauthorized)
		}
	}
}

func TestNoScopeAndInvalidJWTReachNoBackend(t *testing.T) {
	issuer := newIssuer(t)
	p := testPolicy(t)
	p.MCP = issuer.config()
	s := makeService(t, p, &fakeBackend{}, Options{})
	verify, err := newAccessTokenVerifier(context.Background(), p.MCP, issuer.server.Client())
	if err != nil {
		t.Fatal(err)
	}
	h, err := s.httpHandler(verify)
	if err != nil {
		t.Fatal(err)
	}
	r := httptest.NewRequest(http.MethodPost, "https://cq.example/mcp", strings.NewReader(`{}`))
	r.Header.Set("Authorization", "Bearer "+issuer.token(t, map[string]any{"scope": "cq:read"}))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 403 {
		t.Fatal("missing base scope", w.Code)
	}
	// Configuration validation never performs a request to the provider.
	_, err = s.HTTPHandler()
	if err != nil {
		t.Fatal(err)
	}
	if !secureResource(&url.URL{Scheme: "http", Host: "localhost:1234", Path: "/mcp"}) {
		t.Fatal("local resource rejected")
	}
}

func TestBoundedHTTPConcurrency(t *testing.T) {
	entered := make(chan struct{}, 16)
	release := make(chan struct{})
	h := protectHTTP(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { entered <- struct{}{}; <-release; w.WriteHeader(204) }), []string{"cq.example"}, nil)
	done := make(chan struct{}, 16)
	for range 16 {
		go func() {
			w := httptest.NewRecorder()
			h.ServeHTTP(w, httptest.NewRequest("GET", "https://cq.example/mcp", nil))
			done <- struct{}{}
		}()
	}
	for range 16 {
		<-entered
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", "https://cq.example/mcp", nil))
	if w.Code != 429 {
		t.Fatal("unbounded concurrency", w.Code)
	}
	close(release)
	for range 16 {
		<-done
	}
}

func TestMetadataNeverRetrievesValues(t *testing.T) {
	p := testPolicy(t)
	p.MCP = model.MCP{Issuer: "https://issuer.example", Audience: "https://cq.example/mcp", JWKSURL: "https://issuer.example/keys"}
	b := &fakeBackend{}
	s := makeService(t, p, b, Options{})
	h, err := s.httpHandler(func(context.Context, string, *http.Request) (*auth.TokenInfo, error) {
		return nil, auth.ErrInvalidToken
	})
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { r.Host = "cq.example"; h.ServeHTTP(w, r) }))
	defer server.Close()
	response, err := http.Get(server.URL + "/.well-known/oauth-protected-resource")
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if _, err := io.Copy(io.Discard, response.Body); err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != 200 || b.reads+b.runs+b.statuses != 0 {
		t.Fatal("metadata reached backend")
	}
}
