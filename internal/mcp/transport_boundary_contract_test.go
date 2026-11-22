package mcp

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/maldous/CircularQuay/internal/fault"
	"github.com/maldous/CircularQuay/internal/model"
	"github.com/modelcontextprotocol/go-sdk/auth"
)

func TestMCPTransportBoundaryContractChangedPolicyRefusesHandler(t *testing.T) {
	for _, invalid := range []bool{false, true} {
		t.Run(map[bool]string{false: "reload-error", true: "invalid-policy"}[invalid], func(t *testing.T) {
			assertMCPTransportChangedPolicy(t, invalid)
		})
	}
}

func assertMCPTransportChangedPolicy(t *testing.T, invalid bool) {
	t.Helper()
	config := mcpTransportBoundaryPolicy(t)
	backend := &fakeBackend{}
	calls := 0
	service := makeService(t, config, backend, Options{PolicyProvider: func() (model.Policy, error) {
		calls++
		if calls == 1 {
			return config, nil
		}
		if invalid {
			return model.Policy{Version: 99}, nil
		}
		return model.Policy{}, errors.New("SYNTHETIC_PRIVATE_RELOAD_DETAIL")
	}})
	handler, err := service.HTTPHandler()
	if handler != nil || calls != 2 {
		t.Fatalf("changed policy exposed a handler or used a stale snapshot: handler=%v calls=%d", handler != nil, calls)
	}
	requireMCPTransportBoundaryError(t, err, fault.Invalid, "mcp resource")
	assertMCPServiceRefusalNoBackendCalls(t, backend)
}

func mcpTransportBoundaryPolicy(t *testing.T) model.Policy {
	t.Helper()
	config := testPolicy(t)
	config.MCP = model.MCP{
		Issuer: "https://issuer.example.test", JWKSURL: "https://issuer.example.test/keys",
		Audience: "https://cq.example.test/mcp", Hosts: []string{"cq.example.test"},
		Origins: []string{"https://client.example.test"},
	}
	return config
}

func TestMCPTransportBoundaryContractForwardedHeadersCannotGrantTLS(t *testing.T) {
	reached := 0
	handler := mcpTransportBoundaryHandler(&reached)
	request := httptest.NewRequest(http.MethodPost, "http://cq.example.test/mcp", nil)
	request.RemoteAddr = "192.0.2.42:4321"
	request.Header.Set("Forwarded", "for=127.0.0.1;proto=https;host=cq.example.test")
	request.Header.Set("X-Forwarded-Proto", "https")
	request.Header.Set("Origin", "https://client.example.test")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	assertMCPTransportBoundaryResponse(t, response, http.StatusForbidden, "TLS required\n")
	if reached != 0 {
		t.Fatal("forwarded plaintext request reached the next handler")
	}
}

func TestMCPTransportBoundaryContractOnlyExactBrowserOriginReachesHandler(t *testing.T) {
	reached := 0
	handler := mcpTransportBoundaryHandler(&reached)
	for _, origin := range []string{"https://client.example.test", "https://client.example.test.attacker.test"} {
		request := httptest.NewRequest(http.MethodPost, "https://cq.example.test/mcp", nil)
		request.Header.Set("Origin", origin)
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		assertMCPTransportOrigin(t, response, origin)
	}
	if reached != 1 {
		t.Fatalf("browser-origin fence selected %d handlers, want one", reached)
	}
}

func mcpTransportBoundaryHandler(reached *int) http.Handler {
	return protectHTTP(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		(*reached)++
		w.WriteHeader(http.StatusNoContent)
	}), []string{"cq.example.test"}, []string{"https://client.example.test"})
}

func assertMCPTransportOrigin(t *testing.T, response *httptest.ResponseRecorder, origin string) {
	t.Helper()
	if origin == "https://client.example.test" {
		assertMCPTransportBoundaryResponse(t, response, http.StatusNoContent, "")
		return
	}
	assertMCPTransportBoundaryResponse(t, response, http.StatusForbidden, "request origin denied\n")
}

func assertMCPTransportBoundaryResponse(t *testing.T, response *httptest.ResponseRecorder, status int, body string) {
	t.Helper()
	if response.Code != status || response.Body.String() != body {
		t.Fatalf("transport response=%d %q, want %d %q", response.Code, response.Body.String(), status, body)
	}
	want := map[string]string{"Cache-Control": "private, no-store", "Pragma": "no-cache", "X-Content-Type-Options": "nosniff"}
	for name, value := range want {
		if response.Header().Get(name) != value {
			t.Fatalf("transport response omitted %s", name)
		}
	}
}

