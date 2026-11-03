package assurance

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/maldous/CircularQuay/internal/platform"
)

func TestQualifiedToolsRejectsDigestBeforeVersionProbe(t *testing.T) {
	manifest := manifestFixture(t)
	verified, err := LoadToolManifest(writeManifest(t, manifest), fixtureLocations(t))
	if err != nil {
		t.Fatal(err)
	}
	verified.Tools[0].SHA256 = digest([]byte("wrong"))
	calls := 0
	tools := newQualifiedTools(verified, func(context.Context, string, []string, string, []string) commandResult {
		calls++
		return commandResult{}
	})
	result := tools.run(context.Background(), "git", []string{"status"}, t.TempDir(), nil)
	if result.exitCode == 0 || calls != 0 || tools.probes["git"].Code != "binary_digest_mismatch" {
		t.Fatal("digest mismatch launched a command")
	}
}

func TestQualifiedToolsRejectsWrongVersionBeforeCommand(t *testing.T) {
	manifest := manifestFixture(t)
	verified, err := LoadToolManifest(writeManifest(t, manifest), fixtureLocations(t))
	if err != nil {
		t.Fatal(err)
	}
	calls := 0
	tools := newQualifiedTools(verified, func(context.Context, string, []string, string, []string) commandResult {
		calls++
		return commandResult{output: []byte("different version")}
	})
	result := tools.run(context.Background(), "git", []string{"status"}, t.TempDir(), nil)
	if result.exitCode == 0 || calls != 1 || tools.probes["git"].Code != "version_output_mismatch" {
		t.Fatal("wrong version reached the requested command")
	}
}

func TestFileURLPreservesUnixAndWindowsLocalPaths(t *testing.T) {
	for _, test := range []struct {
		path    string
		windows bool
		want    string
	}{
		{"/private/vuln db/#snapshot", false, "file:///private/vuln%20db/%23snapshot"},
		{`C:\private\vuln db\#snapshot`, true, "file:///C:/private/vuln%20db/%23snapshot"},
	} {
		if got := fileURLForPath(test.path, test.windows); got != test.want {
			t.Fatalf("file URL for %q = %q, want %q", test.path, got, test.want)
		}
	}
}

func TestExternalCheckRejectsRuleInputDriftBeforeLaunchingAnalyzer(t *testing.T) {
	manifest := manifestFixture(t)
	verified, err := LoadToolManifest(writeManifest(t, manifest), fixtureLocations(t))
	if err != nil {
		t.Fatal(err)
	}
	input := verified.Tools[6].DeclaredRules.Input.Path // gitleaks fixture
	if err = os.WriteFile(input, []byte("changed\n"), 0600); err != nil {
		t.Fatal(err)
	}
	calls := 0
	tools := newQualifiedTools(verified, func(_ context.Context, binary string, _ []string, _ string, _ []string) commandResult {
		if binary == verified.Tools[6].Path {
			calls++
		}
		return commandResult{output: []byte("version-output-gitleaks")}
	})
	home := filepath.Join(t.TempDir(), "home")
	if err := platform.EnsurePrivateDirectory(home); err != nil {
		t.Fatal(err)
	}
	check := externalCheck(context.Background(), tools, t.TempDir(), []string{"HOME=" + home}, false, externalToolCheck{name: "secret_scan", id: "gitleaks"})
	if check.Code != "rules_input_drift_before_execution" || calls != 0 {
		t.Fatalf("changed input launched analyzer or passed: %+v calls=%d", check, calls)
	}
}

func TestExternalCheckRejectsRuleInputDriftAfterAnalyzer(t *testing.T) {
	manifest := manifestFixture(t)
	verified, err := LoadToolManifest(writeManifest(t, manifest), fixtureLocations(t))
	if err != nil {
		t.Fatal(err)
	}
	input := verified.Tools[6].DeclaredRules.Input.Path // gitleaks fixture
	tools := newQualifiedTools(verified, func(_ context.Context, binary string, args []string, _ string, _ []string) commandResult {
		if binary == verified.Tools[6].Path && len(args) > 0 && args[0] == "git" {
			if err := os.WriteFile(input, []byte("changed\n"), 0600); err != nil {
				t.Fatal(err)
			}
		}
		return commandResult{output: []byte("version-output-" + toolIDForPath(verified, binary))}
	})
	home := filepath.Join(t.TempDir(), "home")
	if err := platform.EnsurePrivateDirectory(home); err != nil {
		t.Fatal(err)
	}
	check := externalCheck(context.Background(), tools, t.TempDir(), []string{"HOME=" + home}, false, externalToolCheck{name: "secret_scan", id: "gitleaks"})
	if check.Code != "rules_input_drift_after_execution" {
		t.Fatalf("changed input result accepted: %+v", check)
	}
}

func toolIDForPath(manifest VerifiedToolManifest, path string) string {
	for _, tool := range manifest.Tools {
		if tool.Path == path {
			return tool.ID
		}
	}
	return "unknown"
}

func TestExternalCheckRejectsSourceControlledWaiversBeforeAnalyzer(t *testing.T) {
	for _, test := range []struct {
		name, id, path, content string
	}{
		{"gitleaks ignore", "gitleaks", ".gitleaksignore", "ignored\n"},
		{"staticcheck config", "staticcheck", "staticcheck.conf", "checks = [\"all\"]\n"},
		{"gosec config", "gosec", ".gosec.json", "{}\n"},
		{"actionlint config", "actionlint", ".github/actionlint.yaml", "ignore:\n  - SC2086\n"},
		{"staticcheck comment", "staticcheck", "sample.go", "package sample\n//lint:ignore SA0000 rationale\nfunc sample() {}\n"},
		{"gosec comment", "gosec", "sample.go", "package sample\n// #nosec G204 rationale\nfunc sample() {}\n"},
	} {
		t.Run(test.name, func(t *testing.T) {
			assertSourceWaiverDenied(t, test.id, test.path, test.content)
		})
	}
}

func assertSourceWaiverDenied(t *testing.T, id, relative, content string) {
	t.Helper()
	root := t.TempDir()
	path := filepath.Join(root, relative)
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	manifest := manifestFixture(t)
	verified, err := LoadToolManifest(writeManifest(t, manifest), fixtureLocations(t))
	if err != nil {
		t.Fatal(err)
	}
	check, launched := sourceWaiverCheck(verified, root, id)
	if check.Code != "source_controlled_analyzer_waiver" || !check.Required || launched {
		t.Fatalf("waiver was not denied before launch: %+v launched=%v", check, launched)
	}
}

func sourceWaiverCheck(manifest VerifiedToolManifest, root, id string) (Check, bool) {
	target := toolPath(manifest, id)
	launched := false
	tools := newQualifiedTools(manifest, func(_ context.Context, binary string, _ []string, _ string, _ []string) commandResult {
		launched = launched || binary == target
		return commandResult{}
	})
	return externalCheck(context.Background(), tools, root, nil, true, externalToolCheck{name: id, id: id}), launched
}

func toolPath(manifest VerifiedToolManifest, id string) string {
	for _, tool := range manifest.Tools {
		if tool.ID == id {
			return tool.Path
		}
	}
	return ""
}

func TestSourceAnalyzerWaiverIgnoresQuotedMarker(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "sample.go")
	content := "package sample\nconst fixture = \"//lint:ignore SA0000\"\n"
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	if sourceAnalyzerWaiver(root, "staticcheck") {
		t.Fatal("quoted fixture marker was treated as a suppression comment")
	}
}
