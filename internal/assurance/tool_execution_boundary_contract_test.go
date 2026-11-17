package assurance

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"
)

func TestToolExecutionBoundaryContractUnselectedToolCannotReachCallback(t *testing.T) {
	tools := toolExecutionBoundaryTools(t, []string{"git"}, func(context.Context, string, []string, string, []string) commandResult {
		t.Fatal("unselected tool reached the fake command callback")
		return commandResult{}
	})
	got := tools.run(context.Background(), "go", []string{"version"}, "unused", nil)
	want := commandResult{exitCode: 127, missing: true}
	if !reflect.DeepEqual(got, want) || tools.probes["go"] != (ToolProbe{ID: "go", Status: "not_run", Code: "tool_not_selected"}) {
		t.Fatalf("unselected tool was not withheld: result=%+v probe=%+v", got, tools.probes["go"])
	}
}

func TestToolExecutionBoundaryContractProbeFailureWithholdsRequestedAction(t *testing.T) {
	for _, scenario := range []string{"missing", "nonzero"} {
		t.Run(scenario, func(t *testing.T) { toolExecutionBoundaryProbeRefusal(t, scenario) })
	}
}

func toolExecutionBoundaryProbeRefusal(t *testing.T, scenario string) {
	t.Helper()
	calls := 0
	tools := toolExecutionBoundaryTools(t, []string{"git"}, func(_ context.Context, _ string, args []string, _ string, _ []string) commandResult {
		calls++
		if calls != 1 || !reflect.DeepEqual(args, []string{"--version"}) {
			t.Fatal("failed version probe reached the requested action")
		}
		if scenario == "missing" {
			return commandResult{exitCode: 127, missing: true}
		}
		return commandResult{exitCode: 7}
	})
	got := tools.run(context.Background(), "git", []string{"status"}, "unused", nil)
	toolExecutionBoundaryRefused(t, tools, "git", "version_probe_failed", got)
	if calls != 1 {
		t.Fatalf("version probe callback count = %d", calls)
	}
}

func TestToolExecutionBoundaryContractProbeExitAndDigestControls(t *testing.T) {
	git := VerifiedTool{DeclaredVersionOutputSHA256: digest([]byte("synthetic-version"))}
	probe := commandResult{output: []byte("synthetic-version")}
	if code := probeCode("git", git, probe); code != "" {
		t.Fatalf("matching successful Git probe rejected: %s", code)
	}
	probe = commandResult{exitCode: 2}
	git.DeclaredVersionOutputSHA256 = digest(nil)
	if code := probeCode("gofmt", git, probe); code != "" {
		t.Fatalf("matching gofmt help exit rejected: %s", code)
	}
	if code := probeCode("git", git, probe); code != "version_probe_failed" {
		t.Fatalf("gofmt-specific exit admitted for Git: %s", code)
	}
}

func TestToolExecutionBoundaryContractBinaryDriftAfterProbeStopsAction(t *testing.T) {
	calls := 0
	tools := toolExecutionBoundaryTools(t, []string{"git"}, func(_ context.Context, binary string, args []string, _ string, _ []string) commandResult {
		calls++
		if calls != 1 || !reflect.DeepEqual(args, []string{"--version"}) {
			t.Fatal("changed binary reached the requested action")
		}
		if err := os.WriteFile(binary, []byte("changed synthetic tool\n"), 0700); err != nil {
			t.Fatal(err)
		}
		return commandResult{output: []byte("version-output-git")}
	})
	got := tools.run(context.Background(), "git", []string{"status"}, "unused", nil)
	toolExecutionBoundaryRefused(t, tools, "git", "binary_changed_after_probe", got)
	if calls != 1 {
		t.Fatalf("changed-binary callback count = %d", calls)
	}
}

func TestToolExecutionBoundaryContractGofmtRequiresActualToolchainIdentity(t *testing.T) {
	for _, scenario := range []string{"missing Go", "different directories", "non-Go files"} {
		t.Run(scenario, func(t *testing.T) { toolExecutionBoundaryGofmtRefusal(t, scenario) })
	}
}

