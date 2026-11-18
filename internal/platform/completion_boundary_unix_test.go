//go:build !windows

package platform

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/maldous/CircularQuay/internal/fault"
)

func TestCompletionUnixOwnedSecurityAndHelpers(t *testing.T) {
	dir := secureTempDir(t)
	missing := filepath.Join(dir, "absent", "file")
	if checkNativeSecurity(missing, true) {
		t.Fatal("missing path trusted")
	}
	path := completionFile(t, "insecure")
	if err := os.Chmod(path, 0o666); err != nil {
		t.Fatal(err)
	}
	if trustedExecutableAliases(path) {
		t.Fatal("mutable alias admitted")
	}
	if checkNativeSecurity(path, true) {
		t.Fatal("insecure file trusted")
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if trustedDirectory(info) {
		t.Fatal("regular file admitted as directory")
	}
	if syncDirectory(missing) == nil || chmodInParent(missing, 0o600) == nil {
		t.Fatal("missing parent operation succeeded")
	}
}

func TestCompletionUnixActualAliasTraversalBound(t *testing.T) {
	dir := secureTempDir(t)
	// The actual traversal follows this self-link until its forty-link bound.
	if err := os.Symlink("loop", filepath.Join(dir, "loop")); err != nil {
		t.Fatal(err)
	}
	if trustedExecutableAliases(filepath.Join(dir, "missing")) || trustedExecutableAliases(filepath.Join(dir, "loop")) {
		t.Fatal("missing or cyclic alias admitted")
	}
	link := filepath.Join(dir, "temporary")
	if err := os.Symlink("missing", link); err != nil {
		t.Fatal(err)
	}
	info, err := os.Lstat(link)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(link); err != nil {
		t.Fatal(err)
	}
	walk := executableAliasWalk{current: dir}
	if walk.follow(link, info) {
		t.Fatal("removed actual link followed")
	}
}

func TestCompletionUnixDirectoryChangeAfterAdmission(t *testing.T) {
	parent := secureTempDir(t)
	dir := filepath.Join(parent, "directory")
	if err := EnsurePrivateDirectory(dir); err != nil {
		t.Fatal(err)
	}
	err := protectedDestinationWithDirectory(dir, filepath.Join(dir, "target"), func(path string) error {
		if err := EnsurePrivateDirectory(path); err != nil {
			return err
		}
		if err := os.Rename(path, path+"-moved"); err != nil {
			return err
		}
		return os.Symlink(path+"-moved", path)
	})
	completionFault(t, err, fault.Trust, "symbolic destination directory")
	if _, err := os.Lstat(filepath.Join(dir, "target")); !os.IsNotExist(err) {
		t.Fatal("destination touched after directory change")
	}
}
