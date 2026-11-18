package assurance

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"sort"
	"strings"
	"testing"

	"github.com/maldous/CircularQuay/internal/platform"
)

// These fixtures enter the production manifest pipeline, not runForTest.
// Every command, Git response and version probe is supplied in process. The
// resulting report describes fabricated source/tool records, never execution
// or qualification of Git, Go, an analyzer, or this repository's real source.
type auditPipelineFixture struct {
	root        string
	output      string
	manifest    string
	declaration ToolManifest
	files       map[string][]byte
	blobs       map[string][]byte
	inputs      map[string]auditPipelineInput
	toolPaths   map[string]string
	calls       map[string]int
	totalCalls  int
	bare        bool
	blockReport bool
}

type auditPipelineInput struct {
	sha256 string
	info   os.FileInfo
}

func TestAuditPipelineContractPublicManifestRefusal(t *testing.T) {
	fixture := newAuditPipelineFixture(t, true, false, false)
	fixture.replaceManifest(t, []byte("{"))
	report, err := Run(context.Background(), fixture.options())
	if err != nil {
		t.Fatal(err)
	}
	assertAuditPipelineFailure(t, report, "tool_manifest", "tool_manifest_invalid", false)
	fixture.assertReportFile(t, report)
	fixture.assertInputs(t)
}

func TestAuditPipelineContractMandatoryBindingsRefuseBeforeCommands(t *testing.T) {
	for _, missing := range []string{"go", "git", "gofmt"} {
		t.Run(missing, func(t *testing.T) {
			fixture := newAuditPipelineFixture(t, true, false, false)
			fixture.removeTool(t, missing)
			report, err := run(context.Background(), fixture.options(), fixture.command(t))
			if err != nil || fixture.totalCalls != 0 {
				t.Fatalf("missing binding reached commands: calls=%d err=%v", fixture.totalCalls, err)
			}
			assertAuditPipelineFailure(t, report, "tool_manifest", "required_tool_missing", true)
			fixture.assertReportFile(t, report)
			fixture.assertInputs(t)
		})
	}
}

func TestAuditPipelineContractHelperFailurePreservesExistingFile(t *testing.T) {
	fixture := newAuditPipelineFixture(t, true, false, false)
	helper := filepath.Join(fixture.output, "helpers")
	auditPipelineWrite(t, helper, []byte("retained helper destination"))
	fixture.rememberInput(t, helper, []byte("retained helper destination"))
	report, err := run(context.Background(), fixture.options(), fixture.command(t))
	if err != nil || fixture.totalCalls != 0 {
		t.Fatalf("helper refusal reached commands: calls=%d err=%v", fixture.totalCalls, err)
	}
	assertAuditPipelineFailure(t, report, "tool_manifest", "helper_resolution_invalid", true)
	fixture.assertReportFile(t, report)
	fixture.assertInputs(t)
}

func TestAuditPipelineContractSourceRefusalStopsBeforeTargetTools(t *testing.T) {
	fixture := newAuditPipelineFixture(t, true, false, false)
	fixture.bare = true
	report, err := run(context.Background(), fixture.options(), fixture.command(t))
	if err != nil {
		t.Fatal(err)
	}
	assertAuditPipelineFailure(t, report, "source_identity_before_checks", "source_identity_invalid", true)
	fixture.assertNoTargets(t)
	fixture.assertReportFile(t, report)
	fixture.assertInputs(t)
}

func TestAuditPipelineContractEmptyGoScopeRetainsBoundIdentity(t *testing.T) {
	fixture := newAuditPipelineFixture(t, false, false, false)
	report, err := run(context.Background(), fixture.options(), fixture.command(t))
	if err != nil || report.Source == nil || report.Commit != strings.Repeat("a", 40) {
		t.Fatalf("empty scope lost its bound fixture identity: err=%v", err)
	}
	assertAuditPipelineCheck(t, report, "source_go_scope", "failed", "source_scope_missing", true)
	if report.Passed || report.ProductionQualified || report.Coverage != (Coverage{}) {
		t.Fatal("empty source scope became a qualified result")
	}
	fixture.assertNoTargets(t)
	fixture.assertReportFile(t, report)
	fixture.assertInputs(t)
}

