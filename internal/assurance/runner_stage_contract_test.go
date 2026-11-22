package assurance

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type stageContractScenario struct {
	name, profile, failedStage string
	wantPassed                 bool
}

// Exercise the production stage path rather than the legacy runForTest copy.
// The oracle is the persisted report, not a sequence of mock command calls.
func TestProductionStageReportContract(t *testing.T) {
	for _, scenario := range []stageContractScenario{
		{"source retains unselected analyzers", "source", "", true},
		{"unit failure remains required", "source", "unit_tests", false},
		{"uncovered statements remain required", "source", "statement_coverage", false},
		{"vet failure remains required", "source", "go_vet", false},
		{"enterprise analyzer failure is not qualification", "enterprise", "gosec", false},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			audit, tools, scope := stageContractFixture(t, scenario.profile, scenario.failedStage)
			audit.recordChecks(context.Background(), tools, Check{Name: "format", Required: true, Status: "passed"}, scope)
			audit.report.finish()
			report := stageContractPersistedReport(t, audit)
			assertStageContractAcceptance(t, report, scenario)
			assertStageContractRequiredChecks(t, report, scenario.failedStage)
			assertStageContractCoverage(t, report, scenario.failedStage)
			assertStageContractAnalyzers(t, report, scenario)
			assertStageContractBoundaries(t, report, scenario.profile)
		})
	}
}

func stageContractPersistedReport(t *testing.T, audit auditRun) Report {
	t.Helper()
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
	return report
}

func assertStageContractAcceptance(t *testing.T, report Report, scenario stageContractScenario) {
	t.Helper()
	if report.Passed != scenario.wantPassed || report.ProductionQualified {
		t.Fatalf("incorrect audit acceptance: passed=%v production_qualified=%v", report.Passed, report.ProductionQualified)
	}
}

func assertStageContractRequiredChecks(t *testing.T, report Report, failedStage string) {
	t.Helper()
	for _, name := range []string{"format", "unit_tests", "statement_coverage", "go_vet"} {
		status, code := stageContractRequiredOutcome(name, failedStage)
		check := stageContractCheck(t, report, name, status, code, true)
		if name == failedStage && name != "statement_coverage" {
			assertStageContractFailure(t, check)
		}
	}
}

func stageContractRequiredOutcome(name, failedStage string) (string, string) {
	if name != failedStage {
		return "passed", ""
	}
	if name == "statement_coverage" {
		return "failed", "coverage_below_100_percent"
	}
	return "failed", "command_failed"
}

func assertStageContractFailure(t *testing.T, check Check) {
	t.Helper()
	if check.ExitCode != 9 || check.Failure == nil {
		t.Fatalf("required Go failure lost its exit status or bounded location: %+v", check)
	}
	if check.Failure.State != "observed" || len(check.Failure.Locations) != 1 {
		t.Fatalf("required Go failure lost its bounded location: %+v", check)
	}
	if check.Failure.Locations[0] != (GoFailureLocation{Path: "main.go", Line: 1}) {
		t.Fatalf("required Go failure has the wrong location: %+v", check)
	}
}

func assertStageContractCoverage(t *testing.T, report Report, failedStage string) {
	t.Helper()
	wantCovered := int64(1)
	if failedStage == "statement_coverage" {
		wantCovered = 0
	}
	if report.Coverage.Total != 1 || report.Coverage.Covered != wantCovered {
		t.Fatalf("coverage was skipped or replaced by a command result: %+v", report.Coverage)
	}
}

func assertStageContractAnalyzers(t *testing.T, report Report, scenario stageContractScenario) {
	t.Helper()
	for _, name := range []string{"staticcheck", "gosec", "govulncheck", "secret_scan", "workflow_lint"} {
		status, code, required := stageContractAnalyzerOutcome(name, scenario)
		stageContractCheck(t, report, name, status, code, required)
	}
}

func stageContractAnalyzerOutcome(name string, scenario stageContractScenario) (string, string, bool) {
	if scenario.profile != "enterprise" {
		return "not_run", "tool_not_selected", false
	}
	if name == scenario.failedStage {
		return "failed", "command_failed", true
	}
	return "passed", "", true
}

