package assurance

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/maldous/CircularQuay/internal/platform"
)

func TestToolManifestSchemaAndParserAgreeOnV1Structure(t *testing.T) {
	manifest := manifestFixture(t)
	assertSchemaAndParserAccept(t, manifest)

	for _, mutate := range []func(*ToolManifest){
		func(value *ToolManifest) { value.Tools[0].ID = "unknown" },
		func(value *ToolManifest) { value.Tools[0].ID = "GIT" },
		func(value *ToolManifest) { value.Tools[1].ID = "git" },
		func(value *ToolManifest) { value.Tools[0].Path = "relative-tool" },
		func(value *ToolManifest) { value.Tools[3].Rules = nil },
		func(value *ToolManifest) { value.Tools[0].Rules = &ToolRules{RulesetSHA256: digest([]byte("rules"))} },
		func(value *ToolManifest) { value.Tools[8].Rules.ServerVersion = "" },
		func(value *ToolManifest) { value.Tools[0].Version = "line\nbreak" },
	} {
		candidate := cloneManifest(t, manifest)
		mutate(&candidate)
		data, err := json.Marshal(candidate)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := ParseToolManifest(data); err == nil {
			t.Fatal("parser accepted structurally invalid manifest")
		}
		if err := validateToolManifestSchema(t, data); err == nil {
			t.Fatal("schema accepted structurally invalid manifest")
		}
	}
}

func TestToolManifestRejectsExplicitEmptyForbiddenRuleFields(t *testing.T) {
	for _, key := range []string{"server_version", "active_rules_sha256", "profile_sha256", "gate_sha256"} {
		value := manifestJSONValue(t, manifestFixture(t))
		tools := value["tools"].([]any)
		tool := tools[3].(map[string]any)
		tool["rules"].(map[string]any)[key] = ""
		assertSchemaAndParserRejectValue(t, value)
	}
	value := manifestJSONValue(t, manifestFixture(t))
	value["tools"].([]any)[0].(map[string]any)["rules"] = nil
	assertSchemaAndParserRejectValue(t, value)
}

func TestToolManifestRejectsControlCharacters(t *testing.T) {
	for _, control := range []string{"\n", "\r", "\t", "\x7f"} {
		value := manifestJSONValue(t, manifestFixture(t))
		tools := value["tools"].([]any)
		tools[0].(map[string]any)["version"] = "fixture" + control
		assertSchemaAndParserRejectValue(t, value)

		value = manifestJSONValue(t, manifestFixture(t))
		tools = value["tools"].([]any)
		tools[0].(map[string]any)["path"] = "/private" + control + "tool"
		assertSchemaAndParserRejectValue(t, value)
	}
	value := manifestJSONValue(t, manifestFixture(t))
	value["tools"].([]any)[0].(map[string]any)["version"] = "é"
	assertSchemaAndParserRejectValue(t, value)
	for _, label := range []string{"", "   "} {
		value := manifestJSONValue(t, manifestFixture(t))
		value["tools"].([]any)[0].(map[string]any)["version"] = label
		assertSchemaAndParserRejectValue(t, value)
	}
}

func TestToolManifestPortablePathSyntaxIsNotAHostOpen(t *testing.T) {
	manifest := manifestFixture(t)
	manifest.Tools[0].Path = `C:\operator-tools\git.exe`
	assertSchemaAndParserAccept(t, manifest)
	if _, err := LoadToolManifest(writeManifest(t, manifest), fixtureLocations(t)); err == nil {
		t.Fatal("non-local portable declaration opened as a local tool")
	}
	for _, path := range []string{`\\server\share\tool.exe`, `\\?\C:\tool.exe`, `//server/tool`} {
		candidate := cloneManifest(t, manifest)
		candidate.Tools[0].Path = path
		data, err := json.Marshal(candidate)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = ParseToolManifest(data); err == nil {
			t.Fatalf("non-local path %q accepted", path)
		}
	}
}

func TestToolManifestPurePathGuardsRejectRemoteAndDeviceSyntax(t *testing.T) {
	for _, path := range []string{`\\server\share\tool.exe`, `\\?\C:\tool.exe`, `\\.\C:\tool.exe`, `//server/tool`, "/tool\tname"} {
		if validPortableToolPath(path) || localAbsoluteToolPath(path) {
			t.Fatalf("remote or device path passed a pure guard: %q", path)
		}
	}
}

