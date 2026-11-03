package state

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/maldous/CircularQuay/internal/fault"
	"github.com/maldous/CircularQuay/internal/platform"
)

func privateStateDirectory(t *testing.T) string {
	t.Helper()
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	directory := filepath.Join(root, "state")
	if err := platform.EnsurePrivateDirectory(directory); err != nil {
		t.Fatal(err)
	}
	return directory
}

func TestAtomicMetadataAndProcessLock(t *testing.T) {
	s := Store{Directory: privateStateDirectory(t)}
	requireState(t, s, "not_provisioned")
	if err := s.Write(Record{Workload: "demo", State: "ready", Mode: "offline"}); err != nil {
		t.Fatal(err)
	}
	requireState(t, s, "ready")
	unlock := requireLock(t, s)
	if _, err := s.Lock("demo"); !fault.Is(err, fault.Unavailable) {
		t.Fatal("concurrent reconciliation allowed")
	}
	requireUnlock(t, unlock)
	requireUnlock(t, requireLock(t, s))
}

func requireState(t *testing.T, store Store, want string) {
	t.Helper()
	record, err := store.Read("demo")
	if err != nil {
		t.Fatal(err)
	}
	if record.State != want {
		t.Fatalf("state = %q, want %q", record.State, want)
	}
}

func requireLock(t *testing.T, store Store) func() error {
	t.Helper()
	release, err := store.Lock("demo")
	if err != nil {
		t.Fatal(err)
	}
	return release
}

func requireUnlock(t *testing.T, release func() error) {
	t.Helper()
	if err := release(); err != nil {
		t.Fatal(err)
	}
}

func TestInvalidAndCorruptState(t *testing.T) {
	s := Store{Directory: privateStateDirectory(t)}
	for _, name := range []string{"../escape", "", strings.Repeat("x", 65)} {
		if _, err := s.Read(name); !fault.Is(err, fault.Invalid) {
			t.Fatal("invalid name accepted")
		}
	}
	if _, err := (Store{}).Read("demo"); !fault.Is(err, fault.Invalid) {
		t.Fatal("empty state directory")
	}
	p := filepath.Join(s.Directory, "demo.json")
	for _, body := range []string{"not-json", `{"workload":"other"}`, `{"workload":"demo","extra":"field"}`} {
		writeProtectedStateFixture(t, p, []byte(body))
		if _, err := s.Read("demo"); !fault.Is(err, fault.Trust) {
			t.Fatal("corrupt state accepted")
		}
	}
	writeProtectedStateFixture(t, p, make([]byte, (1<<20)+1))
	if _, err := s.Read("demo"); !fault.Is(err, fault.Trust) {
		t.Fatal("oversized state accepted")
	}
}

func TestUnsafeStateDirectory(t *testing.T) {
	d := t.TempDir()
	p := filepath.Join(d, "file")
	if err := os.WriteFile(p, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := (Store{Directory: p}).Read("demo"); err == nil {
		t.Fatal("non-directory accepted")
	}
	if runtime.GOOS != "windows" {
		if err := os.Chmod(d, 0o777); err != nil {
			t.Fatal(err)
		}
		if _, err := (Store{Directory: d}).Read("demo"); !fault.Is(err, fault.Trust) {
			t.Fatal("writable state directory accepted")
		}
	}
}

func TestStateRejectsTrailingJSON(t *testing.T) {
	s := Store{Directory: privateStateDirectory(t)}
	p := filepath.Join(s.Directory, "demo.json")
	for _, body := range []string{`{"workload":"demo"} {}`, `{"workload":"demo"} trailing`, `{"workload":"demo","workload":"demo"}`} {
		writeProtectedStateFixture(t, p, []byte(body))
		if _, err := s.Read("demo"); !fault.Is(err, fault.Trust) {
			t.Errorf("ambiguous metadata accepted: %q", body)
		}
	}
}

func writeProtectedStateFixture(t *testing.T, path string, data []byte) {
	t.Helper()
	if err := platform.WriteProtected(path, data); err != nil {
		t.Fatal(err)
	}
}

func TestStateRejectsSymlinkFiles(t *testing.T) {
	s := Store{Directory: privateStateDirectory(t)}
	target := filepath.Join(t.TempDir(), "target")
	if err := os.WriteFile(target, []byte(`{"workload":"demo"}`), 0600); err != nil {
		t.Fatal(err)
	}
	for _, suffix := range []string{".json", ".json.lock"} {
		p := filepath.Join(s.Directory, "demo"+suffix)
		if err := os.Symlink(target, p); err != nil {
			t.Skipf("symlink unavailable: %v", err)
		}
		if suffix == ".json" {
			if _, err := s.Read("demo"); !fault.Is(err, fault.Trust) {
				t.Error("symlink state read accepted")
			}
			if err := s.Write(Record{Workload: "demo"}); !fault.Is(err, fault.Trust) {
				t.Error("symlink state destination accepted")
			}
		} else {
			release, err := s.Lock("demo")
			if err == nil {
				_ = release()
				t.Error("symlink lock accepted")
			}
		}
	}
}
