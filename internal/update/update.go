// Package update implements bounded release awareness. It never downloads,
// installs, executes, or activates an update.
package update

import (
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"time"

	"github.com/maldous/CircularQuay/internal/fault"
	"github.com/maldous/CircularQuay/internal/jsondocument"
	"github.com/maldous/CircularQuay/internal/model"
	"github.com/maldous/CircularQuay/internal/platform"
	"github.com/maldous/CircularQuay/internal/transport"
	"golang.org/x/mod/semver"
)

const (
	defaultFeedEndpoint = "https://api.github.com/repos/maldous/CircularQuay/releases"
	maximumFeedBytes    = 1 << 20
	cacheName           = "release-awareness.json"
	releaseResponse     = "release awareness response"
)

// Options supplies process-local dependencies. StateDirectory is a protected
// cquay state directory; it is never derived from an environment variable.
// client is test-only. Production callers cannot substitute the configured
// no-proxy, redirect-denying, trust-bound transport.
type Options struct {
	Version        string
	StateDirectory string
	Now            func() time.Time
	client         *http.Client
}

// Result contains public release metadata only. It never contains feed URLs,
// credentials, response bodies, or unvalidated provider links.
type Result struct {
	Status         string    `json:"status"`
	Channel        string    `json:"channel,omitempty"`
	CurrentVersion string    `json:"current_version,omitempty"`
	Available      string    `json:"available_version,omitempty"`
	InstructionURL string    `json:"instruction_url,omitempty"`
	CheckedAt      time.Time `json:"checked_at,omitempty"`
	Cached         bool      `json:"cached,omitempty"`
}

type cache struct {
	CheckedAt time.Time `json:"checked_at"`
	Channel   string    `json:"channel"`
	Config    string    `json:"config"`
	Status    string    `json:"status"`
	Available string    `json:"available,omitempty"`
}

type release struct {
	TagName    string `json:"tag_name"`
	Draft      bool   `json:"draft"`
	Prerelease bool   `json:"prerelease"`
}

// EnforceMinimum compares complete SemVer values using SemVer precedence, which
// intentionally ignores build metadata. It is a local policy floor rather
// than evidence that the executable was obtained from a signed release.
// A prerelease cannot satisfy a stable production minimum. An explicitly
// prerelease minimum permits its SemVer prerelease progression.
// An empty minimum does not constrain execution.
func EnforceMinimum(version, minimum string) error {
	if minimum == "" {
		return nil
	}
	running, runningOK := completeVersion(version)
	declared, declaredOK := completeVersion(minimum)
	if !runningOK || !declaredOK || prereleaseBelowStableFloor(running, declared) || semver.Compare(running, declared) < 0 {
		return fault.New(fault.Denied, "minimum runtime version")
	}
	return nil
}

// Check obtains the newest eligible release from the configured HTTPS feed.
// A fresh protected cache avoids network work. Callers decide whether a failed
// advisory check is displayed or returned as an explicit command failure.
func Check(ctx context.Context, config *model.Updates, options Options) (Result, error) {
	result := Result{Status: "disabled", CurrentVersion: options.Version}
	if config == nil || config.Check == "never" {
		return result, nil
	}
	if !validInteractiveConfig(*config) || !validVersion(options.Version) {
		return Result{}, fault.New(fault.Invalid, "release awareness configuration")
	}
	endpoint, err := feedEndpoint(config.FeedEndpoint)
	if err != nil {
		return Result{}, err
	}
	identity := configurationFingerprint(endpoint, *config)
	now := options.now()
	if cached, ok := readFreshCache(config, options, now, identity); ok {
		return resultFromCache(options.Version, *config, cached, true), nil
	}
	client, err := configuredClient(*config, options.client)
	if err != nil {
		return Result{}, err
	}
	available, status, err := fetch(ctx, client, endpoint, *config)
	if err != nil {
		return Result{}, err
	}
	entry := cache{CheckedAt: now, Channel: config.Channel, Config: identity, Status: status, Available: available}
	if err := writeCache(options.StateDirectory, entry); err != nil {
		return Result{}, err
	}
	return resultFromCache(options.Version, *config, entry, false), nil
}

func (o Options) now() time.Time {
	if o.Now != nil {
		return o.Now().UTC()
	}
	return time.Now().UTC()
}

