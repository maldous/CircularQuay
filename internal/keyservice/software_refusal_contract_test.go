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

// These contracts use only the existing fabricated key fixtures and an
// in-process resolver. They neither connect to a KMS nor enroll a profile.
func TestSoftwareRefusalContractRequiresExplicitResolver(t *testing.T) {
	adapter, err := NewSoftwareKMS(nil)
	assertSoftwareRefusal(t, err, fault.Invalid, "software_kek_resolver")
	if adapter != nil {
		t.Fatal("missing resolver produced a usable adapter")
	}
}

func TestSoftwareRefusalContractInvalidUnwrapNeverResolves(t *testing.T) {
	for _, entry := range []struct {
		name       string
		mutate     func(*UnwrapRequest)
		nilContext bool
		kind       fault.Kind
		operation  string
	}{
		{name: "nil_context", nilContext: true, kind: fault.Denied, operation: "software_cancelled"},
		{name: "provider", mutate: func(r *UnwrapRequest) { r.Association.Provider = "unselected" }, kind: fault.Invalid, operation: "software_binding"},
		{name: "purpose", mutate: func(r *UnwrapRequest) { r.Context.Purpose = "other-purpose" }, kind: fault.Invalid, operation: "software_binding"},
		{name: "context_identifier", mutate: func(r *UnwrapRequest) { r.Context.Generation = "invalid\nidentifier" }, kind: fault.Invalid, operation: "software_binding"},
		{name: "empty_ciphertext", mutate: func(r *UnwrapRequest) { r.Ciphertext = nil }, kind: fault.Invalid, operation: "software_ciphertext"},
	} {
		t.Run(entry.name, func(t *testing.T) {
			association, binding := softwareFixture()
			request := UnwrapRequest{Association: association, Context: binding, Ciphertext: []byte("not-reached")}
			if entry.mutate != nil {
				entry.mutate(&request)
			}
			ctx := context.Background()
			if entry.nilContext {
				ctx = nil
			}
			resolver := &softwareResolver{key: softwareTestKEK()}
			result, err := softwareAdapter(t, resolver).Unwrap(ctx, request)
			assertSoftwareRefusal(t, err, entry.kind, entry.operation)
			if result.Plaintext != nil || result.Receipt != (KMSReceipt{}) || len(resolver.issued) != 0 {
				t.Fatal("invalid unwrap resolved material or returned a partial result")
			}
		})
	}
}

func TestSoftwareRefusalContractClosedEnvelopeBeforeResolver(t *testing.T) {
	ciphertext := softwareRefusalCiphertext(t)
	for _, entry := range []struct {
		name   string
		mutate func(*testing.T, map[string]json.RawMessage)
		kind   fault.Kind
	}{
		{"same_count_wrong_member", func(_ *testing.T, envelope map[string]json.RawMessage) {
			delete(envelope, "aad")
			envelope["unselected"] = json.RawMessage(`"YQ"`)
		}, fault.Invalid},
		{"non_string_part", func(_ *testing.T, envelope map[string]json.RawMessage) {
			envelope["encrypted_key"] = json.RawMessage(`42`)
		}, fault.Invalid},
		{"protected_not_object", func(t *testing.T, envelope map[string]json.RawMessage) {
			envelope["protected"] = softwareRawJSONString(t, base64.RawURLEncoding.EncodeToString([]byte(`[]`)))
		}, fault.Trust},
		{"protected_wrong_member", func(t *testing.T, envelope map[string]json.RawMessage) {
			softwareRefusalHeader(t, envelope, func(header map[string]json.RawMessage) {
				delete(header, "kid")
				header["unselected"] = json.RawMessage(`"synthetic"`)
			})
		}, fault.Trust},
		{"protected_non_string", func(t *testing.T, envelope map[string]json.RawMessage) {
			softwareRefusalHeader(t, envelope, func(header map[string]json.RawMessage) {
				header[softwareHeaderKV] = json.RawMessage(`42`)
			})
		}, fault.Trust},
		{"protected_control", func(t *testing.T, envelope map[string]json.RawMessage) {
			softwareRefusalHeader(t, envelope, func(header map[string]json.RawMessage) {
				header[softwareHeaderCA] = json.RawMessage(`"invalid\nattestation"`)
			})
		}, fault.Trust},
	} {
		t.Run(entry.name, func(t *testing.T) {
			var envelope map[string]json.RawMessage
			if err := json.Unmarshal(ciphertext, &envelope); err != nil {
				t.Fatal(err)
			}
			entry.mutate(t, envelope)
			resolver := &softwareResolver{key: softwareTestKEK()}
			association, binding := softwareFixture()
			result, err := softwareAdapter(t, resolver).Unwrap(context.Background(), UnwrapRequest{
				Association: association, Context: binding, Ciphertext: softwareMarshal(t, envelope),
			})
			assertSoftwareRefusal(t, err, entry.kind, "software_jwe")
			if result.Plaintext != nil || result.Receipt != (KMSReceipt{}) || len(resolver.issued) != 0 {
				t.Fatal("closed-envelope refusal resolved material or returned a partial result")
			}
		})
	}
}

