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
