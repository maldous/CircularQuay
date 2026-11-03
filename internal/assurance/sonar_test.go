package assurance

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func sonarFixture(writer http.ResponseWriter, request *http.Request) {
	writer.Header().Set("Content-Type", "application/json")
	var response string
	switch request.URL.Path {
	case "/api/qualitygates/project_status":
		response = `{"projectStatus":{"status":"OK"}}`
	case "/api/issues/search":
		response = `{"total":0,"paging":{"total":0}}`
	case "/api/hotspots/search":
		response = `{"paging":{"total":0}}`
	case "/api/measures/component":
		response = `{"component":{"measures":[{"metric":"coverage","value":"100"},{"metric":"duplicated_lines_density","value":"0"}]}}`
	case "/api/authentication/validate":
		response = `{"valid":true}`
	case "/api/user_tokens/generate":
		response = `{"token":"synthetic-scanner-token"}`
	default:
		response = `{}`
	}
	_, _ = io.WriteString(writer, response)
}

func TestInspectSonar(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(sonarFixture))
	defer server.Close()
	summary, err := inspectSonar(context.Background(), server.URL, "synthetic-token", "fixture")
	if err != nil || summary.Gate != "OK" || summary.Coverage != 100 || summary.Issues != 0 {
		t.Fatal("valid Sonar fixture rejected")
	}
}

func TestSonarRejectsRedirectAndRawErrors(t *testing.T) {
	contacted := false
	target := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { contacted = true }))
	defer target.Close()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Location", target.URL)
		w.WriteHeader(http.StatusFound)
		_, _ = io.WriteString(w, "synthetic-secret-do-not-reflect")
	}))
	defer server.Close()
	client := newSonarClient(server.URL, "synthetic-token", "")
	err := client.call(context.Background(), http.MethodGet, "/api/example", nil, nil)
	if err == nil || contacted || strings.Contains(err.Error(), "synthetic-secret") {
		t.Fatal("unsafe Sonar response handling")
	}
}

func TestSonarMalformedMetrics(t *testing.T) {
	for _, response := range []string{`{}`, `{"component":{"measures":[{"metric":"coverage","value":"secret"}]}}`, `{"component":{"measures":[{"metric":"coverage","value":"101"}]}}`} {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/api/measures/component" {
				_, _ = io.WriteString(w, response)
				return
			}
			sonarFixture(w, r)
		}))
		if _, err := inspectSonar(context.Background(), server.URL, "fixture", "key"); err == nil {
			t.Fatal("invalid metrics accepted")
		}
		server.Close()
	}
}

func TestSonarPrivateStateAndBootstrap(t *testing.T) {
	directory := filepath.Join(t.TempDir(), "state")
	if err := InitSonar(directory); err != nil {
		t.Fatal(err)
	}
	if err := InitSonar(directory); err == nil {
		t.Fatal("existing bootstrap state overwritten")
	}
	if err := InitSonar(""); err == nil {
		t.Fatal("empty bootstrap directory accepted")
	}
	server := httptest.NewServer(http.HandlerFunc(sonarFixture))
	defer server.Close()
	if err := BootstrapSonar(context.Background(), directory, server.URL, "fixture"); err != nil {
		t.Fatal(err)
	}
	if _, err := readToken(filepath.Join(directory, "scanner.token")); err != nil {
		t.Fatal("scanner token not protected")
	}
	if err := BootstrapSonar(context.Background(), directory, "https://public.example.com", "fixture"); err == nil {
		t.Fatal("public server bootstrap accepted")
	}
	if err := BootstrapSonar(context.Background(), directory, "invalid", "fixture"); err == nil {
		t.Fatal("invalid endpoint accepted")
	}
	if err := writeNewPrivate(filepath.Join(directory, "scanner.token"), []byte("other")); err == nil {
		t.Fatal("private state overwritten")
	}
}

func TestSonarClientAuthorizationNotInURL(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.String(), "synthetic-token") {
			t.Error("credential in URL")
		}
		name, password, ok := r.BasicAuth()
		if !ok || name != "synthetic-token" || password != "" {
			t.Error("missing protected header")
		}
		if err := r.ParseForm(); err != nil {
			t.Error(err)
		}
		if r.Form.Get("name") != "fixture" {
			t.Error("form parameters lost")
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()
	if err := newSonarClient(server.URL, "synthetic-token", "").call(context.Background(), http.MethodPost, "/api/fixture", url.Values{"name": {"fixture"}}, nil); err != nil {
		t.Fatal(err)
	}
}

func TestSonarCredentialFilesHaveNoPublicPermissions(t *testing.T) {
	directory := t.TempDir()
	if err := InitSonar(directory); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"compose.env", "admin.password"} {
		info, err := os.Stat(filepath.Join(directory, name))
		if err != nil || info.Mode().Perm()&0077 != 0 {
			t.Fatal("bootstrap file not private")
		}
	}
}
