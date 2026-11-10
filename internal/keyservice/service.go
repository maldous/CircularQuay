package keyservice

import (
	"context"
	"strings"
	"unicode/utf8"

	"github.com/maldous/CircularQuay/internal/fault"
)

// Service applies profile-bound enrollment and custody invariants around
// caller-provided provider primitives. It does not select a fallback, retry,
// cache material, discover credentials, or implement cryptography.
type Service struct {
	authorizer  Authorizer
	lookup      EnrollmentLookup
	store       EnrollmentStore
	provisioner EnrollmentProvisioner
	kms         ProviderKMS
	age         AgeCustodian
}

// Dependencies supplies the explicit collaborators required by Service.
type Dependencies struct {
	Authorizer  Authorizer
	Store       EnrollmentStore
	Provisioner EnrollmentProvisioner
	KMS         ProviderKMS
	Age         AgeCustodian
}

// NewService validates explicit collaborators without performing I/O.
func NewService(dependencies Dependencies) (*Service, error) {
	if dependencies.Authorizer == nil || dependencies.Store == nil || dependencies.Provisioner == nil || dependencies.KMS == nil || dependencies.Age == nil {
		return nil, fault.New(fault.Invalid, "keyservice_dependencies")
	}
	return &Service{
		authorizer:  dependencies.Authorizer,
		lookup:      dependencies.Store,
		store:       dependencies.Store,
		provisioner: dependencies.Provisioner,
		kms:         dependencies.KMS,
		age:         dependencies.Age,
	}, nil
}

// Enroll authorizes and provisions both associations before enabling a profile.
// A partial or uncertain provision returns EnrollmentReconcile without enabling a
// half-profile. Distributed external creation is not claimed to be atomic.
func (s *Service) Enroll(ctx context.Context, request EnrollmentRequest) (EnrollmentResult, error) {
	if err := checkContext(ctx); err != nil {
		return EnrollmentResult{}, err
	}
	if !validEnrollmentRequest(request) {
		return EnrollmentResult{}, fault.New(fault.Invalid, "keyservice_enrollment")
	}
	if err := s.authorize(ctx, request.Binding, Artifact{}, OperationEnroll, Enrollment{}); err != nil {
		return EnrollmentResult{}, err
	}
	result, err := s.provisioner.Provision(ctx, request)
	if err != nil {
		return reconcileProvision(result, err)
	}
	return s.completeEnrollment(ctx, request, result)
}

func (s *Service) completeEnrollment(ctx context.Context, request EnrollmentRequest, result EnrollmentResult) (EnrollmentResult, error) {
	if err := checkContext(ctx); err != nil {
		return reconcile(err)
	}
	if result.Disposition == EnrollmentReconcile {
		return EnrollmentResult{Disposition: EnrollmentReconcile}, nil
	}
	if result.Disposition != EnrollmentEnabled || !validEnrollmentFor(request.Binding, result.Enrollment) {
		return reconcile(fault.New(fault.Trust, "keyservice_enrollment"))
	}
	if err := s.authorize(ctx, request.Binding, Artifact{}, OperationEnroll, result.Enrollment); err != nil {
		return reconcile(err)
	}
	if err := checkContext(ctx); err != nil {
		return reconcile(err)
	}
	return s.commitEnrollment(ctx, request, result.Enrollment)
}

func (s *Service) commitEnrollment(ctx context.Context, request EnrollmentRequest, enrollment Enrollment) (EnrollmentResult, error) {
	disposition, err := s.store.Enable(ctx, request, enrollment)
	if err != nil {
		return reconcile(providerFailure(err))
	}
	if disposition != EnrollmentEnabled {
		return EnrollmentResult{Disposition: EnrollmentReconcile}, nil
	}
	if err := checkContext(ctx); err != nil {
		return reconcile(err)
	}
	current, found, err := s.lookup.Active(ctx, request.Binding)
	if err != nil || !found || !sameEnrollment(current, enrollment) {
		return reconcile(enrollmentConfirmationFailure(err))
	}
	if err := s.authorize(ctx, request.Binding, Artifact{}, OperationEnroll, current); err != nil {
		return reconcile(err)
	}
	return EnrollmentResult{Disposition: EnrollmentEnabled, Enrollment: enrollment}, nil
}

func reconcileProvision(_ EnrollmentResult, err error) (EnrollmentResult, error) {
	return reconcile(providerFailure(err))
}

func reconcile(err error) (EnrollmentResult, error) {
	return EnrollmentResult{Disposition: EnrollmentReconcile}, err
}

func enrollmentConfirmationFailure(err error) error {
	if err != nil {
		return providerFailure(err)
	}
	return fault.New(fault.Denied, "keyservice_enrollment_confirmation")
}

