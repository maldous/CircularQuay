package keyservice

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/maldous/CircularQuay/internal/fault"
)

func TestSoftwareKMSRoundTripAndResolverOwnership(t *testing.T) {
	resolver := &softwareResolver{key: softwareTestKEK()}
	adapter := softwareAdapter(t, resolver)
	association, binding := softwareFixture()
	caller := softwareTestDEK()
	wrapped, err := adapter.Wrap(context.Background(), WrapRequest{Association: association, Context: binding, Plaintext: caller})
	if err != nil || string(caller) != string(softwareTestDEK()) || !resolver.allCleared() {
		t.Fatalf("wrap error=%v caller=%q keys_cleared=%t", err, caller, resolver.allCleared())
	}
	plain, err := adapter.Unwrap(context.Background(), UnwrapRequest{Association: association, Context: binding, Ciphertext: wrapped.Ciphertext})
	if err != nil || string(plain.Plaintext) != string(softwareTestDEK()) || !resolver.allCleared() {
		t.Fatalf("unwrap error=%v plaintext=%q keys_cleared=%t", err, plain.Plaintext, resolver.allCleared())
	}
}

func TestSoftwareKMSRejectsCrossBindingAndTamper(t *testing.T) {
	adapter := softwareAdapter(t, &softwareResolver{key: softwareTestKEK()})
	association, binding := softwareFixture()
	wrapped, err := adapter.Wrap(context.Background(), WrapRequest{Association: association, Context: binding, Plaintext: softwareTestDEK()})
	if err != nil {
		t.Fatal(err)
	}
	for _, request := range softwareRejectedRequests(t, association, binding, wrapped.Ciphertext) {
		if result, err := adapter.Unwrap(context.Background(), request); err == nil || result.Plaintext != nil {
			t.Fatalf("%s accepted result=%q error=%v", request.Context.ArtifactID, result.Plaintext, err)
		}
	}
}

func TestSoftwareKMSCancellationAndMalformedKeyMaterialReturnNoOutput(t *testing.T) {
	resolver := &softwareResolver{key: softwareTestKEK(), cancel: true}
	adapter := softwareAdapter(t, resolver)
	association, binding := softwareFixture()
	ctx, cancel := context.WithCancel(context.Background())
	resolver.contextCancel = cancel
	result, err := adapter.Wrap(ctx, WrapRequest{Association: association, Context: binding, Plaintext: softwareTestDEK()})
	if err == nil || result.Ciphertext != nil || !resolver.allCleared() {
		t.Fatalf("cancelled wrap result=%q error=%v keys_cleared=%t", result.Ciphertext, err, resolver.allCleared())
	}
	resolver.cancel = false
	resolver.mismatch = true
	result, err = adapter.Wrap(context.Background(), WrapRequest{Association: association, Context: binding, Plaintext: softwareTestDEK()})
	if err == nil || result.Ciphertext != nil || !resolver.allCleared() {
		t.Fatalf("mismatched resolver result=%q error=%v keys_cleared=%t", result.Ciphertext, err, resolver.allCleared())
	}
}

func TestSoftwareKMSRequiresExactDataKeyLengthBeforeResolver(t *testing.T) {
	resolver := &softwareResolver{key: softwareTestKEK()}
	adapter := softwareAdapter(t, resolver)
	association, binding := softwareFixture()
	for _, plaintext := range [][]byte{make([]byte, softwareKeyBytes-1), make([]byte, softwareKeyBytes+1)} {
		result, err := adapter.Wrap(context.Background(), WrapRequest{Association: association, Context: binding, Plaintext: plaintext})
		if err == nil || result.Ciphertext != nil {
			t.Fatalf("invalid size=%d result=%q error=%v", len(plaintext), result.Ciphertext, err)
		}
	}
	if len(resolver.issued) != 0 {
		t.Fatalf("resolver called for invalid data key: %d", len(resolver.issued))
	}
}

