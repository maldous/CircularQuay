//go:build !windows

package platform

import (
	"os"
	"path/filepath"
	"syscall"
)

// checkNativeSecurity requires an executable or protected file to be owned by
// root or this process and to sit below an immutable ownership chain. A
// root-owned sticky directory such as /tmp is safe for an owner-only child:
// another user cannot replace that child. All other writable ancestors fail.
func checkNativeSecurity(path string, secret bool) bool {
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		return false
	}
	info, err := os.Stat(resolved)
	if err != nil || !trustedEntry(info, secret) {
		return false
	}
	return trustedAncestors(filepath.Dir(resolved))
}

func trustedEntry(info os.FileInfo, secret bool) bool {
	if info.IsDir() {
		return trustedDirectory(info)
	}
	return trustedFile(info, secret)
}

func trustedAncestors(path string) bool {
	for {
		info, err := os.Stat(path)
		if err != nil || !trustedDirectory(info) {
			return false
		}
		parent := filepath.Dir(path)
		if parent == path {
			return true
		}
		path = parent
	}
}

func trustedFile(info os.FileInfo, secret bool) bool {
	return info.Mode().IsRegular() && secureMode(info.Mode(), secret) && trustedOwner(info)
}

func trustedDirectory(info os.FileInfo) bool {
	if !info.IsDir() || !trustedOwner(info) {
		return false
	}
	return trustedUnixDirectoryMode(info.Mode(), ownerID(info))
}

func trustedOwner(info os.FileInfo) bool {
	return trustedUnixOwner(ownerID(info))
}

func trustedUnixDirectoryMode(mode os.FileMode, owner int) bool {
	if mode.Perm()&0o022 == 0 {
		return true
	}
	return mode&os.ModeSticky != 0 && owner == 0
}

func trustedUnixOwner(owner int) bool { return owner == 0 || owner == os.Geteuid() }

func ownerID(info os.FileInfo) int {
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return -1
	}
	return int(stat.Uid)
}

func resolvedPathIsOriginal(path string) bool {
	resolved, err := filepath.EvalSymlinks(path)
	return err == nil && filepath.Clean(resolved) == filepath.Clean(path)
}
