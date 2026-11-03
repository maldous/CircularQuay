package mcp

import (
	"context"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/maldous/CircularQuay/internal/fault"
	"github.com/maldous/CircularQuay/internal/identity"
	"github.com/maldous/CircularQuay/internal/model"
	"github.com/maldous/CircularQuay/internal/policy"
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
	resourceURL := configuredResource(cfg)
	resource, err := url.Parse(resourceURL)
	if err != nil || !secureResource(resource) || len(configuredIssuers(cfg)) == 0 {
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
		return s.server(principalFromToken(ti, client))
	}, &sdk.StreamableHTTPOptions{
		Stateless: true, JSONResponse: true, MaxRequestBodyBytes: maximumRequestBytes,
		PropagateRequestCancellation: true, Logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	})
	protected := auth.RequireBearerToken(verifier, &auth.RequireBearerTokenOptions{ResourceMetadataURL: metadataURL, Scopes: requiredTransportScopes(cfg)})(boundTokenDeadline(stream))
	mux := http.NewServeMux()
	mux.Handle(path, protected)
	metadata := auth.ProtectedResourceMetadataHandler(&oauthex.ProtectedResourceMetadata{
		Resource: resourceURL, AuthorizationServers: configuredIssuers(cfg),
		ScopesSupported: configuredScopes(cfg),
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

// boundTokenDeadline constrains a request to the lifetime of its verified
// bearer token. It runs after RequireBearerToken has established TokenInfo and
// before MCP dispatch, so provider or tool work cannot outlive the identity
// that authorized it. context.WithDeadline retains any earlier server deadline.
func boundTokenDeadline(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := auth.TokenInfoFromContext(r.Context())
		if token == nil || token.Expiration.IsZero() || !time.Now().Before(token.Expiration) {
			http.Error(w, "invalid access token", http.StatusUnauthorized)
			return
		}
		ctx, cancel := context.WithDeadline(r.Context(), token.Expiration)
		defer cancel()
		next.ServeHTTP(w, r.WithContext(ctx))
	})
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

// NewAccessTokenVerifier uses the shared strict access-token verifier for named
// authorities or the compact single-issuer RFC 9068 contract. Public resource
// metadata and provider authentication remain separate from token verification.
func NewAccessTokenVerifier(ctx context.Context, cfg model.MCP) (auth.TokenVerifier, error) {
	client := &http.Client{Timeout: 10 * time.Second, CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse }}
	return newAccessTokenVerifier(ctx, cfg, client)
}

func newAccessTokenVerifier(ctx context.Context, cfg model.MCP, client *http.Client) (auth.TokenVerifier, error) {
	if cfg.Authorities != nil {
		return newNamedAccessTokenVerifier(ctx, cfg, client)
	}
	resource, err := url.Parse(cfg.Audience)
	if err != nil || !secureResource(resource) {
		return nil, fault.New(fault.Invalid, "mcp resource audience")
	}
	verify, err := identity.NewVerifier(ctx, identity.Authority{
		Name: "legacy", Issuer: cfg.Issuer, JWKSURL: cfg.JWKSURL,
		Audience: cfg.Audience, Profile: identity.ProfileRFC9068,
	}, client)
	if err != nil {
		return nil, err
	}
	return func(ctx context.Context, raw string, _ *http.Request) (*auth.TokenInfo, error) {
		principal, err := verify(ctx, raw)
		if err != nil {
			return nil, auth.ErrInvalidToken
		}
		return &auth.TokenInfo{
			UserID: principal.Subject, Expiration: principal.ExpiresAt, Scopes: principal.Scopes,
			Extra: map[string]any{"client": principal.Client, "roles": principal.Roles, "authority_fingerprint": policy.MCPLegacyFingerprint(cfg)},
		}, nil
	}, nil
}
