package keyservice

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/go-jose/go-jose/v4"
	"github.com/maldous/CircularQuay/internal/fault"
	"github.com/maldous/CircularQuay/internal/jsondocument"
)

const (
	softwareProvider  = "software-jwe"
	softwareAlgorithm = "A256KW+A256GCM"
	softwareKeyBytes  = 32
	softwareCipherMax = 512 << 10
	softwareHeaderKV  = "cquay_key_version"
	softwareHeaderCA  = "cquay_configuration_attestation"
)

// SoftwareKEK is one resolver-owned software key-encryption key. Ownership of
// Key transfers to SoftwareKMS, which clears it best-effort after one request.
// A software KEK is neither a hardware key nor non-exportable.
type SoftwareKEK struct {
	KeyID      string
	KeyVersion string
	Key        []byte
}

// SoftwareKEKResolver returns the distinct runtime KEK selected for exactly
// one persisted association. It must not discover credentials from the
// environment or a host profile. Implementations keep KEK storage separate
// from application data; this adapter cannot make a shared runtime compromise
// harmless.
type SoftwareKEKResolver interface {
	ResolveSoftwareKEK(context.Context, KMSAssociation) (SoftwareKEK, error)
}

// SoftwareKMS implements ProviderKMS with the fixed software JWE profile:
// A256KW key wrapping and A256GCM content encryption. The maintained go-jose
// implementation supplies key generation and nonce generation. This adapter
// does not provide HSM, non-exportable-key, hosted-service, or cross-product
// JWE interoperability guarantees.
type SoftwareKMS struct {
	resolver SoftwareKEKResolver
}

// NewSoftwareKMS creates an adapter with an explicit KEK resolver only.
func NewSoftwareKMS(resolver SoftwareKEKResolver) (*SoftwareKMS, error) {
	if resolver == nil {
		return nil, fault.New(fault.Invalid, "software_kek_resolver")
	}
	return &SoftwareKMS{resolver: resolver}, nil
}

// Wrap encrypts short data-key material with an association-bound JWE.
func (s *SoftwareKMS) Wrap(ctx context.Context, request WrapRequest) (WrapResult, error) {
	if err := validateSoftwareWrapRequest(ctx, request); err != nil {
		return WrapResult{}, err
	}
	key, err := s.resolve(ctx, request.Association)
	if err != nil {
		return WrapResult{}, err
	}
	defer clearBytes(key)
	options := softwareOptions(request.Association)
	encrypter, err := jose.NewEncrypter(jose.A256GCM, jose.Recipient{Algorithm: jose.A256KW, Key: key, KeyID: request.Association.KeyID}, options)
	if err != nil {
		return WrapResult{}, fault.New(fault.Trust, "software_encrypter")
	}
	envelope, err := encrypter.EncryptWithAuthData(request.Plaintext, softwareAAD(request.Context))
	if err != nil {
		return WrapResult{}, fault.New(fault.Trust, "software_encrypt")
	}
	if err := softwareContextError(ctx); err != nil {
		return WrapResult{}, err
	}
	ciphertext := []byte(envelope.FullSerialize())
	if err := validateSoftwareJWE(ciphertext, request.Association, request.Context); err != nil {
		clearBytes(ciphertext)
		return WrapResult{}, err
	}
	return WrapResult{Ciphertext: ciphertext, Receipt: softwareReceipt(request.Association, request.Context)}, nil
}

// Unwrap decrypts only the fixed, association-bound JWE profile.
func (s *SoftwareKMS) Unwrap(ctx context.Context, request UnwrapRequest) (UnwrapResult, error) {
	if err := validateSoftwareUnwrapRequest(ctx, request); err != nil {
		return UnwrapResult{}, err
	}
	if err := validateSoftwareJWE(request.Ciphertext, request.Association, request.Context); err != nil {
		return UnwrapResult{}, err
	}
	key, err := s.resolve(ctx, request.Association)
	if err != nil {
		return UnwrapResult{}, err
	}
	defer clearBytes(key)
	envelope, err := jose.ParseEncryptedJSON(string(request.Ciphertext), []jose.KeyAlgorithm{jose.A256KW}, []jose.ContentEncryption{jose.A256GCM})
	if err != nil {
		return UnwrapResult{}, fault.New(fault.Trust, "software_jwe")
	}
	plaintext, err := envelope.Decrypt(key)
	if err != nil {
		clearBytes(plaintext)
		return UnwrapResult{}, fault.New(fault.Trust, "software_decrypt")
	}
	if err := softwareContextError(ctx); err != nil {
		clearBytes(plaintext)
		return UnwrapResult{}, err
	}
	if !validSoftwareDEK(plaintext) {
		clearBytes(plaintext)
		return UnwrapResult{}, fault.New(fault.Trust, "software_plaintext")
	}
	return UnwrapResult{Plaintext: plaintext, Receipt: softwareReceipt(request.Association, request.Context)}, nil
}

