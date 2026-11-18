//go:build windows

package platform

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/maldous/CircularQuay/internal/fault"
	"golang.org/x/sys/windows"
)

func TestCreateProtectedExclusiveAcceptsReadableWindowsParent(t *testing.T) {
	parent := filepath.Join(t.TempDir(), "readable")
	if err := os.Mkdir(parent, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := setWindowsTestDACL(parent, "FR"); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(parent, "credential.json")
	if err := CreateProtectedExclusive(path, []byte("synthetic")); err != nil {
		t.Fatal(err)
	}
	if err := ValidateFileMetadata(path, true); err != nil {
		t.Fatalf("created file was not privately protected: %v", err)
	}
	if nativeDACLContainsEveryone(t, path) {
		t.Fatal("created file inherited Everyone access")
	}
}

func TestCreateProtectedExclusiveRejectsWritableWindowsParent(t *testing.T) {
	parent := filepath.Join(t.TempDir(), "writable")
	if err := os.Mkdir(parent, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := setWindowsTestDACL(parent, "FA"); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(parent, "credential.json")
	if err := CreateProtectedExclusive(path, []byte("synthetic")); !fault.Is(err, fault.Trust) {
		t.Fatalf("writable parent error = %v", err)
	}
	if _, err := os.Lstat(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("rejected parent created a file: %v", err)
	}
}

func setWindowsTestDACL(path, everyoneRights string) error {
	user, err := windows.GetCurrentProcessToken().GetTokenUser()
	if err != nil {
		return err
	}
	sddl := "D:P(A;;FA;;;SY)(A;;FA;;;BA)(A;;FA;;;" + user.User.Sid.String() + ")(A;;" + everyoneRights + ";;;WD)"
	sd, err := windows.SecurityDescriptorFromString(sddl)
	if err != nil {
		return err
	}
	dacl, _, err := sd.DACL()
	if err != nil {
		return err
	}
	return windows.SetNamedSecurityInfo(path, windows.SE_FILE_OBJECT, windows.DACL_SECURITY_INFORMATION|windows.PROTECTED_DACL_SECURITY_INFORMATION, nil, nil, dacl, nil)
}
