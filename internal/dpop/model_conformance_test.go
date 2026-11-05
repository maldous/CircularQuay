package dpop

import (
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"net/http"
	"testing"
	"time"

	jose "github.com/go-jose/go-jose/v4"
)

// TestModelConformanceSenderConstraint exercises the implementation traces
// represented by formal/SenderConstraint.tla using the durable local store.
func TestModelConformanceSenderConstraint(t *testing.T) {
	now := time.Date(2026, time.November, 4, 2, 0, 0, 0, time.UTC)
	state := privateStateDirectory(t)
	store, err := NewPersistentStore(state)
	if err != nil {
		t.Fatal(err)
	}
	store.now = func() time.Time { return now }
	keyA := conformanceKey(t)
	keyB := conformanceKey(t)
	token := "model-token"
	resource := "https://mcp.example.test/mcp"
	config := Config{Resource: resource, MaxAge: time.Minute, Skew: 5 * time.Second}
	verifier := Verifier{Config: config, Store: store, Now: func() time.Time { return now }}
	jktA := conformanceJKT(t, keyA)

	challenge := signedProof(t, keyA, resource, token, now, "model-jti-000001", "")
	_, err = verifier.Verify(context.Background(), conformanceRequest(t, resource, challenge), token, "fp-a", jktA, now.Add(time.Hour))
	var nonceError *NonceError
	if !errors.As(err, &nonceError) || nonceError.Value == "" {
		t.Fatalf("missing required nonce challenge: %v", err)
	}

	proof := signedProof(t, keyA, resource, token, now, "model-jti-000002", nonceError.Value)
	assertConformanceInvalid(t, verifier, conformanceAttempt{request: conformanceRequest(t, resource, proof), token: token, authority: "fp-b", jkt: jktA, expiry: now.Add(time.Hour)}, "authority fingerprint did not fence nonce-bound proof")
	assertConformanceInvalid(t, verifier, conformanceAttempt{request: conformanceRequest(t, resource, proof), token: "other-token", authority: "fp-a", jkt: jktA, expiry: now.Add(time.Hour)}, "token binding mismatch accepted")
	assertConformanceInvalid(t, verifier, conformanceAttempt{request: conformanceRequest(t, resource, proof), token: token, authority: "fp-a", jkt: conformanceJKT(t, keyB), expiry: now.Add(time.Hour)}, "key binding mismatch accepted")
	wrongPath := conformanceRequest(t, "https://mcp.example.test/other", proof)
	assertConformanceInvalid(t, verifier, conformanceAttempt{request: wrongPath, token: token, authority: "fp-a", jkt: jktA, expiry: now.Add(time.Hour)}, "resource binding mismatch accepted")

	if _, err := verifier.Verify(context.Background(), conformanceRequest(t, resource, proof), token, "fp-a", jktA, now.Add(time.Hour)); err != nil {
		t.Fatalf("valid binding did not commit atomically: %v", err)
	}
	restarted, err := NewPersistentStore(state)
	if err != nil {
		t.Fatal(err)
	}
	restarted.now = func() time.Time { return now }
	verifier.Store = restarted
	assertConformanceInvalid(t, verifier, conformanceAttempt{request: conformanceRequest(t, resource, proof), token: token, authority: "fp-a", jkt: jktA, expiry: now.Add(time.Hour)}, "restart cleared replay fence")

	assertConformanceCommitExpiryFence(t, config, keyA, jktA)
}

type conformanceAttempt struct {
	request   *http.Request
	token     string
	authority string
	jkt       string
	expiry    time.Time
}

func assertConformanceInvalid(t *testing.T, verifier Verifier, attempt conformanceAttempt, message string) {
	t.Helper()
	if _, err := verifier.Verify(context.Background(), attempt.request, attempt.token, attempt.authority, attempt.jkt, attempt.expiry); !errors.Is(err, ErrInvalid) {
		t.Fatalf("%s: %v", message, err)
	}
}

// assertConformanceCommitExpiryFence advances only the store clock between
// proof verification and Consume. The old nonce remains live, while the new
// token/proof expiry closes exactly at the store's linearization point.
func assertConformanceCommitExpiryFence(t *testing.T, config Config, key *ecdsa.PrivateKey, jkt string) {
	t.Helper()
	validationTime := time.Date(2026, time.November, 4, 3, 0, 0, 0, time.UTC)
	commitTime := validationTime.Add(time.Second)
	store, err := NewPersistentStore(privateStateDirectory(t))
	if err != nil {
		t.Fatal(err)
	}
	store.now = func() time.Time { return validationTime }
	verifier := Verifier{Config: config, Store: store, Now: func() time.Time { return validationTime }}
	oldToken := "long-lived-token"
	challenge := signedProof(t, key, config.Resource, oldToken, validationTime, "model-jti-000003", "")
	_, err = verifier.Verify(context.Background(), conformanceRequest(t, config.Resource, challenge), oldToken, "fp-a", jkt, validationTime.Add(time.Hour))
	var nonceError *NonceError
	if !errors.As(err, &nonceError) || nonceError.Value == "" {
		t.Fatalf("long-lived token did not receive nonce: %v", err)
	}
	store.now = func() time.Time { return commitTime }
	shortToken := "short-lived-token"
	proof := signedProof(t, key, config.Resource, shortToken, validationTime, "model-jti-000004", nonceError.Value)
	assertConformanceInvalid(t, verifier, conformanceAttempt{request: conformanceRequest(t, config.Resource, proof), token: shortToken, authority: "fp-a", jkt: jkt, expiry: commitTime}, "store commit accepted proof at expiry")
	cache, err := store.read()
	if err != nil {
		t.Fatal(err)
	}
	binding := digest("fp-a", jkt, config.Resource)
	if _, exists := cache.Entries[nonceKey(nonceError.Value)]; !exists {
		t.Fatal("expiry failure consumed the prior nonce")
	}
	if _, exists := cache.Entries[digest(binding, "model-jti-000004")]; exists {
		t.Fatal("expiry failure persisted a replay entry")
	}
}

func conformanceKey(t *testing.T) *ecdsa.PrivateKey {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	return key
}

func conformanceJKT(t *testing.T, key *ecdsa.PrivateKey) string {
	t.Helper()
	jwk := jose.JSONWebKey{Key: &key.PublicKey}
	thumbprint, err := jwk.Thumbprint(crypto.SHA256)
	if err != nil {
		t.Fatal(err)
	}
	return base64.RawURLEncoding.EncodeToString(thumbprint)
}

func conformanceRequest(t *testing.T, resource, proof string) *http.Request {
	t.Helper()
	request, err := http.NewRequest(http.MethodPost, resource, nil)
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("DPoP", proof)
	return request
}
