package securitycontext

import (
	"bytes"
	"crypto"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	jose "github.com/go-jose/go-jose/v4"
	"github.com/maldous/CircularQuay/internal/fault"
	"github.com/maldous/CircularQuay/internal/jsondocument"
)

const (
	contextType    = "cq-security-context+jws"
	contextCTY     = "application/cq-security-context+json;v=1"
	contextPurpose = "security_context_v1"
)

type envelope struct {
	Payload    string      `json:"payload"`
	Signatures []signature `json:"signatures"`
}

type signature struct {
	Protected string `json:"protected"`
	Signature string `json:"signature"`
}

type header struct {
	Type    string `json:"typ"`
	Content string `json:"cty"`
	Alg     string `json:"alg"`
	KID     string `json:"kid"`
	Purpose string `json:"purpose"`
}

// Verify validates one bounded general JWS against caller-supplied protected
// enrollment trust and aggregate snapshot. It performs no I/O or trust lookup.
// Verify returns only an unadmitted cryptographic verification result.
func Verify(raw []byte, input Input) (VerifiedEnvelope, error) {
	if !validInput(input) || len(raw) == 0 || len(raw) > MaxEnvelopeBytes || !utf8.Valid(raw) {
		return VerifiedEnvelope{}, fault.New(fault.Invalid, "security_context_input")
	}
	env, err := parseEnvelope(raw)
	if err != nil {
		return VerifiedEnvelope{}, err
	}
	payload, err := decodeCanonical(env.Payload, maxPayloadBytes)
	if err != nil {
		return VerifiedEnvelope{}, err
	}
	if err := verifySignatures(env, payload, input.Trust); err != nil {
		return VerifiedEnvelope{}, err
	}
	claims, err := parseClaims(payload)
	if err != nil {
		return VerifiedEnvelope{}, err
	}
	hash := payloadHash(payload)
	if err := validateClaims(claims, hash, input); err != nil {
		return VerifiedEnvelope{}, err
	}
	return VerifiedEnvelope{claims: cloneClaims(claims), payloadHash: hash, seal: &verifiedSeal{}}, nil
}

func parseEnvelope(raw []byte) (envelope, error) {
	var env envelope
	if jsondocument.Decode(raw, &env) != nil || len(env.Signatures) == 0 || len(env.Signatures) > maxSignatures {
		return envelope{}, fault.New(fault.Invalid, "security_context_envelope")
	}
	return env, nil
}

func verifySignatures(env envelope, payload []byte, trust Trust) error {
	if len(env.Signatures) < trust.RequiredSigners || len(env.Signatures) > maxSignatures {
		return fault.New(fault.Trust, "security_context_threshold")
	}
	seen := make(map[string]struct{}, len(env.Signatures))
	for _, sig := range env.Signatures {
		if err := verifySignature(sig, env.Payload, payload, trust, seen); err != nil {
			return err
		}
	}
	if len(seen) < trust.RequiredSigners {
		return fault.New(fault.Trust, "security_context_threshold")
	}
	return nil
}

func verifySignature(sig signature, encodedPayload string, payload []byte, trust Trust, seen map[string]struct{}) error {
	protected, err := decodeCanonical(sig.Protected, 4096)
	if err != nil {
		return err
	}
	var h header
	if jsondocument.Decode(protected, &h) != nil || !validHeader(h) {
		return fault.New(fault.Invalid, "security_context_header")
	}
	key, found := trustedKey(trust, h)
	if !found {
		return fault.New(fault.Trust, "security_context_signer")
	}
	identity, ok := keyMaterialIdentity(key)
	if !ok {
		return fault.New(fault.Trust, "security_context_signer")
	}
	if _, duplicate := seen[identity]; duplicate {
		return fault.New(fault.Trust, "security_context_signer")
	}
	if _, err = decodeCanonical(sig.Signature, 4096); err != nil {
		return err
	}
	verified, err := verifySingleton(encodedPayload, sig, key)
	if err != nil || !bytes.Equal(verified, payload) {
		return fault.New(fault.Trust, "security_context_signature")
	}
	seen[identity] = struct{}{}
	return nil
}

