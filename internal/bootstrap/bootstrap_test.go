package bootstrap

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/maldous/CircularQuay/internal/fault"
)

const testRef = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return fn(request)
}

func TestFetchNormalizesImmutableGitHubSources(t *testing.T) {
	data := []byte(`{"version":1}`)
	path := "config/policy.json"
	wantURL := "https://raw.githubusercontent.com/acme/cquay/" + testRef + "/" + path
	for _, sourceURL := range []string{
		"https://github.com/acme/cquay/tree/" + testRef,
		"https://github.com/acme/cquay/blob/" + testRef + "/" + path,
		wantURL,
	} {
		t.Run(sourceURL, func(t *testing.T) {
			assertImmutableSourceNormalization(t, sourceURL, path, wantURL, data)
		})
	}
}

func assertImmutableSourceNormalization(t *testing.T, sourceURL, path, wantURL string, data []byte) {
	t.Helper()
	fetcher := testFetcher(t, immutableResponse(t, wantURL, data))
	result, err := fetcher.Fetch(context.Background(), testRequest(sourceURL, path, data))
	if err != nil {
		t.Fatal(err)
	}
	if result.CanonicalSourceURL != wantURL || string(result.Bytes) != string(data) || result.SHA256 != digest(data) {
		t.Fatalf("unexpected result: %#v", result)
	}
}

func immutableResponse(t *testing.T, wantURL string, data []byte) roundTripFunc {
	t.Helper()
	return func(request *http.Request) (*http.Response, error) {
		assertImmutableRequest(t, request, wantURL)
		return response(http.StatusOK, data, int64(len(data))), nil
	}
}

func assertImmutableRequest(t *testing.T, request *http.Request, wantURL string) {
	t.Helper()
	if request.URL.String() != wantURL || request.Method != http.MethodGet {
		t.Fatalf("request = %s %s", request.Method, request.URL)
	}
	for _, header := range []string{"Authorization", "Cookie", "Proxy-Authorization"} {
		if request.Header.Get(header) != "" {
			t.Fatalf("unexpected %s header", header)
		}
	}
}

func TestFetchRejectsMutableOrAmbiguousSourceBeforeHTTP(t *testing.T) {
	path := "config/policy.json"
	data := []byte(`{"version":1}`)
	valid := "https://github.com/acme/cquay/tree/" + testRef
	for _, change := range []func(*Request){
		func(request *Request) { request.SourceURL = "https://github.com/acme/cquay/tree/main" },
		func(request *Request) { request.SourceURL = valid + "/config" },
		func(request *Request) {
			request.SourceURL = "https://github.com/acme/cquay/blob/" + testRef + "/config/other.json"
		},
		func(request *Request) {
			request.SourceURL = "https://raw.githubusercontent.com/acme/cquay/" + testRef + "/config/%70olicy.json"
		},
		func(request *Request) { request.SourceURL = "https://github.com/acme/%zz" },
		func(request *Request) { request.SourceURL = "https://github.com/../cquay/tree/" + testRef },
		func(request *Request) { request.SourceURL = "https://github.com/acme/../tree/" + testRef },
		func(request *Request) { request.SourceURL = strings.Repeat("x", 8193) },
		func(request *Request) { request.SourceURL = valid + "?ref=main" },
		func(request *Request) { request.SourceURL = "https://user@example.com/acme/cquay/tree/" + testRef },
		func(request *Request) { request.SourceURL = "https://github.com:443/acme/cquay/tree/" + testRef },
		func(request *Request) { request.ConfigPath = "config/../policy.json" },
		func(request *Request) { request.ConfigPath = "config/policy%2Ejson" },
		func(request *Request) { request.ConfigPath = strings.Repeat("a", 4097) },
		func(request *Request) { request.ConfigRef = strings.ToUpper(testRef) },
		func(request *Request) { request.ExpectedSHA256 = "" },
	} {
		request := testRequest(valid, path, data)
		change(&request)
		calls := 0
		fetcher := testFetcher(t, func(*http.Request) (*http.Response, error) {
			calls++
			return nil, nil
		})
		if _, err := fetcher.Fetch(context.Background(), request); !fault.Is(err, fault.Invalid) || calls != 0 {
			t.Fatalf("unsafe request reached HTTP: error=%v calls=%d", err, calls)
		}
	}
}

func TestFetchRejectsMalformedComponentsBeforeHTTP(t *testing.T) {
	data := []byte(`{"version":1}`)
	for _, change := range []func(*Request){
		func(request *Request) { request.SourceURL = "https://github.com/-/cquay/tree/" + testRef },
		func(request *Request) { request.SourceURL = "https://github.com/acme/tree" },
		func(request *Request) { request.ConfigPath = "config/space name.json" },
		func(request *Request) { request.ConfigPath = "config/" + strings.Repeat("a", 101) },
	} {
		request := testRequest("https://github.com/acme/cquay/tree/"+testRef, "config/policy.json", data)
		change(&request)
		if _, err := normalize(request); !fault.Is(err, fault.Invalid) {
			t.Fatalf("malformed component accepted: %v", err)
		}
	}
}

