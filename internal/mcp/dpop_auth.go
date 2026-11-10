package mcp

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/maldous/CircularQuay/internal/dpop"
	"github.com/maldous/CircularQuay/internal/model"
	"github.com/maldous/CircularQuay/internal/observability"
	"github.com/modelcontextprotocol/go-sdk/auth"
)

type accessTokenOptions struct {
	verifier    auth.TokenVerifier
	config      model.MCP
	replay      dpop.Store
	metadataURL string
	scopes      []string
}

type accessTokenHandler struct {
	accessTokenOptions
	bearer func(http.Handler) http.Handler
}

// requireAccessToken accepts Bearer tokens for ordinary authorities and DPoP
// only for authorities that explicitly select sender constraints. It verifies
// a raw token once, before proof verification and the SDK middleware handoff.
func requireAccessToken(options accessTokenOptions) func(http.Handler) http.Handler {
	handler := accessTokenHandler{
		accessTokenOptions: options,
		bearer: auth.RequireBearerToken(cachedTokenVerifier(), &auth.RequireBearerTokenOptions{
			ResourceMetadataURL: options.metadataURL,
			Scopes:              options.scopes,
		}),
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			handler.serve(w, r, next)
		})
	}
}

func (h accessTokenHandler) serve(w http.ResponseWriter, r *http.Request, next http.Handler) {
	ctx, complete := observability.Start(r.Context(), observability.Operation{Phase: observability.PhaseAuthorization, ProviderKind: "mcp"})
	r = r.WithContext(ctx)
	scheme, token, info, ok := verifiedRequest(r, h.verifier)
	if !ok {
		complete(observability.OutcomeDenied)
		writeTokenChallenge(w, h.metadataURL, challengeScheme(scheme), "")
		return
	}
	if acceptsBearer(scheme, info, r) {
		complete(observability.OutcomeSuccess)
		serveVerifiedBearer(w, r, token, info, h.bearer, next)
		return
	}
	if scheme != "DPoP" {
		complete(observability.OutcomeDenied)
		writeTokenChallenge(w, h.metadataURL, "DPoP", "")
		return
	}
	h.serveDPoP(w, r, token, info, next, complete)
}

func verifiedRequest(r *http.Request, verifier auth.TokenVerifier) (string, string, *auth.TokenInfo, bool) {
	scheme, token, ok := requestToken(r.Header.Values("Authorization"))
	if !ok {
		return scheme, token, nil, false
	}
	info, err := verifier(r.Context(), token, r)
	return scheme, token, info, err == nil && info != nil
}

func acceptsBearer(scheme string, info *auth.TokenInfo, r *http.Request) bool {
	requireDPoP, _ := info.Extra["dpop_required"].(bool)
	return scheme == "Bearer" && !requireDPoP && len(r.Header.Values("DPoP")) == 0
}

func serveVerifiedBearer(w http.ResponseWriter, r *http.Request, token string, info *auth.TokenInfo, bearer func(http.Handler) http.Handler, next http.Handler) {
	ctx := context.WithValue(r.Context(), verifiedTokenKey{}, verifiedToken{raw: token, info: info})
	bearer(next).ServeHTTP(w, r.WithContext(ctx))
}

func (h accessTokenHandler) serveDPoP(w http.ResponseWriter, r *http.Request, token string, info *auth.TokenInfo, next http.Handler, complete func(observability.Outcome)) {
	constraint, fingerprint, jkt, ok := dpopConstraint(h.config, info)
	if r.Header.Values("DPoP") == nil || h.replay == nil || !ok {
		complete(observability.OutcomeDenied)
		writeTokenChallenge(w, h.metadataURL, "DPoP", "")
		return
	}
	proof := dpop.Verifier{Config: dpop.Config{Resource: h.config.Resource, MaxAge: seconds(constraint.ProofMaxAgeSeconds), Skew: seconds(constraint.ClockSkewSeconds)}, Store: h.replay}
	nextNonce, err := proof.Verify(r.Context(), r, token, fingerprint, jkt, info.Expiration)
	if writeDPoPFailure(w, h.metadataURL, err) {
		complete(observability.OutcomeDenied)
		return
	}
	authenticatedRequest := r.Clone(context.WithValue(r.Context(), verifiedTokenKey{}, verifiedToken{raw: token, info: info}))
	authenticatedRequest.Header = r.Header.Clone()
	authenticatedRequest.Header.Set("Authorization", "Bearer "+token)
	w.Header().Set("DPoP-Nonce", nextNonce)
	complete(observability.OutcomeSuccess)
	h.bearer(next).ServeHTTP(w, authenticatedRequest)
}

func dpopConstraint(config model.MCP, info *auth.TokenInfo) (*model.SenderConstraint, string, string, bool) {
	required, _ := info.Extra["dpop_required"].(bool)
	authority, authorityOK := info.Extra["authority"].(string)
	fingerprint, fingerprintOK := info.Extra["authority_fingerprint"].(string)
	jkt, jktOK := info.Extra["confirmation_jkt"].(string)
	constraint := config.Authorities[authority].SenderConstraint
	return constraint, fingerprint, jkt, required && authorityOK && authority != "" && fingerprintOK && fingerprint != "" && jktOK && jkt != "" && constraint != nil
}

func writeDPoPFailure(w http.ResponseWriter, metadataURL string, err error) bool {
	if err == nil {
		return false
	}
	var nonceErr *dpop.NonceError
	if errors.As(err, &nonceErr) {
		writeTokenChallenge(w, metadataURL, "DPoP", nonceErr.Value)
	} else {
		writeTokenChallenge(w, metadataURL, "DPoP", "")
	}
	return true
}

func challengeScheme(scheme string) string {
	if scheme == "DPoP" {
		return scheme
	}
	return "Bearer"
}

type verifiedTokenKey struct{}
type verifiedToken struct {
	raw  string
	info *auth.TokenInfo
}

// cachedTokenVerifier hands only this request's verified token to the SDK.
// Reverification here would lose the preceding proof decision and can produce
// a different authority result during key rotation. A missing handoff denies.
func cachedTokenVerifier() auth.TokenVerifier {
	return func(ctx context.Context, raw string, request *http.Request) (*auth.TokenInfo, error) {
		if cached, ok := ctx.Value(verifiedTokenKey{}).(verifiedToken); ok && cached.raw == raw && cached.info != nil {
			return cached.info, nil
		}
		return nil, auth.ErrInvalidToken
	}
}

func requestToken(values []string) (string, string, bool) {
	if len(values) != 1 {
		return "", "", false
	}
	fields := strings.Fields(values[0])
	if len(fields) != 2 || fields[1] == "" {
		return "", "", false
	}
	if strings.EqualFold(fields[0], "bearer") {
		return "Bearer", fields[1], true
	}
	if strings.EqualFold(fields[0], "dpop") {
		return "DPoP", fields[1], true
	}
	return "", "", false
}

func seconds(value int) time.Duration { return time.Duration(value) * time.Second }

func writeTokenChallenge(w http.ResponseWriter, metadataURL, scheme, nonce string) {
	if nonce != "" {
		w.Header().Set("DPoP-Nonce", nonce)
		w.Header().Set("WWW-Authenticate", `DPoP error="use_dpop_nonce", resource_metadata=`+fmt.Sprintf("%q", metadataURL))
	} else {
		w.Header().Set("WWW-Authenticate", scheme+" resource_metadata="+fmt.Sprintf("%q", metadataURL))
	}
	http.Error(w, "invalid access token", http.StatusUnauthorized)
}
