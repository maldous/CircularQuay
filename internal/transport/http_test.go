package transport

import (
	"context"
	"crypto/tls"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/maldous/CircularQuay/internal/fault"
	"github.com/maldous/CircularQuay/internal/platform"
)

func privateCredentialPath(t *testing.T) string {
	t.Helper()
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	directory := filepath.Join(root, "credential")
	if err := platform.EnsurePrivateDirectory(directory); err != nil {
		t.Fatal(err)
	}
	return filepath.Join(directory, "token")
}

func TestEndpointAndPathBoundary(t *testing.T) {
	for _, raw := range []string{"http://host", "https://user:secret@host", "https://host?token=secret", "https://host/#x", "https://", "::::"} {
		if _, err := Endpoint(raw); !fault.Is(err, fault.Trust) {
			t.Fatal("unsafe endpoint accepted")
		}
	}
	u, err := URL("https://host/base/", "v1", "nested/resource")
	if err != nil || u.Path != "/base/v1/nested/resource" {
		t.Fatal("incorrect trusted path")
	}
	for _, path := range []string{"../escape", "/absolute", "a//b", "a?b", "a#b", "a\\b", "a\x00"} {
		if _, err := URL("https://host", path); !fault.Is(err, fault.Invalid) {
			t.Fatal("unsafe resource accepted")
		}
	}
	if _, err := EndpointWithoutQuery(nil); !fault.Is(err, fault.Trust) {
		t.Fatal("nil endpoint accepted")
	}
}

func TestBoundedHTTPAndClassification(t *testing.T) {
	server := httptest.NewTLSServer(responseHandler())
	defer server.Close()
	c := Client{HTTP: server.Client()}
	assertResponseCases(t, c, server.URL)
	assertRequestCases(t, c, server.URL)
	assertTransportClassification(t)
}

func responseHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/ok":
			_, _ = io.WriteString(w, `{"value":"synthetic"}`)
		case "/extra":
			_, _ = io.WriteString(w, `{} {}`)
		case "/bad":
			_, _ = io.WriteString(w, `not-json`)
		case "/large":
			_, _ = io.WriteString(w, strings.Repeat("x", maxPayload+1))
		case "/redirect":
			http.Redirect(w, r, "https://unexpected.invalid", http.StatusFound)
		default:
			w.WriteHeader(http.StatusForbidden)
			_, _ = io.WriteString(w, "DO-NOT-RETURN-THIS-BODY")
		}
	}
}

func assertResponseCases(t *testing.T, c Client, endpoint string) {
	t.Helper()
	for _, test := range []struct {
		path string
		kind fault.Kind
	}{{"ok", ""}, {"extra", fault.Trust}, {"bad", fault.Trust}, {"large", fault.Trust}, {"redirect", fault.Trust}, {"deny", fault.Denied}} {
		assertResponseCase(t, c, endpoint, test.path, test.kind)
	}
}

func assertResponseCase(t *testing.T, c Client, endpoint, path string, kind fault.Kind) {
	t.Helper()
	u, err := URL(endpoint, path)
	if err != nil {
		t.Fatal(err)
	}
	var result map[string]any
	err = c.DoJSON(context.Background(), http.MethodGet, u, nil, nil, &result)
	if kind == "" && (err != nil || result["value"] != "synthetic") {
		t.Fatal("valid response failed")
	}
	if kind != "" && (!fault.Is(err, kind) || strings.Contains(err.Error(), "DO-NOT")) {
		t.Fatal("unsafe response handling")
	}
}

