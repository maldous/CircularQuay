package update

import (
	"context"
	"crypto/tls"
	"errors"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/maldous/CircularQuay/internal/fault"
	"github.com/maldous/CircularQuay/internal/model"
	"github.com/maldous/CircularQuay/internal/platform"
)

type roundTrip func(*http.Request) (*http.Response, error)

func (fn roundTrip) RoundTrip(request *http.Request) (*http.Response, error) { return fn(request) }

func TestCheckUsesProtectedCacheAndCanonicalInstruction(t *testing.T) {
	directory := updateStateDirectory(t)
	now := time.Date(2026, time.November, 4, 0, 0, 0, 0, time.UTC)
	requests := 0
	client := &http.Client{Transport: roundTrip(func(request *http.Request) (*http.Response, error) {
		requests++
		if request.URL.String() != "https://updates.example.test/releases" {
			t.Fatalf("unexpected endpoint %q", request.URL)
		}
		return releaseResponse(`[{"tag_name":"v1.2.0","draft":false,"prerelease":false}]`), nil
	})}
	config := interactiveConfig()
	options := Options{Version: "1.0.0", StateDirectory: directory, Now: func() time.Time { return now }, client: client}

	first, err := Check(context.Background(), config, options)
	if err != nil {
		t.Fatal(err)
	}
	if first.Status != "available" || first.Cached || first.InstructionURL != "https://github.com/maldous/CircularQuay/releases/tag/v1.2.0" {
		t.Fatalf("unexpected first result: %#v", first)
	}
	second, err := Check(context.Background(), config, options)
	if err != nil {
		t.Fatal(err)
	}
	if !second.Cached || requests != 1 {
		t.Fatalf("cache was not used: %#v requests=%d", second, requests)
	}
	data, err := platform.ReadProtected(filepath.Join(directory, cacheName), true)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "updates.example.test") || strings.Contains(string(data), "Authorization") {
		t.Fatalf("cache stored transport material: %q", data)
	}
}

func TestCheckRejectsMalformedResponseWithoutCredentialDisclosure(t *testing.T) {
	directory := updateStateDirectory(t)
	token := filepath.Join(directory, "feed-token")
	if err := platform.CreateProtectedExclusive(token, []byte("synthetic-token-value")); err != nil {
		t.Fatal(err)
	}
	config := interactiveConfig()
	config.TokenFile = token
	config.FeedEndpoint = "https://updates.example.test/releases"
	client := &http.Client{Transport: roundTrip(func(request *http.Request) (*http.Response, error) {
		if request.Header.Get("Authorization") != "Bearer synthetic-token-value" {
			t.Fatal("protected token was not supplied as a bearer credential")
		}
		return releaseResponse(`[{"tag_name":"v1.2.0"} trailing`), nil
	})}

	_, err := Check(context.Background(), config, Options{Version: "1.0.0", StateDirectory: directory, client: client})
	if !fault.Is(err, fault.Trust) || strings.Contains(err.Error(), "synthetic-token-value") {
		t.Fatalf("malformed feed error was unsafe: %v", err)
	}
}

func TestCheckRejectsNonCanonicalTagButAllowsNoEligibleStableRelease(t *testing.T) {
	config := interactiveConfig()
	for _, body := range []string{`[{"tag_name":"1.2.0"}]`, `[{"tag_name":"v1.2"}]`, `[{"tag_name":"v1.2.0+build.1"}]`} {
		client := &http.Client{Transport: roundTrip(func(*http.Request) (*http.Response, error) { return releaseResponse(body), nil })}
		if _, err := Check(context.Background(), config, Options{Version: "1.0.0", StateDirectory: updateStateDirectory(t), client: client}); !fault.Is(err, fault.Trust) {
			t.Fatalf("invalid release metadata accepted: %v", err)
		}
	}
	client := &http.Client{Transport: roundTrip(func(*http.Request) (*http.Response, error) {
		return releaseResponse(`[{"tag_name":"v1.2.0-rc.1","prerelease":true}]`), nil
	})}
	result, err := Check(context.Background(), config, Options{Version: "1.0.0", StateDirectory: updateStateDirectory(t), client: client})
	if err != nil || result.Status != "no_eligible_release" || result.Available != "" {
		t.Fatalf("no eligible stable release was not reported correctly: %#v %v", result, err)
	}
	config.Channel = "release-candidate"
	result, err = Check(context.Background(), config, Options{Version: "1.0.0", StateDirectory: updateStateDirectory(t), client: client})
	if err != nil || result.Status != "available" || result.Available != "v1.2.0-rc.1" {
		t.Fatalf("release-candidate channel did not select its eligible release: %#v %v", result, err)
	}
}

