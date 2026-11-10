package keyservice

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/maldous/CircularQuay/internal/fault"
)

func TestEnrollRejectsIncompleteAssociationWithoutEnable(t *testing.T) {
	store := &memoryStore{}
	provisioner := provisionFunc(func(context.Context, EnrollmentRequest) (EnrollmentResult, error) {
		result := EnrollmentResult{Disposition: EnrollmentEnabled, Enrollment: fixtureEnrollment()}
		result.Enrollment.Age = AgeAssociation{}
		return result, nil
	})
	service := fixtureService(t, store, provisioner, &recordingKMS{}, &recordingAge{})
	_, err := service.Enroll(context.Background(), fixtureRequest())
	if !fault.Is(err, fault.Trust) || store.enabled != 0 {
		t.Fatalf("incomplete enrollment enabled=%d error=%v", store.enabled, err)
	}
}

func TestEnrollReconcileNeverEnablesHalfProfile(t *testing.T) {
	store := &memoryStore{}
	provisioner := provisionFunc(func(context.Context, EnrollmentRequest) (EnrollmentResult, error) {
		return EnrollmentResult{Disposition: EnrollmentReconcile}, nil
	})
	service := fixtureService(t, store, provisioner, &recordingKMS{}, &recordingAge{})
	result, err := service.Enroll(context.Background(), fixtureRequest())
	if err != nil || result.Disposition != EnrollmentReconcile || store.enabled != 0 {
		t.Fatalf("reconcile result=%+v enabled=%d error=%v", result, store.enabled, err)
	}
}

func TestEnrollStoreUncertaintyReturnsReconcile(t *testing.T) {
	store := &memoryStore{enableErr: errors.New("storage diagnostic")}
	service := fixtureService(t, store, provisionFunc(enabledProvision), &recordingKMS{}, &recordingAge{})
	result, err := service.Enroll(context.Background(), fixtureRequest())
	if result.Disposition != EnrollmentReconcile || !fault.Is(err, fault.Apply) || strings.Contains(err.Error(), "storage diagnostic") {
		t.Fatalf("store uncertainty result=%+v error=%v", result, err)
	}
}

func TestEnrollExpectedGenerationConflictReconciles(t *testing.T) {
	request := fixtureRequest()
	request.ExpectedGeneration = "other-generation"
	store := &memoryStore{enrollment: fixtureEnrollment(), found: true}
	service := fixtureService(t, store, provisionFunc(enabledProvision), &recordingKMS{}, &recordingAge{})
	result, err := service.Enroll(context.Background(), request)
	if err != nil || result.Disposition != EnrollmentReconcile || store.enabled != 0 {
		t.Fatalf("generation result=%+v enabled=%d error=%v", result, store.enabled, err)
	}
}

func TestEnrollReauthorizesBeforeEnable(t *testing.T) {
	store := &memoryStore{}
	authorizer := &recordingAuthorizer{}
	age := &recordingAge{}
	service, err := NewService(Dependencies{Authorizer: authorizer, Store: store, Provisioner: provisionFunc(enabledProvision), KMS: &recordingKMS{}, Age: age})
	if err != nil {
		t.Fatal(err)
	}
	result, err := service.Enroll(context.Background(), fixtureRequest())
	if err != nil || result.Disposition != EnrollmentEnabled || store.enabled != 1 || authorizer.calls != 3 || age.calls != 0 {
		t.Fatalf("result=%+v enabled=%d authorization=%d age_calls=%d error=%v", result, store.enabled, authorizer.calls, age.calls, err)
	}
}

