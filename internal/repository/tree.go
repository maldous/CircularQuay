package repository

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/maldous/CircularQuay/internal/fault"
	"github.com/maldous/CircularQuay/internal/policy"
)

func (s *store) readTree(ctx context.Context, commit string) (map[string]string, error) {
	data, err := s.command(ctx, maximumReleaseBytes, "ls-tree", "-r", "-z", "--full-tree", commit)
	if err != nil {
		return nil, failure(fault.Trust, "tree")
	}
	tree := make(map[string]string)
	casefold := make(map[string]bool)
	for _, entry := range bytes.Split(data, []byte{0}) {
		if len(entry) == 0 {
			continue
		}
		metadata, name, found := bytes.Cut(entry, []byte{'\t'})
		fields := strings.Fields(string(metadata))
		path := string(name)
		if !found || len(fields) != 3 || fields[0] != "100644" || fields[1] != "blob" || !validCommit(fields[2]) || !policy.ValidRepositoryPath(path) || casefold[strings.ToLower(path)] {
			return nil, failure(fault.Trust, "tree_entry")
		}
		// A file may not be the parent of another file on any platform.
		for parent := filepath.ToSlash(filepath.Dir(path)); parent != "."; parent = filepath.ToSlash(filepath.Dir(parent)) {
			if _, exists := tree[parent]; exists {
				return nil, failure(fault.Trust, "tree_collision")
			}
		}
		casefold[strings.ToLower(path)] = true
		tree[path] = fields[2]
		if len(tree) > maximumFiles {
			return nil, failure(fault.Trust, "tree_size")
		}
	}
	if len(tree) == 0 {
		return nil, failure(fault.Trust, "empty_tree")
	}
	return tree, nil
}

func (s *store) materialize(ctx context.Context, commit string) (string, error) {
	tree, err := s.readTree(ctx, commit)
	if err != nil {
		return "", err
	}
	releases := filepath.Join(s.repo.Directory, "releases")
	if err = os.MkdirAll(releases, 0o700); err != nil {
		return "", failure(fault.Trust, "releases_directory")
	}
	if err = protectedDirectory(releases); err != nil {
		return "", err
	}
	destination := filepath.Join(releases, commit)
	if _, err = os.Lstat(destination); err == nil {
		if err = verifyRelease(destination, tree); err != nil {
			return "", err
		}
		return destination, nil
	} else if !os.IsNotExist(err) {
		return "", failure(fault.Trust, "release_directory")
	}
	staging, err := os.MkdirTemp(releases, ".release-*")
	if err != nil {
		return "", failure(fault.Trust, "release_staging")
	}
	defer os.RemoveAll(staging)
	var total int64
	for path, object := range tree {
		// cat-file reads exact signed blob IDs. It does not apply .gitattributes,
		// checkout filters, archive attributes or executable mode from a worktree.
		data, err := s.command(ctx, maximumFileBytes, "cat-file", "blob", object)
		if err != nil || blobID(data, len(object)) != object {
			return "", failure(fault.Trust, "blob")
		}
		total += int64(len(data))
		if total > maximumReleaseBytes {
			return "", failure(fault.Trust, "release_size")
		}
		file := filepath.Join(staging, filepath.FromSlash(path))
		if err = os.MkdirAll(filepath.Dir(file), 0o700); err != nil {
			return "", failure(fault.Trust, "release_parent")
		}
		if err = os.WriteFile(file, data, 0o400); err != nil {
			return "", failure(fault.Trust, "release_write")
		}
	}
	if err = verifyRelease(staging, tree); err != nil {
		return "", err
	}
	if err = os.Rename(staging, destination); err != nil {
		return "", failure(fault.Trust, "release_accept")
	}
	return destination, nil
}
