package securitycontext

import (
	"bytes"
	"crypto/ed25519"
	"strconv"
	"testing"

	jose "github.com/go-jose/go-jose/v4"
	"github.com/maldous/CircularQuay/internal/fault"
)

func TestVerifyRefusesInvalidEnrollmentMetadataBeforeSignatureVerification(t *testing.T) {
	cases := []struct {
		name   string
		change func(*Trust)
	}{
		{"tenant", func(trust *Trust) { trust.Binding.Tenant = "" }},
		{"context", func(trust *Trust) { trust.Binding.ContextID = "" }},
		{"trust domain", func(trust *Trust) { trust.Binding.TrustDomain = "" }},
		{"audience", func(trust *Trust) { trust.Binding.Audience = "" }},
		{"issuer", func(trust *Trust) { trust.Issuer = "" }},
		{"profile", func(trust *Trust) { trust.ProfileID = "" }},
		{"profile revision", func(trust *Trust) { trust.ProfileRevision = "" }},
	}
	for _, scenario := range cases {
		t.Run(scenario.name, func(t *testing.T) {
			input, private := fixture(t)
			raw := signed(t, claimsFor(input), private, "key-1", nil)
			assertValidEnrollmentEnvelope(t, raw, input)
			scenario.change(&input.Trust)
			assertInvalidEnrollmentEnvelope(t, raw, input)
		})
	}
}

func TestVerifyRefusesInvalidEnrollmentThresholdAndKeyCount(t *testing.T) {
	for _, required := range []int{0, maxSignatures + 1} {
		t.Run("threshold "+strconv.Itoa(required), func(t *testing.T) {
			input, private := fixture(t)
			raw := signed(t, claimsFor(input), private, "key-1", nil)
			assertValidEnrollmentEnvelope(t, raw, input)
			input.Trust.RequiredSigners = required
			assertInvalidEnrollmentEnvelope(t, raw, input)
		})
	}
	input, private := fixture(t)
	raw := signed(t, claimsFor(input), private, "key-1", nil)
	input.Trust.Keys = syntheticEnrollmentKeys(input.Trust.Keys[0], maxSignatures)
	assertValidEnrollmentEnvelope(t, raw, input)
	input.Trust.Keys = syntheticEnrollmentKeys(input.Trust.Keys[0], maxSignatures+1)
	assertInvalidEnrollmentEnvelope(t, raw, input)
	input.Trust.Keys = nil
	assertInvalidEnrollmentEnvelope(t, raw, input)
}

func syntheticEnrollmentKeys(first TrustedKey, count int) []TrustedKey {
	keys := []TrustedKey{first}
	for index := 1; index < count; index++ {
		// These fixed seeds are fabricated test material, never enrollment secrets.
		seed := bytes.Repeat([]byte{byte(index)}, ed25519.SeedSize)
		public := ed25519.NewKeyFromSeed(seed).Public().(ed25519.PublicKey)
		keys = append(keys, TrustedKey{KID: "synthetic-" + strconv.Itoa(index), Algorithm: jose.EdDSA, Key: public})
	}
	return keys
}

func assertValidEnrollmentEnvelope(t *testing.T, raw []byte, input Input) {
	t.Helper()
	verified, err := Verify(raw, input)
	if err != nil || !verified.Verified() {
		t.Fatalf("otherwise-valid enrollment did not verify: %v", err)
	}
}

func assertInvalidEnrollmentEnvelope(t *testing.T, raw []byte, input Input) {
	t.Helper()
	verified, err := Verify(raw, input)
	if !fault.Is(err, fault.Invalid) || verified.Verified() {
		t.Fatalf("invalid enrollment escaped input refusal: %v, verified %t", err, verified.Verified())
	}
}
