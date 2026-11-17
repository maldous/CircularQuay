//go:build linux

package quota

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"unsafe"

	"golang.org/x/sys/unix"
)

func TestNativeBoundaryContractIdentityPrecedesFileCapability(t *testing.T) {
	sentinel := errors.New("synthetic identity refusal")
	err := validateUnprivilegedExecution("unused", func() error { return sentinel }, func(string) ([]byte, error) {
		t.Fatal("identity refusal reached capability lookup")
		return nil, nil
	})
	if !errors.Is(err, sentinel) {
		t.Fatal("identity error was not retained")
	}
	err = validateUnprivilegedExecution(filepath.Join(t.TempDir(), "missing"), func() error { return nil }, func(string) ([]byte, error) {
		t.Fatal("missing file reached capability lookup")
		return nil, nil
	})
	if !errors.Is(err, ErrPrivileged) {
		t.Fatal("missing helper was admitted")
	}
}

func TestNativeBoundaryContractCapabilityReadIsFixedAndFailClosed(t *testing.T) {
	sentinel := errors.New("synthetic proc refusal")
	if got := readCapabilityStatusWith(func(path string) ([]byte, error) {
		if path != "/proc/self/status" {
			t.Fatal("capability metadata path expanded")
		}
		return []byte("not admitted"), sentinel
	}); got != nil {
		t.Fatal("failed capability read retained partial metadata")
	}
	want := []byte("synthetic bounded metadata")
	got := readCapabilityStatusWith(func(string) ([]byte, error) { return want, nil })
	if !bytes.Equal(got, want) {
		t.Fatal("successful capability metadata was lost")
	}
}

func TestNativeBoundaryContractXattrErrorAbsenceAndValueRemainDistinct(t *testing.T) {
	for _, result := range []string{"absent", "error", "value"} {
		t.Run(result, func(t *testing.T) { nativeBoundaryXattr(t, result) })
	}
}

func nativeBoundaryXattr(t *testing.T, result string) {
	t.Helper()
	sentinel := errors.New("synthetic xattr refusal")
	got, err := fileCapabilityWith("owned-fixture", nativeBoundaryXattrReader(t, result, sentinel))
	if result == "value" {
		if err != nil || !bytes.Equal(got, []byte{1, 2, 3}) {
			t.Fatal("successful capability bytes were not bounded to returned size")
		}
		return
	}
	if got != nil || (result == "error" && !errors.Is(err, sentinel)) || (result == "absent" && err != nil) {
		t.Fatal("capability absence and error were conflated")
	}
}

func nativeBoundaryXattrReader(t *testing.T, result string, sentinel error) func(string, string, []byte) (int, error) {
	t.Helper()
	return func(path, name string, destination []byte) (int, error) {
		if path != "owned-fixture" || name != "security.capability" || len(destination) != 64 {
			t.Fatal("capability lookup changed its fixed attribute or bound")
		}
		switch result {
		case "absent":
			return 0, unix.ENODATA
		case "error":
			return 0, sentinel
		default:
			copy(destination, []byte{1, 2, 3})
			return 3, nil
		}
	}
}

func TestNativeBoundaryContractQuotaABIAndErrnoWithoutSyscall(t *testing.T) {
	for _, errno := range []syscall.Errno{0, syscall.EPERM} {
		var metadata quotaStatV
		calls := 0
		invoke := func(directory, command, id int, address unsafe.Pointer) syscall.Errno {
			calls++
			nativeBoundaryABIArguments(t, directory, command, id)
			if address != unsafe.Pointer(&metadata) {
				t.Fatal("quota ABI metadata address changed")
			}
			return errno
		}
		err := quotaControlWith(17, qXGetQStatV, 23, unsafe.Pointer(&metadata), invoke)
		if calls != 1 || (errno == 0 && err != nil) || (errno != 0 && !errors.Is(err, errno)) {
			t.Fatal("quota errno mapping did not retain success/refusal distinction")
		}
	}
}

func nativeBoundaryABIArguments(t *testing.T, directory, command, id int) {
	t.Helper()
	if directory != 17 || command != qXGetQStatV || id != 23 {
		t.Fatal("quota ABI invocation changed")
	}
}

func TestNativeBoundaryContractOwnedHelperPositiveControl(t *testing.T) {
	path := filepath.Join(t.TempDir(), "inert-helper")
	if err := os.WriteFile(path, []byte("not executed"), 0600); err != nil {
		t.Fatal(err)
	}
	err := validateUnprivilegedExecution(path, func() error { return nil }, func(got string) ([]byte, error) {
		if got != path {
			t.Fatal("helper identity changed")
		}
		return nil, nil
	})
	if err != nil {
		t.Fatal("in-process identity/capability control refused an owned regular file")
	}
}
