package repository

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/maldous/CircularQuay/internal/fault"
	"github.com/maldous/CircularQuay/internal/platform"
)

// Only explicit metadata errors, fully supplied finite Git replies and owned
// files participate. None of these callbacks selects a native tool or quota.
func TestFilesystemLifetimeContractMissingVerifierStopsBeforeValidation(t *testing.T) {
	calls := 0
	program, err := trustedSSHKeygenWithLookup(func(name string) (string, error) {
		calls++
		if name != "ssh-keygen" {
			t.Fatal("discovery selected an unrelated helper")
		}
		return "", errors.New("PRIVATE_DISCOVERY_DETAIL")
	})
	assertFilesystemLifetimeFault(t, err, fault.Unsupported, "ssh_signature_dependency")
	if program != "" || calls != 1 {
		t.Fatal("failed discovery returned a program or repeated lookup")
	}
}

func TestFilesystemLifetimeContractInitializationLookupFailureCannotRunGit(t *testing.T) {
	fixture := newRepositoryLifecycleFixture(t)
	calls := 0
	err := fixture.store.initializeWithLookup(context.Background(), func(path string) (os.FileInfo, error) {
		calls++
		if path != fixture.store.git {
			t.Fatal("metadata read escaped the fixed owned Git path")
		}
		return nil, errors.New("PRIVATE_METADATA_DETAIL")
	})
	assertFilesystemLifetimeFault(t, err, fault.Trust, "git_directory")
	if calls != 1 || len(fixture.calls) != 0 {
		t.Fatal("failed metadata read entered Git or repeated lookup")
	}
	assertRepositoryLifecycleBytes(t, filepath.Join(fixture.store.git, "config"), []byte(bareConfig))
}

func TestFilesystemLifetimeContractInitializationFailuresPreserveOwnerAndCleanTemplate(t *testing.T) {
	for _, phase := range []string{"command", "directory", "config"} {
		t.Run(phase, func(t *testing.T) { assertFilesystemInitialization(t, phase) })
	}
}

func assertFilesystemInitialization(t *testing.T, phase string) {
	t.Helper()
	fixture := newRepositoryLifecycleFixture(t)
	if err := os.RemoveAll(fixture.store.git); err != nil {
		t.Fatal(err)
	}
	probe := &filesystemInitializationProbe{fixture: fixture, phase: phase, previous: fixture.store.gitOperations.execute}
	fixture.store.gitOperations.execute = probe.execute
	release, err := fixture.store.fetch(context.Background(), fixture.fetchOperations())
	assertRepositoryLifecycleRefusal(t, release, err, fault.Trust)
	if !reflect.DeepEqual(fixture.calls, []string{"version", "init"}) {
		t.Fatal("failed initialization reached transport or later work")
	}
	assertRepositoryLifecycleCleaned(t, fixture.store)
	assertFilesystemNoTemplate(t, fixture.store.repo.Directory)
	probe.assertOwner(t)
}

type filesystemInitializationProbe struct {
	fixture  *repositoryLifecycleFixture
	phase    string
	previous func(context.Context, []string, int64, ...string) ([]byte, error)
}

func (p *filesystemInitializationProbe) execute(ctx context.Context, env []string, limit int64, args ...string) ([]byte, error) {
	if len(args) < 2 || args[1] != "init" {
		return p.previous(ctx, env, limit, args...)
	}
	p.assertInvocation(ctx, env, limit, args)
	p.fixture.calls = append(p.fixture.calls, "init")
	if p.phase == "command" {
		return nil, failure(fault.Trust, "synthetic_init_refusal")
	}
	if p.phase == "directory" {
		writeProtectedRepositoryFixture(p.fixture.t, p.fixture.store.git, []byte("OWNED_CONCURRENT_ENTRY"))
		return nil, nil
	}
	if err := platform.EnsurePrivateDirectory(p.fixture.store.git); err != nil {
		p.fixture.t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(p.fixture.store.git, "config"), 0o700); err != nil {
		p.fixture.t.Fatal(err)
	}
	return nil, nil
}

