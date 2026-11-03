package assurance

import (
	"context"
	"crypto/sha256"
	"debug/buildinfo"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/maldous/CircularQuay/internal/platform"
)

// qualifiedTools resolves audit commands only from a verified manifest. It
// rechecks executable bytes and records a bounded version probe before each
// requested command. It does not establish publisher provenance or sandbox
// the program it starts.
type qualifiedTools struct {
	manifest     VerifiedToolManifest
	command      commandFunc
	probes       map[string]ToolProbe
	helperCopies map[string]string
}

// ToolProbe records the observed status of a tool's fixed version probe.
type ToolProbe struct {
	ID     string `json:"id"`
	Status string `json:"status"`
	Code   string `json:"code,omitempty"`
}

func newQualifiedTools(manifest VerifiedToolManifest, command commandFunc) *qualifiedTools {
	return &qualifiedTools{manifest: manifest, command: command, probes: map[string]ToolProbe{}, helperCopies: map[string]string{}}
}

func (tools *qualifiedTools) tool(id string) (VerifiedTool, bool) {
	for _, tool := range tools.manifest.Tools {
		if tool.ID == id {
			return tool, true
		}
	}
	return VerifiedTool{}, false
}

func (tools *qualifiedTools) dependenciesMatch() bool {
	for _, id := range []string{"git", "go"} {
		tool, found := tools.tool(id)
		if !found || !toolMatchesDigest(tool) {
			return false
		}
	}
	return tools.helpersMatch()
}

func (tools *qualifiedTools) commandFor(id string) commandFunc {
	return func(ctx context.Context, _ string, arguments []string, directory string, environment []string) commandResult {
		return tools.run(ctx, id, arguments, directory, environment)
	}
}

func (tools *qualifiedTools) run(ctx context.Context, id string, arguments []string, directory string, environment []string) commandResult {
	tool, found := tools.tool(id)
	if !found {
		tools.probes[id] = ToolProbe{ID: id, Status: "not_run", Code: "tool_not_selected"}
		return commandResult{exitCode: 127, missing: true}
	}
	return tools.runVerified(ctx, id, tool, arguments, directory, environment)
}

func (tools *qualifiedTools) runVerified(ctx context.Context, id string, tool VerifiedTool, arguments []string, directory string, environment []string) commandResult {
	if !tools.helpersMatch() {
		return tools.failed(id, "helper_digest_mismatch")
	}
	if !toolMatchesDigest(tool) {
		return tools.failed(id, "binary_digest_mismatch")
	}
	probe := tools.command(ctx, tool.Path, versionArguments(id), directory, environment)
	if code := probeCode(id, tool, probe); code != "" {
		return tools.failed(id, code)
	}
	if id == "gofmt" && !sameGofmtToolchain(tool.Path, tools.goPath()) {
		return tools.failed(id, "gofmt_toolchain_mismatch")
	}
	if !toolMatchesDigest(tool) {
		return tools.failed(id, "binary_changed_after_probe")
	}
	if !tools.helpersMatch() {
		return tools.failed(id, "helper_changed_after_probe")
	}
	tools.probes[id] = ToolProbe{ID: id, Status: "passed"}
	result := tools.command(ctx, tool.Path, arguments, directory, environment)
	if !tools.helpersMatch() {
		return tools.failed(id, "helper_changed_during_execution")
	}
	return result
}

func (tools *qualifiedTools) failed(id, code string) commandResult {
	tools.probes[id] = ToolProbe{ID: id, Status: "failed", Code: code}
	return commandResult{exitCode: 1}
}

func toolMatchesDigest(tool VerifiedTool) bool {
	if platform.ValidateFileMetadata(tool.Path, false) != nil || platform.ValidateExecutable(tool.Path) != nil {
		return false
	}
	actual, err := executableSHA256(tool.Path)
	return err == nil && actual == tool.SHA256
}

func probeCode(id string, tool VerifiedTool, probe commandResult) string {
	if probe.missing || !probeExitExpected(id, probe.exitCode) {
		return "version_probe_failed"
	}
	digest := sha256.Sum256(probe.output)
	if hex.EncodeToString(digest[:]) != tool.DeclaredVersionOutputSHA256 {
		return "version_output_mismatch"
	}
	return ""
}

func (tools *qualifiedTools) goPath() string {
	tool, found := tools.tool("go")
	if !found {
		return ""
	}
	return tool.Path
}

func versionArguments(id string) []string {
	if id == "go" {
		return []string{"version"}
	}
	if id == "gofmt" {
		return []string{"-h"}
	}
	return []string{"--version"}
}

func probeExitExpected(id string, exitCode int) bool {
	return exitCode == 0 || id == "gofmt" && exitCode == 2
}

func sameGofmtToolchain(gofmt, goBinary string) bool {
	if goBinary == "" || !strings.EqualFold(filepath.Dir(gofmt), filepath.Dir(goBinary)) {
		return false
	}
	gofmtInfo, gofmtErr := buildinfo.ReadFile(gofmt)
	goInfo, goErr := buildinfo.ReadFile(goBinary)
	return gofmtErr == nil && goErr == nil && gofmtInfo.GoVersion != "" && gofmtInfo.GoVersion == goInfo.GoVersion
}