// Wrap invokes only the current profile's selected remote KMS association.
func (s *Service) Wrap(ctx context.Context, binding ProfileBinding, artifact Artifact, plaintext []byte) ([]byte, error) {
	if !validArtifact(artifact) || !validKeyMaterial(plaintext) {
		return nil, fault.New(fault.Invalid, "keyservice_plaintext")
	}
	enrollment, err := s.active(ctx, binding, artifact, OperationWrap)
	if err != nil {
		return nil, err
	}
	request := WrapRequest{Association: enrollment.KMS, Context: associationContext(enrollment, artifact), Plaintext: copyBytes(plaintext)}
	defer clearBytes(request.Plaintext)
	result, err := s.kms.Wrap(ctx, request)
	defer clearBytes(result.Ciphertext)
	if err != nil {
		return nil, providerFailure(err)
	}
	if !validKMSResult(result.Receipt, request.Association, request.Context) || !validKeyMaterial(result.Ciphertext) {
		return nil, fault.New(fault.Trust, "keyservice_wrapped_key")
	}
	if err := s.reauthorizeRelease(ctx, binding, artifact, OperationWrap, enrollment); err != nil {
		return nil, err
	}
	return copyBytes(result.Ciphertext), nil
}

// Unwrap returns data-key bytes only after re-reading and re-authorizing the
// exact profile association after the remote operation completes.
func (s *Service) Unwrap(ctx context.Context, binding ProfileBinding, artifact Artifact, ciphertext []byte) ([]byte, error) {
	if !validArtifact(artifact) || !validKeyMaterial(ciphertext) {
		return nil, fault.New(fault.Invalid, "keyservice_ciphertext")
	}
	enrollment, err := s.active(ctx, binding, artifact, OperationUnwrap)
	if err != nil {
		return nil, err
	}
	request := UnwrapRequest{Association: enrollment.KMS, Context: associationContext(enrollment, artifact), Ciphertext: copyBytes(ciphertext)}
	defer clearBytes(request.Ciphertext)
	result, err := s.kms.Unwrap(ctx, request)
	defer clearBytes(result.Plaintext)
	if err != nil {
		return nil, providerFailure(err)
	}
	if !validKMSResult(result.Receipt, request.Association, request.Context) || !validKeyMaterial(result.Plaintext) {
		return nil, fault.New(fault.Trust, "keyservice_unwrapped_key")
	}
	if err := s.reauthorizeRelease(ctx, binding, artifact, OperationUnwrap, enrollment); err != nil {
		return nil, err
	}
	return copyBytes(result.Plaintext), nil
}

// RetrieveAgePrivateKey deliberately obtains sensitive private-key bytes from
// the authenticated hosted age service. It is never called by Enroll, Wrap, or
// Unwrap and does not imply an offline profile or a fallback authorization.
func (s *Service) RetrieveAgePrivateKey(ctx context.Context, binding ProfileBinding, artifact Artifact) ([]byte, error) {
	if !validArtifact(artifact) {
		return nil, fault.New(fault.Invalid, "keyservice_artifact")
	}
	enrollment, err := s.active(ctx, binding, artifact, OperationRetrieveAge)
	if err != nil {
		return nil, err
	}
	key, err := s.age.RetrievePrivateKey(ctx, AgePrivateKeyRequest{Association: enrollment.Age, Context: associationContext(enrollment, artifact)})
	defer clearBytes(key)
	if err != nil {
		return nil, providerFailure(err)
	}
	if !validKeyMaterial(key) {
		return nil, fault.New(fault.Trust, "keyservice_age_private_key")
	}
	if err := s.reauthorizeRelease(ctx, binding, artifact, OperationRetrieveAge, enrollment); err != nil {
		return nil, err
	}
	return copyBytes(key), nil
}

func (s *Service) active(ctx context.Context, binding ProfileBinding, artifact Artifact, operation Operation) (Enrollment, error) {
	if err := checkContext(ctx); err != nil {
		return Enrollment{}, err
	}
	if !validBinding(binding) || !validArtifact(artifact) {
		return Enrollment{}, fault.New(fault.Invalid, "keyservice_binding")
	}
	enrollment, found, err := s.lookup.Active(ctx, binding)
	if err != nil {
		return Enrollment{}, providerFailure(err)
	}
	if !found || !validEnrollmentFor(binding, enrollment) {
		return Enrollment{}, fault.New(fault.Denied, "keyservice_enrollment")
	}
	if err := s.authorize(ctx, binding, artifact, operation, enrollment); err != nil {
		return Enrollment{}, err
	}
	return enrollment, checkContext(ctx)
}

func (s *Service) reauthorizeRelease(ctx context.Context, binding ProfileBinding, artifact Artifact, operation Operation, original Enrollment) error {
	if err := checkContext(ctx); err != nil {
		return err
	}
	current, found, err := s.lookup.Active(ctx, binding)
	if err != nil {
		return providerFailure(err)
	}
	if !found || !sameEnrollment(original, current) {
		return fault.New(fault.Denied, "keyservice_enrollment_changed")
	}
	if err := s.authorize(ctx, binding, artifact, operation, current); err != nil {
		return err
	}
	return checkContext(ctx)
}