func assertStageContractBoundaries(t *testing.T, report Report, profile string) {
	t.Helper()
	if profile == "enterprise" {
		stageContractCheck(t, report, "sonarqube", "failed", "sonar_runtime_binding_unavailable", true)
	} else {
		stageContractCheck(t, report, "sonarqube", "not_run", "sonar_not_configured", false)
	}
	stageContractCheck(t, report, "native_reboot_and_restore", "not_run", "requires_qualified_native_environment", false)
	stageContractCheck(t, report, "independent_review", "not_run", "requires_independent_reviewer", false)
}

const stageContractPrivateMarker = "synthetic-private-command-diagnostic"

func stageContractFixture(t *testing.T, profile, failedStage string) (auditRun, *qualifiedTools, goSourceScope) {
	t.Helper()
	verified := stageContractSelectedTools(t, profile)
	root, output, home := stageContractDirectories(t)
	audit := auditRun{
		root: root, output: output,
		options:     Options{Profile: profile, GoBinary: toolPath(verified, "go")},
		environment: []string{"HOME=" + home},
		report: Report{Version: 1, Profile: profile, ProductionQualified: true,
			Source: &SourceIdentity{Files: []SourceFile{{Path: "main.go"}}}},
	}
	command := stageContractCommand(t, verified, output, failedStage)
	return audit, newQualifiedTools(verified, command), goSourceScope{packages: []string{"."}, coverage: []string{strings.TrimSuffix(modulePrefix, "/")}}
}

func stageContractSelectedTools(t *testing.T, profile string) VerifiedToolManifest {
	t.Helper()
	verified, err := LoadToolManifest(writeManifest(t, manifestFixture(t)), fixtureLocations(t))
	if err != nil {
		t.Fatal(err)
	}
	if profile != "source" {
		return verified
	}
	var selected []VerifiedTool
	for _, tool := range verified.Tools {
		if tool.ID == "go" || tool.ID == "git" {
			selected = append(selected, tool)
		}
	}
	verified.Tools = selected
	return verified
}

func stageContractDirectories(t *testing.T) (string, string, string) {
	t.Helper()
	root := privateAssuranceDirectory(t, "source")
	output := privateAssuranceDirectory(t, "evidence")
	home := privateAssuranceDirectory(t, "home")
	if err := os.WriteFile(filepath.Join(root, "main.go"), []byte("package main\nfunc main() {}\n"), 0600); err != nil {
		t.Fatal(err)
	}
	return root, output, home
}

// Synthetic executable bytes are digest-checked, but never executed. Only
// this callback supplies fixed version probes and fabricated stage results.
func stageContractCommand(t *testing.T, verified VerifiedToolManifest, output, failedStage string) commandFunc {
	t.Helper()
	return func(_ context.Context, binary string, args []string, _ string, _ []string) commandResult {
		id := toolIDForPath(verified, binary)
		if id == "unknown" {
			t.Fatal("production stage selected an undeclared executable")
		}
		if stageContractVersionProbe(args) {
			return commandResult{output: []byte("version-output-" + id)}
		}
		stage := id
		if id == "go" {
			stage = stageContractGoStage(t, args, output, failedStage)
		}
		return stageContractResult(stage, failedStage, args)
	}
}

func stageContractVersionProbe(args []string) bool {
	return len(args) > 0 && (args[0] == "version" || args[0] == "--version")
}

func stageContractGoStage(t *testing.T, args []string, output, failedStage string) string {
	t.Helper()
	if len(args) == 0 {
		return "go"
	}
	switch args[0] {
	case "test":
		writeStageContractCoverage(t, output, failedStage)
		return "unit_tests"
	case "vet":
		return "go_vet"
	default:
		return "go"
	}
}

func writeStageContractCoverage(t *testing.T, output, failedStage string) {
	t.Helper()
	count := "1"
	if failedStage == "statement_coverage" {
		count = "0"
	}
	coverage := "mode: atomic\n" + modulePrefix + "main.go:1.1,1.2 1 " + count + "\n"
	if err := os.WriteFile(filepath.Join(output, "coverage.out"), []byte(coverage), 0600); err != nil {
		t.Fatal(err)
	}
}

func stageContractResult(stage, failedStage string, args []string) commandResult {
	result := commandResult{output: []byte("main.go:1:1: " + stageContractPrivateMarker + "\n")}
	if stage == failedStage {
		result.exitCode = 9
		result.goFailure = parseGoFailureCapture(args, result.output, false)
	}
	return result
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