func TestFetchRejectsUntrustedResponses(t *testing.T) {
	valid := "https://github.com/acme/cquay/tree/" + testRef
	data := []byte(`{"version":1}`)
	for _, test := range []struct {
		name     string
		response func() *http.Response
		request  func(Request) Request
	}{
		{"redirect", redirectResponse, identityRequest},
		{"missing", func() *http.Response { return response(http.StatusNotFound, nil, 0) }, identityRequest},
		{"oversized_header", func() *http.Response { return response(http.StatusOK, data, maxArtifactSize+1) }, identityRequest},
		{"oversized_stream", func() *http.Response {
			return response(http.StatusOK, []byte(strings.Repeat("x", maxArtifactSize+1)), -1)
		}, identityRequest},
		{"invalid_json", func() *http.Response { return response(http.StatusOK, []byte(`{"version":1} trailing`), -1) }, identityRequest},
		{"duplicate_json", func() *http.Response { return response(http.StatusOK, []byte(`{"version":1,"version":1}`), -1) }, identityRequest},
		{"digest_mismatch", func() *http.Response { return response(http.StatusOK, data, -1) }, func(request Request) Request { request.ExpectedSHA256 = strings.Repeat("b", 64); return request }},
	} {
		t.Run(test.name, func(t *testing.T) {
			fetcher := testFetcher(t, func(*http.Request) (*http.Response, error) { return test.response(), nil })
			_, err := fetcher.Fetch(context.Background(), test.request(testRequest(valid, "config/policy.json", data)))
			if !fault.Is(err, fault.Trust) && !(test.name == "missing" && fault.Is(err, fault.Missing)) {
				t.Fatalf("untrusted response accepted: %v", err)
			}
		})
	}
}

func TestFetchHonorsCanceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	calls := 0
	fetcher := testFetcher(t, func(*http.Request) (*http.Response, error) {
		calls++
		return nil, nil
	})
	if _, err := fetcher.Fetch(ctx, testRequest("https://github.com/acme/cquay/tree/"+testRef, "config/policy.json", []byte(`{"version":1}`))); !fault.Is(err, fault.Apply) || calls != 0 {
		t.Fatalf("canceled request reached HTTP: error=%v calls=%d", err, calls)
	}
}

func TestFetchClassifiesTransportAndResponseReadFailures(t *testing.T) {
	data := []byte(`{"version":1}`)
	request := testRequest("https://github.com/acme/cquay/tree/"+testRef, "config/policy.json", data)
	marker := errors.New("SYNTHETIC_TRANSPORT_MARKER")
	fetcher := testFetcher(t, func(*http.Request) (*http.Response, error) { return nil, marker })
	if _, err := fetcher.Fetch(context.Background(), request); !fault.Is(err, fault.Trust) || strings.Contains(err.Error(), marker.Error()) {
		t.Fatalf("transport error was not safely classified: %v", err)
	}
	timeoutFetcher := testFetcher(t, func(*http.Request) (*http.Response, error) { return nil, context.DeadlineExceeded })
	if _, err := timeoutFetcher.Fetch(context.Background(), request); !fault.Is(err, fault.Unavailable) {
		t.Fatalf("timeout was not classified as unavailable: %v", err)
	}
	fetcher = testFetcher(t, func(*http.Request) (*http.Response, error) {
		return responseWithError(http.StatusOK, -1, marker), nil
	})
	if _, err := fetcher.Fetch(context.Background(), request); !fault.Is(err, fault.Trust) {
		t.Fatalf("body read error was not rejected: %v", err)
	}
}

func TestFetcherDefaultsDoNotRetainClientState(t *testing.T) {
	if NewFetcher(nil).httpClient() == nil {
		t.Fatal("default hardened client missing")
	}
	var absent *Fetcher
	if absent.httpClient() == nil {
		t.Fatal("nil fetcher default client missing")
	}
}

func testFetcher(t *testing.T, roundTrip roundTripFunc) *Fetcher {
	t.Helper()
	return NewFetcher(&http.Client{Transport: roundTrip})
}

func testRequest(sourceURL, path string, data []byte) Request {
	return Request{SourceURL: sourceURL, ConfigRef: testRef, ConfigPath: path, ExpectedSHA256: digest(data)}
}

func digest(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func response(status int, data []byte, length int64) *http.Response {
	return &http.Response{StatusCode: status, ContentLength: length, Body: io.NopCloser(strings.NewReader(string(data))), Header: make(http.Header)}
}

func responseWithError(status int, length int64, err error) *http.Response {
	return &http.Response{StatusCode: status, ContentLength: length, Body: io.NopCloser(errorReader{err: err}), Header: make(http.Header)}
}

func redirectResponse() *http.Response {
	result := response(http.StatusFound, nil, 0)
	result.Header.Set("Location", "https://unexpected.invalid/config.json")
	return result
}

func identityRequest(request Request) Request { return request }

type errorReader struct{ err error }

func (reader errorReader) Read([]byte) (int, error) { return 0, reader.err }
