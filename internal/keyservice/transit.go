package keyservice

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/maldous/CircularQuay/internal/fault"
	"github.com/maldous/CircularQuay/internal/jsondocument"
	"github.com/maldous/CircularQuay/internal/transport"
)

const (
	transitProviderOpenBao = "openbao-transit"
	transitProviderVault   = "vault-transit"
	transitAlgorithm       = "aes256-gcm96"
	maximumTransitMaterial = 64 << 10
	maximumTransitToken    = 64 << 10
	maximumTransitOpaque   = 4096

	transitForbiddenControlCharacters = "\x00\r\n"
)

// CredentialReader resolves an adapter-owned opaque credential reference.
// Implementations must not consult environment variables or shell profiles.
type CredentialReader interface {
	Token(context.Context, string) (string, error)
}

// TransitConfiguration is protected adapter configuration for one safe KMS
// association. It is never placed in enrollment records, provider requests,
// receipts, audit output, or application responses. Its version constraints are
// explicit enrollment expectations; ConfigurationAttestation remains an opaque
// trusted record marker, not a cryptographic assertion from Transit.
type TransitConfiguration struct {
	Endpoint             string
	Mount                string
	Namespace            string
	CredentialReference  string
	MinDecryptionVersion string
	MinEncryptionVersion string
}

// TransitConfigurationResolver supplies protected Transit connection metadata
// for an exact persisted association. It must not infer a provider, endpoint,
// or credential from ambient configuration.
type TransitConfigurationResolver interface {
	TransitConfiguration(context.Context, KMSAssociation) (TransitConfiguration, error)
}

// ProtectedCredentialReader resolves a credential reference as a protected
// absolute token file. Deployments with an opaque credential store inject their
// own reader instead; Transit never discovers credentials itself.
type ProtectedCredentialReader struct{}

// Token reads a protected Transit token without retaining it.
func (ProtectedCredentialReader) Token(ctx context.Context, reference string) (string, error) {
	if err := transitContextError(ctx); err != nil {
		return "", err
	}
	return transport.ReadCredentialFile(reference)
}

// Transit implements ProviderKMS through the documented OpenBao and Vault
// Transit encrypt/decrypt API. It requires only read access to the exact key
// endpoint and update access to that key's encrypt/decrypt endpoints. It does
// not create keys, infer key attributes, cache key material, or select a
// provider from ambient configuration. Transit preserves availability only from
// the transport boundary's narrow, typed connection or request-timeout proof.
type Transit struct {
	client         transport.Client
	configurations TransitConfigurationResolver
	credentials    CredentialReader
}

// NewTransit accepts only explicit, protected collaborators. The transport
// client enforces HTTPS endpoint validation, bounded payloads, and redirect
// denial. Configuration and credentials remain outside public enrollment data.
func NewTransit(client transport.Client, configurations TransitConfigurationResolver, credentials CredentialReader) (*Transit, error) {
	if configurations == nil || credentials == nil {
		return nil, fault.New(fault.Invalid, "transit_dependencies")
	}
	return &Transit{client: client, configurations: configurations, credentials: credentials}, nil
}

// Wrap encrypts one short data key under the exact enrolled Transit key.
func (t *Transit) Wrap(ctx context.Context, request WrapRequest) (WrapResult, error) {
	if err := transitWrapRequest(request); err != nil {
		return WrapResult{}, err
	}
	configuration, token, _, err := t.authorize(ctx, request.Association, request.Context, transitActionWrap)
	if err != nil {
		return WrapResult{}, err
	}
	endpoint, err := transitURL(configuration, "encrypt", request.Association.KeyID)
	if err != nil {
		return WrapResult{}, err
	}
	keyVersion, ok := transitVersionNumber(request.Association.KeyVersion)
	if !ok {
		return WrapResult{}, fault.New(fault.Trust, "transit_key_configuration")
	}
	body := transitEncryptRequest{
		Plaintext:  base64.StdEncoding.EncodeToString(request.Plaintext),
		Context:    base64.StdEncoding.EncodeToString(transitDerivedContext(request.Context)),
		KeyVersion: keyVersion,
	}
	var response transitEncryptResponse
	if err := t.client.DoJSON(ctx, http.MethodPost, endpoint, transitHeaders(token, configuration.Namespace), body, &response); err != nil {
		return WrapResult{}, transitProviderFailure(ctx, err)
	}
	if err := transitContextError(ctx); err != nil {
		return WrapResult{}, err
	}
	version, err := transitCiphertextVersion(response.Data.Ciphertext)
	if err != nil || response.Data.KeyVersion.String() != request.Association.KeyVersion || version != request.Association.KeyVersion {
		return WrapResult{}, fault.New(fault.Trust, "transit_encrypt_response")
	}
	return WrapResult{
		Ciphertext: []byte(response.Data.Ciphertext),
		Receipt:    transitReceipt(request.Association, request.Context),
	}, nil
}