func (s *SoftwareKMS) resolve(ctx context.Context, association KMSAssociation) ([]byte, error) {
	if err := softwareContextError(ctx); err != nil {
		return nil, err
	}
	resolved, err := s.resolver.ResolveSoftwareKEK(ctx, association)
	if err != nil {
		clearBytes(resolved.Key)
		if contextErr := softwareContextError(ctx); contextErr != nil {
			return nil, contextErr
		}
		return nil, softwareResolverFailure(err)
	}
	if err := softwareContextError(ctx); err != nil {
		clearBytes(resolved.Key)
		return nil, err
	}
	if resolved.KeyID != association.KeyID || resolved.KeyVersion != association.KeyVersion || len(resolved.Key) != softwareKeyBytes {
		clearBytes(resolved.Key)
		return nil, fault.New(fault.Trust, "software_kek")
	}
	return resolved.Key, nil
}

func validateSoftwareWrapRequest(ctx context.Context, request WrapRequest) error {
	if err := validateSoftwareBinding(ctx, request.Association, request.Context); err != nil {
		return err
	}
	if !validSoftwareDEK(request.Plaintext) {
		return fault.New(fault.Invalid, "software_plaintext")
	}
	return nil
}

func validateSoftwareUnwrapRequest(ctx context.Context, request UnwrapRequest) error {
	if err := validateSoftwareBinding(ctx, request.Association, request.Context); err != nil {
		return err
	}
	if len(request.Ciphertext) == 0 || len(request.Ciphertext) > softwareCipherMax || !utf8.Valid(request.Ciphertext) {
		return fault.New(fault.Invalid, "software_ciphertext")
	}
	return nil
}

func validateSoftwareBinding(ctx context.Context, association KMSAssociation, binding AssociationContext) error {
	if err := softwareContextError(ctx); err != nil {
		return err
	}
	if !validSoftwareAssociation(association) || !validSoftwareContext(binding) || binding.Purpose != association.Purpose {
		return fault.New(fault.Invalid, "software_binding")
	}
	return nil
}

func validSoftwareAssociation(association KMSAssociation) bool {
	return association.Provider == softwareProvider && association.Algorithm == softwareAlgorithm && validKMS(association) && validSoftwareHeaderMetadata(association)
}

func validSoftwareHeaderMetadata(association KMSAssociation) bool {
	return validIdentifier(association.KeyID, 1024) && validIdentifier(association.KeyVersion, maxIdentifierBytes) && validIdentifier(association.ConfigurationAttestation, 1024)
}

func validSoftwareContext(binding AssociationContext) bool {
	return validIdentifier(binding.Tenant, maxIdentifierBytes) &&
		validIdentifier(binding.Profile, maxIdentifierBytes) &&
		validIdentifier(binding.Application, maxIdentifierBytes) &&
		validIdentifier(binding.Principal, maxIdentifierBytes) &&
		validIdentifier(binding.Generation, maxIdentifierBytes) &&
		validIdentifier(binding.Purpose, maxIdentifierBytes) &&
		validIdentifier(binding.ArtifactID, maxIdentifierBytes) &&
		validIdentifier(binding.ArtifactVersion, maxIdentifierBytes)
}

func validSoftwareDEK(value []byte) bool { return len(value) == softwareKeyBytes }

func softwareAAD(binding AssociationContext) []byte {
	fields := []string{binding.Tenant, binding.Profile, binding.Application, binding.Principal, binding.Generation, binding.Purpose, binding.ArtifactID, binding.ArtifactVersion}
	var value strings.Builder
	value.WriteString("cquay-keyservice-software-jwe-v1")
	for _, field := range fields {
		value.WriteString("|")
		value.WriteString(strconv.Itoa(len(field)))
		value.WriteString(":")
		value.WriteString(field)
	}
	return []byte(value.String())
}

func softwareReceipt(association KMSAssociation, binding AssociationContext) KMSReceipt {
	return KMSReceipt{Provider: association.Provider, KeyID: association.KeyID, KeyVersion: association.KeyVersion, Algorithm: association.Algorithm, ConfigurationAttestation: association.ConfigurationAttestation, Context: binding}
}

func softwareContextError(ctx context.Context) error {
	if ctx == nil || ctx.Err() != nil {
		return fault.New(fault.Denied, "software_cancelled")
	}
	return nil
}

func softwareResolverFailure(error) error {
	return fault.New(fault.Trust, "software_kek")
}

func softwareOptions(association KMSAssociation) *jose.EncrypterOptions {
	return (&jose.EncrypterOptions{}).
		WithHeader(jose.HeaderKey(softwareHeaderKV), association.KeyVersion).
		WithHeader(jose.HeaderKey(softwareHeaderCA), association.ConfigurationAttestation)
}

type softwareEnvelope struct {
	Protected    string `json:"protected"`
	EncryptedKey string `json:"encrypted_key"`
	IV           string `json:"iv"`
	Ciphertext   string `json:"ciphertext"`
	Tag          string `json:"tag"`
	AAD          string `json:"aad"`
}

