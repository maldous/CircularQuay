//go:build windows

package platform

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"unsafe"

	"golang.org/x/sys/windows"
)

func TestDescriptorSecurityRequiresTrustedOwnersAndReplacementSafeAncestors(t *testing.T) {
	user := currentTokenSID(t)
	for _, test := range []struct {
		name, owner, aces string
		ancestor          bool
		want              bool
	}{
		{"current owner", user.String(), privateTestACEs(user.String()), false, true},
		{"trusted installer owner", trustedInstallerSID, privateTestACEs(user.String()), false, true},
		{"trusted installer writer", user.String(), privateTestACEs(user.String()) + "(A;;FA;;;" + trustedInstallerSID + ")", false, true},
		{"foreign owner", "S-1-5-21-1-2-3-1001", privateTestACEs(user.String()), false, false},
		{"unrelated service writer", user.String(), privateTestACEs(user.String()) + "(A;;FA;;;S-1-5-80-1-2-3-4-5)", false, false},
		{"untrusted delete child", user.String(), privateTestACEs(user.String()) + "(A;;0x40;;;WD)", true, false},
		{"untrusted directory creation", user.String(), privateTestACEs(user.String()) + "(A;;GW;;;WD)", true, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := descriptorSecure(testSecurityDescriptor(t, test.owner, test.aces), user, true, test.ancestor); got != test.want {
				t.Fatalf("descriptorSecure() = %t, want %t", got, test.want)
			}
		})
	}
}

func TestServiceSIDTrustRequiresCurrentTokenMembership(t *testing.T) {
	user := currentTokenSID(t)
	service, err := windows.StringToSid("S-1-5-80-1-2-3-4-5")
	if err != nil {
		t.Fatal(err)
	}
	nonService, err := windows.StringToSid("S-1-5-21-1-2-3-1001")
	if err != nil {
		t.Fatal(err)
	}
	if trustedOwnerSID(service, user) {
		t.Fatal("arbitrary service SID trusted as an owner")
	}
	if serviceSIDMember(nonService, func(*windows.SID) (bool, error) { return true, nil }) {
		t.Fatal("non-service SID accepted from membership callback")
	}
	if serviceSIDMember(service, func(*windows.SID) (bool, error) { return false, nil }) {
		t.Fatal("unrelated service SID accepted without token membership")
	}
	if serviceSIDMember(service, func(*windows.SID) (bool, error) { return false, errors.New("membership unavailable") }) {
		t.Fatal("membership error accepted")
	}
	if !serviceSIDMember(service, func(*windows.SID) (bool, error) { return true, nil }) {
		t.Fatal("active service SID rejected")
	}
}

func TestProtectedFileRejectsUnsafeParentDeleteChild(t *testing.T) {
	parent := filepath.Join(t.TempDir(), "private")
	if err := EnsurePrivateDirectory(parent); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(parent, "values.json")
	if err := WriteProtected(path, []byte("synthetic")); err != nil {
		t.Fatal(err)
	}
	if err := ValidateFileMetadata(path, true); err != nil {
		t.Fatalf("trusted protected child rejected: %v", err)
	}
	setTestDACL(t, parent, privateTestACEs(currentTokenSID(t).String())+"(A;;0x40;;;WD)")
	if err := ValidateFileMetadata(path, true); err == nil {
		t.Fatal("parent FILE_DELETE_CHILD grant accepted for protected child")
	}
}

func TestNativeSecurityRejectsReparseComponentBeforeACLTraversal(t *testing.T) {
	directory := filepath.Join(t.TempDir(), "private")
	if err := EnsurePrivateDirectory(directory); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(directory, "target")
	if err := EnsurePrivateDirectory(target); err != nil {
		t.Fatal(err)
	}
	if err := WriteProtected(filepath.Join(target, "synthetic"), []byte("synthetic")); err != nil {
		t.Fatal(err)
	}
	junction := filepath.Join(directory, "target-junction")
	if err := createTestJunction(junction, target); err != nil {
		t.Fatal(err)
	}
	if checkNativeSecurity(junction, false) {
		t.Fatal("final reparse-point path accepted")
	}
	if checkNativeSecurity(filepath.Join(junction, "synthetic"), false) {
		t.Fatal("ancestor reparse-point path accepted")
	}
}

