//go:build !windows

package platform

import (
	"github.com/maldous/CircularQuay/internal/fault"
	"os"
	"path/filepath"
	"testing"
)

func TestCreateProtectedExclusiveRejectsSymbolicDestinations(t *testing.T) {
	root := secureTempDir(t)
	outside := filepath.Join(secureTempDir(t), "outside")
	final := filepath.Join(root, "final-link")
	if err := os.Symlink(outside, final); err != nil {
		t.Fatal(err)
	}
	if err := CreateProtectedExclusive(final, []byte("synthetic")); err == nil {
		t.Fatal("existing symbolic destination accepted")
	}
	ancestor := filepath.Join(root, "ancestor-link")
	if err := os.Symlink(filepath.Dir(outside), ancestor); err != nil {
		t.Fatal(err)
	}
	if err := CreateProtectedExclusive(filepath.Join(ancestor, "outside"), []byte("synthetic")); !fault.Is(err, fault.Trust) {
		t.Fatal("symbolic parent accepted")
	}
	if _, err := os.Lstat(outside); !os.IsNotExist(err) {
		t.Fatal("rejected path created a file outside its declared parent")
	}
}

func TestCreateProtectedExclusiveRequiresReplacementProtection(t *testing.T) {
	root := secureTempDir(t)
	if err := os.Chmod(root, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := CreateProtectedExclusive(filepath.Join(root, "private"), []byte("synthetic")); err != nil {
		t.Fatal("trusted publicly readable directory rejected")
	}
	if err := os.Chmod(root, 0o775); err != nil {
		t.Fatal(err)
	}
	if err := CreateProtectedExclusive(filepath.Join(root, "denied"), []byte("synthetic")); !fault.Is(err, fault.Trust) {
		t.Fatal("group-writable creation directory accepted")
	}
}
