package identity

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/maldous/CircularQuay/internal/fault"
	"github.com/maldous/CircularQuay/internal/jsondocument"
	"github.com/maldous/CircularQuay/internal/transport"
)

const (
	maximumTokenBytes = 16384
	maximumClaimItems = 64
	maximumClaimText  = 512
)

type authorityConfig struct {
	tenant string
}

func newVerifier(ctx context.Context, authority Authority, client *http.Client, now func() time.Time) (Verifier, error) {
	config, err := validatedAuthority(authority)
	if err != nil {
		return nil, err
	}
	if authority.TrustStoreFile != "" {
		if client != nil {
			return nil, fault.New(fault.Invalid, "identity trust client")
		}
		client, err = transport.TrustStoreHTTPClient(authority.TrustStoreFile, authority.TrustStoreSHA256)
		if err != nil {
			return nil, fault.New(fault.Trust, "identity trust store")
		}
		// Match the public MCP verifier's established key-retrieval deadline.
		client.Timeout = 10 * time.Second
	}
	keyClient := keyClient(client)
	keyContext := oidc.ClientContext(ctx, keyClient)
	keySet := oidc.NewRemoteKeySet(keyContext, authority.JWKSURL)
	verifier := oidc.NewVerifier(authority.Issuer, keySet, &oidc.Config{
		ClientID:             authority.Audience,
		SupportedSigningAlgs: []string{oidc.RS256, oidc.ES256},
		Now:                  now,
	})
	return func(ctx context.Context, raw string) (Principal, error) {
		return verify(ctx, raw, authority, config, verifier, now)
	}, nil
}

// ValidateAuthority checks an Authority without opening a network connection.
func ValidateAuthority(authority Authority) error {
	_, err := validatedAuthority(authority)
	return err
}

func validatedAuthority(authority Authority) (authorityConfig, error) {
	if !validAuthorityFields(authority) {
		return authorityConfig{}, fault.New(fault.Invalid, "identity authority")
	}
	issuer, keys, err := authorityURLs(authority)
	if err != nil {
		return authorityConfig{}, fault.New(fault.Invalid, "identity authority")
	}
	if !sameOrigin(issuer, keys) {
		return authorityConfig{}, fault.New(fault.Trust, "identity key origin")
	}
	config := authorityConfig{tenant: entraTenantFor(authority.Profile, issuer)}
	if authority.Profile == ProfileMicrosoftEntraV2 && (!validGUID(config.tenant) || !validGUID(authority.Audience)) {
		return authorityConfig{}, fault.New(fault.Invalid, "identity entra issuer")
	}
	return config, nil
}

func validAuthorityFields(authority Authority) bool {
	return validName(authority.Name) && validText(authority.Audience) && validProfile(authority.Profile) && validFilters(authority) && validTrustStore(authority)
}

func validTrustStore(authority Authority) bool {
	if authority.TrustStoreFile == "" && authority.TrustStoreSHA256 == "" {
		return true
	}
	return filepath.IsAbs(authority.TrustStoreFile) && filepath.Clean(authority.TrustStoreFile) == authority.TrustStoreFile && validTrustStoreDigest(authority.TrustStoreSHA256)
}

func validTrustStoreDigest(value string) bool {
	if len(value) != 64 {
		return false
	}
	for _, character := range value {
		if !(character >= '0' && character <= '9' || character >= 'a' && character <= 'f') {
			return false
		}
	}
	return true
}

func authorityURLs(authority Authority) (*url.URL, *url.URL, error) {
	issuer, issuerErr := parseAuthorityURL(authority.Issuer)
	keys, keysErr := parseAuthorityURL(authority.JWKSURL)
	if issuerErr != nil || keysErr != nil {
		return nil, nil, ErrInvalidToken
	}
	return issuer, keys, nil
}

func sameOrigin(first, second *url.URL) bool {
	return first.Scheme == second.Scheme && first.Host == second.Host
}

func entraTenantFor(profile Profile, issuer *url.URL) string {
	if profile != ProfileMicrosoftEntraV2 {
		return ""
	}
	return entraTenant(issuer)
}

