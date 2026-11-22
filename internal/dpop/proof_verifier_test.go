package dpop

import (
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	jose "github.com/go-jose/go-jose/v4"
	"github.com/golang-jwt/jwt/v5"
)

const proofResource = "https://mcp.example.test/mcp"

func TestVerifierAcceptsES256AndRS256Proofs(t *testing.T) {
	now := proofNow()
	for _, test := range []struct {
		name string
		key  crypto.Signer
	}{
		{name: "ES256", key: proofECKey(t)},
		{name: "RS256", key: proofRSAKey(t, 2048)},
	} {
		t.Run(test.name, func(t *testing.T) {
			store := &proofStore{}
			proof, jkt := validProof(t, test.key, now, "nonce")
			next, err := proofVerifier(now, store).Verify(context.Background(), proofRequest(t, proofResource, proof), "synthetic-token", "authority", jkt, now.Add(time.Minute))
			if err != nil || next == "" || store.consumes != 1 || store.challenges != 0 {
				t.Fatalf("valid %s proof rejected: next=%q err=%v", test.name, next, err)
			}
		})
	}
}

func TestVerifierRejectsBindingMismatchesBeforeStore(t *testing.T) {
	now := proofNow()
	key := proofECKey(t)
	proof, jkt := validProof(t, key, now, "nonce")
	for _, test := range []struct {
		name    string
		proof   string
		request string
		token   string
		jkt     string
		expiry  time.Time
	}{
		{name: "method", proof: proofWithClaim(t, key, now, "htm", "GET"), request: proofResource, token: "synthetic-token", jkt: jkt, expiry: now.Add(time.Minute)},
		{name: "resource", proof: proofWithClaim(t, key, now, "htu", proofResource+"/other"), request: proofResource, token: "synthetic-token", jkt: jkt, expiry: now.Add(time.Minute)},
		{name: "access token", proof: proofWithToken(t, key, now, "another-token"), request: proofResource, token: "synthetic-token", jkt: jkt, expiry: now.Add(time.Minute)},
		{name: "thumbprint", proof: proofWithKey(t, proofECKey(t), now), request: proofResource, token: "synthetic-token", jkt: jkt, expiry: now.Add(time.Minute)},
		{name: "stale", proof: proofWithIssuedAt(t, key, now.Add(-2*time.Minute)), request: proofResource, token: "synthetic-token", jkt: jkt, expiry: now.Add(time.Minute)},
		{name: "future", proof: proofWithIssuedAt(t, key, now.Add(time.Minute)), request: proofResource, token: "synthetic-token", jkt: jkt, expiry: now.Add(time.Minute)},
		{name: "actual path", proof: proof, request: proofResource + "/other", token: "synthetic-token", jkt: jkt, expiry: now.Add(time.Minute)},
		{name: "actual query", proof: proof, request: proofResource + "?other=1", token: "synthetic-token", jkt: jkt, expiry: now.Add(time.Minute)},
		{name: "expired token", proof: proof, request: proofResource, token: "synthetic-token", jkt: jkt, expiry: now.Add(-time.Second)},
	} {
		t.Run(test.name, func(t *testing.T) {
			store := &proofStore{}
			_, err := proofVerifier(now, store).Verify(context.Background(), proofRequest(t, test.request, test.proof), test.token, "authority", test.jkt, test.expiry)
			if err == nil || store.challenges != 0 || store.consumes != 0 {
				t.Fatalf("invalid proof reached store: %v", err)
			}
		})
	}
}

func TestVerifierRejectsMultipleProofHeadersBeforeStore(t *testing.T) {
	now := proofNow()
	key := proofECKey(t)
	proof, jkt := validProof(t, key, now, "nonce")
	request := proofRequest(t, proofResource, proof)
	request.Header.Add("DPoP", proof)
	store := &proofStore{}
	_, err := proofVerifier(now, store).Verify(context.Background(), request, "synthetic-token", "authority", jkt, now.Add(time.Minute))
	if err == nil || store.challenges != 0 || store.consumes != 0 {
		t.Fatalf("multiple proofs reached store: %v", err)
	}
}

func TestVerifierBoundsReplayLifetimeToTokenExpiry(t *testing.T) {
	now := proofNow()
	key := proofECKey(t)
	proof, jkt := validProof(t, key, now, "nonce")
	store := &proofStore{}
	expiry := now.Add(10 * time.Second)
	if _, err := proofVerifier(now, store).Verify(context.Background(), proofRequest(t, proofResource, proof), "synthetic-token", "authority", jkt, expiry); err != nil || !store.expires.Equal(expiry) {
		t.Fatalf("token expiry did not bound replay lifetime: %v", err)
	}
}

