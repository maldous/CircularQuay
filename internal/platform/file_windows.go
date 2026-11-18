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

const (
	systemSID           = "S-1-5-18"
	administratorsSID   = "S-1-5-32-544"
	trustedInstallerSID = "S-1-5-80-956008885-3418522649-1831038044-1853292631-2271478464"
)

// checkNativeSecurity rejects an untrusted owner or a DACL that lets an
// untrusted principal replace the protected object through any ancestor.
// TrustedInstaller is deliberately an exact SID: it is the Windows component
// servicing principal, not a general NT SERVICE allowance.
func checkNativeSecurity(path string, secret bool) bool {
	// Name-based security descriptor queries do not establish that the lexical
	// path reaches the reviewed object. Reject reparse components first while
	// retaining case and short-name aliases accepted by resolvedPathIsOriginal.
	if !resolvedPathIsOriginal(path) {
		return false
	}
	user, err := windows.GetCurrentProcessToken().GetTokenUser()
	if err != nil {
		return false
	}
	for current, ancestor := filepath.Clean(path), false; ; ancestor = true {
		if !securityDescriptorSecure(current, user.User.Sid, secret, ancestor) {
			return false
		}
		parent := filepath.Dir(current)
		if parent == current {
			return true
		}
		current = parent
	}
}

func securityDescriptorSecure(path string, user *windows.SID, secret, ancestor bool) bool {
	sd, err := windows.GetNamedSecurityInfo(path, windows.SE_FILE_OBJECT, windows.OWNER_SECURITY_INFORMATION|windows.DACL_SECURITY_INFORMATION)
	if err != nil || sd == nil {
		return false
	}
	return descriptorSecure(sd, user, secret, ancestor)
}

func descriptorSecure(sd *windows.SECURITY_DESCRIPTOR, user *windows.SID, secret, ancestor bool) bool {
	owner, _, err := sd.Owner()
	if err != nil || !trustedOwnerSID(owner, user) {
		return false
	}
	acl, _, err := sd.DACL()
	if err != nil || acl == nil {
		return false
	}
	for i := uint32(0); i < uint32(acl.AceCount); i++ {
		if !allowedACE(acl, i, user, secret, ancestor) {
			return false
		}
	}
	return true
}

func allowedACE(acl *windows.ACL, index uint32, user *windows.SID, secret, ancestor bool) bool {
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
	if ace.Header.AceFlags&windows.INHERIT_ONLY_ACE != 0 {
		return true
	}
	if trustedACEPrincipal(aceSID(ace), user) {
		return true
	}
	if ancestor {
		return !replacesChild(ace.Mask)
	}
	return !secret && !writesFile(ace.Mask)
}

func aceSID(ace *windows.ACCESS_ALLOWED_ACE) *windows.SID {
	return (*windows.SID)(unsafe.Pointer(&ace.SidStart))
}

// trustedOwnerSID intentionally does not include arbitrary service SIDs: an
// owner can rewrite its DACL. TrustedInstaller is an exact OS principal.
func trustedOwnerSID(sid, user *windows.SID) bool {
	return sid != nil && (sid.Equals(user) || trustedSystemSID(sid.String()))
}

func trustedACEPrincipal(sid, user *windows.SID) bool {
	return trustedOwnerSID(sid, user) || activeServiceSID(sid)
}

func trustedSystemSID(value string) bool {
	return value == systemSID || value == administratorsSID || value == trustedInstallerSID
}

func activeServiceSID(sid *windows.SID) bool {
	return serviceSIDMember(sid, windows.GetCurrentProcessToken().IsMember)
}

func serviceSIDMember(sid *windows.SID, isMember func(*windows.SID) (bool, error)) bool {
	if sid == nil || !isServiceSID(sid.String()) {
		return false
	}
	member, err := isMember(sid)
	return err == nil && member
}

func writesFile(mask windows.ACCESS_MASK) bool {
	// FILE_DELETE_CHILD is not exported by the pinned x/sys version.
	// WinNT.h: https://learn.microsoft.com/en-us/windows/win32/fileio/file-access-rights-constants
	const fileDeleteChild = 0x0040
	const unsafeWrite = windows.FILE_WRITE_DATA | windows.FILE_APPEND_DATA | windows.FILE_WRITE_EA | fileDeleteChild | windows.FILE_WRITE_ATTRIBUTES | windows.DELETE | windows.WRITE_DAC | windows.WRITE_OWNER | windows.GENERIC_WRITE | windows.GENERIC_ALL
	return mask&unsafeWrite != 0
}

func replacesChild(mask windows.ACCESS_MASK) bool {
	// FILE_DELETE_CHILD is directory-specific. Generic write intentionally does
	// not appear here: on a directory it grants creation/metadata rights, not
	// deletion of an existing protected child or its security descriptor.
	const fileDeleteChild = 0x0040
	const replacementRights = fileDeleteChild | windows.DELETE | windows.WRITE_DAC | windows.WRITE_OWNER | windows.GENERIC_ALL
	return mask&replacementRights != 0
}
