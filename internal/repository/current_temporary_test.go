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

func TestCurrentOfflineVerificationUsesPrivateTemporary(t *testing.T) {
	fixture := newSignedRepositoryFixture(t)
	first := fixture.writeCommit(t, `{"value":"synthetic-first"}`, true)
	repo := fixture.repository(first)
	installSyntheticTransport(t, repo, fixture)
	expected := fetchRepository(t, repo)
	fixture.removeToken(t)

	previous := runGit
	t.Cleanup(func() { runGit = previous })
	verificationTemporary := ""
	runGit = func(ctx context.Context, environment []string, limit int64, arguments ...string) ([]byte, error) {
		if slices.Contains(arguments, "verify-commit") {
			verificationTemporary = assertPrivateVerificationTemporary(t, environment)
			joined := strings.Join(environment, "\n")
			if strings.Contains(joined, "Authorization:") || strings.Contains(joined, "synthetic-fetch-token") {
				t.Fatal("offline verification inherited fetch credentials")
			}
		}
		return previous(ctx, environment, limit, arguments...)
	}

	current, err := Current(repo)
	if err != nil || current != expected || verificationTemporary == "" {
		t.Fatalf("offline current result: current=%q expected=%q temporary=%q err=%v", current, expected, verificationTemporary, err)
	}
	if _, err = os.Lstat(verificationTemporary); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("verification temporary survived Current: %v", err)
	}
}

func TestCurrentContextCancellationPreventsOfflineVerification(t *testing.T) {
	fixture := newSignedRepositoryFixture(t)
	first := fixture.writeCommit(t, `{"value":"synthetic-first"}`, true)
	repo := fixture.repository(first)
	installSyntheticTransport(t, repo, fixture)
	_ = fetchRepository(t, repo)
	fixture.removeToken(t)

	previous := runGit
	t.Cleanup(func() { runGit = previous })
	called := false
	runGit = func(context.Context, []string, int64, ...string) ([]byte, error) {
		called = true
		return nil, errors.New("cancelled Current must not start Git verification")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	current, err := CurrentContext(ctx, repo)
	if called || current != "" || !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled Current result: called=%t current=%q err=%v", called, current, err)
	}
}

func TestCurrentContextCancellationDuringVerificationCleansTemporary(t *testing.T) {
	fixture := newSignedRepositoryFixture(t)
	first := fixture.writeCommit(t, `{"value":"synthetic-first"}`, true)
	repo := fixture.repository(first)
	installSyntheticTransport(t, repo, fixture)
	_ = fetchRepository(t, repo)
	fixture.removeToken(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	previous := runGit
	t.Cleanup(func() { runGit = previous })
	temporary := ""
	runGit = func(ctx context.Context, environment []string, limit int64, arguments ...string) ([]byte, error) {
		data, err := previous(ctx, environment, limit, arguments...)
		if slices.Contains(arguments, "cat-file") && slices.Contains(arguments, "commit") {
			temporary = assertPrivateVerificationTemporary(t, environment)
			cancel()
		}
		return data, err
	}
	current, err := CurrentContext(ctx, repo)
	if current != "" || !errors.Is(err, context.Canceled) || temporary == "" {
		t.Fatalf("cancelled verification result: current=%q temporary=%q err=%v", current, temporary, err)
	}
	if _, err = os.Lstat(temporary); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("cancelled verification left temporary directory: %v", err)
	}
}

func TestCurrentVerificationTemporaryRestoresPriorStateAndFailsClosed(t *testing.T) {
	directory := privateRepositoryDirectory(t, "current-temporary")
	s := &store{repo: model.Repository{Directory: directory}, temporary: filepath.Join(directory, ".fetch-tmp")}
	cleanup, err := prepareCurrentVerificationTemporary(s)
	if err != nil {
		t.Fatal(err)
	}
	temporary := s.temporary
	if temporary == filepath.Join(directory, ".fetch-tmp") {
		t.Fatal("current verification reused fetch temporary")
	}
	assertPrivateVerificationTemporary(t, mustGitEnvironment(t, s))
	if err = cleanup(); err != nil {
		t.Fatal(err)
	}
	if s.temporary != filepath.Join(directory, ".fetch-tmp") {
		t.Fatal("current verification cleanup did not restore prior temporary state")
	}
	if _, err = os.Lstat(temporary); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("verification temporary was not removed: %v", err)
	}

	missing := &store{repo: model.Repository{Directory: filepath.Join(directory, "missing")}}
	if _, err = prepareCurrentVerificationTemporary(missing); !fault.Is(err, fault.Trust) {
		t.Fatalf("temporary creation failure did not fail closed: %v", err)
	}
}

func mustGitEnvironment(t *testing.T, s *store) []string {
	t.Helper()
	environment, err := s.environment(false, false)
	if err != nil {
		t.Fatal(err)
	}
	return environment
}

func assertPrivateVerificationTemporary(t *testing.T, environment []string) string {
	t.Helper()
	var temporary string
	for _, name := range []string{"TMPDIR", "TEMP", "TMP"} {
		value, found := environmentSetting(environment, name)
		if !found || value == "" {
			t.Fatalf("verification command lacks %s", name)
		}
		if temporary != "" && temporary != value {
			t.Fatalf("verification command has inconsistent temporary directories: %q and %q", temporary, value)
		}
		temporary = value
	}
	if err := protectedDirectory(temporary); err != nil {
		t.Fatalf("verification temporary is not protected: %v", err)
	}
	return temporary
}

func environmentSetting(environment []string, name string) (string, bool) {
	for _, entry := range environment {
		key, value, found := strings.Cut(entry, "=")
		if found && key == name {
			return value, true
		}
	}
	return "", false
}
