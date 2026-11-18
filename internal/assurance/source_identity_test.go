package assurance

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestSourceIdentityRejectsIgnoredGoFileBeforeCommands(t *testing.T) {
	root := fixtureRepository(t)
	commitFixtureFile(t, root, ".gitignore", []byte("ignored_test.go\n"))
	if err := os.WriteFile(filepath.Join(root, "ignored_test.go"), []byte("package main\n"), 0600); err != nil {
		t.Fatal(err)
	}
	called := false
	command := func(context.Context, string, []string, string, []string) commandResult {
		called = true
		return commandResult{}
	}
	output := privateAssuranceDirectory(t, "evidence")
	report, err := runForTest(context.Background(), Options{Source: root, Output: output, Profile: "source"}, command)
	if err != nil || called || report.Passed || !hasFailedSourceIdentity(report, "source_identity_before_checks") {
		t.Fatalf("ignored source result = %+v, called = %t, error = %v", report, called, err)
	}
	if _, err = os.Lstat(filepath.Join(output, "report.json")); err != nil {
		t.Fatalf("failed source evidence was not retained: %v", err)
	}
}

func TestSourceIdentityRejectsSymlinkedSourceBeforeCommands(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("native Windows reparse-point qualification is separate")
	}
	root := fixtureRepository(t)
	commitFixtureFile(t, root, ".gitignore", []byte("ignored_test.go\n"))
	if err := os.WriteFile(filepath.Join(root, "ignored_test.go"), []byte("package main\n"), 0600); err != nil {
		t.Fatal(err)
	}
	alias := filepath.Join(privateAssuranceDirectory(t, "aliases"), "source")
	if err := os.Symlink(root, alias); err != nil {
		t.Fatal(err)
	}
	called := false
	command := func(context.Context, string, []string, string, []string) commandResult {
		called = true
		return commandResult{}
	}
	report, err := runForTest(context.Background(), Options{Source: alias, Output: privateAssuranceDirectory(t, "evidence"), Profile: "source"}, command)
	if err != nil || called || report.Passed || !hasFailedSourceIdentity(report, "source_identity_before_checks") {
		t.Fatalf("symlink source result = %+v, called = %t, error = %v", report, called, err)
	}
}

func TestSourceIdentityDetectsPostCommandDrift(t *testing.T) {
	root := fixtureRepository(t)
	output := privateAssuranceDirectory(t, "evidence")
	command := sourceAuditCommand(t, "synthetic")
	drift := func(ctx context.Context, binary string, args []string, directory string, environment []string) commandResult {
		if binary == "go" && len(args) > 0 && args[0] == "test" {
			if err := os.WriteFile(filepath.Join(root, "main.go"), []byte("package main\n// drift\n"), 0600); err != nil {
				t.Fatal(err)
			}
		}
		return command(ctx, binary, args, directory, environment)
	}
	report, err := runForTest(context.Background(), Options{Source: root, Output: output, Profile: "source"}, drift)
	if err != nil || report.Passed || !hasFailedSourceIdentity(report, "source_identity_after_checks") {
		t.Fatalf("post-command drift result = %+v, %v", report, err)
	}
}

func TestSourceIdentityAllowsSelectedPrivateOutput(t *testing.T) {
	root := fixtureRepository(t)
	output := filepath.Join(root, ".local", "audit-results", "fixture")
	report, err := runForTest(context.Background(), Options{Source: root, Output: output, Profile: "source"}, sourceAuditCommand(t, "synthetic"))
	if err != nil || !report.Passed || report.Source == nil || report.Source.ManifestSHA256 == "" {
		t.Fatalf("private output result = %+v, %v", report, err)
	}
}

