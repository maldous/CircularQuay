package securitycontext

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"strconv"
	"strings"
	"testing"
	"time"

	jose "github.com/go-jose/go-jose/v4"
	"github.com/maldous/CircularQuay/internal/fault"
)

func TestVerifyRejectsBoundedAdversarialEnvelopes(t *testing.T) {
	input, private := fixture(t)
	valid := signed(t, claimsFor(input), private, "key-1", nil)
	cases := []struct {
		name string
		raw  []byte
		kind fault.Kind
	}{
		{"tampered_signature", tamperSignature(t, valid), fault.Trust},
		{"unknown_protected", signed(t, claimsFor(input), private, "key-1", map[string]any{"unknown": "x"}), fault.Invalid},
		{"critical_header", signed(t, claimsFor(input), private, "key-1", map[string]any{"crit": []string{"purpose"}}), fault.Invalid},
		{"unprotected_header", addUnprotected(t, valid), fault.Invalid},
		{"duplicate_root_member", duplicatePayloadMember(), fault.Invalid},
		{"oversized", append(valid, make([]byte, MaxEnvelopeBytes)...), fault.Invalid},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := Verify(tc.raw, input); !fault.Is(err, tc.kind) {
				t.Fatalf("Verify() error = %v, want %s", err, tc.kind)
			}
		})
	}
}

func TestVerifyRejectsGraphBindingMismatch(t *testing.T) {
	input, private := fixture(t)
	claims := claimsFor(input)
	claims.PolicyGraphSHA256 = hashText("different graph")
	if _, err := Verify(signed(t, claims, private, "key-1", nil), input); !fault.Is(err, fault.Trust) {
		t.Fatalf("graph mismatch error = %v", err)
	}
}

func TestVerifyRejectsThresholdAliasAndExpiry(t *testing.T) {
	input, private := fixture(t)
	input.Trust.RequiredSigners = 2
	second, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	input.Trust.Keys = append(input.Trust.Keys, TrustedKey{KID: "key-2", Algorithm: jose.ES256, Key: &second.PublicKey})
	one := signed(t, claimsFor(input), private, "key-1", nil)
	aliased := duplicateSignature(t, one)
	if _, err := Verify(aliased, input); !fault.Is(err, fault.Trust) {
		t.Fatalf("alias threshold error = %v", err)
	}
	input, private = fixture(t)
	expired := claimsFor(input)
	expired.ExpiresAt = input.Now.Add(-time.Second)
	if _, err := Verify(signed(t, expired, private, "key-1", nil), input); !fault.Is(err, fault.Expired) {
		t.Fatalf("expiry error = %v", err)
	}
}

func TestVerifyRejectsDuplicatePublicMaterialAndAcceptsDistinctThreshold(t *testing.T) {
	input, private := fixture(t)
	input.Trust.RequiredSigners = 2
	input.Trust.Keys = append(input.Trust.Keys, TrustedKey{KID: "same-key-new-kid", Algorithm: jose.ES256, Key: &private.PublicKey})
	if _, err := Verify(signed(t, claimsFor(input), private, "key-1", nil), input); !fault.Is(err, fault.Invalid) {
		t.Fatalf("duplicate public material error = %v", err)
	}
	input, private = fixture(t)
	input.Trust.RequiredSigners = 2
	second, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	input.Trust.Keys = append(input.Trust.Keys, TrustedKey{KID: "key-2", Algorithm: jose.ES256, Key: &second.PublicKey})
	first := signed(t, claimsFor(input), private, "key-1", nil)
	other := signed(t, claimsFor(input), second, "key-2", nil)
	if _, err = Verify(combineSignatures(t, first, other), input); err != nil {
		t.Fatalf("distinct 2-of-2 threshold = %v", err)
	}
}

func TestCanonicalECDSAKeyValidation(t *testing.T) {
	_, private := fixture(t)
	if _, ok := canonicalECDSAKey(&private.PublicKey, elliptic.P256()); !ok {
		t.Fatal("valid ES256 public key rejected")
	}
	wrongCurve, err := ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	cases := []crypto.PublicKey{
		&ecdsa.PublicKey{Curve: elliptic.P256()},
		private,
		(*ecdsa.PublicKey)(nil),
		&wrongCurve.PublicKey,
		struct{}{},
	}
	for _, value := range cases {
		if _, ok := canonicalECDSAKey(value, elliptic.P256()); ok {
			t.Fatal("unsupported ECDSA public key accepted")
		}
	}
	invalidPoint := private.PublicKey
	invalidPoint.Curve = elliptic.P384()
	if _, ok := canonicalECDSAKey(&invalidPoint, elliptic.P384()); ok {
		t.Fatal("point from a different curve accepted")
	}
}