func TestSoftwareRefusalContractResolverCancellationClearsMaterial(t *testing.T) {
	for _, operation := range []string{"wrap", "unwrap"} {
		t.Run(operation, func(t *testing.T) {
			ciphertext := softwareRefusalCiphertext(t)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			resolver := &softwareResolver{
				key: softwareTestKEK(), cancel: true, contextCancel: cancel,
				err: errors.New("SYNTHETIC_RESOLVER_DIAGNOSTIC"),
			}
			adapter := softwareAdapter(t, resolver)
			association, binding := softwareFixture()
			var err error
			if operation == "wrap" {
				var result WrapResult
				result, err = adapter.Wrap(ctx, WrapRequest{Association: association, Context: binding, Plaintext: softwareTestDEK()})
				if result.Ciphertext != nil || result.Receipt != (KMSReceipt{}) {
					t.Fatal("canceled wrap returned a partial result")
				}
			} else {
				var result UnwrapResult
				result, err = adapter.Unwrap(ctx, UnwrapRequest{Association: association, Context: binding, Ciphertext: ciphertext})
				if result.Plaintext != nil || result.Receipt != (KMSReceipt{}) {
					t.Fatal("canceled unwrap returned a partial result")
				}
			}
			assertSoftwareRefusal(t, err, fault.Denied, "software_cancelled")
			if strings.Contains(err.Error(), "SYNTHETIC_RESOLVER_DIAGNOSTIC") || len(resolver.issued) != 1 || !resolver.allCleared() {
				t.Fatal("cancellation leaked a diagnostic, retried, or retained resolver material")
			}
		})
	}
}

func softwareRefusalCiphertext(t *testing.T) []byte {
	t.Helper()
	association, binding := softwareFixture()
	result, err := softwareAdapter(t, &softwareResolver{key: softwareTestKEK()}).Wrap(context.Background(), WrapRequest{
		Association: association, Context: binding, Plaintext: softwareTestDEK(),
	})
	if err != nil || len(result.Ciphertext) == 0 {
		t.Fatalf("synthetic envelope setup: %v", err)
	}
	return result.Ciphertext
}

func softwareRefusalHeader(t *testing.T, envelope map[string]json.RawMessage, mutate func(map[string]json.RawMessage)) {
	t.Helper()
	encoded, err := base64.RawURLEncoding.Strict().DecodeString(jsonString(t, envelope["protected"]))
	if err != nil {
		t.Fatal(err)
	}
	var header map[string]json.RawMessage
	if err := json.Unmarshal(encoded, &header); err != nil {
		t.Fatal(err)
	}
	mutate(header)
	envelope["protected"] = softwareRawJSONString(t, base64.RawURLEncoding.EncodeToString(softwareMarshal(t, header)))
}

func assertSoftwareRefusal(t *testing.T, err error, kind fault.Kind, operation string) {
	t.Helper()
	var safe *fault.Error
	if !errors.As(err, &safe) || safe.Kind != kind || safe.Operation != operation {
		t.Fatalf("refusal = %v, want %s/%s", err, kind, operation)
	}
}