func TestMCPTransportBoundaryContractMissingDPoPProofNeverTouchesReplay(t *testing.T) {
	store := &dpopTestStore{nonce: "SYNTHETIC_PRIVATE_NONCE"}
	verified := 0
	reached := 0
	config := model.MCP{Resource: "https://cq.example.test/mcp", Authorities: map[string]model.IdentityAuthority{
		"bound": {SenderConstraint: &model.SenderConstraint{Type: "dpop", ProofMaxAgeSeconds: 60, Nonce: "required", ReplayStore: "local_persistent"}},
	}}
	options := accessTokenOptions{config: config, replay: store, metadataURL: "https://cq.example.test/meta"}
	options.verifier = func(_ context.Context, raw string, _ *http.Request) (*auth.TokenInfo, error) {
		verified++
		if raw != "SYNTHETIC_PRIVATE_TOKEN" {
			t.Fatal("verifier received a different synthetic token")
		}
		return &auth.TokenInfo{Expiration: time.Now().Add(time.Hour), Extra: map[string]any{
			"dpop_required": true, "authority": "bound", "authority_fingerprint": "synthetic-fingerprint", "confirmation_jkt": "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA",
		}}, nil
	}
	handler := requireAccessToken(options)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { reached++ }))
	request := httptest.NewRequest(http.MethodPost, "https://cq.example.test/mcp", nil)
	request.Header.Set("Authorization", "DPoP SYNTHETIC_PRIVATE_TOKEN")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	assertMCPTransportDPoPChallenge(t, response)
	if verified != 1 || reached != 0 || store.challenge != 0 || store.consume != 0 {
		t.Fatalf("missing proof crossed a verification/replay/effect boundary: verified=%d reached=%d challenges=%d consumes=%d", verified, reached, store.challenge, store.consume)
	}
}

func TestMCPTransportBoundaryContractInvalidDPoPTokenHasSafeChallenge(t *testing.T) {
	verified := 0
	reached := 0
	handler := requireAccessToken(accessTokenOptions{
		verifier: func(_ context.Context, raw string, _ *http.Request) (*auth.TokenInfo, error) {
			verified++
			if raw != "SYNTHETIC_PRIVATE_TOKEN" {
				t.Fatal("verifier received a different synthetic token")
			}
			return nil, errors.New("SYNTHETIC_PRIVATE_VERIFIER_DETAIL")
		},
		metadataURL: "https://cq.example.test/meta",
	})(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { reached++ }))
	request := httptest.NewRequest(http.MethodPost, "https://cq.example.test/mcp", nil)
	request.Header.Set("Authorization", "DPoP SYNTHETIC_PRIVATE_TOKEN")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	assertMCPTransportDPoPChallenge(t, response)
	if verified != 1 || reached != 0 {
		t.Fatalf("invalid token reached a handler: verified=%d reached=%d", verified, reached)
	}
}

func assertMCPTransportDPoPChallenge(t *testing.T, response *httptest.ResponseRecorder) {
	t.Helper()
	if response.Code != http.StatusUnauthorized || response.Body.String() != "invalid access token\n" {
		t.Fatalf("DPoP refusal exposed unexpected status or body: %d %q", response.Code, response.Body.String())
	}
	if response.Header().Get("WWW-Authenticate") != `DPoP resource_metadata="https://cq.example.test/meta"` || response.Header().Get("DPoP-Nonce") != "" {
		t.Fatal("DPoP refusal lost its fixed scheme, metadata URL or no-replay boundary")
	}
}

func TestMCPTransportBoundaryContractUnrepresentableAuditTimeNeverAppends(t *testing.T) {
	path := privateAuditPath(t)
	auditor := newTestAuditor(t, path)
	event := Event{Time: time.Date(2030, time.January, 2, 3, 4, 5, 0, time.UTC), Subject: "local:synthetic", Workload: "alpha", Operation: "read", Outcome: "release_authorized"}
	if err := auditor.Record(context.Background(), event); err != nil {
		t.Fatal("valid audit control refused", err)
	}
	want := readMCPTransportAudit(t, path)
	for _, year := range []int{-1, 10000} {
		event.Time = time.Date(year, time.January, 2, 3, 4, 5, 0, time.UTC)
		err := auditor.Record(context.Background(), event)
		requireMCPTransportBoundaryError(t, err, fault.Invalid, "audit receipt")
		if !bytes.Equal(readMCPTransportAudit(t, path), want) {
			t.Fatal("unrepresentable audit event changed the protected destination")
		}
	}
}

func readMCPTransportAudit(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func requireMCPTransportBoundaryError(t *testing.T, err error, kind fault.Kind, operation string) {
	t.Helper()
	var typed *fault.Error
	if !errors.As(err, &typed) {
		t.Fatalf("missing typed refusal: %v", err)
	}
	if err != typed || typed.Kind != kind || typed.Operation != operation || err.Error() != string(kind)+": "+operation {
		t.Fatalf("refusal changed category, operation or disclosure boundary: %v", err)
	}
}
