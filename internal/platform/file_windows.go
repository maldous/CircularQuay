//go:build windows

package platform

import (
	"os"
	"path/filepath"
	"strings"
	"unsafe"

	"golang.org/x/sys/windows"
)

func secureMode(_ os.FileMode, _ bool) bool { return true }

// resolvedPathIsOriginal rejects every Windows reparse point in an existing
// protected path. EvalSymlinks also rewrites harmless case and short-name
// aliases, so textual comparison would reject a secure path on Windows.
func resolvedPathIsOriginal(path string) bool {
	cleaned := filepath.Clean(path)
	volume := filepath.VolumeName(cleaned)
	if volume == "" {
		return false
	}
	current := volume + `\`
	components := strings.Split(strings.TrimPrefix(cleaned[len(volume):], `\`), `\`)
	for _, component := range components {
		if component == "" {
			continue
		}
		current = filepath.Join(current, component)
		name, err := windows.UTF16PtrFromString(current)
		if err != nil {
			return false
		}
		attributes, err := windows.GetFileAttributes(name)
		if err != nil || attributes&windows.FILE_ATTRIBUTE_REPARSE_POINT != 0 {
			return false
		}
	}
	return true
}

func replaceFile(from, to string) error {
	f, err := windows.UTF16PtrFromString(from)
	if err != nil {
		return err
	}
	t, err := windows.UTF16PtrFromString(to)
	if err != nil {
		return err
	}
	return windows.MoveFileEx(f, t, windows.MOVEFILE_REPLACE_EXISTING|windows.MOVEFILE_WRITE_THROUGH)
}

func syncDirectory(_ string) error { return nil }

func protectFile(path string) error { return grantFile(path, "") }

func protectDirectory(path string) error { return grantDirectory(path) }

func grantFile(path, sid string) error {
	return grant(path, sid, "")
}

func grantDirectory(path string) error {
	// OI and CI propagate the private DACL to ordinary files and directories
	// subsequently created beneath this directory. File and service-SID grants
	// remain non-inheritable and therefore cannot widen child access.
	return grant(path, "", "OICI")
}

func grant(path, sid, inheritance string) error {
	user, err := windows.GetCurrentProcessToken().GetTokenUser()
	if err != nil {
		return err
	}
	ace := "(A;" + inheritance + ";FA;;;"
	sddl := "D:P" + ace + "SY)" + ace + "BA)" + ace + user.User.Sid.String() + ")"
	if sid != "" {
		sddl += "(A;;FR;;;" + sid + ")"
	}
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

// checkNativeSecurity rejects secret access by principals other than this
// process's identities, SYSTEM or Administrators. Public ciphertext may be read
// broadly, but must remain protected against unauthorized replacement. The
// unsafe conversion is limited to native ACEs returned by x/sys.
func checkNativeSecurity(path string, secret bool) bool {
	sd, err := windows.GetNamedSecurityInfo(path, windows.SE_FILE_OBJECT, windows.DACL_SECURITY_INFORMATION)
	if err != nil || sd == nil {
		return false
	}
	acl, _, err := sd.DACL()
	if err != nil || acl == nil {
		return false
	}
	user, err := windows.GetCurrentProcessToken().GetTokenUser()
	if err != nil {
		return false
	}
	for i := uint32(0); i < uint32(acl.AceCount); i++ {
		if !allowedACE(acl, i, user.User.Sid, secret) {
			return false
		}
	}
	return true
}

func allowedACE(acl *windows.ACL, index uint32, user *windows.SID, secret bool) bool {
	var ace *windows.ACCESS_ALLOWED_ACE
	if windows.GetAce(acl, index, &ace) != nil || ace == nil {
		return false
	}
	if ace.Header.AceType == windows.ACCESS_DENIED_ACE_TYPE {
		return true
	}
	if ace.Header.AceType != windows.ACCESS_ALLOWED_ACE_TYPE {
		return false
	}
	if !secret && !writesFile(ace.Mask) {
		return true
	}
	return allowedPrincipal(ace, user)
}

func allowedPrincipal(ace *windows.ACCESS_ALLOWED_ACE, user *windows.SID) bool {
	sid := (*windows.SID)(unsafe.Pointer(&ace.SidStart))
	text := sid.String()
	if text == "S-1-5-18" || text == "S-1-5-32-544" || sid.Equals(user) {
		return true
	}
	member, err := windows.GetCurrentProcessToken().IsMember(sid)
	return err == nil && member && isServiceSID(text)
}

func writesFile(mask windows.ACCESS_MASK) bool {
	// FILE_DELETE_CHILD is not exported by the pinned x/sys version.
	// WinNT.h: https://learn.microsoft.com/en-us/windows/win32/fileio/file-access-rights-constants
	const fileDeleteChild = 0x0040
	const unsafeWrite = windows.FILE_WRITE_DATA | windows.FILE_APPEND_DATA | windows.FILE_WRITE_EA | fileDeleteChild | windows.FILE_WRITE_ATTRIBUTES | windows.DELETE | windows.WRITE_DAC | windows.WRITE_OWNER | windows.GENERIC_WRITE | windows.GENERIC_ALL
	return mask&unsafeWrite != 0
}
