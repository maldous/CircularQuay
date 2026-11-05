// Package platform implements narrowly scoped native delivery primitives.
package platform

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/maldous/CircularQuay/internal/fault"
	"github.com/maldous/CircularQuay/internal/model"
)

// MaxDocumentSize bounds secret documents and native subprocess output.
const MaxDocumentSize = 2 << 20

var appendMu sync.Mutex

// EnsurePrivateDirectory establishes a local owner-controlled state directory.
// Existing insecure directories fail rather than having their grants rewritten.
func EnsurePrivateDirectory(path string) error {
	if !filepath.IsAbs(path) {
		return fault.New(fault.Invalid, "absolute private directory required")
	}
	created, err := privateDirectoryAbsent(path)
	if err != nil {
		return fault.New(fault.Trust, "inspect private directory")
	}
	if err := os.MkdirAll(path, 0o700); err != nil {
		return fault.New(fault.Apply, "create private directory")
	}
	if !resolvedPathIsOriginal(path) {
		return fault.New(fault.Trust, "symbolic private directory")
	}
	if err := protectNewDirectory(path, created); err != nil {
		return err
	}
	if !privateDirectorySecure(path) {
		return fault.New(fault.Trust, "private directory permissions")
	}
	return nil
}

// ValidatePrivateDirectory checks an existing private directory without
// creating it or repairing ownership, ACLs, or permissions. Callers use it
// before reading an operator-controlled directory tree.
func ValidatePrivateDirectory(path string) error {
	if !filepath.IsAbs(path) || !resolvedPathIsOriginal(path) || !privateDirectorySecure(path) {
		return fault.New(fault.Trust, "private directory permissions")
	}
	return nil
}

func privateDirectoryAbsent(path string) (bool, error) {
	_, err := os.Lstat(path)
	if err == nil {
		return false, nil
	}
	if os.IsNotExist(err) {
		return true, nil
	}
	return false, err
}

func protectNewDirectory(path string, created bool) error {
	if !created {
		return nil
	}
	if err := protectDirectory(path); err != nil {
		return fault.New(fault.Apply, "protect private directory")
	}
	return nil
}

func privateDirectorySecure(path string) bool {
	info, err := os.Lstat(path)
	return err == nil && info.IsDir() && secureMode(info.Mode(), true) && checkNativeSecurity(path, true)
}

// AppendProtected appends one bounded JSON receipt. A single configured writer
// and a local filesystem are required; external log transport is not provided.
func AppendProtected(path string, data []byte) error {
	data, err := normalizeReceipt(path, data)
	if err != nil {
		return err
	}
	appendMu.Lock()
	defer appendMu.Unlock()
	if err := EnsurePrivateDirectory(filepath.Dir(path)); err != nil {
		return err
	}
	f, err := openReceiptLog(path)
	if err != nil {
		return err
	}
	defer f.Close()
	if !openedRegularFileMatches(f, path) {
		return fault.New(fault.Trust, "receipt log changed")
	}
	if n, err := f.Write(data); err != nil || n != len(data) {
		return fault.New(fault.Apply, "write receipt")
	}
	if err = f.Sync(); err != nil {
		return fault.New(fault.Apply, "sync receipt")
	}
	return nil
}

func openedRegularFileMatches(file *os.File, path string) bool {
	opened, openedErr := file.Stat()
	named, namedErr := os.Lstat(path)
	return openedErr == nil && namedErr == nil && named.Mode().IsRegular() && os.SameFile(opened, named)
}

func normalizeReceipt(path string, data []byte) ([]byte, error) {
	// Normalize one object to one JSONL record so callers cannot accidentally
	// concatenate records or introduce an ambiguous multiline audit stream.
	if !filepath.IsAbs(path) || len(data) == 0 || len(data) > 65536 {
		return nil, fault.New(fault.Invalid, "bounded JSON receipt required")
	}
	var object map[string]json.RawMessage
	if json.Unmarshal(data, &object) != nil || object == nil {
		return nil, fault.New(fault.Invalid, "JSON receipt object required")
	}
	var compact bytes.Buffer
	if json.Compact(&compact, data) != nil {
		return nil, fault.New(fault.Invalid, "JSON receipt required")
	}
	return append(compact.Bytes(), '\n'), nil
}

