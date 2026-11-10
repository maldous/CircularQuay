package policy

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/url"
	"path/filepath"
	"strings"
	"time"

	"github.com/maldous/CircularQuay/internal/identity"
	"github.com/maldous/CircularQuay/internal/model"
	"github.com/maldous/CircularQuay/internal/transport"
)

func validateNamedAuthorities(config model.MCP) error {
	if !validNamedAuthorityContract(config) {
		return invalid("mcp_authority_contract")
	}
	if !validNamedResource(config.Resource) {
		return invalid("mcp_resource")
	}
	seen := make(map[string]bool)
	for name, authority := range config.Authorities {
		if !identifier.MatchString(name) || validateIdentityAuthority(name, authority) != nil {
			return invalid("mcp_named_authority")
		}
		key := authority.Issuer + "\x00" + authority.Audience
		if seen[key] {
			return invalid("mcp_ambiguous_authority")
		}
		seen[key] = true
	}
	return validateMCPLists(config)
}

func validNamedAuthorityContract(config model.MCP) bool {
	return config.Issuer == "" && config.Audience == "" && config.JWKSURL == "" && len(config.Authorities) > 0 && len(config.Authorities) <= 8
}

func validNamedResource(raw string) bool {
	resource, err := url.Parse(raw)
	return err == nil && secureEndpoint(raw, true) && resource.Path != "" && resource.Path != "/" && resource.RawPath == ""
}

func validateIdentityAuthority(name string, authority model.IdentityAuthority) error {
	if identity.ValidateAuthority(VerifierAuthority(name, authority)) != nil {
		return invalid("identity_profile")
	}
	if !secureEndpoint(authority.Issuer, false) || !secureEndpoint(authority.JWKSURL, false) || !sameIdentityOrigin(authority) || !validAudience(authority.Audience) {
		return invalid("identity_trust")
	}
	if !validTrustStoreDeclaration(authority) {
		return invalid("identity_trust_store")
	}
	if !validIdentityClaimFilters(authority) {
		return invalid("identity_claim_filter")
	}
	if !validIdentityValues(authority.RequiredScopes, true) {
		return invalid("identity_scope")
	}
	if !validSenderConstraint(authority.SenderConstraint) {
		return invalid("identity_sender_constraint")
	}
	return validateTokenPermissions(authority.Permissions)
}

func validIdentityClaimFilters(authority model.IdentityAuthority) bool {
	for _, values := range [][]string{authority.Clients, authority.Subjects, authority.RequiredRoles} {
		if !validIdentityValues(values, false) {
			return false
		}
	}
	return true
}

// VerifierAuthority maps a non-secret policy declaration into the shared access
// token verifier. It does not retrieve keys, authenticate or grant an operation.
func VerifierAuthority(name string, authority model.IdentityAuthority) identity.Authority {
	return identity.Authority{
		Name: name, Issuer: authority.Issuer, JWKSURL: authority.JWKSURL,
		TrustStoreFile: authority.TrustStoreFile, TrustStoreSHA256: authority.TrustStoreSHA256,
		Audience: authority.Audience, Profile: identity.Profile(authority.Profile),
		Clients: authority.Clients, Subjects: authority.Subjects,
		RequiredScopes: authority.RequiredScopes, RequiredRoles: authority.RequiredRoles,
		RequireDPoP: authority.SenderConstraint != nil,
	}
}

func validTrustStoreDeclaration(authority model.IdentityAuthority) bool {
	if authority.TrustStoreFile == "" && authority.TrustStoreSHA256 == "" {
		return true
	}
	return filepath.IsAbs(authority.TrustStoreFile) && filepath.Clean(authority.TrustStoreFile) == authority.TrustStoreFile && lowerHexDigest(authority.TrustStoreSHA256)
}

func lowerHexDigest(value string) bool {
	if len(value) != sha256.Size*2 {
		return false
	}
	for _, character := range value {
		if !(character >= '0' && character <= '9' || character >= 'a' && character <= 'f') {
			return false
		}
	}
	return true
}

// ValidateLoadedTrustStores checks protected trust material only after policy
// parsing. Parse stays side-effect free so staged configuration can be checked
// before it is published.
func ValidateLoadedTrustStores(config model.MCP) error {
	for _, authority := range config.Authorities {
		if authority.TrustStoreFile == "" {
			continue
		}
		if err := transport.ValidateTrustStore(authority.TrustStoreFile, authority.TrustStoreSHA256); err != nil {
			return invalid("identity_trust_store")
		}
	}
	return nil
}

func validSenderConstraint(value *model.SenderConstraint) bool {
	if value == nil {
		return true
	}
	return value.Type == "dpop" && value.Nonce == "required" && value.ReplayStore == "local_persistent" && value.ProofMaxAgeSeconds >= 1 && value.ProofMaxAgeSeconds <= 300 && value.ClockSkewSeconds >= 0 && value.ClockSkewSeconds <= 30 && time.Duration(value.ClockSkewSeconds)*time.Second <= time.Duration(value.ProofMaxAgeSeconds)*time.Second
}

func sameIdentityOrigin(authority model.IdentityAuthority) bool {
	issuer, issuerErr := url.Parse(authority.Issuer)
	keys, keyErr := url.Parse(authority.JWKSURL)
	return issuerErr == nil && keyErr == nil && issuer.Scheme == keys.Scheme && issuer.Host == keys.Host && issuer.RawPath == "" && keys.RawPath == ""
}

func validAudience(audience string) bool {
	return safePrincipal(audience) && !strings.ContainsAny(audience, " \t")
}

func validIdentityValues(values []string, scopes bool) bool {
	if len(values) > 64 {
		return false
	}
	seen := make(map[string]bool, len(values))
	for _, value := range values {
		valid := safePrincipal(value)
		if scopes {
			valid = validScope(value)
		}
		if !valid || len(value) > 512 || seen[value] {
			return false
		}
		seen[value] = true
	}
	return true
}

func validScope(value string) bool {
	for _, character := range value {
		if character < 0x21 || character > 0x7e || character == '"' || character == '\\' {
			return false
		}
	}
	return value != ""
}

func validateTokenPermissions(permissions map[string]model.TokenPermission) error {
	if len(permissions) == 0 || len(permissions) > len(operations) {
		return invalid("identity_permissions")
	}
	for operation, permission := range permissions {
		if !operations[operation] || len(permission.Scopes)+len(permission.Roles) == 0 {
			return invalid("identity_permission")
		}
		if !validIdentityValues(permission.Scopes, true) || !validIdentityValues(permission.Roles, false) {
			return invalid("identity_permission_claims")
		}
	}
	return nil
}

// MCPAuthorityFingerprint binds a verified HTTP principal to the trusted
// resource/authority contract that authenticated it. Rechecking against the
// current policy invalidates in-flight authority after trust or permission
// changes. Listen addresses, audit locations and other authorities are excluded.
func MCPAuthorityFingerprint(config model.MCP, name string) string {
	authority, found := config.Authorities[name]
	if !found {
		return ""
	}
	data, err := json.Marshal(struct {
		Resource  string
		Name      string
		Authority model.IdentityAuthority
	}{config.Resource, name, authority})
	if err != nil {
		return ""
	}
	digest := sha256.Sum256(data)
	return hex.EncodeToString(digest[:])
}
