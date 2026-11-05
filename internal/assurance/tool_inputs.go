package assurance

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"hash"
	"io"
	"os"
	"path/filepath"
	"sort"

	"github.com/maldous/CircularQuay/internal/platform"
)

const (
	maxToolInputBytes   = 4 << 20
	maxToolInputFiles   = 4096
	maxToolInputEntries = 8192
	maxToolInputTotal   = 64 << 20
)

func bindToolInputs(tools []ToolEntry, source, output string) error {
	for _, tool := range tools {
		if tool.Rules == nil || tool.Rules.Input == nil {
			continue
		}
		actual, err := bindToolInput(tool.Rules.Kind, tool.Rules.Input.Path, source, output)
		if err != nil || actual != tool.Rules.Input.SHA256 {
			return errors.New("tool rule input mismatch")
		}
	}
	return nil
}

// toolInputMatches re-reads a manifest-bound input immediately around a tool
// invocation. It detects ordinary drift, but is not an atomic-use or sandbox
// guarantee against an actor able to replace and restore files.
func toolInputMatches(rules *ToolRules) bool {
	if rules == nil || rules.Input == nil {
		return true
	}
	actual, err := digestToolInput(rules.Kind, rules.Input.Path)
	return err == nil && actual == rules.Input.SHA256
}

func digestToolInput(kind, path string) (string, error) {
	if kind == "file" {
		return bindToolInputFile(path, "", "")
	}
	return bindToolInputTree(path, "", "")
}

func bindToolInput(kind, path, source, output string) (string, error) {
	if !localAbsoluteToolPath(path) {
		return "", errors.New("invalid tool rule input")
	}
	if kind == "file" {
		return bindToolInputFile(path, source, output)
	}
	return bindToolInputTree(path, source, output)
}

func bindToolInputFile(path, source, output string) (string, error) {
	file, err := platform.OpenProtected(path, true)
	if err != nil {
		return "", err
	}
	defer file.Close()
	canonical, err := canonicalRegularPath(path)
	if err != nil || (source != "" && pathsOverlap(canonical, source)) || (output != "" && pathsOverlap(canonical, output)) {
		return "", errors.New("invalid tool rule input")
	}
	return boundedInputDigest(file, canonical)
}

func bindToolInputTree(path, source, output string) (string, error) {
	if err := platform.ValidatePrivateDirectory(path); err != nil {
		return "", err
	}
	canonical, err := canonicalDirectory(path)
	if err != nil || (source != "" && pathsOverlap(canonical, source)) || (output != "" && pathsOverlap(canonical, output)) {
		return "", errors.New("invalid tool rule input")
	}
	files, err := protectedTreeFiles(canonical)
	if err != nil || len(files) == 0 {
		return "", errors.New("invalid tool rule snapshot")
	}
	return digestInputFiles(canonical, files)
}

func protectedTreeFiles(root string) ([]string, error) {
	files := []string{}
	entries := 0
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		entries++
		if walkErr != nil || entries > maxToolInputEntries || entry.Type()&os.ModeSymlink != 0 {
			return errors.New("invalid tool rule tree")
		}
		if entry.IsDir() {
			return platform.ValidatePrivateDirectory(path)
		}
		if !entry.Type().IsRegular() {
			return errors.New("invalid tool rule tree")
		}
		if len(files) >= maxToolInputFiles {
			return errors.New("tool rule file limit")
		}
		files = append(files, path)
		return nil
	})
	return files, err
}

func digestInputFiles(root string, files []string) (string, error) {
	sort.Strings(files)
	hash := sha256.New()
	var total int64
	for _, path := range files {
		relative, err := filepath.Rel(root, path)
		if err != nil || relative == "." || filepath.IsAbs(relative) {
			return "", errors.New("invalid tool rule tree")
		}
		file, err := platform.OpenProtected(path, true)
		if err != nil {
			return "", err
		}
		digest, size, err := boundedInputFileDigest(file, path)
		closeErr := file.Close()
		if err != nil || closeErr != nil || total+size > maxToolInputTotal {
			return "", errors.New("invalid tool rule tree")
		}
		total += size
		_, _ = io.WriteString(hash, filepath.ToSlash(relative)+"\x00"+digest+"\x00")
	}
	return digestHash(hash), nil
}

func boundedInputDigest(file *os.File, path string) (string, error) {
	digest, _, err := boundedInputFileDigest(file, path)
	return digest, err
}

func boundedInputFileDigest(file *os.File, path string) (string, int64, error) {
	before, err := file.Stat()
	if err != nil || !before.Mode().IsRegular() || before.Size() > maxToolInputBytes {
		return "", 0, errors.New("invalid tool rule input")
	}
	hash := sha256.New()
	read, err := io.Copy(hash, io.LimitReader(file, maxToolInputBytes+1))
	if err != nil || read > maxToolInputBytes || !inputFileUnchanged(file, path, before, read) {
		return "", 0, errors.New("tool rule input changed")
	}
	return digestHash(hash), read, nil
}

func inputFileUnchanged(file *os.File, path string, before os.FileInfo, read int64) bool {
	after, statErr := file.Stat()
	named, nameErr := os.Lstat(path)
	return statErr == nil && nameErr == nil && read == before.Size() && os.SameFile(before, named) && after.Size() == before.Size() && after.Mode() == before.Mode()
}

func digestHash(value hash.Hash) string {
	return hex.EncodeToString(value.Sum(nil))
}

func pathsOverlap(first, second string) bool {
	return containedBy(first, second) || containedBy(second, first)
}