func TestToolManifestAllowsSelectedKnownToolSubset(t *testing.T) {
	manifest := manifestFixture(t)
	manifest.Tools = manifest.Tools[:1]
	assertSchemaAndParserAccept(t, manifest)
	if _, err := LoadToolManifest(writeManifest(t, manifest), fixtureLocations(t)); err != nil {
		t.Fatalf("selected known subset rejected: %v", err)
	}
}

func TestToolManifestSchemaAndParserRejectWrongRuleKinds(t *testing.T) {
	for _, mutation := range []struct {
		index int
		kind  string
	}{{3, "file"}, {4, "sonar_remote"}, {5, "embedded"}, {6, "vulndb_snapshot"}, {7, "file"}, {8, "embedded"}} {
		value := manifestJSONValue(t, manifestFixture(t))
		tools := value["tools"].([]any)
		tools[mutation.index].(map[string]any)["rules"].(map[string]any)["kind"] = mutation.kind
		assertSchemaAndParserRejectValue(t, value)
	}
}

func TestToolManifestRejectsForbiddenNullRuleInputs(t *testing.T) {
	for _, index := range []int{3, 8} {
		value := manifestJSONValue(t, manifestFixture(t))
		tools := value["tools"].([]any)
		tools[index].(map[string]any)["rules"].(map[string]any)["input"] = nil
		assertSchemaAndParserRejectValue(t, value)
	}
}

func TestLoadToolManifestRejectsUnsafeRuleInputs(t *testing.T) {
	t.Run("source contained file", func(t *testing.T) {
		manifest, locations := manifestFixture(t), fixtureLocations(t)
		input := manifest.Tools[6].Rules.Input
		input.Path = writeRuleInput(t, filepath.Join(locations.Source, "rules.toml"), "rules")
		input.SHA256 = digest([]byte("rules"))
		manifest.Tools[6].Rules.RulesetSHA256 = input.SHA256
		assertManifestRejected(t, writeManifest(t, manifest), locations)
	})
	t.Run("snapshot drift", func(t *testing.T) {
		manifest := manifestFixture(t)
		path := filepath.Join(manifest.Tools[5].Rules.Input.Path, "index.json")
		if err := platform.WriteProtected(path, []byte("changed")); err != nil {
			t.Fatal(err)
		}
		assertManifestRejected(t, writeManifest(t, manifest), fixtureLocations(t))
	})
	t.Run("empty snapshot", func(t *testing.T) {
		manifest := manifestFixture(t)
		path := filepath.Join(manifest.Tools[5].Rules.Input.Path, "index.json")
		if err := os.Remove(path); err != nil {
			t.Fatal(err)
		}
		assertManifestRejected(t, writeManifest(t, manifest), fixtureLocations(t))
	})
}

func TestLoadToolManifestBindsProtectedExternalTools(t *testing.T) {
	manifest := manifestFixture(t)
	path := writeManifest(t, manifest)
	verified, err := LoadToolManifest(path, fixtureLocations(t))
	if err != nil {
		t.Fatalf("load protected fixture: %v", err)
	}
	if len(verified.Tools) != len(toolIDs) || verified.Tools[0].SHA256 != manifest.Tools[0].SHA256 {
		t.Fatalf("unexpected verified identity: %+v", verified)
	}
}

func TestLoadToolManifestRejectsUnsafeBindings(t *testing.T) {
	t.Run("source contained manifest", rejectSourceContainedManifest)
	t.Run("output contained tool", rejectOutputContainedTool)
	t.Run("binary digest mismatch", rejectToolDigestMismatch)
	t.Run("canonical duplicate tool", rejectDuplicateToolPath)
	t.Run("unprotected manifest", rejectUnprotectedManifest)
	t.Run("symbolic tool", rejectSymbolicTool)
	t.Run("symbolic tool ancestor", rejectSymbolicToolAncestor)
}

func rejectSourceContainedManifest(t *testing.T) {
	manifest, locations := manifestFixture(t), fixtureLocations(t)
	path := filepath.Join(locations.Source, "tools.json")
	writeManifestAt(t, path, manifest)
	assertManifestRejected(t, path, locations)
}

