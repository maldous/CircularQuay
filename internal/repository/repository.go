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
	"github.com/maldous/CircularQuay/internal/quota"
)

const maximumFileBytes = 1 << 20
const maximumReleaseBytes = 16 << 20
const maximumFiles = 1024
const maximumRepositoryBytes = 512 << 20
const maximumRepositoryInodes = 100000
const repositoryTemporaryInodes = 128

type accepted struct {
	URL    string `json:"url"`
	Ref    string `json:"ref"`
	Commit string `json:"commit"`
}

type store struct {
	repo      model.Repository
	git       string
	signers   []byte
	temporary string
}

type fetchOperations struct {
	validateIdentity func() error
	assertQuota      func(string) error
	prepareTemporary func(*store) (func(), error)
	candidate        func(context.Context, *store) (string, error)
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
	return s.fetch(ctx, nativeFetchOperations())
}

func nativeFetchOperations() fetchOperations {
	return fetchOperations{
		validateIdentity: quota.ValidateFetchIdentity,
		assertQuota:      assertRepositoryQuota,
		prepareTemporary: prepareFetchTemporary,
		candidate: func(ctx context.Context, s *store) (string, error) {
			return s.fetchCandidate(ctx)
		},
	}
}

func (s *store) fetch(ctx context.Context, operations fetchOperations) (string, error) {
	if err := operations.validateIdentity(); err != nil {
		return "", failure(fault.Unsupported, "disk_quota_identity")
	}
	release, err := acquireFetchLock(s.repo.Directory)
	if err != nil {
		return "", err
	}
	defer release()
	if err = operations.assertQuota(s.repo.Directory); err != nil {
		return "", err
	}
	cleanup, err := operations.prepareTemporary(s)
	if err != nil {
		return "", err
	}
	defer cleanup()
	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	if err = s.requireHardenedSHA1(ctx); err != nil {
		return "", err
	}
	previous, err := s.readAccepted()
	if err != nil {
		return "", err
	}
	commit, err := operations.candidate(ctx, s)
	if err != nil {
		return "", err
	}
	return s.acceptCandidate(ctx, previous, commit)
}

func assertRepositoryQuota(directory string) error {
	err := quota.Check(quota.Native(), directory, repositoryQuotaLimits())
	if errors.Is(err, quota.ErrInsufficient) {
		return failure(fault.Unavailable, "disk_quota")
	}
	return mapQuotaError(err)
}

func repositoryQuotaLimits() quota.Limits {
	return quota.Limits{
		MaximumBytes:  maximumRepositoryBytes,
		MaximumInodes: maximumRepositoryInodes,
		ReserveBytes:  maximumReleaseBytes,
		ReserveInodes: maximumFiles + repositoryTemporaryInodes,
	}
}

func mapQuotaError(err error) error {
	if err == nil {
		return nil
	}
	return failure(fault.Unsupported, "disk_quota")
}

func prepareFetchTemporary(s *store) (func(), error) {
	directory := filepath.Join(s.repo.Directory, ".fetch-tmp")
	if err := platform.EnsurePrivateDirectory(directory); err != nil {
		return nil, failure(fault.Trust, "temporary_directory")
	}
	if err := protectedDirectory(directory); err != nil {
		return nil, err
	}
	s.temporary = directory
	return func() { s.temporary = "" }, nil
}

// prepareCurrentVerificationTemporary gives native Git and OpenSSH a protected
// temporary directory for offline signature verification. Current deliberately
// does not inherit a caller's temporary environment or reuse Fetch's lifetime.
func prepareCurrentVerificationTemporary(s *store) (func() error, error) {
	directory, err := os.MkdirTemp(s.repo.Directory, ".verify-tmp-")
	if err != nil {
		return nil, failure(fault.Trust, "verification_temporary")
	}
	if err = platform.EnsurePrivateDirectory(directory); err != nil {
		_ = os.RemoveAll(directory)
		return nil, failure(fault.Trust, "verification_temporary")
	}
	if err = protectedDirectory(directory); err != nil {
		_ = os.RemoveAll(directory)
		return nil, err
	}
	previous := s.temporary
	s.temporary = directory
	return func() error {
		s.temporary = previous
		if err := os.RemoveAll(directory); err != nil {
			return failure(fault.Trust, "verification_temporary")
		}
		return nil
	}, nil
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
	if err := s.verifyObjectGraph(ctx, commit); err != nil {
		return "", err
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

// Current validates the last accepted data without network access using a
// background context. New callers should pass their request context to
// CurrentContext so cancellation reaches native verification promptly.
func Current(repo model.Repository) (string, error) {
	return CurrentContext(context.Background(), repo)
}

// CurrentContext validates the last accepted data without network access.
// Cached data remains available during transport outages; provider decryption
// policy still controls whether a workload is allowed to use it. Verification
// is bounded to one minute and honors cancellation from ctx.
func CurrentContext(ctx context.Context, repo model.Repository) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	s, err := openStore(repo, false)
	if err != nil {
		return "", err
	}
	ctx, cancel := context.WithTimeout(ctx, time.Minute)
	defer cancel()
	return s.currentRelease(ctx, repo)
}

func (s *store) currentRelease(ctx context.Context, repo model.Repository) (string, error) {
	state, err := s.readAccepted()
	if err != nil {
		return "", err
	}
	if state.Commit == "" {
		return "", failure(fault.Missing, "accepted_release")
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}
	cleanup, err := prepareCurrentVerificationTemporary(s)
	if err != nil {
		return "", err
	}
	release, verificationErr := s.verifyCurrentRelease(ctx, repo, state.Commit)
	cleanupErr := cleanup()
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if verificationErr != nil {
		return "", verificationErr
	}
	if cleanupErr != nil {
		return "", cleanupErr
	}
	return release, nil
}

func (s *store) verifyCurrentRelease(ctx context.Context, repo model.Repository, commit string) (string, error) {
	if err := s.checkGitConfig(); err != nil {
		return "", err
	}
	if err := s.verifyCommit(ctx, commit); err != nil {
		return "", err
	}
	if err := s.ancestor(ctx, repo.InitialCommit, commit); err != nil {
		return "", err
	}
	if err := s.requireHardenedSHA1(ctx); err != nil {
		return "", err
	}
	if err := s.verifyObjectGraph(ctx, commit); err != nil {
		return "", err
	}
	tree, err := s.readTree(ctx, commit)
	if err != nil {
		return "", err
	}
	release := filepath.Join(repo.Directory, "releases", commit)
	if err = s.verifyRelease(ctx, release, tree); err != nil {
		return "", err
	}
	return release, nil
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