func readFreshCache(config *model.Updates, options Options, now time.Time, identity string) (cache, bool) {
	if options.StateDirectory == "" {
		return cache{}, false
	}
	data, err := platform.ReadProtected(filepath.Join(options.StateDirectory, cacheName), true)
	if err != nil || len(data) == 0 || len(data) > maximumFeedBytes {
		return cache{}, false
	}
	var entry cache
	if jsondocument.Decode(data, &entry) != nil || entry.Channel != config.Channel || entry.Config != identity || !validCache(entry, now, config.IntervalHours) || !validCachedStatus(entry.Status) {
		return cache{}, false
	}
	if entry.Available != "" && !validTag(entry.Available) {
		return cache{}, false
	}
	return entry, true
}

func validCachedStatus(status string) bool {
	return status == "current" || status == "no_release" || status == "no_eligible_release"
}

func validCache(entry cache, now time.Time, intervalHours int) bool {
	if entry.CheckedAt.IsZero() || intervalHours < 1 || entry.CheckedAt.After(now) {
		return false
	}
	return now.Sub(entry.CheckedAt) < time.Duration(intervalHours)*time.Hour
}

func writeCache(directory string, entry cache) error {
	if directory == "" {
		return fault.New(fault.Invalid, "release awareness state")
	}
	data, err := json.Marshal(entry)
	if err != nil || len(data) > maximumFeedBytes {
		return fault.New(fault.Apply, "release awareness cache")
	}
	return platform.WriteProtected(filepath.Join(directory, cacheName), data)
}

func resultFromCache(version string, config model.Updates, entry cache, cached bool) Result {
	result := Result{Status: entry.Status, Channel: config.Channel, CurrentVersion: version, CheckedAt: entry.CheckedAt, Cached: cached}
	if entry.Available == "" {
		return result
	}
	result.Available = entry.Available
	if current, ok := completeVersion(version); ok && semver.Compare(entry.Available, current) > 0 {
		result.Status = "available"
		result.InstructionURL = instructionURL(entry.Available)
	}
	return result
}

func configuredClient(config model.Updates, supplied *http.Client) (*http.Client, error) {
	if supplied != nil {
		copyClient := *supplied
		copyClient.CheckRedirect = transport.DenyRedirect
		copyClient.Timeout = time.Duration(config.TimeoutSeconds) * time.Second
		return &copyClient, nil
	}
	var client *http.Client
	var err error
	if config.TrustStoreFile != "" {
		client, err = transport.TrustStoreHTTPClientNoProxy(config.TrustStoreFile, config.TrustStoreSHA256)
		if err != nil {
			return nil, err
		}
	} else {
		client = transport.NewHTTPClient(releaseFeedTransport())
	}
	client.Timeout = time.Duration(config.TimeoutSeconds) * time.Second
	return client, nil
}

func releaseFeedTransport() *http.Transport {
	dialer := &net.Dialer{Timeout: 10 * time.Second, KeepAlive: 30 * time.Second}
	return &http.Transport{
		Proxy:                 nil,
		DialContext:           dialer.DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: 10 * time.Second,
		ExpectContinueTimeout: time.Second,
		TLSClientConfig:       &tls.Config{MinVersion: tls.VersionTLS12},
	}
}

func feedEndpoint(raw string) (*url.URL, error) {
	if raw == "" {
		raw = defaultFeedEndpoint
	}
	endpoint, err := url.Parse(raw)
	if err != nil || endpoint.Scheme != "https" || endpoint.Hostname() == "" || endpoint.User != nil || endpoint.RawQuery != "" || endpoint.Fragment != "" || endpoint.Opaque != "" || strings.ContainsAny(raw, "\x00\r\n") {
		return nil, fault.New(fault.Invalid, "release awareness endpoint")
	}
	return endpoint, nil
}

func fetch(ctx context.Context, client *http.Client, endpoint *url.URL, config model.Updates) (string, string, error) {
	request, err := feedRequest(ctx, endpoint, config.TokenFile)
	if err != nil {
		return "", "", err
	}
	releases, err := requestReleases(client, request)
	if err != nil {
		return "", "", err
	}
	available, status, err := selectRelease(releases, config.Channel)
	if err != nil {
		return "", "", err
	}
	return available, status, nil
}

