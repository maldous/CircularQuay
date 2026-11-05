// Package dpop verifies the restricted RFC 9449 sender-constraint profile used
// by CQ MCP resources. It never issues access tokens or holds a
// client private key.
package dpop

import (
	"context"
	"crypto"
	"crypto/ecdh"
	"crypto/ecdsa"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	jose "github.com/go-jose/go-jose/v4"
	"github.com/golang-jwt/jwt/v5"
	"github.com/maldous/CircularQuay/internal/jsondocument"
)

const (
	maxProofBytes = 16 << 10
	maxJTIBytes   = 512
)

var (
	// ErrInvalid denotes a rejected proof without exposing token or key data.
	ErrInvalid = errors.New("invalid DPoP proof")
	// ErrNonce identifies a challenge that requires a fresh server nonce.
	ErrNonce = errors.New("DPoP nonce required")
)

// Config is a validated non-secret resource-server policy.
type Config struct {
	// Resource is the policy-selected canonical external URL, never a header.
	Resource string
	// MaxAge bounds the age of an accepted proof at validation.
	MaxAge time.Duration
	// Skew is the explicitly permitted clock difference, bounded by MaxAge.
	Skew time.Duration
}

// NonceError tells the HTTP boundary to issue the RFC 9449 challenge. Its value
// is opaque and must not be logged.
type NonceError struct {
	// Value is the opaque nonce for the client's next proof.
	Value string
}

// Error returns a fixed message without the nonce value.
func (e *NonceError) Error() string { return ErrNonce.Error() }

// Unwrap identifies the nonce challenge without exposing its value.
func (e *NonceError) Unwrap() error { return ErrNonce }

// Store persists replay and nonce decisions. Implementations must make Consume
// atomic across every process that can accept this resource.
type Store interface {
	// Challenge persists a nonce for the exact authority/key/resource binding.
	Challenge(ctx context.Context, binding string, expires time.Time) (string, error)
	// Consume records the replay identifier and replaces a one-use nonce before
	// any caller can authorize the protected operation.
	Consume(ctx context.Context, binding, nonce, replay string, expires time.Time) (string, error)
}

// Verifier validates a proof against an already verified access token.
type Verifier struct {
	Config Config
	Store  Store
	Now    func() time.Time
}

// Verify checks the presented DPoP proof and atomically consumes its nonce and
// proof identifier. It returns the next nonce for the response.
func (v Verifier) Verify(ctx context.Context, request *http.Request, token, authorityFingerprint, tokenJKT string, tokenExpiry time.Time) (string, error) {
	now := v.clock()
	if !v.validInput(ctx, request, token, authorityFingerprint, tokenJKT, tokenExpiry, now) {
		return "", ErrInvalid
	}
	if !requestMatchesResource(request, v.Config.Resource) {
		return "", ErrInvalid
	}
	proof, err := parseProof(request.Header.Values("DPoP"))
	if err != nil || !proof.valid(v.Config, request.Method, token, tokenJKT, now) {
		return "", ErrInvalid
	}
	return v.consume(ctx, proof, authorityFingerprint, tokenJKT, tokenExpiry)
}

func (v Verifier) validInput(ctx context.Context, request *http.Request, token, authorityFingerprint, tokenJKT string, tokenExpiry, now time.Time) bool {
	return ctx != nil && ctx.Err() == nil && request != nil && v.Store != nil && token != "" && validConfig(v.Config) && validJKT(tokenJKT) && authorityFingerprint != "" && !tokenExpiry.IsZero() && now.Before(tokenExpiry)
}

func (v Verifier) consume(ctx context.Context, proof proof, authorityFingerprint, tokenJKT string, tokenExpiry time.Time) (string, error) {
	binding := digest(authorityFingerprint, tokenJKT, v.Config.Resource)
	expires := proofExpiry(proof, v.Config, tokenExpiry)
	if proof.Nonce == "" {
		nonce, issueErr := v.Store.Challenge(ctx, binding, expires)
		if issueErr != nil {
			return "", ErrInvalid
		}
		return "", &NonceError{Value: nonce}
	}
	next, err := v.Store.Consume(ctx, binding, proof.Nonce, digest(binding, proof.ID), expires)
	if err != nil {
		return "", ErrInvalid
	}
	return next, nil
}

