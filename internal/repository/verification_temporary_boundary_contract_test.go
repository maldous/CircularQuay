package repository

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/maldous/CircularQuay/internal/fault"
	"github.com/maldous/CircularQuay/internal/platform"
)

func TestVerificationTemporaryBoundaryContractPreparationFailureRestoresState(t *testing.T) {
	for _, stage := range []string{"create", "ensure", "verify"} {
		t.Run(stage, func(t *testing.T) {
			assertVerificationTemporaryPreparationRefusal(t, stage)
		})
	}
}

func assertVerificationTemporaryPreparationRefusal(t *testing.T, stage string) {
	t.Helper()
	fixture := newVerificationTemporaryFixture(t, stage)
	cleanup, err := prepareVerificationTemporary(fixture.store, fixture.operations())
	if cleanup != nil {
		t.Fatal("failed preparation returned a usable cleanup/lifetime handle")
	}
	if stage == "verify" {
		assertVerificationTemporaryError(t, err, "private directory permissions")
	} else {
		assertVerificationTemporaryError(t, err, "repository_verification_temporary")
	}
	fixture.assertPrevious(t)
	want := map[string][]string{
		"create": {"create"},
		"ensure": {"create", "ensure", "remove"},
		"verify": {"create", "ensure", "verify", "remove"},
	}
	if !reflect.DeepEqual(fixture.events, want[stage]) {
		t.Fatal("preparation refusal changed validation/cleanup order")
	}
	if fixture.directory != "" {
		assertVerificationTemporaryAbsent(t, fixture.directory)
	}
}

func TestVerificationTemporaryBoundaryContractCleanupFailureIsNotSuccess(t *testing.T) {
	fixture := newVerificationTemporaryFixture(t, "remove")
	cleanup, err := prepareVerificationTemporary(fixture.store, fixture.operations())
	if err != nil || cleanup == nil || fixture.store.temporary != fixture.directory {
		t.Fatal("valid owned preparation did not establish its scoped directory", err)
	}
	assertVerificationTemporaryError(t, cleanup(), "repository_verification_temporary")
	fixture.assertPrevious(t)
	if !reflect.DeepEqual(fixture.events, []string{"create", "ensure", "verify", "remove"}) {
		t.Fatal("cleanup failure changed operation order")
	}
	// The failed removal is explicitly not evidence of successful cleanup.
	// The enclosing owned TempDir remains responsible for this test fixture.
	info, err := os.Stat(fixture.directory)
	if err != nil || !info.IsDir() {
		t.Fatal("failed removal was incorrectly described as deleting its directory")
	}
}

func TestVerificationTemporaryBoundaryContractProductionDefaultsOwnLifecycle(t *testing.T) {
	fixture := newVerificationTemporaryFixture(t, "")
	cleanup, err := prepareCurrentVerificationTemporary(fixture.store)
	if err != nil || cleanup == nil {
		t.Fatal("default filesystem preparation rejected an owned valid store", err)
	}
	directory := fixture.store.temporary
	if directory == fixture.previous || filepath.Dir(directory) != fixture.store.repo.Directory {
		t.Fatal("default preparation reused prior scope or escaped its owned cache")
	}
	if err := platform.ValidatePrivateDirectory(directory); err != nil {
		t.Fatal("default temporary directory was not private", err)
	}
	if err := cleanup(); err != nil {
		t.Fatal("default owned cleanup failed", err)
	}
	fixture.assertPrevious(t)
	assertVerificationTemporaryAbsent(t, directory)
}

type verificationTemporaryFixture struct {
	t         *testing.T
	store     *store
	stage     string
	previous  string
	directory string
	events    []string
}

func newVerificationTemporaryFixture(t *testing.T, stage string) *verificationTemporaryFixture {
	t.Helper()
	s := repositoryRefusalFixture(t)
	previous := filepath.Join(s.repo.Directory, "previous-owned-temporary")
	if err := platform.EnsurePrivateDirectory(previous); err != nil {
		t.Fatal(err)
	}
	if err := platform.WriteProtected(filepath.Join(previous, "sentinel"), []byte("SYNTHETIC_PREVIOUS_SCOPE")); err != nil {
		t.Fatal(err)
	}
	s.temporary = previous
	return &verificationTemporaryFixture{t: t, store: s, stage: stage, previous: previous}
}

func (fixture *verificationTemporaryFixture) operations() verificationTemporaryOperations {
	return verificationTemporaryOperations{
		create: fixture.create,
		ensure: fixture.ensure,
		verify: fixture.verify,
		remove: fixture.remove,
	}
}

func (fixture *verificationTemporaryFixture) create(parent, pattern string) (string, error) {
	fixture.t.Helper()
	fixture.events = append(fixture.events, "create")
	if parent != fixture.store.repo.Directory || pattern != ".verify-tmp-" {
		fixture.t.Fatal("temporary creation escaped the fixed owned parent/prefix")
	}
	if fixture.stage == "create" {
		return "", errors.New("SYNTHETIC_PRIVATE_CREATE_FAILURE")
	}
	directory, err := os.MkdirTemp(parent, pattern)
	fixture.directory = directory
	return directory, err
}

func (fixture *verificationTemporaryFixture) ensure(path string) error {
	fixture.assertPath(path, "ensure")
	if fixture.stage == "ensure" {
		return errors.New("SYNTHETIC_PRIVATE_VALIDATION_FAILURE")
	}
	return platform.EnsurePrivateDirectory(path)
}

func (fixture *verificationTemporaryFixture) verify(path string) error {
	fixture.assertPath(path, "verify")
	if fixture.stage == "verify" {
		return fault.New(fault.Trust, "private directory permissions")
	}
	return protectedDirectory(path)
}

func (fixture *verificationTemporaryFixture) remove(path string) error {
	fixture.assertPath(path, "remove")
	if fixture.stage == "remove" {
		return errors.New("SYNTHETIC_PRIVATE_REMOVAL_FAILURE")
	}
	return os.RemoveAll(path)
}

func (fixture *verificationTemporaryFixture) assertPath(path, operation string) {
	fixture.t.Helper()
	fixture.events = append(fixture.events, operation)
	if path == "" || path != fixture.directory || filepath.Dir(path) != fixture.store.repo.Directory {
		fixture.t.Fatal("filesystem collaborator received a path outside its owned allocation")
	}
}

func (fixture *verificationTemporaryFixture) assertPrevious(t *testing.T) {
	t.Helper()
	if fixture.store.temporary != fixture.previous {
		t.Fatal("verification temporary did not restore the preceding scope")
	}
	data, err := os.ReadFile(filepath.Join(fixture.previous, "sentinel"))
	if err != nil || !bytes.Equal(data, []byte("SYNTHETIC_PREVIOUS_SCOPE")) {
		t.Fatal("verification lifetime changed the preceding owned data")
	}
}

func assertVerificationTemporaryAbsent(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Lstat(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("successful cleanup left its owned directory behind")
	}
}

func assertVerificationTemporaryError(t *testing.T, err error, operation string) {
	t.Helper()
	var typed *fault.Error
	if !errors.As(err, &typed) {
		t.Fatal("temporary failure was not a safe typed error")
	}
	if err != typed || typed.Kind != fault.Trust || typed.Operation != operation || err.Error() != "trust_failure: "+operation {
		t.Fatal("temporary failure changed its category or leaked filesystem details")
	}
}
