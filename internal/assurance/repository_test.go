package assurance

import (
	"os"
	"path/filepath"
	"testing"
)

func fixtureRepository(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	for _, name := range requiredFiles {
		if err := os.WriteFile(filepath.Join(root, name), []byte("synthetic\n"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(root, "main.go"), []byte("package main\n"), 0600); err != nil {
		t.Fatal(err)
	}
	return root
}

func TestRepositoryPolicy(t *testing.T) {
	root := fixtureRepository(t)
	if result := repositoryCheck(root); result.Status != "passed" {
		t.Fatalf("valid fixture rejected: %+v", result)
	}
	if err := os.Mkdir(filepath.Join(root, ".git"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".git", "invalid:name"), nil, 0600); err != nil {
		t.Fatal(err)
	}
	if result := repositoryCheck(root); result.Status != "passed" {
		t.Fatal("git internals were scanned")
	}
	if err := os.WriteFile(filepath.Join(root, "Main.go"), nil, 0600); err != nil {
		t.Fatal(err)
	}
	if result := repositoryCheck(root); result.Code != "repository_naming_violation" {
		t.Fatal("case collision accepted")
	}
	if err := os.Remove(filepath.Join(root, "README.md")); err != nil {
		t.Fatal(err)
	}
	if result := repositoryCheck(root); result.Code != "required_metadata_missing" {
		t.Fatal("missing metadata accepted")
	}
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