// Unwrap decrypts one exact-version Transit ciphertext under the enrolled key.
func (t *Transit) Unwrap(ctx context.Context, request UnwrapRequest) (UnwrapResult, error) {
	if err := transitUnwrapRequest(request); err != nil {
		return UnwrapResult{}, err
	}
	ciphertext := string(request.Ciphertext)
	if !validTransitCiphertext(ciphertext) {
		return UnwrapResult{}, fault.New(fault.Invalid, "transit_ciphertext")
	}
	configuration, token, _, err := t.authorize(ctx, request.Association, request.Context, transitActionUnwrap)
	if err != nil {
		return UnwrapResult{}, err
	}
	if !transitCiphertextMatchesVersion(ciphertext, request.Association.KeyVersion) {
		return UnwrapResult{}, fault.New(fault.Trust, "transit_ciphertext_version")
	}
	endpoint, err := transitURL(configuration, "decrypt", request.Association.KeyID)
	if err != nil {
		return UnwrapResult{}, err
	}
	body := transitDecryptRequest{
		Ciphertext: ciphertext,
		Context:    base64.StdEncoding.EncodeToString(transitDerivedContext(request.Context)),
	}
	var response transitDecryptResponse
	if err := t.client.DoJSON(ctx, http.MethodPost, endpoint, transitHeaders(token, configuration.Namespace), body, &response); err != nil {
		return UnwrapResult{}, transitProviderFailure(ctx, err)
	}
	if err := transitContextError(ctx); err != nil {
		return UnwrapResult{}, err
	}
	plaintext, ok := transitPlaintext(response.Data.Plaintext)
	if !ok {
		return UnwrapResult{}, fault.New(fault.Trust, "transit_decrypt_response")
	}
	return UnwrapResult{
		Plaintext: plaintext,
		Receipt:   transitReceipt(request.Association, request.Context),
	}, nil
}

func validTransitCiphertext(value string) bool {
	_, err := transitCiphertextVersion(value)
	return err == nil
}

func transitCiphertextMatchesVersion(ciphertext, expectedVersion string) bool {
	version, err := transitCiphertextVersion(ciphertext)
	return err == nil && version == expectedVersion
}

func transitPlaintext(value string) ([]byte, bool) {
	plaintext, err := base64.StdEncoding.DecodeString(value)
	return plaintext, err == nil && validTransitMaterial(plaintext)
}

func transitReceipt(association KMSAssociation, context AssociationContext) KMSReceipt {
	return KMSReceipt{
		Provider:                 association.Provider,
		KeyID:                    association.KeyID,
		KeyVersion:               association.KeyVersion,
		Algorithm:                association.Algorithm,
		ConfigurationAttestation: association.ConfigurationAttestation,
		Context:                  context,
	}
}

func (t *Transit) authorize(ctx context.Context, association KMSAssociation, request AssociationContext, action transitAction) (TransitConfiguration, string, transitKeyConfig, error) {
	if err := transitContextError(ctx); err != nil {
		return TransitConfiguration{}, "", transitKeyConfig{}, err
	}
	if err := validateTransitAssociation(association, request); err != nil {
		return TransitConfiguration{}, "", transitKeyConfig{}, err
	}
	configuration, err := t.privateConfiguration(ctx, association)
	if err != nil {
		return TransitConfiguration{}, "", transitKeyConfig{}, err
	}
	token, err := t.credential(ctx, configuration)
	if err != nil {
		return TransitConfiguration{}, "", transitKeyConfig{}, err
	}
	config, err := t.nativeKeyConfig(ctx, association, configuration, token, action)
	if err != nil {
		return TransitConfiguration{}, "", transitKeyConfig{}, err
	}
	return configuration, token, config, nil
}

