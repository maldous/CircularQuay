package repository

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/maldous/CircularQuay/internal/fault"
	"github.com/maldous/CircularQuay/internal/model"
)

func TestCurrentRejectsAcceptedStateBeforeGit(t *testing.T) {
	repo, release, commit := fetchedCurrentFixture(t)
	acceptedPath := filepath.Join(repo.Directory, "accepted.json")
	cases := []struct {
		name string
		data []byte
		kind fault.Kind
	}{
		{"missing", nil, fault.Missing},
		{"malformed", []byte(`{"url":`), fault.Trust},
		{"wrong_url", acceptedStateJSON("https://example.invalid/other.git", repo.Ref, commit), fault.Trust},
		{"wrong_ref", acceptedStateJSON(repo.URL, "refs/heads/other", commit), fault.Trust},
		{"invalid_commit", acceptedStateJSON(repo.URL, repo.Ref, "not-a-commit"), fault.Trust},
	}
	previous := runGit
	t.Cleanup(func() { runGit = previous })
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			assertRejectedAcceptedState(t, repo, release, acceptedPath, test.data, test.kind, previous)
		})
	}
}

func assertRejectedAcceptedState(t *testing.T, repo model.Repository, release, acceptedPath string, data []byte, kind fault.Kind, previous gitRunner) {
	t.Helper()
	writeOrRemoveAcceptedState(t, acceptedPath, data)
	called := false
	runGit = func(context.Context, []string, int64, ...string) ([]byte, error) {
		called = true
		return nil, errors.New("Git must not run for rejected accepted state")
	}
	current, err := Current(repo)
	if called || current != "" || !fault.Is(err, kind) {
		t.Fatalf("accepted state denial: called=%t current=%q err=%v", called, current, err)
	}
	if _, err := os.Stat(release); err != nil {
		t.Fatalf("rejected state damaged the previously trusted release: %v", err)
	}
	runGit = previous
}

func writeOrRemoveAcceptedState(t *testing.T, path string, data []byte) {
	t.Helper()
	if data == nil {
		if err := os.Remove(path); err != nil {
			t.Fatal(err)
		}
		return
	}
	writeProtectedRepositoryFixture(t, path, data)
}

func TestCurrentRejectsChangedGitConfigBeforeGit(t *testing.T) {
	repo, release, _ := fetchedCurrentFixture(t)
	writeProtectedRepositoryFixture(t, filepath.Join(repo.Directory, "objects.git", "config"), []byte(strings.Replace(bareConfig, "filemode = false", "filemode = true", 1)))
	previous := runGit
	t.Cleanup(func() { runGit = previous })
	called := false
	runGit = func(context.Context, []string, int64, ...string) ([]byte, error) {
		called = true
		return nil, errors.New("Git must not run with changed local config")
	}
	current, err := Current(repo)
	if called || current != "" || !fault.Is(err, fault.Trust) {
		t.Fatalf("changed config result: called=%t current=%q err=%v", called, current, err)
	}
	if _, err := os.Stat(release); err != nil {
		t.Fatalf("changed config damaged prior release: %v", err)
	}
}

func TestCurrentStopsBeforeReleaseForGitIntegrityFailures(t *testing.T) {
	for _, test := range []struct {
		name, rejectedCommand string
		output                []byte
		kind                  fault.Kind
	}{
		{"unsupported_sha1", "version", []byte("git version 2.12.9\n"), fault.Unsupported},
		{"object_graph", "fsck", nil, fault.Trust},
	} {
		t.Run(test.name, func(t *testing.T) {
			assertCurrentStopsForGitIntegrityFailure(t, test.rejectedCommand, test.output, test.kind)
		})
	}
}

type gitRunner func(context.Context, []string, int64, ...string) ([]byte, error)

func assertCurrentStopsForGitIntegrityFailure(t *testing.T, rejectedCommand string, output []byte, kind fault.Kind) {
	t.Helper()
	repo, release, _ := fetchedCurrentFixture(t)
	previous := runGit
	t.Cleanup(func() { runGit = previous })
	seenRejected := false
	runGit = integrityRejectingGit(t, previous, rejectedCommand, output, &seenRejected)
	current, err := Current(repo)
	if !seenRejected || current != "" || !fault.Is(err, kind) {
		t.Fatalf("Git integrity denial: rejected=%t current=%q err=%v", seenRejected, current, err)
	}
	if _, err := os.Stat(release); err != nil {
		t.Fatalf("Git integrity denial damaged prior release: %v", err)
	}
}

func integrityRejectingGit(t *testing.T, previous gitRunner, rejectedCommand string, output []byte, seen *bool) gitRunner {
	t.Helper()
	return func(ctx context.Context, environment []string, limit int64, arguments ...string) ([]byte, error) {
		assertReplacementObjectsDisabled(t, environment)
		if slices.Contains(arguments, "ls-tree") || slices.Contains(arguments, "hash-object") {
			t.Fatal("Current read release tree after Git integrity failure")
		}
		if !slices.Contains(arguments, rejectedCommand) {
			return previous(ctx, environment, limit, arguments...)
		}
		*seen = true
		if output != nil {
			return output, nil
		}
		return nil, errors.New("synthetic Git integrity failure")
	}
}

func TestCurrentRejectsMaterializedReleaseTreeMismatch(t *testing.T) {
	repo, release, _ := fetchedCurrentFixture(t)
	writeProtectedRepositoryFixture(t, filepath.Join(release, "unexpected.json"), []byte(`{"value":"untrusted"}`))
	current, err := Current(repo)
	if current != "" || !fault.Is(err, fault.Trust) {
		t.Fatalf("materialized release tree mismatch result: current=%q err=%v", current, err)
	}
}

func fetchedCurrentFixture(t *testing.T) (model.Repository, string, string) {
	t.Helper()
	fixture := newSignedRepositoryFixture(t)
	commit := fixture.writeCommit(t, `{"value":"synthetic-first"}`, true)
	repo := fixture.repository(commit)
	installSyntheticTransport(t, repo, fixture)
	release := fetchRepository(t, repo)
	fixture.removeToken(t)
	return repo, release, commit
}

func acceptedStateJSON(url, ref, commit string) []byte {
	return []byte(`{"url":"` + url + `","ref":"` + ref + `","commit":"` + commit + `"}`)
}

func assertReplacementObjectsDisabled(t *testing.T, environment []string) {
	t.Helper()
	value, found := environmentSetting(environment, "GIT_NO_REPLACE_OBJECTS")
	if !found || value != "1" {
		t.Fatal("Current permitted Git replacement objects")
	}
}
