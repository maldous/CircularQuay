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
	return parseTree(data)
}

func parseTree(data []byte) (map[string]string, error) {
	tree := make(map[string]string)
	paths := make(map[string]treePath)
	for _, entry := range bytes.Split(data, []byte{0}) {
		if err := addTreeEntry(tree, paths, entry); err != nil {
			return nil, err
		}
	}
	if len(tree) == 0 {
		return nil, failure(fault.Trust, "empty_tree")
	}
	return tree, nil
}

func addTreeEntry(tree map[string]string, paths map[string]treePath, entry []byte) error {
	if len(entry) == 0 {
		return nil
	}
	path, object, err := parseTreeEntry(entry)
	if err != nil {
		return err
	}
	if err := registerTreePath(paths, path); err != nil {
		return err
	}
	tree[path] = object
	if len(tree) > maximumFiles {
		return failure(fault.Trust, "tree_size")
	}
	return nil
}

func parseTreeEntry(entry []byte) (string, string, error) {
	metadata, name, found := bytes.Cut(entry, []byte{'\t'})
	fields := strings.Fields(string(metadata))
	path := string(name)
	if !found || len(fields) != 3 || fields[0] != "100644" || fields[1] != "blob" || !validCommit(fields[2]) || !policy.ValidRepositoryPath(path) {
		return "", "", failure(fault.Trust, "tree_entry")
	}
	return path, fields[2], nil
}

func (s *store) materialize(ctx context.Context, commit string) (string, error) {
	tree, err := s.readTree(ctx, commit)
	if err != nil {
		return "", err
	}
	releases, destination, accepted, err := s.releaseDestination(ctx, commit, tree)
	if err != nil || accepted {
		return destination, err
	}
	staging, err := os.MkdirTemp(releases, ".release-*")
	if err != nil {
		return "", failure(fault.Trust, "release_staging")
	}
	defer os.RemoveAll(staging)
	if err = s.writeTree(ctx, staging, tree); err != nil {
		return "", err
	}
	if err = s.verifyRelease(ctx, staging, tree); err != nil {
		return "", err
	}
	if err = os.Rename(staging, destination); err != nil {
		return "", failure(fault.Trust, "release_accept")
	}
	return destination, nil
}

func (s *store) releaseDestination(ctx context.Context, commit string, tree map[string]string) (string, string, bool, error) {
	releases := filepath.Join(s.repo.Directory, "releases")
	if err := platform.EnsurePrivateDirectory(releases); err != nil {
		return "", "", false, failure(fault.Trust, "releases_directory")
	}
	if err := protectedDirectory(releases); err != nil {
		return "", "", false, err
	}
	destination := filepath.Join(releases, commit)
	if _, err := os.Lstat(destination); err == nil {
		if err = s.verifyRelease(ctx, destination, tree); err != nil {
			return "", "", false, err
		}
		return releases, destination, true, nil
	} else if !os.IsNotExist(err) {
		return "", "", false, failure(fault.Trust, "release_directory")
	}
	return releases, destination, false, nil
}

func (s *store) writeTree(ctx context.Context, staging string, tree map[string]string) error {
	var total int64
	for path, object := range tree {
		data, err := s.readBlob(ctx, object)
		if err != nil {
			return err
		}
		total += int64(len(data))
		if total > maximumReleaseBytes {
			return failure(fault.Trust, "release_size")
		}
		if err = writeMaterializedFile(staging, path, data); err != nil {
			return err
		}
	}
	return nil
}

func (s *store) readBlob(ctx context.Context, object string) ([]byte, error) {
	// cat-file reads exact signed blob IDs. It does not apply .gitattributes,
	// checkout filters, archive attributes or executable mode from a worktree.
	data, err := s.command(ctx, maximumFileBytes, "cat-file", "blob", object)
	if err != nil {
		return nil, failure(fault.Trust, "blob")
	}
	return data, nil
}

func writeMaterializedFile(staging, path string, data []byte) error {
	file := filepath.Join(staging, filepath.FromSlash(path))
	if err := platform.EnsurePrivateDirectory(filepath.Dir(file)); err != nil {
		return failure(fault.Trust, "release_parent")
	}
	if err := platform.WriteProtected(file, data); err != nil {
		return failure(fault.Trust, "release_write")
	}
	return nil
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
