// Package identity verifies narrowly configured external OAuth access tokens.
package identity

import (
	"context"
	"errors"
	"net/http"
	"time"
)

// Profile identifies a supported signed access-token claim profile.
type Profile string

const (
	// ProfileRFC9068 verifies RFC 9068 JWT access tokens with an at+jwt header.
	ProfileRFC9068 Profile = "rfc9068"
	// ProfileMicrosoftEntraV2 verifies Microsoft Entra v2 access tokens.
	ProfileMicrosoftEntraV2 Profile = "microsoft-entra-v2"
)

// Authority identifies one named issuer and its fixed access-token contract.
// A Microsoft Entra v2 Authority requires a GUID API audience and a GUID tenant
// in its issuer path. Clients and Subjects are optional any-exact allowlists;
// RequiredScopes and RequiredRoles are optional all-of requirements. An empty
// list leaves that dimension to policy.
type Authority struct {
	Name           string
	Issuer         string
	JWKSURL        string
	Audience       string
	Profile        Profile
	Clients        []string
	Subjects       []string
	RequiredScopes []string
	RequiredRoles  []string
}

// Principal is the signed identity and permissions extracted from an access
// token. Subject is issuer-qualified: RFC 9068 uses "issuer|sub", and
// Microsoft Entra v2 uses "issuer|tid/oid".
type Principal struct {
	Authority string
	Issuer    string
	Subject   string
	Client    string
	Scopes    []string
	Roles     []string
	ExpiresAt time.Time
}

// Verifier verifies one raw bearer access token for an Authority.
type Verifier func(context.Context, string) (Principal, error)

// ErrInvalidToken is returned when a raw bearer token fails the Authority's
// signature, profile, claim, or configured permission requirements.
var ErrInvalidToken = errors.New("invalid access token")

// NewVerifier builds a strict verifier for one named Authority. The supplied
// HTTP client is used only to obtain public keys and is wrapped to bound bodies
// and deny redirects.
func NewVerifier(ctx context.Context, authority Authority, client *http.Client) (Verifier, error) {
	return newVerifier(ctx, authority, client, time.Now)
}
