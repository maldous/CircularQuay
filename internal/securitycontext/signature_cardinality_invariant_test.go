package securitycontext

import (
	"encoding/base64"
	"encoding/json"
	"reflect"
	"testing"

	"github.com/maldous/CircularQuay/internal/fault"
)

func TestSignatureCardinalityInvariantOneSuccessfulNewIdentityPerSignature(t *testing.T) {
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
	seen := map[string]struct{}{}
	if err := verifySignature(env.Signatures[0], env.Payload, payload, input.Trust, seen); err != nil || len(seen) != 1 {
		t.Fatal("successful signature did not add exactly one material identity")
	}
	if err := verifySignature(env.Signatures[0], env.Payload, payload, input.Trust, seen); !fault.Is(err, fault.Trust) || len(seen) != 1 {
		t.Fatal("duplicate material increased the signer threshold")
	}
	input.Trust.RequiredSigners = 2
	if err := verifySignatures(env, payload, input.Trust); !fault.Is(err, fault.Trust) {
		t.Fatal("initial signature-count threshold was weakened")
	}
}

func TestSingletonEncodingInvariantHasOnlyClosedStringFields(t *testing.T) {
	env := reflect.TypeOf(envelope{})
	sig := reflect.TypeOf(signature{})
	if env.NumField() != 2 || env.Field(0).Type.Kind() != reflect.String || env.Field(1).Type != reflect.SliceOf(sig) {
		t.Fatal("singleton envelope no longer has its reviewed closed shape")
	}
	if sig.NumField() != 2 || sig.Field(0).Type.Kind() != reflect.String || sig.Field(1).Type.Kind() != reflect.String {
		t.Fatal("singleton signature no longer has its reviewed closed shape")
	}
	assertSingletonNoCustomEncoding(t, env)
	assertSingletonNoCustomEncoding(t, sig)
}

func assertSingletonNoCustomEncoding(t *testing.T, value reflect.Type) {
	t.Helper()
	if value.NumMethod() != 0 || reflect.PointerTo(value).NumMethod() != 0 {
		t.Fatal("custom methods require revisiting the singleton encoding proof")
	}
}