func TestCheckDistinguishesNoPublishedReleaseAndInvalidatesChangedFeedCache(t *testing.T) {
	directory := updateStateDirectory(t)
	now := time.Date(2026, time.November, 4, 0, 0, 0, 0, time.UTC)
	requests := 0
	client := &http.Client{Transport: roundTrip(func(*http.Request) (*http.Response, error) {
		requests++
		return releaseResponse(`[]`), nil
	})}
	config := interactiveConfig()
	options := Options{Version: "1.0.0", StateDirectory: directory, Now: func() time.Time { return now }, client: client}
	result, err := Check(context.Background(), config, options)
	if err != nil || result.Status != "no_release" {
		t.Fatalf("empty feed result: %#v %v", result, err)
	}
	config.FeedEndpoint = "https://mirror.example.test/releases"
	if _, err := Check(context.Background(), config, options); err != nil || requests != 2 {
		t.Fatalf("feed change reused cached response: requests=%d err=%v", requests, err)
	}
}

func TestCheckIgnoresAmbiguousCacheDocument(t *testing.T) {
	directory := updateStateDirectory(t)
	config := interactiveConfig()
	endpoint, err := feedEndpoint(config.FeedEndpoint)
	if err != nil {
		t.Fatal(err)
	}
	entry := `{"checked_at":"2026-11-04T00:00:00Z","channel":"stable","channel":"stable","config":"` + configurationFingerprint(endpoint, *config) + `","status":"current"}`
	if err := platform.WriteProtected(filepath.Join(directory, cacheName), []byte(entry)); err != nil {
		t.Fatal(err)
	}
	requests := 0
	client := &http.Client{Transport: roundTrip(func(*http.Request) (*http.Response, error) {
		requests++
		return releaseResponse(`[]`), nil
	})}
	_, err = Check(context.Background(), config, Options{Version: "1.0.0", StateDirectory: directory, Now: func() time.Time { return time.Date(2026, time.November, 4, 1, 0, 0, 0, time.UTC) }, client: client})
	if err != nil || requests != 1 {
		t.Fatalf("ambiguous cache was accepted: requests=%d err=%v", requests, err)
	}
}

func TestMinimumVersionAppliesWithoutNetworkConfiguration(t *testing.T) {
	for _, candidate := range [][2]string{{"1.0.0", "v1.0.0"}, {"1.0.0+build.7", "v1.0.0"}, {"1.0.0", "v1.0.0+build.7"}, {"1.0.0-rc.2", "v1.0.0-rc.1"}, {"999.0.0-dev.1", "v1.0.0-rc.1"}} {
		if err := EnforceMinimum(candidate[0], candidate[1]); err != nil {
			t.Fatalf("valid floor %q >= %q denied: %v", candidate[0], candidate[1], err)
		}
	}
	for _, candidate := range []string{"1.0.0-alpha.1", "1.0.0-rc.2", "1.0.0-dev", "999.0.0-alpha.1", "999.0.0-rc.1", "999.0.0-dev.1", "0.9.9", "1.0", "not-a-version"} {
		if err := EnforceMinimum(candidate, "1.0.0"); !fault.Is(err, fault.Denied) {
			t.Fatalf("minimum accepted %q: %v", candidate, err)
		}
	}
}

func TestCheckRejectsUnsafeEndpointBeforeNetwork(t *testing.T) {
	config := interactiveConfig()
	config.FeedEndpoint = "http://updates.example.test/releases"
	client := &http.Client{Transport: roundTrip(func(*http.Request) (*http.Response, error) {
		return nil, errors.New("network should not be called")
	})}
	_, err := Check(context.Background(), config, Options{Version: "1.0.0", StateDirectory: updateStateDirectory(t), client: client})
	if !fault.Is(err, fault.Invalid) {
		t.Fatalf("unsafe endpoint accepted: %v", err)
	}
}

func TestReleaseFeedTransportDoesNotInheritDefaultTransportTLS(t *testing.T) {
	original := http.DefaultTransport
	poisoned := &http.Transport{TLSClientConfig: &tls.Config{ServerName: "poisoned.example.test", Certificates: []tls.Certificate{{}}}}
	http.DefaultTransport = poisoned
	t.Cleanup(func() { http.DefaultTransport = original })

	transport := releaseFeedTransport()
	if transport == poisoned || transport.Proxy != nil || transport.DialContext == nil || transport.ResponseHeaderTimeout != 10*time.Second || transport.TLSHandshakeTimeout != 10*time.Second || transport.TLSClientConfig == nil || transport.TLSClientConfig.RootCAs != nil || transport.TLSClientConfig.ServerName != "" || len(transport.TLSClientConfig.Certificates) != 0 || transport.TLSClientConfig.MinVersion != tls.VersionTLS12 {
		t.Fatalf("release feed transport inherited unsafe default transport state: %#v", transport.TLSClientConfig)
	}
	client, err := configuredClient(*interactiveConfig(), nil)
	if err != nil || client.Transport == poisoned {
		t.Fatalf("configured client reused global default transport: %v", err)
	}
}

func interactiveConfig() *model.Updates {
	return &model.Updates{Check: "interactive", Channel: "stable", IntervalHours: 24, TimeoutSeconds: 5, FeedEndpoint: "https://updates.example.test/releases"}
}

func updateStateDirectory(t *testing.T) string {
	t.Helper()
	directory := filepath.Join(t.TempDir(), "state")
	if err := platform.EnsurePrivateDirectory(directory); err != nil {
		t.Fatal(err)
	}
	return directory
}

func releaseResponse(body string) *http.Response {
	return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}
}