func verifySingleton(payload string, sig signature, key TrustedKey) ([]byte, error) {
	one, err := json.Marshal(envelope{Payload: payload, Signatures: []signature{sig}})
	if err != nil {
		return nil, fault.New(fault.Apply, "security_context_singleton")
	}
	jws, err := jose.ParseSignedJSON(string(one), []jose.SignatureAlgorithm{key.Algorithm})
	if err != nil {
		return nil, fault.New(fault.Trust, "security_context_signature")
	}
	value, err := jws.Verify(key.Key)
	if err != nil {
		return nil, fault.New(fault.Trust, "security_context_signature")
	}
	return value, nil
}

func trustedKey(trust Trust, h header) (TrustedKey, bool) {
	for _, key := range trust.Keys {
		if key.KID == h.KID && key.Algorithm == jose.SignatureAlgorithm(h.Alg) {
			return key, true
		}
	}
	return TrustedKey{}, false
}

func validHeader(h header) bool {
	return h.Type == contextType && h.Content == contextCTY && h.Purpose == contextPurpose && validText(h.KID, 256) && validAlgorithm(h.Alg)
}

func parseClaims(payload []byte) (Claims, error) {
	var claims Claims
	if jsondocument.Decode(payload, &claims) != nil {
		return Claims{}, fault.New(fault.Invalid, "security_context_claims")
	}
	return claims, nil
}

func validateClaims(c Claims, hash string, input Input) error {
	if !matchingIdentity(c, input.Trust) || c.PolicyGraphSHA256 != input.GraphDigest || !validClaims(c) {
		return fault.New(fault.Trust, "security_context_claims")
	}
	if !c.NotBefore.Before(c.ExpiresAt) || !input.Now.Before(c.ExpiresAt) || input.Now.Before(c.NotBefore) || !input.Now.Before(c.RevocationFreshUntil) {
		return fault.New(fault.Expired, "security_context_freshness")
	}
	return validateSuccessor(c, hash, input.Snapshot)
}

func validateSuccessor(c Claims, hash string, snapshot AggregateSnapshot) error {
	if !equalReference(c.AuthorityGeneration, snapshot.Authority) || !equalReference(c.EnrollmentGeneration, snapshot.Enrollment) {
		return fault.New(fault.Denied, "security_context_generation")
	}
	if equalContext(c.ContextGeneration, snapshot.Context, hash) {
		return nil
	}
	if successorContext(c.ContextGeneration, snapshot.Context.Reference) {
		return nil
	}
	return fault.New(fault.Denied, "security_context_generation")
}

func equalContext(g ContextGeneration, accepted AcceptedContext, hash string) bool {
	return g.RecoveryEpoch == accepted.Reference.RecoveryEpoch && g.Sequence == accepted.Reference.Sequence && g.PreviousAcceptedPayloadSHA256 == accepted.PreviousPayload && hash == accepted.Reference.AcceptedPayloadSHA256
}

func successorContext(g ContextGeneration, accepted GenerationReference) bool {
	_, ok := decimal(g.RecoveryEpoch)
	if !ok || g.PreviousAcceptedPayloadSHA256 != accepted.AcceptedPayloadSHA256 || g.RecoveryEpoch != accepted.RecoveryEpoch {
		return false
	}
	sequence, ok := decimal(g.Sequence)
	if !ok || sequence == 0 || accepted.Sequence == "" {
		return false
	}
	current, ok := decimal(accepted.Sequence)
	return ok && current != ^uint64(0) && sequence == current+1
}

func equalReference(first, second GenerationReference) bool {
	return first.RecoveryEpoch == second.RecoveryEpoch && first.Sequence == second.Sequence && first.AcceptedPayloadSHA256 == second.AcceptedPayloadSHA256 && validReference(first)
}

func validClaims(c Claims) bool {
	return validClaimIdentity(c) && validClaimBinding(c) && validClaimGenerations(c) && validClaimRequirements(c)
}