func TestWrapPreservesOnlyTypedAvailability(t *testing.T) {
	store := &memoryStore{enrollment: fixtureEnrollment(), found: true}
	kms := &recordingKMS{wrapError: fault.New(fault.Unavailable, "synthetic")}
	service := fixtureService(t, store, provisionFunc(enabledProvision), kms, &recordingAge{})
	if _, err := service.Wrap(context.Background(), fixtureBinding(), fixtureArtifact(), []byte("data-key")); !fault.Is(err, fault.Unavailable) {
		t.Fatalf("typed availability = %v", err)
	}
	kms.wrapError = errors.New("provider diagnostic")
	if _, err := service.Wrap(context.Background(), fixtureBinding(), fixtureArtifact(), []byte("data-key")); !fault.Is(err, fault.Apply) || strings.Contains(err.Error(), "provider diagnostic") {
		t.Fatalf("generic provider error = %v", err)
	}
}

func TestWrapRejectsMismatchedReceiptWithoutCiphertext(t *testing.T) {
	service := fixtureService(t, &memoryStore{enrollment: fixtureEnrollment(), found: true}, provisionFunc(enabledProvision), kmsFunc{wrap: func(_ context.Context, request WrapRequest) (WrapResult, error) {
		receipt := receipt(request.Association, request.Context)
		receipt.KeyVersion = "other"
		return WrapResult{Ciphertext: []byte("wrapped"), Receipt: receipt}, nil
	}}, &recordingAge{})
	value, err := service.Wrap(context.Background(), fixtureBinding(), fixtureArtifact(), []byte("data-key"))
	if value != nil || !fault.Is(err, fault.Trust) {
		t.Fatalf("mismatched receipt ciphertext=%q error=%v", value, err)
	}
}

func TestWrapClearsServiceBuffersWithoutMutatingCaller(t *testing.T) {
	caller := []byte("data-key")
	providerCiphertext := []byte("wrapped")
	var received []byte
	service := fixtureService(t, &memoryStore{enrollment: fixtureEnrollment(), found: true}, provisionFunc(enabledProvision), kmsFunc{wrap: func(_ context.Context, request WrapRequest) (WrapResult, error) {
		received = request.Plaintext
		receipt := receipt(request.Association, request.Context)
		receipt.KeyID = "other"
		return WrapResult{Ciphertext: providerCiphertext, Receipt: receipt}, nil
	}}, &recordingAge{})
	if _, err := service.Wrap(context.Background(), fixtureBinding(), fixtureArtifact(), caller); !fault.Is(err, fault.Trust) {
		t.Fatalf("rejected wrap error=%v", err)
	}
	if !zeroed(received) || !zeroed(providerCiphertext) || string(caller) != "data-key" {
		t.Fatalf("cleanup input=%q ciphertext=%q caller=%q", received, providerCiphertext, caller)
	}
}

func TestWrapClearsProviderBufferAfterAuthorizedCopy(t *testing.T) {
	providerCiphertext := []byte("wrapped")
	service := fixtureService(t, &memoryStore{enrollment: fixtureEnrollment(), found: true}, provisionFunc(enabledProvision), kmsFunc{wrap: func(_ context.Context, request WrapRequest) (WrapResult, error) {
		return WrapResult{Ciphertext: providerCiphertext, Receipt: receipt(request.Association, request.Context)}, nil
	}}, &recordingAge{})
	value, err := service.Wrap(context.Background(), fixtureBinding(), fixtureArtifact(), []byte("data-key"))
	if err != nil || string(value) != "wrapped" || !zeroed(providerCiphertext) {
		t.Fatalf("authorized wrap value=%q provider=%q error=%v", value, providerCiphertext, err)
	}
}

func TestWrapDeniesChangedAssociationAfterProviderWait(t *testing.T) {
	first, second := fixtureEnrollment(), fixtureEnrollment()
	second.Generation = "generation-2"
	store := &memoryStore{sequence: []Enrollment{first, second}, found: true}
	service := fixtureService(t, store, provisionFunc(enabledProvision), &recordingKMS{wrap: []byte("wrapped")}, &recordingAge{})
	value, err := service.Wrap(context.Background(), fixtureBinding(), fixtureArtifact(), []byte("data-key"))
	if value != nil || !fault.Is(err, fault.Denied) {
		t.Fatalf("changed association ciphertext=%q error=%v", value, err)
	}
}