func toolExecutionBoundaryGofmtRefusal(t *testing.T, scenario string) {
	t.Helper()
	manifest := toolExecutionBoundaryManifest(t, []string{"go", "gofmt"})
	if scenario == "missing Go" {
		manifest.Tools = manifest.Tools[1:]
	}
	if scenario == "different directories" {
		path := filepath.Join(privateAssuranceDirectory(t, "other-go"), "go")
		manifest.Tools[0].Path = writeTool(t, path, "synthetic tool go\n")
	}
	verified := toolExecutionBoundaryLoad(t, manifest)
	calls := 0
	tools := newQualifiedTools(verified, func(_ context.Context, binary string, args []string, _ string, _ []string) commandResult {
		calls++
		if calls != 1 || binary != toolPath(verified, "gofmt") || !reflect.DeepEqual(args, []string{"-h"}) {
			t.Fatal("unidentified gofmt toolchain reached the requested action")
		}
		return commandResult{output: []byte("version-output-gofmt")}
	})
	got := tools.run(context.Background(), "gofmt", []string{"-l", "fixture.go"}, "unused", nil)
	toolExecutionBoundaryRefused(t, tools, "gofmt", "gofmt_toolchain_mismatch", got)
	if calls != 1 {
		t.Fatalf("gofmt probe callback count = %d", calls)
	}
}

func TestToolExecutionBoundaryContractMandatoryDependenciesRemainCurrent(t *testing.T) {
	for _, scenario := range []string{"stable", "missing git", "missing go", "changed git", "changed go"} {
		t.Run(scenario, func(t *testing.T) { toolExecutionBoundaryDependencies(t, scenario) })
	}
}

func toolExecutionBoundaryDependencies(t *testing.T, scenario string) {
	t.Helper()
	ids := []string{"git", "go"}
	if scenario == "missing git" {
		ids = []string{"go"}
	}
	if scenario == "missing go" {
		ids = []string{"git"}
	}
	tools := toolExecutionBoundaryTools(t, ids, func(context.Context, string, []string, string, []string) commandResult {
		t.Fatal("dependency integrity check executed a command")
		return commandResult{}
	})
	for _, id := range []string{"git", "go"} {
		if scenario == "changed "+id {
			if err := os.WriteFile(toolPath(tools.manifest, id), []byte("changed dependency\n"), 0700); err != nil {
				t.Fatal(err)
			}
		}
	}
	if got := tools.dependenciesMatch(); got != (scenario == "stable") {
		t.Fatalf("dependency currentness for %q = %t", scenario, got)
	}
}

func TestToolExecutionBoundaryContractHelperCopyReusesButNeverOverwrites(t *testing.T) {
	for _, scenario := range []string{"reuse", "wrong basename", "existing different bytes", "missing parent"} {
		t.Run(scenario, func(t *testing.T) { toolExecutionBoundaryCopy(t, scenario) })
	}
}

func toolExecutionBoundaryCopy(t *testing.T, scenario string) {
	t.Helper()
	root := privateAssuranceDirectory(t, "copy-boundary")
	name := canonicalToolName("git")
	if scenario == "wrong basename" {
		name = canonicalToolName("not-git")
	}
	content := []byte("synthetic helper boundary\n")
	source := writeTool(t, filepath.Join(root, name), string(content))
	target := filepath.Join(ensurePrivateDirectory(t, filepath.Join(root, "copies")), canonicalToolName("git"))
	tool := VerifiedTool{ID: "git", Path: source, SHA256: digest(content)}
	if scenario == "reuse" {
		toolExecutionBoundaryReuse(t, tool, target, content)
		return
	}
	var prior os.FileInfo
	if scenario == "existing different bytes" {
		writeTool(t, target, "preserved existing helper\n")
		prior = toolExecutionBoundaryInfo(t, target)
	}
	if scenario == "missing parent" {
		target = filepath.Join(root, "missing", canonicalToolName("git"))
	}
	toolExecutionBoundaryCopyRefused(t, scenario, copyQualifiedHelper(tool, target))
	toolExecutionBoundaryBytes(t, source, content)
	if prior != nil {
		toolExecutionBoundaryBytes(t, target, []byte("preserved existing helper\n"))
		if !os.SameFile(prior, toolExecutionBoundaryInfo(t, target)) {
			t.Fatal("refused helper replaced the existing entry")
		}
		return
	}
	if _, err := os.Lstat(target); !os.IsNotExist(err) {
		t.Fatalf("refused helper left a destination: %v", err)
	}
}

func toolExecutionBoundaryCopyRefused(t *testing.T, scenario string, err error) {
	t.Helper()
	if err == nil {
		t.Fatal("invalid helper destination or identity was accepted")
	}
	if scenario != "missing parent" && !errors.Is(err, os.ErrInvalid) {
		t.Fatalf("helper identity refusal lost its fixed error: %v", err)
	}
}