func validClaimIdentity(c Claims) bool {
	return c.Version == 1 && validText(c.ContextID, 256) && validText(c.ProfileID, 256) && validText(c.ProfileRevision, 256) && validText(c.Tenant, 256)
}

func validClaimBinding(c Claims) bool {
	return validText(c.Issuer, 1024) && validText(c.Audience, 1024) && validSHA256(c.PolicyGraphSHA256) && validText(c.PolicyRevision, 256)
}

func validClaimGenerations(c Claims) bool {
	return validContextGeneration(c.ContextGeneration) && validReference(c.AuthorityGeneration) && validReference(c.EnrollmentGeneration)
}

func validClaimRequirements(c Claims) bool {
	return validClaimTimes(c) && validRequirements(c.Requirements) && validCapabilities(c.Capabilities) && capabilitiesCover(c.Capabilities, c.ExpiresAt)
}

func validClaimTimes(c Claims) bool {
	return validUTCTime(c.NotBefore) && validUTCTime(c.ExpiresAt) && validUTCTime(c.RevocationFreshUntil)
}

func validInput(input Input) bool {
	return input.Now.UTC().Equal(input.Now) && validSHA256(input.GraphDigest) && validTrust(input.Trust) && validSnapshot(input.Snapshot)
}

func validTrust(t Trust) bool {
	if !validTrustMetadata(t) || !validTrustKeyCount(t) {
		return false
	}
	return uniqueTrustedKeys(t.Keys)
}

func validTrustMetadata(t Trust) bool {
	return validBinding(t.Binding) && validText(t.Issuer, 1024) && validText(t.ProfileID, 256) && validText(t.ProfileRevision, 256)
}

func validTrustKeyCount(t Trust) bool {
	return t.RequiredSigners >= 1 && t.RequiredSigners <= maxSignatures && len(t.Keys) >= t.RequiredSigners && len(t.Keys) <= maxSignatures
}

func uniqueTrustedKeys(keys []TrustedKey) bool {
	seenKID, seenIdentity := map[string]struct{}{}, map[string]struct{}{}
	for _, key := range keys {
		identity, ok := keyMaterialIdentity(key)
		if !ok || !validText(key.KID, 256) {
			return false
		}
		if _, exists := seenKID[key.KID]; exists {
			return false
		}
		if _, exists := seenIdentity[identity]; exists {
			return false
		}
		seenKID[key.KID], seenIdentity[identity] = struct{}{}, struct{}{}
	}
	return true
}

func validSnapshot(s AggregateSnapshot) bool {
	return validAccepted(s.Context) && validReference(s.Authority) && validReference(s.Enrollment) && validDecimal(s.TrustSetEpoch) && validDecimal(s.RevocationEpoch) && validDecimal(s.Fence)
}

func validAccepted(a AcceptedContext) bool {
	return validReference(a.Reference) && validSHA256(a.PreviousPayload)
}
func validBinding(b Binding) bool {
	return validText(b.Tenant, 256) && validText(b.ContextID, 256) && validText(b.TrustDomain, 256) && validText(b.Audience, 1024)
}
func validRequirements(r Requirements) bool { return validText(r.Custody, 256) }

func validCapabilities(values []CapabilityEvidence) bool {
	if values == nil || len(values) > 32 {
		return false
	}
	seen := map[string]struct{}{}
	for _, value := range values {
		if !validCapability(value) {
			return false
		}
		if _, exists := seen[value.ControlID]; exists {
			return false
		}
		seen[value.ControlID] = struct{}{}
	}
	return true
}

func validCapability(value CapabilityEvidence) bool {
	return validText(value.ControlID, 256) && validText(value.ImplementationID, 256) && validText(value.Version, 256) && validText(value.Platform, 256) && validUTCTime(value.ExpiresAt)
}

func capabilitiesCover(values []CapabilityEvidence, contextExpiry time.Time) bool {
	for _, value := range values {
		if value.ExpiresAt.Before(contextExpiry) {
			return false
		}
	}
	return true
}