func validateSoftwareJWE(ciphertext []byte, association KMSAssociation, binding AssociationContext) error {
	if len(ciphertext) == 0 || len(ciphertext) > softwareCipherMax || !utf8.Valid(ciphertext) {
		return fault.New(fault.Invalid, "software_jwe")
	}
	fields, err := strictSoftwareObject(ciphertext)
	if err != nil || !exactFields(fields, "protected", "encrypted_key", "iv", "ciphertext", "tag", "aad") {
		return fault.New(fault.Invalid, "software_jwe")
	}
	envelope, ok := softwareEnvelopeFrom(fields)
	if !ok || !validSoftwareParts(envelope) {
		return fault.New(fault.Invalid, "software_jwe")
	}
	if !softwareHeaderMatches(envelope.Protected, association) || !softwareAADMatches(envelope.AAD, binding) {
		return fault.New(fault.Trust, "software_jwe")
	}
	return nil
}

func softwareEnvelopeFrom(fields map[string]json.RawMessage) (softwareEnvelope, bool) {
	var envelope softwareEnvelope
	values := []*string{&envelope.Protected, &envelope.EncryptedKey, &envelope.IV, &envelope.Ciphertext, &envelope.Tag, &envelope.AAD}
	for index, name := range []string{"protected", "encrypted_key", "iv", "ciphertext", "tag", "aad"} {
		if err := json.Unmarshal(fields[name], values[index]); err != nil || !validJSONText(*values[index], softwareCipherMax) {
			return softwareEnvelope{}, false
		}
	}
	return envelope, true
}

func validSoftwareParts(envelope softwareEnvelope) bool {
	return validSoftwareEncryptionParts(envelope) && validSoftwareCipherParts(envelope)
}

func validSoftwareEncryptionParts(envelope softwareEnvelope) bool {
	return validBase64Part(envelope.Protected, 1, maxKeyReference) &&
		validBase64Part(envelope.EncryptedKey, 40, 40) &&
		validBase64Part(envelope.IV, 12, 12)
}

func validSoftwareCipherParts(envelope softwareEnvelope) bool {
	return validBase64Part(envelope.Ciphertext, softwareKeyBytes, softwareKeyBytes) &&
		validBase64Part(envelope.Tag, 16, 16) &&
		validBase64Part(envelope.AAD, 1, maxKeyReference)
}

func validBase64Part(value string, minimum, maximum int) bool {
	decoded, err := base64.RawURLEncoding.Strict().DecodeString(value)
	return err == nil && base64.RawURLEncoding.EncodeToString(decoded) == value && len(decoded) >= minimum && len(decoded) <= maximum
}

func softwareHeaderMatches(encoded string, association KMSAssociation) bool {
	protected, err := base64.RawURLEncoding.Strict().DecodeString(encoded)
	if err != nil {
		return false
	}
	fields, err := strictSoftwareObject(protected)
	if err != nil || !exactFields(fields, "alg", "enc", "kid", softwareHeaderKV, softwareHeaderCA) {
		return false
	}
	values, valid := decodedSoftwareHeaderFields(fields)
	return valid && softwareHeaderValuesMatch(values, association)
}

func decodedSoftwareHeaderFields(fields map[string]json.RawMessage) (map[string]string, bool) {
	values := make(map[string]string, len(fields))
	for name, raw := range fields {
		var value string
		if err := json.Unmarshal(raw, &value); err != nil || !validJSONText(value, maxKeyReference) {
			return nil, false
		}
		values[name] = value
	}
	return values, true
}

func softwareHeaderValuesMatch(values map[string]string, association KMSAssociation) bool {
	return values["alg"] == string(jose.A256KW) &&
		values["enc"] == string(jose.A256GCM) &&
		values["kid"] == association.KeyID &&
		values[softwareHeaderKV] == association.KeyVersion &&
		values[softwareHeaderCA] == association.ConfigurationAttestation
}

func softwareAADMatches(encoded string, binding AssociationContext) bool {
	value, err := base64.RawURLEncoding.Strict().DecodeString(encoded)
	return err == nil && base64.RawURLEncoding.EncodeToString(value) == encoded && bytes.Equal(value, softwareAAD(binding))
}

func strictSoftwareObject(input []byte) (map[string]json.RawMessage, error) {
	var fields map[string]json.RawMessage
	if jsondocument.Decode(input, &fields) != nil || fields == nil {
		return nil, fault.New(fault.Invalid, "software_jwe")
	}
	return fields, nil
}

func exactFields(fields map[string]json.RawMessage, required ...string) bool {
	if len(fields) != len(required) {
		return false
	}
	for _, name := range required {
		if _, found := fields[name]; !found {
			return false
		}
	}
	return true
}

func validJSONText(value string, maximum int) bool {
	return value != "" && len(value) <= maximum && utf8.ValidString(value) && !strings.ContainsAny(value, "\x00\r\n")
}
