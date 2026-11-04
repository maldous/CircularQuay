package assurance

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/maldous/CircularQuay/internal/platform"
)

// runForTest keeps legacy source-identity fixtures isolated from production
// execution. Production Run and run always require a verified tool manifest.
func runForTest(ctx context.Context, options Options, command commandFunc) (Report, error) {
	audit, err := prepareAudit(options)
	if err != nil {
		return Report{}, err
	}
	if audit.options.GoBinary == "" {
		audit.options.GoBinary = "go"
	}
	identity, err := bindSource(ctx, runCommand, audit.root, audit.output, audit.environment)
	if err != nil {
		audit.report.Checks = append(audit.report.Checks, failedSourceIdentityCheck("source_identity_before_checks"))
		audit.report.finish()
		return writeFailedReport(audit.output, audit.report)
	}
	audit.report.Source = &identity
	recordSourceIdentity(ctx, command, audit.root, audit.options.GoBinary, audit.environment, &audit.report)
	audit.report.Checks = append(audit.report.Checks, repositoryCheck(audit.root))
	scope, err := sourceGoScope(audit.root, &identity)
	if err != nil {
		audit.report.Checks = append(audit.report.Checks, Check{Name: "source_go_scope", Required: true, Status: "failed", Code: "source_scope_missing"})
		audit.report.finish()
		return writeFailedReport(audit.output, audit.report)
	}
	format, err := checkFormatting(ctx, command, audit.root, audit.options.GoBinary, audit.environment, scope.files)
	if err != nil {
		return audit.report, err
	}
	audit.report.Checks = append(audit.report.Checks, format)
	coveragePath := filepath.Join(audit.output, "coverage.out")
	testArguments := append([]string{"test", "-count=1", "-covermode=atomic", "-coverpkg=" + strings.Join(scope.coverage, ","), "-coverprofile=" + coveragePath}, scope.packages...)
	audit.report.Checks = append(audit.report.Checks, executeGoSourceCheck(ctx, command, "unit_tests", audit.options.GoBinary, testArguments, audit.root, audit.environment, scope, audit.report.Source))
	audit.report.Checks = append(audit.report.Checks, coverageCheck(coveragePath, &audit.report.Coverage))
	audit.report.Checks = append(audit.report.Checks, executeGoSourceCheck(ctx, command, "go_vet", audit.options.GoBinary, append([]string{"vet"}, scope.packages...), audit.root, audit.environment, scope, audit.report.Source))
	addTestExternalChecks(ctx, command, audit.root, audit.environment, audit.options.Profile == "enterprise", scope, &audit.report)
	addSonar(ctx, command, audit.root, audit.environment, audit.options, coveragePath, &audit.report)
	audit.recordAfterIdentity(ctx, identity, runCommand)
	audit.report.finish()
	if err = WriteReport(audit.output, audit.report); err != nil {
		return audit.report, err
	}
	return audit.report, nil
}

func addTestExternalChecks(ctx context.Context, command commandFunc, root string, environment []string, required bool, scope goSourceScope, report *Report) {
	for _, check := range []externalToolCheck{{"staticcheck", "staticcheck", scope.packages}, {"gosec", "gosec", append([]string{"-quiet"}, sourcePackageArguments(report.Source)...)}, {"govulncheck", "govulncheck", scope.packages}, {"secret_scan", "gitleaks", []string{"git", "--redact=100", "--no-banner", "--ignore-gitleaks-allow", "."}}, {"workflow_lint", "actionlint", nil}} {
		report.Checks = append(report.Checks, command.execute(ctx, check.name, required, check.id, check.arguments, root, environment))
	}
}

func TestRunSourceProfileRedactsSubprocessOutput(t *testing.T) {
	root := fixtureRepository(t)
	output := privateAssuranceDirectory(t, "evidence")
	const secret = "synthetic-marker-do-not-log"
	fake := sourceAuditCommand(t, secret)

	report, err := runForTest(context.Background(), Options{Source: root, Output: output, Profile: "source"}, fake)
	if err != nil {
		t.Fatal(err)
	}
	if !report.Passed || report.ProductionQualified || report.Coverage.Percent != 100 {
		t.Fatalf("unexpected report result: %+v", report)
	}
	assertDirtySourceFails(t, root, output, fake)
	data, err := os.ReadFile(filepath.Join(output, "report.json"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), secret) {
		t.Fatal("subprocess value leaked into evidence")
	}
	var decoded Report
	if json.Unmarshal(data, &decoded) != nil || decoded.GoVersion != "go1.27.3" {
		t.Fatal("invalid version evidence")
	}
}