func (s *Service) authorize(ctx context.Context, binding ProfileBinding, artifact Artifact, operation Operation, enrollment Enrollment) error {
	if err := s.authorizer.Authorize(ctx, binding, artifact, operation, enrollment); err != nil {
		return authorizationFailure(err)
	}
	return checkContext(ctx)
}

func associationContext(enrollment Enrollment, artifact Artifact) AssociationContext {
	return AssociationContext{Tenant: enrollment.Binding.Tenant, Profile: enrollment.Binding.Profile, Application: enrollment.Binding.Application, Principal: enrollment.Binding.Principal, Generation: enrollment.Generation, Purpose: enrollment.KMS.Purpose, ArtifactID: artifact.ID, ArtifactVersion: artifact.Version}
}

func checkContext(ctx context.Context) error {
	if ctx == nil || ctx.Err() != nil {
		return fault.New(fault.Denied, "keyservice_cancelled")
	}
	return nil
}

func validEnrollmentRequest(request EnrollmentRequest) bool {
	return validBinding(request.Binding) && validIdentifier(request.OperationID, maxIdentifierBytes) && validOptionalIdentifier(request.ExpectedGeneration, maxIdentifierBytes)
}

func validEnrollmentFor(binding ProfileBinding, enrollment Enrollment) bool {
	return enrollment.Binding == binding && validBinding(enrollment.Binding) && validIdentifier(enrollment.Generation, maxIdentifierBytes) && validKMS(enrollment.KMS) && validAge(enrollment.Age)
}

func validArtifact(artifact Artifact) bool {
	return validIdentifier(artifact.ID, maxIdentifierBytes) && validIdentifier(artifact.Version, maxIdentifierBytes)
}

func validBinding(binding ProfileBinding) bool {
	return validIdentifier(binding.Tenant, maxIdentifierBytes) && validIdentifier(binding.Profile, maxIdentifierBytes) && validIdentifier(binding.Application, maxIdentifierBytes) && validIdentifier(binding.Principal, maxIdentifierBytes)
}

func validKMS(association KMSAssociation) bool {
	return validIdentifier(association.Provider, maxIdentifierBytes) &&
		validIdentifier(association.KeyID, maxKeyReference) &&
		validIdentifier(association.KeyVersion, maxIdentifierBytes) &&
		validIdentifier(association.Algorithm, maxIdentifierBytes) &&
		validIdentifier(association.Purpose, maxIdentifierBytes) &&
		validIdentifier(association.ConfigurationAttestation, maxKeyReference)
}

func validKMSResult(receipt KMSReceipt, association KMSAssociation, context AssociationContext) bool {
	return receipt.Provider == association.Provider &&
		receipt.KeyID == association.KeyID &&
		receipt.KeyVersion == association.KeyVersion &&
		receipt.Algorithm == association.Algorithm &&
		receipt.ConfigurationAttestation == association.ConfigurationAttestation &&
		receipt.Context == context
}

func validAge(association AgeAssociation) bool {
	return validIdentifier(association.ServiceKeyID, maxKeyReference) && validIdentifier(association.PublicRecipient, maxKeyReference)
}

func validIdentifier(value string, limit int) bool {
	return value != "" && len(value) <= limit && utf8.ValidString(value) && !strings.ContainsAny(value, "\x00\r\n")
}

func validOptionalIdentifier(value string, limit int) bool {
	return value == "" || validIdentifier(value, limit)
}

func validKeyMaterial(value []byte) bool { return len(value) > 0 && len(value) <= maxKeyMaterial }

func copyBytes(value []byte) []byte { return append([]byte(nil), value...) }

// clearBytes is best-effort cleanup for Service-owned temporary material. Go's
// runtime and provider implementations can retain copies, so this is not a
// cryptographic zeroization guarantee.
func clearBytes(value []byte) { clear(value) }

func sameEnrollment(left, right Enrollment) bool {
	return left == right
}

func authorizationFailure(err error) error {
	if kind, found := faultKind(err); found {
		return fault.New(kind, "keyservice_authorization")
	}
	return fault.New(fault.Denied, "keyservice_authorization")
}

func providerFailure(err error) error {
	if kind, found := faultKind(err); found {
		return fault.New(kind, "keyservice_provider")
	}
	return fault.New(fault.Apply, "keyservice_provider")
}

func faultKind(err error) (fault.Kind, bool) {
	for _, kind := range []fault.Kind{fault.Invalid, fault.Trust, fault.Denied, fault.Unavailable, fault.Missing, fault.Expired, fault.Unsupported, fault.Apply} {
		if fault.Is(err, kind) {
			return kind, true
		}
	}
	return "", false
}