func proofExpiry(proof proof, config Config, tokenExpiry time.Time) time.Time {
	expires := time.Unix(proof.IssuedAt, 0).Add(config.MaxAge + config.Skew)
	if tokenExpiry.Before(expires) {
		return tokenExpiry
	}
	return expires
}

func (v Verifier) clock() time.Time {
	if v.Now != nil {
		return v.Now()
	}
	return time.Now()
}

func validConfig(c Config) bool {
	u, err := url.Parse(c.Resource)
	return err == nil && validResourceURL(u, c.Resource) && validTiming(c)
}

func validResourceURL(u *url.URL, raw string) bool {
	return u.String() == raw && validResourceScheme(u) && u.Host != "" && u.User == nil && u.Path != "" && u.Path != "/" && u.RawPath == "" && u.RawQuery == "" && u.Fragment == ""
}

func validResourceScheme(u *url.URL) bool {
	if u.Scheme == "https" {
		return true
	}
	ip := net.ParseIP(u.Hostname())
	return u.Scheme == "http" && (u.Hostname() == "localhost" || ip != nil && ip.IsLoopback())
}

func validTiming(c Config) bool {
	return c.MaxAge >= time.Second && c.MaxAge <= 5*time.Minute && c.Skew >= 0 && c.Skew <= 30*time.Second && c.Skew <= c.MaxAge
}

func requestMatchesResource(request *http.Request, resource string) bool {
	u, err := url.Parse(resource)
	return err == nil && request.URL != nil && request.URL.RawQuery == "" && request.URL.EscapedPath() == u.EscapedPath() && requestOriginMatchesResource(request, u)
}

func requestOriginMatchesResource(request *http.Request, resource *url.URL) bool {
	// The configured URL establishes the public origin. A TLS-terminating local
	// proxy must preserve that Host; forwarded headers never establish it. The
	// outer HTTP boundary separately enforces TLS or its trusted loopback hop.
	if !strings.EqualFold(request.Host, resource.Host) {
		return false
	}
	if request.URL.Host != "" && !strings.EqualFold(request.URL.Host, resource.Host) {
		return false
	}
	return request.URL.Scheme == "" || request.URL.Scheme == resource.Scheme
}

type proof struct {
	ID        string `json:"jti"`
	Method    string `json:"htm"`
	URI       string `json:"htu"`
	IssuedAt  int64  `json:"iat"`
	TokenHash string `json:"ath"`
	Nonce     string `json:"nonce,omitempty"`
	alg       string
	key       crypto.PublicKey
	jkt       string
	signed    string
	signature string
}

type proofHeader struct {
	Type      string          `json:"typ"`
	Algorithm string          `json:"alg"`
	JWK       json.RawMessage `json:"jwk"`
}

func parseProof(values []string) (proof, error) {
	parts, err := proofParts(values)
	if err != nil {
		return proof{}, ErrInvalid
	}
	header, key, jkt, err := proofKey(parts[0])
	if err != nil {
		return proof{}, ErrInvalid
	}
	result, err := proofClaims(parts[1])
	if err != nil {
		return proof{}, ErrInvalid
	}
	result.alg, result.key, result.jkt, result.signed, result.signature = header.Algorithm, key, jkt, parts[0]+"."+parts[1], parts[2]
	if !result.signatureValid() {
		return proof{}, ErrInvalid
	}
	return result, nil
}

func proofParts(values []string) ([]string, error) {
	if len(values) != 1 || len(values[0]) == 0 || len(values[0]) > maxProofBytes {
		return nil, ErrInvalid
	}
	parts := strings.Split(values[0], ".")
	if len(parts) != 3 || parts[0] == "" || parts[1] == "" || parts[2] == "" {
		return nil, ErrInvalid
	}
	return parts, nil
}

func proofKey(encoded string) (proofHeader, crypto.PublicKey, string, error) {
	headerData, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		return proofHeader{}, nil, "", ErrInvalid
	}
	var header proofHeader
	if jsondocument.Decode(headerData, &header) != nil || header.Type != "dpop+jwt" || !allowedAlgorithm(header.Algorithm) {
		return proofHeader{}, nil, "", ErrInvalid
	}
	key, jkt, err := publicJWK(header.JWK, header.Algorithm)
	if err != nil {
		return proofHeader{}, nil, "", ErrInvalid
	}
	return header, key, jkt, nil
}

