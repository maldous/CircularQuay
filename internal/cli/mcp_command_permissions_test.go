//go:build !windows

package cli

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"

	"github.com/maldous/CircularQuay/internal/model"
	"github.com/maldous/CircularQuay/internal/platform"
)

func TestActionIdentityRequiresProtectedPinnedFile(t *testing.T) {
	directory := privateCLIDirectory(t)
	path := filepath.Join(directory, "synthetic-command")
	contents := []byte("synthetic identity fixture; never executed\n")
	if err := platform.CreateProtectedExclusive(path, contents); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, 0o700); err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(contents)
	workload := model.Workload{Command: []string{path}, CommandSHA256: hex.EncodeToString(digest[:])}
	if !actionIdentityAllowed(workload) {
		t.Fatal("protected pinned identity was denied")
	}
	// Keep the digest unchanged: a content-only check would still admit this
	// group/world-mutable file before repository/provider resolution.
	if err := os.Chmod(path, 0o777); err != nil {
		t.Fatal(err)
	}
	if actionIdentityAllowed(workload) {
		t.Fatal("mutable pinned identity was admitted")
	}
}