func (t *Transit) privateConfiguration(ctx context.Context, association KMSAssociation) (TransitConfiguration, error) {
	configuration, err := t.configurations.TransitConfiguration(ctx, association)
	if err != nil {
		return TransitConfiguration{}, transitConfigurationFailure(err)
	}
	if err := transitContextError(ctx); err != nil {
		return TransitConfiguration{}, err
	}
	if err := validateTransitConfiguration(configuration); err != nil {
		return TransitConfiguration{}, err
	}
	return configuration, nil
}

func (t *Transit) credential(ctx context.Context, configuration TransitConfiguration) (string, error) {
	token, err := t.credentials.Token(ctx, configuration.CredentialReference)
	if err != nil {
		return "", transitCredentialFailure(err)
	}
	if err := transitContextError(ctx); err != nil {
		return "", err
	}
	if !validTransitToken(token) {
		return "", fault.New(fault.Trust, "transit_credential")
	}
	return token, nil
}

func (t *Transit) nativeKeyConfig(ctx context.Context, association KMSAssociation, configuration TransitConfiguration, token string, action transitAction) (transitKeyConfig, error) {
	endpoint, err := transitURL(configuration, "keys", association.KeyID)
	if err != nil {
		return transitKeyConfig{}, err
	}
	var response transitKeyResponse
	if err := t.client.DoJSON(ctx, http.MethodGet, endpoint, transitHeaders(token, configuration.Namespace), nil, &response); err != nil {
		return transitKeyConfig{}, transitProviderFailure(ctx, err)
	}
	if err := transitContextError(ctx); err != nil {
		return transitKeyConfig{}, err
	}
	config, err := response.Data.config()
	if err != nil || !transitConfigMatches(association, configuration, config, action) {
		return transitKeyConfig{}, fault.New(fault.Trust, "transit_key_configuration")
	}
	return config, nil
}

func transitWrapRequest(request WrapRequest) error {
	if !validTransitMaterial(request.Plaintext) {
		return fault.New(fault.Invalid, "transit_wrap_request")
	}
	return nil
}

func transitUnwrapRequest(request UnwrapRequest) error {
	if !validTransitMaterial(request.Ciphertext) {
		return fault.New(fault.Invalid, "transit_unwrap_request")
	}
	return nil
}

func validateTransitAssociation(association KMSAssociation, request AssociationContext) error {
	if association.Provider != transitProviderOpenBao && association.Provider != transitProviderVault {
		return fault.New(fault.Unsupported, "transit_provider")
	}
	if association.Algorithm != transitAlgorithm || !validTransitOpaque(association.KeyID) || !validTransitVersion(association.KeyVersion) || !validTransitOpaque(association.Purpose) || !validTransitOpaque(association.ConfigurationAttestation) {
		return fault.New(fault.Unsupported, "transit_association")
	}
	if !validTransitContext(request) || request.Purpose != association.Purpose {
		return fault.New(fault.Invalid, "transit_context")
	}
	return nil
}

func validateTransitConfiguration(configuration TransitConfiguration) error {
	if !validTransitOpaque(configuration.Mount) ||
		!validTransitOpaque(configuration.CredentialReference) ||
		!validTransitNamespace(configuration.Namespace) ||
		!validTransitMinimumVersion(configuration.MinDecryptionVersion) ||
		!validTransitMinimumVersion(configuration.MinEncryptionVersion) {
		return fault.New(fault.Invalid, "transit_configuration")
	}
	if _, err := transport.Endpoint(configuration.Endpoint); err != nil {
		return fault.New(fault.Trust, "transit_endpoint")
	}
	return nil
}

type transitAction uint8

const (
	transitActionWrap transitAction = iota + 1
	transitActionUnwrap
)

func transitConfigMatches(association KMSAssociation, configuration TransitConfiguration, config transitKeyConfig, action transitAction) bool {
	if !transitConfigKeyBindingMatches(association, configuration, config) {
		return false
	}
	if !transitConfigSecurityMatches(config) || !config.hasVersion(association.KeyVersion) {
		return false
	}
	if !transitVersionInRange(association.KeyVersion, config.minDecryptionVersion, config.latestVersion) {
		return false
	}
	return transitConfigActionMatches(association.KeyVersion, config, action)
}