func rejectOutputContainedTool(t *testing.T) {
	manifest, locations := manifestFixture(t), fixtureLocations(t)
	manifest.Tools[0].Path = writeTool(t, filepath.Join(locations.Output, "git"), "tool-git")
	manifest.Tools[0].SHA256 = digest([]byte("tool-git"))
	assertManifestRejected(t, writeManifest(t, manifest), locations)
}

func rejectToolDigestMismatch(t *testing.T) {
	manifest := manifestFixture(t)
	manifest.Tools[0].SHA256 = digest([]byte("other"))
	assertManifestRejected(t, writeManifest(t, manifest), fixtureLocations(t))
}

func rejectDuplicateToolPath(t *testing.T) {
	manifest := manifestFixture(t)
	manifest.Tools[1].Path, manifest.Tools[1].SHA256 = manifest.Tools[0].Path, manifest.Tools[0].SHA256
	assertManifestRejected(t, writeManifest(t, manifest), fixtureLocations(t))
}

func rejectUnprotectedManifest(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("native ACL fixture requires an explicit non-owner ACE")
	}
	path := filepath.Join(privateAssuranceDirectory(t, "unprotected-manifest"), "tools.json")
	data, err := json.Marshal(manifestFixture(t))
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(path, data, 0644); err != nil {
		t.Fatal(err)
	}
	if err = os.Chmod(path, 0644); err != nil {
		t.Fatal(err)
	}
	assertManifestRejected(t, path, fixtureLocations(t))
}

func rejectSymbolicTool(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symbolic-link creation needs native test privileges")
	}
	manifest := manifestFixture(t)
	link := filepath.Join(privateAssuranceDirectory(t, "tool-link"), "git")
	if err := os.Symlink(manifest.Tools[0].Path, link); err != nil {
		t.Fatal(err)
	}
	manifest.Tools[0].Path = link
	assertManifestRejected(t, writeManifest(t, manifest), fixtureLocations(t))
}

func rejectSymbolicToolAncestor(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symbolic-link creation needs native test privileges")
	}
	manifest := manifestFixture(t)
	link := filepath.Join(privateAssuranceDirectory(t, "tool-link-parent"), "bin")
	if err := os.Symlink(filepath.Dir(manifest.Tools[0].Path), link); err != nil {
		t.Fatal(err)
	}
	manifest.Tools[0].Path = filepath.Join(link, filepath.Base(manifest.Tools[0].Path))
	assertManifestRejected(t, writeManifest(t, manifest), fixtureLocations(t))
}

func TestLoadToolManifestDoesNotUseLexicalPathPrefixes(t *testing.T) {
	root := privateAssuranceDirectory(t, "prefix")
	locations := ToolLocations{
		Source: ensurePrivateDirectory(t, filepath.Join(root, "source")),
		Output: ensurePrivateDirectory(t, filepath.Join(root, "output")),
	}
	toolDir := ensurePrivateDirectory(t, filepath.Join(root, "source-tools"))
	manifestDir := ensurePrivateDirectory(t, filepath.Join(root, "manifest"))
	manifest := manifestWithTools(t, toolDir)
	if _, err := LoadToolManifest(writeManifestAt(t, filepath.Join(manifestDir, "tools.json"), manifest), locations); err != nil {
		t.Fatalf("sibling path with source prefix rejected: %v", err)
	}
}

func manifestFixture(t *testing.T) ToolManifest {
	t.Helper()
	return manifestWithTools(t, ensurePrivateDirectory(t, filepath.Join(privateAssuranceDirectory(t, "tools"), "bin")))
}

func manifestWithTools(t *testing.T, directory string) ToolManifest {
	t.Helper()
	tools := make([]ToolEntry, 0, len(toolIDs))
	for _, id := range toolIDs {
		content := []byte("synthetic tool " + id + "\n")
		path := writeTool(t, filepath.Join(directory, id), string(content))
		entry := ToolEntry{ID: id, Path: path, SHA256: digest(content), Version: "fixture-v1", VersionOutputSHA256: digest([]byte("version-output-" + id))}
		if requiresRules(id) {
			entry.Rules = fixtureRules(t, id, entry.SHA256, directory)
			if id == "sonar-scanner" {
				entry.Rules.ServerVersion = "fixture-server-v1"
				entry.Rules.ActiveRulesSHA256 = digest([]byte("active-rules"))
				entry.Rules.ProfileSHA256 = digest([]byte("profile"))
				entry.Rules.GateSHA256 = digest([]byte("gate"))
			}
		}
		tools = append(tools, entry)
	}
	return ToolManifest{Version: toolManifestVersion, Platform: runtime.GOOS + "/" + runtime.GOARCH, Tools: tools}
}