func (p *filesystemInitializationProbe) assertInvocation(ctx context.Context, env []string, limit int64, args []string) {
	p.fixture.t.Helper()
	p.assertInvocationContext(ctx, env, limit, args)
	if args[0] != "--git-dir="+p.fixture.store.git || args[2] != "--bare" || args[3] != "--quiet" || args[5] != p.fixture.store.git {
		p.fixture.t.Fatal("init changed its fixed owned invocation")
	}
	if !strings.HasPrefix(args[4], "--template=") {
		p.fixture.t.Fatal("init lost its isolated template")
	}
	template := strings.TrimPrefix(args[4], "--template=")
	if filepath.Dir(template) != p.fixture.store.repo.Directory || !strings.HasPrefix(filepath.Base(template), ".template-") {
		p.fixture.t.Fatal("init selected a template outside owned state")
	}
}

func (p *filesystemInitializationProbe) assertInvocationContext(ctx context.Context, env []string, limit int64, args []string) {
	p.fixture.t.Helper()
	if ctx.Err() != nil || limit != 4096 || !reflect.DeepEqual(env, []string{"SYNTHETIC_BOUNDARY=1"}) || len(args) != 6 {
		p.fixture.t.Fatal("init escaped bounded context, arguments or environment")
	}
}

func (p *filesystemInitializationProbe) assertOwner(t *testing.T) {
	t.Helper()
	if p.phase == "directory" {
		assertRepositoryLifecycleBytes(t, p.fixture.store.git, []byte("OWNED_CONCURRENT_ENTRY"))
	}
	if p.phase == "config" {
		entry, err := os.Stat(filepath.Join(p.fixture.store.git, "config"))
		if err != nil || !entry.IsDir() {
			t.Fatal("config refusal replaced its owned directory", err)
		}
	}
	if _, err := os.Lstat(filepath.Join(p.fixture.store.repo.Directory, "accepted.json")); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("initialization failure published accepted state")
	}
}

func assertFilesystemNoTemplate(t *testing.T, directory string) {
	t.Helper()
	entries, err := os.ReadDir(directory)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".template-") {
			t.Fatal("failed initialization retained its template")
		}
	}
}

func TestFilesystemLifetimeContractStagingFailureWithholdsRelease(t *testing.T) {
	fixture := newRepositoryLifecycleFixture(t)
	cleanup := prepareFilesystemLifetime(t, fixture.store)
	defer cleanup()
	calls := 0
	release, err := fixture.store.materializeWithStaging(context.Background(), fixture.commit, func(parent, pattern string) (string, error) {
		calls++
		if parent != filepath.Join(fixture.store.repo.Directory, "releases") || pattern != ".release-*" {
			t.Fatal("staging escaped the owned private releases root")
		}
		return "", errors.New("PRIVATE_STAGING_DETAIL")
	})
	assertFilesystemLifetimeFault(t, err, fault.Trust, "release_staging")
	if release != "" || calls != 1 || !reflect.DeepEqual(fixture.calls, []string{"ls-tree"}) {
		t.Fatal("staging failure released data or entered blob work")
	}
	assertNoMaterializationStaging(t, fixture)
}

func TestFilesystemLifetimeContractReleaseLookupFailureReturnsWholeZero(t *testing.T) {
	fixture := newRepositoryLifecycleFixture(t)
	calls := 0
	releases, destination, accepted, err := fixture.store.releaseDestinationWithLookup(context.Background(), fixture.commit, map[string]string{"values.json": fixture.object}, func(path string) (os.FileInfo, error) {
		calls++
		if path != fixture.releaseDirectory() {
			t.Fatal("release lookup escaped its admitted commit path")
		}
		return nil, errors.New("PRIVATE_RELEASE_METADATA_DETAIL")
	})
	assertFilesystemLifetimeFault(t, err, fault.Trust, "release_directory")
	if releases != "" || destination != "" || accepted || calls != 1 || len(fixture.calls) != 0 {
		t.Fatal("failed lookup returned a partial accepted destination or entered Git")
	}
}

