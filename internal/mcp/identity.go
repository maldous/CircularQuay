package mcp

import (
	"context"
	"net/http"
	"sort"

	"github.com/maldous/CircularQuay/internal/identity"
	"github.com/maldous/CircularQuay/internal/model"
	"github.com/maldous/CircularQuay/internal/policy"
	"github.com/modelcontextprotocol/go-sdk/auth"
)

func configuredResource(config model.MCP) string {
	if config.Authorities != nil {
		return config.Resource
	}
	return config.Audience
}

func configuredIssuers(config model.MCP) []string {
	if config.Authorities == nil {
		if config.Issuer == "" {
			return nil
		}
		return []string{config.Issuer}
	}
	issuers := make(map[string]bool)
	for _, authority := range config.Authorities {
		issuers[authority.Issuer] = true
	}
	return sortedKeys(issuers)
}

func configuredScopes(config model.MCP) []string {
	if config.Authorities == nil {
		return []string{"cq:mcp", "cq:list", "cq:status", "cq:run", "cq:read"}
	}
	scopes := make(map[string]bool)
	for _, authority := range config.Authorities {
		for _, scope := range authority.RequiredScopes {
			scopes[scope] = true
		}
		for _, permission := range authority.Permissions {
			for _, scope := range permission.Scopes {
				scopes[scope] = true
			}
		}
	}
	return sortedKeys(scopes)
}

func sortedKeys(values map[string]bool) []string {
	result := make([]string, 0, len(values))
	for value := range values {
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

func requiredTransportScopes(config model.MCP) []string {
	if config.Authorities != nil {
		// Named verifiers enforce their own issuer-specific required claims.
		// A resource-wide hard-coded scope would reject valid role-only tokens.
		return nil
	}
	return []string{"cq:mcp"}
}

func principalFromToken(token *auth.TokenInfo, client string) Principal {
	authority, _ := token.Extra["authority"].(string)
	fingerprint, _ := token.Extra["authority_fingerprint"].(string)
	roles, _ := token.Extra["roles"].([]string)
	return Principal{
		Subject: token.UserID, Client: client, HTTP: true, Scopes: token.Scopes,
		ExpiresAt: token.Expiration, Authority: authority, Roles: roles, AuthorityFingerprint: fingerprint,
	}
}

type namedVerifier struct {
	name        string
	fingerprint string
	requireDPoP bool
	verify      identity.Verifier
}

func newNamedAccessTokenVerifierWithTrustStore(ctx context.Context, config model.MCP, client *http.Client, authorityScopedClient bool) (auth.TokenVerifier, error) {
	if err := policy.ValidateMCP(config); err != nil {
		return nil, err
	}
	verifiers := make([]namedVerifier, 0, len(config.Authorities))
	for _, name := range sortedAuthorityNames(config) {
		authority := config.Authorities[name]
		keyClient := client
		if authority.TrustStoreFile != "" && authorityScopedClient {
			// TrustStoreHTTPClient is created inside identity.NewVerifier. It must
			// not inherit the shared system-root client used by another authority.
			keyClient = nil
		}
		verify, err := identity.NewVerifier(ctx, policy.VerifierAuthority(name, authority), keyClient)
		if err != nil {
			return nil, err
		}
		verifiers = append(verifiers, namedVerifier{name: name, fingerprint: policy.MCPAuthorityFingerprint(config, name), requireDPoP: authority.SenderConstraint != nil, verify: verify})
	}
	return func(ctx context.Context, raw string, _ *http.Request) (*auth.TokenInfo, error) {
		return verifyNamedToken(ctx, raw, verifiers)
	}, nil
}

func sortedAuthorityNames(config model.MCP) []string {
	names := make(map[string]bool, len(config.Authorities))
	for name := range config.Authorities {
		names[name] = true
	}
	return sortedKeys(names)
}

// Only locally enrolled verifiers are tried. Token-controlled claims cannot
// introduce a key URL or a new authority. Ambiguous acceptance is denied rather
// than selecting permissions from whichever map entry happens to run first.
func verifyNamedToken(ctx context.Context, raw string, verifiers []namedVerifier) (*auth.TokenInfo, error) {
	var result *auth.TokenInfo
	for _, verifier := range verifiers {
		principal, err := verifier.verify(ctx, raw)
		if err != nil {
			continue
		}
		if result != nil || verifier.fingerprint == "" {
			return nil, auth.ErrInvalidToken
		}
		result = &auth.TokenInfo{
			UserID: principal.Subject, Expiration: principal.ExpiresAt, Scopes: principal.Scopes,
			Extra: map[string]any{"client": principal.Client, "roles": principal.Roles, "authority": verifier.name, "authority_fingerprint": verifier.fingerprint, "dpop_required": verifier.requireDPoP, "confirmation_jkt": principal.ConfirmationJKT},
		}
	}
	if result == nil {
		return nil, auth.ErrInvalidToken
	}
	return result, nil
}