func matchingIdentity(c Claims, trust Trust) bool {
	return c.ContextID == trust.Binding.ContextID && c.Tenant == trust.Binding.Tenant && c.Issuer == trust.Issuer && c.Audience == trust.Binding.Audience && c.ProfileID == trust.ProfileID && c.ProfileRevision == trust.ProfileRevision
}
func validAlgorithm(value string) bool {
	return value == string(jose.ES256) || value == string(jose.ES384) || value == string(jose.EdDSA)
}
func keyMaterialIdentity(key TrustedKey) (string, bool) {
	if !validAlgorithm(string(key.Algorithm)) {
		return "", false
	}
	encoded, ok := canonicalPublicKey(key)
	if !ok {
		return "", false
	}
	return payloadHash(append([]byte(key.Algorithm+":"), encoded...)), true
}
func canonicalPublicKey(key TrustedKey) ([]byte, bool) {
	switch key.Algorithm {
	case jose.ES256:
		return canonicalECDSAKey(key.Key, elliptic.P256())
	case jose.ES384:
		return canonicalECDSAKey(key.Key, elliptic.P384())
	case jose.EdDSA:
		value, ok := key.Key.(ed25519.PublicKey)
		if !ok || len(value) != ed25519.PublicKeySize {
			return nil, false
		}
		return append([]byte(nil), value...), true
	default:
		return nil, false
	}
}
func canonicalECDSAKey(value crypto.PublicKey, curve elliptic.Curve) ([]byte, bool) {
	key, ok := value.(*ecdsa.PublicKey)
	if !ok || key == nil || key.Curve != curve || !(&jose.JSONWebKey{Key: key}).Valid() {
		return nil, false
	}
	// Structural validation rejects missing components before Bytes validates the point.
	encoded, err := key.Bytes()
	if err != nil || len(encoded) == 0 {
		return nil, false
	}
	return encoded, true
}
func validContextGeneration(g ContextGeneration) bool {
	return validDecimal(g.RecoveryEpoch) && validDecimal(g.Sequence) && decimalNonzero(g.Sequence) && validSHA256(g.PreviousAcceptedPayloadSHA256)
}
func validReference(g GenerationReference) bool {
	return validDecimal(g.RecoveryEpoch) && validDecimal(g.Sequence) && validSHA256(g.AcceptedPayloadSHA256)
}
func validDecimal(value string) bool   { _, ok := decimal(value); return ok }
func decimalNonzero(value string) bool { parsed, ok := decimal(value); return ok && parsed != 0 }
func decimal(value string) (uint64, bool) {
	if value == "" || len(value) > 20 || strings.Trim(value, "0123456789") != "" || len(value) > 1 && value[0] == '0' {
		return 0, false
	}
	parsed, err := strconv.ParseUint(value, 10, 64)
	return parsed, err == nil
}
func validSHA256(value string) bool {
	if len(value) != 64 {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil && strings.ToLower(value) == value
}
func validText(value string, limit int) bool {
	return value != "" && len(value) <= limit && utf8.ValidString(value) && !strings.ContainsAny(value, "\x00\r\n")
}
func validUTCTime(value time.Time) bool {
	return !value.IsZero() && value.Location() == time.UTC
}
func decodeCanonical(value string, limit int) ([]byte, error) {
	if value == "" || len(value) > limit*2 || strings.ContainsAny(value, "=\r\n") {
		return nil, fault.New(fault.Invalid, "security_context_encoding")
	}
	decoded, err := base64.RawURLEncoding.Strict().DecodeString(value)
	if err != nil || len(decoded) > limit || base64.RawURLEncoding.EncodeToString(decoded) != value {
		return nil, fault.New(fault.Invalid, "security_context_encoding")
	}
	return decoded, nil
}
func payloadHash(value []byte) string {
	digest := sha256.Sum256(value)
	return hex.EncodeToString(digest[:])
}
func cloneClaims(value Claims) Claims {
	copied := value
	copied.Capabilities = append([]CapabilityEvidence(nil), value.Capabilities...)
	return copied
}
