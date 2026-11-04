package assurance

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/maldous/CircularQuay/internal/platform"
)

func TestAuditGosecScopeExcludesGeneratedOutputPackages(t *testing.T) {
	root := fixtureRepository(t)
	if err := os.Mkdir(filepath.Join(root, "nested"), 0700); err != nil {
		t.Fatal(err)
	}
	commitFixtureFile(t, root, "nested/value.go", []byte("package nested\n"))
	output := filepath.Join(root, "audit-results", "source")
	fake := sourceAuditCommand(t, "synthetic")
	called := false
	command := func(ctx context.Context, binary string, args []string, directory string, environment []string) commandResult {
		if binary == "gosec" {
			called = true
			assertBoundScannerPackages(t, args)
		}
		if binary == "go" && len(args) > 0 && args[0] == "test" {
			writeGeneratedAuditPackage(t, output)
		}
		return fake(ctx, binary, args, directory, environment)
	}
	report, err := runForTest(context.Background(), Options{Source: root, Output: output, Profile: "source"}, command)
	if err != nil || !called || !report.Passed {
		t.Fatalf("bound scanner scope result: called=%t report=%+v err=%v", called, report, err)
	}
}

func TestAuditGoToolsUseManifestScopeAfterOutputGeneratesGo(t *testing.T) {
	root := fixtureRepository(t)
	if err := os.Mkdir(filepath.Join(root, "nested"), 0700); err != nil {
		t.Fatal(err)
	}
	commitFixtureFile(t, root, "nested/value.go", []byte("package nested\n"))
	output := filepath.Join(root, "audit-results", "source")
	fake := sourceAuditCommand(t, "synthetic")
	seen := map[string]bool{}
	command := func(ctx context.Context, binary string, args []string, directory string, environment []string) commandResult {
		switch toolName(binary, args) {
		case "format":
			assertManifestFormatFiles(t, args, root)
			seen["format"] = true
		case "unit_tests":
			writeGeneratedAuditPackage(t, output)
			assertManifestTestScope(t, args, output)
			seen["unit_tests"] = true
		case "go_vet", "staticcheck", "govulncheck":
			name := toolName(binary, args)
			assertManifestPackageScope(t, name, args)
			seen[name] = true
		}
		return fake(ctx, binary, args, directory, environment)
	}
	report, err := runForTest(context.Background(), Options{Source: root, Output: output, Profile: "source"}, command)
	if err != nil || !report.Passed {
		t.Fatalf("manifest tool scope result: report=%+v err=%v", report, err)
	}
	for _, name := range []string{"format", "unit_tests", "go_vet", "staticcheck", "govulncheck"} {
		if !seen[name] {
			t.Fatalf("%s did not receive a manifest-bound scope", name)
		}
	}
}

func TestAuditFailsBeforeTargetToolsWithoutManifestGoScope(t *testing.T) {
	root := privateAssuranceDirectory(t, "repository")
	for _, name := range requiredFiles {
		if err := platform.WriteProtected(filepath.Join(root, name), []byte("synthetic\n")); err != nil {
			t.Fatal(err)
		}
	}
	initializeFixtureGit(t, root)
	targetCalled := false
	fake := sourceAuditCommand(t, "synthetic")
	command := func(ctx context.Context, binary string, args []string, directory string, environment []string) commandResult {
		if toolName(binary, args) != "" {
			targetCalled = true
		}
		return fake(ctx, binary, args, directory, environment)
	}
	report, err := runForTest(context.Background(), Options{Source: root, Output: filepath.Join(root, "audit-results", "source"), Profile: "source"}, command)
	if err != nil || report.Passed || targetCalled || !hasFailedScopeCheck(report) {
		t.Fatalf("empty manifest Go scope result: called=%t report=%+v err=%v", targetCalled, report, err)
	}
}