func TestVerifyGenerationReplaySuccessorAndFence(t *testing.T) {
	input, private := fixture(t)
	current := signed(t, claimsFor(input), private, "key-1", nil)
	effective, err := Verify(current, input)
	if err != nil || !effective.Verified() {
		t.Fatalf("initial verify = %#v, %v", effective, err)
	}
	currentHash := payloadHashFromEnvelope(t, current)
	input.Snapshot.Context = AcceptedContext{Reference: GenerationReference{RecoveryEpoch: "0", Sequence: "1", AcceptedPayloadSHA256: currentHash}, PreviousPayload: input.Snapshot.Context.Reference.AcceptedPayloadSHA256}
	if _, err = Verify(current, input); err != nil {
		t.Fatalf("idempotent replay = %v", err)
	}
	next := claimsFor(input)
	next.ContextGeneration = ContextGeneration{RecoveryEpoch: "0", Sequence: "2", PreviousAcceptedPayloadSHA256: currentHash}
	if _, err = Verify(signed(t, next, private, "key-1", nil), input); err != nil {
		t.Fatalf("successor = %v", err)
	}
	next.AuthorityGeneration.Sequence = "2"
	if _, err = Verify(signed(t, next, private, "key-1", nil), input); !fault.Is(err, fault.Denied) {
		t.Fatalf("torn authority fence = %v", err)
	}
}

func TestVerifyAdmitsFirstNormalContextAtMaximumRecoveryEpoch(t *testing.T) {
	input, private := fixture(t)
	maximumEpoch := strconv.FormatUint(^uint64(0), 10)
	bootstrap := input.Snapshot.Context.Reference.AcceptedPayloadSHA256
	input.Snapshot.Context = AcceptedContext{Reference: GenerationReference{RecoveryEpoch: maximumEpoch, Sequence: "0", AcceptedPayloadSHA256: bootstrap}, PreviousPayload: bootstrap}
	claims := claimsFor(input)
	claims.ContextGeneration = ContextGeneration{RecoveryEpoch: maximumEpoch, Sequence: "1", PreviousAcceptedPayloadSHA256: bootstrap}
	if _, err := Verify(signed(t, claims, private, "key-1", nil), input); err != nil {
		t.Fatalf("first context in maximum recovery epoch = %v", err)
	}
}

func TestVerifiedEnvelopeIsUnadmitted(t *testing.T) {
	var empty VerifiedEnvelope
	if empty.Verified() || empty.Summary() != (Summary{}) {
		t.Fatal("zero VerifiedEnvelope must not claim verification or expose metadata")
	}
	input, private := fixture(t)
	claims := claimsFor(input)
	claims.Capabilities = []CapabilityEvidence{{ControlID: "control", ImplementationID: "implementation", Version: "1", Platform: "test", ExpiresAt: input.Now.Add(time.Minute)}}
	effective, err := Verify(signed(t, claims, private, "key-1", nil), input)
	if err != nil || !effective.Verified() {
		t.Fatalf("Verify() = %#v, %v", effective, err)
	}
}

func fixture(t *testing.T) (Input, *ecdsa.PrivateKey) {
	t.Helper()
	private, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	boot := hashText("bootstrap")
	authority := GenerationReference{RecoveryEpoch: "0", Sequence: "1", AcceptedPayloadSHA256: hashText("authority")}
	enrollment := GenerationReference{RecoveryEpoch: "0", Sequence: "1", AcceptedPayloadSHA256: hashText("enrollment")}
	binding := Binding{Tenant: "tenant-a", ContextID: "context-a", TrustDomain: "native-a", Audience: "cq://native-a"}
	return Input{Trust: Trust{Binding: binding, Issuer: "local-baseline", ProfileID: "default-local", ProfileRevision: "1", Keys: []TrustedKey{{KID: "key-1", Algorithm: jose.ES256, Key: &private.PublicKey}}, RequiredSigners: 1}, Snapshot: AggregateSnapshot{Context: AcceptedContext{Reference: GenerationReference{RecoveryEpoch: "0", Sequence: "0", AcceptedPayloadSHA256: boot}, PreviousPayload: boot}, Authority: authority, Enrollment: enrollment, TrustSetEpoch: "1", RevocationEpoch: "1", Fence: "1"}, GraphDigest: hashText("graph"), Now: time.Now().UTC()}, private
}

