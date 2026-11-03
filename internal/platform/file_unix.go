//go:build !windows

package platform

import "os"

func secureMode(mode os.FileMode, secret bool) bool {
	if secret {
		return mode.Perm()&0o077 == 0
	}
	return mode.Perm()&0o022 == 0
}

func replaceFile(from, to string) error { return os.Rename(from, to) }

func protectFile(path string) error { return os.Chmod(path, 0o600) }

func protectDirectory(path string) error { return os.Chmod(path, 0o700) }

func syncDirectory(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return f.Sync()
}
