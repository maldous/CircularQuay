//go:build windows

package repository

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/maldous/CircularQuay/internal/fault"
)

func TestGitEnvironmentDoesNotInheritProgramDataOverride(t *testing.T) {
	poison := filepath.Join(privateRepositoryDirectory(t, "caller"), "not-system-program-data")
	t.Setenv("ProgramData", poison)
	t.Setenv("SSH_AUTH_SOCK", "synthetic-agent")
	environment, err := (&store{}).environment(false, false)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, entry := range environment {
		name, value, _ := strings.Cut(entry, "=")
		if strings.EqualFold(name, "SSH_AUTH_SOCK") {
			t.Fatal("Git inherited caller SSH agent")
		}
		if strings.EqualFold(name, "ProgramData") {
			if found {
				t.Fatal("Git support directory is ambiguous")
			}
			assertNativeSupportDirectory(t, value, poison)
			found = true
		}
	}
	if !found {
		t.Fatal("native Git support directory missing")
	}
}

func assertNativeSupportDirectory(t *testing.T, value, poison string) {
	t.Helper()
	if value == poison || !filepath.IsAbs(value) {
		t.Fatal("Git support directory was inherited or relative")
	}
	info, err := os.Stat(value)
	if err != nil || !info.IsDir() {
		t.Fatal("Git support directory is unavailable")
	}
}

func TestGitEnvironmentRejectsUnavailableSupportDirectory(t *testing.T) {
	for _, directory := range []string{"", "relative", "C:\\data\nambiguous", "C:\\data\x00ambiguous"} {
		if _, err := programDataEnvironment(directory, nil); !fault.Is(err, fault.Unsupported) {
			t.Fatal("invalid native support directory accepted")
		}
	}
	if _, err := programDataEnvironment(`C:\ProgramData`, errors.New("synthetic OS lookup failure")); !fault.Is(err, fault.Unsupported) {
		t.Fatal("failed OS support-directory lookup accepted")
	}
}
