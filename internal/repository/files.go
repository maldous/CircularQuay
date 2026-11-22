package repository

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"time"

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

func verifyRelease(directory string, tree map[string]string) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	return (&store{}).verifyRelease(ctx, directory, tree)
}

func (s *store) verifyRelease(ctx context.Context, directory string, tree map[string]string) error {
	if err := protectedDirectory(directory); err != nil {
		return err
	}
	verifier := releaseVerifier{store: s, ctx: ctx, directory: directory, tree: tree}
	err := filepath.WalkDir(directory, verifier.visit)
	if err != nil {
		return err
	}
	if verifier.count != len(tree) {
		return failure(fault.Trust, "release_missing")
	}
	return nil
}

// releaseVerifier checks that materialized data remains the exact signed tree.
// It keeps the file and aggregate byte limits separate from Git transport limits.
type releaseVerifier struct {
	store     *store
	ctx       context.Context
	directory string
	tree      map[string]string
	count     int
	total     int64
}

func (v *releaseVerifier) visit(path string, entry os.DirEntry, walkError error) error {
	if walkError != nil || entry == nil {
		return failure(fault.Trust, "release_walk")
	}
	if entry.IsDir() {
		return verifyReleaseDirectory(path)
	}
	return v.verifyFile(path, entry)
}

func verifyReleaseDirectory(path string) error {
	if err := platform.EnsurePrivateDirectory(path); err != nil {
		return failure(fault.Trust, "release_directory")
	}
	return nil
}

func (v *releaseVerifier) verifyFile(path string, entry os.DirEntry) error {
	expected, err := v.expectedBlob(path)
	if err != nil {
		return err
	}
	data, err := readProtected(path, true, maximumFileBytes)
	if err != nil {
		return err
	}
	if err = verifyReleaseFile(entry, data); err != nil {
		return err
	}
	if err = v.store.verifyMaterializedBlob(v.ctx, path, expected); err != nil {
		return err
	}
	v.count++
	v.total += int64(len(data))
	if v.total > maximumReleaseBytes {
		return failure(fault.Trust, "release_size")
	}
	return nil
}

func (v releaseVerifier) expectedBlob(path string) (string, error) {
	relative, err := filepath.Rel(v.directory, path)
	if err != nil {
		return "", failure(fault.Trust, "release_path")
	}
	expected, exists := v.tree[filepath.ToSlash(relative)]
	if !exists {
		return "", failure(fault.Trust, "release_extra")
	}
	return expected, nil
}

func verifyReleaseFile(entry os.DirEntry, data []byte) error {
	info, err := entry.Info()
	if err != nil || info.Mode().Perm()&0o111 != 0 {
		return failure(fault.Trust, "release_content")
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