func validProfile(profile Profile) bool {
	return profile == ProfileRFC9068 || profile == ProfileMicrosoftEntraV2
}

func validName(value string) bool {
	if len(value) == 0 || len(value) > 128 {
		return false
	}
	for index, character := range value {
		if !isNameCharacter(character) || index == 0 && !isInitialNameCharacter(character) {
			return false
		}
	}
	return true
}

func isNameCharacter(character rune) bool {
	return isAlphaNumeric(character) || character == '.' || character == '_' || character == '-'
}

func isInitialNameCharacter(character rune) bool {
	return isAlphaNumeric(character)
}

func isAlphaNumeric(character rune) bool {
	return character >= 'a' && character <= 'z' || character >= 'A' && character <= 'Z' || character >= '0' && character <= '9'
}

func validText(value string) bool {
	return value != "" && len(value) <= maximumClaimText && !strings.ContainsAny(value, "\x00\r\n")
}

func validFilters(authority Authority) bool {
	return validValues(authority.Clients) && validValues(authority.Subjects) && validValues(authority.RequiredScopes) && validValues(authority.RequiredRoles)
}

func validValues(values []string) bool {
	if len(values) > maximumClaimItems {
		return false
	}
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		if !validText(value) {
			return false
		}
		if _, exists := seen[value]; exists {
			return false
		}
		seen[value] = struct{}{}
	}
	return true
}

func parseAuthorityURL(raw string) (*url.URL, error) {
	u, err := url.Parse(raw)
	if err != nil || u.User != nil || u.Host == "" || u.RawQuery != "" || u.Fragment != "" {
		return nil, ErrInvalidToken
	}
	if u.Scheme == "https" {
		return u, nil
	}
	if u.Scheme == "http" && isLoopback(u.Hostname()) {
		return u, nil
	}
	return nil, ErrInvalidToken
}

func isLoopback(host string) bool {
	return host == "localhost" || net.ParseIP(host) != nil && net.ParseIP(host).IsLoopback()
}

func entraTenant(issuer *url.URL) string {
	segments := strings.Split(strings.Trim(issuer.Path, "/"), "/")
	if len(segments) != 2 || segments[0] == "" || segments[1] != "v2.0" {
		return ""
	}
	return segments[0]
}

func keyClient(client *http.Client) *http.Client {
	if client == nil {
		return transport.NewHTTPClient(nil)
	}
	bounded := transport.NewHTTPClient(client.Transport)
	if client.Timeout > 0 && client.Timeout < bounded.Timeout {
		bounded.Timeout = client.Timeout
	}
	return bounded
}

func verify(ctx context.Context, raw string, authority Authority, config authorityConfig, verifier *oidc.IDTokenVerifier, now func() time.Time) (Principal, error) {
	header, payload, err := tokenDocuments(raw)
	if err != nil || !accessHeader(header, authority.Profile) {
		return Principal{}, ErrInvalidToken
	}
	token, err := verifier.Verify(ctx, raw)
	if err != nil {
		return Principal{}, ErrInvalidToken
	}
	claims, err := parseClaims(payload, token, authority.RequireDPoP)
	if err != nil {
		return Principal{}, ErrInvalidToken
	}
	principal, err := profilePrincipal(authority, config, token, claims, now())
	if err != nil || !permittedPrincipal(authority, principal) {
		return Principal{}, ErrInvalidToken
	}
	return principal, nil
}

type tokenClaims struct {
	ClientID  string          `json:"client_id"`
	AZP       string          `json:"azp"`
	Scope     string          `json:"scope"`
	Scopes    string          `json:"scp"`
	Roles     []string        `json:"roles"`
	ID        string          `json:"jti"`
	IssuedAt  int64           `json:"iat"`
	NotBefore int64           `json:"nbf"`
	Version   string          `json:"ver"`
	TenantID  string          `json:"tid"`
	ObjectID  string          `json:"oid"`
	CNF       json.RawMessage `json:"cnf"`
	Nonce     json.RawMessage `json:"nonce"`
}

