package transport

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"testing"

	"github.com/maldous/CircularQuay/internal/fault"
	"github.com/maldous/CircularQuay/internal/platform"
)

func TestResponseBoundaryRejectsOverflowAfterLimit(t *testing.T) {
	body := &boundedResponseBody{ReadCloser: io.NopCloser(strings.NewReader("x"))}
	if count, err := body.Read(make([]byte, 1)); count != 0 || !errors.Is(err, ErrResponseTooLarge) {
		t.Fatalf("exhausted response boundary returned %d bytes, %v", count, err)
	}
	zero := &boundedResponseBody{ReadCloser: io.NopCloser(strings.NewReader("synthetic")), remaining: 9}
	if count, err := zero.Read(nil); count != 0 || err != nil || zero.remaining != 9 {
		t.Fatalf("empty read consumed response allowance: %d, %v, %d", count, err, zero.remaining)
	}
}

func TestRequestEndpointRefusalPrecedesRoundTrip(t *testing.T) {
	requests := 0
	client := Client{HTTP: NewHTTPClient(roundTripFunc(func(*http.Request) (*http.Response, error) {
		requests++
		return nil, errors.New("synthetic-private-upstream-detail")
	}))}
	endpoint := &url.URL{Scheme: "http", Host: "provider.invalid"}
	err := client.DoJSON(context.Background(), http.MethodGet, endpoint, nil, nil, nil)
	if !fault.Is(err, fault.Trust) || requests != 0 {
		t.Fatalf("invalid endpoint reached transport: %v, requests %d", err, requests)
	}
	if _, err := URL("http://provider.invalid", "resource"); !fault.Is(err, fault.Trust) {
		t.Fatal("path composition accepted an untrusted base")
	}
	composed, err := URL("https://provider.invalid/base/", "", "resource", "")
	if err != nil || composed.Path != "/base/resource" {
		t.Fatalf("empty optional path components changed resource: %v, %v", composed, err)
	}
}

func TestUnconfiguredTransportStillBoundsAndRedactsReadFailure(t *testing.T) {
	original := http.DefaultTransport
	t.Cleanup(func() { http.DefaultTransport = original })
	body := &trackedReadCloser{Reader: boundaryReadError{}}
	requests := 0
	http.DefaultTransport = roundTripFunc(func(request *http.Request) (*http.Response, error) {
		requests++
		if _, ok := request.Context().Deadline(); !ok {
			t.Fatal("default transport request has no deadline")
		}
		return &http.Response{StatusCode: http.StatusOK, Body: body}, nil
	})
	client := Client{HTTP: &http.Client{}}
	err := client.DoJSON(context.Background(), http.MethodGet, testProviderURL(t), nil, nil, nil)
	if !fault.Is(err, fault.Trust) || requests != 1 || !body.closed {
		t.Fatalf("partial discarded body was accepted or left open: %v, requests %d, closed %t", err, requests, body.closed)
	}
	if strings.Contains(err.Error(), "synthetic-private") {
		t.Fatal("reader detail escaped the transport boundary")
	}
}

type boundaryReadError struct{}

func (boundaryReadError) Read([]byte) (int, error) {
	return 0, errors.New("synthetic-private-read-detail")
}

func TestTrustStoreRejectsInvalidDigestBeforeFileLookup(t *testing.T) {
	missing := filepath.Join(canonicalTrustStoreDirectory(t), "absent.pem")
	for _, digest := range []string{"", strings.Repeat("A", 64), strings.Repeat("g", 64)} {
		if err := ValidateTrustStore(missing, digest); !fault.Is(err, fault.Invalid) {
			t.Fatalf("invalid digest reached missing-file trust handling: %v", err)
		}
	}
	if err := ValidateTrustStore(missing, strings.Repeat("0", 64)); !fault.Is(err, fault.Trust) {
		t.Fatalf("missing protected trust store was accepted: %v", err)
	}
}

func TestTrustStoreRejectsEmptyAndOversizedProtectedBundles(t *testing.T) {
	path := filepath.Join(canonicalTrustStoreDirectory(t), "authority-roots.pem")
	for _, data := range [][]byte{nil, bytes.Repeat([]byte("x"), maxTrustStoreBytes+1)} {
		if err := platform.WriteProtected(path, data); err != nil {
			t.Fatal(err)
		}
		digest := sha256.Sum256(data)
		if err := ValidateTrustStore(path, hex.EncodeToString(digest[:])); !fault.Is(err, fault.Invalid) {
			t.Fatalf("invalid trust-store size was accepted: %v", err)
		}
	}
}

func TestTrustStoreRejectsEmptyPoolAndCertificateCountOverflow(t *testing.T) {
	root, _ := syntheticTrustStoreCertificate(t)
	for _, data := range [][]byte{[]byte(" \n\t"), bytes.Repeat(root, maxTrustStoreCertificates+1)} {
		if _, err := certificatePool(data); !fault.Is(err, fault.Invalid) {
			t.Fatalf("empty or over-limit root pool was accepted: %v", err)
		}
	}
	block, _ := pem.Decode(root)
	block.Headers = map[string]string{"Synthetic": "private-header"}
	if _, err := certificatePool(pem.EncodeToMemory(block)); !fault.Is(err, fault.Invalid) {
		t.Fatalf("PEM headers were accepted: %v", err)
	}
}

func TestTrustStoreRejectsUnsupportedAmbientTransport(t *testing.T) {
	root, _ := syntheticTrustStoreCertificate(t)
	path := filepath.Join(canonicalTrustStoreDirectory(t), "authority-roots.pem")
	if err := platform.WriteProtected(path, root); err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(root)
	original := http.DefaultTransport
	http.DefaultTransport = roundTripFunc(func(*http.Request) (*http.Response, error) {
		t.Fatal("constructing a trust-store client made a request")
		return nil, nil
	})
	t.Cleanup(func() { http.DefaultTransport = original })
	if _, err := TrustStoreHTTPClient(path, hex.EncodeToString(digest[:])); !fault.Is(err, fault.Trust) {
		t.Fatalf("unsupported ambient transport was accepted: %v", err)
	}
	if _, err := TrustStoreHTTPClientNoProxy(path, hex.EncodeToString(digest[:])); err != nil {
		t.Fatalf("independent no-proxy transport inherited ambient failure: %v", err)
	}
}
