package transport

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/maldous/CircularQuay/internal/fault"
)

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
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/ok":
			w.Header().Set("Content-Type", "application/json")
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
	}))
	defer server.Close()
	c := Client{HTTP: server.Client()}
	for _, test := range []struct {
		path string
		kind fault.Kind
	}{{"ok", ""}, {"extra", fault.Trust}, {"bad", fault.Trust}, {"large", fault.Trust}, {"redirect", fault.Trust}, {"deny", fault.Denied}} {
		u, err := URL(server.URL, test.path)
		if err != nil {
			t.Fatal(err)
		}
		var result map[string]any
		err = c.DoJSON(context.Background(), http.MethodGet, u, nil, nil, &result)
		if test.kind == "" {
			if err != nil || result["value"] != "synthetic" {
				t.Fatal("valid response failed")
			}
		} else if !fault.Is(err, test.kind) || strings.Contains(err.Error(), "DO-NOT") {
			t.Fatal("unsafe response handling")
		}
	}
	u, _ := URL(server.URL, "ok")
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
	for code, kind := range map[int]fault.Kind{401: fault.Denied, 403: fault.Denied, 404: fault.Missing, 302: fault.Trust, 429: fault.Trust, 500: fault.Trust} {
		if !fault.Is(Status(code), kind) {
			t.Fatal("incorrect status class")
		}
	}
	if !fault.Is(classifyTransport(&net.OpError{Op: "dial", Err: errors.New("unavailable")}), fault.Unavailable) {
		t.Fatal("network failure misclassified")
	}
	if !fault.Is(classifyTransport(&net.DNSError{Err: "unavailable"}), fault.Unavailable) {
		t.Fatal("DNS failure misclassified")
	}
	if !fault.Is(classifyTransport(context.DeadlineExceeded), fault.Unavailable) {
		t.Fatal("timeout misclassified")
	}
	if !fault.Is(classifyTransport(errors.New("private upstream body")), fault.Trust) {
		t.Fatal("unknown failure misclassified")
	}
}

func TestCredentialFileBoundary(t *testing.T) {
	path := filepath.Join(t.TempDir(), "token")
	if _, err := ReadCredentialFile("relative"); !fault.Is(err, fault.Invalid) {
		t.Fatal("relative path accepted")
	}
	if _, err := ReadCredentialFile(path); !fault.Is(err, fault.Denied) {
		t.Fatal("missing token accepted")
	}
	if err := os.WriteFile(path, []byte("synthetic-token\n"), 0600); err != nil {
		t.Fatal(err)
	}
	value, err := ReadCredentialFile(path)
	if err != nil || value != "synthetic-token" {
		t.Fatal("protected file not read")
	}
	for _, data := range []string{"", "bad\nheader", strings.Repeat("x", 65537)} {
		if err := os.WriteFile(path, []byte(data), 0600); err != nil {
			t.Fatal(err)
		}
		if _, err := ReadCredentialFile(path); !fault.Is(err, fault.Trust) {
			t.Fatal("invalid token accepted")
		}
	}
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
