package mcp

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/maldous/CircularQuay/internal/fault"
	"github.com/maldous/CircularQuay/internal/model"
	"github.com/modelcontextprotocol/go-sdk/auth"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/modelcontextprotocol/go-sdk/oauthex"
)

const maximumRequestBytes = 65536

// HTTPHandler exposes a stateless protected resource. The caller owns TLS and
// http.Server deadlines; the default CLI binds loopback until explicitly configured.
func (s *Service) HTTPHandler() (http.Handler, error) {
	cfg := s.current().MCP
	verifier, err := NewAccessTokenVerifier(context.Background(), cfg)
	if err != nil {
		return nil, err
	}
	return s.httpHandler(verifier)
}

func (s *Service) httpHandler(verifier auth.TokenVerifier) (http.Handler, error) {
	cfg := s.current().MCP
	resource, err := url.Parse(cfg.Audience)
	if err != nil || !secureResource(resource) || cfg.Issuer == "" {
		return nil, fault.New(fault.Invalid, "mcp resource")
	}
	path := resource.Path
	if path == "" || path == "/" {
		return nil, fault.New(fault.Invalid, "mcp resource path")
	}
	metadataPath := "/.well-known/oauth-protected-resource" + path
	metadataURL := resource.Scheme + "://" + resource.Host + metadataPath
	stream := sdk.NewStreamableHTTPHandler(func(r *http.Request) *sdk.Server {
		ti := auth.TokenInfoFromContext(r.Context())
		if ti == nil {
			return nil
		}
		client, _ := ti.Extra["client"].(string)
		return s.server(Principal{Subject: ti.UserID, Client: client, Scopes: ti.Scopes, HTTP: true})
	}, &sdk.StreamableHTTPOptions{
		Stateless: true, JSONResponse: true, MaxRequestBodyBytes: maximumRequestBytes,
		PropagateRequestCancellation: true, Logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	})
	protected := auth.RequireBearerToken(verifier, &auth.RequireBearerTokenOptions{ResourceMetadataURL: metadataURL, Scopes: []string{"cquay:mcp"}})(stream)
	mux := http.NewServeMux()
	mux.Handle(path, protected)
	metadata := auth.ProtectedResourceMetadataHandler(&oauthex.ProtectedResourceMetadata{
		Resource: cfg.Audience, AuthorizationServers: []string{cfg.Issuer},
		ScopesSupported: []string{"cquay:mcp", "cquay:list", "cquay:status", "cquay:run", "cquay:read"},
	})
	mux.Handle(metadataPath, metadata)
	if metadataPath != "/.well-known/oauth-protected-resource" {
		mux.Handle("/.well-known/oauth-protected-resource", metadata)
	}
	hosts := cfg.Hosts
	if len(hosts) == 0 {
		hosts = []string{resource.Host}
	}
	return protectHTTP(mux, hosts, cfg.Origins), nil
}

func secureResource(u *url.URL) bool {
	if u == nil || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		return false
	}
	return u.Scheme == "https" || (u.Scheme == "http" && isLoopback(u.Hostname()))
}

func isLoopback(host string) bool {
	return host == "localhost" || net.ParseIP(host) != nil && net.ParseIP(host).IsLoopback()
}

func protectHTTP(next http.Handler, hosts, origins []string) http.Handler {
	concurrent := make(chan struct{}, 16)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "private, no-store")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Pragma", "no-cache")
		if !secureTransport(r) {
			http.Error(w, "TLS required", http.StatusForbidden)
			return
		}
		if !permittedHost(r.Host, hosts) || !permittedOrigin(r.Header.Get("Origin"), origins) {
			http.Error(w, "request origin denied", http.StatusForbidden)
			return
		}
		if r.ContentLength > maximumRequestBytes {
			http.Error(w, "request too large", http.StatusRequestEntityTooLarge)
			return
		}
		select {
		case concurrent <- struct{}{}:
			defer func() { <-concurrent }()
		default:
			http.Error(w, "request capacity exceeded", http.StatusTooManyRequests)
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, maximumRequestBytes)
		ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
		defer cancel()
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// A local TLS-terminating proxy may connect over loopback. Forwarded headers
// cannot turn a remote plaintext request into a trusted transport.
func secureTransport(r *http.Request) bool {
	if r.TLS != nil {
		return true
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	return err == nil && isLoopback(host)
}

func permittedHost(host string, allowed []string) bool {
	if strings.ContainsAny(host, "\x00\r\n/@\\") {
		return false
	}
	for _, candidate := range allowed {
		if strings.EqualFold(host, candidate) {
			return true
		}
	}
	return false
}

func permittedOrigin(origin string, allowed []string) bool {
	if origin == "" {
		return true
	} // Native bearer clients need no browser Origin.
	if origin == "null" {
		return false
	}
	for _, candidate := range allowed {
		if origin == candidate {
			return true
		}
	}
	return false
}

// NewAccessTokenVerifier supports a deliberately narrow RFC9068 JWT profile:
// at+jwt, trusted issuer/JWKS, resource audience, exp/iat/jti and client_id.
// Opaque tokens require a separately qualified introspection implementation.
func NewAccessTokenVerifier(ctx context.Context, cfg model.MCP) (auth.TokenVerifier, error) {
	client := &http.Client{Timeout: 10 * time.Second, CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse }}
	return newAccessTokenVerifier(ctx, cfg, client)
}

