package securitycontext

import (
	"crypto/ed25519"
	"encoding/json"
	"strconv"
	"strings"
	"testing"

	jose "github.com/go-jose/go-jose/v4"
	"github.com/maldous/CircularQuay/internal/fault"
)

func TestVerifyRejectsUnenrolledSignerWithoutRoutingMetadata(t *testing.T) {
	input, private := fixture(t)
	raw := signed(t, claimsFor(input), private, "synthetic-private-key-id", nil)
	assertBoundaryRefusal(t, raw, input, fault.Trust)
}

func TestVerifyRequiresEnoughDistinctEnrolledSignatures(t *testing.T) {
	input, private := fixture(t)
	second, _ := fixture(t)
	input.Trust.Keys = append(input.Trust.Keys, TrustedKey{KID: "key-2", Algorithm: second.Trust.Keys[0].Algorithm, Key: second.Trust.Keys[0].Key})
	input.Trust.RequiredSigners = 2
	assertBoundaryRefusal(t, signed(t, claimsFor(input), private, "key-1", nil), input, fault.Trust)
}

func TestVerifyRejectsNoncanonicalSignatureFields(t *testing.T) {
	input, private := fixture(t)
	valid := signed(t, claimsFor(input), private, "key-1", nil)
	for _, field := range []string{"protected", "signature"} {
		t.Run(field, func(t *testing.T) {
			var env envelope
			if err := json.Unmarshal(valid, &env); err != nil {
				t.Fatal(err)
			}
			if field == "protected" {
				env.Signatures[0].Protected = "="
			} else {
				env.Signatures[0].Signature = "="
			}
			raw, err := json.Marshal(env)
			if err != nil {
				t.Fatal(err)
			}
			assertBoundaryRefusal(t, raw, input, fault.Invalid)
		})
	}
}

func TestVerifyRejectsInvalidEnrolledKeyMetadata(t *testing.T) {
	for _, scenario := range []string{"duplicate key id", "unsupported algorithm", "malformed Ed25519"} {
		t.Run(scenario, func(t *testing.T) {
			input, private := fixture(t)
			switch scenario {
			case "duplicate key id":
				second, _ := fixture(t)
				input.Trust.Keys = append(input.Trust.Keys, second.Trust.Keys[0])
			case "unsupported algorithm":
				input.Trust.Keys[0].Algorithm = jose.HS256
			case "malformed Ed25519":
				input.Trust.Keys[0].Algorithm = jose.EdDSA
				input.Trust.Keys[0].Key = ed25519.PublicKey{1}
			}
			assertBoundaryRefusal(t, signed(t, claimsFor(input), private, "key-1", nil), input, fault.Invalid)
		})
	}
}

func TestVerifyRejectsInvalidSignedCapabilitySets(t *testing.T) {
	for _, scenario := range []string{"absent", "over limit", "malformed", "duplicate control"} {
		t.Run(scenario, func(t *testing.T) {
			input, private := fixture(t)
			claims := claimsFor(input)
			capability := CapabilityEvidence{ControlID: "control", ImplementationID: "implementation", Version: "1", Platform: "synthetic", ExpiresAt: claims.ExpiresAt}
			switch scenario {
			case "absent":
				claims.Capabilities = nil
			case "over limit":
				claims.Capabilities = boundaryCapabilities(33, capability)
			case "malformed":
				claims.Capabilities = []CapabilityEvidence{{}}
			case "duplicate control":
				claims.Capabilities = []CapabilityEvidence{capability, capability}
			}
			assertBoundaryRefusal(t, signed(t, claims, private, "key-1", nil), input, fault.Trust)
		})
	}
}

func TestVerifyAcceptsCapabilityCountLimit(t *testing.T) {
	input, private := fixture(t)
	claims := claimsFor(input)
	capability := CapabilityEvidence{ImplementationID: "implementation", Version: "1", Platform: "synthetic", ExpiresAt: claims.ExpiresAt}
	claims.Capabilities = boundaryCapabilities(32, capability)
	verified, err := Verify(signed(t, claims, private, "key-1", nil), input)
	if err != nil || !verified.Verified() {
		t.Fatalf("valid capability set at the count limit was rejected: %v", err)
	}
}

func boundaryCapabilities(count int, exemplar CapabilityEvidence) []CapabilityEvidence {
	capabilities := make([]CapabilityEvidence, count)
	for index := range capabilities {
		capabilities[index] = exemplar
		capabilities[index].ControlID = "control-" + strconv.Itoa(index)
	}
	return capabilities
}

func TestVerifyRejectsSkippedSuccessorSequence(t *testing.T) {
	input, private := fixture(t)
	claims := claimsFor(input)
	claims.ContextGeneration.Sequence = "2"
	assertBoundaryRefusal(t, signed(t, claims, private, "key-1", nil), input, fault.Denied)
}

func assertBoundaryRefusal(t *testing.T, raw []byte, input Input, kind fault.Kind) {
	t.Helper()
	verified, err := Verify(raw, input)
	if !fault.Is(err, kind) {
		t.Fatalf("context refusal = %v, want %s", err, kind)
	}
	if verified.Verified() || verified.Summary() != (Summary{}) {
		t.Fatal("rejected context exposed verified routing metadata")
	}
	if strings.Contains(err.Error(), "synthetic-private") {
		t.Fatal("untrusted context detail escaped the typed refusal")
	}
}
