package configuration

import (
	"context"
	"path/filepath"
	"strings"

	"github.com/maldous/CircularQuay/internal/fault"
	"github.com/maldous/CircularQuay/internal/model"
	"github.com/maldous/CircularQuay/internal/policy"
)

func (r resolver) readInclude(ctx context.Context, include model.Include) ([]byte, string, []string, error) {
	if include.Repository == nil {
		return r.readLocal(include)
	}
	return r.readRepository(ctx, include)
}

func (r resolver) readLocal(include model.Include) ([]byte, string, []string, error) {
	if !within(include.BaseDirectory, include.File) {
		return nil, "", nil, failure(fault.Trust, "include_location")
	}
	data, err := r.readFile(include.File, include.Visibility == "local")
	if err != nil || len(data) > maximumFragmentBytes || !matchesDigest(data, include.SHA256) {
		return nil, "", nil, failure(fault.Trust, "include_file")
	}
	return data, include.File, nil, nil
}

func (r resolver) readRepository(ctx context.Context, include model.Include) ([]byte, string, []string, error) {
	if include.Refresh == "offline" && !r.now().Before(*include.CacheExpiresAt) {
		return nil, "", nil, failure(fault.Expired, "include_cache")
	}
	var (
		release string
		err     error
	)
	if include.Refresh == "online" {
		release, err = r.fetch(ctx, *include.Repository, include.Commit)
	} else {
		release, err = r.current(ctx, *include.Repository)
	}
	if err != nil || !isAbsoluteClean(release) || filepath.Base(release) != include.Commit {
		return nil, "", nil, failure(fault.Trust, "include_release")
	}
	file := filepath.Join(release, filepath.FromSlash(include.Path))
	if !within(release, file) {
		return nil, "", nil, failure(fault.Trust, "include_path")
	}
	data, err := r.readFile(file, false)
	if err != nil || len(data) > maximumFragmentBytes || !matchesDigest(data, include.SHA256) {
		return nil, "", nil, failure(fault.Trust, "include_content")
	}
	roots := []string{include.Repository.Directory}
	return data, file, roots, nil
}

func decodeFragment(data []byte, include model.Include) (model.Policy, error) {
	if isYAML(data) {
		return model.Policy{}, failure(fault.Unsupported, "include_yaml")
	}
	child, err := policyDecode(data, filepath.Join(include.BaseDirectory, "fragment.json"))
	if err != nil {
		return model.Policy{}, failure(fault.Invalid, "include_document")
	}
	if !validFragmentDocument(data, child) || !fragmentPathsWithin(child, include.BaseDirectory) {
		return model.Policy{}, failure(fault.Invalid, "include_fragment")
	}
	return child, nil
}

func isYAML(data []byte) bool {
	return !strings.HasPrefix(strings.TrimSpace(string(data)), "{")
}

var policyDecode = func(data []byte, path string) (model.Policy, error) {
	return policy.DecodeDocument(data, path)
}

func fragmentPathsWithin(child model.Policy, base string) bool {
	for _, source := range child.Sources {
		if !sourcePathsWithin(source, base) {
			return false
		}
	}
	return true
}

func sourcePathsWithin(source model.Source, base string) bool {
	for _, reference := range []string{source.TokenFile, source.AgeIdentityFile} {
		if reference != "" && !within(base, reference) {
			return false
		}
	}
	if source.Type == "json" || source.Type == "sops" {
		return source.Path != "" && within(base, source.Path)
	}
	return source.Type != "env" || source.Path == "" || within(base, source.Path)
}
