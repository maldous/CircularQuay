//go:build !windows

package platform

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/maldous/CircularQuay/internal/fault"
)

func TestPlatformFileProcessBoundarySymbolicParentRefuses(t *testing.T) {
	path, data, before := platformFileProcessFixture(t)
	alias := filepath.Join(secureTempDir(t), "parent-alias")
	if err := os.Symlink(filepath.Dir(path), alias); err != nil {
		t.Fatal(err)
	}
	err := ValidateProtectedPath(filepath.Join(alias, filepath.Base(path)))
	assertPlatformFileProcessFault(t, err, fault.Trust, "symbolic protected path")
	assertPlatformFileProcessUnchanged(t, path, data, before)
}

func TestPlatformFileProcessBoundaryReadonlyStageClosesWithoutWrite(t *testing.T) {
	path, data, before := platformFileProcessFixture(t)
	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = file.Close() })
	err = writeAndSync(file, []byte("must not be written"))
	assertPlatformFileProcessFault(t, err, fault.Apply, "write staged file")
	if _, err := file.Stat(); !errors.Is(err, os.ErrClosed) {
		t.Fatal("failed write left the readonly stage descriptor open")
	}
	assertPlatformFileProcessUnchanged(t, path, data, before)
}
