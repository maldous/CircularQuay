package assurance

import (
	"context"
	"encoding/json"
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
	case "/api/qualityprofiles/search":
		profiles := []sonarQualityProfile{}
		for _, language := range []string{"go", "json", "yaml", "docker", "xml", "web", "text", "secrets"} {
			profiles = append(profiles, sonarQualityProfile{Key: "fixture-" + language, Name: "Sonar way comprehensive", Language: language, BuiltIn: true, ActiveRules: 23})
		}
		_ = json.NewEncoder(writer).Encode(map[string]any{"profiles": profiles})
		return
	case "/api/qualityprofiles/copy":
		response = `{"key":"copied-profile","name":"CircularQuay Go comprehensive"}`
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
	password, err := readToken(filepath.Join(directory, "admin.password"))
	if err != nil || len(password) < 12 || !strings.ContainsAny(password, "ABCDEFGHIJKLMNOPQRSTUVWXYZ") || !strings.ContainsAny(password, "abcdefghijklmnopqrstuvwxyz") || !strings.ContainsAny(password, "0123456789") || !strings.ContainsAny(password, "!@#$%") {
		t.Fatal("bootstrap password does not satisfy server policy")
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

func TestSonarStrictConditionsAndProfiles(t *testing.T) {
	conditions := map[string]string{}
	languages := map[string]bool{}
	complexity := ""
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Error(err)
		}
		switch r.URL.Path {
		case "/api/qualitygates/create_condition", "/api/qualitygates/update_condition":
			conditions[r.Form.Get("metric")] = r.Form.Get("op") + ":" + r.Form.Get("error")
		case "/api/qualityprofiles/add_project":
			languages[r.Form.Get("language")] = true
		case "/api/qualityprofiles/activate_rule":
			if r.Form.Get("rule") == "go:S3776" {
				complexity = r.Form.Get("params")
			}
		}
		sonarFixture(w, r)
	}))
	defer server.Close()
	client := newSonarClient(server.URL, "synthetic", "")
	if err := configureSonarGate(context.Background(), client, "fixture"); err != nil {
		t.Fatal(err)
	}
	if err := configureSonarProfiles(context.Background(), client, "fixture"); err != nil {
		t.Fatal(err)
	}
	expected := map[string]string{"violations": "GT:0", "new_violations": "GT:0", "coverage": "LT:100", "new_coverage": "LT:100", "duplicated_lines_density": "GT:0", "new_duplicated_lines_density": "GT:0", "security_hotspots_reviewed": "LT:100", "new_security_hotspots_reviewed": "LT:100"}
	if len(conditions) != len(expected) {
		t.Fatal("strict gate conditions incomplete")
	}
	for metric, want := range expected {
		if conditions[metric] != want {
			t.Fatalf("wrong strict threshold for %s", metric)
		}
	}
	if len(languages) != 8 || complexity != "threshold=10" {
		t.Fatal("comprehensive profiles or complexity limit missing")
	}
}
