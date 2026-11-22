package securitycontext

import (
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"testing"
	"time"

	jose "github.com/go-jose/go-jose/v4"
	"github.com/maldous/CircularQuay/internal/fault"
)

// These tests use existing synthetic fixtures. They exercise maintained JOSE
// algorithms and signed hostile-input boundaries only.
func TestVerifyMaintainedES384AndEd25519(t *testing.T) {
	input, _ := fixture(t)
	es384, err := ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	input.Trust.Keys = []TrustedKey{{KID: "p384", Algorithm: jose.ES384, Key: &es384.PublicKey}}
	if _, err = Verify(signWith(t, claimsFor(input), es384, jose.ES384, "p384"), input); err != nil {
		t.Fatalf("ES384 verification = %v", err)
	}
	input, _ = fixture(t)
	public, private, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	input.Trust.Keys = []TrustedKey{{KID: "ed25519", Algorithm: jose.EdDSA, Key: public}}
	if _, err = Verify(signWith(t, claimsFor(input), private, jose.EdDSA, "ed25519"), input); err != nil {
		t.Fatalf("Ed25519 verification = %v", err)
	}
}

func TestVerifyRejectsTypedClaimBoundaries(t *testing.T) {
	input, private := fixture(t)
	cases := []struct {
		name   string
		mutate func(*Claims)
		kind   fault.Kind
	}{
		{"sequence_zero", func(c *Claims) { c.ContextGeneration.Sequence = "0" }, fault.Trust},
		{"expired_capability", func(c *Claims) {
			c.Capabilities = []CapabilityEvidence{{ControlID: "c", ImplementationID: "i", Version: "1", Platform: "p", ExpiresAt: input.Now.Add(-time.Second)}}
		}, fault.Trust},
		{"expired_revocation", func(c *Claims) { c.RevocationFreshUntil = input.Now.Add(-time.Second) }, fault.Expired},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			claims := claimsFor(input)
			tc.mutate(&claims)
			if _, err := Verify(signWith(t, claims, private, jose.ES256, "key-1"), input); !fault.Is(err, tc.kind) {
				t.Fatalf("Verify() error = %v, want %s", err, tc.kind)
			}
		})
	}
}

func TestVerifyRejectsHostileSignedWireForms(t *testing.T) {
	input, private := fixture(t)
	valid := signWith(t, claimsFor(input), private, jose.ES256, "key-1")
	cases := []struct {
		name string
		raw  []byte
		kind fault.Kind
	}{
		{"noncanonical_payload_encoding", replacePayload(t, valid, "A="), fault.Invalid},
		{"duplicate_signed_claim", signRaw(t, []byte(`{"version":1,"version":1}`), private, jose.ES256, "key-1"), fault.Invalid},
		{"wrong_purpose", alterProtected(t, valid, "purpose", "other"), fault.Invalid},
		{"algorithm_confusion", alterProtected(t, valid, "alg", "ES384"), fault.Trust},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := Verify(tc.raw, input); !fault.Is(err, tc.kind) {
				t.Fatalf("Verify() error = %v, want %s", err, tc.kind)
			}
		})
	}
}

func TestVerifiedEnvelopeDoesNotExposeSensitiveBindings(t *testing.T) {
	input, private := fixture(t)
	verified, err := Verify(signWith(t, claimsFor(input), private, jose.ES256, "key-1"), input)
	if err != nil || !verified.Verified() {
		t.Fatalf("Verify() = %v", err)
	}
	summary := verified.Summary()
	if summary.ContextID == "" || summary.ProfileID == "" || summary.ProfileRevision == "" {
		t.Fatal("narrow routing summary missing")
	}
	if (VerifiedEnvelope{}).Verified() || (VerifiedEnvelope{}).Summary() != (Summary{}) {
		t.Fatal("zero envelope presented verification metadata")
	}
}

func signWith(t *testing.T, claims Claims, key any, algorithm jose.SignatureAlgorithm, kid string) []byte {
	t.Helper()
	payload, err := json.Marshal(claims)
	if err != nil {
		t.Fatal(err)
	}
	options := (&jose.SignerOptions{}).WithType(contextType).WithContentType(contextCTY).WithHeader(jose.HeaderKey("kid"), kid).WithHeader(jose.HeaderKey("purpose"), contextPurpose)
	signer, err := jose.NewSigner(jose.SigningKey{Algorithm: algorithm, Key: key}, options)
	if err != nil {
		t.Fatal(err)
	}
	jws, err := signer.Sign(payload)
	if err != nil {
		t.Fatal(err)
	}
	var flattened struct {
		Payload   string `json:"payload"`
		Protected string `json:"protected"`
		Signature string `json:"signature"`
	}
	if err = json.Unmarshal([]byte(jws.FullSerialize()), &flattened); err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(envelope{Payload: flattened.Payload, Signatures: []signature{{Protected: flattened.Protected, Signature: flattened.Signature}}})
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func signRaw(t *testing.T, payload []byte, key any, algorithm jose.SignatureAlgorithm, kid string) []byte {
	t.Helper()
	options := (&jose.SignerOptions{}).WithType(contextType).WithContentType(contextCTY).WithHeader(jose.HeaderKey("kid"), kid).WithHeader(jose.HeaderKey("purpose"), contextPurpose)
	signer, err := jose.NewSigner(jose.SigningKey{Algorithm: algorithm, Key: key}, options)
	if err != nil {
		t.Fatal(err)
	}
	jws, err := signer.Sign(payload)
	if err != nil {
		t.Fatal(err)
	}
	var flattened struct {
		Payload   string `json:"payload"`
		Protected string `json:"protected"`
		Signature string `json:"signature"`
	}
	if err = json.Unmarshal([]byte(jws.FullSerialize()), &flattened); err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(envelope{Payload: flattened.Payload, Signatures: []signature{{Protected: flattened.Protected, Signature: flattened.Signature}}})
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func replacePayload(t *testing.T, raw []byte, payload string) []byte {
	t.Helper()
	var env envelope
	if err := json.Unmarshal(raw, &env); err != nil {
		t.Fatal(err)
	}
	env.Payload = payload
	changed, err := json.Marshal(env)
	if err != nil {
		t.Fatal(err)
	}
	return changed
}

func alterProtected(t *testing.T, raw []byte, name, value string) []byte {
	t.Helper()
	var env envelope
	if err := json.Unmarshal(raw, &env); err != nil {
		t.Fatal(err)
	}
	decoded, err := base64.RawURLEncoding.DecodeString(env.Signatures[0].Protected)
	if err != nil {
		t.Fatal(err)
	}
	var header map[string]any
	if err = json.Unmarshal(decoded, &header); err != nil {
		t.Fatal(err)
	}
	header[name] = value
	encoded, err := json.Marshal(header)
	if err != nil {
		t.Fatal(err)
	}
	env.Signatures[0].Protected = base64.RawURLEncoding.EncodeToString(encoded)
	changed, err := json.Marshal(env)
	if err != nil {
		t.Fatal(err)
	}
	return changed
}
