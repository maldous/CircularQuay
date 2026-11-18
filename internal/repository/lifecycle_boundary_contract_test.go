package repository

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/maldous/CircularQuay/internal/fault"
	"github.com/maldous/CircularQuay/internal/model"
	"github.com/maldous/CircularQuay/internal/platform"
)

// This cohort checks orchestration and owned-file lifetimes. Synthetic command
// replies are not cryptographic signatures, Git execution or quota attestation.
func TestRepositoryLifecycleBoundaryContractFetchRefusalsNeverPublishState(t *testing.T) {
	for _, test := range []struct {
		stage string
		kind  fault.Kind
	}{
		{"version", fault.Unsupported},
		{"fetch", fault.Unavailable},
		{"rev-parse", fault.Trust},
		{"cat-file commit", fault.Trust},
		{"verify-commit", fault.Trust},
		{"merge-base", fault.Trust},
		{"fsck", fault.Trust},
		{"ls-tree", fault.Trust},
		{"cat-file blob", fault.Trust},
		{"hash-object", fault.Trust},
	} {
		t.Run(test.stage, func(t *testing.T) {
			assertRepositoryLifecycleFetchRefusal(t, test.stage, test.kind)
		})
	}
}

func assertRepositoryLifecycleFetchRefusal(t *testing.T, stage string, kind fault.Kind) {
	t.Helper()
	fixture := newRepositoryLifecycleFixture(t)
	fixture.fail = stage
	release, err := fixture.store.fetch(context.Background(), fixture.fetchOperations())
	assertRepositoryLifecycleRefusal(t, release, err, kind)
	if fixture.calls[len(fixture.calls)-1] != stage {
		t.Fatalf("pipeline continued after refusal: %v", fixture.calls)
	}
	assertRepositoryLifecycleCleaned(t, fixture.store)
	if _, err := os.Lstat(filepath.Join(fixture.store.repo.Directory, "accepted.json")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("refused candidate published accepted state: %v", err)
	}
	entries, err := os.ReadDir(filepath.Join(fixture.store.repo.Directory, "releases"))
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatal("refused candidate left a release or staging entry")
	}
}

func TestRepositoryLifecycleBoundaryContractSyntheticAcceptanceAndCurrent(t *testing.T) {
	fixture := newRepositoryLifecycleFixture(t)
	release, err := fixture.store.fetch(context.Background(), fixture.fetchOperations())
	assertRepositoryLifecycleAccepted(t, fixture, release, err)
	assertRepositoryLifecycleCleaned(t, fixture.store)
	fixture.calls = nil
	current, err := fixture.store.currentRelease(context.Background(), fixture.store.repo)
	if err != nil || current != release {
		t.Fatalf("owned current release was not retained: %q, %v", current, err)
	}
	if fixture.store.temporary != "" {
		t.Fatal("Current retained temporary state")
	}
	assertRepositoryLifecycleBytes(t, filepath.Join(release, "values.json"), []byte("OWNED_SYNTHETIC_DATA"))
}

func assertRepositoryLifecycleAccepted(t *testing.T, fixture *repositoryLifecycleFixture, release string, err error) {
	t.Helper()
	if err != nil || release != fixture.releaseDirectory() {
		t.Fatalf("complete synthetic pipeline did not publish exact release: %q, %v", release, err)
	}
	assertRepositoryLifecycleBytes(t, filepath.Join(release, "values.json"), []byte("OWNED_SYNTHETIC_DATA"))
	state, readErr := fixture.store.readAccepted()
	if readErr != nil || state.Commit != fixture.commit || state.URL != fixture.store.repo.URL || state.Ref != fixture.store.repo.Ref {
		t.Fatalf("accepted metadata changed: %#v, %v", state, readErr)
	}
}

func TestRepositoryLifecycleBoundaryContractCurrentRefusalsRetainPriorData(t *testing.T) {
	for _, stage := range []string{"merge-base", "ls-tree", "hash-object"} {
		t.Run(stage, func(t *testing.T) {
			fixture := newRepositoryLifecycleFixture(t)
			fixture.prepareAccepted(t)
			fixture.fail = stage
			current, err := fixture.store.currentRelease(context.Background(), fixture.store.repo)
			assertRepositoryLifecycleRefusal(t, current, err, fault.Trust)
			assertRepositoryLifecycleBytes(t, filepath.Join(fixture.releaseDirectory(), "values.json"), []byte("OWNED_SYNTHETIC_DATA"))
			if fixture.store.temporary != "" || fixture.calls[len(fixture.calls)-1] != stage {
				t.Fatal("current refusal leaked temporary state or continued verification")
			}
		})
	}
}