func feedRequest(ctx context.Context, endpoint *url.URL, tokenFile string) (*http.Request, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return nil, fault.New(fault.Invalid, "release awareness request")
	}
	request.Header.Set("Accept", "application/vnd.github+json")
	if tokenFile == "" {
		return request, nil
	}
	token, err := transport.ReadCredentialFile(tokenFile)
	if err != nil {
		return nil, err
	}
	request.Header.Set("Authorization", "Bearer "+token)
	return request, nil
}

func requestReleases(client *http.Client, request *http.Request) ([]release, error) {
	response, err := client.Do(request)
	if err != nil {
		return nil, fault.New(fault.Unavailable, "release awareness feed")
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return nil, fault.New(fault.Unavailable, "release awareness feed")
	}
	return decodeReleases(response.Body)
}

func decodeReleases(body io.Reader) ([]release, error) {
	data, err := io.ReadAll(io.LimitReader(body, maximumFeedBytes+1))
	if err != nil || len(data) > maximumFeedBytes {
		return nil, fault.New(fault.Trust, releaseResponse)
	}
	var releases []release
	decoder := json.NewDecoder(bytes.NewReader(data))
	if decoder.Decode(&releases) != nil || decoder.Decode(&struct{}{}) != io.EOF {
		return nil, fault.New(fault.Trust, releaseResponse)
	}
	return releases, nil
}

func selectRelease(releases []release, channel string) (string, string, error) {
	best := ""
	published := false
	eligible := false
	for _, candidate := range releases {
		if candidate.Draft {
			continue
		}
		published = true
		version, allowed, err := eligibleRelease(candidate, channel)
		if err != nil {
			return "", "", err
		}
		if !allowed {
			continue
		}
		eligible = true
		best = newerRelease(best, version)
	}
	return releaseSelection(best, published, eligible)
}

func eligibleRelease(candidate release, channel string) (string, bool, error) {
	if channel == "stable" && candidate.Prerelease {
		return "", false, nil
	}
	version, ok := canonicalTag(candidate.TagName)
	if !ok {
		return "", false, fault.New(fault.Trust, releaseResponse)
	}
	if channel == "stable" && semver.Prerelease(version) != "" {
		return "", false, nil
	}
	return version, true, nil
}

func newerRelease(current, candidate string) string {
	if current == "" || semver.Compare(candidate, current) > 0 {
		return candidate
	}
	return current
}

func releaseSelection(best string, published, eligible bool) (string, string, error) {
	if !published {
		return "", "no_release", nil
	}
	if !eligible {
		return "", "no_eligible_release", nil
	}
	return best, "current", nil
}

func canonicalVersion(value string) (string, bool) {
	if !strings.HasPrefix(value, "v") {
		value = "v" + value
	}
	canonical := semver.Canonical(value)
	return canonical, canonical != "" && canonical == value
}

func completeVersion(value string) (string, bool) {
	if !strings.HasPrefix(value, "v") {
		value = "v" + value
	}
	canonical := semver.Canonical(value)
	base := strings.TrimSuffix(value, semver.Build(value))
	return value, canonical != "" && canonical == base
}

func prereleaseBelowStableFloor(running, minimum string) bool {
	return semver.Prerelease(minimum) == "" && semver.Prerelease(running) != ""
}

func validVersion(value string) bool {
	_, ok := completeVersion(value)
	return ok
}

func canonicalTag(value string) (string, bool) {
	if !strings.HasPrefix(value, "v") {
		return "", false
	}
	return canonicalVersion(value)
}

func validTag(value string) bool {
	_, ok := canonicalTag(value)
	return ok
}

func validInteractiveConfig(config model.Updates) bool {
	return config.Check == "interactive" && (config.Channel == "stable" || config.Channel == "release-candidate") && config.IntervalHours >= 1 && config.IntervalHours <= 720 && config.TimeoutSeconds >= 1 && config.TimeoutSeconds <= 30 && ((config.TrustStoreFile == "") == (config.TrustStoreSHA256 == ""))
}

func configurationFingerprint(endpoint *url.URL, config model.Updates) string {
	value := endpoint.String() + "\x00" + config.TokenFile + "\x00" + config.TrustStoreFile + "\x00" + config.TrustStoreSHA256
	digest := sha256.Sum256([]byte(value))
	return hex.EncodeToString(digest[:])
}

func instructionURL(version string) string {
	return "https://github.com/maldous/CircularQuay/releases/tag/" + version
}
