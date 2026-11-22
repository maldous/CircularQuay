//go:build !windows

package assurance

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/sys/unix"
)

func TestResidualInventoryContractOwnedSymlinkRefusedBeforeTraversal(t *testing.T) {
	root := privateAssuranceDirectory(t, "residual-symlink")
	target := sourceInputOwnedFile(t, root, "owned.go", []byte("package fixture\n"))
	link := filepath.Join(root, "linked.go")
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	entry := sourceInputEntry(t, link)
	assets := assetValidator{root: root, seen: map[string]bool{}}
	if err := assets.visit(link, entry, nil); err == nil || err.Error() != "repository symlink forbidden" || len(assets.seen) != 0 {
		t.Fatal("repository callback followed an owned symlink")
	}
	inventory := &sourceInventory{}
	if err := inventory.visit(link, entry, nil); err == nil || err.Error() != "source symlink forbidden" || len(inventory.files) != 0 {
		t.Fatal("Go inventory followed an owned symlink")
	}
	assertResidualSourceSymlink(t, root, link, entry)
	toolExecutionBoundaryBytes(t, target, []byte("package fixture\n"))
}

func assertResidualSourceSymlink(t *testing.T, root, link string, entry fs.DirEntry) {
	t.Helper()
	bound := sourceFileInventory{root: root, tracked: map[string]bool{"linked.go": true}}
	if err := bound.validate(link, entry); err == nil || err.Error() != "source symlink forbidden" {
		t.Fatal("tracked name concealed an owned symlink")
	}
	if err := skipGitMetadata(root, link, entry); err == nil || err.Error() != "source git metadata is a symlink" {
		t.Fatal("Git metadata callback followed an owned symlink")
	}
}

func TestResidualInventoryContractFIFOIsRefusedWithoutOpen(t *testing.T) {
	root := privateAssuranceDirectory(t, "residual-fifo")
	path := filepath.Join(root, "owned.pipe")
	if err := unix.Mkfifo(path, 0600); err != nil {
		t.Fatal(err)
	}
	files, err := protectedTreeFiles(root)
	if err == nil || err.Error() != "invalid tool rule tree" || len(files) != 0 {
		t.Fatal("rule inventory admitted or opened a nonregular owned FIFO")
	}
}