func TestUnwrapDeniesChangedAssociationAfterProviderWait(t *testing.T) {
	first, second := fixtureEnrollment(), fixtureEnrollment()
	second.Generation = "generation-2"
	store := &memoryStore{sequence: []Enrollment{first, second}, found: true}
	service := fixtureService(t, store, provisionFunc(enabledProvision), &recordingKMS{unwrap: []byte("plain")}, &recordingAge{})
	plaintext, err := service.Unwrap(context.Background(), fixtureBinding(), fixtureArtifact(), []byte("cipher"))
	if plaintext != nil || !fault.Is(err, fault.Denied) {
		t.Fatalf("changed association plaintext=%q error=%v", plaintext, err)
	}
}

func TestUnwrapRejectsMismatchedReceiptWithoutPlaintext(t *testing.T) {
	service := fixtureService(t, &memoryStore{enrollment: fixtureEnrollment(), found: true}, provisionFunc(enabledProvision), kmsFunc{unwrap: func(_ context.Context, request UnwrapRequest) (UnwrapResult, error) {
		receipt := receipt(request.Association, request.Context)
		receipt.Algorithm = "other"
		return UnwrapResult{Plaintext: []byte("plain"), Receipt: receipt}, nil
	}}, &recordingAge{})
	value, err := service.Unwrap(context.Background(), fixtureBinding(), fixtureArtifact(), []byte("cipher"))
	if value != nil || !fault.Is(err, fault.Trust) {
		t.Fatalf("mismatched receipt plaintext=%q error=%v", value, err)
	}
}

func TestUnwrapClearsProviderPlaintextAfterAuthorizedCopy(t *testing.T) {
	providerPlaintext := []byte("plain")
	service := fixtureService(t, &memoryStore{enrollment: fixtureEnrollment(), found: true}, provisionFunc(enabledProvision), kmsFunc{unwrap: func(_ context.Context, request UnwrapRequest) (UnwrapResult, error) {
		return UnwrapResult{Plaintext: providerPlaintext, Receipt: receipt(request.Association, request.Context)}, nil
	}}, &recordingAge{})
	value, err := service.Unwrap(context.Background(), fixtureBinding(), fixtureArtifact(), []byte("cipher"))
	if err != nil || string(value) != "plain" || !zeroed(providerPlaintext) {
		t.Fatalf("authorized unwrap value=%q provider=%q error=%v", value, providerPlaintext, err)
	}
}

func TestRetrieveAgePrivateKeyIsExplicitAndReturnsCopy(t *testing.T) {
	age := &recordingAge{key: []byte("AGE-SECRET")}
	service := fixtureService(t, &memoryStore{enrollment: fixtureEnrollment(), found: true}, provisionFunc(enabledProvision), &recordingKMS{}, age)
	if age.calls != 0 {
		t.Fatal("fixture age custodian called before explicit retrieval")
	}
	first, err := service.RetrieveAgePrivateKey(context.Background(), fixtureBinding(), fixtureArtifact())
	if err != nil || age.calls != 1 {
		t.Fatalf("retrieve calls=%d error=%v", age.calls, err)
	}
	first[0] = 'X'
	if !zeroed(age.key) {
		t.Fatal("service-owned custodian private key buffer was retained")
	}
}

func TestRetrieveAgePrivateKeyDeniesChangedEnrollmentAfterWait(t *testing.T) {
	first, second := fixtureEnrollment(), fixtureEnrollment()
	second.Generation = "generation-2"
	service := fixtureService(t, &memoryStore{sequence: []Enrollment{first, second}, found: true}, provisionFunc(enabledProvision), &recordingKMS{}, &recordingAge{key: []byte("AGE-SECRET")})
	value, err := service.RetrieveAgePrivateKey(context.Background(), fixtureBinding(), fixtureArtifact())
	if value != nil || !fault.Is(err, fault.Denied) {
		t.Fatalf("changed enrollment key=%q error=%v", value, err)
	}
}

