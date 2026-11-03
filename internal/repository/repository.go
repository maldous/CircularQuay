// Package repository synchronizes signed, non-executable configuration data.
// Download authorization and commit-signature authorization are independent.
package repository

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/maldous/CircularQuay/internal/fault"
	"github.com/maldous/CircularQuay/internal/jsondocument"
	"github.com/maldous/CircularQuay/internal/model"
	"github.com/maldous/CircularQuay/internal/platform"
	"github.com/maldous/CircularQuay/internal/policy"
)

const maximumFileBytes = 1 << 20
const maximumReleaseBytes = 16 << 20
const maximumFiles = 1024

type accepted struct {
	URL    string `json:"url"`
	Ref    string `json:"ref"`
	Commit string `json:"commit"`
}

type store struct {
	repo    model.Repository
	git     string
	signers []byte
}

func failure(kind fault.Kind, operation string) error {
	return fault.New(kind, "repository_"+operation)
}

// Fetch accepts only a signed descendant of both the pinned first commit and the
// last accepted commit. It never executes a program obtained from the repository.
// Concurrent attempts fail safely; an interrupted lock requires operator review.
func Fetch(ctx context.Context, repo model.Repository) (string, error) {
	s, err := openStore(repo, true)
	if err != nil {
		return "", err
	}
	release, err := acquireFetchLock(repo.Directory)
	if err != nil {
		return "", err
	}
	defer release()
	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	previous, err := s.readAccepted()
	if err != nil {
		return "", err
	}
	commit, err := s.fetchCandidate(ctx)
	if err != nil {
		return "", err
	}
	return s.acceptCandidate(ctx, previous, commit)
}

func acquireFetchLock(directory string) (func(), error) {
	lock := filepath.Join(directory, "fetch.lock")
	if err := os.Mkdir(lock, 0o700); err != nil {
		return nil, failure(fault.Unavailable, "locked")
	}
	return func() { _ = os.Remove(lock) }, nil
}

func (s *store) fetchCandidate(ctx context.Context) (string, error) {
	if err := s.initialize(ctx); err != nil {
		return "", err
	}
	if _, err := s.run(ctx, true, 4096, "fetch", "--quiet", "--no-tags", "--no-recurse-submodules", "--no-write-fetch-head", "--force", "--", s.repo.URL, "+"+s.repo.Ref+":refs/cquay/candidate"); err != nil {
		return "", failure(fault.Unavailable, "fetch")
	}
	data, err := s.command(ctx, 256, "rev-parse", "--verify", "refs/cquay/candidate^{commit}")
	if err != nil {
		return "", err
	}
	commit := strings.TrimSpace(string(data))
	if !validCommit(commit) {
		return "", failure(fault.Trust, "commit")
	}
	return commit, nil
}

func (s *store) acceptCandidate(ctx context.Context, previous accepted, commit string) (string, error) {
	if err := s.verifyCommit(ctx, commit); err != nil {
		return "", err
	}
	if err := s.ancestor(ctx, s.repo.InitialCommit, commit); err != nil {
		return "", err
	}
	if previous.Commit != "" {
		if err := s.ancestor(ctx, previous.Commit, commit); err != nil {
			return "", err
		}
	}
	directory, err := s.materialize(ctx, commit)
	if err != nil {
		return "", err
	}
	if err := s.writeAccepted(accepted{URL: s.repo.URL, Ref: s.repo.Ref, Commit: commit}); err != nil {
		return "", err
	}
	return directory, nil
}

// Current validates the last accepted data without network access. Cached data
// remains available during transport outages; provider decryption policy still
// controls whether a workload is allowed to use it.
func Current(repo model.Repository) (string, error) {
	s, err := openStore(repo, false)
	if err != nil {
		return "", err
	}
	state, err := s.readAccepted()
	if err != nil {
		return "", err
	}
	if state.Commit == "" {
		return "", failure(fault.Missing, "accepted_release")
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	if err = s.checkGitConfig(); err != nil {
		return "", err
	}
	if err = s.verifyCommit(ctx, state.Commit); err != nil {
		return "", err
	}
	if err = s.ancestor(ctx, repo.InitialCommit, state.Commit); err != nil {
		return "", err
	}
	tree, err := s.readTree(ctx, state.Commit)
	if err != nil {
		return "", err
	}
	directory := filepath.Join(repo.Directory, "releases", state.Commit)
	if err = verifyRelease(directory, tree); err != nil {
		return "", err
	}
	return directory, nil
}

func openStore(repo model.Repository, create bool) (*store, error) {
	if err := policy.ValidateRepository(repo); err != nil || repo.URL == "" {
		return nil, failure(fault.Invalid, "declaration")
	}
	if create {
		if err := platform.EnsurePrivateDirectory(repo.Directory); err != nil {
			return nil, failure(fault.Trust, "directory")
		}
	}
	if err := protectedDirectory(repo.Directory); err != nil {
		return nil, err
	}
	signers, err := readProtected(repo.AllowedSignersFile, false, 64<<10)
	if err != nil || len(strings.TrimSpace(string(signers))) == 0 {
		return nil, failure(fault.Trust, "signers")
	}
	return &store{repo: repo, git: filepath.Join(repo.Directory, "objects.git"), signers: signers}, nil
}

func (s *store) readAccepted() (accepted, error) {
	var result accepted
	path := filepath.Join(s.repo.Directory, "accepted.json")
	if _, err := os.Lstat(path); errors.Is(err, os.ErrNotExist) {
		return result, nil
	}
	data, err := readProtected(path, true, 4096)
	if err != nil {
		return result, err
	}
	if jsondocument.Decode(data, &result) != nil || result.URL != s.repo.URL || result.Ref != s.repo.Ref || !validCommit(result.Commit) {
		return accepted{}, failure(fault.Trust, "accepted_state")
	}
	return result, nil
}

func (s *store) writeAccepted(state accepted) error {
	data, err := json.Marshal(state)
	if err != nil {
		return failure(fault.Trust, "accepted_state")
	}
	return platform.WriteProtected(filepath.Join(s.repo.Directory, "accepted.json"), data)
}