func TestAuditPipelineContractSyntheticControlAndTaggedHold(t *testing.T) {
	for _, tagged := range []bool{false, true} {
		t.Run(fmt.Sprintf("tagged=%t", tagged), func(t *testing.T) {
			fixture := newAuditPipelineFixture(t, true, tagged, true)
			report, err := run(context.Background(), fixture.options(), fixture.command(t))
			if err != nil {
				t.Fatal(err)
			}
			fixture.assertCompletedPipeline(t, report, !tagged)
			if tagged {
				assertAuditPipelineCheck(t, report, taggedIntegrationCheckName, "not_run", taggedIntegrationCheckCode, true)
			}
			fixture.assertReportFile(t, report)
			fixture.assertInputs(t)
		})
	}
}

func TestAuditPipelineContractReportWriteErrorsRetainState(t *testing.T) {
	for _, late := range []bool{false, true} {
		t.Run(fmt.Sprintf("after-checks=%t", late), func(t *testing.T) {
			fixture := newAuditPipelineFixture(t, true, false, late)
			if late {
				fixture.blockReport = true
			} else {
				fixture.replaceManifest(t, []byte("{"))
				fixture.obstructReport(t)
			}
			report, err := run(context.Background(), fixture.options(), fixture.command(t))
			if err == nil || err.Error() != "private audit report write failed" {
				t.Fatalf("report write failure was not returned: %v", err)
			}
			assertAuditPipelineReturnedWriteFailure(t, fixture, report, late)
			fixture.assertInputs(t)
		})
	}
}

func newAuditPipelineFixture(t *testing.T, withGo, tagged, buildInfo bool) *auditPipelineFixture {
	t.Helper()
	fixture := &auditPipelineFixture{
		root:      privateAssuranceDirectory(t, "source"),
		output:    privateAssuranceDirectory(t, "evidence"),
		files:     map[string][]byte{},
		blobs:     map[string][]byte{},
		inputs:    map[string]auditPipelineInput{},
		toolPaths: map[string]string{},
		calls:     map[string]int{},
	}
	fixture.createSource(t, withGo, tagged)
	fixture.createTools(t, buildInfo)
	fixture.saveManifest(t)
	return fixture
}

func (fixture *auditPipelineFixture) createSource(t *testing.T, withGo, tagged bool) {
	t.Helper()
	for _, name := range requiredFiles {
		fixture.files[name] = []byte("fabricated source metadata\n")
	}
	fixture.files["go.mod"] = []byte("module github.com/maldous/CircularQuay\n")
	if withGo {
		fixture.files["value.go"] = []byte("package main\nfunc main() { _ = 1 }\n")
	}
	if tagged {
		fixture.files["internal/integration/fixture_test.go"] = []byte("//go:build integration\n\npackage integration\n")
	}
	for index, name := range fixture.sourceNames() {
		data := fixture.files[name]
		path := filepath.Join(fixture.root, filepath.FromSlash(name))
		auditPipelineWrite(t, path, data)
		fixture.rememberInput(t, path, data)
		// These finite labels are synthetic Git identities, not actual commits.
		fixture.blobs[fmt.Sprintf("%040x", index+1)] = data
	}
}