func TestSourceGoScopeSeparatesVerifiedTaggedIntegration(t *testing.T) {
	root := fixtureRepository(t)
	writeTaggedIntegrationFixture(t, root, "wiremock_test.go", "//go:build integration\n\npackage integration\n")
	writeTaggedIntegrationFixture(t, root, "wiremock_provider_test.go", "//go:build integration\n\npackage integration\n")
	identity := boundFixtureIdentity(t, root)
	scope, err := sourceGoScope(root, &identity)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(scope.packages, []string{"."}) || !slices.Equal(scope.coverage, []string{strings.TrimSuffix(modulePrefix, "/")}) {
		t.Fatalf("untagged scope changed: %#v", scope)
	}
	if !slices.Equal(scope.taggedIntegrationPackages, []string{"./internal/integration"}) {
		t.Fatalf("tagged integration scope = %#v", scope.taggedIntegrationPackages)
	}
	for _, name := range []string{"wiremock_test.go", "wiremock_provider_test.go"} {
		if !slices.Contains(scope.files, filepath.Join(root, "internal", "integration", name)) {
			t.Fatalf("tagged integration file escaped formatting scope: %#v", scope.files)
		}
	}
	check, present := taggedIntegrationQualificationCheck(scope)
	assertRequiredTaggedIntegrationHold(t, check, present)
}

func assertRequiredTaggedIntegrationHold(t *testing.T, check Check, present bool) {
	t.Helper()
	if !present || check.Name != taggedIntegrationCheckName || !check.Required || check.Status != "not_run" || check.Code != taggedIntegrationCheckCode {
		t.Fatalf("tagged integration qualification check = %#v, present=%t", check, present)
	}
}

func TestSourceGoScopeRejectsInvalidTaggedIntegrationFiles(t *testing.T) {
	cases := map[string]string{
		"missing build tag":       "package integration\n",
		"wrong build tag":         "//go:build smoke\n\npackage integration\n",
		"combined build tag":      "//go:build integration && windows\n\npackage integration\n",
		"missing tag separator":   "//go:build integration\npackage integration\n",
		"duplicate build tag":     "//go:build integration\n//go:build integration\n\npackage integration\n",
		"legacy build tag":        "//go:build integration\n// +build integration\n\npackage integration\n",
		"block comment placement": "/* synthetic */\n//go:build integration\n\npackage integration\n",
		"production source":       "//go:build integration\n\npackage integration\nfunc value() {}\n",
	}
	for name, data := range cases {
		t.Run(name, func(t *testing.T) {
			root := fixtureRepository(t)
			filename := "invalid_test.go"
			if name == "production source" {
				filename = "invalid.go"
			}
			writeTaggedIntegrationFixture(t, root, filename, data)
			identity := boundFixtureIdentity(t, root)
			if _, err := sourceGoScope(root, &identity); err == nil {
				t.Fatal("invalid tagged integration file accepted")
			}
		})
	}
}