func toolExecutionBoundaryReuse(t *testing.T, tool VerifiedTool, target string, content []byte) {
	t.Helper()
	if err := copyQualifiedHelper(tool, target); err != nil {
		t.Fatal(err)
	}
	before := toolExecutionBoundaryInfo(t, target)
	if err := copyQualifiedHelper(tool, target); err != nil {
		t.Fatalf("verified helper was not reusable: %v", err)
	}
	after := toolExecutionBoundaryInfo(t, target)
	if !os.SameFile(before, after) || before.Mode() != after.Mode() || !existingHelperMatches(tool, target) {
		t.Fatal("helper reuse changed its entry, mode or identity")
	}
	toolExecutionBoundaryBytes(t, target, content)
	toolExecutionBoundaryBytes(t, tool.Path, content)
}

func TestToolExecutionBoundaryContractDescriptorCopyClosesOnReadFailure(t *testing.T) {
	for _, name := range []string{"owned descriptor", "closed source"} {
		t.Run(name, func(t *testing.T) {
			toolExecutionBoundaryDescriptorCopy(t, name == "closed source")
		})
	}
}

func toolExecutionBoundaryDescriptorCopy(t *testing.T, closed bool) {
	t.Helper()
	root := privateAssuranceDirectory(t, "descriptor-copy")
	content := []byte("owned descriptor bytes\n")
	path := writeTool(t, filepath.Join(root, "source"), string(content))
	source, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = source.Close() })
	target := filepath.Join(root, "destination")
	destination, err := os.OpenFile(target, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = destination.Close() })
	if closed {
		if err = source.Close(); err != nil {
			t.Fatal(err)
		}
	}
	count, err := copyHelperBytes(destination, source)
	toolExecutionBoundaryCopyOutcome(t, closed, count, err, int64(len(content)))
	written, writeErr := destination.Write([]byte("destination must be closed"))
	if written != 0 || !errors.Is(writeErr, os.ErrClosed) {
		t.Fatalf("copy did not close destination: written=%d error=%v", written, writeErr)
	}
	want := content
	if closed {
		want = []byte{}
	}
	toolExecutionBoundaryBytes(t, target, want)
	toolExecutionBoundaryBytes(t, path, content)
	if info := toolExecutionBoundaryInfo(t, target); runtime.GOOS != "windows" && info.Mode().Perm() != 0600 {
		t.Fatal("descriptor copy enabled an unverified helper")
	}
}

func toolExecutionBoundaryCopyOutcome(t *testing.T, closed bool, count int64, err error, size int64) {
	t.Helper()
	if closed {
		if !errors.Is(err, os.ErrInvalid) || count != 0 {
			t.Fatalf("failed descriptor copy was accepted: bytes=%d err=%v", count, err)
		}
		return
	}
	if err != nil || count != size {
		t.Fatalf("stable descriptor copy failed: bytes=%d err=%v", count, err)
	}
}

func toolExecutionBoundaryManifest(t *testing.T, ids []string) ToolManifest {
	t.Helper()
	manifest := manifestFixture(t)
	var selected []ToolEntry
	for _, tool := range manifest.Tools {
		for _, id := range ids {
			if tool.ID == id {
				selected = append(selected, tool)
			}
		}
	}
	manifest.Tools = selected
	return manifest
}

func toolExecutionBoundaryLoad(t *testing.T, manifest ToolManifest) VerifiedToolManifest {
	t.Helper()
	verified, err := LoadToolManifest(writeManifest(t, manifest), fixtureLocations(t))
	if err != nil {
		t.Fatal(err)
	}
	return verified
}

func toolExecutionBoundaryTools(t *testing.T, ids []string, command commandFunc) *qualifiedTools {
	t.Helper()
	return newQualifiedTools(toolExecutionBoundaryLoad(t, toolExecutionBoundaryManifest(t, ids)), command)
}

func toolExecutionBoundaryRefused(t *testing.T, tools *qualifiedTools, id, code string, got commandResult) {
	t.Helper()
	if !reflect.DeepEqual(got, commandResult{exitCode: 1}) || tools.probes[id] != (ToolProbe{ID: id, Status: "failed", Code: code}) {
		t.Fatalf("tool refusal did not withhold output/action: result=%+v probe=%+v", got, tools.probes[id])
	}
}

func toolExecutionBoundaryBytes(t *testing.T, path string, want []byte) {
	t.Helper()
	got, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(got, want) {
		t.Fatalf("owned fixture bytes changed: bytes=%d err=%v", len(got), err)
	}
}

// File.Stat binds Windows file identity before any subsequent name operation;
// close immediately so the fixture does not rely on a held sharing-mode lock.
func toolExecutionBoundaryInfo(t *testing.T, path string) os.FileInfo {
	t.Helper()
	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	info, statErr := file.Stat()
	closeErr := file.Close()
	if statErr != nil || closeErr != nil {
		t.Fatalf("owned fixture identity capture failed: stat=%v close=%v", statErr, closeErr)
	}
	return info
}
