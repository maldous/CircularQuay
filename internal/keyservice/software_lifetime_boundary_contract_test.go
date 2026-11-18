package keyservice

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"github.com/go-jose/go-jose/v4"
	"github.com/maldous/CircularQuay/internal/fault"
)

// These are per-operation lifetime/failure contracts, not injected malformed
// JOSE private state or a claim that a fixed valid constructor can fail.
func TestSoftwareLifetimeBoundaryContractEncryptionFailureClearsResolverKey(t *testing.T) {
	resolver := &softwareResolver{key: softwareTestKEK()}
	adapter := softwareAdapter(t, resolver)
	association, binding := softwareFixture()
	plaintext := softwareTestDEK()
	before := bytes.Clone(plaintext)
	factory := softwareLifetimeFactory(t, func() {}, true)
	result, err := adapter.wrapWithEncrypter(context.Background(), WrapRequest{Association: association, Context: binding, Plaintext: plaintext}, factory)
	assertSoftwareLifetimeFault(t, err, fault.Trust, "software_encrypt")
	if result.Ciphertext != nil || result.Receipt != (KMSReceipt{}) || !resolver.allCleared() || !bytes.Equal(plaintext, before) {
		t.Fatal("failed encryption retained resolver material, changed caller bytes or released a partial result")
	}
}

func TestSoftwareLifetimeBoundaryContractCancellationAfterRealEncryptionReturnsNoEnvelope(t *testing.T) {
	resolver := &softwareResolver{key: softwareTestKEK()}
	adapter := softwareAdapter(t, resolver)
	association, binding := softwareFixture()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	factory := softwareLifetimeFactory(t, cancel, false)
	result, err := adapter.wrapWithEncrypter(ctx, WrapRequest{Association: association, Context: binding, Plaintext: softwareTestDEK()}, factory)
	assertSoftwareLifetimeFault(t, err, fault.Denied, "software_cancelled")
	if result.Ciphertext != nil || result.Receipt != (KMSReceipt{}) || !resolver.allCleared() {
		t.Fatal("post-encryption cancellation released an envelope/receipt or retained resolver key")
	}
}

func TestSoftwareLifetimeBoundaryContractCancellationAfterRealDecryptClearsPlaintext(t *testing.T) {
	resolver := &softwareResolver{key: softwareTestKEK()}
	adapter := softwareAdapter(t, resolver)
	association, binding := softwareFixture()
	wrapped, err := adapter.Wrap(context.Background(), WrapRequest{Association: association, Context: binding, Plaintext: softwareTestDEK()})
	if err != nil {
		t.Fatal(err)
	}
	before := bytes.Clone(wrapped.Ciphertext)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var decrypted []byte
	decrypt := func(envelope *jose.JSONWebEncryption, key any) ([]byte, error) {
		plaintext, err := envelope.Decrypt(key)
		if err != nil || !bytes.Equal(plaintext, softwareTestDEK()) {
			t.Fatal("real JOSE decrypt did not produce the exact synthetic key")
		}
		decrypted = plaintext
		cancel()
		return plaintext, nil
	}
	result, err := adapter.unwrapWithDecrypt(ctx, UnwrapRequest{Association: association, Context: binding, Ciphertext: wrapped.Ciphertext}, decrypt)
	assertSoftwareLifetimeFault(t, err, fault.Denied, "software_cancelled")
	assertSoftwareLifetimeCleared(t, result, resolver, decrypted)
	if !bytes.Equal(wrapped.Ciphertext, before) {
		t.Fatal("cancelled decrypt changed caller ciphertext")
	}
}

func assertSoftwareLifetimeCleared(t *testing.T, result UnwrapResult, resolver *softwareResolver, decrypted []byte) {
	t.Helper()
	if result.Plaintext != nil || result.Receipt != (KMSReceipt{}) || len(decrypted) != softwareKeyBytes {
		t.Fatal("cancelled decrypt returned partial metadata or did not reach real plaintext")
	}
	if !zeroed(decrypted) || !resolver.allCleared() {
		t.Fatal("cancelled decrypt retained owned plaintext or resolver key material")
	}
}

