//go:build windows

package repository

import (
	"path/filepath"
	"strings"

	"github.com/maldous/CircularQuay/internal/fault"
	"golang.org/x/sys/windows"
)

// Win32 OpenSSH requires ProgramData during compatibility initialization,
// before standard streams are initialized. Resolve the OS known folder rather
// than inheriting a caller's environment override or user SSH configuration.
// See PowerShell/openssh-portable's win32compat/w32fd.c:init_prog_paths.
func nativeGitEnvironment() ([]string, error) {
	return programDataEnvironment(windows.KnownFolderPath(windows.FOLDERID_ProgramData, 0))
}

func programDataEnvironment(directory string, err error) ([]string, error) {
	if err != nil || !filepath.IsAbs(directory) || strings.ContainsAny(directory, "\x00\r\n") {
		return nil, failure(fault.Unsupported, "windows_support_directory")
	}
	return []string{"ProgramData=" + directory}, nil
}
