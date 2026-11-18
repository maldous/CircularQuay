package assurance

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/maldous/CircularQuay/internal/platform"
)

func TestToolEnvironmentContractUnavailableInputsCannotReachCommand(t *testing.T) {
	root := privateAssuranceDirectory(t, "helpers")
	file := filepath.Join(root, "occupied")
	if err := platform.WriteProtected(file, []byte("owned")); err != nil {
		t.Fatal(err)
	}
	tools := newQualifiedTools(VerifiedToolManifest{}, func(context.Context, string, []string, string, []string) commandResult {
		t.Fatal("environment admission executed a command")
		return commandResult{}
	})
	for _, directory := range []string{filepath.Join(file, "child"), filepath.Join(root, "fresh")} {
		if environment, err := tools.environment([]string{"PATH=untrusted"}, directory); err == nil || environment != nil {
			t.Fatal("invalid environment admission returned partial child authority")
		}
	}
	if tools.goPath() != "" || toolMatchesDigest(VerifiedTool{Path: filepath.Join(root, "absent")}) {
		t.Fatal("missing tool identity was accepted")
	}
}

func TestToolEnvironmentContractGoRootAndNativeHelpersRefuseUnboundBytes(t *testing.T) {
	root := privateAssuranceDirectory(t, "helpers")
	for _, path := range []string{"", filepath.Join(root, "go"), filepath.Join(root, "missing", "bin", canonicalToolName("go"))} {
		tools := newQualifiedTools(VerifiedToolManifest{Tools: []VerifiedTool{{ID: "go", Path: path}}}, nil)
		if value, err := tools.goRoot(); err == nil || value != "" {
			t.Fatal("invalid Go root produced a usable directory")
		}
	}
	tools := newQualifiedTools(VerifiedToolManifest{Helpers: []VerifiedHelper{{ID: sshKeygenHelperID, Path: filepath.Join(root, "missing")}}}, nil)
	if err := tools.copyNativeHelpers(root); !errors.Is(err, os.ErrPermission) || len(tools.helperCopies) != 0 {
		t.Fatal("unbound native helper was prepared")
	}
	if tools.manifest.Helpers[0].matches() || helperExecutable(filepath.Join(root, "missing")) {
		t.Fatal("missing native helper was executable")
	}
}

func TestToolEnvironmentContractEnableRequiresOwnedExistingRootAndFile(t *testing.T) {
	root := privateAssuranceDirectory(t, "helpers")
	for _, target := range []string{filepath.Join(root, "absent", canonicalToolName("git")), filepath.Join(root, canonicalToolName("git"))} {
		if err := enableHelper(target); err == nil {
			t.Fatal("missing helper root or file became executable")
		}
		if _, err := os.Lstat(target); !os.IsNotExist(err) {
			t.Fatal("failed enable created a helper")
		}
	}
}

func TestToolEnvironmentContractRawRulesFailClosed(t *testing.T) {
	if !hasRawRulesField([]byte("not-json"), "server_version") {
		t.Fatal("malformed rule metadata was treated as absent")
	}
}

func TestToolEnvironmentContractLateAdmissionFailuresReturnNoEnvironment(t *testing.T) {
	for _, stage := range []string{"native-helper", "go-root"} {
		t.Run(stage, func(t *testing.T) { toolEnvironmentLateRefusal(t, stage) })
	}
}

func toolEnvironmentLateRefusal(t *testing.T, stage string) {
	t.Helper()
	directory := privateAssuranceDirectory(t, "not-bin")
	var selected []VerifiedTool
	for _, id := range []string{"git", "go"} {
		content := "synthetic inert " + id
		path := writeTool(t, filepath.Join(directory, canonicalToolName(id)), content)
		selected = append(selected, VerifiedTool{ID: id, Path: path, SHA256: digest([]byte(content))})
	}
	tools := newQualifiedTools(VerifiedToolManifest{Tools: selected}, nil)
	if stage == "native-helper" {
		tools.manifest.Helpers = []VerifiedHelper{{ID: sshKeygenHelperID, Path: filepath.Join(directory, "missing")}}
	}
	environment, err := tools.environment([]string{"PATH=not-admitted"}, privateAssuranceDirectory(t, "copies"))
	if err == nil || environment != nil {
		t.Fatal("late helper/root refusal returned partial process authority")
	}
}