func TestWrapCancellationDuringProviderWaitReturnsNoResult(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	service := fixtureService(t, &memoryStore{enrollment: fixtureEnrollment(), found: true}, provisionFunc(enabledProvision), kmsFunc{wrap: func(_ context.Context, request WrapRequest) (WrapResult, error) {
		cancel()
		return WrapResult{Ciphertext: []byte("wrapped"), Receipt: receipt(request.Association, request.Context)}, nil
	}}, &recordingAge{})
	value, err := service.Wrap(ctx, fixtureBinding(), fixtureArtifact(), []byte("data-key"))
	if value != nil || !fault.Is(err, fault.Denied) {
		t.Fatalf("cancelled provider wait ciphertext=%q error=%v", value, err)
	}
}

func TestAuthorizationAndLookupErrorsAreSanitized(t *testing.T) {
	authorizer := &recordingAuthorizer{err: errors.New("authorization secret")}
	service, err := NewService(Dependencies{Authorizer: authorizer, Store: &memoryStore{enrollment: fixtureEnrollment(), found: true}, Provisioner: provisionFunc(enabledProvision), KMS: &recordingKMS{}, Age: &recordingAge{}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = service.Wrap(context.Background(), fixtureBinding(), fixtureArtifact(), []byte("data-key")); !fault.Is(err, fault.Denied) || strings.Contains(err.Error(), "authorization secret") {
		t.Fatalf("authorization error=%v", err)
	}
	lookup := &memoryStore{activeErr: errors.New("lookup secret")}
	service = fixtureService(t, lookup, provisionFunc(enabledProvision), &recordingKMS{}, &recordingAge{})
	if _, err = service.Wrap(context.Background(), fixtureBinding(), fixtureArtifact(), []byte("data-key")); !fault.Is(err, fault.Apply) || strings.Contains(err.Error(), "lookup secret") {
		t.Fatalf("lookup error=%v", err)
	}
}

func TestCancellationPreventsProviderEffects(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	kms := &recordingKMS{}
	service := fixtureService(t, &memoryStore{enrollment: fixtureEnrollment(), found: true}, provisionFunc(enabledProvision), kms, &recordingAge{})
	if _, err := service.Wrap(ctx, fixtureBinding(), fixtureArtifact(), []byte("data-key")); !fault.Is(err, fault.Denied) || kms.wrapCalls != 0 {
		t.Fatalf("cancelled wrap calls=%d error=%v", kms.wrapCalls, err)
	}
}

func fixtureService(t *testing.T, store EnrollmentStore, provisioner EnrollmentProvisioner, kms ProviderKMS, age AgeCustodian) *Service {
	t.Helper()
	service, err := NewService(Dependencies{Authorizer: &recordingAuthorizer{}, Store: store, Provisioner: provisioner, KMS: kms, Age: age})
	if err != nil {
		t.Fatal(err)
	}
	return service
}

func fixtureBinding() ProfileBinding {
	return ProfileBinding{Tenant: "tenant", Profile: "profile", Application: "application", Principal: "principal"}
}

func fixtureArtifact() Artifact {
	return Artifact{ID: "workload", Version: "revision-1"}
}

func fixtureRequest() EnrollmentRequest {
	return EnrollmentRequest{Binding: fixtureBinding(), OperationID: "operation-1"}
}

func fixtureEnrollment() Enrollment {
	return Enrollment{
		Binding: fixtureBinding(), Generation: "generation-1",
		KMS: KMSAssociation{Provider: "openbao-transit", KeyID: "workload-key", KeyVersion: "1", ConfigurationAttestation: "attestation-1", Algorithm: "aes256-gcm96", Purpose: "workload-data-key"},
		Age: AgeAssociation{ServiceKeyID: "age-record-1", PublicRecipient: "age1synthetic"},
	}
}

func enabledProvision(_ context.Context, request EnrollmentRequest) (EnrollmentResult, error) {
	enrollment := fixtureEnrollment()
	enrollment.Binding = request.Binding
	return EnrollmentResult{Disposition: EnrollmentEnabled, Enrollment: enrollment}, nil
}

type provisionFunc func(context.Context, EnrollmentRequest) (EnrollmentResult, error)

func (f provisionFunc) Provision(ctx context.Context, request EnrollmentRequest) (EnrollmentResult, error) {
	return f(ctx, request)
}

type recordingAuthorizer struct {
	calls int
	err   error
}

func (a *recordingAuthorizer) Authorize(_ context.Context, _ ProfileBinding, _ Artifact, _ Operation, _ Enrollment) error {
	a.calls++
	return a.err
}

type memoryStore struct {
	enrollment Enrollment
	sequence   []Enrollment
	found      bool
	enabled    int
	reads      int
	enableErr  error
	activeErr  error
}

func (s *memoryStore) Active(_ context.Context, _ ProfileBinding) (Enrollment, bool, error) {
	if s.activeErr != nil {
		return Enrollment{}, false, s.activeErr
	}
	if len(s.sequence) > 0 {
		index := s.reads
		if index >= len(s.sequence) {
			index = len(s.sequence) - 1
		}
		s.reads++
		return s.sequence[index], s.found, nil
	}
	return s.enrollment, s.found, nil
}

func (s *memoryStore) Enable(_ context.Context, request EnrollmentRequest, enrollment Enrollment) (EnrollmentDisposition, error) {
	if s.enableErr != nil {
		return EnrollmentReconcile, s.enableErr
	}
	if request.ExpectedGeneration != "" && request.ExpectedGeneration != s.enrollment.Generation {
		return EnrollmentReconcile, nil
	}
	s.enabled++
	s.enrollment, s.found = enrollment, true
	return EnrollmentEnabled, nil
}

type kmsFunc struct {
	wrap   func(context.Context, WrapRequest) (WrapResult, error)
	unwrap func(context.Context, UnwrapRequest) (UnwrapResult, error)
}

func (f kmsFunc) Wrap(ctx context.Context, request WrapRequest) (WrapResult, error) {
	if f.wrap == nil {
		return WrapResult{}, fault.New(fault.Unsupported, "fixture_wrap")
	}
	return f.wrap(ctx, request)
}

func (f kmsFunc) Unwrap(ctx context.Context, request UnwrapRequest) (UnwrapResult, error) {
	if f.unwrap == nil {
		return UnwrapResult{}, fault.New(fault.Unsupported, "fixture_unwrap")
	}
	return f.unwrap(ctx, request)
}

type recordingKMS struct {
	wrap      []byte
	unwrap    []byte
	wrapError error
	unwrapErr error
	wrapCalls int
}

func (k *recordingKMS) Wrap(_ context.Context, request WrapRequest) (WrapResult, error) {
	k.wrapCalls++
	return WrapResult{Ciphertext: k.wrap, Receipt: receipt(request.Association, request.Context)}, k.wrapError
}

func (k *recordingKMS) Unwrap(_ context.Context, request UnwrapRequest) (UnwrapResult, error) {
	return UnwrapResult{Plaintext: k.unwrap, Receipt: receipt(request.Association, request.Context)}, k.unwrapErr
}

func receipt(association KMSAssociation, context AssociationContext) KMSReceipt {
	return KMSReceipt{Provider: association.Provider, KeyID: association.KeyID, KeyVersion: association.KeyVersion, Algorithm: association.Algorithm, ConfigurationAttestation: association.ConfigurationAttestation, Context: context}
}

func zeroed(value []byte) bool {
	for _, item := range value {
		if item != 0 {
			return false
		}
	}
	return true
}

type recordingAge struct {
	key   []byte
	err   error
	calls int
}

func (a *recordingAge) RetrievePrivateKey(_ context.Context, _ AgePrivateKeyRequest) ([]byte, error) {
	a.calls++
	return a.key, a.err
}
