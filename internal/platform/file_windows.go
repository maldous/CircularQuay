//go:build windows

package platform

import (
	"os"
	"unsafe"

	"golang.org/x/sys/windows"
)

func secureMode(_ os.FileMode, _ bool) bool { return true }

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

func protectDirectory(path string) error { return grantFile(path, "") }

func grantFile(path, sid string) error {
	user, err := windows.GetCurrentProcessToken().GetTokenUser()
	if err != nil {
		return err
	}
	sddl := "D:P(A;;FA;;;SY)(A;;FA;;;BA)(A;;FA;;;" + user.User.Sid.String() + ")"
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
		var ace *windows.ACCESS_ALLOWED_ACE
		if windows.GetAce(acl, i, &ace) != nil || ace == nil {
			return false
		}
		if ace.Header.AceType == windows.ACCESS_DENIED_ACE_TYPE {
			continue
		}
		if ace.Header.AceType != windows.ACCESS_ALLOWED_ACE_TYPE {
			return false
		}
		const unsafeWrite = windows.FILE_WRITE_DATA | windows.FILE_APPEND_DATA | windows.FILE_WRITE_EA | windows.FILE_DELETE_CHILD | windows.FILE_WRITE_ATTRIBUTES | windows.DELETE | windows.WRITE_DAC | windows.WRITE_OWNER | windows.GENERIC_WRITE | windows.GENERIC_ALL
		if !secret && ace.Mask&unsafeWrite == 0 {
			continue
		}
		sid := (*windows.SID)(unsafe.Pointer(&ace.SidStart))
		text := sid.String()
		if text == "S-1-5-18" || text == "S-1-5-32-544" || sid.Equals(user.User.Sid) {
			continue
		}
		member, memberErr := windows.GetCurrentProcessToken().IsMember(sid)
		if memberErr != nil || !member || !isServiceSID(text) {
			return false
		}
	}
	return true
}