func assertRequestCases(t *testing.T, c Client, endpoint string) {
	t.Helper()
	u, _ := URL(endpoint, "ok")
	if err := c.DoJSON(context.Background(), http.MethodPost, u, nil, map[string]string{"synthetic": "request"}, nil); err != nil {
		t.Fatal(err)
	}
	if err := c.DoJSON(context.Background(), http.MethodPost, u, map[string]string{"Authorization": "bad\nvalue"}, nil, nil); !fault.Is(err, fault.Invalid) {
		t.Fatal("header injection allowed")
	}
	if err := c.DoJSON(context.Background(), "invalid\nmethod", u, nil, nil, nil); !fault.Is(err, fault.Invalid) {
		t.Fatal("invalid method accepted")
	}
	if err := c.DoJSON(context.Background(), http.MethodPost, u, nil, make(chan int), nil); !fault.Is(err, fault.Invalid) {
		t.Fatal("invalid request encoding accepted")
	}
	if err := c.DoJSON(context.Background(), http.MethodPost, u, nil, strings.Repeat("x", maxPayload+1), nil); !fault.Is(err, fault.Invalid) {
		t.Fatal("oversized request accepted")
	}
	if err := (Client{}).DoJSON(context.Background(), http.MethodGet, u, nil, nil, nil); !fault.Is(err, fault.Trust) {
		t.Fatal("untrusted TLS accepted")
	}
	if err := c.DoJSON(context.Background(), http.MethodGet, u, map[string]string{"bad\nname": "value"}, nil, nil); !fault.Is(err, fault.Invalid) {
		t.Fatal("invalid header name accepted")
	}
	if err := c.DoJSON(context.Background(), http.MethodGet, u, map[string]string{"bad name": "value"}, nil, nil); !fault.Is(err, fault.Invalid) {
		t.Fatal("non-token header name accepted")
	}
}

func assertTransportClassification(t *testing.T) {
	t.Helper()
	for code, kind := range map[int]fault.Kind{401: fault.Denied, 403: fault.Denied, 404: fault.Missing, 302: fault.Trust, 429: fault.Trust, 500: fault.Trust} {
		if !fault.Is(Status(code), kind) {
			t.Fatal("incorrect status class")
		}
	}
	assertPositiveConnectionFailures(t)
	assertTransportTrustFailures(t)
	assertCancellationAndTimeoutClassification(t)
}

func assertPositiveConnectionFailures(t *testing.T) {
	t.Helper()
	for _, errno := range nativeConnectionErrnos() {
		err := classifyTransport(context.Background(), context.Background(), &net.OpError{Op: "dial", Err: errno})
		if !fault.Is(err, fault.Unavailable) {
			t.Fatalf("known connection failure misclassified: %v", err)
		}
	}
	for _, err := range []error{&net.DNSError{IsTemporary: true}, &net.DNSError{IsTimeout: true}} {
		if classified := classifyTransport(context.Background(), context.Background(), err); !fault.Is(classified, fault.Unavailable) {
			t.Fatalf("positive DNS failure misclassified: %v", classified)
		}
	}
}

func assertTransportTrustFailures(t *testing.T) {
	t.Helper()
	for _, err := range []error{
		&net.OpError{Op: "dial", Err: errors.New("synthetic")},
		&net.DNSError{IsNotFound: true},
		tls.RecordHeaderError{},
		tls.AlertError(42),
		errors.New("synthetic"),
	} {
		if classified := classifyTransport(context.Background(), context.Background(), err); !fault.Is(classified, fault.Trust) || fault.Is(classified, fault.Unavailable) {
			t.Fatalf("unproven transport failure was not trust: %v", classified)
		}
	}
	if classified := responseReadFailure(&net.OpError{Op: "read", Err: responseReadFixtureError()}); !fault.Is(classified, fault.Trust) || fault.Is(classified, fault.Unavailable) {
		t.Fatalf("partial response error was not trust: %v", classified)
	}
}

func responseReadFixtureError() error {
	native := nativeConnectionErrnos()
	if len(native) > 0 {
		return native[0]
	}
	return errors.New("synthetic")
}