func newAccessTokenVerifier(ctx context.Context, cfg model.MCP, client *http.Client) (auth.TokenVerifier, error) {
	issuer, issuerErr := url.Parse(cfg.Issuer)
	keys, keyErr := url.Parse(cfg.JWKSURL)
	resource, resourceErr := url.Parse(cfg.Audience)
	if !validIssuerConfiguration(issuer, keys, resource, issuerErr, keyErr, resourceErr) {
		return nil, fault.New(fault.Invalid, "mcp issuer configuration")
	}
	// A cross-origin key set could disclose a credential or accept another issuer.
	if issuer.Scheme != keys.Scheme || issuer.Host != keys.Host {
		return nil, fault.New(fault.Trust, "mcp key origin")
	}
	ctx = oidc.ClientContext(ctx, boundedKeyClient(client))
	keySet := oidc.NewRemoteKeySet(ctx, cfg.JWKSURL)
	verifier := oidc.NewVerifier(cfg.Issuer, keySet, &oidc.Config{ClientID: cfg.Audience, SupportedSigningAlgs: []string{oidc.RS256, oidc.ES256}})
	return func(ctx context.Context, raw string, _ *http.Request) (*auth.TokenInfo, error) {
		return verifiedTokenInfo(ctx, raw, verifier, cfg.Issuer)
	}, nil
}

func validIssuerConfiguration(issuer, keys, resource *url.URL, issuerErr, keyErr, resourceErr error) bool {
	return issuerErr == nil && keyErr == nil && resourceErr == nil && secureResource(issuer) && secureResource(keys) && secureResource(resource)
}

func verifiedTokenInfo(ctx context.Context, raw string, verifier *oidc.IDTokenVerifier, issuer string) (*auth.TokenInfo, error) {
	if !accessHeader(raw) {
		return nil, auth.ErrInvalidToken
	}
	token, err := verifier.Verify(ctx, raw)
	if err != nil || token.Subject == "" || len(token.Subject) > 512 {
		return nil, auth.ErrInvalidToken
	}
	claims, err := verifiedClaims(token)
	if err != nil {
		return nil, auth.ErrInvalidToken
	}
	return &auth.TokenInfo{UserID: issuer + "|" + token.Subject, Expiration: token.Expiry, Scopes: strings.Fields(claims.Scope), Extra: map[string]any{"client": claims.ClientID}}, nil
}

type accessClaims struct {
	ClientID        string `json:"client_id"`
	AuthorizedParty string `json:"azp"`
	Scope           string `json:"scope"`
	ID              string `json:"jti"`
	IssuedAt        int64  `json:"iat"`
	NotBefore       int64  `json:"nbf"`
}

// Only signed access-token claims become authority. iat/jti are required by
// RFC9068; jti is not retained or logged, and bearer replay prevention still
// belongs to the issuer, short token lifetimes and the protected transport.
func verifiedClaims(token *oidc.IDToken) (accessClaims, error) {
	var claims accessClaims
	if token.Claims(&claims) != nil {
		return claims, auth.ErrInvalidToken
	}
	if !validClaimTimes(claims, token.Expiry) || !validClaimClient(claims) || !validClaimText(token.Subject, claims) {
		return claims, auth.ErrInvalidToken
	}
	return claims, nil
}

func validClaimTimes(claims accessClaims, expiresAt time.Time) bool {
	now := time.Now().Unix()
	return claims.NotBefore <= now && claims.IssuedAt > 0 && claims.IssuedAt <= now+30 && claims.IssuedAt <= expiresAt.Unix()
}

func validClaimClient(claims accessClaims) bool {
	return claims.ClientID != "" && len(claims.ClientID) <= 256 && (claims.AuthorizedParty == "" || claims.AuthorizedParty == claims.ClientID)
}

func validClaimText(subject string, claims accessClaims) bool {
	return claims.ID != "" && len(claims.ID) <= 512 && !strings.ContainsAny(subject+claims.ClientID+claims.ID+claims.Scope, "\x00\r\n")
}

// Remote keys are public, but a compromised issuer must not allocate unbounded
// memory in the resource server. Redirects are never followed to another origin.
func boundedKeyClient(client *http.Client) *http.Client {
	keyClient := *client
	keyClient.Timeout = 10 * time.Second
	keyClient.CheckRedirect = func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse }
	transport := keyClient.Transport
	if transport == nil {
		transport = http.DefaultTransport
	}
	keyClient.Transport = boundedKeyTransport{transport}
	return &keyClient
}

type boundedKeyTransport struct{ base http.RoundTripper }
type limitedKeyBody struct {
	io.Reader
	io.Closer
}

func (t boundedKeyTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	response, err := t.base.RoundTrip(r)
	if err != nil {
		return nil, err
	}
	response.Body = limitedKeyBody{Reader: io.LimitReader(response.Body, 1<<20), Closer: response.Body}
	return response, nil
}

func accessHeader(raw string) bool {
	if len(raw) > 16384 {
		return false
	}
	parts := strings.Split(raw, ".")
	if len(parts) != 3 {
		return false
	}
	header, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return false
	}
	var h struct {
		Type string `json:"typ"`
	}
	return json.Unmarshal(header, &h) == nil && strings.EqualFold(h.Type, "at+jwt")
}