func assertDirtySourceFails(t *testing.T, root, output string, fake commandFunc) {
	t.Helper()
	dirtyCommand := func(ctx context.Context, binary string, args []string, directory string, environment []string) commandResult {
		if binary == "go" && len(args) > 0 && args[0] == "test" {
			if err := os.WriteFile(filepath.Join(root, "main.go"), []byte("package main\n// drift\n"), 0600); err != nil {
				t.Fatal(err)
			}
		}
		return fake(ctx, binary, args, directory, environment)
	}
	dirtyReport, dirtyErr := runForTest(context.Background(), Options{Source: root, Output: output + "-dirty", Profile: "source"}, dirtyCommand)
	if dirtyErr != nil || dirtyReport.Passed || dirtyReport.Coverage.Percent != 100 {
		t.Fatal("uncommitted source passed despite otherwise complete synthetic checks")
	}
}

func sourceAuditCommand(t *testing.T, secret string) commandFunc {
	t.Helper()
	return func(_ context.Context, binary string, args []string, _ string, _ []string) commandResult {
		if binary == "git" {
			return sourceAuditGitResult(args)
		}
		if binary == "go" {
			return sourceAuditGoResult(t, args, secret)
		}
		if binary == "gofmt" {
			return commandResult{}
		}
		return commandResult{missing: true, exitCode: 127, output: []byte(secret)}
	}
}

func sourceAuditGitResult(args []string) commandResult {
	if len(args) > 0 && args[0] == "status" {
		return commandResult{}
	}
	return commandResult{output: []byte(strings.Repeat("a", 40) + "\n")}
}

func sourceAuditGoResult(t *testing.T, args []string, secret string) commandResult {
	t.Helper()
	if len(args) > 0 && args[0] == "version" {
		return commandResult{output: []byte("go version go1.27.3 linux/amd64\n")}
	}
	if len(args) == 0 || args[0] != "test" {
		return commandResult{}
	}
	assertOwnedCoverageScope(t, args)
	writeSyntheticCoverage(t, args)
	return commandResult{output: []byte(secret)}
}

func assertOwnedCoverageScope(t *testing.T, args []string) {
	t.Helper()
	joined := strings.Join(args, " ")
	if strings.Contains(joined, "./...") || !strings.Contains(joined, "-coverpkg="+strings.TrimSuffix(modulePrefix, "/")) {
		t.Fatal("coverage scope is not explicitly bound to owned code")
	}
}

func writeSyntheticCoverage(t *testing.T, args []string) {
	t.Helper()
	for _, arg := range args {
		if strings.HasPrefix(arg, "-coverprofile=") {
			if err := os.WriteFile(strings.TrimPrefix(arg, "-coverprofile="), []byte("mode: atomic\n"+modulePrefix+"main.go:1.1,2.2 1 1\n"), 0600); err != nil {
				t.Fatal(err)
			}
		}
	}
}

func TestEnterpriseCannotPassWithoutSonar(t *testing.T) {
	root := fixtureRepository(t)
	file := filepath.Join(privateAssuranceDirectory(t, "credential"), "token")
	if err := platform.WriteProtected(file, []byte("synthetic-token\n")); err != nil {
		t.Fatal(err)
	}
	report := Report{Commit: strings.Repeat("a", 40)}
	fake := sonarScannerFixture(t)
	coverage := filepath.Join(privateAssuranceDirectory(t, "coverage"), "coverage.out")
	addSonar(context.Background(), fake, root, nil, Options{Profile: "enterprise"}, coverage, &report)
	report.finish()
	if report.Passed {
		t.Fatal("missing mandatory Sonar accepted")
	}
	for _, raw := range []string{"ftp://localhost", "http://example.com", "https://user:token@example.com", "https://example.com/?token=x", "https://example.com/#fragment"} {
		if validSonarURL(raw) {
			t.Fatalf("unsafe URL accepted: %s", raw)
		}
	}
	for _, raw := range []string{"https://sonar.example.com", "http://localhost:9000", "http://127.0.0.1:9000", "http://[::1]:9000"} {
		if !validSonarURL(raw) {
			t.Fatal("safe URL rejected")
		}
	}
	report.Checks = nil
	server := httptest.NewServer(http.HandlerFunc(sonarFixture))
	defer server.Close()
	addSonar(context.Background(), fake, root, nil, Options{Profile: "enterprise", SonarURL: server.URL, SonarProjectKey: "key", SonarTokenFile: file}, coverage, &report)
	if len(report.Checks) != 2 || report.Checks[0].Status != "passed" || report.Checks[1].Status != "passed" {
		t.Fatal("valid scanner invocation rejected")
	}
}

func sonarScannerFixture(t *testing.T) commandFunc {
	t.Helper()
	return func(_ context.Context, binary string, args []string, _ string, _ []string) commandResult {
		if binary == "sonar-scanner" {
			writeSonarTaskMetadata(t, args)
		}
		return commandResult{}
	}
}