func TestSourceIdentityDisablesGitReplacementObjects(t *testing.T) {
	root := fixtureRepository(t)
	fenced := fixtureGitEnvironment(t)
	head := strings.TrimSpace(string(fixtureGit(t, root, fenced, "rev-parse", "HEAD")))
	tree := strings.TrimSpace(string(fixtureGit(t, root, fenced, "rev-parse", "HEAD^{tree}")))
	commitFixtureFile(t, root, "main.go", []byte("package main\n// replacement\n"))
	replacement := strings.TrimSpace(string(fixtureGit(t, root, nil, "rev-parse", "HEAD")))
	fixtureGit(t, root, nil, "replace", head, replacement)
	unfenced := withoutEnvironment(fenced, "GIT_NO_REPLACE_OBJECTS")
	if got := strings.TrimSpace(string(fixtureGit(t, root, unfenced, "rev-parse", "HEAD^{tree}"))); got == tree {
		t.Fatal("replacement regression did not change an unfenced tree")
	}
	fixtureGit(t, root, fenced, "reset", "--hard", "-q", head)
	output := privateAssuranceDirectory(t, "evidence")
	identity, err := bindSource(context.Background(), runCommand, root, output, auditEnvironment(filepath.Join(output, "home")))
	if err != nil || identity.Commit != head || identity.Tree != tree {
		t.Fatalf("replacement object changed source identity: %+v, %v", identity, err)
	}
}

func withoutEnvironment(environment []string, key string) []string {
	prefix := key + "="
	result := make([]string, 0, len(environment))
	for _, value := range environment {
		if !strings.HasPrefix(value, prefix) {
			result = append(result, value)
		}
	}
	return result
}

func TestSourceIdentityRejectsRepositorySubdirectoryBeforeCommands(t *testing.T) {
	root := fixtureRepository(t)
	directory := filepath.Join(root, "nested")
	if err := os.Mkdir(directory, 0700); err != nil {
		t.Fatal(err)
	}
	commitFixtureFile(t, root, "nested/main.go", []byte("package nested\n"))
	called := false
	command := func(context.Context, string, []string, string, []string) commandResult {
		called = true
		return commandResult{}
	}
	report, err := runForTest(context.Background(), Options{Source: directory, Output: privateAssuranceDirectory(t, "evidence"), Profile: "source"}, command)
	if err != nil || called || report.Passed || !hasFailedSourceIdentity(report, "source_identity_before_checks") {
		t.Fatalf("subdirectory source result = %+v, called = %t, error = %v", report, called, err)
	}
}

func TestSourceIdentityPathAndWorktreeMetadataGuards(t *testing.T) {
	for _, path := range []string{"../outside", "a/../outside", "a\\b", "C:drive", "/absolute", "a\x00b", "a\tb", "a\x7fb", "a/b"} {
		if got, want := safeRepositoryPath(path), path == "a/b"; got != want {
			t.Fatalf("path %q accepted = %t, want %t", path, got, want)
		}
	}
	root := privateAssuranceDirectory(t, "worktree")
	metadata := privateAssuranceDirectory(t, "metadata")
	path := filepath.Join(root, ".git")
	if err := os.WriteFile(path, []byte("gitdir: "+metadata+"\n"), 0600); err != nil {
		t.Fatal(err)
	}
	entry, err := os.ReadDir(root)
	if err != nil || len(entry) != 1 || !validGitWorktreeFile(root, path) {
		t.Fatal("valid worktree metadata rejected")
	}
	inventory := sourceFileInventory{root: root, tracked: map[string]bool{}}
	if err = inventory.visit(path, entry[0], nil); err != nil {
		t.Fatalf("valid worktree metadata was not excluded: %v", err)
	}
}

