package platform

import (
	"context"
	"crypto/sha256"
	"errors"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/maldous/CircularQuay/internal/fault"
	"github.com/maldous/CircularQuay/internal/model"
)

func TestPlatformFileProcessBoundaryNULPathsDoNotCreateFiles(t *testing.T) {
	root := secureTempDir(t)
	path := filepath.Join(root, "invalid\x00name")
	for _, test := range []struct {
		name      string
		kind      fault.Kind
		operation string
	}{
		{"directory", fault.Trust, "inspect private directory"},
		{"receipt", fault.Apply, "open receipt log"},
		{"destination", fault.Apply, "inspect destination"},
		{"lock", fault.Apply, "open lock file"},
	} {
		t.Run(test.name, func(t *testing.T) {
			err := invokePlatformFileProcessNUL(t, test.name, path)
			assertPlatformFileProcessFault(t, err, test.kind, test.operation)
			entries, err := os.ReadDir(root)
			if err != nil || len(entries) != 0 {
				t.Fatal("invalid path created a file or staging entry")
			}
		})
	}
}

func invokePlatformFileProcessNUL(t *testing.T, name, path string) error {
	t.Helper()
	switch name {
	case "directory":
		return EnsurePrivateDirectory(path)
	case "receipt":
		return AppendProtected(path, []byte(`{"status":"synthetic"}`))
	case "destination":
		return WriteProtected(path, []byte("synthetic"))
	default:
		file, err := OpenLockFile(path)
		if file != nil {
			_ = file.Close()
			t.Fatal("invalid lock path returned a descriptor")
		}
		return err
	}
}

func TestPlatformFileProcessBoundaryRegularParentPreservesFile(t *testing.T) {
	parent, data, before := platformFileProcessFixture(t)
	path := filepath.Join(parent, "receipt")
	err := AppendProtected(path, []byte(`{"status":"synthetic"}`))
	assertPlatformFileProcessFault(t, err, fault.Apply, "create private directory")
	file, err := OpenLockFile(path)
	if file != nil {
		_ = file.Close()
		t.Fatal("regular parent returned a lock descriptor")
	}
	assertPlatformFileProcessFault(t, err, fault.Apply, "create private directory")
	assertPlatformFileProcessUnchanged(t, parent, data, before)
}

func TestPlatformFileProcessBoundaryRelativeMetadataAndMissingParentRefuse(t *testing.T) {
	assertPlatformFileProcessFault(t, ValidateProtectedPath("relative"), fault.Trust, "absolute protected path required")
	root := secureTempDir(t)
	path := filepath.Join(root, "absent", "file")
	file, err := openInParent(path, os.O_RDONLY, 0)
	if err == nil || file != nil {
		t.Fatal("missing anchored parent was opened or created")
	}
	if _, err := os.Lstat(filepath.Dir(path)); !os.IsNotExist(err) {
		t.Fatal("anchored open created an absent parent")
	}
}