func TestSoftwareKMSRejectsStrictEnvelopeViolationsBeforeResolver(t *testing.T) {
	resolver := &softwareResolver{key: softwareTestKEK()}
	adapter := softwareAdapter(t, resolver)
	association, binding := softwareFixture()
	wrapped, err := adapter.Wrap(context.Background(), WrapRequest{Association: association, Context: binding, Plaintext: softwareTestDEK()})
	if err != nil {
		t.Fatal(err)
	}
	for _, ciphertext := range [][]byte{
		softwareTamper(t, tamperCiphertext, wrapped.Ciphertext),
		softwareDuplicateAAD(t, wrapped.Ciphertext),
		softwareUnknownField(t, wrapped.Ciphertext),
		append(append([]byte(nil), wrapped.Ciphertext...), 0xff),
		make([]byte, softwareCipherMax+1),
	} {
		result, unwrapErr := adapter.Unwrap(context.Background(), UnwrapRequest{Association: association, Context: binding, Ciphertext: ciphertext})
		if unwrapErr == nil || result.Plaintext != nil {
			t.Fatalf("strict envelope accepted result=%q error=%v", result.Plaintext, unwrapErr)
		}
	}
	if len(resolver.issued) != 1 {
		t.Fatalf("resolver called for invalid envelope: %d", len(resolver.issued))
	}
}

func TestSoftwareKMSRejectsPurposeMismatchAndOversizedHeaderBeforeResolver(t *testing.T) {
	resolver := &softwareResolver{key: softwareTestKEK()}
	adapter := softwareAdapter(t, resolver)
	association, binding := softwareFixture()
	wrongPurpose := binding
	wrongPurpose.Purpose = "other-purpose"
	oversized := association
	oversized.KeyID = strings.Repeat("k", 1025)
	for _, request := range []WrapRequest{{Association: association, Context: wrongPurpose, Plaintext: softwareTestDEK()}, {Association: oversized, Context: binding, Plaintext: softwareTestDEK()}} {
		result, err := adapter.Wrap(context.Background(), request)
		if err == nil || result.Ciphertext != nil {
			t.Fatalf("invalid binding result=%q error=%v", result.Ciphertext, err)
		}
	}
	if len(resolver.issued) != 0 {
		t.Fatalf("resolver called for invalid binding: %d", len(resolver.issued))
	}
}

func TestSoftwareKMSClearsResolverBufferOnResolverError(t *testing.T) {
	resolver := &softwareResolver{key: softwareTestKEK(), err: errors.New("resolver diagnostic")}
	adapter := softwareAdapter(t, resolver)
	association, binding := softwareFixture()
	result, err := adapter.Wrap(context.Background(), WrapRequest{Association: association, Context: binding, Plaintext: softwareTestDEK()})
	if err == nil || result.Ciphertext != nil || !resolver.allCleared() {
		t.Fatalf("resolver result=%q error=%v keys_cleared=%t", result.Ciphertext, err, resolver.allCleared())
	}
}