func openReceiptLog(path string) (*os.File, error) {
	f, err := openInParent(path, os.O_WRONLY|os.O_APPEND|os.O_CREATE|os.O_EXCL, 0o600)
	if err == nil {
		if err := protectFile(path); err != nil {
			_ = f.Close()
			return nil, fault.New(fault.Apply, "protect receipt log")
		}
		return f, nil
	}
	if !os.IsExist(err) {
		return nil, fault.New(fault.Apply, "open receipt log")
	}
	if err := ValidateProtectedPath(path); err != nil {
		return nil, err
	}
	f, err = openInParent(path, os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return nil, fault.New(fault.Apply, "open receipt log")
	}
	return f, nil
}

// ValidateProtectedPath checks native protection without reading file contents.
func ValidateProtectedPath(path string) error {
	if !filepath.IsAbs(path) {
		return fault.New(fault.Trust, "absolute protected path required")
	}
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() || !secureMode(info.Mode(), true) || !checkNativeSecurity(path, true) {
		return fault.New(fault.Trust, "protected path permissions")
	}
	if !resolvedPathIsOriginal(path) {
		return fault.New(fault.Trust, "symbolic protected path")
	}
	return nil
}

// OpenProtected opens a trusted regular file without following its final
// symbolic link. The returned descriptor refers to the same protected file
// that was validated; callers must close it. Secret files must deny group and
// other access on Unix platforms.
func OpenProtected(path string, secret bool) (*os.File, error) {
	before, err := protectedFileInfo(path, secret)
	if err != nil {
		return nil, err
	}
	f, err := openInParent(path, os.O_RDONLY, 0)
	if err != nil {
		return nil, fault.New(fault.Denied, "open protected file")
	}
	after, err := f.Stat()
	if err != nil || !os.SameFile(before, after) {
		_ = f.Close()
		return nil, fault.New(fault.Trust, "protected file changed")
	}
	return f, nil
}

// ReadProtected reads a bounded protected file using OpenProtected.
func ReadProtected(path string, secret bool) ([]byte, error) {
	f, err := OpenProtected(path, secret)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, MaxDocumentSize+1))
	if err != nil || len(data) > MaxDocumentSize {
		return nil, fault.New(fault.Invalid, "protected file size")
	}
	return data, nil
}

// ValidateFileMetadata verifies a protected file path without reading contents.
func ValidateFileMetadata(path string, secret bool) error {
	_, err := protectedFileInfo(path, secret)
	return err
}

func protectedFileInfo(path string, secret bool) (os.FileInfo, error) {
	if !filepath.IsAbs(path) {
		return nil, fault.New(fault.Trust, "absolute file path required")
	}
	info, err := os.Lstat(path)
	if err != nil {
		return nil, fault.New(fault.Missing, "read protected file")
	}
	if !info.Mode().IsRegular() || !secureMode(info.Mode(), secret) || !checkNativeSecurity(path, secret) {
		return nil, fault.New(fault.Trust, "protected file permissions")
	}
	if !resolvedPathIsOriginal(path) {
		return nil, fault.New(fault.Trust, "symbolic file path")
	}
	return info, nil
}

// WriteProtected replaces one authorized file atomically with owner-only access.
func WriteProtected(path string, data []byte) error {
	if !filepath.IsAbs(path) || len(data) > MaxDocumentSize {
		return fault.New(fault.Invalid, "protected destination")
	}
	dir := filepath.Dir(path)
	if err := protectedDestination(dir, path); err != nil {
		return err
	}
	name, err := stageProtectedFile(dir, data)
	if err != nil {
		return err
	}
	defer os.Remove(name)
	return activateStagedFile(name, path, dir)
}

func protectedDestination(dir, path string) error {
	if err := EnsurePrivateDirectory(dir); err != nil {
		return err
	}
	if !resolvedPathIsOriginal(dir) {
		return fault.New(fault.Trust, "symbolic destination directory")
	}
	info, err := os.Lstat(path)
	if err == nil && !info.Mode().IsRegular() {
		return fault.New(fault.Trust, "nonregular destination")
	}
	if err != nil && !os.IsNotExist(err) {
		return fault.New(fault.Apply, "inspect destination")
	}
	return nil
}

