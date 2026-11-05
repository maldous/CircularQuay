package mcp

import (
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	jose "github.com/go-jose/go-jose/v4"
	"github.com/golang-jwt/jwt/v5"
	"github.com/maldous/CircularQuay/internal/model"
	"github.com/modelcontextprotocol/go-sdk/auth"
)

func TestRequireAccessTokenSeparatesBearerAndDPoPAuthorities(t *testing.T) {
	config := model.MCP{Resource: "https://mcp.example.test/mcp", Authorities: map[string]model.IdentityAuthority{
		"bound": {SenderConstraint: &model.SenderConstraint{Type: "dpop", ProofMaxAgeSeconds: 60, Nonce: "required", ReplayStore: "local_persistent"}},
	}}
	verifier := func(_ context.Context, raw string, _ *http.Request) (*auth.TokenInfo, error) {
		if raw == "bearer" {
			return &auth.TokenInfo{Expiration: time.Now().Add(time.Minute), Extra: map[string]any{}}, nil
		}
		if raw == "bound" {
			return &auth.TokenInfo{Expiration: time.Now().Add(time.Minute), Extra: map[string]any{"dpop_required": true, "authority": "bound", "authority_fingerprint": "fingerprint", "confirmation_jkt": "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"}}, nil
		}
		return nil, auth.ErrInvalidToken
	}
	handler := requireAccessToken(accessTokenOptions{verifier: verifier, config: config, metadataURL: "https://mcp.example.test/.well-known/oauth-protected-resource/mcp"})(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) }))
	request := httptest.NewRequest(http.MethodPost, "https://mcp.example.test/mcp", nil)
	request.Header.Set("Authorization", "Bearer bearer")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusNoContent {
		t.Fatalf("ordinary bearer authority denied: %d", response.Code)
	}
	request = httptest.NewRequest(http.MethodPost, "https://mcp.example.test/mcp", nil)
	request.Header.Set("Authorization", "Bearer bound")
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusUnauthorized || response.Header().Get("WWW-Authenticate") == "" {
		t.Fatalf("bound token downgraded or unchallenged: %d %q", response.Code, response.Header().Get("WWW-Authenticate"))
	}
	request = httptest.NewRequest(http.MethodPost, "https://mcp.example.test/mcp", nil)
	request.Header.Add("Authorization", "Bearer bearer")
	request.Header.Add("Authorization", "Bearer bearer")
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("duplicate authorization header accepted: %d", response.Code)
	}
}

func TestRequestTokenRejectsAmbiguousAndMalformedHeaders(t *testing.T) {
	for _, values := range [][]string{nil, {"Bearer"}, {"Bearer one two"}, {"Basic token"}, {"Bearer a", "Bearer b"}} {
		if _, _, ok := requestToken(values); ok {
			t.Fatalf("ambiguous header accepted: %#v", values)
		}
	}
}

type dpopTestStore struct {
	nonce              string
	challenge, consume int
}

func (s *dpopTestStore) Challenge(context.Context, string, time.Time) (string, error) {
	s.challenge++
	return s.nonce, nil
}
func (s *dpopTestStore) Consume(context.Context, string, string, string, time.Time) (string, error) {
	s.consume++
	return "next", nil
}

func TestRequireAccessTokenValidDPoPProofAndSchemeFence(t *testing.T) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	jwk := jose.JSONWebKey{Key: &key.PublicKey}
	thumb, err := jwk.Thumbprint(crypto.SHA256)
	if err != nil {
		t.Fatal(err)
	}
	jkt := base64.RawURLEncoding.EncodeToString(thumb)
	config := model.MCP{Resource: "https://mcp.example.test/mcp", Authorities: map[string]model.IdentityAuthority{"bound": {SenderConstraint: &model.SenderConstraint{Type: "dpop", ProofMaxAgeSeconds: 60, Nonce: "required", ReplayStore: "local_persistent"}}, "ordinary": {}}}
	info := &auth.TokenInfo{Expiration: time.Now().Add(time.Minute), Extra: map[string]any{"dpop_required": true, "authority": "bound", "authority_fingerprint": "fingerprint", "confirmation_jkt": jkt}}
	calls := 0
	verifier := func(context.Context, string, *http.Request) (*auth.TokenInfo, error) { calls++; return info, nil }
	store := &dpopTestStore{nonce: "nonce"}
	reached := 0
	h := requireAccessToken(accessTokenOptions{verifier: verifier, config: config, replay: store, metadataURL: "https://mcp.example.test/meta"})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if auth.TokenInfoFromContext(r.Context()) != info {
			t.Error("SDK token missing")
		}
		reached++
		w.WriteHeader(204)
	}))
	proof := signedDPoP(t, key, config.Resource, "bound", time.Now(), "1234567890123456", "")
	r := httptest.NewRequest("POST", config.Resource, nil)
	r.Header.Set("Authorization", "DPoP bound")
	r.Header.Set("DPoP", proof)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	assertDPoPChallenge(t, w, reached, store.challenge)
	proof = signedDPoP(t, key, config.Resource, "bound", time.Now(), "1234567890123457", "nonce")
	r = httptest.NewRequest("POST", config.Resource, nil)
	r.Header.Set("Authorization", "DPoP bound")
	r.Header.Set("DPoP", proof)
	w = httptest.NewRecorder()
	h.ServeHTTP(w, r)
	assertDPoPSuccess(t, w, reached, calls, store.consume)
	r = httptest.NewRequest("POST", config.Resource, nil)
	r.Header.Set("Authorization", "Bearer bound")
	r.Header.Set("DPoP", proof)
	w = httptest.NewRecorder()
	h.ServeHTTP(w, r)
	assertNoDowngrade(t, w, store.consume)
}

func assertDPoPChallenge(t *testing.T, w *httptest.ResponseRecorder, reached, challenges int) {
	t.Helper()
	if w.Code != http.StatusUnauthorized || w.Header().Get("DPoP-Nonce") != "nonce" || reached != 0 || challenges != 1 {
		t.Fatal("nonce challenge failed")
	}
}

func assertDPoPSuccess(t *testing.T, w *httptest.ResponseRecorder, reached, calls, consumes int) {
	t.Helper()
	if w.Code != http.StatusNoContent || reached != 1 || calls != 2 || consumes != 1 {
		t.Fatal("valid DPoP handoff failed")
	}
}

func assertNoDowngrade(t *testing.T, w *httptest.ResponseRecorder, consumes int) {
	t.Helper()
	if w.Code != http.StatusUnauthorized || consumes != 1 {
		t.Fatal("bearer downgrade consumed proof")
	}
}

func signedDPoP(t *testing.T, key *ecdsa.PrivateKey, uri, token string, now time.Time, jti, nonce string) string {
	t.Helper()
	jwk, _ := json.Marshal(jose.JSONWebKey{Key: &key.PublicKey})
	h, _ := json.Marshal(map[string]any{"typ": "dpop+jwt", "alg": "ES256", "jwk": json.RawMessage(jwk)})
	sum := sha256.Sum256([]byte(token))
	c := map[string]any{"jti": jti, "htm": "POST", "htu": uri, "iat": now.Unix(), "ath": base64.RawURLEncoding.EncodeToString(sum[:])}
	if nonce != "" {
		c["nonce"] = nonce
	}
	p, _ := json.Marshal(c)
	signed := base64.RawURLEncoding.EncodeToString(h) + "." + base64.RawURLEncoding.EncodeToString(p)
	sig, err := jwt.SigningMethodES256.Sign(signed, key)
	if err != nil {
		t.Fatal(err)
	}
	return signed + "." + base64.RawURLEncoding.EncodeToString(sig)
}
