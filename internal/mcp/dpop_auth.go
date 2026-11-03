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
	"github.com/modelcontextprotocol/go-sdk/auth"
)

// requireAccessToken accepts Bearer tokens for ordinary authorities and DPoP
// only for authorities that explicitly select sender constraints. It verifies
// a raw token once, before proof verification and the SDK middleware handoff.
func requireAccessToken(verifier auth.TokenVerifier, config model.MCP, replay dpop.Store, metadataURL string, scopes []string) func(http.Handler) http.Handler {
	bearer := auth.RequireBearerToken(cachedTokenVerifier(), &auth.RequireBearerTokenOptions{ResourceMetadataURL: metadataURL, Scopes: scopes})
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			handleAccessToken(w, r, verifier, bearer, next, config, replay, metadataURL)
		})
	}
}

func handleAccessToken(w http.ResponseWriter, r *http.Request, verifier auth.TokenVerifier, bearer func(http.Handler) http.Handler, next http.Handler, config model.MCP, replay dpop.Store, metadataURL string) {
	scheme, token, info, ok := verifiedRequest(r, verifier)
	if !ok {
		writeTokenChallenge(w, metadataURL, challengeScheme(scheme), "")
		return
	}
	if acceptsBearer(scheme, info, r) {
		serveVerifiedBearer(w, r, token, info, bearer, next)
		return
	}
	if scheme != "DPoP" {
		writeTokenChallenge(w, metadataURL, "DPoP", "")
		return
	}
	serveVerifiedDPoP(w, r, token, info, bearer, next, config, replay, metadataURL)
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

func serveVerifiedDPoP(w http.ResponseWriter, r *http.Request, token string, info *auth.TokenInfo, bearer func(http.Handler) http.Handler, next http.Handler, config model.MCP, replay dpop.Store, metadataURL string) {
	constraint, fingerprint, jkt, ok := dpopConstraint(config, info)
	if r.Header.Values("DPoP") == nil || replay == nil || !ok {
		writeTokenChallenge(w, metadataURL, "DPoP", "")
		return
	}
	proof := dpop.Verifier{Config: dpop.Config{Resource: config.Resource, MaxAge: seconds(constraint.ProofMaxAgeSeconds), Skew: seconds(constraint.ClockSkewSeconds)}, Store: replay}
	nextNonce, err := proof.Verify(r.Context(), r, token, fingerprint, jkt, info.Expiration)
	if writeDPoPFailure(w, metadataURL, err) {
		return
	}
	copy := r.Clone(context.WithValue(r.Context(), verifiedTokenKey{}, verifiedToken{raw: token, info: info}))
	copy.Header = r.Header.Clone()
	copy.Header.Set("Authorization", "Bearer "+token)
	w.Header().Set("DPoP-Nonce", nextNonce)
	bearer(next).ServeHTTP(w, copy)
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
