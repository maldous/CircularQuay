package assurance

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Exercise the production stage path rather than the legacy runForTest copy.
// The oracle is the persisted report, not a sequence of mock command calls.
func TestProductionStageReportContract(t *testing.T) {
	for _, test := range []struct {
		name, profile, failedStage string
		wantPassed                 bool
	}{
		{"source retains unselected analyzers", "source", "", true},
		{"unit failure remains required", "source", "unit_tests", false},
		{"uncovered statements remain required", "source", "statement_coverage", false},
		{"vet failure remains required", "source", "go_vet", false},
		{"enterprise analyzer failure is not qualification", "enterprise", "gosec", false},
	} {
		t.Run(test.name, func(t *testing.T) {
			audit, tools, scope := stageContractFixture(t, test.profile, test.failedStage)
			audit.recordChecks(context.Background(), tools, Check{Name: "format", Required: true, Status: "passed"}, scope)
			audit.report.finish()
			if err := WriteReport(audit.output, audit.report); err != nil {
				t.Fatal(err)
			}
			data, err := os.ReadFile(filepath.Join(audit.output, "report.json"))
			if err != nil {
				t.Fatal(err)
			}
			if strings.Contains(string(data), stageContractPrivateMarker) {
				t.Fatal("raw command diagnostics leaked into the report")
			}
			var report Report
			if err = json.Unmarshal(data, &report); err != nil {
				t.Fatal(err)
			}
			if report.Passed != test.wantPassed || report.ProductionQualified {
				t.Fatalf("incorrect audit acceptance: passed=%v production_qualified=%v", report.Passed, report.ProductionQualified)
			}
			for _, name := range []string{"format", "unit_tests", "statement_coverage", "go_vet"} {
				status, code := "passed", ""
				if name == test.failedStage {
					status, code = "failed", "command_failed"
					if name == "statement_coverage" {
						code = "coverage_below_100_percent"
					}
				}
				check := stageContractCheck(t, report, name, status, code, true)
				if name == test.failedStage && name != "statement_coverage" {
					if check.ExitCode != 9 || check.Failure == nil || check.Failure.State != "observed" || len(check.Failure.Locations) != 1 || check.Failure.Locations[0] != (GoFailureLocation{Path: "main.go", Line: 1}) {
						t.Fatalf("required Go failure lost its exit status or bounded location: %+v", check)
					}
				}
			}
			wantCovered := int64(1)
			if test.failedStage == "statement_coverage" {
				wantCovered = 0
			}
			if report.Coverage.Total != 1 || report.Coverage.Covered != wantCovered {
				t.Fatalf("coverage was skipped or replaced by a command result: %+v", report.Coverage)
			}
			for _, name := range []string{"staticcheck", "gosec", "govulncheck", "secret_scan", "workflow_lint"} {
				status, code := "not_run", "tool_not_selected"
				if test.profile == "enterprise" {
					status, code = "passed", ""
					if name == test.failedStage {
						status, code = "failed", "command_failed"
					}
				}
				stageContractCheck(t, report, name, status, code, test.profile == "enterprise")
			}
			if test.profile == "enterprise" {
				stageContractCheck(t, report, "sonarqube", "failed", "sonar_runtime_binding_unavailable", true)
			} else {
				stageContractCheck(t, report, "sonarqube", "not_run", "sonar_not_configured", false)
			}
			stageContractCheck(t, report, "native_reboot_and_restore", "not_run", "requires_qualified_native_environment", false)
			stageContractCheck(t, report, "independent_review", "not_run", "requires_independent_reviewer", false)
		})
	}
}

const stageContractPrivateMarker = "synthetic-private-command-diagnostic"

func stageContractFixture(t *testing.T, profile, failedStage string) (auditRun, *qualifiedTools, goSourceScope) {
	t.Helper()
	verified, err := LoadToolManifest(writeManifest(t, manifestFixture(t)), fixtureLocations(t))
	if err != nil {
		t.Fatal(err)
	}
	if profile == "source" {
		var selected []VerifiedTool
		for _, tool := range verified.Tools {
			if tool.ID == "go" || tool.ID == "git" {
				selected = append(selected, tool)
			}
		}
		verified.Tools = selected
	}
	root := privateAssuranceDirectory(t, "source")
	output := privateAssuranceDirectory(t, "evidence")
	home := privateAssuranceDirectory(t, "home")
	if err = os.WriteFile(filepath.Join(root, "main.go"), []byte("package main\nfunc main() {}\n"), 0600); err != nil {
		t.Fatal(err)
	}
	audit := auditRun{
		root: root, output: output,
		options:     Options{Profile: profile, GoBinary: toolPath(verified, "go")},
		environment: []string{"HOME=" + home},
		report: Report{Version: 1, Profile: profile, ProductionQualified: true,
			Source: &SourceIdentity{Files: []SourceFile{{Path: "main.go"}}}},
	}
	// Synthetic executable bytes are digest-checked, but never executed. Only
	// this callback supplies fixed version probes and fabricated stage results.
	command := func(_ context.Context, binary string, args []string, _ string, _ []string) commandResult {
		id := toolIDForPath(verified, binary)
		if id == "unknown" {
			t.Fatal("production stage selected an undeclared executable")
		}
		if len(args) > 0 && (args[0] == "version" || args[0] == "--version") {
			return commandResult{output: []byte("version-output-" + id)}
		}
		stage := id
		if id == "go" && len(args) > 0 {
			if args[0] == "test" {
				stage = "unit_tests"
				count := "1"
				if failedStage == "statement_coverage" {
					count = "0"
				}
				coverage := "mode: atomic\n" + modulePrefix + "main.go:1.1,1.2 1 " + count + "\n"
				if err := os.WriteFile(filepath.Join(output, "coverage.out"), []byte(coverage), 0600); err != nil {
					t.Fatal(err)
				}
			} else if args[0] == "vet" {
				stage = "go_vet"
			}
		}
		result := commandResult{output: []byte("main.go:1:1: " + stageContractPrivateMarker + "\n")}
		if stage == failedStage {
			result.exitCode = 9
			result.goFailure = parseGoFailureCapture(args, result.output, false)
		}
		return result
	}
	return audit, newQualifiedTools(verified, command), goSourceScope{packages: []string{"."}, coverage: []string{strings.TrimSuffix(modulePrefix, "/")}}
}

func stageContractCheck(t *testing.T, report Report, name, status, code string, required bool) Check {
	t.Helper()
	var found []Check
	for _, check := range report.Checks {
		if check.Name == name {
			found = append(found, check)
		}
	}
	if len(found) != 1 {
		t.Fatalf("report must contain exactly one %s check, got %d", name, len(found))
	}
	check := found[0]
	if check.Status != status || check.Code != code || check.Required != required {
		t.Fatalf("%s outcome = %+v; want status=%s code=%s required=%v", name, check, status, code, required)
	}
	return check
}
