package assurance

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"testing"

	"github.com/maldous/CircularQuay/internal/platform"
)

func sonarFixture(writer http.ResponseWriter, request *http.Request) {
	writer.Header().Set("Content-Type", "application/json")
	if request.URL.Path == "/api/ce/task" {
		component := "fixture"
		if request.URL.Query().Get("id") == "fixture-keytask" {
			component = "key"
		}
		_, _ = io.WriteString(writer, `{"task":{"componentKey":"`+component+`","status":"SUCCESS","analysisId":"fixture-analysis","warningCount":0}}`)
		return
	}
	responses := map[string]string{
		"/api/qualitygates/project_status": `{"projectStatus":{"status":"OK"}}`,
		"/api/project_analyses/search":     `{"analyses":[{"key":"fixture-analysis","revision":"` + strings.Repeat("a", 40) + `"}]}`,
		"/api/issues/search":               `{"total":0,"paging":{"total":0}}`,
		"/api/hotspots/search":             `{"paging":{"total":0}}`,
		"/api/measures/component":          `{"component":{"measures":[{"metric":"coverage","value":"100"},{"metric":"duplicated_lines_density","value":"0"}]}}`,
		"/api/authentication/validate":     `{"valid":true}`,
		"/api/qualityprofiles/copy":        `{"key":"copied-profile","name":"CircularQuay Go comprehensive"}`,
		"/api/user_tokens/generate":        `{"token":"synthetic-scanner-token"}`,
	}

	if request.URL.Path == "/api/qualityprofiles/search" {
		profiles := []sonarQualityProfile{}
		for _, language := range []string{"go", "json", "yaml", "docker", "xml", "web", "text", "secrets"} {
			profiles = append(profiles, sonarQualityProfile{Key: "fixture-" + language, Name: "Sonar way comprehensive", Language: language, BuiltIn: true, ActiveRules: 23})
		}
		_ = json.NewEncoder(writer).Encode(map[string]any{"profiles": profiles})
		return
	}
	response, ok := responses[request.URL.Path]
	if !ok {
		response = `{}`
	}
	_, _ = io.WriteString(writer, response)
}

func TestInspectSonar(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(sonarFixture))
	defer server.Close()
	summary, err := inspectSonar(context.Background(), server.URL, "synthetic-token", "fixture", strings.Repeat("a", 40), "fixture-task")
	if err != nil || summary.Gate != "OK" || summary.Coverage != 100 || summary.Issues != 0 {
		t.Fatal("valid Sonar fixture rejected")
	}
}

func TestInspectSonarRejectsConcurrentAnalysis(t *testing.T) {
	analysisReads := 0
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path == "/api/project_analyses/search" {
			analysisReads++
			if analysisReads > 1 {
				_, _ = io.WriteString(writer, `{"analyses":[{"key":"replacement-analysis","revision":"`+strings.Repeat("a", 40)+`"}]}`)
				return
			}
		}
		sonarFixture(writer, request)
	}))
	defer server.Close()
	_, err := inspectSonar(context.Background(), server.URL, "synthetic-token", "fixture", strings.Repeat("a", 40), "fixture-task")
	if err == nil || analysisReads != 2 {
		t.Fatal("mixed analysis evidence accepted")
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
		if _, err := inspectSonar(context.Background(), server.URL, "fixture", "fixture", strings.Repeat("a", 40), "fixture-task"); err == nil {
			t.Fatal("invalid metrics accepted")
		}
		server.Close()
	}
}

func TestInspectSonarFailsClosedForTaskWarningsAndContractMismatches(t *testing.T) {
	cases := []struct {
		name, route, response string
	}{
		{"missing_warning_count", "/api/ce/task", `{"task":{"componentKey":"fixture","status":"SUCCESS","analysisId":"fixture-analysis"}}`},
		{"pending_task", "/api/ce/task", `{"task":{"componentKey":"fixture","status":"PENDING","analysisId":"fixture-analysis","warningCount":0}}`},
		{"revision_mismatch", "/api/project_analyses/search", `{"analyses":[{"key":"fixture-analysis","revision":"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"}]}`},
		{"issue_total_mismatch", "/api/issues/search", `{"total":0,"paging":{"total":1}}`},
		{"missing_hotspot_count", "/api/hotspots/search", `{"paging":{}}`},
		{"missing_duplication_metric", "/api/measures/component", `{"component":{"measures":[{"metric":"coverage","value":"100"}]}}`},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			server := sonarOverrideServer(test.route, test.response)
			defer server.Close()
			_, err := inspectSonar(context.Background(), server.URL, "fixture", "fixture", strings.Repeat("a", 40), "fixture-task")
			if err == nil {
				t.Fatal("unsafe Sonar contract accepted")
			}
		})
	}
}

func TestSonarTaskWarningsPreventCleanQualification(t *testing.T) {
	server := sonarOverrideServer("/api/ce/task", `{"task":{"componentKey":"fixture","status":"SUCCESS","analysisId":"fixture-analysis","warningCount":1}}`)
	defer server.Close()
	summary, err := inspectSonar(context.Background(), server.URL, "fixture", "fixture", strings.Repeat("a", 40), "fixture-task")
	if err != nil || summary.clean() || summary.Warnings != 1 {
		t.Fatal("Sonar task warning passed qualification")
	}
}

func TestInspectSonarRejectsTaskPermissionFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path == "/api/ce/task" {
			writer.WriteHeader(http.StatusForbidden)
			return
		}
		sonarFixture(writer, request)
	}))
	defer server.Close()
	if _, err := inspectSonar(context.Background(), server.URL, "fixture", "fixture", strings.Repeat("a", 40), "fixture-task"); err == nil {
		t.Fatal("Sonar task permission failure accepted")
	}
}

func sonarOverrideServer(route, response string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path == route {
			_, _ = io.WriteString(writer, response)
			return
		}
		sonarFixture(writer, request)
	}))
}

func TestSonarPrivateStateAndBootstrap(t *testing.T) {
	directory := privateAssuranceDirectory(t, "sonar-state")
	if err := InitSonar(directory); err != nil {
		t.Fatal(err)
	}
	assertBootstrapPassword(t, directory)
	assertBootstrapStateCannotBeOverwritten(t, directory)
	server := httptest.NewServer(http.HandlerFunc(sonarFixture))
	defer server.Close()
	assertBootstrapServerValidation(t, directory, server.URL)
}

func assertBootstrapPassword(t *testing.T, directory string) {
	t.Helper()
	password, err := readToken(filepath.Join(directory, "admin.password"))
	if err != nil || len(password) < 12 || !strings.ContainsAny(password, "ABCDEFGHIJKLMNOPQRSTUVWXYZ") || !strings.ContainsAny(password, "abcdefghijklmnopqrstuvwxyz") || !strings.ContainsAny(password, "0123456789") || !strings.ContainsAny(password, "!@#$%") {
		t.Fatal("bootstrap password does not satisfy server policy")
	}
}

func assertBootstrapStateCannotBeOverwritten(t *testing.T, directory string) {
	t.Helper()
	if err := InitSonar(directory); err == nil {
		t.Fatal("existing bootstrap state overwritten")
	}
	if err := InitSonar(""); err == nil {
		t.Fatal("empty bootstrap directory accepted")
	}
}

func assertBootstrapServerValidation(t *testing.T, directory, endpoint string) {
	t.Helper()
	if err := BootstrapSonar(context.Background(), directory, endpoint, "fixture"); err != nil {
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
	directory := privateAssuranceDirectory(t, "sonar-state")
	if err := InitSonar(directory); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"compose.env", "admin.password"} {
		if err := platform.ValidateFileMetadata(filepath.Join(directory, name), true); err != nil {
			t.Fatal("bootstrap file not private")
		}
	}
}

func TestSonarStrictConditionsAndProfiles(t *testing.T) {
	conditions := map[string]string{}
	languages := map[string]bool{}
	complexity := ""
	observe := func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Error(err)
		}
		switch r.URL.Path {
		case "/api/qualitygates/create_condition", "/api/qualitygates/update_condition":
			conditions[r.Form.Get("metric")] = r.Form.Get("op") + ":" + r.Form.Get("error")
		case "/api/qualityprofiles/add_project":
			languages[r.Form.Get("language")] = true
		}
		if r.URL.Path == "/api/qualityprofiles/activate_rule" && r.Form.Get("rule") == "go:S3776" {
			complexity = r.Form.Get("params")
		}
		sonarFixture(w, r)
	}
	server := httptest.NewServer(http.HandlerFunc(observe))
	defer server.Close()
	client := newSonarClient(server.URL, "synthetic", "")
	configureStrictSonar(t, client)
	assertStrictGate(t, conditions)
	assertStrictProfiles(t, languages, complexity)
}

func configureStrictSonar(t *testing.T, client *sonarClient) {
	t.Helper()
	if err := configureSonarGate(context.Background(), client, "fixture"); err != nil {
		t.Fatal(err)
	}
	if err := configureSonarProfiles(context.Background(), client, "fixture"); err != nil {
		t.Fatal(err)
	}
}

func assertStrictGate(t *testing.T, conditions map[string]string) {
	t.Helper()
	expected := map[string]string{"violations": "GT:0", "new_violations": "GT:0", "coverage": "LT:100", "new_coverage": "LT:100", "duplicated_lines_density": "GT:0", "new_duplicated_lines_density": "GT:0", "security_hotspots_reviewed": "LT:100", "new_security_hotspots_reviewed": "LT:100"}
	if len(conditions) != len(expected) {
		t.Fatal("strict gate conditions incomplete")
	}
	for metric, want := range expected {
		if conditions[metric] != want {
			t.Fatalf("wrong strict threshold for %s", metric)
		}
	}
}

func assertStrictProfiles(t *testing.T, languages map[string]bool, complexity string) {
	t.Helper()
	if len(languages) != 8 || complexity != "threshold=10" {
		t.Fatal("comprehensive profiles or complexity limit missing")
	}
}