func parseClaims(payload []byte, token *oidc.IDToken, requireDPoP bool) (tokenClaims, error) {
	var fields map[string]json.RawMessage
	if jsondocument.Decode(payload, &fields) != nil || nullClaim(fields) {
		return tokenClaims{}, ErrInvalidToken
	}
	var claims tokenClaims
	if json.Unmarshal(payload, &claims) != nil || unsupportedProof(claims, requireDPoP) {
		return tokenClaims{}, ErrInvalidToken
	}
	if !validIdentityClaimText(token.Subject, claims) || !validPermissionClaimText(claims) {
		return tokenClaims{}, ErrInvalidToken
	}
	return claims, nil
}

func nullClaim(fields map[string]json.RawMessage) bool {
	for _, name := range []string{"client_id", "azp", "scope", "scp", "roles", "jti", "iat", "nbf", "ver", "tid", "oid", "cnf", "nonce"} {
		if string(fields[name]) == "null" {
			return true
		}
	}
	return false
}

func unsupportedProof(claims tokenClaims, requireDPoP bool) bool {
	if claims.Nonce != nil {
		return true
	}
	if !requireDPoP {
		return claims.CNF != nil
	}
	return confirmationJKT(claims.CNF) == ""
}

func confirmationJKT(raw json.RawMessage) string {
	if raw == nil {
		return ""
	}
	var value struct {
		JKT string `json:"jkt"`
	}
	if jsondocument.Decode(raw, &value) != nil || !validJKT(value.JKT) {
		return ""
	}
	return value.JKT
}

func validJKT(value string) bool {
	decoded, err := base64.RawURLEncoding.DecodeString(value)
	return err == nil && len(decoded) == 32 && base64.RawURLEncoding.EncodeToString(decoded) == value
}

func validIdentityClaimText(subject string, claims tokenClaims) bool {
	return validTokenText(subject) && validTokenTextOrEmpty(claims.AZP) && validTokenTextOrEmpty(claims.ClientID) && validTokenTextOrEmpty(claims.ID)
}

func validPermissionClaimText(claims tokenClaims) bool {
	_, validScope := splitScopes(claims.Scope)
	_, validEntraScopes := splitScopes(claims.Scopes)
	return validScope && validEntraScopes && validValues(claims.Roles)
}

func splitScopes(raw string) ([]string, bool) {
	if raw == "" {
		return nil, true
	}
	scopes := strings.Split(raw, " ")
	if len(scopes) > maximumClaimItems {
		return nil, false
	}
	seen := make(map[string]struct{}, len(scopes))
	for _, scope := range scopes {
		if !validScope(scope) || scopeSeen(seen, scope) {
			return nil, false
		}
	}
	return scopes, true
}

func validScope(value string) bool {
	if value == "" || len(value) > maximumClaimText {
		return false
	}
	for _, character := range []byte(value) {
		if character == '"' || character == '\\' || character < '!' || character > '~' {
			return false
		}
	}
	return true
}

func scopeSeen(seen map[string]struct{}, scope string) bool {
	if _, exists := seen[scope]; exists {
		return true
	}
	seen[scope] = struct{}{}
	return false
}

func validTokenText(value string) bool {
	return validText(value)
}

func validTokenTextOrEmpty(value string) bool {
	return value == "" || validText(value)
}

func profilePrincipal(authority Authority, config authorityConfig, token *oidc.IDToken, claims tokenClaims, now time.Time) (Principal, error) {
	if !validTimes(claims, token.Expiry, now) {
		return Principal{}, ErrInvalidToken
	}
	if authority.Profile == ProfileRFC9068 {
		return rfc9068Principal(authority, token, claims)
	}
	return entraPrincipal(authority, config, token, claims)
}

func validTimes(claims tokenClaims, expiry, now time.Time) bool {
	current := now.Unix()
	return !now.IsZero() && now.Before(expiry) && claims.IssuedAt > 0 && claims.IssuedAt <= current+30 && claims.IssuedAt <= expiry.Unix() && claims.NotBefore <= current
}