func transitConfigKeyBindingMatches(association KMSAssociation, configuration TransitConfiguration, config transitKeyConfig) bool {
	return config.name == association.KeyID &&
		config.algorithm == association.Algorithm &&
		config.minDecryptionVersion == configuration.MinDecryptionVersion &&
		config.minEncryptionVersion == configuration.MinEncryptionVersion
}

func transitConfigSecurityMatches(config transitKeyConfig) bool {
	return config.derived && !config.convergent && !config.exportable && !config.allowPlaintextBackup
}

func transitConfigActionMatches(version string, config transitKeyConfig, action transitAction) bool {
	switch action {
	case transitActionUnwrap:
		return true
	case transitActionWrap:
		return transitVersionInRange(version, config.minEncryptionVersion, config.latestVersion)
	default:
		return false
	}
}

func transitURL(configuration TransitConfiguration, operation, keyID string) (*url.URL, error) {
	endpoint, err := transport.URL(configuration.Endpoint, "v1", configuration.Mount, operation, keyID)
	if err != nil {
		return nil, fault.New(fault.Invalid, "transit_resource")
	}
	return endpoint, nil
}

func transitHeaders(token, namespace string) map[string]string {
	headers := map[string]string{"X-Vault-Token": token}
	if namespace != "" {
		headers["X-Vault-Namespace"] = namespace
	}
	return headers
}

func validTransitMaterial(value []byte) bool {
	return len(value) > 0 && len(value) <= maximumTransitMaterial
}

func validTransitToken(value string) bool {
	return value != "" && len(value) <= maximumTransitToken && utf8.ValidString(value) && !strings.ContainsAny(value, transitForbiddenControlCharacters)
}

func validTransitOpaque(value string) bool {
	return value != "" && len(value) <= maximumTransitOpaque && utf8.ValidString(value) && !strings.ContainsAny(value, transitForbiddenControlCharacters)
}

func validTransitNamespace(value string) bool {
	return value == "" || validTransitOpaque(value)
}

func validTransitVersion(value string) bool {
	if value == "" || len(value) > 10 || (len(value) > 1 && value[0] == '0') {
		return false
	}
	for _, character := range value {
		if character < '0' || character > '9' {
			return false
		}
	}
	version, err := strconv.ParseUint(value, 10, 32)
	return err == nil && version > 0
}

func validTransitMinimumVersion(value string) bool {
	if value == "0" {
		return true
	}
	return validTransitVersion(value)
}

func transitVersionInRange(value, minimum, maximum string) bool {
	version, ok := transitComparableVersion(value, false)
	if !ok {
		return false
	}
	minimumVersion, ok := transitComparableVersion(minimum, true)
	if !ok {
		return false
	}
	maximumVersion, ok := transitComparableVersion(maximum, false)
	return ok && minimumVersion <= version && version <= maximumVersion
}

func transitComparableVersion(value string, allowZero bool) (uint64, bool) {
	if (allowZero && !validTransitMinimumVersion(value)) || (!allowZero && !validTransitVersion(value)) {
		return 0, false
	}
	version, err := strconv.ParseUint(value, 10, 32)
	return version, err == nil
}

func transitVersionNumber(value string) (uint64, bool) {
	if !validTransitVersion(value) {
		return 0, false
	}
	version, err := strconv.ParseUint(value, 10, 32)
	return version, err == nil && version > 0
}

func validTransitContext(value AssociationContext) bool {
	return validTransitOpaque(value.Tenant) &&
		validTransitOpaque(value.Profile) &&
		validTransitOpaque(value.Application) &&
		validTransitOpaque(value.Principal) &&
		validTransitOpaque(value.Generation) &&
		validTransitOpaque(value.Purpose) &&
		validTransitOpaque(value.ArtifactID) &&
		validTransitOpaque(value.ArtifactVersion)
}

func transitDerivedContext(value AssociationContext) []byte {
	fields := []string{value.Tenant, value.Profile, value.Application, value.Principal, value.Generation, value.Purpose, value.ArtifactID, value.ArtifactVersion}
	var result strings.Builder
	result.WriteString("cq-keyservice-transit-v1")
	for _, field := range fields {
		fmt.Fprintf(&result, "|%d:%s", len(field), field)
	}
	return []byte(result.String())
}