func TestParseProofRejectsMalformedProtectedHeaders(t *testing.T) {
	key := proofECKey(t)
	now := proofNow()
	for _, test := range []struct {
		name   string
		header map[string]any
		body   string
	}{
		{name: "unknown", header: map[string]any{"kid": "unexpected"}},
		{name: "critical", header: map[string]any{"crit": []string{"typ"}}},
		{name: "algorithm", header: map[string]any{"alg": "HS256"}},
		{name: "duplicate payload", body: `{"jti":"first","jti":"second"}`},
		{name: "null payload", body: `{"jti":null,"htm":"POST","htu":"https://mcp.example.test/mcp","iat":1,"ath":"x","nonce":"nonce"}`},
	} {
		t.Run(test.name, func(t *testing.T) {
			proof := rawProof(t, key, now, test.header, test.body)
			if _, err := parseProof([]string{proof}); err == nil {
				t.Fatal("malformed protected proof accepted")
			}
		})
	}
	for _, header := range []map[string]any{{"jwk": privateProofJWK(t, key)}, {"jwk": map[string]any{"kty": "oct", "k": "c3ludGhldGlj"}}} {
		if _, err := parseProof([]string{rawProof(t, key, now, header, "")}); err == nil {
			t.Fatal("non-public JWK accepted")
		}
	}
}

func TestParseProofRejectsInvalidIdentifier(t *testing.T) {
	key := proofECKey(t)
	claims := baseClaims(proofNow())
	claims["jti"] = "short"
	if _, err := parseProof([]string{rawProof(t, key, proofNow(), nil, marshalProofValue(t, claims))}); err == nil {
		t.Fatal("short proof identifier accepted")
	}
}

func TestParseProofRejectsMalformedSignatureAndLimits(t *testing.T) {
	key := proofECKey(t)
	now := proofNow()
	proof, _ := validProof(t, key, now, "nonce")
	for _, test := range []struct {
		name  string
		value string
	}{
		{name: "structure", value: "not-a-jws"},
		{name: "signature", value: tamperedProof(t, proof)},
		{name: "size", value: strings.Repeat("a", maxProofBytes+1)},
	} {
		t.Run(test.name, func(t *testing.T) {
			if _, err := parseProof([]string{test.value}); err == nil {
				t.Fatal("malformed proof accepted")
			}
		})
	}
}

func tamperedProof(t *testing.T, proof string) string {
	t.Helper()
	parts := strings.Split(proof, ".")
	signature, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil || len(signature) == 0 {
		t.Fatal("invalid signed proof")
	}
	signature[0] ^= 1
	parts[2] = base64.RawURLEncoding.EncodeToString(signature)
	return strings.Join(parts, ".")
}

func TestPublicJWKRejectsIncompatibleAndUndersizedKeys(t *testing.T) {
	ec := proofECKey(t)
	rsa := proofRSAKey(t, 1024)
	for _, test := range []struct {
		algorithm string
		key       crypto.Signer
	}{
		{algorithm: "RS256", key: ec},
		{algorithm: "RS256", key: rsa},
	} {
		raw, err := json.Marshal(jose.JSONWebKey{Key: publicProofKey(test.key)})
		if err != nil {
			t.Fatal(err)
		}
		if _, _, err := publicJWK(raw, test.algorithm); err == nil {
			t.Fatal("incompatible key accepted")
		}
	}
}

func proofNow() time.Time { return time.Date(2026, time.November, 4, 1, 2, 3, 0, time.UTC) }

func proofVerifier(now time.Time, store Store) Verifier {
	return Verifier{Config: Config{Resource: proofResource, MaxAge: time.Minute, Skew: 5 * time.Second}, Store: store, Now: func() time.Time { return now }}
}

type proofStore struct {
	challenges, consumes int
	expires              time.Time
}

func (s *proofStore) Challenge(_ context.Context, _ string, expires time.Time) (string, error) {
	s.challenges++
	s.expires = expires
	return "nonce", nil
}

func (s *proofStore) Consume(_ context.Context, _, _, _ string, expires time.Time) (string, error) {
	s.consumes++
	s.expires = expires
	return "next-nonce", nil
}

func proofECKey(t *testing.T) *ecdsa.PrivateKey {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	return key
}