func TestTaggedIntegrationSourceReadRejectsOversizedFile(t *testing.T) {
	directory := privateAssuranceDirectory(t, "tagged-integration-source")
	filename := filepath.Join(directory, "oversized_test.go")
	if err := os.WriteFile(filename, make([]byte, maxSourceFileBytes+1), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := boundedSourceRead(filename); err == nil {
		t.Fatal("oversized tagged integration source accepted")
	}
}

func TestSourceGoScopeKeepsSimilarIntegrationDirectoryInUntaggedScope(t *testing.T) {
	root := fixtureRepository(t)
	directory := filepath.Join(root, "internal", "integration2")
	if err := os.MkdirAll(directory, 0700); err != nil {
		t.Fatal(err)
	}
	commitFixtureFile(t, root, filepath.Join("internal", "integration2", "ordinary_test.go"), []byte("package integration2\n"))
	identity := boundFixtureIdentity(t, root)
	scope, err := sourceGoScope(root, &identity)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(scope.packages, []string{".", "./internal/integration2"}) || len(scope.taggedIntegrationPackages) != 0 {
		t.Fatalf("similar directory escaped untagged scope: %#v", scope)
	}
}

func TestAuditReportsTaggedIntegrationLaneAsRequiredNotRun(t *testing.T) {
	root := fixtureRepository(t)
	writeTaggedIntegrationFixture(t, root, "wiremock_test.go", "//go:build integration\n\npackage integration\n")
	output := filepath.Join(root, "audit-results", "source")
	fake := sourceAuditCommand(t, "synthetic")
	command := func(ctx context.Context, binary string, args []string, directory string, environment []string) commandResult {
		assertUntaggedAuditCommand(t, binary, args)
		return fake(ctx, binary, args, directory, environment)
	}
	report, err := runForTest(context.Background(), Options{Source: root, Output: output, Profile: "source"}, command)
	if err != nil || report.Passed || report.Coverage.Percent != 100 {
		t.Fatalf("tagged integration report was not an explicit qualification hold: %#v, %v", report, err)
	}
	for _, check := range report.Checks {
		if check.Name == taggedIntegrationCheckName && check.Required && check.Status == "not_run" && check.Code == taggedIntegrationCheckCode {
			return
		}
	}
	t.Fatal("missing required tagged integration qualification check")
}

func assertUntaggedAuditCommand(t *testing.T, binary string, args []string) {
	t.Helper()
	if binary == "go" && len(args) > 0 && (args[0] == "test" || args[0] == "vet") && strings.Contains(strings.Join(args, " "), "./internal/integration") {
		t.Fatal("untagged Go command received tagged integration package")
	}
}

func writeTaggedIntegrationFixture(t *testing.T, root, name, data string) {
	t.Helper()
	directory := filepath.Join(root, "internal", "integration")
	if err := os.MkdirAll(directory, 0700); err != nil {
		t.Fatal(err)
	}
	commitFixtureFile(t, root, filepath.Join("internal", "integration", name), []byte(data))
}

func boundFixtureIdentity(t *testing.T, root string) SourceIdentity {
	t.Helper()
	identity, err := bindSource(context.Background(), runCommand, root, filepath.Join(root, "audit-results", "source"), auditEnvironment(filepath.Join(privateAssuranceDirectory(t, "identity"), "home")))
	if err != nil {
		t.Fatal(err)
	}
	return identity
}

func toolName(binary string, args []string) string {
	if binary == "gofmt" {
		return "format"
	}
	if len(args) == 0 {
		return ""
	}
	if binary == "go" && args[0] == "test" {
		return "unit_tests"
	}
	if binary == "go" && args[0] == "vet" {
		return "go_vet"
	}
	if binary == "staticcheck" || binary == "govulncheck" {
		return binary
	}
	return ""
}

func assertManifestFormatFiles(t *testing.T, args []string, root string) {
	t.Helper()
	want := []string{"-l", filepath.Join(root, "main.go"), filepath.Join(root, "nested", "value.go")}
	if !slices.Equal(args, want) {
		t.Fatalf("format scope escaped manifest Go files: %q", args)
	}
}

func assertManifestTestScope(t *testing.T, args []string, output string) {
	t.Helper()
	coverage := strings.TrimSuffix(modulePrefix, "/") + "," + strings.TrimSuffix(modulePrefix, "/") + "/nested"
	want := []string{"test", "-count=1", "-covermode=atomic", "-coverpkg=" + coverage, "-coverprofile=" + filepath.Join(output, "coverage.out"), ".", "./nested"}
	if !slices.Equal(args, want) {
		t.Fatalf("test or coverage scope escaped manifest packages: %q", args)
	}
}

func assertManifestPackageScope(t *testing.T, name string, args []string) {
	t.Helper()
	want := []string{".", "./nested"}
	if name == "go_vet" {
		want = append([]string{"vet"}, want...)
	}
	if !slices.Equal(args, want) {
		t.Fatalf("package tool scope escaped manifest packages: %q", args)
	}
}

func hasFailedScopeCheck(report Report) bool {
	for _, check := range report.Checks {
		if check.Name == "source_go_scope" && check.Status == "failed" && check.Code == "source_scope_missing" {
			return true
		}
	}
	return false
}

func assertBoundScannerPackages(t *testing.T, args []string) {
	t.Helper()
	if !slices.Equal(args, []string{"-quiet", ".", "./nested"}) {
		t.Fatalf("scanner scope escaped the bound packages: %q", args)
	}
}

func writeGeneratedAuditPackage(t *testing.T, output string) {
	t.Helper()
	directory := filepath.Join(output, "cache", "mod", "synthetic-dependency")
	if err := os.MkdirAll(directory, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(directory, "generated.go"), []byte("package dependency\n"), 0600); err != nil {
		t.Fatal(err)
	}
}