func transitCiphertextVersion(value string) (string, error) {
	if !validTransitCiphertextText(value) {
		return "", errors.New("invalid Transit ciphertext")
	}
	version, payload, ok := transitCiphertextParts(value)
	if !ok || !validTransitCiphertextPayload(payload) {
		return "", errors.New("invalid Transit ciphertext")
	}
	return version, nil
}

func validTransitCiphertextText(value string) bool {
	return len(value) <= maximumTransitMaterial*2 &&
		utf8.ValidString(value) &&
		!strings.ContainsAny(value, transitForbiddenControlCharacters)
}

func transitCiphertextParts(value string) (string, string, bool) {
	parts := strings.Split(value, ":")
	if len(parts) != 3 || parts[0] != "vault" || len(parts[1]) < 2 || parts[1][0] != 'v' || !validTransitVersion(parts[1][1:]) || parts[2] == "" {
		return "", "", false
	}
	return parts[1][1:], parts[2], true
}

func validTransitCiphertextPayload(value string) bool {
	decoded, err := base64.StdEncoding.DecodeString(value)
	return err == nil && len(decoded) > 0 && len(decoded) <= maximumTransitMaterial*2
}

func transitContextError(ctx context.Context) error {
	if ctx == nil || ctx.Err() != nil {
		return fault.New(fault.Denied, "transit_cancelled")
	}
	return nil
}

func transitConfigurationFailure(err error) error {
	var safe *fault.Error
	if errors.As(err, &safe) {
		switch safe.Kind {
		case fault.Invalid:
			return fault.New(fault.Invalid, "transit_configuration")
		case fault.Denied:
			return fault.New(fault.Denied, "transit_configuration")
		case fault.Missing:
			return fault.New(fault.Missing, "transit_configuration")
		}
	}
	return fault.New(fault.Trust, "transit_configuration")
}

func transitCredentialFailure(err error) error {
	var safe *fault.Error
	if errors.As(err, &safe) {
		switch safe.Kind {
		case fault.Invalid:
			return fault.New(fault.Invalid, "transit_credential")
		case fault.Denied:
			return fault.New(fault.Denied, "transit_credential")
		}
	}
	return fault.New(fault.Trust, "transit_credential")
}

func transitProviderFailure(ctx context.Context, err error) error {
	if contextErr := transitContextError(ctx); contextErr != nil {
		return contextErr
	}
	var safe *fault.Error
	if errors.As(err, &safe) {
		if safe.Kind == fault.Unavailable && transitAvailabilityOperation(safe.Operation) {
			return fault.New(fault.Unavailable, "transit_availability")
		}
		switch safe.Kind {
		case fault.Denied:
			return fault.New(fault.Denied, "transit_provider")
		case fault.Missing:
			return fault.New(fault.Missing, "transit_provider")
		case fault.Invalid:
			return fault.New(fault.Invalid, "transit_provider")
		case fault.Unsupported:
			return fault.New(fault.Unsupported, "transit_provider")
		}
	}
	return fault.New(fault.Trust, "transit_provider")
}

func transitAvailabilityOperation(operation string) bool {
	return operation == "provider_connection" || operation == "provider_timeout"
}

type transitEncryptRequest struct {
	Plaintext  string `json:"plaintext"`
	Context    string `json:"context"`
	KeyVersion uint64 `json:"key_version"`
}

type transitDecryptRequest struct {
	Ciphertext string `json:"ciphertext"`
	Context    string `json:"context"`
}

type transitEncryptResponse struct {
	Data transitEncryptData `json:"data"`
}

type transitEncryptData struct {
	Ciphertext string      `json:"ciphertext"`
	KeyVersion json.Number `json:"key_version"`
}

type transitDecryptResponse struct {
	Data transitDecryptData `json:"data"`
}

type transitDecryptData struct {
	Plaintext string `json:"plaintext"`
}

type transitKeyResponse struct {
	Data transitKeyData `json:"data"`
}