func (fixture *auditPipelineFixture) createTools(t *testing.T, buildInfo bool) {
	t.Helper()
	directory := ensurePrivateDirectory(t, filepath.Join(privateAssuranceDirectory(t, "tools"), "bin"))
	var image []byte
	if buildInfo {
		image = auditPipelineOwnImage(t)
	}
	fixture.declaration = ToolManifest{Version: toolManifestVersion, Platform: runtime.GOOS + "/" + runtime.GOARCH}
	for _, id := range []string{"git", "go", "gofmt"} {
		data := []byte("fabricated tool " + id + "\n")
		if buildInfo && id != "git" {
			data = image
		}
		path := auditPipelineExclusiveTool(t, filepath.Join(directory, canonicalToolName(id)), data)
		fixture.toolPaths[path] = id
		fixture.rememberInput(t, path, data)
		fixture.declaration.Tools = append(fixture.declaration.Tools, ToolEntry{
			ID: id, Path: path, SHA256: digest(data), Version: "synthetic fixture only",
			VersionOutputSHA256: digest(auditPipelineVersion(id)),
		})
	}
}

func auditPipelineOwnImage(t *testing.T) []byte {
	t.Helper()
	// Owned copies of this test image satisfy the real build-info relationship
	// checked for go/gofmt. Neither copy is ever executed; no PATH/toolchain lookup
	// or compiler invocation occurs, and probe results remain explicit fixtures.
	name, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	name, err = filepath.EvalSymlinks(name)
	if err != nil {
		t.Fatal(err)
	}
	file, err := os.Open(name)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	before := auditPipelineImageInfo(t, file)
	data := auditPipelineReadImage(t, file, before)
	named, err := os.Lstat(name)
	if err != nil || !os.SameFile(before, named) {
		t.Fatal("owned test-image name changed during the bounded read")
	}
	return data
}

func auditPipelineImageInfo(t *testing.T, file *os.File) os.FileInfo {
	t.Helper()
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Size() <= 0 || info.Size() > maxToolBinaryBytes {
		t.Fatal("owned test image is not a bounded regular file")
	}
	return info
}

func auditPipelineReadImage(t *testing.T, file *os.File, before os.FileInfo) []byte {
	t.Helper()
	data, err := io.ReadAll(io.LimitReader(file, before.Size()+1))
	if err != nil || int64(len(data)) != before.Size() {
		t.Fatal("owned test-image bytes differ from its descriptor size")
	}
	after := auditPipelineImageInfo(t, file)
	if !os.SameFile(before, after) || after.Size() != before.Size() || !after.ModTime().Equal(before.ModTime()) {
		t.Fatal("owned test-image descriptor changed during the bounded read")
	}
	return data
}

func auditPipelineExclusiveTool(t *testing.T, path string, data []byte) string {
	t.Helper()
	if len(data) == 0 || int64(len(data)) > maxToolBinaryBytes {
		t.Fatal("owned tool fixture exceeds the executable byte bound")
	}
	// Protect an empty owned reservation through the document writer, retaining
	// its Unix/Windows controls without treating executable bytes as a document.
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		t.Fatal(err)
	}
	if err = file.Close(); err != nil {
		t.Fatal(err)
	}
	auditPipelineWrite(t, path, nil)
	auditPipelinePopulateTool(t, path, data)
	if err = os.Chmod(path, 0700); err != nil {
		t.Fatal(err)
	}
	return path
}