type testReparseDataBuffer struct {
	header testReparseDataBufferHeader
	detail [windows.MAXIMUM_REPARSE_DATA_BUFFER_SIZE]byte
}

type testReparseDataBufferHeader struct {
	tag    uint32
	length uint16
	unused uint16
}

type testMountPointReparseBuffer struct {
	substituteOffset uint16
	substituteLength uint16
	printOffset      uint16
	printLength      uint16
	path             [1]uint16
}

// createTestJunction creates a test-owned mount-point reparse point directly,
// avoiding os.Symlink's developer-mode or elevated privilege requirement.
func createTestJunction(link, target string) error {
	if err := os.Mkdir(link, 0o700); err != nil {
		return err
	}
	linkName, err := windows.UTF16PtrFromString(link)
	if err != nil {
		return err
	}
	handle, err := windows.CreateFile(linkName, windows.GENERIC_WRITE, 0, nil, windows.OPEN_EXISTING, windows.FILE_FLAG_OPEN_REPARSE_POINT|windows.FILE_FLAG_BACKUP_SEMANTICS, 0)
	if err != nil {
		return err
	}
	defer windows.CloseHandle(handle)

	data, err := testMountPointData(target)
	if err != nil {
		return err
	}
	var reparse testReparseDataBuffer
	reparse.header.tag = windows.IO_REPARSE_TAG_MOUNT_POINT
	reparse.header.length = uint16(len(data))
	copy(reparse.detail[:], data)
	var returned uint32
	return windows.DeviceIoControl(handle, windows.FSCTL_SET_REPARSE_POINT, (*byte)(unsafe.Pointer(&reparse.header)), uint32(unsafe.Sizeof(reparse.header))+uint32(reparse.header.length), nil, 0, &returned, nil)
}

func testMountPointData(target string) ([]byte, error) {
	substitute, err := windows.UTF16FromString(`\??\` + target)
	if err != nil {
		return nil, err
	}
	printName, err := windows.UTF16FromString(target)
	if err != nil {
		return nil, err
	}
	// The mount-point buffer retains each UTF-16 terminator. The name-length
	// fields below deliberately exclude them, as required by the native layout.
	path := append(substitute, printName...)
	length := int(unsafe.Offsetof(testMountPointReparseBuffer{}.path)) + len(path)*2
	data := make([]byte, length)
	buffer := (*testMountPointReparseBuffer)(unsafe.Pointer(&data[0]))
	buffer.substituteLength = uint16((len(substitute) - 1) * 2)
	buffer.printOffset = uint16(len(substitute) * 2)
	buffer.printLength = uint16((len(printName) - 1) * 2)
	copy(unsafe.Slice(&buffer.path[0], len(path)), path)
	return data, nil
}

func currentTokenSID(t *testing.T) *windows.SID {
	t.Helper()
	user, err := windows.GetCurrentProcessToken().GetTokenUser()
	if err != nil {
		t.Fatal(err)
	}
	return user.User.Sid
}

func privateTestACEs(user string) string {
	return "(A;;FA;;;SY)(A;;FA;;;BA)(A;;FA;;;" + user + ")"
}

func testSecurityDescriptor(t *testing.T, owner, aces string) *windows.SECURITY_DESCRIPTOR {
	t.Helper()
	sd, err := windows.SecurityDescriptorFromString("O:" + owner + "D:P" + aces)
	if err != nil {
		t.Fatal(err)
	}
	return sd
}

func setTestDACL(t *testing.T, path, aces string) {
	t.Helper()
	sd := testSecurityDescriptor(t, currentTokenSID(t).String(), aces)
	dacl, _, err := sd.DACL()
	if err != nil {
		t.Fatal(err)
	}
	if err = windows.SetNamedSecurityInfo(path, windows.SE_FILE_OBJECT, windows.DACL_SECURITY_INFORMATION|windows.PROTECTED_DACL_SECURITY_INFORMATION, nil, nil, dacl, nil); err != nil {
		t.Fatal(err)
	}
}
