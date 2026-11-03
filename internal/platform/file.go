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
	_, before := os.Lstat(path)
	created := os.IsNotExist(before)
	if before != nil && !created {
		return fault.New(fault.Trust, "inspect private directory")
	}
	if err := os.MkdirAll(path, 0o700); err != nil {
		return fault.New(fault.Apply, "create private directory")
	}
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil || filepath.Clean(resolved) != filepath.Clean(path) {
		return fault.New(fault.Trust, "symbolic private directory")
	}
	if created {
		if err := protectDirectory(path); err != nil {
			return fault.New(fault.Apply, "protect private directory")
		}
	}
	info, err := os.Lstat(path)
	if err != nil || !info.IsDir() || !secureMode(info.Mode(), true) || !checkNativeSecurity(path, true) {
		return fault.New(fault.Trust, "private directory permissions")
	}
	return nil
}

// AppendProtected appends one bounded JSON receipt. A single configured writer
// and a local filesystem are required; external log transport is not provided.
func AppendProtected(path string, data []byte) error {
	// Normalize one object to one JSONL record so callers cannot accidentally
	// concatenate records or introduce an ambiguous multiline audit stream.
	if !filepath.IsAbs(path) || len(data) == 0 || len(data) > 65536 {
		return fault.New(fault.Invalid, "bounded JSON receipt required")
	}
	var object map[string]json.RawMessage
	if json.Unmarshal(data, &object) != nil || object == nil {
		return fault.New(fault.Invalid, "JSON receipt object required")
	}
	var compact bytes.Buffer
	if json.Compact(&compact, data) != nil {
		return fault.New(fault.Invalid, "JSON receipt required")
	}
	data = append(compact.Bytes(), '\n')
	appendMu.Lock()
	defer appendMu.Unlock()
	if err := EnsurePrivateDirectory(filepath.Dir(path)); err != nil {
		return err
	}
	created := false
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_APPEND|os.O_CREATE|os.O_EXCL, 0o600)
	if os.IsExist(err) {
		if err = ValidateProtectedPath(path); err != nil {
			return err
		}
		f, err = os.OpenFile(path, os.O_WRONLY|os.O_APPEND, 0o600)
	} else if err == nil {
		created = true
	}
	if err != nil {
		return fault.New(fault.Apply, "open receipt log")
	}
	defer f.Close()
	if created {
		if err = protectFile(path); err != nil {
			return fault.New(fault.Apply, "protect receipt log")
		}
	}
	current, err := f.Stat()
	named, namedErr := os.Lstat(path)
	if err != nil || namedErr != nil || !os.SameFile(current, named) || !named.Mode().IsRegular() {
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

// ValidateProtectedPath checks native protection without reading file contents.
func ValidateProtectedPath(path string) error {
	if !filepath.IsAbs(path) {
		return fault.New(fault.Trust, "absolute protected path required")
	}
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() || !secureMode(info.Mode(), true) || !checkNativeSecurity(path, true) {
		return fault.New(fault.Trust, "protected path permissions")
	}
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil || filepath.Clean(resolved) != filepath.Clean(path) {
		return fault.New(fault.Trust, "symbolic protected path")
	}
	return nil
}

// ReadProtected reads a regular file without following a final symbolic link.
// Secret files must deny group and other access on Unix platforms.
func ReadProtected(path string, secret bool) ([]byte, error) {
	if !filepath.IsAbs(path) {
		return nil, fault.New(fault.Trust, "absolute file path required")
	}
	before, err := os.Lstat(path)
	if err != nil {
		return nil, fault.New(fault.Missing, "read protected file")
	}
	if !before.Mode().IsRegular() || !secureMode(before.Mode(), secret) || !checkNativeSecurity(path, secret) {
		return nil, fault.New(fault.Trust, "protected file permissions")
	}
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil || filepath.Clean(resolved) != filepath.Clean(path) {
		return nil, fault.New(fault.Trust, "symbolic file path")
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, fault.New(fault.Denied, "open protected file")
	}
	defer f.Close()
	after, err := f.Stat()
	if err != nil || !os.SameFile(before, after) {
		return nil, fault.New(fault.Trust, "protected file changed")
	}
	data, err := io.ReadAll(io.LimitReader(f, MaxDocumentSize+1))
	if err != nil || len(data) > MaxDocumentSize {
		return nil, fault.New(fault.Invalid, "protected file size")
	}
	return data, nil
}

// WriteProtected replaces one authorized file atomically with owner-only access.
func WriteProtected(path string, data []byte) error {
	if !filepath.IsAbs(path) || len(data) > MaxDocumentSize {
		return fault.New(fault.Invalid, "protected destination")
	}
	dir := filepath.Dir(path)
	if err := EnsurePrivateDirectory(dir); err != nil {
		return err
	}
	resolved, err := filepath.EvalSymlinks(dir)
	if err != nil || filepath.Clean(resolved) != filepath.Clean(dir) {
		return fault.New(fault.Trust, "symbolic destination directory")
	}
	if info, err := os.Lstat(path); err == nil && !info.Mode().IsRegular() {
		return fault.New(fault.Trust, "nonregular destination")
	} else if err != nil && !os.IsNotExist(err) {
		return fault.New(fault.Apply, "inspect destination")
	}
	f, err := os.CreateTemp(dir, ".cquay-stage-*")
	if err != nil {
		return fault.New(fault.Apply, "stage protected file")
	}
	name := f.Name()
	defer os.Remove(name)
	if err = protectFile(name); err != nil {
		_ = f.Close()
		return fault.New(fault.Apply, "protect staging ACL")
	}
	if err = writeAndSync(f, data); err != nil {
		return err
	}
	if err = replaceFile(name, path); err != nil {
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
	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_RDWR, 0600)
	created := err == nil
	if os.IsExist(err) {
		if err = ValidateProtectedPath(path); err != nil {
			return nil, err
		}
		f, err = os.OpenFile(path, os.O_RDWR, 0600)
	}
	if err != nil {
		return nil, fault.New(fault.Apply, "open lock file")
	}
	if created {
		if err = protectFile(path); err != nil {
			_ = f.Close()
			return nil, fault.New(fault.Apply, "protect lock file")
		}
	}
	opened, statErr := f.Stat()
	named, namedErr := os.Lstat(path)
	if statErr != nil || namedErr != nil || !os.SameFile(opened, named) || ValidateProtectedPath(path) != nil {
		_ = f.Close()
		return nil, fault.New(fault.Trust, "lock file changed")
	}
	return f, nil
}
