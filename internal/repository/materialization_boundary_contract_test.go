package repository

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/maldous/CircularQuay/internal/fault"
)

func TestRepositoryMaterializationBoundaryContractCurrentPreparationStopsVerification(t *testing.T) {
	fixture := newRepositoryLifecycleFixture(t)
	fixture.prepareAccepted(t)
	wanted := failure(fault.Trust, "verification_temporary")
	calls := 0
	prepare := func(s *store) (func() error, error) {
		calls++
		if s != fixture.store {
			t.Fatal("preparation received another store")
		}
		return nil, wanted
	}
	current, err := fixture.store.currentReleaseWithPreparation(context.Background(), fixture.store.repo, prepare)
	if current != "" || err != wanted || calls != 1 || len(fixture.calls) != 0 {
		t.Fatal("failed preparation reached verification or produced a release")
	}
	assertRepositoryLifecycleBytes(t, filepath.Join(fixture.releaseDirectory(), "values.json"), []byte("OWNED_SYNTHETIC_DATA"))
}

func TestRepositoryMaterializationBoundaryContractCurrentCleanupFailureCannotSucceed(t *testing.T) {
	fixture := newRepositoryLifecycleFixture(t)
	fixture.prepareAccepted(t)
	prepare := failingCurrentCleanup(t, false, func() {})
	current, err := fixture.store.currentReleaseWithPreparation(context.Background(), fixture.store.repo, prepare)
	assertRepositoryLifecycleRefusal(t, current, err, fault.Trust)
	if fixture.store.temporary != "" {
		t.Fatal("cleanup failure retained the temporary selection")
	}
	assertRepositoryLifecycleBytes(t, filepath.Join(fixture.releaseDirectory(), "values.json"), []byte("OWNED_SYNTHETIC_DATA"))
}