func (tools *qualifiedTools) report() ToolManifestReport {
	report := ToolManifestReport{SHA256: tools.manifest.ManifestSHA256, Helpers: tools.helperIdentities()}
	for _, tool := range tools.manifest.Tools {
		probe, found := tools.probes[tool.ID]
		if !found {
			probe = ToolProbe{ID: tool.ID, Status: "not_run", Code: "not_selected_by_profile"}
		}
		report.Tools = append(report.Tools, ToolIdentity{ID: tool.ID, Path: tool.Path, SHA256: tool.SHA256, Version: tool.DeclaredVersion, Rules: tool.DeclaredRules, Probe: probe})
	}
	return report
}

func (tools *qualifiedTools) environment(environment []string, helperDirectory string) ([]string, error) {
	if err := platform.EnsurePrivateDirectory(helperDirectory); err != nil {
		return nil, err
	}
	for _, id := range []string{"git", "go"} {
		tool, found := tools.tool(id)
		if !found || !toolMatchesDigest(tool) || copyQualifiedHelper(tool, filepath.Join(helperDirectory, canonicalToolName(id))) != nil {
			return nil, os.ErrPermission
		}
	}
	if err := tools.copyNativeHelpers(helperDirectory); err != nil {
		return nil, err
	}
	goRoot, err := tools.goRoot()
	if err != nil {
		return nil, err
	}
	filtered := make([]string, 0, len(environment)+1)
	for _, value := range environment {
		name, _, _ := strings.Cut(value, "=")
		if !strings.EqualFold(name, "PATH") && !strings.EqualFold(name, "GOROOT") {
			filtered = append(filtered, value)
		}
	}
	// Child tools can only resolve the mandatory helper names from their own
	// already-qualified copies, including explicitly declared native dependencies.
	// Commands themselves always use absolute
	// manifest paths; this PATH is deliberately not an analyzer tool directory.
	return append(filtered, "PATH="+helperDirectory, "GOROOT="+goRoot), nil
}

func (tools *qualifiedTools) goRoot() (string, error) {
	goTool, found := tools.tool("go")
	if !found || filepath.Base(filepath.Dir(goTool.Path)) != "bin" {
		return "", os.ErrInvalid
	}
	root, err := canonicalDirectory(filepath.Dir(filepath.Dir(goTool.Path)))
	if err != nil || !toolMatchesDigest(goTool) {
		return "", os.ErrInvalid
	}
	return root, nil
}

func canonicalToolName(id string) string {
	if runtime.GOOS == "windows" {
		return id + ".exe"
	}
	return id
}

func copyQualifiedHelper(tool VerifiedTool, target string) error {
	return copyHelperBinary(helperBinary{id: tool.ID, path: tool.Path, sha256: tool.SHA256}, target)
}

type helperBinary struct {
	id, path, sha256 string
}

func copyHelperBinary(binary helperBinary, target string) error {
	if filepath.Base(binary.path) != canonicalToolName(binary.id) {
		return os.ErrInvalid
	}
	if existingHelperBinaryMatches(binary, target) {
		return nil
	}
	if _, err := os.Lstat(target); err == nil {
		return os.ErrInvalid
	}
	if err := writeHelper(binary.path, target); err != nil {
		return err
	}
	digest, err := executableSHA256(target)
	if err != nil || digest != binary.sha256 {
		_ = os.Remove(target)
		return os.ErrInvalid
	}
	return enableHelper(target)
}

func existingHelperMatches(tool VerifiedTool, target string) bool {
	return existingHelperBinaryMatches(helperBinary{id: tool.ID, path: tool.Path, sha256: tool.SHA256}, target)
}

func existingHelperBinaryMatches(binary helperBinary, target string) bool {
	if platform.ValidateFileMetadata(target, false) != nil {
		return false
	}
	digest, err := executableSHA256(target)
	return err == nil && digest == binary.sha256 && helperExecutable(target)
}

func helperExecutable(path string) bool {
	if platform.ValidateExecutable(path) != nil {
		return false
	}
	info, err := os.Lstat(path)
	return err == nil && (runtime.GOOS == "windows" || info.Mode().Perm() == 0700)
}

func writeHelper(sourcePath, target string) error {
	source, opened, err := openToolFile(sourcePath)
	if err != nil {
		return err
	}
	defer source.Close()
	root, err := os.OpenRoot(filepath.Dir(target))
	if err != nil {
		return err
	}
	defer root.Close()
	name := filepath.Base(target)
	// Keep an incomplete or unverified helper non-executable. Rooted exclusive
	// creation prevents a target basename from escaping the private helper root.
	destination, err := root.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return err
	}
	read, err := copyHelperBytes(destination, source)
	if err != nil || unchangedToolFile(sourcePath, opened, source, read) != nil {
		_ = root.Remove(name)
		return os.ErrInvalid
	}
	return nil
}

func copyHelperBytes(destination, source *os.File) (int64, error) {
	read, copyErr := io.Copy(destination, io.LimitReader(source, maxToolBinaryBytes+1))
	syncErr := destination.Sync()
	closeErr := destination.Close()
	if copyErr != nil || syncErr != nil || closeErr != nil || read > maxToolBinaryBytes {
		return 0, os.ErrInvalid
	}
	return read, nil
}

func enableHelper(path string) error {
	root, err := os.OpenRoot(filepath.Dir(path))
	if err != nil {
		return err
	}
	defer root.Close()
	// Only a complete digest-matched copy gains owner execute permission.
	// Group and other users never receive read, write, or execute permission.
	if err = root.Chmod(filepath.Base(path), 0700); err != nil {
		return err
	}
	if !helperExecutable(path) {
		return os.ErrPermission
	}
	return nil
}