func assertCancellationAndTimeoutClassification(t *testing.T) {
	t.Helper()
	timedOut, cancelTimeout := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	defer cancelTimeout()
	if err := classifyTransport(context.Background(), timedOut, context.DeadlineExceeded); !fault.Is(err, fault.Unavailable) {
		t.Fatalf("bounded timeout misclassified: %v", err)
	}
	caller, cancelCaller := context.WithCancel(context.Background())
	cancelCaller()
	if err := classifyTransport(caller, timedOut, tls.RecordHeaderError{}); !fault.Is(err, fault.Denied) || fault.Is(err, fault.Unavailable) {
		t.Fatalf("caller cancellation misclassified: %v", err)
	}
	if err := classifyTransport(context.Background(), timedOut, tls.RecordHeaderError{}); !fault.Is(err, fault.Trust) || fault.Is(err, fault.Unavailable) {
		t.Fatalf("TLS error lost precedence to timeout: %v", err)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return fn(request)
}

func TestTransportRejectsMalformedRoundTripperResponse(t *testing.T) {
	u, err := URL("https://provider.invalid", "resource")
	if err != nil {
		t.Fatal(err)
	}
	client := Client{HTTP: &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK}, nil
	})}}
	if err := client.DoJSON(context.Background(), http.MethodGet, u, nil, nil, nil); !fault.Is(err, fault.Trust) {
		t.Fatal("malformed round trip response accepted")
	}
}

func TestInjectedClientWithoutTimeoutGetsBoundedRequest(t *testing.T) {
	u, err := URL("https://provider.invalid", "resource")
	if err != nil {
		t.Fatal(err)
	}
	deadlineSet := false
	client := Client{HTTP: &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		_, deadlineSet = request.Context().Deadline()
		return &http.Response{StatusCode: http.StatusNoContent, Body: io.NopCloser(strings.NewReader(""))}, nil
	})}}
	if err := client.DoJSON(context.Background(), http.MethodGet, u, nil, nil, nil); err != nil || !deadlineSet {
		t.Fatal("injected client request was not bounded")
	}
}

func TestHardenedHTTPClientRejectsRedirectsAndOversizedBodies(t *testing.T) {
	t.Run("redirect", assertHardenedRedirect)
	t.Run("content length", assertHardenedDeclaredOversize)
	t.Run("chunked", assertHardenedChunkedOversize)
	t.Run("limit exact", assertHardenedExactLimit)
}

func TestHardenedResponseSizeClassifiesAsTrust(t *testing.T) {
	for _, test := range []struct {
		name          string
		contentLength int64
		status        int
		out           any
	}{
		{name: "decoded chunked", contentLength: -1, status: http.StatusOK, out: &map[string]any{}},
		{name: "discarded chunked", contentLength: -1, status: http.StatusOK},
		{name: "declared", contentLength: maxPayload + 1, status: http.StatusOK},
		{name: "error response body", contentLength: -1, status: http.StatusInternalServerError},
	} {
		t.Run(test.name, func(t *testing.T) {
			client := hardenedOversizedClient(test.status, test.contentLength)
			err := client.DoJSON(context.Background(), http.MethodGet, testProviderURL(t), nil, nil, test.out)
			if !fault.Is(err, fault.Trust) || fault.Is(err, fault.Unavailable) {
				t.Fatalf("oversized response classification = %v", err)
			}
		})
	}
}

func hardenedOversizedClient(status int, contentLength int64) Client {
	return Client{HTTP: NewHTTPClient(roundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode:    status,
			ContentLength: contentLength,
			Body:          io.NopCloser(strings.NewReader(strings.Repeat("x", maxPayload+1))),
		}, nil
	}))}
}

func testProviderURL(t *testing.T) *url.URL {
	t.Helper()
	u, err := URL("https://provider.invalid", "resource")
	if err != nil {
		t.Fatal(err)
	}
	return u
}

func assertHardenedRedirect(t *testing.T) {
	t.Helper()
	requests := 0
	client := NewHTTPClient(roundTripFunc(func(*http.Request) (*http.Response, error) {
		requests++
		return &http.Response{StatusCode: http.StatusFound, Header: http.Header{"Location": {"https://redirect.invalid"}}, Body: io.NopCloser(strings.NewReader(""))}, nil
	}))
	if _, err := client.Get("https://provider.invalid"); !errors.Is(err, errRedirectDenied) || requests != 1 {
		t.Fatalf("redirect result = %v after %d requests", err, requests)
	}
}