func TestRepositoryMaterializationBoundaryContractCancellationHasCleanupPriority(t *testing.T) {
	fixture := newRepositoryLifecycleFixture(t)
	fixture.prepareAccepted(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	current, err := fixture.store.currentReleaseWithPreparation(ctx, fixture.store.repo, failingCurrentCleanup(t, true, cancel))
	if current != "" || !errors.Is(err, context.Canceled) || fixture.store.temporary != "" {
		t.Fatalf("post-verification cancellation was replaced by cleanup outcome: %q, %v", current, err)
	}
	assertRepositoryLifecycleBytes(t, filepath.Join(fixture.releaseDirectory(), "values.json"), []byte("OWNED_SYNTHETIC_DATA"))
}

func failingCurrentCleanup(t *testing.T, cancelAfterCleanup bool, cancel context.CancelFunc) func(*store) (func() error, error) {
	t.Helper()
	return func(s *store) (func() error, error) {
		cleanup, err := prepareCurrentVerificationTemporary(s)
		if err != nil {
			return nil, err
		}
		return func() error {
			if err := cleanup(); err != nil {
				t.Fatal(err)
			}
			if cancelAfterCleanup {
				cancel()
			}
			return failure(fault.Trust, "verification_temporary")
		}, nil
	}
}

func TestRepositoryMaterializationBoundaryContractCancelledCurrentSkipsPreparation(t *testing.T) {
	fixture := newRepositoryLifecycleFixture(t)
	fixture.prepareAccepted(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	prepare := func(*store) (func() error, error) {
		t.Fatal("cancelled current release reached preparation")
		return nil, nil
	}
	current, err := fixture.store.currentReleaseWithPreparation(ctx, fixture.store.repo, prepare)
	if current != "" || !errors.Is(err, context.Canceled) || len(fixture.calls) != 0 {
		t.Fatal("cancelled current release published data or reached fake Git")
	}
}

func TestRepositoryMaterializationBoundaryContractChangedConfigStopsFetchCandidate(t *testing.T) {
	fixture := newRepositoryLifecycleFixture(t)
	writeProtectedRepositoryFixture(t, filepath.Join(fixture.store.git, "config"), []byte("UNTRUSTED_CONFIG"))
	release, err := fixture.store.fetch(context.Background(), fixture.fetchOperations())
	assertRepositoryLifecycleRefusal(t, release, err, fault.Trust)
	if strings.Join(fixture.calls, ",") != "version" {
		t.Fatalf("changed config reached transport: %v", fixture.calls)
	}
	assertRepositoryLifecycleBytes(t, filepath.Join(fixture.store.git, "config"), []byte("UNTRUSTED_CONFIG"))
	assertRepositoryLifecycleCleaned(t, fixture.store)
}

func TestRepositoryMaterializationBoundaryContractMalformedResolvedCommitStopsAcceptance(t *testing.T) {
	fixture := newRepositoryLifecycleFixture(t)
	previous := fixture.store.gitOperations.execute
	fixture.store.gitOperations.execute = func(ctx context.Context, env []string, limit int64, arguments ...string) ([]byte, error) {
		data, err := previous(ctx, env, limit, arguments...)
		if len(arguments) >= 2 && arguments[1] == "rev-parse" {
			return []byte("not-a-commit\n"), nil
		}
		return data, err
	}
	release, err := fixture.store.fetch(context.Background(), fixture.fetchOperations())
	assertRepositoryLifecycleRefusal(t, release, err, fault.Trust)
	if fixture.calls[len(fixture.calls)-1] != "rev-parse" {
		t.Fatal("malformed resolved commit reached signature or release work")
	}
	assertRepositoryLifecycleCleaned(t, fixture.store)
}

func TestRepositoryMaterializationBoundaryContractOccupiedReleaseRootPreservesOwner(t *testing.T) {
	fixture := newRepositoryLifecycleFixture(t)
	root := filepath.Join(fixture.store.repo.Directory, "releases")
	writeProtectedRepositoryFixture(t, root, []byte("EXISTING_OWNER"))
	release, err := fixture.store.fetch(context.Background(), fixture.fetchOperations())
	assertRepositoryLifecycleRefusal(t, release, err, fault.Trust)
	assertRepositoryLifecycleBytes(t, root, []byte("EXISTING_OWNER"))
	assertRepositoryLifecycleCleaned(t, fixture.store)
}

func TestRepositoryMaterializationBoundaryContractRenameCollisionNeverReplacesOwner(t *testing.T) {
	fixture := newRepositoryLifecycleFixture(t)
	fixture.afterHash = func() {
		if err := os.Mkdir(fixture.releaseDirectory(), 0o700); err != nil {
			t.Fatal(err)
		}
		writeProtectedRepositoryFixture(t, filepath.Join(fixture.releaseDirectory(), "owner"), []byte("EXISTING_OWNER"))
	}
	release, err := fixture.store.fetch(context.Background(), fixture.fetchOperations())
	assertRepositoryLifecycleRefusal(t, release, err, fault.Trust)
	assertRepositoryLifecycleBytes(t, filepath.Join(fixture.releaseDirectory(), "owner"), []byte("EXISTING_OWNER"))
	assertRepositoryLifecycleCleaned(t, fixture.store)
	assertNoMaterializationStaging(t, fixture)
}

func TestRepositoryMaterializationBoundaryContractAggregateByteLimitCleansStage(t *testing.T) {
	fixture := newRepositoryLifecycleFixture(t)
	data := bytes.Repeat([]byte("x"), maximumFileBytes)
	var tree strings.Builder
	for index := 0; index < maximumReleaseBytes/maximumFileBytes+1; index++ {
		fmt.Fprintf(&tree, "100644 blob %s\tfile-%02d.json\x00", fixture.object, index)
	}
	overrideMaterializationReplies(fixture, tree.String(), data)
	release, err := fixture.store.fetch(context.Background(), fixture.fetchOperations())
	assertRepositoryLifecycleRefusal(t, release, err, fault.Trust)
	assertRepositoryLifecycleCleaned(t, fixture.store)
	assertNoMaterializationStaging(t, fixture)
}

func overrideMaterializationReplies(fixture *repositoryLifecycleFixture, tree string, data []byte) {
	previous := fixture.store.gitOperations.execute
	fixture.store.gitOperations.execute = func(ctx context.Context, env []string, limit int64, arguments ...string) ([]byte, error) {
		result, err := previous(ctx, env, limit, arguments...)
		if err != nil {
			return nil, err
		}
		switch repositoryLifecycleAction(arguments[1:]) {
		case "ls-tree":
			return []byte(tree), nil
		case "cat-file blob":
			return data, nil
		default:
			return result, nil
		}
	}
}

func TestRepositoryMaterializationBoundaryContractExistingExactReleaseIsReconciled(t *testing.T) {
	fixture := newRepositoryLifecycleFixture(t)
	fixture.prepareAccepted(t)
	path := filepath.Join(fixture.releaseDirectory(), "values.json")
	before := eagerMaterializationIdentity(t, path)
	release, err := fixture.store.fetch(context.Background(), fixture.fetchOperations())
	if err != nil || release != fixture.releaseDirectory() {
		t.Fatalf("exact existing release was not reconciled: %q, %v", release, err)
	}
	after, err := os.Stat(path)
	if err != nil || !os.SameFile(before, after) {
		t.Fatal("reconciliation replaced existing materialized data")
	}
	assertNoMaterializationStaging(t, fixture)
}

// Open-file Stat captures Windows identity before the operation. A path-derived
// FileInfo may otherwise load its file ID lazily after replacement occurred.
func eagerMaterializationIdentity(t *testing.T, path string) os.FileInfo {
	t.Helper()
	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	info, statErr := file.Stat()
	closeErr := file.Close()
	if statErr != nil || closeErr != nil {
		t.Fatalf("capture owned file identity: stat=%v close=%v", statErr, closeErr)
	}
	return info
}

func assertNoMaterializationStaging(t *testing.T, fixture *repositoryLifecycleFixture) {
	t.Helper()
	entries, err := os.ReadDir(filepath.Join(fixture.store.repo.Directory, "releases"))
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".release-") {
			t.Fatal("refusal/reconciliation left owned staging data")
		}
	}
}
