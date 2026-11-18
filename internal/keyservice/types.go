// Package keyservice defines profile-bound remote custody decisions.
//
// It owns no provider credentials, network listener, storage implementation, or
// cryptography. Callers supply authenticated authorization, association storage,
// enrollment provisioning, and established provider primitives.
package keyservice

import "context"

const (
	maxIdentifierBytes = 256
	maxKeyReference    = 4096
	maxKeyMaterial     = 64 << 10
)

// ProfileBinding identifies one enrolled application profile and principal.
// Values are opaque, non-secret identifiers; they are never inferred from an
// operating-system account, shell profile, or provider credential.
type ProfileBinding struct {
	Tenant      string
	Profile     string
	Application string
	Principal   string
}

// Artifact identifies one approved opaque workload or encrypted bundle revision.
// It contains no configuration graph, source reference, endpoint, or secret.
type Artifact struct {
	ID      string
	Version string
}

// KMSAssociation is safe, persisted identity for the exact remote KMS resource
// authorized for a profile. Provider connection and credential references stay
// inside a concrete adapter; they are not enrollment/audit/response metadata.
// KeyVersion is the exact pinned native-provider key version, distinct from
// Enrollment.Generation and never request-selected.
type KMSAssociation struct {
	Provider                 string
	KeyID                    string
	KeyVersion               string
	Algorithm                string
	Purpose                  string
	ConfigurationAttestation string
}

// AgeAssociation identifies the hosted service record for an age private key.
// It contains public/opaque metadata only; it never contains private key bytes.
type AgeAssociation struct {
	ServiceKeyID    string
	PublicRecipient string
}

// Enrollment is the enabled custody association for one exact profile binding.
// Both KMS and hosted-age associations are mandatory before it can be enabled.
type Enrollment struct {
	Binding    ProfileBinding
	Generation string
	KMS        KMSAssociation
	Age        AgeAssociation
}

// EnrollmentRequest asks an already authenticated authority to provision both
// profile-bound associations. OperationID makes the caller's provisioning action
// idempotent; it is not a credential or a key identifier.
type EnrollmentRequest struct {
	Binding            ProfileBinding
	OperationID        string
	ExpectedGeneration string
}

// EnrollmentDisposition distinguishes an enabled complete association from an
// external provisioning outcome that requires reconciliation.
type EnrollmentDisposition string

const (
	// EnrollmentEnabled means both associations were provisioned and stored active.
	EnrollmentEnabled EnrollmentDisposition = "enabled"
	// EnrollmentReconcile means no partial profile may be enabled; an uncertain
	// full activation requires durable confirmation before it is reported enabled.
	EnrollmentReconcile EnrollmentDisposition = "reconcile"
)

// EnrollmentResult contains only safe association status and metadata.
type EnrollmentResult struct {
	Disposition EnrollmentDisposition
	Enrollment  Enrollment
}

// Operation is an exact custody action authorized against the current enrollment.
type Operation string

const (
	OperationEnroll      Operation = "enroll"
	OperationWrap        Operation = "wrap"
	OperationUnwrap      Operation = "unwrap"
	OperationRetrieveAge Operation = "retrieve_age_private_key"
)

// AssociationContext is built by Service from the active enrollment. Provider
// implementations must bind their maintained cryptographic operation to these
// exact fields; callers cannot supply an arbitrary AAD digest.
type AssociationContext struct {
	Tenant          string
	Profile         string
	Application     string
	Principal       string
	Generation      string
	Purpose         string
	ArtifactID      string
	ArtifactVersion string
}

// WrapRequest is passed to an established provider KMS primitive. Plaintext is
// intended to be a short data key and must never be logged or serialized.
type WrapRequest struct {
	Association KMSAssociation
	Context     AssociationContext
	Plaintext   []byte
}

// UnwrapRequest is passed to an established provider KMS primitive. Ciphertext
// must be bound by the provider to the supplied association context.
type UnwrapRequest struct {
	Association KMSAssociation
	Context     AssociationContext
	Ciphertext  []byte
}

// AgePrivateKeyRequest authorizes a deliberate hosted private-key retrieval.
// Enrollment never calls this operation automatically, including after KMS
// failure. The returned private key is sensitive []byte material only.
type AgePrivateKeyRequest struct {
	Association AgeAssociation
	Context     AssociationContext
}

// Authorizer verifies the current caller's authority for one exact binding and
// operation. It must not treat a global or ambient grant as profile authority.
type Authorizer interface {
	Authorize(context.Context, ProfileBinding, Artifact, Operation, Enrollment) error
}

// EnrollmentLookup returns the current active association. A false result means
// no enrollment is active for the exact requested profile.
type EnrollmentLookup interface {
	Active(context.Context, ProfileBinding) (Enrollment, bool, error)
}

// EnrollmentStore records an already complete enrollment as active. It must
// never report an association enabled when a partial or uncertain mutation
// requires reconciliation.
type EnrollmentStore interface {
	EnrollmentLookup
	// Enable atomically consumes request.OperationID and conditionally enables
	// enrollment only when the active generation equals ExpectedGeneration. An
	// uncertain or partial mutation returns EnrollmentReconcile, never enabled.
	Enable(context.Context, EnrollmentRequest, Enrollment) (EnrollmentDisposition, error)
}

// EnrollmentProvisioner creates or verifies both associations for one idempotent
// enrollment request. It returns EnrollmentReconcile for partial or uncertain
// external state and must not report it as enabled.
type EnrollmentProvisioner interface {
	Provision(context.Context, EnrollmentRequest) (EnrollmentResult, error)
}

// KMSReceipt records bounded, non-secret provider identity observed by an adapter.
// Service compares it to the exact active association before exposing either
// result. It is not a substitute for provider cryptographic attestation.
type KMSReceipt struct {
	Provider                 string
	KeyID                    string
	KeyVersion               string
	Algorithm                string
	ConfigurationAttestation string
	Context                  AssociationContext
}

// WrapResult is a KMS ciphertext plus its non-secret observed identity. Ownership
// of Ciphertext transfers to Service, which best-effort clears it after copying
// an authorized result or rejecting it.
type WrapResult struct {
	Ciphertext []byte
	Receipt    KMSReceipt
}

// UnwrapResult is a KMS plaintext data key plus its non-secret observed identity.
// Plaintext ownership transfers to Service and is best-effort cleared after an
// authorized copy or rejection. It must never be logged or serialized.
type UnwrapResult struct {
	Plaintext []byte
	Receipt   KMSReceipt
}

// ProviderKMS is implemented by an established remote KMS adapter. It performs
// no authorization decision: Service verifies the current exact enrollment and
// adapter-observed identity before returning material, then reauthorizes after
// the provider wait.
type ProviderKMS interface {
	Wrap(context.Context, WrapRequest) (WrapResult, error)
	Unwrap(context.Context, UnwrapRequest) (UnwrapResult, error)
}

// AgeCustodian is the authenticated hosted age-private-key service boundary.
// Retrieval is explicit; implementations must not log, JSON encode, or cache the
// returned bytes. Ownership of returned bytes transfers to Service, which
// best-effort clears them after an authorized copy or rejection. A remote SOPS
// key-service bridge may instead keep this material remote, but enrollment
// itself never downloads it. Go cannot guarantee cryptographic memory erasure.
type AgeCustodian interface {
	RetrievePrivateKey(context.Context, AgePrivateKeyRequest) ([]byte, error)
}
