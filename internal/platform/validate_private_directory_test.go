package platform

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestValidatePrivateDirectoryReadOnlyTrust(t *testing.T) {
	root := secureTempDir(t)
	private := filepath.Join(root, "private")
	if err := EnsurePrivateDirectory(private); err != nil {
		t.Fatal(err)
	}
	if err := ValidatePrivateDirectory(private); err != nil {
		t.Fatalf("private directory rejected: %v", err)
	}
	file := filepath.Join(root, "file")
	if err := os.WriteFile(file, nil, 0600); err != nil {
		t.Fatal(err)
	}
	if err := ValidatePrivateDirectory(file); err == nil {
		t.Fatal("regular file accepted as private directory")
	}
	if err := ValidatePrivateDirectory(filepath.Join(root, "missing")); err == nil {
		t.Fatal("missing directory accepted")
	}
}

func TestValidatePrivateDirectoryRejectsUnsafeAliases(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("native Windows reparse fixture requires junction creation")
	}
	root := secureTempDir(t)
	private := filepath.Join(root, "private")
	if err := EnsurePrivateDirectory(private); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, "link")
	if err := os.Symlink(private, link); err != nil {
		t.Fatal(err)
	}
	if err := ValidatePrivateDirectory(link); err == nil {
		t.Fatal("symbolic directory accepted")
	}
	insecure := filepath.Join(root, "insecure")
	if err := os.Mkdir(insecure, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(insecure, 0755); err != nil {
		t.Fatal(err)
	}
	if err := ValidatePrivateDirectory(insecure); err == nil {
		t.Fatal("group-readable directory accepted")
	}
}
