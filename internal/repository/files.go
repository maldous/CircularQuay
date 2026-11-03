package repository

import (
	"bytes"
	"crypto/sha1" // Git's SHA-1 object format; commit authenticity uses SSH signatures.
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/maldous/CircularQuay/internal/fault"
)

func protectedDirectory(path string) error {
	info, err := os.Lstat(path)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 || info.Mode().Perm()&0o077 != 0 {
		return failure(fault.Trust, "directory")
	}
	// Every existing ancestor must resolve without symlink redirection. Ancestors
	// may be public (for example /tmp), but the cache itself is private.
	for current := filepath.Dir(path); ; current = filepath.Dir(current) {
		info, err = os.Lstat(current)
		if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return failure(fault.Trust, "directory_ancestor")
		}
		if current == filepath.Dir(current) {
			break
		}
	}
	return nil
}

func readProtected(path string, private bool, limit int64) ([]byte, error) {
	before, err := os.Lstat(path)
	mask := os.FileMode(0o022)
	if private {
		mask = 0o077
	}
	if err != nil || !before.Mode().IsRegular() || before.Mode().Perm()&mask != 0 || before.Size() > limit {
		return nil, failure(fault.Trust, "file")
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, failure(fault.Trust, "file")
	}
	defer file.Close()
	after, err := file.Stat()
	if err != nil || !os.SameFile(before, after) {
		return nil, failure(fault.Trust, "file")
	}
	data, err := io.ReadAll(io.LimitReader(file, limit+1))
	if err != nil || int64(len(data)) > limit {
		return nil, failure(fault.Trust, "file_size")
	}
	return data, nil
}

func validCommit(commit string) bool {
	if len(commit) != 40 && len(commit) != 64 {
		return false
	}
	for _, character := range commit {
		if !(character >= '0' && character <= '9' || character >= 'a' && character <= 'f') {
			return false
		}
	}
	return true
}

func blobID(data []byte, length int) string {
	header := []byte(fmt.Sprintf("blob %d\x00", len(data)))
	if length == 40 {
		hash := sha1.New()
		hash.Write(header)
		hash.Write(data)
		return hex.EncodeToString(hash.Sum(nil))
	}
	hash := sha256.New()
	hash.Write(header)
	hash.Write(data)
	return hex.EncodeToString(hash.Sum(nil))
}

func verifyRelease(directory string, tree map[string]string) error {
	if err := protectedDirectory(directory); err != nil {
		return err
	}
	count := 0
	var total int64
	err := filepath.WalkDir(directory, func(path string, entry os.DirEntry, walkError error) error {
		if walkError != nil || entry == nil {
			return failure(fault.Trust, "release_walk")
		}
		if entry.IsDir() {
			info, err := entry.Info()
			if err != nil || info.Mode().Perm()&0o077 != 0 {
				return failure(fault.Trust, "release_directory")
			}
			return nil
		}
		relative, err := filepath.Rel(directory, path)
		if err != nil {
			return failure(fault.Trust, "release_path")
		}
		name := filepath.ToSlash(relative)
		expected, exists := tree[name]
		if !exists {
			return failure(fault.Trust, "release_extra")
		}
		data, err := readProtected(path, true, maximumFileBytes)
		if err != nil {
			return err
		}
		info, err := entry.Info()
		if err != nil || info.Mode().Perm()&0o111 != 0 || !bytes.Equal([]byte(blobID(data, len(expected))), []byte(expected)) {
			return failure(fault.Trust, "release_content")
		}
		count++
		total += int64(len(data))
		if total > maximumReleaseBytes {
			return failure(fault.Trust, "release_size")
		}
		return nil
	})
	if err != nil {
		return err
	}
	if count != len(tree) {
		return failure(fault.Trust, "release_missing")
	}
	return nil
}

func safeToken(data []byte) (string, error) {
	token := strings.TrimSpace(string(data))
	if token == "" || strings.ContainsAny(token, "\x00\r\n") {
		return "", failure(fault.Trust, "token")
	}
	return token, nil
}