func auditPipelinePopulateTool(t *testing.T, path string, data []byte) {
	t.Helper()
	file, err := os.OpenFile(path, os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	if count, err := file.Write(data); err != nil || count != len(data) {
		t.Fatal("owned executable fixture write was incomplete")
	}
	if err = file.Sync(); err != nil {
		t.Fatal(err)
	}
	if err = file.Close(); err != nil {
		t.Fatal(err)
	}
}

func auditPipelineWrite(t *testing.T, path string, data []byte) {
	t.Helper()
	if err := platform.EnsurePrivateDirectory(filepath.Dir(path)); err != nil {
		t.Fatal(err)
	}
	if err := platform.WriteProtected(path, data); err != nil {
		t.Fatal(err)
	}
}

func (fixture *auditPipelineFixture) saveManifest(t *testing.T) {
	t.Helper()
	data, err := json.Marshal(fixture.declaration)
	if err != nil {
		t.Fatal(err)
	}
	if fixture.manifest == "" {
		fixture.manifest = filepath.Join(privateAssuranceDirectory(t, "manifest"), "tools.json")
	}
	fixture.replaceManifest(t, data)
}

func (fixture *auditPipelineFixture) replaceManifest(t *testing.T, data []byte) {
	t.Helper()
	auditPipelineWrite(t, fixture.manifest, data)
	fixture.rememberInput(t, fixture.manifest, data)
}

func (fixture *auditPipelineFixture) removeTool(t *testing.T, missing string) {
	t.Helper()
	var selected []ToolEntry
	for _, entry := range fixture.declaration.Tools {
		if entry.ID != missing {
			selected = append(selected, entry)
		}
	}
	fixture.declaration.Tools = selected
	fixture.saveManifest(t)
}

func (fixture *auditPipelineFixture) options() Options {
	return Options{Source: fixture.root, Output: fixture.output, Profile: "source", ToolManifest: fixture.manifest}
}

func (fixture *auditPipelineFixture) command(t *testing.T) commandFunc {
	t.Helper()
	return func(_ context.Context, binary string, args []string, directory string, _ []string) commandResult {
		fixture.totalCalls++
		if fixture.totalCalls > 256 || directory != fixture.root {
			t.Fatal("production pipeline left its finite command fixture")
		}
		id, found := fixture.toolPaths[binary]
		if !found {
			t.Fatalf("unbound synthetic command path: %q", binary)
		}
		switch id {
		case "git":
			return fixture.git(t, args)
		case "go":
			return fixture.goCommand(t, args)
		case "gofmt":
			return fixture.formatCommand(t, args)
		default:
			t.Fatal("unexpected fixture tool")
			return commandResult{}
		}
	}
}

func auditPipelineVersion(id string) []byte {
	if id == "go" {
		return []byte("go version " + expectedGoVersion + " synthetic/fixture\n")
	}
	return []byte("pipeline fixture version " + id + "\n")
}

func (fixture *auditPipelineFixture) git(t *testing.T, args []string) commandResult {
	t.Helper()
	if reflect.DeepEqual(args, []string{"--version"}) {
		return commandResult{output: auditPipelineVersion("git")}
	}
	selected := auditPipelineGitArguments(t, args)
	switch selected[0] {
	case "rev-parse":
		if len(selected) != 2 {
			t.Fatal("unexpected synthetic revision arguments")
		}
		return commandResult{output: []byte(fixture.revision(t, selected[1]))}
	case "ls-tree":
		if !reflect.DeepEqual(selected, []string{"ls-tree", "-r", "-z", "HEAD"}) {
			t.Fatal("unexpected synthetic tree arguments")
		}
		return commandResult{output: fixture.treeRecords()}
	case "cat-file":
		return fixture.blobResult(t, selected)
	default:
		t.Fatal("unexpected synthetic Git request")
		return commandResult{}
	}
}

func auditPipelineGitArguments(t *testing.T, args []string) []string {
	t.Helper()
	prefix := []string{"-c", "core.hooksPath=/dev/null", "-c", "core.fsmonitor=false"}
	if len(args) < 5 || !reflect.DeepEqual(args[:4], prefix) {
		t.Fatal("source identity omitted its fixed Git command fence")
	}
	return args[4:]
}

func (fixture *auditPipelineFixture) blobResult(t *testing.T, args []string) commandResult {
	t.Helper()
	if len(args) != 3 || args[1] != "blob" {
		t.Fatal("unexpected synthetic blob arguments")
	}
	data, found := fixture.blobs[args[2]]
	if !found {
		t.Fatal("unbound synthetic Git object")
	}
	return commandResult{output: data}
}

func (fixture *auditPipelineFixture) revision(t *testing.T, selector string) string {
	t.Helper()
	switch selector {
	case "--is-bare-repository":
		return fmt.Sprintf("%t\n", fixture.bare)
	case "--show-toplevel":
		return fixture.root + "\n"
	case "--show-prefix":
		return "\n"
	case "HEAD":
		return strings.Repeat("a", 40) + "\n"
	case "HEAD^{tree}":
		return strings.Repeat("b", 40) + "\n"
	default:
		t.Fatal("unexpected synthetic revision selector")
		return ""
	}
}

func (fixture *auditPipelineFixture) sourceNames() []string {
	var names []string
	for name := range fixture.files {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func (fixture *auditPipelineFixture) treeRecords() []byte {
	var records bytes.Buffer
	for index, name := range fixture.sourceNames() {
		fmt.Fprintf(&records, "100644 blob %040x\t%s\x00", index+1, name)
	}
	return records.Bytes()
}

func (fixture *auditPipelineFixture) goCommand(t *testing.T, args []string) commandResult {
	t.Helper()
	if reflect.DeepEqual(args, []string{"version"}) {
		return commandResult{output: auditPipelineVersion("go")}
	}
	if reflect.DeepEqual(args, []string{"vet", "."}) {
		fixture.calls["vet"]++
		if fixture.blockReport {
			fixture.obstructReport(t)
		}
		return commandResult{}
	}
	want := []string{"test", "-count=1", "-covermode=atomic", "-coverpkg=" + strings.TrimSuffix(modulePrefix, "/"), "-coverprofile=" + filepath.Join(fixture.output, "coverage.out"), "."}
	if !reflect.DeepEqual(args, want) {
		t.Fatal("Go check left the fixture's exact source package scope")
	}
	fixture.calls["test"]++
	// This one-statement parser fixture is not measured candidate coverage.
	// Only the later hosted Source run may measure coverage of this new test.
	profile := "mode: atomic\n" + modulePrefix + "value.go:2.15,2.20 1 1\n"
	auditPipelineWrite(t, filepath.Join(fixture.output, "coverage.out"), []byte(profile))
	return commandResult{output: []byte("synthetic tool detail must not be persisted")}
}

func (fixture *auditPipelineFixture) formatCommand(t *testing.T, args []string) commandResult {
	t.Helper()
	if reflect.DeepEqual(args, []string{"-h"}) {
		return commandResult{output: auditPipelineVersion("gofmt")}
	}
	want := []string{"-l"}
	for _, name := range fixture.sourceNames() {
		if strings.HasSuffix(name, ".go") {
			want = append(want, filepath.Join(fixture.root, filepath.FromSlash(name)))
		}
	}
	if !reflect.DeepEqual(args, want) {
		t.Fatal("formatter did not receive the manifest's exact source files")
	}
	fixture.calls["format"]++
	return commandResult{}
}

func (fixture *auditPipelineFixture) obstructReport(t *testing.T) {
	t.Helper()
	directory := filepath.Join(fixture.output, "report.json")
	if err := os.Mkdir(directory, 0700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(directory, "retained")
	data := []byte("original output must survive refusal")
	auditPipelineWrite(t, path, data)
	fixture.rememberInput(t, path, data)
}

func assertAuditPipelineFailure(t *testing.T, report Report, name, code string, manifest bool) {
	t.Helper()
	expected := Report{
		Version: 1, Profile: "source", Platform: runtime.GOOS + "/" + runtime.GOARCH,
		StartedAt: report.StartedAt, CompletedAt: report.CompletedAt, Limitations: report.Limitations,
		Checks: []Check{{Name: name, Required: true, Status: "failed", Code: code}},
	}
	if manifest {
		expected.ToolManifest = report.ToolManifest
		if report.ToolManifest == nil || report.ToolManifest.SHA256 == "" {
			t.Fatal("bound manifest refusal lost its identity")
		}
	}
	if !reflect.DeepEqual(report, expected) || report.StartedAt.IsZero() || report.CompletedAt.IsZero() {
		t.Fatal("failed pipeline changed the whole expected report shape")
	}
}

func assertAuditPipelineCheck(t *testing.T, report Report, name, status, code string, required bool) {
	t.Helper()
	var selected []Check
	for _, check := range report.Checks {
		if check.Name == name {
			selected = append(selected, check)
		}
	}
	if len(selected) != 1 || selected[0].Status != status || selected[0].Code != code || selected[0].Required != required {
		t.Fatalf("unexpected %s fixture check", name)
	}
}

func (fixture *auditPipelineFixture) assertCompletedPipeline(t *testing.T, report Report, passed bool) {
	t.Helper()
	if report.Passed != passed || report.ProductionQualified || report.Coverage != (Coverage{Covered: 1, Total: 1, Percent: 100, Files: 1}) {
		t.Fatal("synthetic pipeline changed result, coverage fixture, or nonproduction boundary")
	}
	assertAuditPipelineBoundIdentity(t, report)
	for _, name := range []string{"format", "unit_tests", "statement_coverage", "go_vet", "source_identity_before_checks", "source_identity_after_checks"} {
		assertAuditPipelineCheck(t, report, name, "passed", "", true)
	}
	if fixture.calls["format"] != 1 || fixture.calls["test"] != 1 || fixture.calls["vet"] != 1 {
		t.Fatal("synthetic pipeline skipped or repeated a target check")
	}
}

func assertAuditPipelineBoundIdentity(t *testing.T, report Report) {
	t.Helper()
	if report.Source == nil || report.Commit != strings.Repeat("a", 40) || report.GoVersion != expectedGoVersion || report.ToolManifest == nil {
		t.Fatal("completed pipeline lost a declared source or tool identity")
	}
}

func assertAuditPipelineReturnedWriteFailure(t *testing.T, fixture *auditPipelineFixture, report Report, late bool) {
	t.Helper()
	if late {
		fixture.assertCompletedPipeline(t, report, true)
	} else {
		assertAuditPipelineFailure(t, report, "tool_manifest", "tool_manifest_invalid", false)
		fixture.assertNoTargets(t)
	}
	info, err := os.Stat(filepath.Join(fixture.output, "report.json"))
	if err != nil || !info.IsDir() {
		t.Fatal("report refusal replaced the existing output directory")
	}
}

func (fixture *auditPipelineFixture) assertNoTargets(t *testing.T) {
	t.Helper()
	if fixture.calls["format"] != 0 || fixture.calls["test"] != 0 || fixture.calls["vet"] != 0 {
		t.Fatal("a refused pipeline reached target tools")
	}
}

func (fixture *auditPipelineFixture) assertReportFile(t *testing.T, report Report) {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(fixture.output, "report.json"))
	if err != nil {
		t.Fatal(err)
	}
	var decoded Report
	if json.Unmarshal(data, &decoded) != nil {
		t.Fatal("pipeline did not persist a complete report")
	}
	want, wantErr := json.Marshal(report)
	got, gotErr := json.Marshal(decoded)
	if wantErr != nil || gotErr != nil || !bytes.Equal(want, got) || bytes.Contains(data, []byte("synthetic tool detail must not be persisted")) {
		t.Fatal("persisted report differs or retains raw synthetic tool output")
	}
}

func (fixture *auditPipelineFixture) assertInputs(t *testing.T) {
	t.Helper()
	for path, expected := range fixture.inputs {
		data, err := os.ReadFile(path)
		if err != nil || digest(data) != expected.sha256 {
			t.Fatal("production pipeline changed a retained fixture input or output")
		}
		info, err := os.Lstat(path)
		if err != nil || !os.SameFile(info, expected.info) || info.Mode() != expected.info.Mode() {
			t.Fatal("production pipeline replaced or re-permissioned a retained fixture")
		}
	}
}

func (fixture *auditPipelineFixture) rememberInput(t *testing.T, path string, data []byte) {
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
	named, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() || !os.SameFile(info, named) || info.Mode() != named.Mode() {
		t.Fatalf("owned fixture input is not a stable regular file: %v", err)
	}
	fixture.inputs[path] = auditPipelineInput{sha256: digest(data), info: info}
}