func fixtureRules(t *testing.T, id, binaryDigest, directory string) *ToolRules {
	t.Helper()
	if id == "staticcheck" || id == "gosec" || id == "actionlint" {
		return &ToolRules{Kind: "embedded", RulesetSHA256: binaryDigest}
	}
	if id == "gitleaks" {
		data := []byte("fixture gitleaks rules\n")
		path := writeTool(t, filepath.Join(filepath.Dir(directory), "gitleaks-rules.toml"), string(data))
		if err := os.Chmod(path, 0600); err != nil {
			t.Fatal(err)
		}
		return &ToolRules{Kind: "file", RulesetSHA256: digest(data), Input: &ToolInput{Path: path, SHA256: digest(data)}}
	}
	if id == "govulncheck" {
		return fixtureVulnDBRules(t, directory)
	}
	return &ToolRules{Kind: "sonar_remote", RulesetSHA256: digest([]byte("rules-" + id))}
}

func fixtureVulnDBRules(t *testing.T, directory string) *ToolRules {
	t.Helper()
	path := ensurePrivateDirectory(t, filepath.Join(filepath.Dir(directory), "vulndb"))
	if err := platform.WriteProtected(filepath.Join(path, "index.json"), []byte("fixture vulndb\n")); err != nil {
		t.Fatal(err)
	}
	files, err := protectedTreeFiles(path)
	if err != nil {
		t.Fatal(err)
	}
	digest, err := digestInputFiles(path, files)
	if err != nil {
		t.Fatal(err)
	}
	return &ToolRules{Kind: "vulndb_snapshot", RulesetSHA256: digest, Input: &ToolInput{Path: path, SHA256: digest}}
}

func fixtureLocations(t *testing.T) ToolLocations {
	t.Helper()
	root := privateAssuranceDirectory(t, "locations")
	return ToolLocations{
		Source: ensurePrivateDirectory(t, filepath.Join(root, "source")),
		Output: ensurePrivateDirectory(t, filepath.Join(root, "output")),
	}
}

func ensurePrivateDirectory(t *testing.T, path string) string {
	t.Helper()
	if err := platform.EnsurePrivateDirectory(path); err != nil {
		t.Fatal(err)
	}
	return path
}

func writeTool(t *testing.T, path, contents string) string {
	t.Helper()
	if err := platform.WriteProtected(path, []byte(contents)); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, 0700); err != nil {
		t.Fatal(err)
	}
	return path
}

func writeRuleInput(t *testing.T, path, contents string) string {
	t.Helper()
	if err := platform.WriteProtected(path, []byte(contents)); err != nil {
		t.Fatal(err)
	}
	return path
}

func writeManifest(t *testing.T, manifest ToolManifest) string {
	t.Helper()
	return writeManifestAt(t, filepath.Join(privateAssuranceDirectory(t, "manifest"), "tools.json"), manifest)
}

func writeManifestAt(t *testing.T, path string, manifest ToolManifest) string {
	t.Helper()
	data, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	if err = platform.WriteProtected(path, data); err != nil {
		t.Fatal(err)
	}
	return path
}

func assertManifestRejected(t *testing.T, path string, locations ToolLocations) {
	t.Helper()
	if _, err := LoadToolManifest(path, locations); err == nil {
		t.Fatal("unsafe manifest binding accepted")
	}
}

func assertSchemaAndParserAccept(t *testing.T, manifest ToolManifest) {
	t.Helper()
	data, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	if err = validateToolManifestSchema(t, data); err != nil {
		t.Fatalf("schema rejected valid manifest: %v", err)
	}
	if _, err = ParseToolManifest(data); err != nil {
		t.Fatalf("parser rejected valid manifest: %v", err)
	}
}

func assertSchemaAndParserRejectValue(t *testing.T, value map[string]any) {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	if err = validateToolManifestSchema(t, data); err == nil {
		t.Fatal("schema accepted invalid manifest")
	}
	if _, err = ParseToolManifest(data); err == nil {
		t.Fatal("parser accepted invalid manifest")
	}
}

