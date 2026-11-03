//go:build windows

package platform

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/maldous/CircularQuay/internal/fault"
	"github.com/maldous/CircularQuay/internal/model"
)

func TestTrustedSecretLaunchRequiresProtectedDLLSearchDirectories(t *testing.T) {
	root := filepath.Join(t.TempDir(), "launch")
	work, executable, pathDirectory := protectedLaunchFixture(t, root)
	environment := []string{"PATH=" + pathDirectory}
	if directory, err := launchDirectory(work, secretGeneration(), executable, environment); err != nil || directory != work.Directory {
		t.Fatalf("protected launch directory = %q, %v", directory, err)
	}

	for _, test := range []struct {
		name   string
		unsafe func(model.Workload, string, string) string
	}{
		{"working directory", func(work model.Workload, _ string, _ string) string { return work.Directory }},
		{"executable directory", func(_ model.Workload, executable, _ string) string { return filepath.Dir(executable) }},
		{"PATH directory", func(_ model.Workload, _ string, path string) string { return path }},
	} {
		t.Run(test.name, func(t *testing.T) {
			caseWork, caseExecutable, casePath := protectedLaunchFixture(t, filepath.Join(t.TempDir(), "launch"))
			if err := setWindowsTestDACL(test.unsafe(caseWork, caseExecutable, casePath), "GW"); err != nil {
				t.Fatal(err)
			}
			if _, err := launchDirectory(caseWork, secretGeneration(), caseExecutable, []string{"PATH=" + casePath}); !fault.Is(err, fault.Trust) {
				t.Fatalf("writable DLL search directory accepted: %v", err)
			}
		})
	}
}

func TestTrustedSecretLaunchAcceptsReadOnlySharedSearchDirectories(t *testing.T) {
	work, executable, pathDirectory := protectedLaunchFixture(t, filepath.Join(t.TempDir(), "launch"))
	for _, directory := range []string{work.Directory, filepath.Dir(executable), pathDirectory} {
		if err := setWindowsTestDACL(directory, "GR"); err != nil {
			t.Fatal(err)
		}
	}
	if directory, err := launchDirectory(work, secretGeneration(), executable, []string{"PATH=" + pathDirectory}); err != nil || directory != work.Directory {
		t.Fatalf("read-only shared directories rejected: %q, %v", directory, err)
	}
}

func TestTrustedSecretLaunchRejectsRelativeAndEmptyPATHDirectories(t *testing.T) {
	root := filepath.Join(t.TempDir(), "launch")
	work, executable, _ := protectedLaunchFixture(t, root)
	for _, value := range []string{"", `relative;C:\approved`, `C:\approved;;C:\other`} {
		if _, err := launchDirectory(work, secretGeneration(), executable, []string{"PATH=" + value}); !fault.Is(err, fault.Trust) {
			t.Fatalf("PATH %q accepted: %v", value, err)
		}
	}
}

func TestTrustedSecretLaunchChecksCurrentDirectoryWhenOmitted(t *testing.T) {
	root := filepath.Join(t.TempDir(), "launch")
	work, executable, pathDirectory := protectedLaunchFixture(t, root)
	current := filepath.Join(root, "current")
	if err := EnsurePrivateDirectory(current); err != nil {
		t.Fatal(err)
	}
	t.Chdir(current)
	work.Directory = ""
	if directory, err := launchDirectory(work, secretGeneration(), executable, []string{"PATH=" + pathDirectory}); err != nil || directory != current {
		t.Fatalf("current directory = %q, %v", directory, err)
	}
	if err := setWindowsTestDACL(current, "GW"); err != nil {
		t.Fatal(err)
	}
	if _, err := launchDirectory(work, secretGeneration(), executable, []string{"PATH=" + pathDirectory}); !fault.Is(err, fault.Trust) {
		t.Fatalf("writable current directory accepted: %v", err)
	}
}

func TestTrustedSecretLaunchRejectsReparseSearchDirectories(t *testing.T) {
	for _, test := range []struct {
		name  string
		apply func(*model.Workload, *string, string)
	}{
		{"working directory", func(work *model.Workload, _ *string, junction string) { work.Directory = junction }},
		{"PATH directory", func(_ *model.Workload, path *string, junction string) { *path = junction }},
	} {
		t.Run(test.name, func(t *testing.T) {
			root := filepath.Join(t.TempDir(), "launch")
			work, executable, pathDirectory := protectedLaunchFixture(t, root)
			junction := filepath.Join(root, "search-junction")
			if err := createTestJunction(junction, pathDirectory); err != nil {
				t.Fatal(err)
			}
			test.apply(&work, &pathDirectory, junction)
			if _, err := launchDirectory(work, secretGeneration(), executable, []string{"PATH=" + pathDirectory}); !fault.Is(err, fault.Trust) {
				t.Fatalf("reparse search directory accepted: %v", err)
			}
		})
	}
}

func TestUntrustedOrSecretlessLaunchSkipsDLLBoundary(t *testing.T) {
	work := model.Workload{Directory: "relative", UntrustedCode: true}
	if directory, err := launchDirectory(work, secretGeneration(), `C:\missing\helper.exe`, []string{"PATH=relative"}); err != nil || directory != work.Directory {
		t.Fatalf("untrusted contract changed: %q, %v", directory, err)
	}
	work.UntrustedCode = false
	if directory, err := launchDirectory(work, model.Generation{}, `C:\missing\helper.exe`, []string{"PATH=relative"}); err != nil || directory != work.Directory {
		t.Fatalf("secretless contract changed: %q, %v", directory, err)
	}
}

func protectedLaunchFixture(t *testing.T, root string) (model.Workload, string, string) {
	t.Helper()
	workDirectory := filepath.Join(root, "work")
	executableDirectory := filepath.Join(root, "bin")
	pathDirectory := filepath.Join(root, "path")
	for _, directory := range []string{workDirectory, executableDirectory, pathDirectory} {
		if err := EnsurePrivateDirectory(directory); err != nil {
			t.Fatal(err)
		}
	}
	executable := filepath.Join(executableDirectory, "helper.exe")
	if err := os.WriteFile(executable, []byte("synthetic"), 0o700); err != nil {
		t.Fatal(err)
	}
	return model.Workload{Directory: workDirectory}, executable, pathDirectory
}

func secretGeneration() model.Generation {
	return model.Generation{Values: map[string]model.Value{"APP_SECRET": {Data: "synthetic"}}}
}