func claimsFor(input Input) Claims {
	return Claims{Version: 1, ContextID: input.Trust.Binding.ContextID, ProfileID: input.Trust.ProfileID, ProfileRevision: input.Trust.ProfileRevision, Tenant: input.Trust.Binding.Tenant, Issuer: input.Trust.Issuer, Audience: input.Trust.Binding.Audience, PolicyGraphSHA256: input.GraphDigest, PolicyRevision: "1", ContextGeneration: ContextGeneration{RecoveryEpoch: "0", Sequence: "1", PreviousAcceptedPayloadSHA256: input.Snapshot.Context.Reference.AcceptedPayloadSHA256}, AuthorityGeneration: input.Snapshot.Authority, EnrollmentGeneration: input.Snapshot.Enrollment, NotBefore: input.Now.Add(-time.Minute), ExpiresAt: input.Now.Add(time.Minute), RevocationFreshUntil: input.Now.Add(time.Minute), Requirements: Requirements{Custody: "local"}, Capabilities: []CapabilityEvidence{}}
}

func signed(t *testing.T, claims Claims, key *ecdsa.PrivateKey, kid string, extra map[string]any) []byte {
	t.Helper()
	payload, err := json.Marshal(claims)
	if err != nil {
		t.Fatal(err)
	}
	options := (&jose.SignerOptions{}).WithType(contextType).WithContentType(contextCTY).WithHeader(jose.HeaderKey("kid"), kid).WithHeader(jose.HeaderKey("purpose"), contextPurpose)
	for name, value := range extra {
		options.WithHeader(jose.HeaderKey(name), value)
	}
	signer, err := jose.NewSigner(jose.SigningKey{Algorithm: jose.ES256, Key: key}, options)
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

func tamperSignature(t *testing.T, raw []byte) []byte {
	t.Helper()
	var env envelope
	if err := json.Unmarshal(raw, &env); err != nil {
		t.Fatal(err)
	}
	env.Signatures[0].Signature = strings.Repeat("A", len(env.Signatures[0].Signature))
	changed, err := json.Marshal(env)
	if err != nil {
		t.Fatal(err)
	}
	return changed
}
func addUnprotected(t *testing.T, raw []byte) []byte {
	t.Helper()
	var value map[string]any
	if err := json.Unmarshal(raw, &value); err != nil {
		t.Fatal(err)
	}
	value["signatures"].([]any)[0].(map[string]any)["header"] = map[string]any{"kid": "key-1"}
	changed, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return changed
}
func duplicatePayloadMember() []byte {
	return []byte(`{"payload":"x","payload":"x","signatures":[]}`)
}
func duplicateSignature(t *testing.T, raw []byte) []byte {
	t.Helper()
	var env envelope
	if err := json.Unmarshal(raw, &env); err != nil {
		t.Fatal(err)
	}
	env.Signatures = append(env.Signatures, env.Signatures[0])
	changed, err := json.Marshal(env)
	if err != nil {
		t.Fatal(err)
	}
	return changed
}
func combineSignatures(t *testing.T, first, second []byte) []byte {
	t.Helper()
	var left, right envelope
	if err := json.Unmarshal(first, &left); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(second, &right); err != nil || left.Payload != right.Payload {
		t.Fatal("signed payload mismatch")
	}
	left.Signatures = append(left.Signatures, right.Signatures...)
	combined, err := json.Marshal(left)
	if err != nil {
		t.Fatal(err)
	}
	return combined
}
func payloadHashFromEnvelope(t *testing.T, raw []byte) string {
	t.Helper()
	var env envelope
	if err := json.Unmarshal(raw, &env); err != nil {
		t.Fatal(err)
	}
	payload, err := base64.RawURLEncoding.DecodeString(env.Payload)
	if err != nil {
		t.Fatal(err)
	}
	return payloadHash(payload)
}
func hashText(value string) string {
	digest := sha256.Sum256([]byte(value))
	return hex.EncodeToString(digest[:])
}