func assertHardenedDeclaredOversize(t *testing.T) {
	t.Helper()
	body := &trackedReadCloser{Reader: strings.NewReader("oversized")}
	client := NewHTTPClient(roundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK, ContentLength: maxPayload + 1, Body: body}, nil
	}))
	if _, err := client.Get("https://provider.invalid"); !errors.Is(err, ErrResponseTooLarge) || !body.closed {
		t.Fatalf("declared oversized response result = %v, closed = %t", err, body.closed)
	}
}

func assertHardenedChunkedOversize(t *testing.T) {
	t.Helper()
	body := &trackedReadCloser{Reader: strings.NewReader(strings.Repeat("x", maxPayload+1))}
	client := NewHTTPClient(roundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK, ContentLength: -1, Body: body}, nil
	}))
	response, err := client.Get("https://provider.invalid")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = io.ReadAll(response.Body); !errors.Is(err, ErrResponseTooLarge) {
		t.Fatalf("chunked oversized response result = %v", err)
	}
	assertBodyClosed(t, response.Body, body)
}

func assertHardenedExactLimit(t *testing.T) {
	t.Helper()
	data := strings.Repeat("x", maxPayload)
	body := &trackedReadCloser{Reader: strings.NewReader(data)}
	client := NewHTTPClient(roundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK, ContentLength: -1, Body: body}, nil
	}))
	response, err := client.Get("https://provider.invalid")
	if err != nil {
		t.Fatal(err)
	}
	read, err := io.ReadAll(response.Body)
	if err != nil || string(read) != data {
		t.Fatalf("exact response result = %d bytes, %v", len(read), err)
	}
	var probe [1]byte
	if count, err := response.Body.Read(probe[:]); count != 0 || !errors.Is(err, io.EOF) {
		t.Fatalf("exact response EOF = %d bytes, %v", count, err)
	}
	assertBodyClosed(t, response.Body, body)
}

func assertBodyClosed(t *testing.T, response io.Closer, body *trackedReadCloser) {
	t.Helper()
	if err := response.Close(); err != nil || !body.closed {
		t.Fatalf("response close result = %v, closed = %t", err, body.closed)
	}
}

type trackedReadCloser struct {
	io.Reader
	closed bool
}

func (body *trackedReadCloser) Close() error {
	body.closed = true
	return nil
}

func TestCredentialFileBoundary(t *testing.T) {
	path := privateCredentialPath(t)
	if _, err := ReadCredentialFile("relative"); !fault.Is(err, fault.Invalid) {
		t.Fatal("relative path accepted")
	}
	if _, err := ReadCredentialFile(path); !fault.Is(err, fault.Denied) {
		t.Fatal("missing token accepted")
	}
	writeCredential(t, path, "synthetic-token\n")
	value, err := ReadCredentialFile(path)
	if err != nil || value != "synthetic-token" {
		t.Fatal("protected file not read")
	}
	assertInvalidCredentials(t, path)
	assertCredentialDirectoryDenied(t, path)
}

func writeCredential(t *testing.T, path, data string) {
	t.Helper()
	if err := platform.WriteProtected(path, []byte(data)); err != nil {
		t.Fatal(err)
	}
}

func assertInvalidCredentials(t *testing.T, path string) {
	t.Helper()
	for _, data := range []string{"", "bad\nheader", strings.Repeat("x", 65537)} {
		writeCredential(t, path, data)
		if _, err := ReadCredentialFile(path); !fault.Is(err, fault.Trust) {
			t.Fatal("invalid token accepted")
		}
	}
}

func assertCredentialDirectoryDenied(t *testing.T, path string) {
	t.Helper()
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(path, 0700); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadCredentialFile(path); !fault.Is(err, fault.Trust) {
		t.Fatal("directory token accepted")
	}
}

func FuzzEndpointNeverAcceptsCredentialURL(f *testing.F) {
	for _, raw := range []string{"https://example.invalid", "http://host", "https://user:secret@host", "https://host?token=value"} {
		f.Add(raw)
	}
	f.Fuzz(func(t *testing.T, raw string) {
		u, err := Endpoint(raw)
		if err == nil && (u.User != nil || u.Scheme != "https" || u.RawQuery != "" || u.Fragment != "") {
			t.Fatal("unsafe accepted URL")
		}
	})
}
