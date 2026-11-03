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
)

func TestRunSourceProfileRedactsSubprocessOutput(t *testing.T) {
	root := fixtureRepository(t)
	output := filepath.Join(t.TempDir(), "evidence")
	const secret = "synthetic-marker-do-not-log"
	fake := func(_ context.Context, binary string, args []string, _ string, _ []string) commandResult {
		if binary == "git" {
			if len(args) > 0 && args[0] == "status" {
				return commandResult{}
			}
			return commandResult{output: []byte(strings.Repeat("a", 40) + "\n")}
		}
		if binary == "go" && len(args) > 0 && args[0] == "version" {
			return commandResult{output: []byte("go version go1.27.3 linux/amd64\n")}
		}
		if binary == "go" && len(args) > 0 && args[0] == "test" {
			if !strings.Contains(strings.Join(args, " "), "-coverpkg="+modulePrefix+"...") {
				t.Fatal("coverage scope is not explicitly bound to owned code")
			}
			for _, arg := range args {
				if strings.HasPrefix(arg, "-coverprofile=") {
					if err := os.WriteFile(strings.TrimPrefix(arg, "-coverprofile="), []byte("mode: atomic\n"+modulePrefix+"main.go:1.1,2.2 1 1\n"), 0600); err != nil {
						t.Fatal(err)
					}
				}
			}
			return commandResult{output: []byte(secret)}
		}
		if binary == "gofmt" || binary == "go" {
			return commandResult{}
		}
		return commandResult{missing: true, exitCode: 127, output: []byte(secret)}
	}
	report, err := run(context.Background(), Options{Source: root, Output: output, Profile: "source"}, fake)
	if err != nil {
		t.Fatal(err)
	}
	if !report.Passed || report.ProductionQualified || report.Coverage.Percent != 100 {
		t.Fatalf("unexpected report result: %+v", report)
	}
	dirtyCommand := func(ctx context.Context, binary string, args []string, directory string, environment []string) commandResult {
		if binary == "git" && len(args) > 0 && args[0] == "status" {
			return commandResult{output: []byte(" M source.go\n")}
		}
		return fake(ctx, binary, args, directory, environment)
	}
	dirtyReport, dirtyErr := run(context.Background(), Options{Source: root, Output: output + "-dirty", Profile: "source"}, dirtyCommand)
	if dirtyErr != nil || dirtyReport.Passed || dirtyReport.Coverage.Percent != 100 {
		t.Fatal("uncommitted source passed despite otherwise complete synthetic checks")
	}
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

func TestEnterpriseCannotPassWithoutSonar(t *testing.T) {
	root := fixtureRepository(t)
	file := filepath.Join(t.TempDir(), "token")
	if err := os.WriteFile(file, []byte("synthetic-token\n"), 0600); err != nil {
		t.Fatal(err)
	}
	var report Report
	fake := func(context.Context, string, []string, string, []string) commandResult { return commandResult{} }
	addSonar(context.Background(), fake, root, nil, Options{Profile: "enterprise"}, "coverage.out", &report)
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
	addSonar(context.Background(), fake, root, nil, Options{Profile: "enterprise", SonarURL: server.URL, SonarProjectKey: "key", SonarTokenFile: file}, "coverage.out", &report)
	if len(report.Checks) != 2 || report.Checks[0].Status != "passed" || report.Checks[1].Status != "passed" {
		t.Fatal("valid scanner invocation rejected")
	}
}

func TestTokenHandling(t *testing.T) {
	directory := t.TempDir()
	file := filepath.Join(directory, "token")
	for _, value := range []string{"", "a\nb", "x\x00y", strings.Repeat("x", 4097)} {
		if err := os.WriteFile(file, []byte(value), 0600); err != nil {
			t.Fatal(err)
		}
		if _, err := readToken(file); err == nil {
			t.Fatal("invalid token accepted")
		}
	}
	if err := os.WriteFile(file, []byte("safe-value"), 0600); err != nil {
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

func TestAuditEnvironmentDoesNotInheritCredentials(t *testing.T) {
	t.Setenv("GH_TOKEN", "never-inherit")
	t.Setenv("VAULT_TOKEN", "never-inherit")
	t.Setenv("PATH", "/synthetic/bin")
	env := strings.Join(auditEnvironment("/private/home"), "\n")
	if strings.Contains(env, "never-inherit") || !strings.Contains(env, "PATH=/synthetic/bin") || !strings.Contains(env, "GOTOOLCHAIN=local") {
		t.Fatal("unsafe test environment")
	}
}

func TestAuditOutputProtection(t *testing.T) {
	root := t.TempDir()
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
		check := execute(context.Background(), fake, "fixture", true, "fixture", nil, "", nil)
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
