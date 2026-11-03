package repository

import (
	"bytes"
	"crypto/sha1" // Git's SHA-1 object format; commit authenticity uses SSH signatures.
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/maldous/CircularQuay/internal/fault"
	"github.com/maldous/CircularQuay/internal/platform"
)

func protectedDirectory(path string) error {
	return platform.EnsurePrivateDirectory(path)
}

func readProtected(path string, private bool, limit int64) ([]byte, error) {
	data, err := platform.ReadProtected(path, private)
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > limit {
		return nil, failure(fault.Trust, "file_size")
	}
	return data, nil
}

func validCommit(commit string) bool {
	if len(commit) != 40 {
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
			if err := platform.EnsurePrivateDirectory(path); err != nil {
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
