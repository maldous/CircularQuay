package securitycontext

import (
	"context"
	"crypto"
	"time"

	jose "github.com/go-jose/go-jose/v4"
)

const (
	// MaxEnvelopeBytes bounds an entire JWS JSON envelope.
	MaxEnvelopeBytes = 64 << 10
	maxPayloadBytes  = 32 << 10
	maxSignatures    = 8
)

// Binding is selected from protected enrollment state before parsing a context.
// It is not derived from the context envelope, configuration, SDK, or request.
type Binding struct {
	Tenant      string
	ContextID   string
	TrustDomain string
	Audience    string
}

// TrustedKey is one enrolled public verification key. Its durable material
// identity is derived from canonical public-key bytes; callers cannot supply
// an alias string as a signer-distinctness proof.
type TrustedKey struct {
	KID       string
	Algorithm jose.SignatureAlgorithm
	Key       crypto.PublicKey
}

// Trust is separately protected enrollment data. It fixes the context signer
// and profile contract; Verify never fetches or discovers trust.
type Trust struct {
	Binding         Binding
	Issuer          string
	ProfileID       string
	ProfileRevision string
	Keys            []TrustedKey
	RequiredSigners int
}

// ContextGeneration is signed by a successor context. Its hash names only the
// already accepted predecessor, preventing a current-payload hash fixed point.
type ContextGeneration struct {
	RecoveryEpoch                 string `json:"recovery_epoch"`
	Sequence                      string `json:"sequence"`
	PreviousAcceptedPayloadSHA256 string `json:"previous_accepted_payload_sha256"`
}

// GenerationReference names an independently accepted authority or enrollment
// record. Its payload hash was computed after that other record was verified.
type GenerationReference struct {
	RecoveryEpoch         string `json:"recovery_epoch"`
	Sequence              string `json:"sequence"`
	AcceptedPayloadSHA256 string `json:"accepted_payload_sha256"`
}

// Requirements is a closed, non-secret set of context safeguards. Capability
// measurement and durable enforcement remain caller responsibilities.
type Requirements struct {
	AuditRequired    bool   `json:"audit_required"`
	ApprovalRequired bool   `json:"approval_required"`
	Custody          string `json:"custody"`
	OfflineAllowed   bool   `json:"offline_allowed"`
	RecoveryRequired bool   `json:"recovery_required"`
}

// CapabilityEvidence is a signed requirement declaration, never proof that a
// control has been measured or satisfied. The protected admission layer
// must independently measure it and compare it with this requirement.
type CapabilityEvidence struct {
	ControlID        string    `json:"control_id"`
	ImplementationID string    `json:"implementation_id"`
	Version          string    `json:"version"`
	Platform         string    `json:"platform"`
	ExpiresAt        time.Time `json:"expires_at"`
}

// Claims is the strict, verified context payload. All fields are authenticated
// only after Verify succeeds; callers must not treat decoded Claims as proof.
type Claims struct {
	Version              int                  `json:"version"`
	ContextID            string               `json:"context_id"`
	ProfileID            string               `json:"profile_id"`
	ProfileRevision      string               `json:"profile_revision"`
	Tenant               string               `json:"tenant"`
	Issuer               string               `json:"issuer"`
	Audience             string               `json:"audience"`
	PolicyGraphSHA256    string               `json:"policy_graph_sha256"`
	PolicyRevision       string               `json:"policy_revision"`
	ContextGeneration    ContextGeneration    `json:"context_generation"`
	EnrollmentGeneration GenerationReference  `json:"enrollment_generation"`
	AuthorityGeneration  GenerationReference  `json:"authority_generation"`
	NotBefore            time.Time            `json:"not_before"`
	ExpiresAt            time.Time            `json:"expires_at"`
	RevocationFreshUntil time.Time            `json:"revocation_fresh_until"`
	Requirements         Requirements         `json:"requirements"`
	Capabilities         []CapabilityEvidence `json:"capabilities"`
}

// AcceptedContext records one protected accepted context component.
type AcceptedContext struct {
	Reference       GenerationReference
	PreviousPayload string
}

// AggregateSnapshot is loaded atomically by the protected store.
// Fence changes whenever any component relevant to admission changes.
type AggregateSnapshot struct {
	Context         AcceptedContext
	Authority       GenerationReference
	Enrollment      GenerationReference
	TrustSetEpoch   string
	RevocationEpoch string
	Fence           string
}

// Input consists exclusively of trusted boundary inputs. The protected compiler
// must derive it from protected enrollment/state. Runtime capability measurement
// is deliberately outside this pure verification input.
type Input struct {
	Trust    Trust
	Snapshot AggregateSnapshot
	// GraphDigest is computed by the composition root from exact
	// client-local graph inputs. securitycontext intentionally does not import
	// configuration, so configuration cannot depend on this verifier.
	GraphDigest string
	Now         time.Time
}

// VerifiedEnvelope is an opaque cryptographically verified envelope. It is
// explicitly unadmitted: caller-supplied Input can produce it, so it is never
// authorization proof. The protected compiler/admitter, not this package,
// may convert verified private data into an admission decision.
type VerifiedEnvelope struct {
	claims      Claims
	payloadHash string
	seal        *verifiedSeal
}

type verifiedSeal struct{}

// Verified reports whether this value originated from successful Verify. It
// does not authorize a workload, provider read, or protected effect.
func (e VerifiedEnvelope) Verified() bool { return e.seal != nil }

// Summary exposes only narrow verified routing metadata. It intentionally
// omits graph, payload, provenance, and aggregate-fence digests.
type Summary struct {
	ContextID       string
	ProfileID       string
	ProfileRevision string
}

// Summary returns a safe immutable projection. Invalid envelopes return an
// empty summary and must not be used for authorization.
func (e VerifiedEnvelope) Summary() Summary {
	if !e.Verified() {
		return Summary{}
	}
	return Summary{ContextID: e.claims.ContextID, ProfileID: e.claims.ProfileID, ProfileRevision: e.claims.ProfileRevision}
}

// VerifyInputProvider is the exact dependency the protected compiler satisfies.
// It must atomically obtain protected binding/trust/snapshot inputs, then invoke
// Verify before provider reads or protected effects.
type VerifyInputProvider interface {
	ContextInput(context.Context) (Input, error)
}

// GraphBindingProvider is the separate dependency the composition root
// must satisfy. It supplies the immutable client-local graph binding to Input;
// it does not expose raw configuration or provenance to this package or a
// hosted service.
type GraphBindingProvider interface {
	GraphDigest(context.Context) (string, error)
}

// AggregateStore is the admission dependency for atomic protected state.
// This pure verifier deliberately does not provide an in-memory substitute.
type AggregateStore interface {
	Snapshot(context.Context, Binding) (AggregateSnapshot, error)
}