func proofRSAKey(t *testing.T, bits int) *rsa.PrivateKey {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, bits)
	if err != nil {
		t.Fatal(err)
	}
	return key
}

func validProof(t *testing.T, key crypto.Signer, now time.Time, nonce string) (string, string) {
	t.Helper()
	proof := proofWithKey(t, key, now)
	return proof, proofJKT(t, key)
}

func proofWithKey(t *testing.T, key crypto.Signer, now time.Time) string {
	t.Helper()
	return rawProof(t, key, now, nil, "")
}

func proofWithClaim(t *testing.T, key crypto.Signer, now time.Time, name, value string) string {
	t.Helper()
	return rawProof(t, key, now, nil, claimDocument(t, now, name, value))
}

func proofWithToken(t *testing.T, key crypto.Signer, now time.Time, token string) string {
	t.Helper()
	hash := sha256.Sum256([]byte(token))
	return rawProof(t, key, now, nil, claimDocument(t, now, "ath", base64.RawURLEncoding.EncodeToString(hash[:])))
}

func proofWithIssuedAt(t *testing.T, key crypto.Signer, issued time.Time) string {
	t.Helper()
	return rawProof(t, key, issued, nil, "")
}

func claimDocument(t *testing.T, now time.Time, name, value string) string {
	t.Helper()
	claims := baseClaims(now)
	claims[name] = value
	return marshalProofValue(t, claims)
}

func rawProof(t *testing.T, key crypto.Signer, now time.Time, additions map[string]any, body string) string {
	t.Helper()
	header := baseHeader(t, key)
	for name, value := range additions {
		header[name] = value
	}
	if body == "" {
		body = marshalProofValue(t, baseClaims(now))
	}
	return signProof(t, key, marshalProofValue(t, header), body)
}

func baseHeader(t *testing.T, key crypto.Signer) map[string]any {
	t.Helper()
	jwk := marshalProofValue(t, jose.JSONWebKey{Key: publicProofKey(key)})
	return map[string]any{"typ": "dpop+jwt", "alg": proofAlgorithm(key), "jwk": json.RawMessage(jwk)}
}

func privateProofJWK(t *testing.T, key crypto.Signer) map[string]any {
	t.Helper()
	var fields map[string]any
	if err := json.Unmarshal([]byte(marshalProofValue(t, jose.JSONWebKey{Key: publicProofKey(key)})), &fields); err != nil {
		t.Fatal(err)
	}
	fields["d"] = "private"
	return fields
}

func baseClaims(now time.Time) map[string]any {
	hash := sha256.Sum256([]byte("synthetic-token"))
	return map[string]any{"jti": "0123456789abcdef", "htm": "POST", "htu": proofResource, "iat": now.Unix(), "ath": base64.RawURLEncoding.EncodeToString(hash[:]), "nonce": "nonce"}
}

func marshalProofValue(t *testing.T, value any) string {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func signProof(t *testing.T, key crypto.Signer, header, body string) string {
	t.Helper()
	signed := base64.RawURLEncoding.EncodeToString([]byte(header)) + "." + base64.RawURLEncoding.EncodeToString([]byte(body))
	signature, err := proofSigningMethod(key).Sign(signed, key)
	if err != nil {
		t.Fatal(err)
	}
	return signed + "." + base64.RawURLEncoding.EncodeToString(signature)
}

func proofSigningMethod(key crypto.Signer) jwt.SigningMethod {
	if _, ok := key.(*rsa.PrivateKey); ok {
		return jwt.SigningMethodRS256
	}
	return jwt.SigningMethodES256
}

func proofAlgorithm(key crypto.Signer) string {
	if _, ok := key.(*rsa.PrivateKey); ok {
		return "RS256"
	}
	return "ES256"
}

func publicProofKey(key crypto.Signer) any {
	switch value := key.(type) {
	case *ecdsa.PrivateKey:
		return &value.PublicKey
	case *rsa.PrivateKey:
		return &value.PublicKey
	default:
		return nil
	}
}

func proofJKT(t *testing.T, key crypto.Signer) string {
	t.Helper()
	jwk := jose.JSONWebKey{Key: publicProofKey(key)}
	thumbprint, err := jwk.Thumbprint(crypto.SHA256)
	if err != nil {
		t.Fatal(err)
	}
	return base64.RawURLEncoding.EncodeToString(thumbprint)
}

func proofRequest(t *testing.T, target, proof string) *http.Request {
	t.Helper()
	request, err := http.NewRequest(http.MethodPost, target, nil)
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("DPoP", proof)
	return request
}
