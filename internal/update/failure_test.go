package update

import (
	"context"
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

func TestDisabledReleaseChecksDoNotContactFeed(t *testing.T) {
	for _, config := range []*model.Updates{nil, {Check: "never"}} {
		options := Options{Version: "development", client: noFeedRequests(t)}
		result, err := Check(context.Background(), config, options)
		if err != nil || result.Status != "disabled" {
			t.Fatalf("disabled check did not remain local: %#v %v", result, err)
		}
	}
}

func TestInvalidReleaseConfigurationIsDeniedBeforeNetwork(t *testing.T) {
	tests := []struct {
		name    string
		version string
		change  func(*model.Updates)
	}{
		{"unknown channel", "1.0.0", func(config *model.Updates) { config.Channel = "unknown" }},
		{"malformed runtime", "development", func(*model.Updates) {
			// The runtime version alone supplies this invalid input.
		}},
		{"unpaired trust", "1.0.0", func(config *model.Updates) { config.TrustStoreSHA256 = strings.Repeat("0", 64) }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			config := interactiveConfig()
			test.change(config)
			_, err := Check(context.Background(), config, Options{Version: test.version, client: noFeedRequests(t)})
			if !fault.Is(err, fault.Invalid) {
				t.Fatalf("invalid configuration was accepted: %v", err)
			}
		})
	}
}

func TestUnavailableReleaseFeedDoesNotClaimCurrent(t *testing.T) {
	for _, status := range []int{http.StatusUnauthorized, http.StatusFound, http.StatusTooManyRequests, http.StatusServiceUnavailable} {
		client := &http.Client{Transport: roundTrip(func(*http.Request) (*http.Response, error) {
			response := releaseResponse(`[]`)
			response.StatusCode = status
			return response, nil
		})}
		assertReleaseCheckFailure(t, client, fault.Unavailable)
	}
	client := &http.Client{Transport: roundTrip(func(*http.Request) (*http.Response, error) {
		return nil, errors.New("synthetic feed failure")
	})}
	assertReleaseCheckFailure(t, client, fault.Unavailable)
}

func TestReleaseResponseReadAndSizeFailuresAreTrustFailures(t *testing.T) {
	bodies := []io.Reader{
		releaseReadFailure{},
		strings.NewReader(strings.Repeat(" ", maximumFeedBytes+1)),
	}
	for _, body := range bodies {
		tracked := &releaseTrackedBody{Reader: body}
		client := &http.Client{Transport: roundTrip(func(*http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: http.StatusOK, Body: tracked, Header: make(http.Header)}, nil
		})}
		assertReleaseCheckFailure(t, client, fault.Trust)
		if !tracked.closed {
			t.Fatal("failed release response body was not closed")
		}
	}
}

func TestReleaseCheckFailsWhenProtectedCredentialIsMissing(t *testing.T) {
	config := interactiveConfig()
	config.TokenFile = filepath.Join(updateStateDirectory(t), "missing-synthetic-token")
	result, err := Check(context.Background(), config, Options{Version: "1.0.0", client: noFeedRequests(t)})
	if !fault.Is(err, fault.Denied) || result.Status != "" {
		t.Fatalf("missing credential supplied a release claim: %#v %v", result, err)
	}
}

func TestReleaseCheckFailsWhenPinnedTrustStoreIsMissing(t *testing.T) {
	config := interactiveConfig()
	config.TrustStoreFile = filepath.Join(updateStateDirectory(t), "missing-synthetic-ca")
	config.TrustStoreSHA256 = strings.Repeat("0", 64)
	result, err := Check(context.Background(), config, Options{Version: "1.0.0"})
	if !fault.Is(err, fault.Trust) || result.Status != "" {
		t.Fatalf("missing pinned trust supplied a release claim: %#v %v", result, err)
	}
}

func TestReleaseCheckRequiresWritableProtectedCacheState(t *testing.T) {
	client := &http.Client{Transport: roundTrip(func(*http.Request) (*http.Response, error) {
		return releaseResponse(`[]`), nil
	})}
	result, err := Check(context.Background(), interactiveConfig(), Options{Version: "1.0.0", client: client})
	if !fault.Is(err, fault.Invalid) || result.Status != "" {
		t.Fatalf("missing protected state supplied a release claim: %#v %v", result, err)
	}
}