func TestRepositoryLifecycleBoundaryContractMalformedAcceptedStateStopsCandidate(t *testing.T) {
	fixture := newRepositoryLifecycleFixture(t)
	writeProtectedRepositoryFixture(t, filepath.Join(fixture.store.repo.Directory, "accepted.json"), []byte("not-json"))
	release, err := fixture.store.fetch(context.Background(), fixture.fetchOperations())
	assertRepositoryLifecycleRefusal(t, release, err, fault.Trust)
	if !reflect.DeepEqual(fixture.calls, []string{"version"}) {
		t.Fatalf("invalid prior state reached candidate work: %v", fixture.calls)
	}
	assertRepositoryLifecycleBytes(t, filepath.Join(fixture.store.repo.Directory, "accepted.json"), []byte("not-json"))
	assertRepositoryLifecycleCleaned(t, fixture.store)
}

func TestRepositoryLifecycleBoundaryContractPublicationCollisionPreservesOwner(t *testing.T) {
	fixture := newRepositoryLifecycleFixture(t)
	fixture.afterHash = func() {
		path := filepath.Join(fixture.store.repo.Directory, "accepted.json")
		if err := os.Mkdir(path, 0o700); err != nil {
			t.Fatal(err)
		}
		writeProtectedRepositoryFixture(t, filepath.Join(path, "owner"), []byte("EXISTING_OWNER"))
	}
	release, err := fixture.store.fetch(context.Background(), fixture.fetchOperations())
	assertRepositoryLifecycleRefusal(t, release, err, fault.Trust)
	assertRepositoryLifecycleBytes(t, filepath.Join(fixture.store.repo.Directory, "accepted.json", "owner"), []byte("EXISTING_OWNER"))
	assertRepositoryLifecycleBytes(t, filepath.Join(fixture.releaseDirectory(), "values.json"), []byte("OWNED_SYNTHETIC_DATA"))
	assertRepositoryLifecycleCleaned(t, fixture.store)
}

type repositoryLifecycleFixture struct {
	t         *testing.T
	store     *store
	commit    string
	object    string
	fail      string
	calls     []string
	flags     [][2]bool
	afterHash func()
}

func newRepositoryLifecycleFixture(t *testing.T) *repositoryLifecycleFixture {
	t.Helper()
	directory := privateRepositoryDirectory(t, "lifecycle")
	fixture := &repositoryLifecycleFixture{t: t, commit: strings.Repeat("a", 40), object: strings.Repeat("b", 40)}
	fixture.store = &store{repo: model.Repository{Directory: directory, URL: "https://github.com/example/synthetic.git", Ref: "refs/heads/data", InitialCommit: strings.Repeat("c", 40)}, git: filepath.Join(directory, "objects.git")}
	if err := platform.EnsurePrivateDirectory(fixture.store.git); err != nil {
		t.Fatal(err)
	}
	writeProtectedRepositoryFixture(t, filepath.Join(fixture.store.git, "config"), []byte(bareConfig))
	fixture.store.gitOperations = &gitOperations{environment: fixture.environment, execute: fixture.execute}
	return fixture
}

func (fixture *repositoryLifecycleFixture) environment(s *store, authenticated, signature bool) ([]string, error) {
	fixture.t.Helper()
	if s != fixture.store || s.temporary == "" {
		fixture.t.Fatal("command escaped its owned store or temporary lifetime")
	}
	// Flags remain observable; they never select a real environment or token.
	fixture.flags = append(fixture.flags, [2]bool{authenticated, signature})
	return []string{"SYNTHETIC_BOUNDARY=1"}, nil
}

func (fixture *repositoryLifecycleFixture) execute(ctx context.Context, environment []string, limit int64, arguments ...string) ([]byte, error) {
	fixture.t.Helper()
	if ctx.Err() != nil || limit <= 0 || !reflect.DeepEqual(environment, []string{"SYNTHETIC_BOUNDARY=1"}) {
		fixture.t.Fatal("command escaped finite synthetic metadata")
	}
	if len(arguments) < 2 || arguments[0] != "--git-dir="+fixture.store.git {
		fixture.t.Fatal("command escaped the fixed owned Git directory")
	}
	action := repositoryLifecycleAction(arguments[1:])
	fixture.assertFlags(action)
	fixture.calls = append(fixture.calls, action)
	if action == fixture.fail {
		return nil, failure(fault.Trust, "synthetic_git_refusal")
	}
	return fixture.reply(action)
}

