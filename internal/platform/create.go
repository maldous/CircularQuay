package platform

import (
	"os"
	"path/filepath"

	"github.com/maldous/CircularQuay/internal/fault"
)

// CreateProtectedExclusive creates one bounded owner-only file without
// replacing an existing name. The parent must already exist, be owned by a
// trusted identity, and deny untrusted replacement. Public read access to the
// parent is permitted; the new file is protected before any data is written.
// A failed write removes only this call's file when it still occupies the name.
func CreateProtectedExclusive(path string, data []byte) error {
	return createProtectedExclusiveWithOperations(path, data, defaultCreationBoundaryOperations())
}

func createProtectedExclusiveWithOperations(path string, data []byte, ops creationBoundaryOperations) error {
	if !filepath.IsAbs(path) || len(data) > MaxDocumentSize {
		return fault.New(fault.Invalid, "protected creation destination")
	}
	root, err := openCreationRoot(filepath.Dir(path))
	if err != nil {
		return err
	}
	defer root.Close()
	name := filepath.Base(path)
	file, err := ops.open(root, name, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return fault.New(fault.Apply, "create protected file")
	}
	created, err := file.Stat()
	if err != nil {
		_ = file.Close()
		return fault.New(fault.Apply, "inspect created file")
	}
	complete := false
	defer func() {
		_ = file.Close()
		if !complete {
			removeCreatedFile(root, name, created)
		}
	}()
	if err := ops.populate(file, path, data); err != nil {
		return err
	}
	if err := ops.sync(filepath.Dir(path)); err != nil {
		return fault.New(fault.Apply, "sync created file directory")
	}
	complete = true
	return nil
}

func populateCreatedFile(file *os.File, path string, data []byte) error {
	return populateCreatedFileWithProtection(file, path, data, protectFile)
}

func populateCreatedFileWithProtection(file *os.File, path string, data []byte, protect func(string) error) error {
	if err := protect(path); err != nil {
		return fault.New(fault.Apply, "protect created file")
	}
	if !openedRegularFileMatches(file, path) || ValidateProtectedPath(path) != nil {
		return fault.New(fault.Trust, "created file changed")
	}
	return writeAndSync(file, data)
}

func removeCreatedFile(root *os.Root, name string, created os.FileInfo) {
	named, err := root.Lstat(name)
	if err == nil && named.Mode().IsRegular() && os.SameFile(created, named) {
		_ = root.Remove(name)
	}
}

func openCreationRoot(directory string) (*os.Root, error) {
	return openCreationRootWithOperation(directory, os.OpenRoot)
}

func openCreationRootWithOperation(directory string, open func(string) (*os.Root, error)) (*os.Root, error) {
	before, err := os.Lstat(directory)
	if err != nil || !before.IsDir() || !secureMode(before.Mode(), false) || !checkNativeSecurity(directory, false) || !resolvedPathIsOriginal(directory) {
		return nil, fault.New(fault.Trust, "protected creation directory")
	}
	root, err := open(directory)
	if err != nil {
		return nil, fault.New(fault.Apply, "open protected creation directory")
	}
	opened, err := root.Stat(".")
	if err != nil || !os.SameFile(before, opened) {
		_ = root.Close()
		return nil, fault.New(fault.Trust, "creation directory changed")
	}
	return root, nil
}