func prepareFilesystemLifetime(t *testing.T, store *store) func() {
	t.Helper()
	cleanup, err := prepareCurrentVerificationTemporary(store)
	if err != nil {
		t.Fatal(err)
	}
	return func() {
		if err := cleanup(); err != nil {
			t.Error("owned verification cleanup failed", err)
		}
	}
}

func TestFilesystemLifetimeContractAggregateVerificationLimitHasRealOwnedFiles(t *testing.T) {
	fixture := newRepositoryLifecycleFixture(t)
	cleanup := prepareFilesystemLifetime(t, fixture.store)
	defer cleanup()
	tree := make(map[string]string)
	data := bytes.Repeat([]byte("x"), maximumFileBytes)
	if err := platform.EnsurePrivateDirectory(fixture.releaseDirectory()); err != nil {
		t.Fatal(err)
	}
	for i := 0; i <= maximumReleaseBytes/maximumFileBytes; i++ {
		name := fmt.Sprintf("owned-%02d.json", i)
		tree[name] = fixture.object
		writeProtectedRepositoryFixture(t, filepath.Join(fixture.releaseDirectory(), name), data)
	}
	err := fixture.store.verifyRelease(context.Background(), fixture.releaseDirectory(), tree)
	assertFilesystemLifetimeFault(t, err, fault.Trust, "release_size")
	if len(fixture.calls) != len(tree) {
		t.Fatal("aggregate refusal did not verify every preceding bounded file")
	}
	for _, call := range fixture.calls {
		if call != "hash-object" {
			t.Fatal("verification selected an undeclared fake operation")
		}
	}
}

func TestFilesystemLifetimeContractOwnedWriteCollisionCleansStage(t *testing.T) {
	fixture := newRepositoryLifecycleFixture(t)
	previous := fixture.store.gitOperations.execute
	fixture.store.gitOperations.execute = func(ctx context.Context, env []string, limit int64, args ...string) ([]byte, error) {
		data, err := previous(ctx, env, limit, args...)
		if err == nil && repositoryLifecycleAction(args[1:]) == "cat-file blob" {
			createFilesystemWriteCollision(t, fixture)
		}
		return data, err
	}
	release, err := fixture.store.fetch(context.Background(), fixture.fetchOperations())
	assertFilesystemLifetimeFault(t, err, fault.Trust, "release_write")
	if release != "" || fixture.calls[len(fixture.calls)-1] != "cat-file blob" {
		t.Fatal("write collision entered verification or published a release")
	}
	assertRepositoryLifecycleCleaned(t, fixture.store)
	assertNoMaterializationStaging(t, fixture)
}

func createFilesystemWriteCollision(t *testing.T, fixture *repositoryLifecycleFixture) {
	t.Helper()
	root := filepath.Join(fixture.store.repo.Directory, "releases")
	entries, err := os.ReadDir(root)
	if err != nil || len(entries) != 1 || !strings.HasPrefix(entries[0].Name(), ".release-") {
		t.Fatal("expected one owned staging directory", err)
	}
	if err := os.Mkdir(filepath.Join(root, entries[0].Name(), "values.json"), 0o700); err != nil {
		t.Fatal(err)
	}
}

func assertFilesystemLifetimeFault(t *testing.T, err error, kind fault.Kind, operation string) {
	t.Helper()
	operation = "repository_" + operation
	var typed *fault.Error
	if !errors.As(err, &typed) || err != typed || *typed != (fault.Error{Kind: kind, Operation: operation}) {
		t.Fatal("refusal lost exact sanitized fault", err)
	}
	if errors.Unwrap(err) != nil || err.Error() != string(kind)+": "+operation {
		t.Fatal("refusal retained upstream diagnostic or cause")
	}
}
