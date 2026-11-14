package assurance

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestHelperCopyStaysNonExecutableUntilDigestVerification(t *testing.T) {
	root := privateAssuranceDirectory(t, "helper-copy")
	name := canonicalToolName("git")
	source := writeTool(t, filepath.Join(root, name), "synthetic helper\n")
	target := filepath.Join(ensurePrivateDirectory(t, filepath.Join(root, "copies")), name)
	if err := writeHelper(source, target); err != nil {
		t.Fatal(err)
	}
	assertCopiedHelper(t, source, target)
	tool := VerifiedTool{ID: "git", Path: source, SHA256: digest([]byte("synthetic helper\n"))}
	if existingHelperMatches(tool, target) && runtime.GOOS != "windows" {
		t.Fatal("non-executable copy accepted as a usable helper")
	}
	if err := enableHelper(target); err != nil {
		t.Fatal(err)
	}
	if !existingHelperMatches(tool, target) {
		t.Fatal("complete protected executable copy rejected")
	}
}

func assertCopiedHelper(t *testing.T, source, target string) {
	t.Helper()
	original, err := os.ReadFile(source)
	if err != nil {
		t.Fatal(err)
	}
	copied, err := os.ReadFile(target)
	if err != nil || string(original) != string(copied) {
		t.Fatalf("helper bytes changed: %v", err)
	}
	info, err := os.Stat(target)
	if err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS != "windows" && info.Mode().Perm() != 0600 {
		t.Fatalf("unverified helper mode is %o", info.Mode().Perm())
	}
}

func TestHelperCopyRejectsOversizedSourceWithoutCreatingTarget(t *testing.T) {
	root := privateAssuranceDirectory(t, "oversized-helper")
	source := writeTool(t, filepath.Join(root, canonicalToolName("git")), "fixture")
	if err := os.Truncate(source, maxToolBinaryBytes+1); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(ensurePrivateDirectory(t, filepath.Join(root, "copies")), canonicalToolName("git"))
	if err := writeHelper(source, target); err == nil {
		t.Fatal("oversized helper copied")
	}
	if _, err := os.Lstat(target); !os.IsNotExist(err) {
		t.Fatalf("rejected helper left a target: %v", err)
	}
}

func TestHelperCopyRefusesToReplaceExistingTarget(t *testing.T) {
	root := privateAssuranceDirectory(t, "existing-helper")
	source := writeTool(t, filepath.Join(root, canonicalToolName("git")), "fixture")
	target := writeTool(t, filepath.Join(ensurePrivateDirectory(t, filepath.Join(root, "copies")), canonicalToolName("git")), "preserved")
	if err := writeHelper(source, target); err == nil {
		t.Fatal("existing helper overwritten")
	}
	data, err := os.ReadFile(target)
	if err != nil || string(data) != "preserved" {
		t.Fatalf("existing helper changed: %v", err)
	}
}

func TestManifestRunCannotClaimUnboundSonarPass(t *testing.T) {
	for _, test := range []struct {
		profile, url, status, code string
		required                   bool
	}{
		{"source", "", "not_run", "sonar_not_configured", false},
		{"source", "https://sonar.invalid", "not_run", "sonar_runtime_binding_unavailable", false},
		{"enterprise", "", "failed", "sonar_runtime_binding_unavailable", true},
		{"enterprise", "https://sonar.invalid", "failed", "sonar_runtime_binding_unavailable", true},
	} {
		t.Run(test.profile+test.url, func(t *testing.T) {
			var report Report
			addSonarWithTools(Options{Profile: test.profile, SonarURL: test.url}, &report)
			if len(report.Checks) != 1 {
				t.Fatalf("unexpected checks: %+v", report.Checks)
			}
			check := report.Checks[0]
			if check.Status != test.status || check.Required != test.required || check.Code != test.code {
				t.Fatalf("unbound Sonar result: %+v", check)
			}
		})
	}
}
