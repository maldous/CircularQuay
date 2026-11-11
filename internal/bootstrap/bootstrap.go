// Package bootstrap retrieves a checksum-pinned JSON configuration artifact.
//
// It accepts only immutable GitHub tree, blob, or raw URLs whose full commit
// marker equals the supplied SHA. It does not resolve branches, use GitHub
// credentials, write files, or activate a retrieved policy.
package bootstrap

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/maldous/CircularQuay/internal/fault"
	"github.com/maldous/CircularQuay/internal/jsondocument"
	"github.com/maldous/CircularQuay/internal/transport"
)

const (
	maxArtifactSize = 1 << 20
	fetchTimeout    = 30 * time.Second
)

// Request identifies one immutable, checksum-pinned GitHub configuration.
// ExpectedSHA256 is mandatory because this helper has no interactive trust
// decision or branch-resolution authority.
type Request struct {
	SourceURL      string
	ConfigRef      string
	ConfigPath     string
	ExpectedSHA256 string
}

// Result contains only normalized source metadata and the verified artifact.
// CanonicalSourceURL always identifies the immutable raw.githubusercontent.com
// representation of the accepted GitHub source.
type Result struct {
	CanonicalSourceURL string
	ConfigRef          string
	ConfigPath         string
	Bytes              []byte
	SHA256             string
}

// Fetcher retrieves immutable configuration without retaining cookies,
// authorization headers, redirect behavior, or timeout settings from client.
// A non-nil client contributes only its RoundTripper for synthetic tests or a
// caller-owned trusted network route.
type Fetcher struct{ client *http.Client }

// NewFetcher constructs a credential-free, redirect-denying fetcher. Passing
// nil uses the existing hardened default transport.
func NewFetcher(client *http.Client) *Fetcher {
	var base http.RoundTripper
	if client != nil {
		base = client.Transport
	}
	return &Fetcher{client: transport.NewHTTPClient(base)}
}

// Fetch retrieves and verifies one strict JSON object. It performs no
// filesystem access or policy parsing beyond JSON boundary validation.
func (f *Fetcher) Fetch(ctx context.Context, request Request) (Result, error) {
	if ctx.Err() != nil {
		return Result{}, fault.New(fault.Apply, "github_config_canceled")
	}
	canonical, err := normalize(request)
	if err != nil {
		return Result{}, err
	}
	ctx, cancel := context.WithTimeout(ctx, fetchTimeout)
	defer cancel()
	response, err := f.get(ctx, canonical)
	if err != nil {
		return Result{}, err
	}
	defer response.Body.Close()
	data, err := readArtifact(response)
	if err != nil {
		return Result{}, err
	}
	digest := sha256.Sum256(data)
	actual := hex.EncodeToString(digest[:])
	if actual != request.ExpectedSHA256 {
		return Result{}, fault.New(fault.Trust, "github_config_digest")
	}
	return Result{CanonicalSourceURL: canonical, ConfigRef: request.ConfigRef, ConfigPath: request.ConfigPath, Bytes: data, SHA256: actual}, nil
}

func (f *Fetcher) get(ctx context.Context, canonical string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, canonical, nil)
	if err != nil {
		return nil, fault.New(fault.Invalid, "github_config_request")
	}
	response, err := f.httpClient().Do(req)
	if err == nil {
		return response, nil
	}
	if ctx.Err() != nil || errors.Is(err, context.DeadlineExceeded) {
		return nil, fault.New(fault.Unavailable, "github_config_timeout")
	}
	return nil, fault.New(fault.Trust, "github_config_transport")
}

func readArtifact(response *http.Response) ([]byte, error) {
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return nil, transport.Status(response.StatusCode)
	}
	if response.ContentLength > maxArtifactSize {
		return nil, fault.New(fault.Trust, "github_config_size")
	}
	data, err := io.ReadAll(io.LimitReader(response.Body, maxArtifactSize+1))
	if err != nil || len(data) > maxArtifactSize {
		return nil, fault.New(fault.Trust, "github_config_artifact")
	}
	if !jsonObject(data) {
		return nil, fault.New(fault.Trust, "github_config_json")
	}
	return data, nil
}

func (f *Fetcher) httpClient() *http.Client {
	if f == nil || f.client == nil {
		return transport.NewHTTPClient(nil)
	}
	return f.client
}

