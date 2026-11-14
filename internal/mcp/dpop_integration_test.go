package mcp

import (
	"bytes"
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	jose "github.com/go-jose/go-jose/v4"
	"github.com/golang-jwt/jwt/v5"
	"github.com/maldous/CircularQuay/internal/model"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestCompleteDPoPHTTPMCPReadRejectsReplayAndBearerDowngrade(t *testing.T) {
	fixture := newDPoPIntegrationFixture(t)
	fixture.assertAuthorizedRead(t)
	fixture.assertReplayDenied(t)
	fixture.assertBearerDowngradeDenied(t)
}

type dpopIntegrationFixture struct {
	issuer    *issuerFixture
	token     string
	client    *sdk.ClientSession
	backend   *fakeBackend
	audit     *memoryAudit
	transport *dpopIntegrationTransport
}

func newDPoPIntegrationFixture(t *testing.T) dpopIntegrationFixture {
	t.Helper()
	issuer := newIssuer(t)
	clientKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	jkt := dpopThumbprint(t, clientKey)
	policy := testPolicy(t)
	server := httptest.NewUnstartedServer(http.NotFoundHandler())
	resource := "http://" + server.Listener.Addr().String() + "/mcp"
	policy.MCP = model.MCP{Resource: resource, Hosts: []string{server.Listener.Addr().String()}, AuditPath: privateAuditPath(t), Authorities: map[string]model.IdentityAuthority{
		"workforce": {Issuer: issuer.server.URL, JWKSURL: issuer.server.URL + "/keys", Audience: "https://cq.example/mcp", Profile: "rfc9068", Clients: []string{"agent-a"}, Subjects: []string{issuer.server.URL + "|alice"}, RequiredScopes: []string{"session:approved"}, Permissions: map[string]model.TokenPermission{"read": {Scopes: []string{"secret:use"}}}, SenderConstraint: &model.SenderConstraint{Type: "dpop", ProofMaxAgeSeconds: 60, Nonce: "required", ReplayStore: "local_persistent"}},
	}}
	policy.Grants = []model.Grant{{Authority: "workforce", Subject: issuer.server.URL + "|alice", Client: "agent-a", Workload: "alpha", Operations: []string{"read"}, Keys: []string{"SERVICE_KEY"}}}
	backend := &fakeBackend{value: model.Value{Data: "COMPLETE_DPOP_SYNTHETIC_SECRET"}}
	audit := &memoryAudit{}
	service := makeService(t, policy, backend, Options{Audit: audit, DPoPStateDirectory: filepath.Dir(privateAuditPath(t))})
	verify, err := newAccessTokenVerifier(context.Background(), policy.MCP, issuer.server.Client())
	if err != nil {
		t.Fatal(err)
	}
	handler, err := service.httpHandler(verify)
	if err != nil {
		t.Fatal(err)
	}
	server.Config.Handler = handler
	server.Start()
	t.Cleanup(server.Close)
	token := issuer.token(t, map[string]any{"scope": "session:approved secret:use", "cnf": map[string]any{"jkt": jkt}})
	transport := &dpopIntegrationTransport{base: http.DefaultTransport, token: token, key: clientKey}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	t.Cleanup(cancel)
	client, err := sdk.NewClient(&sdk.Implementation{Name: "dpop-integration", Version: "1"}, nil).Connect(ctx, &sdk.StreamableClientTransport{Endpoint: resource, HTTPClient: &http.Client{Transport: transport}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { client.Close() })
	return dpopIntegrationFixture{issuer: issuer, token: token, client: client, backend: backend, audit: audit, transport: transport}
}

func (f dpopIntegrationFixture) assertAuthorizedRead(t *testing.T) {
	t.Helper()
	result, err := f.client.CallTool(context.Background(), &sdk.CallToolParams{Name: "read_secret", Arguments: map[string]any{"workload": "alpha", "key": "SERVICE_KEY"}})
	if err != nil || result.IsError || len(result.Content) != 1 || !strings.Contains(result.Content[0].(*sdk.TextContent).Text, "COMPLETE_DPOP_SYNTHETIC_SECRET") {
		t.Fatalf("authorized DPoP read failed: result=%v err=%v", result, err)
	}
	if f.backend.reads != 1 || f.backend.principal.Subject != f.issuer.server.URL+"|alice" || f.backend.principal.Client != "agent-a" || !f.backend.hasDeadline || len(f.audit.events) != 1 {
		t.Fatalf("authorized read lost actor, deadline, or audit: reads=%d principal=%+v deadline=%v audit=%d", f.backend.reads, f.backend.principal, f.backend.hasDeadline, len(f.audit.events))
	}
}

func (f dpopIntegrationFixture) assertReplayDenied(t *testing.T) {
	t.Helper()
	f.assertDeniedBeforeBackend(t, f.transport.recorded(t, "DPoP "+f.token), "replayed proof")
}

func (f dpopIntegrationFixture) assertBearerDowngradeDenied(t *testing.T) {
	t.Helper()
	downgrade := f.transport.fresh(t, "Bearer "+f.token)
	accepted := cloneDPoPRequest(t, downgrade)
	f.assertDeniedBeforeBackend(t, downgrade, "bearer downgrade")
	accepted.Header.Set("Authorization", "DPoP "+f.token)
	f.assertFreshProofAccepted(t, accepted)
}

func (f dpopIntegrationFixture) assertDeniedBeforeBackend(t *testing.T, request *http.Request, scenario string) {
	t.Helper()
	response, err := (&http.Client{}).Do(request)
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if response.StatusCode != http.StatusUnauthorized || f.backend.reads != 1 || len(f.audit.events) != 1 {
		t.Fatalf("%s reached protected backend: status=%d reads=%d audit=%d", scenario, response.StatusCode, f.backend.reads, len(f.audit.events))
	}
}

func (f dpopIntegrationFixture) assertFreshProofAccepted(t *testing.T, request *http.Request) {
	t.Helper()
	response, err := (&http.Client{}).Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	f.transport.rememberNonce(response)
	if response.StatusCode != http.StatusOK || f.backend.reads != 2 || len(f.audit.events) != 2 {
		t.Fatalf("fresh DPoP proof was consumed by bearer denial: status=%d reads=%d audit=%d", response.StatusCode, f.backend.reads, len(f.audit.events))
	}
}

type dpopIntegrationTransport struct {
	base                  http.RoundTripper
	token                 string
	key                   *ecdsa.PrivateKey
	mu                    sync.Mutex
	nonce                 string
	sequence              uint64
	method, target, proof string
	body                  []byte
	header                http.Header
}

func (d *dpopIntegrationTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	body, err := dpopRequestBody(request)
	if err != nil {
		return nil, err
	}
	for attempt := 0; attempt < 2; attempt++ {
		response, proof, err := d.sendProof(request, body)
		if err != nil || !d.retryNonce(response, attempt) {
			d.recordToolCall(request, body, proof, response, err)
			return response, err
		}
		d.nonce = response.Header.Get("DPoP-Nonce")
		response.Body.Close()
	}
	return nil, nil
}

func (d *dpopIntegrationTransport) sendProof(request *http.Request, body []byte) (*http.Response, string, error) {
	d.sequence++
	out := request.Clone(request.Context())
	out.Body = io.NopCloser(bytes.NewReader(body))
	out.GetBody = func() (io.ReadCloser, error) { return io.NopCloser(bytes.NewReader(body)), nil }
	out.ContentLength = int64(len(body))
	proof, err := dpopIntegrationProof(d.key, out.Method, out.URL.String(), d.token, d.nonce, d.sequence)
	if err != nil {
		return nil, "", err
	}
	out.Header = request.Header.Clone()
	out.Header.Set("Authorization", "DPoP "+d.token)
	out.Header.Set("DPoP", proof)
	response, err := d.base.RoundTrip(out)
	return response, proof, err
}

func (d *dpopIntegrationTransport) retryNonce(response *http.Response, attempt int) bool {
	return response != nil && attempt == 0 && response.StatusCode == http.StatusUnauthorized && response.Header.Get("DPoP-Nonce") != ""
}

func (d *dpopIntegrationTransport) recordToolCall(request *http.Request, body []byte, proof string, response *http.Response, err error) {
	if err != nil || response == nil || response.StatusCode >= http.StatusBadRequest {
		return
	}
	if nonce := response.Header.Get("DPoP-Nonce"); nonce != "" {
		d.nonce = nonce
	}
	if bytes.Contains(body, []byte(`"method":"tools/call"`)) {
		d.method, d.target, d.proof, d.body, d.header = request.Method, request.URL.String(), proof, append([]byte(nil), body...), request.Header.Clone()
	}
}

func (d *dpopIntegrationTransport) recorded(t *testing.T, authorization string) *http.Request {
	t.Helper()
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.target == "" || d.proof == "" {
		t.Fatal("no successful DPoP protocol request recorded")
	}
	request, err := http.NewRequest(d.method, d.target, bytes.NewReader(d.body))
	if err != nil {
		t.Fatal(err)
	}
	request.Header = d.header.Clone()
	request.Header.Set("Authorization", authorization)
	request.Header.Set("DPoP", d.proof)
	return request
}

func (d *dpopIntegrationTransport) fresh(t *testing.T, authorization string) *http.Request {
	t.Helper()
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.target == "" || d.proof == "" {
		t.Fatal("no successful DPoP protocol request recorded")
	}
	d.sequence++
	proof, err := dpopIntegrationProof(d.key, d.method, d.target, d.token, d.nonce, d.sequence)
	if err != nil {
		t.Fatal(err)
	}
	request, err := http.NewRequest(d.method, d.target, bytes.NewReader(d.body))
	if err != nil {
		t.Fatal(err)
	}
	request.Header = d.header.Clone()
	request.Header.Set("Authorization", authorization)
	request.Header.Set("DPoP", proof)
	return request
}

func (d *dpopIntegrationTransport) rememberNonce(response *http.Response) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if response.StatusCode < http.StatusBadRequest {
		d.nonce = response.Header.Get("DPoP-Nonce")
	}
}

func cloneDPoPRequest(t *testing.T, request *http.Request) *http.Request {
	t.Helper()
	clone := request.Clone(request.Context())
	body, err := request.GetBody()
	if err != nil {
		t.Fatal(err)
	}
	clone.Body = body
	return clone
}

func dpopRequestBody(request *http.Request) ([]byte, error) {
	if request.GetBody != nil {
		body, err := request.GetBody()
		if err != nil {
			return nil, err
		}
		defer body.Close()
		return io.ReadAll(io.LimitReader(body, maximumRequestBytes+1))
	}
	if request.Body == nil {
		return nil, nil
	}
	body, err := io.ReadAll(io.LimitReader(request.Body, maximumRequestBytes+1))
	request.Body.Close()
	request.Body = io.NopCloser(bytes.NewReader(body))
	return body, err
}

func dpopIntegrationProof(key *ecdsa.PrivateKey, method, target, token, nonce string, sequence uint64) (string, error) {
	jwk, err := json.Marshal(jose.JSONWebKey{Key: &key.PublicKey})
	if err != nil {
		return "", err
	}
	header, err := json.Marshal(map[string]any{"typ": "dpop+jwt", "alg": "ES256", "jwk": json.RawMessage(jwk)})
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256([]byte(token))
	claims, err := json.Marshal(map[string]any{"jti": base64.RawURLEncoding.EncodeToString([]byte(time.Now().UTC().Format(time.RFC3339Nano))) + base64.RawURLEncoding.EncodeToString([]byte{byte(sequence)}), "htm": method, "htu": target, "iat": time.Now().Unix(), "ath": base64.RawURLEncoding.EncodeToString(sum[:]), "nonce": nonce})
	if err != nil {
		return "", err
	}
	signed := base64.RawURLEncoding.EncodeToString(header) + "." + base64.RawURLEncoding.EncodeToString(claims)
	signature, err := jwt.SigningMethodES256.Sign(signed, key)
	if err != nil {
		return "", err
	}
	return signed + "." + base64.RawURLEncoding.EncodeToString(signature), nil
}

func dpopThumbprint(t *testing.T, key *ecdsa.PrivateKey) string {
	t.Helper()
	jwk := jose.JSONWebKey{Key: &key.PublicKey}
	thumbprint, err := jwk.Thumbprint(crypto.SHA256)
	if err != nil {
		t.Fatal(err)
	}
	return base64.RawURLEncoding.EncodeToString(thumbprint)
}