type transitKeyData struct {
	Name                 string                 `json:"name"`
	Type                 string                 `json:"type"`
	LatestVersion        json.Number            `json:"latest_version"`
	MinDecryptionVersion json.Number            `json:"min_decryption_version"`
	MinEncryptionVersion json.Number            `json:"min_encryption_version"`
	Keys                 map[string]json.Number `json:"keys"`
	Derived              *bool                  `json:"derived"`
	ConvergentEncryption *bool                  `json:"convergent_encryption"`
	Exportable           *bool                  `json:"exportable"`
	AllowPlaintextBackup *bool                  `json:"allow_plaintext_backup"`
}

func (response *transitEncryptResponse) UnmarshalJSON(data []byte) error {
	var result transitEncryptData
	if err := decodeTransitResponse(data, &result); err != nil {
		return err
	}
	response.Data = result
	return nil
}

func (response *transitDecryptResponse) UnmarshalJSON(data []byte) error {
	var result transitDecryptData
	if err := decodeTransitResponse(data, &result); err != nil {
		return err
	}
	response.Data = result
	return nil
}

func (response *transitKeyResponse) UnmarshalJSON(data []byte) error {
	var result transitKeyData
	if err := decodeTransitResponse(data, &result); err != nil {
		return err
	}
	response.Data = result
	return nil
}

// decodeTransitResponse first validates the complete vendor envelope for UTF-8,
// duplicate fields, nesting, and trailing input. The provider can include
// additional envelope or data fields; only the required adapter fields are
// decoded and semantically validated before use.
func decodeTransitResponse(data []byte, target any) error {
	var envelope map[string]json.RawMessage
	if err := jsondocument.Decode(data, &envelope); err != nil {
		return errors.New("invalid Transit response")
	}
	raw, found := envelope["data"]
	if !found || len(raw) == 0 || json.Unmarshal(raw, target) != nil {
		return errors.New("invalid Transit response")
	}
	return nil
}

type transitKeyConfig struct {
	name                 string
	algorithm            string
	latestVersion        string
	minDecryptionVersion string
	minEncryptionVersion string
	derived              bool
	convergent           bool
	exportable           bool
	allowPlaintextBackup bool
	versions             map[string]struct{}
}

func (response transitKeyResponse) config() (transitKeyConfig, error) {
	return response.Data.config()
}

func (data transitKeyData) config() (transitKeyConfig, error) {
	config, ok := data.keyMetadata()
	if !ok {
		return transitKeyConfig{}, errors.New("invalid Transit key response")
	}
	versions, ok := transitKeyVersions(data.Keys)
	if !ok {
		return transitKeyConfig{}, errors.New("invalid Transit key response")
	}
	config.versions = versions
	return config, nil
}

func (data transitKeyData) keyMetadata() (transitKeyConfig, bool) {
	if !data.hasRequiredMetadata() {
		return transitKeyConfig{}, false
	}
	latestVersion := data.LatestVersion.String()
	minimumDecryption := data.MinDecryptionVersion.String()
	minimumEncryption := data.MinEncryptionVersion.String()
	if !validTransitVersion(latestVersion) ||
		!validTransitMinimumVersion(minimumDecryption) ||
		!validTransitMinimumVersion(minimumEncryption) {
		return transitKeyConfig{}, false
	}
	return transitKeyConfig{
		name:                 data.Name,
		algorithm:            data.Type,
		latestVersion:        latestVersion,
		minDecryptionVersion: minimumDecryption,
		minEncryptionVersion: minimumEncryption,
		derived:              *data.Derived,
		convergent:           *data.ConvergentEncryption,
		exportable:           *data.Exportable,
		allowPlaintextBackup: *data.AllowPlaintextBackup,
	}, true
}

func (data transitKeyData) hasRequiredMetadata() bool {
	return validTransitOpaque(data.Name) &&
		validTransitOpaque(data.Type) &&
		data.Derived != nil &&
		data.ConvergentEncryption != nil &&
		data.Exportable != nil &&
		data.AllowPlaintextBackup != nil &&
		len(data.Keys) > 0
}

func transitKeyVersions(keys map[string]json.Number) (map[string]struct{}, bool) {
	versions := make(map[string]struct{}, len(keys))
	for keyVersion, createdAt := range keys {
		if !validTransitVersion(keyVersion) || !validTransitMinimumVersion(createdAt.String()) {
			return nil, false
		}
		versions[keyVersion] = struct{}{}
	}
	return versions, true
}

func (config transitKeyConfig) hasVersion(version string) bool {
	_, found := config.versions[version]
	return found
}