func normalize(request Request) (string, error) {
	if len(request.SourceURL) > 8192 || len(request.ConfigPath) > 4096 {
		return "", fault.New(fault.Invalid, "github_config_location_size")
	}
	if !fullLowerHex(request.ConfigRef, 40) || !fullLowerHex(request.ExpectedSHA256, 64) {
		return "", fault.New(fault.Invalid, "github_config_pin")
	}
	path, segments, err := safeConfigPath(request.ConfigPath)
	if err != nil {
		return "", err
	}
	u, err := url.Parse(request.SourceURL)
	if err != nil || !validSourceURL(u) {
		return "", fault.New(fault.Invalid, "github_config_url")
	}
	parts := strings.Split(strings.TrimPrefix(u.Path, "/"), "/")
	owner, repository, err := repositoryParts(parts)
	if err != nil {
		return "", err
	}
	if !sourceMatches(u.Host, parts, request.ConfigRef, segments) {
		return "", fault.New(fault.Invalid, "github_config_reference")
	}
	return "https://raw.githubusercontent.com/" + owner + "/" + repository + "/" + request.ConfigRef + "/" + path, nil
}

func validSourceURL(u *url.URL) bool {
	return u != nil && u.Scheme == "https" && u.User == nil && u.RawQuery == "" && u.Fragment == "" && u.Opaque == "" && u.RawPath == "" && !u.ForceQuery
}

func repositoryParts(parts []string) (string, string, error) {
	if len(parts) < 4 || !validRepositoryPart(parts[0]) || !validRepositoryPart(parts[1]) {
		return "", "", fault.New(fault.Invalid, "github_config_repository")
	}
	if parts[0] == "." || parts[0] == ".." || parts[1] == "." || parts[1] == ".." {
		return "", "", fault.New(fault.Invalid, "github_config_repository")
	}
	return parts[0], parts[1], nil
}

func sourceMatches(host string, parts []string, ref string, path []string) bool {
	switch host {
	case "github.com":
		return githubSourceMatches(parts, ref, path)
	case "raw.githubusercontent.com":
		return rawSourceMatches(parts, ref, path)
	default:
		return false
	}
}

func githubSourceMatches(parts []string, ref string, path []string) bool {
	if len(parts) == 4 && parts[2] == "tree" {
		return parts[3] == ref
	}
	return len(parts) == len(path)+4 && parts[2] == "blob" && parts[3] == ref && equalPath(parts[4:], path)
}

func rawSourceMatches(parts []string, ref string, path []string) bool {
	return len(parts) == len(path)+3 && parts[2] == ref && equalPath(parts[3:], path)
}

func safeConfigPath(value string) (string, []string, error) {
	if value == "" || strings.HasPrefix(value, "/") || strings.HasSuffix(value, "/") || strings.ContainsAny(value, "\\%?#\x00\r\n") {
		return "", nil, fault.New(fault.Invalid, "github_config_path")
	}
	parts := strings.Split(value, "/")
	for _, part := range parts {
		if part == "" || part == "." || part == ".." || !validRepositoryPart(part) {
			return "", nil, fault.New(fault.Invalid, "github_config_path")
		}
	}
	return value, parts, nil
}

func validRepositoryPart(value string) bool {
	if !validRepositoryLength(value) || !asciiAlphaNumeric(value[0]) || !asciiAlphaNumeric(value[len(value)-1]) {
		return false
	}
	for _, character := range value {
		if !validRepositoryCharacter(character) {
			return false
		}
	}
	return true
}

func validRepositoryLength(value string) bool { return len(value) > 0 && len(value) <= 100 }

func validRepositoryCharacter(character rune) bool {
	return (character >= 'a' && character <= 'z') || (character >= 'A' && character <= 'Z') || (character >= '0' && character <= '9') || strings.ContainsRune("._-", character)
}

func asciiAlphaNumeric(character byte) bool {
	return character >= 'a' && character <= 'z' || character >= 'A' && character <= 'Z' || character >= '0' && character <= '9'
}

func fullLowerHex(value string, size int) bool {
	if len(value) != size {
		return false
	}
	for _, character := range value {
		if !(character >= '0' && character <= '9' || character >= 'a' && character <= 'f') {
			return false
		}
	}
	return true
}

func equalPath(first, second []string) bool {
	for index := range first {
		if first[index] != second[index] {
			return false
		}
	}
	return true
}

func jsonObject(data []byte) bool {
	var object map[string]any
	return jsondocument.Decode(data, &object) == nil && object != nil
}