func TestSoftwareLifetimeBoundaryContractCancelledResolveNeverCallsResolver(t *testing.T) {
	resolver := &softwareResolver{key: softwareTestKEK()}
	adapter := softwareAdapter(t, resolver)
	association, _ := softwareFixture()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	key, err := adapter.resolve(ctx, association)
	assertSoftwareLifetimeFault(t, err, fault.Denied, "software_cancelled")
	if key != nil || len(resolver.issued) != 0 {
		t.Fatal("cancelled internal resolution reached the resolver or returned key material")
	}
}

func TestSoftwareLifetimeBoundaryContractPerCallFactoryHasRealCryptoPositiveControl(t *testing.T) {
	resolver := &softwareResolver{key: softwareTestKEK()}
	adapter := softwareAdapter(t, resolver)
	association, binding := softwareFixture()
	wrapped, err := adapter.wrapWithEncrypter(context.Background(), WrapRequest{Association: association, Context: binding, Plaintext: softwareTestDEK()}, softwareLifetimeFactory(t, func() {}, false))
	if err != nil || len(wrapped.Ciphertext) == 0 {
		t.Fatal("real JOSE positive control did not produce an envelope")
	}
	result, err := adapter.Unwrap(context.Background(), UnwrapRequest{Association: association, Context: binding, Ciphertext: wrapped.Ciphertext})
	if err != nil || !bytes.Equal(result.Plaintext, softwareTestDEK()) || !resolver.allCleared() {
		t.Fatal("per-call factory did not preserve real JOSE roundtrip and resolver ownership")
	}
	clearBytes(result.Plaintext)
}

func assertSoftwareLifetimeFault(t *testing.T, err error, kind fault.Kind, operation string) {
	t.Helper()
	assertSoftwareRefusal(t, err, kind, operation)
	if err.Error() != string(kind)+": "+operation || errors.Unwrap(err) != nil {
		t.Fatal("software lifetime failure retained upstream diagnostics or an error cause")
	}
}

func softwareLifetimeFactory(t *testing.T, after func(), failEncrypt bool) func(jose.ContentEncryption, jose.Recipient, *jose.EncrypterOptions) (jose.Encrypter, error) {
	t.Helper()
	return func(algorithm jose.ContentEncryption, recipient jose.Recipient, options *jose.EncrypterOptions) (jose.Encrypter, error) {
		assertSoftwareLifetimeArguments(t, algorithm, recipient, options)
		encrypter, err := jose.NewEncrypter(algorithm, recipient, options)
		if err != nil {
			return nil, err
		}
		return softwareLifetimeEncrypter{Encrypter: encrypter, after: after, failEncrypt: failEncrypt}, nil
	}
}

func assertSoftwareLifetimeArguments(t *testing.T, algorithm jose.ContentEncryption, recipient jose.Recipient, options *jose.EncrypterOptions) {
	t.Helper()
	key, valid := recipient.Key.([]byte)
	if algorithm != jose.A256GCM || recipient.Algorithm != jose.A256KW || !valid || len(key) != softwareKeyBytes {
		t.Fatal("factory changed fixed algorithms or resolver-owned key type/length")
	}
	association, _ := softwareFixture()
	if recipient.KeyID != association.KeyID || options == nil {
		t.Fatal("factory changed key identifier or omitted protected options")
	}
	if options.ExtraHeaders[jose.HeaderKey(softwareHeaderKV)] != association.KeyVersion || options.ExtraHeaders[jose.HeaderKey(softwareHeaderCA)] != association.ConfigurationAttestation {
		t.Fatal("factory changed protected version/attestation")
	}
}

type softwareLifetimeEncrypter struct {
	jose.Encrypter
	after       func()
	failEncrypt bool
}

func (encrypter softwareLifetimeEncrypter) EncryptWithAuthData(plaintext, aad []byte) (*jose.JSONWebEncryption, error) {
	if encrypter.failEncrypt {
		// Operation-level I/O-failure stub. This does not replace global entropy,
		// forge a constructor failure or claim an actual entropy failure occurred.
		return nil, errors.New("PRIVATE_ENTROPY_OPERATION_DETAIL")
	}
	envelope, err := encrypter.Encrypter.EncryptWithAuthData(plaintext, aad)
	encrypter.after()
	return envelope, err
}