func stageProtectedFile(dir string, data []byte) (string, error) {
	return stageProtectedFileWithWriter(dir, data, writeAndSync)
}

func stageProtectedFileWithWriter(dir string, data []byte, write func(*os.File, []byte) error) (string, error) {
	file, err := os.CreateTemp(dir, ".cq-stage-*")
	if err != nil {
		return "", fault.New(fault.Apply, "stage protected file")
	}
	name := file.Name()
	removeStage := true
	defer func() {
		_ = file.Close()
		if removeStage {
			_ = os.Remove(name)
		}
	}()
	if err := protectFile(file.Name()); err != nil {
		return "", fault.New(fault.Apply, "protect staging ACL")
	}
	if err := write(file, data); err != nil {
		return "", err
	}
	removeStage = false
	return name, nil
}

func activateStagedFile(name, path, dir string) error {
	if err := replaceFile(name, path); err != nil {
		return fault.New(fault.Apply, "activate protected file")
	}
	if err := syncDirectory(dir); err != nil {
		return fault.New(fault.Apply, "sync destination directory")
	}
	return nil
}

func writeAndSync(f *os.File, data []byte) error {
	if err := f.Chmod(0o600); err != nil {
		_ = f.Close()
		return fault.New(fault.Apply, "protect staged file")
	}
	if _, err := f.Write(data); err != nil {
		_ = f.Close()
		return fault.New(fault.Apply, "write staged file")
	}
	if err := f.Sync(); err != nil {
		_ = f.Close()
		return fault.New(fault.Apply, "sync staged file")
	}
	if err := f.Close(); err != nil {
		return fault.New(fault.Apply, "close staged file")
	}
	return nil
}

// EncodeValues emits a JSON map, not a shell program or environment-file syntax.
func EncodeValues(gen model.Generation) ([]byte, error) {
	values := make(map[string]string, len(gen.Values))
	for key, value := range gen.Values {
		if key == "" || strings.ContainsRune(key, '\x00') || strings.ContainsRune(value.Data, '\x00') {
			return nil, fault.New(fault.Invalid, "invalid delivery value")
		}
		values[key] = value.Data
	}
	data, err := json.Marshal(values)
	if err != nil || len(data) > MaxDocumentSize {
		return nil, fault.New(fault.Invalid, "delivery document")
	}
	return data, nil
}

// OpenLockFile opens an owner-controlled regular file for a native process lock.
// The file must remain at its stable pathname; replacing a locked inode would
// allow two reconcilers to hold different locks for the same workload.
func OpenLockFile(path string) (*os.File, error) {
	if err := EnsurePrivateDirectory(filepath.Dir(path)); err != nil {
		return nil, err
	}
	f, created, err := openOrCreateLock(path)
	if err != nil {
		return nil, err
	}
	if created {
		if err = protectFile(path); err != nil {
			_ = f.Close()
			return nil, fault.New(fault.Apply, "protect lock file")
		}
	}
	if !openedRegularFileMatches(f, path) || ValidateProtectedPath(path) != nil {
		_ = f.Close()
		return nil, fault.New(fault.Trust, "lock file changed")
	}
	return f, nil
}

func openOrCreateLock(path string) (*os.File, bool, error) {
	file, err := openInParent(path, os.O_CREATE|os.O_EXCL|os.O_RDWR, 0600)
	if err == nil {
		return file, true, nil
	}
	if !os.IsExist(err) {
		return nil, false, fault.New(fault.Apply, "open lock file")
	}
	if err := ValidateProtectedPath(path); err != nil {
		return nil, false, err
	}
	file, err = openInParent(path, os.O_RDWR, 0600)
	if err != nil {
		return nil, false, fault.New(fault.Apply, "open lock file")
	}
	return file, false, nil
}

// openInParent anchors a one-name operation to the opened parent directory.
// Callers validate the absolute path and its access policy first; the Root then
// prevents a concurrent replacement from escaping that validated directory.
func openInParent(path string, flags int, perm os.FileMode) (*os.File, error) {
	directory := filepath.Dir(path)
	root, err := os.OpenRoot(directory)
	if err != nil {
		return nil, err
	}
	defer root.Close()
	return root.OpenFile(filepath.Base(path), flags, perm)
}
