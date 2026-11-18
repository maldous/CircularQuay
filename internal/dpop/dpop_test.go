package dpop

import (
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	jose "github.com/go-jose/go-jose/v4"
	"github.com/golang-jwt/jwt/v5"
)

func TestPersistentVerifierRequiresNonceAndRejectsReplayAfterRestart(t *testing.T) {
	now := time.Date(2026, time.November, 4, 1, 2, 3, 0, time.UTC)
	directory := privateStateDirectory(t)
	store, err := NewPersistentStore(directory)
	if err != nil {
		t.Fatal(err)
	}
	store.now = func() time.Time { return now }
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	token := "synthetic.access.token"
	jwkForThumbprint := jose.JSONWebKey{Key: &key.PublicKey}
	thumbprint, thumbprintErr := jwkForThumbprint.Thumbprint(crypto.SHA256)
	if thumbprintErr != nil {
		t.Fatal(thumbprintErr)
	}
	jkt := base64.RawURLEncoding.EncodeToString(thumbprint)
	config := Config{Resource: "https://mcp.example.test/mcp", MaxAge: time.Minute, Skew: 5 * time.Second}
	verifier := Verifier{Config: config, Store: store, Now: func() time.Time { return now }}
	request := func(proof string) *http.Request { return httptestNewRequest(t, proof) }
	first := signedProof(t, key, config.Resource, token, now, "vXFaVJsSkTFyMhYx", "")
	_, err = verifier.Verify(context.Background(), request(first), token, "authority", jkt, now.Add(time.Hour))
	var nonceErr *NonceError
	if !errors.As(err, &nonceErr) || nonceErr.Value == "" {
		t.Fatalf("first request did not receive nonce: %v", err)
	}
	proof := signedProof(t, key, config.Resource, token, now, "QOoycXicq0HdwpdT", nonceErr.Value)
	next, err := verifier.Verify(context.Background(), request(proof), token, "authority", jkt, now.Add(time.Hour))
	if err != nil || next == "" {
		t.Fatalf("nonce-bound proof failed: next=%q err=%v", next, err)
	}
	restarted, err := NewPersistentStore(directory)
	if err != nil {
		t.Fatal(err)
	}
	restarted.now = func() time.Time { return now }
	verifier.Store = restarted
	_, err = verifier.Verify(context.Background(), request(proof), token, "authority", jkt, now.Add(time.Hour))
	if !errors.Is(err, ErrInvalid) {
		t.Fatalf("replayed proof accepted after restart: %v", err)
	}
}

func TestPersistentStoreConcurrentConsumeAdmitsOne(t *testing.T) {
	directory := privateStateDirectory(t)
	store, err := NewPersistentStore(directory)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, time.November, 4, 0, 0, 0, 0, time.UTC)
	store.now = func() time.Time { return now }
	binding := digest("binding")
	nonce, err := store.Challenge(context.Background(), binding, now.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	results := make(chan error, 2)
	for range 2 {
		go func() {
			_, err := store.Consume(context.Background(), binding, nonce, digest("replay"), now.Add(time.Minute))
			results <- err
		}()
	}
	accepted := 0
	for range 2 {
		if <-results == nil {
			accepted++
		}
	}
	if accepted != 1 {
		t.Fatalf("consume accepted %d requests, want exactly one", accepted)
	}
	cache, err := store.read()
	if err != nil {
		t.Fatalf("read cache after concurrent consume: %v", err)
	}
	if len(cache.Entries) != 2 {
		t.Fatalf("cache has %d entries after accepted consume, want replay and next nonce", len(cache.Entries))
	}
}

func signedProof(t *testing.T, key *ecdsa.PrivateKey, resource, token string, now time.Time, jti, nonce string) string {
	t.Helper()
	jwk, jwkErr := json.Marshal(jose.JSONWebKey{Key: &key.PublicKey})
	if jwkErr != nil {
		t.Fatal(jwkErr)
	}
	header := map[string]any{"typ": "dpop+jwt", "alg": "ES256", "jwk": json.RawMessage(jwk)}
	h, err := json.Marshal(header)
	if err != nil {
		t.Fatal(err)
	}
	hash := sha256.Sum256([]byte(token))
	claims := map[string]any{"jti": jti, "htm": "POST", "htu": resource, "iat": now.Unix(), "ath": base64.RawURLEncoding.EncodeToString(hash[:])}
	if nonce != "" {
		claims["nonce"] = nonce
	}
	p, err := json.Marshal(claims)
	if err != nil {
		t.Fatal(err)
	}
	signed := base64.RawURLEncoding.EncodeToString(h) + "." + base64.RawURLEncoding.EncodeToString(p)
	signature, err := jwt.SigningMethodES256.Sign(signed, key)
	if err != nil {
		t.Fatal(err)
	}
	return signed + "." + base64.RawURLEncoding.EncodeToString(signature)
}

func httptestNewRequest(t *testing.T, proof string) *http.Request {
	t.Helper()
	request, err := http.NewRequest(http.MethodPost, "https://mcp.example.test/mcp", nil)
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("DPoP", proof)
	return request
}

func privateStateDirectory(t *testing.T) string {
	t.Helper()
	base, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(base, 0o700); err != nil {
		t.Fatal(err)
	}
	return filepath.Join(base, "state")
}
