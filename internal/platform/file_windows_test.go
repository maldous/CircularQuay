//go:build windows

package platform

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/sys/windows"
)

func TestPrivateDirectoryDACLInheritsToOrdinaryChildren(t *testing.T) {
	parent := filepath.Join(t.TempDir(), "private")
	if err := EnsurePrivateDirectory(parent); err != nil {
		t.Fatal(err)
	}
	child := filepath.Join(parent, "child")
	if err := os.Mkdir(child, 0o700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(child, "values.json")
	if err := os.WriteFile(path, []byte("synthetic"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := EnsurePrivateDirectory(child); err != nil {
		t.Fatalf("inherited private directory rejected: %v", err)
	}
	if err := ValidateFileMetadata(path, true); err != nil {
		t.Fatalf("inherited private file rejected: %v", err)
	}
	for _, protectedPath := range []string{parent, child, path} {
		if !checkNativeSecurity(protectedPath, true) {
			t.Fatalf("native DACL permits an unexpected principal: %s", protectedPath)
		}
		if nativeDACLContainsEveryone(t, protectedPath) {
			t.Fatalf("native DACL grants Everyone: %s", protectedPath)
		}
	}
}

func TestFileGrantDoesNotCreateInheritableACEs(t *testing.T) {
	path := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(path, []byte("synthetic"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := protectFile(path); err != nil {
		t.Fatal(err)
	}
	assertFileACEsAreNotInheritable(t, path)
}

func assertFileACEsAreNotInheritable(t *testing.T, path string) {
	t.Helper()
	acl := fileDACL(t, path)
	for index := uint32(0); index < uint32(acl.AceCount); index++ {
		assertACEIsNotInheritable(t, acl, index)
	}
}

func fileDACL(t *testing.T, path string) *windows.ACL {
	t.Helper()
	sd, err := windows.GetNamedSecurityInfo(path, windows.SE_FILE_OBJECT, windows.DACL_SECURITY_INFORMATION)
	if err != nil {
		t.Fatal(err)
	}
	acl, _, err := sd.DACL()
	if err != nil || acl == nil {
		t.Fatalf("file DACL = %v", err)
	}
	return acl
}

func assertACEIsNotInheritable(t *testing.T, acl *windows.ACL, index uint32) {
	t.Helper()
	var ace *windows.ACCESS_ALLOWED_ACE
	if err := windows.GetAce(acl, index, &ace); err != nil || ace == nil {
		t.Fatalf("file ACE %d = %v", index, err)
	}
	if ace.Header.AceFlags&(windows.OBJECT_INHERIT_ACE|windows.CONTAINER_INHERIT_ACE) != 0 {
		t.Fatalf("file ACE %d is inheritable", index)
	}
}

func nativeDACLContainsEveryone(t *testing.T, path string) bool {
	t.Helper()
	sd, err := windows.GetNamedSecurityInfo(path, windows.SE_FILE_OBJECT, windows.DACL_SECURITY_INFORMATION)
	if err != nil {
		t.Fatal(err)
	}
	sddl := sd.String()
	return strings.Contains(sddl, ";;;WD)") || strings.Contains(sddl, ";;;S-1-1-0)")
}