func rfc9068Principal(authority Authority, token *oidc.IDToken, claims tokenClaims) (Principal, error) {
	if claims.ClientID == "" || claims.ID == "" || claims.AZP != "" && claims.AZP != claims.ClientID {
		return Principal{}, ErrInvalidToken
	}
	scopes, _ := splitScopes(claims.Scope)
	return newPrincipal(authority, token, claims.ClientID, authority.Issuer+"|"+token.Subject, scopes, claims.Roles, confirmationJKT(claims.CNF)), nil
}

func entraPrincipal(authority Authority, config authorityConfig, token *oidc.IDToken, claims tokenClaims) (Principal, error) {
	if claims.Version != "2.0" || claims.TenantID != config.tenant || !validGUID(claims.TenantID) || !validGUID(claims.ObjectID) || !validGUID(claims.AZP) {
		return Principal{}, ErrInvalidToken
	}
	scopes, _ := splitScopes(claims.Scopes)
	if len(scopes) == 0 && len(claims.Roles) == 0 {
		return Principal{}, ErrInvalidToken
	}
	return newPrincipal(authority, token, claims.AZP, authority.Issuer+"|"+claims.TenantID+"/"+claims.ObjectID, scopes, claims.Roles, confirmationJKT(claims.CNF)), nil
}

func newPrincipal(authority Authority, token *oidc.IDToken, client, subject string, scopes, roles []string, confirmation string) Principal {
	return Principal{Authority: authority.Name, Issuer: authority.Issuer, Subject: subject, Client: client, Scopes: scopes, Roles: roles, ExpiresAt: token.Expiry, ConfirmationJKT: confirmation}
}

func permittedPrincipal(authority Authority, principal Principal) bool {
	return containsAny(authority.Clients, principal.Client) && containsAny(authority.Subjects, principal.Subject) && containsAll(authority.RequiredScopes, principal.Scopes) && containsAll(authority.RequiredRoles, principal.Roles)
}

func containsAny(allowed []string, value string) bool {
	if len(allowed) == 0 {
		return true
	}
	for _, candidate := range allowed {
		if candidate == value {
			return true
		}
	}
	return false
}

func containsAll(required, actual []string) bool {
	if len(required) == 0 {
		return true
	}
	available := make(map[string]struct{}, len(actual))
	for _, value := range actual {
		available[value] = struct{}{}
	}
	for _, value := range required {
		if _, exists := available[value]; !exists {
			return false
		}
	}
	return true
}

func accessHeader(header []byte, profile Profile) bool {
	var value struct {
		Type string `json:"typ"`
	}
	if jsondocument.Decode(header, &map[string]json.RawMessage{}) != nil || json.Unmarshal(header, &value) != nil {
		return false
	}
	return profile == ProfileRFC9068 && value.Type == "at+jwt" || profile == ProfileMicrosoftEntraV2 && value.Type == "JWT"
}

func tokenDocuments(raw string) ([]byte, []byte, error) {
	if len(raw) == 0 || len(raw) > maximumTokenBytes {
		return nil, nil, ErrInvalidToken
	}
	parts := strings.Split(raw, ".")
	if len(parts) != 3 {
		return nil, nil, ErrInvalidToken
	}
	header, headerErr := base64.RawURLEncoding.DecodeString(parts[0])
	payload, payloadErr := base64.RawURLEncoding.DecodeString(parts[1])
	if headerErr != nil || payloadErr != nil {
		return nil, nil, ErrInvalidToken
	}
	return header, payload, nil
}

func validGUID(value string) bool {
	if len(value) != 36 {
		return false
	}
	for index, character := range value {
		if index == 8 || index == 13 || index == 18 || index == 23 {
			if character != '-' {
				return false
			}
			continue
		}
		if !isHex(character) {
			return false
		}
	}
	return true
}

func isHex(character rune) bool {
	return character >= '0' && character <= '9' || character >= 'a' && character <= 'f' || character >= 'A' && character <= 'F'
}