func TestSoftwareKMSRejectsAuthenticatedTamperingAndWrongKEK(t *testing.T) {
	resolver := &softwareResolver{key: softwareTestKEK()}
	adapter := softwareAdapter(t, resolver)
	association, binding := softwareFixture()
	wrapped, err := adapter.Wrap(context.Background(), WrapRequest{Association: association, Context: binding, Plaintext: softwareTestDEK()})
	if err != nil {
		t.Fatal(err)
	}
	for _, ciphertext := range [][]byte{softwareFlipJWEPart(t, "ciphertext", wrapped.Ciphertext), softwareFlipJWEPart(t, "tag", wrapped.Ciphertext)} {
		result, unwrapErr := adapter.Unwrap(context.Background(), UnwrapRequest{Association: association, Context: binding, Ciphertext: ciphertext})
		if !fault.Is(unwrapErr, fault.Trust) || result.Plaintext != nil {
			t.Fatalf("authenticated tamper result=%q error=%v", result.Plaintext, unwrapErr)
		}
	}
	if len(resolver.issued) != 3 {
		t.Fatalf("authenticated tamper did not reach JWE decrypt: resolver=%d", len(resolver.issued))
	}
	wrongResolver := &softwareResolver{key: []byte("11111111111111111111111111111111")}
	wrongAdapter := softwareAdapter(t, wrongResolver)
	result, err := wrongAdapter.Unwrap(context.Background(), UnwrapRequest{Association: association, Context: binding, Ciphertext: wrapped.Ciphertext})
	if !fault.Is(err, fault.Trust) || result.Plaintext != nil || !wrongResolver.allCleared() {
		t.Fatalf("wrong kek result=%q error=%v keys_cleared=%t", result.Plaintext, err, wrongResolver.allCleared())
	}
}

func softwareRejectedRequests(t *testing.T, association KMSAssociation, binding AssociationContext, ciphertext []byte) []UnwrapRequest {
	t.Helper()
	wrongTenant := binding
	wrongTenant.Tenant = "other-tenant"
	wrongProfile := binding
	wrongProfile.Profile = "other-profile"
	wrongArtifact := binding
	wrongArtifact.ArtifactID = "other-artifact"
	wrongVersion := association
	wrongVersion.KeyVersion = "2"
	wrongAttestation := association
	wrongAttestation.ConfigurationAttestation = "other-attestation"
	return []UnwrapRequest{
		{Association: association, Context: wrongTenant, Ciphertext: ciphertext},
		{Association: association, Context: wrongProfile, Ciphertext: ciphertext},
		{Association: association, Context: wrongArtifact, Ciphertext: ciphertext},
		{Association: wrongVersion, Context: binding, Ciphertext: ciphertext},
		{Association: wrongAttestation, Context: binding, Ciphertext: ciphertext},
		{Association: association, Context: binding, Ciphertext: softwareTamper(t, tamperAAD, ciphertext)},
		{Association: association, Context: binding, Ciphertext: softwareTamper(t, tamperAlgorithm, ciphertext)},
		{Association: association, Context: binding, Ciphertext: softwareTamper(t, tamperKID, ciphertext)},
		{Association: association, Context: binding, Ciphertext: softwareTamper(t, tamperCiphertext, ciphertext)},
	}
}

type softwareTamperKind string

const (
	tamperAAD        softwareTamperKind = "aad"
	tamperAlgorithm  softwareTamperKind = "alg"
	tamperKID        softwareTamperKind = "kid"
	tamperCiphertext softwareTamperKind = "ciphertext"
)

func softwareTamper(t *testing.T, kind softwareTamperKind, ciphertext []byte) []byte {
	t.Helper()
	var envelope map[string]json.RawMessage
	if err := json.Unmarshal(ciphertext, &envelope); err != nil {
		t.Fatal(err)
	}
	if kind == tamperCiphertext {
		envelope["ciphertext"] = json.RawMessage(`"AA"`)
		return softwareMarshal(t, envelope)
	}
	if kind == tamperAAD {
		envelope["aad"] = json.RawMessage(`"YQ"`)
		return softwareMarshal(t, envelope)
	}
	protected := jsonString(t, envelope["protected"])
	header, err := base64.RawURLEncoding.Strict().DecodeString(protected)
	if err != nil {
		t.Fatal(err)
	}
	var fields map[string]string
	if err := json.Unmarshal(header, &fields); err != nil {
		t.Fatal(err)
	}
	if kind == tamperAlgorithm {
		fields["alg"] = "dir"
	} else {
		fields["kid"] = "other-key"
	}
	updated, err := json.Marshal(fields)
	if err != nil {
		t.Fatal(err)
	}
	envelope["protected"] = json.RawMessage(`"` + base64.RawURLEncoding.EncodeToString(updated) + `"`)
	return softwareMarshal(t, envelope)
}