func TestPlatformFileProcessBoundaryMissingStagePreservesDestination(t *testing.T) {
	path, data, before := platformFileProcessFixture(t)
	name, err := stageProtectedFile(filepath.Dir(path), []byte("new synthetic data"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(name); err != nil {
		t.Fatal(err)
	}
	// This is the internal availability boundary, not a fabricated scheduled
	// race in public WriteProtected or a claim that another actor removed it.
	err = activateStagedFile(name, path, filepath.Dir(path))
	assertPlatformFileProcessFault(t, err, fault.Apply, "activate protected file")
	assertPlatformFileProcessUnchanged(t, path, data, before)
}

func TestPlatformFileProcessBoundaryClosedStageDoesNotWrite(t *testing.T) {
	path, data, before := platformFileProcessFixture(t)
	file, err := os.OpenFile(path, os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	err = writeAndSync(file, []byte("must not be written"))
	assertPlatformFileProcessFault(t, err, fault.Apply, "protect staged file")
	assertPlatformFileProcessUnchanged(t, path, data, before)
}

func TestPlatformFileProcessBoundaryCanceledPublicRunNeverLaunches(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	code, err := Run(ctx, model.Workload{}, model.Generation{}, nil)
	if code != 0 || err != context.Canceled {
		t.Fatal("public run lost cancellation before command admission")
	}
}

func TestPlatformFileProcessBoundaryEnvironmentFailureWithholdsLaunchInputs(t *testing.T) {
	path, data, before := platformFileProcessFixture(t)
	work := model.Workload{Command: []string{path}, Environment: map[string]string{"BAD=KEY": "synthetic"}}
	args, environment, directory, err := launchInputs(work, model.Generation{}, nil)
	if args != nil || environment != nil || directory != "" {
		t.Fatal("invalid environment returned prepared launch inputs")
	}
	assertPlatformFileProcessFault(t, err, fault.Invalid, "environment collision or invalid value")
	assertPlatformFileProcessUnchanged(t, path, data, before)
}

func TestPlatformFileProcessBoundaryExecutableIdentityRefusesBeforeRead(t *testing.T) {
	missing := filepath.Join(secureTempDir(t), "missing")
	assertPlatformFileProcessFault(t, VerifyExecutable(missing, strings.Repeat("0", 64)), fault.Trust, "trusted executable permissions")
	assertPlatformFileProcessFault(t, VerifyExecutable(missing, "00"), fault.Trust, executableIdentityOperation)
	assertPlatformFileProcessFault(t, VerifyExecutable("relative", strings.Repeat("0", 64)), fault.Trust, executableIdentityOperation)
	if _, err := os.Lstat(missing); !os.IsNotExist(err) {
		t.Fatal("executable verification created its missing input")
	}
}

type platformFileProcessErrorReader struct{ err error }

func (reader platformFileProcessErrorReader) Read([]byte) (int, error) { return 0, reader.err }

func TestPlatformFileProcessBoundaryReadErrorHasNoDigest(t *testing.T) {
	failure := errors.New("synthetic bounded read failure")
	reader := io.MultiReader(strings.NewReader("abc"), platformFileProcessErrorReader{err: failure})
	digest, exceeded, err := boundedExecutableDigest(reader, 5)
	if digest != nil || exceeded || err != failure {
		t.Fatal("failed bounded read returned digest data or lost error identity")
	}
	digest, exceeded, err = boundedExecutableDigest(strings.NewReader("abc"), 3)
	want := sha256.Sum256([]byte("abc"))
	if err != nil || exceeded || !bytesEqual(digest, want[:]) {
		t.Fatal("bounded positive digest control changed")
	}
}

func TestPlatformFileProcessBoundaryClosedIdentityIsNotStable(t *testing.T) {
	path, data, before := platformFileProcessFixture(t)
	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = file.Close() })
	opened, err := file.Stat()
	if err != nil || !stableVerifiedExecutable(file, path, opened) {
		t.Fatal("real owned descriptor failed its stable positive control")
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	if stableVerifiedExecutable(file, path, opened) {
		t.Fatal("closed descriptor was accepted as a stable executable identity")
	}
	assertPlatformFileProcessUnchanged(t, path, data, before)
}

func TestPlatformFileProcessBoundaryDigestLengthAndReservedKeys(t *testing.T) {
	for _, test := range []struct {
		left, right []byte
		equal       bool
	}{
		{nil, []byte{1}, false},
		{[]byte{1}, []byte{2}, false},
		{[]byte{1}, []byte{1}, true},
		{nil, []byte{}, true},
	} {
		if bytesEqual(test.left, test.right) != test.equal {
			t.Fatal("pure byte comparison changed length or content semantics")
		}
	}
	for _, key := range []string{"PATH", "NODE_OPTIONS", "git_config_system"} {
		generation := model.Generation{Values: map[string]model.Value{key: {Data: "synthetic"}}}
		got, err := Environment(nil, generation)
		if got != nil {
			t.Fatal("reserved secret environment key produced a process environment")
		}
		assertPlatformFileProcessFault(t, err, fault.Trust, "bootstrap environment denied")
	}
	got, err := Environment(nil, model.Generation{Values: map[string]model.Value{"ORDINARY": {Data: "synthetic"}}})
	if err != nil || !containsPlatformFileProcessEntry(got, "ORDINARY=synthetic") {
		t.Fatal("ordinary owned environment value failed its positive control")
	}
}

func containsPlatformFileProcessEntry(values []string, wanted string) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}

func platformFileProcessFixture(t *testing.T) (string, []byte, os.FileInfo) {
	t.Helper()
	path := filepath.Join(secureTempDir(t), "owned-fixture")
	data := []byte("owned synthetic file bytes")
	if err := WriteProtected(path, data); err != nil {
		t.Fatal(err)
	}
	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	info, statErr := file.Stat()
	closeErr := file.Close()
	if statErr != nil || closeErr != nil {
		t.Fatal("owned fixture identity could not be captured eagerly")
	}
	return path, data, info
}

func assertPlatformFileProcessUnchanged(t *testing.T, path string, data []byte, before os.FileInfo) {
	t.Helper()
	got, err := os.ReadFile(path)
	if err != nil || !reflect.DeepEqual(got, data) {
		t.Fatal("refused operation changed the original owned bytes")
	}
	after, err := os.Stat(path)
	if err != nil || !os.SameFile(before, after) || after.Mode() != before.Mode() {
		t.Fatal("refused operation replaced or re-permissioned the owned fixture")
	}
}

func assertPlatformFileProcessFault(t *testing.T, err error, kind fault.Kind, operation string) {
	t.Helper()
	if !fault.Is(err, kind) || err.Error() != string(kind)+": "+operation || errors.Unwrap(err) != nil {
		t.Fatal("file/process refusal lost its exact safe error contract")
	}
}