func TestReleaseSelectionUsesNewestEligiblePublishedVersion(t *testing.T) {
	client := &http.Client{Transport: roundTrip(func(*http.Request) (*http.Response, error) {
		return releaseResponse(`[
			{"tag_name":"v99.0.0","draft":true},
			{"tag_name":"v1.3.0"},
			{"tag_name":"v1.1.0"},
			{"tag_name":"v2.0.0-rc.1","prerelease":true},
			{"tag_name":"v1.2.0"}
		]`), nil
	})}
	result, err := Check(context.Background(), interactiveConfig(), Options{Version: "v1.0.0", StateDirectory: updateStateDirectory(t), client: client})
	if err != nil || result.Available != "v1.3.0" {
		t.Fatalf("selected a draft, prerelease or older version: %#v %v", result, err)
	}
}

func TestReleaseCacheRejectsInvalidTimeAndTag(t *testing.T) {
	now := time.Date(2026, time.November, 4, 0, 0, 0, 0, time.UTC)
	tests := []struct {
		name    string
		checked time.Time
		tag     string
	}{
		{"zero time", time.Time{}, "v1.1.0"},
		{"future time", now.Add(time.Hour), "v1.1.0"},
		{"expired time", now.Add(-24 * time.Hour), "v1.1.0"},
		{"invalid tag", now, "invalid-tag"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assertInvalidReleaseCacheRefreshes(t, now, test.checked, test.tag)
		})
	}
}

func assertInvalidReleaseCacheRefreshes(t *testing.T, now, checked time.Time, tag string) {
	t.Helper()
	directory := updateStateDirectory(t)
	config := interactiveConfig()
	endpoint, err := feedEndpoint(config.FeedEndpoint)
	if err != nil {
		t.Fatal(err)
	}
	entry := cache{CheckedAt: checked, Channel: config.Channel, Config: configurationFingerprint(endpoint, *config), Status: "current", Available: tag}
	if err := writeCache(directory, entry); err != nil {
		t.Fatal(err)
	}
	requests := 0
	client := &http.Client{Transport: roundTrip(func(*http.Request) (*http.Response, error) {
		requests++
		return releaseResponse(`[]`), nil
	})}
	result, err := Check(context.Background(), config, Options{Version: "1.0.0", StateDirectory: directory, Now: func() time.Time { return now }, client: client})
	if err != nil || requests != 1 || result.Cached || result.Status != "no_release" {
		t.Fatalf("invalid cache was reused: %#v requests=%d err=%v", result, requests, err)
	}
}

func TestReleaseCacheRejectsUnserializableTimestamp(t *testing.T) {
	directory := updateStateDirectory(t)
	entry := cache{CheckedAt: time.Date(10000, time.January, 1, 0, 0, 0, 0, time.UTC)}
	if err := writeCache(directory, entry); !fault.Is(err, fault.Apply) {
		t.Fatalf("invalid cache timestamp was accepted: %v", err)
	}
	data, err := platform.ReadProtected(filepath.Join(directory, cacheName), true)
	if err == nil && len(data) != 0 {
		t.Fatal("invalid cache timestamp produced persisted state")
	}
}

func assertReleaseCheckFailure(t *testing.T, client *http.Client, category fault.Kind) {
	t.Helper()
	result, err := Check(context.Background(), interactiveConfig(), Options{Version: "1.0.0", StateDirectory: updateStateDirectory(t), client: client})
	if !fault.Is(err, category) || result.Status != "" {
		t.Fatalf("failed release check supplied a success claim: %#v %v", result, err)
	}
}

func noFeedRequests(t *testing.T) *http.Client {
	t.Helper()
	return &http.Client{Transport: roundTrip(func(*http.Request) (*http.Response, error) {
		t.Error("invalid or disabled check contacted the release feed")
		return nil, errors.New("unexpected feed request")
	})}
}

type releaseReadFailure struct{}

func (releaseReadFailure) Read([]byte) (int, error) {
	return 0, errors.New("synthetic release reader failure")
}

type releaseTrackedBody struct {
	io.Reader
	closed bool
}

func (body *releaseTrackedBody) Close() error {
	body.closed = true
	return nil
}
