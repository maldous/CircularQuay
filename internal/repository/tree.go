package repository

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/maldous/CircularQuay/internal/fault"
	"github.com/maldous/CircularQuay/internal/platform"
	"github.com/maldous/CircularQuay/internal/policy"
)

func (s *store) readTree(ctx context.Context, commit string) (map[string]string, error) {
	data, err := s.command(ctx, maximumReleaseBytes, "ls-tree", "-r", "-z", "--full-tree", commit)
	if err != nil {
		return nil, failure(fault.Trust, "tree")
	}
	tree := make(map[string]string)
	paths := make(map[string]treePath)
	for _, entry := range bytes.Split(data, []byte{0}) {
		if len(entry) == 0 {
			continue
		}
		metadata, name, found := bytes.Cut(entry, []byte{'\t'})
		fields := strings.Fields(string(metadata))
		path := string(name)
		if !found || len(fields) != 3 || fields[0] != "100644" || fields[1] != "blob" || !validCommit(fields[2]) || !policy.ValidRepositoryPath(path) {
			return nil, failure(fault.Trust, "tree_entry")
		}
		if err := registerTreePath(paths, path); err != nil {
			return nil, err
		}
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
	if err = platform.EnsurePrivateDirectory(releases); err != nil {
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
		if err = platform.EnsurePrivateDirectory(filepath.Dir(file)); err != nil {
			return "", failure(fault.Trust, "release_parent")
		}
		if err = platform.WriteProtected(file, data); err != nil {
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

// A case-folded directory has exactly one portable spelling and cannot also be
// a file. Register every ancestor so acceptance is independent of listing order.
type treePath struct {
	spelling string
	file     bool
}

func registerTreePath(paths map[string]treePath, name string) error {
	segments := strings.Split(name, "/")
	for i := range segments {
		spelling := strings.Join(segments[:i+1], "/")
		key := strings.ToLower(spelling)
		file := i == len(segments)-1
		if existing, ok := paths[key]; ok {
			if existing.spelling != spelling || existing.file || file {
				return failure(fault.Trust, "tree_collision")
			}
		} else {
			paths[key] = treePath{spelling: spelling, file: file}
		}
	}
	return nil
}
