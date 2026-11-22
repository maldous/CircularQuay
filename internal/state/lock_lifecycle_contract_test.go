package state

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/maldous/CircularQuay/internal/fault"
)

func TestLockLifecycleContractPathResolutionRefusesBeforeCreation(t *testing.T) {
	root := t.TempDir()
	s := Store{Directory: filepath.Join(root, "never-created")}
	path, err := s.pathWithResolver("workload", func(value string) (string, error) {
		if value != s.Directory {
			t.Fatal("state resolver received another directory")
		}
		return "", errors.New("synthetic cwd failure")
	})
	if path != "" || !fault.Is(err, fault.Invalid) {
		t.Fatal("unresolvable directory produced a state path")
	}
	if _, err := os.Lstat(s.Directory); !os.IsNotExist(err) {
		t.Fatal("path resolution failure created state")
	}
}

func TestLockLifecycleContractAcquisitionFailureClosesExactlyOnce(t *testing.T) {
	for _, cleanupError := range []bool{false, true} {
		t.Run(map[bool]string{false: "closed", true: "cleanup-error"}[cleanupError], func(t *testing.T) { lockLifecycleRefusal(t, cleanupError) })
	}
}

func lockLifecycleRefusal(t *testing.T, cleanupError bool) {
	t.Helper()
	file, err := os.CreateTemp(t.TempDir(), "owned-lock-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = file.Close() })
	closed := 0
	release, err := lockOpenedFile(file, func(*os.File) error { return errors.New("synthetic busy") }, func(*os.File) error {
		t.Fatal("failed acquisition reached unlock")
		return nil
	}, func(got *os.File) error {
		closed++
		if got != file {
			t.Fatal("cleanup closed a different descriptor")
		}
		if err := got.Close(); err != nil {
			t.Fatal(err)
		}
		if cleanupError {
			return errors.New("synthetic cleanup failure")
		}
		return nil
	})
	want := fault.Unavailable
	if cleanupError {
		want = fault.Apply
	}
	if release != nil || !fault.Is(err, want) || closed != 1 {
		t.Fatal("failed acquisition did not preserve exact cleanup/error semantics")
	}
}