func writeSonarTaskMetadata(t *testing.T, args []string) {
	t.Helper()
	project := sonarProjectKey(args)
	for _, arg := range args {
		if strings.HasPrefix(arg, "-Dsonar.scanner.metadataFilePath=") {
			path := strings.TrimPrefix(arg, "-Dsonar.scanner.metadataFilePath=")
			if err := platform.WriteProtected(path, []byte("ceTaskId=fixture-"+project+"task\nprojectKey="+project+"\n")); err != nil {
				t.Fatal(err)
			}
			return
		}
	}
	t.Fatal("scanner metadata path was not configured")
}

func sonarProjectKey(args []string) string {
	for _, arg := range args {
		if strings.HasPrefix(arg, "-Dsonar.projectKey=") {
			return strings.TrimPrefix(arg, "-Dsonar.projectKey=")
		}
	}
	return ""
}

func TestTokenHandling(t *testing.T) {
	directory := privateAssuranceDirectory(t, "token")
	file := filepath.Join(directory, "token")
	assertInvalidTokens(t, file)
	if err := platform.WriteProtected(file, []byte("safe-value")); err != nil {
		t.Fatal(err)
	}
	if token, err := readToken(file); err != nil || token != "safe-value" {
		t.Fatal("protected token rejected")
	}
	if runtime.GOOS != "windows" {
		if err := os.Chmod(file, 0644); err != nil {
			t.Fatal(err)
		}
		if _, err := readToken(file); err == nil {
			t.Fatal("public token accepted")
		}
	}
	if _, err := readToken(directory); err == nil {
		t.Fatal("directory accepted as token")
	}
}

func assertInvalidTokens(t *testing.T, file string) {
	t.Helper()
	for _, value := range []string{"", "a\nb", "x\x00y", strings.Repeat("x", 4097)} {
		if err := platform.WriteProtected(file, []byte(value)); err != nil {
			t.Fatal(err)
		}
		if _, err := readToken(file); err == nil {
			t.Fatal("invalid token accepted")
		}
	}
}

func TestAuditEnvironmentDoesNotInheritCredentials(t *testing.T) {
	t.Setenv("GH_TOKEN", "never-inherit")
	t.Setenv("VAULT_TOKEN", "never-inherit")
	t.Setenv("PATH", "/synthetic/bin")
	home := filepath.Join(t.TempDir(), "home")
	temporary := filepath.Join(filepath.Dir(home), "tmp")
	env := strings.Join(auditEnvironment(home), "\n")
	if strings.Contains(env, "never-inherit") || !strings.Contains(env, "PATH=/synthetic/bin") || !strings.Contains(env, "GOTOOLCHAIN=local") || !strings.Contains(env, "GOWORK=off") || !strings.Contains(env, "GOENV=off") || !strings.Contains(env, "GOTMPDIR="+temporary) || !strings.Contains(env, "-mod=readonly") || !strings.Contains(env, "GIT_NO_REPLACE_OBJECTS=1") || !strings.Contains(env, "GIT_CONFIG_NOSYSTEM=1") {
		t.Fatal("unsafe test environment")
	}
}

func TestAuditOutputProtection(t *testing.T) {
	root := privateAssuranceDirectory(t, "audit")
	if _, err := prepareOutput(root, ""); err == nil {
		t.Fatal("empty output accepted")
	}
	if _, err := prepareOutput(root, root); err == nil {
		t.Fatal("source root output accepted")
	}
	file := filepath.Join(root, "file")
	if err := os.WriteFile(file, nil, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := prepareOutput(root, file); err == nil {
		t.Fatal("file output accepted")
	}
	if _, err := run(context.Background(), Options{Profile: "unknown"}, nil); err == nil {
		t.Fatal("unknown profile accepted")
	}
}

func TestCommandFailureCategories(t *testing.T) {
	for _, test := range []struct {
		result       commandResult
		status, code string
	}{
		{commandResult{}, "passed", ""},
		{commandResult{missing: true, exitCode: 127}, "not_run", "tool_missing"},
		{commandResult{exitCode: 1, output: []byte("never expose")}, "failed", "command_failed"},
	} {
		fake := func(context.Context, string, []string, string, []string) commandResult { return test.result }
		check := commandFunc(fake).execute(context.Background(), "fixture", true, "fixture", nil, "", nil)
		if check.Status != test.status || check.Code != test.code {
			t.Fatal("invalid command classification")
		}
	}
}

func TestBoundedOutput(t *testing.T) {
	var buffer boundedBuffer
	if n, err := buffer.Write([]byte("safe")); err != nil || n != 4 {
		t.Fatal("write failed")
	}
	if n, err := buffer.Write(make([]byte, 1<<20)); err != nil || n != 1<<20 || !buffer.overflow || buffer.String() != "safe" {
		t.Fatal("unbounded output accepted")
	}
}

func TestRunCommandMissing(t *testing.T) {
	result := runCommand(context.Background(), "cquay-nonexistent-audit-tool", nil, t.TempDir(), auditEnvironment(t.TempDir()))
	if !result.missing || result.exitCode != 127 {
		t.Fatal("missing command not classified")
	}
}
