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

	"github.com/maldous/CircularQuay/internal/platform"
)

func TestFormattingAndCoverageChecksFailClosed(t *testing.T) {
	root := fixtureRepository(t)
	assertFormattingDrift(t, root)
	assertCoverageChecksFailClosed(t)
}

func assertFormattingDrift(t *testing.T, root string) {
	t.Helper()
	changed := func(context.Context, string, []string, string, []string) commandResult {
		return commandResult{output: []byte("main.go\n")}
	}
	check, err := checkFormatting(context.Background(), changed, root, "go", nil, []string{filepath.Join(root, "main.go")})
	if err != nil || check.Status != "failed" || check.Code != "format_changes_required" {
		t.Fatalf("format drift result = %+v, %v", check, err)
	}
}

func assertCoverageChecksFailClosed(t *testing.T) {
	t.Helper()
	directory := privateAssuranceDirectory(t, "coverage")
	profile := filepath.Join(directory, "coverage.out")
	partial := "mode: atomic\n" + modulePrefix + "main.go:1.1,2.1 1 0\n"
	if err := platform.WriteProtected(profile, []byte(partial)); err != nil {
		t.Fatal(err)
	}
	var summary Coverage
	if got := coverageCheck(profile, &summary); got.Code != "coverage_below_100_percent" || summary.Total != 1 {
		t.Fatalf("partial coverage result = %+v, %+v", got, summary)
	}
	full := "mode: atomic\n" + modulePrefix + "main.go:1.1,2.1 1 1\n"
	if err := platform.WriteProtected(profile, []byte(full)); err != nil {
		t.Fatal(err)
	}
	if got := coverageCheck(profile, &summary); got.Status != "passed" || summary.Percent != 100 {
		t.Fatalf("complete coverage result = %+v, %+v", got, summary)
	}
	if got := coverageCheck(filepath.Join(directory, "missing.out"), &summary); got.Code != "coverage_missing_or_invalid" {
		t.Fatalf("missing coverage result = %+v", got)
	}
}

func TestSonarMetadataRejectsAmbiguousTaskIdentity(t *testing.T) {
	directory := privateAssuranceDirectory(t, "metadata")
	for name, data := range map[string]string{
		"valid":     "ceTaskId=task-1\nprojectKey=fixture\n",
		"duplicate": "ceTaskId=task-1\nceTaskId=task-2\n",
		"malformed": "ceTaskId=bad value\n",
	} {
		path := filepath.Join(directory, name)
		if err := platform.WriteProtected(path, []byte(data)); err != nil {
			t.Fatal(err)
		}
		got, err := sonarMetadataTask(path)
		if name == "valid" && (err != nil || got != "task-1") {
			t.Fatalf("valid metadata = %q, %v", got, err)
		}
		if name != "valid" && err == nil {
			t.Fatalf("unsafe metadata accepted: %s", name)
		}
	}
}

func TestSonarMetadataCreationCannotImplyCompletedAnalysis(t *testing.T) {
	directory := privateAssuranceDirectory(t, "new-metadata")
	path, err := newSonarMetadata(directory)
	if err != nil {
		t.Fatal(err)
	}
	if err = platform.ValidateFileMetadata(path, false); err != nil {
		t.Fatalf("metadata protection = %v", err)
	}
	if _, err = sonarMetadataTask(path); err == nil {
		t.Fatal("empty scanner metadata implied a completed task")
	}
}

func TestCreateSonarProjectVerifiesExistingIdentity(t *testing.T) {
	for name, component := range map[string]string{"matching": "fixture", "mismatch": "other"} {
		t.Run(name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/api/projects/create":
					w.WriteHeader(http.StatusConflict)
				case "/api/components/show":
					_, _ = io.WriteString(w, `{"component":{"key":"`+component+`"}}`)
				default:
					w.WriteHeader(http.StatusNotFound)
				}
			}))
			defer server.Close()
			err := createSonarProject(context.Background(), newSonarClient(server.URL, "synthetic", ""), "fixture")
			if (err == nil) != (name == "matching") {
				t.Fatalf("existing project validation error = %v", err)
			}
		})
	}
}

func TestSonarConditionUsesExistingIdentityAndProfilesFailClosed(t *testing.T) {
	var route string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		route = r.URL.Path
		if err := r.ParseForm(); err != nil {
			t.Error(err)
		}
		if r.Form.Get("id") != "condition-1" || r.Form.Get("gateName") != "" {
			t.Error("existing condition was not updated by identity")
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()
	client := newSonarClient(server.URL, "synthetic", "")
	condition := sonarGateCondition{Metric: "coverage", Operator: "LT", Threshold: "100"}
	if err := setSonarCondition(context.Background(), client, "fixture", condition, []sonarGateCondition{{ID: "condition-1", Metric: "coverage"}}); err != nil || route != "/api/qualitygates/update_condition" {
		t.Fatalf("existing condition update = %q, %v", route, err)
	}
	profiles := []sonarQualityProfile{{Key: "go", Name: "Sonar way comprehensive", Language: "go", BuiltIn: true}}
	if _, err := completeSonarProfile(profiles, "go"); err == nil {
		t.Fatal("empty Go profile accepted")
	}
	if _, err := completeSonarProfile(profiles, "json"); err == nil {
		t.Fatal("missing language profile accepted")
	}
}

func TestSonarClientRejectsInvalidJSONWithoutReflectingBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, "synthetic-private-body")
	}))
	defer server.Close()
	var destination struct{}
	err := newSonarClient(server.URL, "synthetic", "").call(context.Background(), http.MethodGet, "/api/example", url.Values{}, &destination)
	if err == nil || strings.Contains(err.Error(), "synthetic-private-body") {
		t.Fatal("invalid Sonar body became reportable detail")
	}
}

func TestSonarAuditTokenDoesNotPersistMalformedCredential(t *testing.T) {
	directory := privateAssuranceDirectory(t, "scanner-token")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/user_tokens/generate" {
			_, _ = io.WriteString(w, `{"token":""}`)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()
	err := createSonarAuditToken(context.Background(), newSonarClient(server.URL, "admin", "synthetic"), directory, "fixture", defaultSonarTokenValidityDays)
	if err == nil {
		t.Fatal("empty scanner credential accepted")
	}
	if _, statErr := os.Lstat(filepath.Join(directory, "scanner.token")); !os.IsNotExist(statErr) {
		t.Fatalf("malformed scanner credential persisted: %v", statErr)
	}
}
