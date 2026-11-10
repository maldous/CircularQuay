//go:build windows

package cli

import (
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/maldous/CircularQuay/internal/fault"
	"github.com/maldous/CircularQuay/internal/platform"
	"golang.org/x/sys/windows"
)

const configInitDiagnosticRoot = "CQ_CONFIG_INIT_DIAGNOSTIC_ROOT"

var errConfigInitDiagnosticChildExists = errors.New("diagnostic child exists")

type configInitDiagnosticRecord struct {
	Case     string `json:"case"`
	Stage    string `json:"stage"`
	Category string `json:"category"`
}

// TestConfigInitWindowsDiagnostic is selected only by the hosted diagnostic
// fixture. It never changes the supplied root or any entry that predates it.
func TestConfigInitWindowsDiagnostic(t *testing.T) {
	root, selected := os.LookupEnv(configInitDiagnosticRoot)
	if !selected {
		t.Skip("diagnostic-not-selected")
	}
	if !validConfigInitDiagnosticRoot(root) {
		t.Fatal("diagnostic root invalid")
	}
	runConfigInitDiagnosticCase(t, root, "ordinary", createOrdinaryDiagnosticChild)
	runConfigInitDiagnosticCase(t, root, "prepared", createPreparedDiagnosticChild)
}

func validConfigInitDiagnosticRoot(root string) bool {
	if !filepath.IsAbs(root) {
		return false
	}
	info, err := os.Lstat(root)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return false
	}
	return configInitDiagnosticRootIsOriginal(root)
}

func configInitDiagnosticRootIsOriginal(root string) bool {
	cleaned := filepath.Clean(root)
	volume := filepath.VolumeName(cleaned)
	if volume == "" {
		return false
	}
	current := volume + `\`
	for _, component := range strings.Split(strings.TrimPrefix(cleaned[len(volume):], `\`), `\`) {
		if component == "" {
			continue
		}
		current = filepath.Join(current, component)
		if !configInitDiagnosticDirectoryIsOriginal(current) {
			return false
		}
	}
	return true
}

func configInitDiagnosticDirectoryIsOriginal(path string) bool {
	name, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return false
	}
	attributes, err := windows.GetFileAttributes(name)
	return err == nil && attributes&windows.FILE_ATTRIBUTE_REPARSE_POINT == 0
}

type diagnosticChildCreator func(string) (string, error)

func runConfigInitDiagnosticCase(t *testing.T, root, name string, create diagnosticChildCreator) {
	t.Helper()
	child, err := create(root)
	if err != nil {
		emitConfigInitDiagnostic(t, configInitDiagnosticRecord{Case: name, Stage: configInitDiagnosticChildStage(name, err), Category: configInitDiagnosticCategory(err)})
		return
	}
	policy := filepath.Join(child, "cq.json")
	app := application{config: policy, runner: Runner{In: strings.NewReader(""), Out: io.Discard, Err: io.Discard}}
	_, err = app.initialize([]string{"--output", policy})
	emitConfigInitDiagnostic(t, configInitDiagnosticRecord{Case: name, Stage: configInitDiagnosticStage(err), Category: configInitDiagnosticCategory(err)})
}

func createOrdinaryDiagnosticChild(root string) (string, error) {
	child := filepath.Join(root, "cq-init-diagnostic-ordinary")
	if diagnosticChildExists(child) {
		return "", errConfigInitDiagnosticChildExists
	}
	if err := os.Mkdir(child, 0o700); err != nil {
		return "", err
	}
	return child, nil
}

func createPreparedDiagnosticChild(root string) (string, error) {
	child := filepath.Join(root, "cq-init-diagnostic-prepared")
	if diagnosticChildExists(child) {
		return "", errConfigInitDiagnosticChildExists
	}
	if err := platform.EnsurePrivateDirectory(child); err != nil {
		return "", err
	}
	return child, nil
}

func diagnosticChildExists(path string) bool {
	_, err := os.Lstat(path)
	return err == nil || !os.IsNotExist(err)
}

func emitConfigInitDiagnostic(t *testing.T, record configInitDiagnosticRecord) {
	t.Helper()
	encoded, err := json.Marshal(record)
	if err != nil {
		t.Fatal("diagnostic encoding failed")
	}
	t.Log(string(encoded))
}

func configInitDiagnosticChildStage(name string, err error) string {
	if errors.Is(err, errConfigInitDiagnosticChildExists) {
		return "child_preexisting"
	}
	if name == "ordinary" {
		return "mkdir_child"
	}
	return configInitDiagnosticPrivateDirectoryStage(err)
}

func configInitDiagnosticPrivateDirectoryStage(err error) string {
	var failure *fault.Error
	if !errors.As(err, &failure) {
		return "unknown"
	}
	switch failure.Operation {
	case "create private directory":
		return "directory_prepare"
	case "inspect private directory":
		return "directory_inspect"
	case "protect private directory":
		return "directory_protection"
	case "symbolic private directory", "private directory permissions":
		return "directory_identity"
	case "absolute private directory required":
		return "directory_input"
	default:
		return "unknown"
	}
}

func configInitDiagnosticStage(err error) string {
	if err == nil {
		return "success"
	}
	var failure *fault.Error
	if !errors.As(err, &failure) {
		return "unknown"
	}
	switch failure.Operation {
	case "init directory":
		return "parent_create"
	case "protected creation directory", "open protected creation directory", "creation directory changed":
		return "parent_identity"
	case "protect created file", "created file changed":
		return "file_protection_identity"
	case "create protected file", "inspect created file":
		return "exclusive_create"
	case "write staged file", "sync staged file", "close staged file", "sync created file directory":
		return "write_sync"
	case "init arguments", "init path", "init encode", "existing policy", "existing fixture", "protected creation destination":
		return "init"
	default:
		return "unknown"
	}
}

func configInitDiagnosticCategory(err error) string {
	if err == nil {
		return "success"
	}
	var failure *fault.Error
	if !errors.As(err, &failure) {
		return "unknown"
	}
	switch failure.Kind {
	case fault.Invalid, fault.Trust, fault.Denied, fault.Unavailable, fault.Missing, fault.Expired, fault.Unsupported, fault.Apply:
		return string(failure.Kind)
	default:
		return "unknown"
	}
}
