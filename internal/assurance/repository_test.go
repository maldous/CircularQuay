package assurance

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/maldous/CircularQuay/internal/platform"
)

func privateAssuranceDirectory(t *testing.T, name string) string {
	t.Helper()
	base := os.TempDir()
	if runtime.GOOS != "windows" {
		base = "/tmp"
	}
	canonical, err := filepath.EvalSymlinks(base)
	if err != nil {
		t.Fatal(err)
	}
	parent, err := os.MkdirTemp(canonical, "cq-assurance-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(parent) })
	directory := filepath.Join(parent, name)
	if err := platform.EnsurePrivateDirectory(directory); err != nil {
		t.Fatal(err)
	}
	return directory
}

func fixtureRepository(t *testing.T) string {
	t.Helper()
	root := privateAssuranceDirectory(t, "repository")
	for _, name := range requiredFiles {
		if err := platform.WriteProtected(filepath.Join(root, name), []byte("synthetic\n")); err != nil {
			t.Fatal(err)
		}
	}
	if err := platform.WriteProtected(filepath.Join(root, "main.go"), []byte("package main\n")); err != nil {
		t.Fatal(err)
	}
	initializeFixtureGit(t, root)
	return root
}

func initializeFixtureGit(t *testing.T, root string) {
	t.Helper()
	for _, arguments := range [][]string{{"init", "-q"}, {"add", "--all"}, {"-c", "user.name=fixture", "-c", "user.email=fixture@example.invalid", "commit", "-qm", "fixture"}} {
		fixtureGit(t, root, nil, arguments...)
	}
}

func fixtureGit(t *testing.T, root string, environment []string, arguments ...string) []byte {
	t.Helper()
	if environment == nil {
		environment = fixtureGitEnvironment(t)
	}
	ctx, cancel := context.WithTimeout(context.Background(), sourceBindTimeout)
	defer cancel()
	result := runCommand(ctx, "git", arguments, root, environment)
	if result.missing || result.exitCode != 0 {
		t.Fatalf("fixture git failed: %q", arguments)
	}
	return result.output
}

func fixtureGitEnvironment(t *testing.T) []string {
	t.Helper()
	return auditEnvironment(filepath.Join(privateAssuranceDirectory(t, "git-environment"), "home"))
}

func TestRepositoryPolicy(t *testing.T) {
	root := fixtureRepository(t)
	assertRepositoryPasses(t, root)
	assertGitDirectoryIgnored(t, root)
	assertFilesystemCaseCollision(t, root)
	assertMissingMetadataRejected(t, root)
}

func assertRepositoryPasses(t *testing.T, root string) {
	t.Helper()
	if result := repositoryCheck(root); result.Status != "passed" {
		t.Fatalf("valid fixture rejected: %+v", result)
	}
}

func assertGitDirectoryIgnored(t *testing.T, root string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(root, ".git", "invalid:name"), nil, 0600); err != nil {
		t.Fatal(err)
	}
	assertRepositoryPasses(t, root)
}

func assertFilesystemCaseCollision(t *testing.T, root string) {
	t.Helper()
	if !fixtureSupportsDistinctCaseNames(t, root) {
		return
	}
	if result := repositoryCheck(root); result.Code != "repository_naming_violation" {
		t.Fatal("case collision accepted")
	}
	for _, name := range []string{"case-probe", "CASE-PROBE"} {
		if err := os.Remove(filepath.Join(root, name)); err != nil {
			t.Fatal(err)
		}
	}
}

func assertMissingMetadataRejected(t *testing.T, root string) {
	t.Helper()
	if err := os.Remove(filepath.Join(root, "README.md")); err != nil {
		t.Fatal(err)
	}
	if result := repositoryCheck(root); result.Code != "required_metadata_missing" {
		t.Fatal("missing metadata accepted")
	}
}

func TestCaseCollisionInvariant(t *testing.T) {
	seen := map[string]bool{}
	if !uniqueAssetPath(seen, "cmd/cq/main.go") {
		t.Fatal("first portable path rejected")
	}
	if uniqueAssetPath(seen, "cmd/cq/Main.go") {
		t.Fatal("case-folded collision accepted")
	}
}

func fixtureSupportsDistinctCaseNames(t *testing.T, root string) bool {
	t.Helper()
	lower := filepath.Join(root, "case-probe")
	upper := filepath.Join(root, "CASE-PROBE")
	if err := platform.WriteProtected(lower, []byte("synthetic\n")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(upper); err == nil {
		if removeErr := os.Remove(lower); removeErr != nil {
			t.Fatal(removeErr)
		}
		return false
	} else if !os.IsNotExist(err) {
		t.Fatal(err)
	}
	if err := platform.WriteProtected(upper, []byte("synthetic\n")); err != nil {
		t.Fatal(err)
	}
	return true
}

func TestAssetNames(t *testing.T) {
	for _, name := range []string{"main.go", "README.md", ".github", "policy-v1.schema.json", "credential_windows.go"} {
		if !validAssetName(name) {
			t.Fatalf("valid name rejected: %s", name)
		}
	}
	for _, name := range []string{"", " file", "file ", "file.", "two words", "naïve", "NUL", "com1.txt", "LPT9.log", "x:y", "a|b", "a\\b"} {
		if validAssetName(name) {
			t.Fatalf("unsafe name accepted: %s", name)
		}
	}
}

func TestSourceInventory(t *testing.T) {
	root := fixtureRepository(t)
	if err := os.Mkdir(filepath.Join(root, "dist"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "dist", "generated.go"), nil, 0600); err != nil {
		t.Fatal(err)
	}
	files, err := sourceFiles(root)
	if err != nil || len(files) != 1 {
		t.Fatal("incorrect owned source inventory")
	}
	if _, err = sourceFiles(filepath.Join(root, "missing")); err == nil {
		t.Fatal("missing source accepted")
	}
	if _, err = sourceFiles(t.TempDir()); err == nil {
		t.Fatal("empty source accepted")
	}
}