func TestSourceIdentityRejectsOversizedRead(t *testing.T) {
	root := privateAssuranceDirectory(t, "oversized")
	path := filepath.Join(root, "source.go")
	if err := os.WriteFile(path, make([]byte, maxSourceFileBytes+1), 0600); err != nil {
		t.Fatal(err)
	}
	info, err := os.Lstat(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = fileSHA256(root, "source.go", info, sourceFileByteLimit("source.go"), nil); err == nil {
		t.Fatal("oversized source read accepted")
	}
}

func TestSourceIdentityAllowsDeclaredRasterBeyondTextLimit(t *testing.T) {
	root := fixtureRepository(t)
	raster := rasterFixtureBytes(maxSourceFileBytes + 1)
	commitFixtureFile(t, root, selectedIconContactSheet, raster)
	output := privateAssuranceDirectory(t, "evidence")
	identity, err := bindSource(context.Background(), runCommand, root, output, auditEnvironment(filepath.Join(output, "home")))
	want := sha256.Sum256(raster)
	file, found := sourceFileAt(identity.Files, selectedIconContactSheet)
	if err != nil || !found || file.SHA256 != hex.EncodeToString(want[:]) {
		t.Fatalf("declared raster identity = %#v, %v", identity, err)
	}
}

func TestSourceIdentityRejectsDeclaredRasterOverBound(t *testing.T) {
	root := fixtureRepository(t)
	commitFixtureFile(t, root, selectedIconContactSheet, rasterFixtureBytes(maxSourceRasterBytes+1))
	output := privateAssuranceDirectory(t, "evidence")
	if _, err := bindSource(context.Background(), runCommand, root, output, auditEnvironment(filepath.Join(output, "home"))); err == nil {
		t.Fatal("oversized declared raster accepted")
	}
}

func TestSourceIdentityRejectsDeclaredRasterWithoutPNGSignature(t *testing.T) {
	root := fixtureRepository(t)
	commitFixtureFile(t, root, selectedIconContactSheet, make([]byte, maxSourceFileBytes+1))
	output := privateAssuranceDirectory(t, "evidence")
	if _, err := bindSource(context.Background(), runCommand, root, output, auditEnvironment(filepath.Join(output, "home"))); err == nil {
		t.Fatal("declared raster without PNG signature accepted")
	}
}

func TestSourceIdentityRejectsDeclaredRasterBlobMismatch(t *testing.T) {
	root := fixtureRepository(t)
	raster := rasterFixtureBytes(maxSourceFileBytes + 1)
	commitFixtureFile(t, root, selectedIconContactSheet, raster)
	raster[len(raster)-1] = 1
	if err := os.WriteFile(filepath.Join(root, selectedIconContactSheet), raster, 0600); err != nil {
		t.Fatal(err)
	}
	output := privateAssuranceDirectory(t, "evidence")
	if _, err := bindSource(context.Background(), runCommand, root, output, auditEnvironment(filepath.Join(output, "home"))); err == nil {
		t.Fatal("declared raster Git blob mismatch accepted")
	}
}

func TestSourceIdentityRetainsTextFileLimit(t *testing.T) {
	root := fixtureRepository(t)
	commitFixtureFile(t, root, "large.go", make([]byte, maxSourceFileBytes+1))
	output := privateAssuranceDirectory(t, "evidence")
	if _, err := bindSource(context.Background(), runCommand, root, output, auditEnvironment(filepath.Join(output, "home"))); err == nil {
		t.Fatal("oversized text source accepted")
	}
}

func TestSourceIdentityRejectsDeclaredRasterReadGrowth(t *testing.T) {
	path := filepath.Join(privateAssuranceDirectory(t, "raster-growth"), "raster.png")
	if err := os.WriteFile(path, rasterFixtureBytes(maxSourceRasterBytes+1), 0600); err != nil {
		t.Fatal(err)
	}
	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	if _, err = boundedFileSHA256(file, maxSourceRasterBytes, maxSourceRasterBytes); err == nil {
		t.Fatal("read growth past declared raster limit accepted")
	}
}

func rasterFixtureBytes(size int64) []byte {
	value := make([]byte, size)
	copy(value, []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'})
	return value
}

func sourceFileAt(files []SourceFile, path string) (SourceFile, bool) {
	for _, file := range files {
		if file.Path == path {
			return file, true
		}
	}
	return SourceFile{}, false
}

func commitFixtureFile(t *testing.T, root, name string, content []byte) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(filepath.Join(root, name)), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, name), content, 0600); err != nil {
		t.Fatal(err)
	}
	initializeFixtureGit(t, root)
}

func hasFailedSourceIdentity(report Report, name string) bool {
	for _, check := range report.Checks {
		if check.Name == name && check.Status == "failed" && check.Code == "source_identity_invalid" {
			return true
		}
	}
	return false
}

func TestSourceIdentityRejectsOutsideOutputOverlap(t *testing.T) {
	root := fixtureRepository(t)
	_, err := prepareOutput(root, filepath.Join(root, ".local", "other"))
	if err == nil || err.Error() != "audit output must not overlap source" {
		t.Fatalf("unexpected output path accepted: %v", err)
	}
}