func softwareDuplicateAAD(t *testing.T, ciphertext []byte) []byte {
	t.Helper()
	var envelope map[string]json.RawMessage
	if err := json.Unmarshal(ciphertext, &envelope); err != nil {
		t.Fatal(err)
	}
	return append(append([]byte(nil), ciphertext[:len(ciphertext)-1]...), append([]byte(`,"aad":`), append(envelope["aad"], '}')...)...)
}

func softwareUnknownField(t *testing.T, ciphertext []byte) []byte {
	t.Helper()
	var envelope map[string]json.RawMessage
	if err := json.Unmarshal(ciphertext, &envelope); err != nil {
		t.Fatal(err)
	}
	envelope["unexpected"] = json.RawMessage(`true`)
	return softwareMarshal(t, envelope)
}

func softwareFlipJWEPart(t *testing.T, name string, ciphertext []byte) []byte {
	t.Helper()
	var envelope map[string]json.RawMessage
	if err := json.Unmarshal(ciphertext, &envelope); err != nil {
		t.Fatal(err)
	}
	value := jsonString(t, envelope[name])
	decoded, err := base64.RawURLEncoding.Strict().DecodeString(value)
	if err != nil || len(decoded) == 0 {
		t.Fatalf("decode %s error=%v", name, err)
	}
	decoded[0] ^= 1
	envelope[name] = softwareRawJSONString(t, base64.RawURLEncoding.EncodeToString(decoded))
	return softwareMarshal(t, envelope)
}

func softwareMarshal(t *testing.T, value any) []byte {
	t.Helper()
	result, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func softwareRawJSONString(t *testing.T, value string) json.RawMessage {
	t.Helper()
	encoded, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return encoded
}

func jsonString(t *testing.T, value json.RawMessage) string {
	t.Helper()
	var result string
	if err := json.Unmarshal(value, &result); err != nil {
		t.Fatal(err)
	}
	return result
}

func softwareAdapter(t *testing.T, resolver SoftwareKEKResolver) *SoftwareKMS {
	t.Helper()
	adapter, err := NewSoftwareKMS(resolver)
	if err != nil {
		t.Fatal(err)
	}
	return adapter
}

func softwareFixture() (KMSAssociation, AssociationContext) {
	association := KMSAssociation{Provider: softwareProvider, KeyID: "software-key", KeyVersion: "1", Algorithm: softwareAlgorithm, Purpose: "workload-data-key", ConfigurationAttestation: "software-attestation"}
	binding := AssociationContext{Tenant: "tenant", Profile: "profile", Application: "application", Principal: "principal", Generation: "generation-1", Purpose: association.Purpose, ArtifactID: "workload", ArtifactVersion: "revision-1"}
	return association, binding
}

func softwareTestKEK() []byte {
	return []byte("0123456789abcdef0123456789abcdef")
}

func softwareTestDEK() []byte {
	return []byte("fedcba9876543210fedcba9876543210")
}

type softwareResolver struct {
	key           []byte
	issued        [][]byte
	cancel        bool
	mismatch      bool
	err           error
	contextCancel context.CancelFunc
}

func (r *softwareResolver) ResolveSoftwareKEK(_ context.Context, association KMSAssociation) (SoftwareKEK, error) {
	issued := append([]byte(nil), r.key...)
	r.issued = append(r.issued, issued)
	if r.cancel && r.contextCancel != nil {
		r.contextCancel()
	}
	keyID := association.KeyID
	if r.mismatch {
		keyID = "other-key"
	}
	return SoftwareKEK{KeyID: keyID, KeyVersion: association.KeyVersion, Key: issued}, r.err
}

func (r *softwareResolver) allCleared() bool {
	for _, issued := range r.issued {
		if !zeroed(issued) {
			return false
		}
	}
	return len(r.issued) > 0
}
