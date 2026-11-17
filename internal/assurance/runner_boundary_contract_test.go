package assurance

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/maldous/CircularQuay/internal/platform"
)

func TestRunnerBoundaryContractManifestBeforePaths(t *testing.T) {
	directory := t.TempDir()
	options := Options{
		Source:   filepath.Join(directory, "absent-source"),
		Output:   filepath.Join(directory, "absent-output"),
		Profile:  "unknown",
		GoBinary: filepath.Join(directory, "absent-go"),
	}
	report, err := Run(context.Background(), options)
	requireRunnerBoundaryError(t, err, "tool manifest is required")
	if !reflect.DeepEqual(report, Report{}) {
		t.Fatal("missing manifest returned audit evidence")
	}
	entries, err := os.ReadDir(directory)
	if err != nil || len(entries) != 0 {
		t.Fatal("missing manifest created source or output artifacts")
	}
}

type runnerBoundaryBindingCase struct {
	name      string
	ids       []string
	goBinary  string
	wantGo    string
	wantError string
}

func TestRunnerBoundaryContractMandatoryToolBinding(t *testing.T) {
	directory := t.TempDir()
	goPath := filepath.Join(directory, "go")
	otherPath := filepath.Join(directory, "other-go")
	for _, scenario := range []runnerBoundaryBindingCase{
		{"missing Go", nil, otherPath, otherPath, "go tool missing"},
		{"missing Git", []string{"go", "gofmt"}, otherPath, otherPath, "required tool missing"},
		{"missing gofmt", []string{"go", "git"}, otherPath, otherPath, "required tool missing"},
		{"different Go", []string{"go", "git", "gofmt"}, otherPath, otherPath, "go tool does not match manifest"},
		{"default Go", []string{"go", "git", "gofmt"}, "", goPath, ""},
		{"matching Go", []string{"go", "git", "gofmt"}, goPath, goPath, ""},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			assertRunnerBoundaryToolBinding(t, directory, scenario)
		})
	}
}

func assertRunnerBoundaryToolBinding(t *testing.T, directory string, scenario runnerBoundaryBindingCase) {
	t.Helper()
	// These fabricated records exercise only the internal metadata-binding
	// decision. They are not loaded manifests or qualified executable identities.
	var selected []VerifiedTool
	for _, id := range scenario.ids {
		selected = append(selected, VerifiedTool{ID: id, Path: filepath.Join(directory, id)})
	}
	calls := 0
	tools := newQualifiedTools(VerifiedToolManifest{Tools: selected}, func(context.Context, string, []string, string, []string) commandResult {
		calls++
		return commandResult{}
	})
	audit := auditRun{options: Options{Profile: "source", GoBinary: scenario.goBinary}}
	want := audit
	want.options.GoBinary = scenario.wantGo
	err := audit.bindMandatoryTools(tools)
	requireRunnerBoundaryError(t, err, scenario.wantError)
	if !reflect.DeepEqual(audit, want) {
		t.Fatal("mandatory tool binding changed more than the successful default Go path")
	}
	if calls != 0 || len(tools.probes) != 0 {
		t.Fatal("metadata binding invoked or probed an executable")
	}
}

func TestRunnerBoundaryContractFormattingNeedsScope(t *testing.T) {
	calls := 0
	command := func(context.Context, string, []string, string, []string) commandResult {
		calls++
		return commandResult{}
	}
	check, err := checkFormatting(context.Background(), command, "synthetic-source-not-read", "synthetic-go-not-read", nil, nil)
	requireRunnerBoundaryError(t, err, "source scope has no Go files")
	if check != (Check{}) || calls != 0 {
		t.Fatal("empty formatting scope invoked a tool or produced check evidence")
	}
}

func TestRunnerBoundaryContractMalformedCoverageKeepsPriorSummary(t *testing.T) {
	directory := privateAssuranceDirectory(t, "runner-invalid-coverage")
	profile := filepath.Join(directory, "coverage.out")
	data := []byte("mode: atomic\nsynthetic-private-untrusted-coverage-tail\n")
	if err := platform.WriteProtected(profile, data); err != nil {
		t.Fatal(err)
	}
	prior := Coverage{Covered: 3, Total: 5, Percent: 60, Files: 2}
	summary := prior
	check := coverageCheck(profile, &summary)
	want := Check{Name: "statement_coverage", Required: true, Status: "failed", Code: "coverage_missing_or_invalid"}
	if check != want {
		t.Fatal("malformed protected coverage did not return the fixed required refusal")
	}
	if summary != prior {
		t.Fatal("malformed protected coverage replaced the prior summary")
	}
}

func TestRunnerBoundaryContractExternalChecksNeedScope(t *testing.T) {
	calls := 0
	tools := newQualifiedTools(VerifiedToolManifest{}, func(context.Context, string, []string, string, []string) commandResult {
		calls++
		return commandResult{}
	})
	prior := Check{Name: "prior", Status: "not_run", Code: "synthetic"}
	report := Report{Checks: []Check{prior}}
	addExternalChecks(context.Background(), tools, "synthetic-root-not-read", nil, false, goSourceScope{}, &report)
	want := Report{Checks: []Check{prior, {Name: "gosec", Required: true, Status: "failed", Code: "source_scope_missing"}}}
	if !reflect.DeepEqual(report, want) {
		t.Fatal("empty analyzer scope did not preserve prior evidence and append the required refusal")
	}
	if calls != 0 || len(tools.probes) != 0 {
		t.Fatal("empty analyzer scope invoked or probed a tool")
	}
}

func requireRunnerBoundaryError(t *testing.T, err error, want string) {
	t.Helper()
	if want == "" {
		if err != nil {
			t.Fatal("valid metadata binding was refused")
		}
		return
	}
	if err == nil || err.Error() != want {
		t.Fatalf("error = %v; want the fixed refusal %q", err, want)
	}
}