func proofClaims(encoded string) (proof, error) {
	payload, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		return proof{}, ErrInvalid
	}
	var raw map[string]json.RawMessage
	var result proof
	if jsondocument.Decode(payload, &raw) != nil || json.Unmarshal(payload, &result) != nil || !validClaims(result) {
		return proof{}, ErrInvalid
	}
	return result, nil
}

func validClaims(proof proof) bool {
	return proof.ID != "" && proof.Method != "" && proof.URI != "" && proof.IssuedAt != 0 && proof.TokenHash != "" && validJTI(proof.ID)
}

func (p proof) signatureValid() bool {
	signature, err := base64.RawURLEncoding.DecodeString(p.signature)
	return err == nil && signingMethod(p.alg).Verify(p.signed, signature, p.key) == nil
}

func (p proof) valid(c Config, method, token, jkt string, now time.Time) bool {
	issued := time.Unix(p.IssuedAt, 0)
	lower := now.Add(-c.MaxAge - c.Skew)
	upper := now.Add(c.Skew)
	tokenHash := sha256.Sum256([]byte(token))
	return !now.IsZero() && !issued.Before(lower) && !issued.After(upper) && p.Method == method && p.URI == c.Resource && subtle.ConstantTimeCompare([]byte(p.TokenHash), []byte(base64.RawURLEncoding.EncodeToString(tokenHash[:]))) == 1 && subtle.ConstantTimeCompare([]byte(p.jkt), []byte(jkt)) == 1
}

func signingMethod(algorithm string) jwt.SigningMethod {
	if algorithm == "ES256" {
		return jwt.SigningMethodES256
	}
	return jwt.SigningMethodRS256
}

func allowedAlgorithm(algorithm string) bool { return algorithm == "ES256" || algorithm == "RS256" }

func validJTI(value string) bool {
	if len(value) < 16 || len(value) > maxJTIBytes {
		return false
	}
	for _, r := range value {
		if r < '!' || r > '~' || r == '"' || r == '\\' {
			return false
		}
	}
	return true
}

func validJKT(value string) bool {
	decoded, err := base64.RawURLEncoding.DecodeString(value)
	return err == nil && len(decoded) == sha256.Size && base64.RawURLEncoding.EncodeToString(decoded) == value
}

func publicJWK(raw json.RawMessage, algorithm string) (crypto.PublicKey, string, error) {
	var fields map[string]json.RawMessage
	if jsondocument.Decode(raw, &fields) != nil || fields == nil || hasPrivateJWK(fields) {
		return nil, "", ErrInvalid
	}
	var key jose.JSONWebKey
	if json.Unmarshal(raw, &key) != nil || !key.Valid() || !key.IsPublic() || !algorithmKeyMatches(algorithm, key.Key) {
		return nil, "", ErrInvalid
	}
	thumbprint, err := key.Thumbprint(crypto.SHA256)
	if err != nil || len(thumbprint) != sha256.Size {
		return nil, "", ErrInvalid
	}
	return key.Key, base64.RawURLEncoding.EncodeToString(thumbprint), nil
}

func algorithmKeyMatches(algorithm string, key any) bool {
	switch value := key.(type) {
	case *ecdsa.PublicKey:
		return algorithm == "ES256" && validES256Key(value)
	case *rsa.PublicKey:
		return algorithm == "RS256" && validRS256Key(value)
	default:
		return false
	}
}

func validES256Key(key *ecdsa.PublicKey) bool {
	if key == nil || key.Curve == nil || key.Curve.Params().Name != "P-256" {
		return false
	}
	encoded, err := key.Bytes()
	if err != nil {
		return false
	}
	_, err = ecdh.P256().NewPublicKey(encoded)
	return err == nil
}

func validRS256Key(key *rsa.PublicKey) bool {
	return key != nil && key.N != nil && key.N.BitLen() >= 2048 && key.N.BitLen() <= 8192 && key.E >= 3 && key.E%2 == 1
}

func hasPrivateJWK(fields map[string]json.RawMessage) bool {
	for _, name := range []string{"d", "k", "p", "q", "dp", "dq", "qi", "oth"} {
		if _, found := fields[name]; found {
			return true
		}
	}
	return false
}

func digest(values ...string) string {
	h := sha256.New()
	for _, value := range values {
		_, _ = h.Write([]byte(value))
		_, _ = h.Write([]byte{0})
	}
	return base64.RawURLEncoding.EncodeToString(h.Sum(nil))
}
