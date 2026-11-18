//go:build integration

package integration

import (
	"context"
	"encoding/json"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/maldous/CircularQuay/internal/fault"
	"github.com/maldous/CircularQuay/internal/model"
	"github.com/maldous/CircularQuay/internal/platform"
	"github.com/maldous/CircularQuay/internal/source"
)

const syntheticValue = "WIREMOCK_SYNTHETIC_VALUE"

func TestWireMockOpenBaoKVv2(t *testing.T) {
	w := startWireMock(t)
	token := tokenFile(t, w.dir)
	t.Run("positive versioned read", func(t *testing.T) { testPositiveRead(t, w, token) })
	t.Run("401 and 403 denial", func(t *testing.T) { testDeniedRead(t, w, token) })
	t.Run("malformed JSON", func(t *testing.T) { testMalformedRead(t, w, token) })
	t.Run("wrong version", func(t *testing.T) { testWrongVersion(t, w, token) })
	t.Run("scope denies before contact", func(t *testing.T) { testScopeBeforeContact(t, w, token) })
}

func testPositiveRead(t *testing.T, w *wireMock, token string) {
	t.Helper()
	w.reset(t)
	w.mapping(t, vaultMapping(http.StatusOK, vaultBody(2)))
	value, err := registry(w).Read(context.Background(), vaultSource(w, token), vaultBinding())
	if err != nil || value.Data != syntheticValue || value.Version != "2" || value.Leased || value.Renewable {
		t.Fatal("positive OpenBao result invalid")
	}
	assertCount(t, w, 1)
}

func testDeniedRead(t *testing.T, w *wireMock, token string) {
	t.Helper()
	for _, status := range []int{http.StatusUnauthorized, http.StatusForbidden} {
		w.reset(t)
		w.mapping(t, vaultMapping(status, `RAW-ERROR-`+syntheticValue))
		value, err := registry(w).Read(context.Background(), vaultSource(w, token), vaultBinding())
		if !fault.Is(err, fault.Denied) || value != (model.Value{}) || strings.Contains(err.Error(), syntheticValue) {
			t.Fatalf("status %d was not a safe denial", status)
		}
		assertCount(t, w, 1)
	}
}

func testMalformedRead(t *testing.T, w *wireMock, token string) {
	t.Helper()
	w.reset(t)
	w.mapping(t, vaultMapping(http.StatusOK, `{"data":`+jsonString(syntheticValue)))
	value, err := registry(w).Read(context.Background(), vaultSource(w, token), vaultBinding())
	if !fault.Is(err, fault.Trust) || value != (model.Value{}) || strings.Contains(err.Error(), syntheticValue) {
		t.Fatal("malformed response was accepted or leaked")
	}
	assertCount(t, w, 1)
}

func testWrongVersion(t *testing.T, w *wireMock, token string) {
	t.Helper()
	w.reset(t)
	w.mapping(t, vaultMapping(http.StatusOK, vaultBody(3)))
	value, err := registry(w).Read(context.Background(), vaultSource(w, token), vaultBinding())
	if !fault.Is(err, fault.Trust) || value != (model.Value{}) || strings.Contains(err.Error(), syntheticValue) {
		t.Fatal("wrong provider version was accepted or leaked")
	}
	assertCount(t, w, 1)
}

func testScopeBeforeContact(t *testing.T, w *wireMock, token string) {
	t.Helper()
	w.reset(t)
	w.mapping(t, vaultMapping(http.StatusOK, vaultBody(2)))
	value, err := registry(w).Read(context.Background(), vaultSource(w, token), model.Binding{Name: "KEY", Resource: "forbidden", Version: "2"})
	if !fault.Is(err, fault.Denied) || value != (model.Value{}) {
		t.Fatal("out-of-scope binding was not denied")
	}
	assertCount(t, w, 0)
}

func registry(w *wireMock) *source.Registry { return source.NewRegistryWithClient(w.client) }
func vaultSource(w *wireMock, token string) model.Source {
	return model.Source{Type: "openbao", Endpoint: w.base, Mount: "kv", TokenFile: token, Resources: []string{"app"}}
}
func vaultBinding() model.Binding { return model.Binding{Name: "KEY", Resource: "app", Version: "2"} }

func tokenFile(t *testing.T, dir string) string {
	t.Helper()
	path := filepath.Join(dir, "token")
	if err := platform.CreateProtectedExclusive(path, []byte("wiremock-test-token\n")); err != nil {
		t.Fatalf("write synthetic token: %v", err)
	}
	return path
}

func vaultMapping(status int, body string) string {
	return `{"request":{"method":"GET","urlPath":"/v1/kv/data/app","queryParameters":{"version":{"equalTo":"2"}},"headers":{"X-Vault-Token":{"equalTo":"wiremock-test-token"}}},"response":{"status":` + strconv.Itoa(status) + `,"headers":{"Content-Type":"application/json"},"body":` + jsonString(body) + `}}`
}

func vaultBody(version int) string {
	return `{"data":{"data":{"KEY":"` + syntheticValue + `"},"metadata":{"version":` + strconv.Itoa(version) + `}},"lease_duration":0}`
}
func jsonString(value string) string { data, _ := json.Marshal(value); return string(data) }
func assertCount(t *testing.T, w *wireMock, want int) {
	t.Helper()
	if got := w.count(t, "/v1/kv/data/app?version=2"); got != want {
		t.Fatalf("WireMock journal count = %d, want %d", got, want)
	}
	w.verifyJournal(t, want)
}
