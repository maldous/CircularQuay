package securitycontext

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"

	jose "github.com/go-jose/go-jose/v4"
	"github.com/maldous/CircularQuay/internal/fault"
)

func TestSignatureBoundaryRejectsMalformedEnrolledMaterialWithoutIdentity(t *testing.T) {
	input, private := fixture(t)
	raw := signed(t, claimsFor(input), private, "key-1", nil)
	var env envelope
	if err := json.Unmarshal(raw, &env); err != nil {
		t.Fatal(err)
	}
	payload, err := base64.RawURLEncoding.DecodeString(env.Payload)
	if err != nil {
		t.Fatal(err)
	}
	input.Trust.Keys[0].Key = struct{}{}
	seen := map[string]struct{}{}
	err = verifySignature(env.Signatures[0], env.Payload, payload, input.Trust, seen)
	if !fault.Is(err, fault.Trust) || len(seen) != 0 {
		t.Fatalf("malformed material escaped signature boundary: %v, identities %d", err, len(seen))
	}
}

func TestSingletonBoundaryRejectsInvalidEncodedSignature(t *testing.T) {
	input, _ := fixture(t)
	sig := signature{Protected: base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"ES256"}`)), Signature: "?"}
	payload, err := verifySingleton(base64.RawURLEncoding.EncodeToString([]byte("synthetic")), sig, input.Trust.Keys[0])
	if !fault.Is(err, fault.Trust) || payload != nil {
		t.Fatalf("invalid singleton signature returned payload: %v, %d bytes", err, len(payload))
	}
}

func TestSuccessorBoundaryRejectsMalformedAndExhaustedGenerations(t *testing.T) {
	cases := []struct {
		name   string
		change func(*ContextGeneration, *GenerationReference)
	}{
		{"noncanonical epoch", func(g *ContextGeneration, _ *GenerationReference) { g.RecoveryEpoch = "01" }},
		{"different epoch", func(g *ContextGeneration, _ *GenerationReference) { g.RecoveryEpoch = "1" }},
		{"different previous payload", func(g *ContextGeneration, _ *GenerationReference) {
			g.PreviousAcceptedPayloadSHA256 = hashText("other")
		}},
		{"empty sequence", func(g *ContextGeneration, _ *GenerationReference) { g.Sequence = "" }},
		{"zero sequence", func(g *ContextGeneration, _ *GenerationReference) { g.Sequence = "0" }},
		{"empty accepted sequence", func(_ *ContextGeneration, a *GenerationReference) { a.Sequence = "" }},
		{"malformed accepted sequence", func(_ *ContextGeneration, a *GenerationReference) { a.Sequence = "not-decimal" }},
		{"exhausted accepted sequence", func(_ *ContextGeneration, a *GenerationReference) { a.Sequence = "18446744073709551615" }},
	}
	for _, scenario := range cases {
		t.Run(scenario.name, func(t *testing.T) {
			hash := hashText("synthetic accepted payload")
			accepted := GenerationReference{RecoveryEpoch: "0", Sequence: "1", AcceptedPayloadSHA256: hash}
			generation := ContextGeneration{RecoveryEpoch: "0", Sequence: "2", PreviousAcceptedPayloadSHA256: hash}
			scenario.change(&generation, &accepted)
			if successorContext(generation, accepted) {
				t.Fatal("invalid generation accepted as the next context")
			}
		})
	}
}

func TestSuccessorBoundaryAcceptsLastRepresentableSequenceWithoutWrapping(t *testing.T) {
	hash := hashText("synthetic accepted payload")
	accepted := GenerationReference{RecoveryEpoch: "0", Sequence: "18446744073709551614", AcceptedPayloadSHA256: hash}
	generation := ContextGeneration{RecoveryEpoch: "0", Sequence: "18446744073709551615", PreviousAcceptedPayloadSHA256: hash}
	if !successorContext(generation, accepted) {
		t.Fatal("last representable successor was rejected")
	}
	accepted.Sequence = generation.Sequence
	generation.Sequence = "1"
	if successorContext(generation, accepted) {
		t.Fatal("exhausted sequence wrapped into a new authorization generation")
	}
}

func TestCanonicalKeyBoundaryRefusesUnsupportedAlgorithm(t *testing.T) {
	key := TrustedKey{Algorithm: jose.HS256, Key: []byte("synthetic unsupported material")}
	if encoded, ok := canonicalPublicKey(key); ok || encoded != nil {
		t.Fatal("unsupported public-key algorithm produced canonical material")
	}
	if identity, ok := keyMaterialIdentity(key); ok || identity != "" {
		t.Fatal("unsupported algorithm produced a trusted material identity")
	}
}

func TestDecimalBoundaryRejectsNoncanonicalAndOverflowingValues(t *testing.T) {
	for _, value := range []string{"", "01", "-1", "1.0", "1\n", "18446744073709551616", strings.Repeat("1", 21)} {
		if _, ok := decimal(value); ok {
			t.Fatalf("noncanonical decimal accepted: %q", value)
		}
	}
	if value, ok := decimal("18446744073709551615"); !ok || value != ^uint64(0) {
		t.Fatal("maximum canonical uint64 was not preserved")
	}
	if validSHA256(strings.Repeat("f", 63)) {
		t.Fatal("short digest accepted as a SHA-256 identity")
	}
}

func TestCanonicalPayloadBoundaryRejectsMalformedTrailingBitsAndDecodedOverflow(t *testing.T) {
	for _, scenario := range []struct {
		value string
		limit int
	}{{"?", 2}, {"Zh", 2}, {"Zm9v", 2}} {
		decoded, err := decodeCanonical(scenario.value, scenario.limit)
		if !fault.Is(err, fault.Invalid) || decoded != nil {
			t.Fatalf("invalid encoded payload accepted: %q, %v", scenario.value, err)
		}
	}
}