func manifestJSONValue(t *testing.T, manifest ToolManifest) map[string]any {
	t.Helper()
	data, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	value := map[string]any{}
	if err = json.Unmarshal(data, &value); err != nil {
		t.Fatal(err)
	}
	return value
}

func validateToolManifestSchema(t *testing.T, data []byte) error {
	t.Helper()
	value := map[string]any{}
	if err := json.Unmarshal(data, &value); err != nil {
		t.Fatal(err)
	}
	return resolvedToolManifestSchema(t).Validate(value)
}

func cloneManifest(t *testing.T, manifest ToolManifest) ToolManifest {
	t.Helper()
	data, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	var clone ToolManifest
	if err = json.Unmarshal(data, &clone); err != nil {
		t.Fatal(err)
	}
	return clone
}

func resolvedToolManifestSchema(t *testing.T) *jsonschema.Resolved {
	t.Helper()
	var schema jsonschema.Schema
	if err := json.Unmarshal(ToolManifestSchema, &schema); err != nil {
		t.Fatal(err)
	}
	resolved, err := schema.Resolve(nil)
	if err != nil {
		t.Fatalf("tool manifest schema invalid: %v", err)
	}
	return resolved
}

func TestToolManifestDigestUsesSHA256(t *testing.T) {
	sum := sha256.Sum256([]byte("fixture"))
	if got, want := digest([]byte("fixture")), hex.EncodeToString(sum[:]); got != want {
		t.Fatalf("digest = %q, want %q", got, want)
	}
}

func TestProtectedTreeFilesEnforcesExactFileLimit(t *testing.T) {
	root := ensurePrivateDirectory(t, filepath.Join(privateAssuranceDirectory(t, "file-limit"), "snapshot"))
	for index := 0; index < maxToolInputFiles; index++ {
		writeRuleInput(t, filepath.Join(root, fmt.Sprintf("%04d", index)), "x")
	}
	files, err := protectedTreeFiles(root)
	if err != nil || len(files) != maxToolInputFiles {
		t.Fatalf("exact file limit rejected: files=%d err=%v", len(files), err)
	}
	writeRuleInput(t, filepath.Join(root, "overflow"), "x")
	if _, err = protectedTreeFiles(root); err == nil {
		t.Fatal("file count above exact limit accepted")
	}
}

func TestProtectedTreeFilesEnforcesEntryLimitAndEmptyTrees(t *testing.T) {
	root := ensurePrivateDirectory(t, filepath.Join(privateAssuranceDirectory(t, "entry-limit"), "snapshot"))
	for index := 0; index < maxToolInputEntries-1; index++ {
		if err := os.Mkdir(filepath.Join(root, fmt.Sprintf("%04d", index)), 0700); err != nil {
			t.Fatal(err)
		}
	}
	files, err := protectedTreeFiles(root)
	if err != nil || len(files) != 0 {
		t.Fatalf("exact entry limit rejected: files=%d err=%v", len(files), err)
	}
	if err = os.Mkdir(filepath.Join(root, "overflow"), 0700); err != nil {
		t.Fatal(err)
	}
	if _, err = protectedTreeFiles(root); err == nil {
		t.Fatal("entry count above exact limit accepted")
	}
	if _, err = bindToolInputTree(root, privateAssuranceDirectory(t, "source"), privateAssuranceDirectory(t, "output")); err == nil {
		t.Fatal("empty tree accepted as snapshot")
	}
}

func TestToolInputOverlapAndPostReadNameFence(t *testing.T) {
	root := privateAssuranceDirectory(t, "input-fence")
	if !pathsOverlap(filepath.Join(root, "inputs"), root) || pathsOverlap(filepath.Join(root, "one"), filepath.Join(root, "two")) {
		t.Fatal("path overlap relation incorrect")
	}
	path := writeRuleInput(t, filepath.Join(root, "input"), "before")
	file, err := openFixtureRuleInputForReplacement(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	before, err := file.Stat()
	if err != nil {
		t.Fatal(err)
	}
	if err = platform.WriteProtected(path, []byte("after!")); err != nil {
		t.Fatal(err)
	}
	if inputFileUnchanged(file, path, before, before.Size()) {
		t.Fatal("replaced named path accepted after descriptor read")
	}
}