func (fixture *repositoryLifecycleFixture) assertFlags(action string) {
	fixture.t.Helper()
	wanted := [2]bool{action == "fetch", action == "verify-commit"}
	if len(fixture.flags) == 0 || fixture.flags[len(fixture.flags)-1] != wanted {
		fixture.t.Fatal("command lost its authenticated-fetch/signature-verification classification")
	}
}

func repositoryLifecycleAction(arguments []string) string {
	if arguments[0] == "cat-file" && len(arguments) >= 2 {
		return "cat-file " + arguments[1]
	}
	return arguments[0]
}

func (fixture *repositoryLifecycleFixture) reply(action string) ([]byte, error) {
	fixture.t.Helper()
	switch action {
	case "version":
		return []byte("git version 2.50.0\n"), nil
	case "rev-parse":
		return []byte(fixture.commit + "\n"), nil
	case "cat-file commit":
		return []byte("tree " + fixture.object + "\ngpgsig -----BEGIN SSH SIGNATURE-----\n fixture\n\nsynthetic\n"), nil
	case "ls-tree":
		return []byte("100644 blob " + fixture.object + "\tvalues.json\x00"), nil
	case "cat-file blob":
		return []byte("OWNED_SYNTHETIC_DATA"), nil
	default:
		return fixture.replyVerification(action)
	}
}

func (fixture *repositoryLifecycleFixture) replyVerification(action string) ([]byte, error) {
	fixture.t.Helper()
	switch action {
	case "fetch", "verify-commit", "merge-base", "fsck":
		return nil, nil
	case "hash-object":
		if fixture.afterHash != nil {
			fixture.afterHash()
			fixture.afterHash = nil
		}
		return []byte(fixture.object + "\n"), nil
	default:
		fixture.t.Fatalf("undeclared synthetic Git action: %s", action)
		return nil, nil
	}
}

func (fixture *repositoryLifecycleFixture) fetchOperations() fetchOperations {
	return fetchOperations{
		validateIdentity: func() error { return nil },
		assertQuota:      func(string) error { return nil },
		prepareTemporary: prepareFetchTemporary,
		candidate: func(ctx context.Context, s *store) (string, error) {
			return s.fetchCandidate(ctx)
		},
	}
}

func (fixture *repositoryLifecycleFixture) releaseDirectory() string {
	return filepath.Join(fixture.store.repo.Directory, "releases", fixture.commit)
}

func (fixture *repositoryLifecycleFixture) prepareAccepted(t *testing.T) {
	t.Helper()
	if err := platform.EnsurePrivateDirectory(fixture.releaseDirectory()); err != nil {
		t.Fatal(err)
	}
	writeProtectedRepositoryFixture(t, filepath.Join(fixture.releaseDirectory(), "values.json"), []byte("OWNED_SYNTHETIC_DATA"))
	if err := fixture.store.writeAccepted(accepted{URL: fixture.store.repo.URL, Ref: fixture.store.repo.Ref, Commit: fixture.commit}); err != nil {
		t.Fatal(err)
	}
}

func assertRepositoryLifecycleRefusal(t *testing.T, result string, err error, kind fault.Kind) {
	t.Helper()
	if result != "" || !fault.Is(err, kind) || strings.Contains(err.Error(), "PRIVATE_DATA") {
		t.Fatalf("refusal published a result or unsafe diagnostic: %q, %v", result, err)
	}
}

func assertRepositoryLifecycleBytes(t *testing.T, path string, wanted []byte) {
	t.Helper()
	actual, err := os.ReadFile(path)
	if err != nil || string(actual) != string(wanted) {
		t.Fatalf("owned bytes changed: %v", err)
	}
}

func assertRepositoryLifecycleCleaned(t *testing.T, s *store) {
	t.Helper()
	if s.temporary != "" {
		t.Fatal("fetch temporary state survived completion")
	}
	if _, err := os.Lstat(filepath.Join(s.repo.Directory, "fetch.lock")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("fetch lock survived completion: %v", err)
	}
}
