package assurance

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func manifestWithNativeHelper(t *testing.T) ToolManifest {
	t.Helper()
	manifest := manifestFixture(t)
	canonicalFixtureHelperNames(t, &manifest)
	path := filepath.Join(privateAssuranceDirectory(t, "native-helper"), canonicalToolName("ssh-keygen"))
	writeTool(t, path, "synthetic native helper\n")
	manifest.Helpers = []ToolHelper{{ID: "ssh-keygen", Path: path, SHA256: digest([]byte("synthetic native helper\n"))}}
	return manifest
}

func canonicalFixtureHelperNames(t *testing.T, manifest *ToolManifest) {
	t.Helper()
	for index, tool := range manifest.Tools {
		if tool.ID == "git" || tool.ID == "go" {
			path := filepath.Join(filepath.Dir(tool.Path), canonicalToolName(tool.ID))
			manifest.Tools[index].Path = writeTool(t, path, "synthetic tool "+tool.ID+"\n")
		}
	}
}

func TestNativeHelperSchemaAndParserContract(t *testing.T) {
	manifest := manifestWithNativeHelper(t)
	assertSchemaAndParserAccept(t, manifest)
	for _, mutate := range []func(map[string]any){
		func(value map[string]any) { value["helpers"] = nil },
		func(value map[string]any) { value["helpers"] = []any{} },
		func(value map[string]any) {
			value["helpers"] = append(value["helpers"].([]any), value["helpers"].([]any)[0])
		},
		func(value map[string]any) { firstHelper(value)["id"] = "SSH-KEYGEN" },
		func(value map[string]any) { firstHelper(value)["id"] = "arbitrary" },
		func(value map[string]any) { firstHelper(value)["path"] = "relative" },
		func(value map[string]any) { firstHelper(value)["path"] = "/private/line\nbreak" },
		func(value map[string]any) { firstHelper(value)["sha256"] = strings.Repeat("A", 64) },
		func(value map[string]any) { firstHelper(value)["version"] = "invented" },
		func(value map[string]any) { delete(firstHelper(value), "sha256") },
	} {
		value := manifestJSONValue(t, manifest)
		mutate(value)
		assertSchemaAndParserRejectValue(t, value)
	}
}

func firstHelper(value map[string]any) map[string]any {
	return value["helpers"].([]any)[0].(map[string]any)
}

func TestNativeHelperLoadingRejectsUnsafeIdentities(t *testing.T) {
	for _, scenario := range []string{"digest", "basename", "source", "output", "tool-path"} {
		t.Run(scenario, func(t *testing.T) {
			manifest := manifestWithNativeHelper(t)
			locations := fixtureLocations(t)
			mutateNativeHelper(t, &manifest, locations, scenario)
			if _, err := LoadToolManifest(writeManifest(t, manifest), locations); err == nil {
				t.Fatal("unsafe native dependency accepted")
			}
		})
	}
}

func mutateNativeHelper(t *testing.T, manifest *ToolManifest, locations ToolLocations, scenario string) {
	t.Helper()
	helper := &manifest.Helpers[0]
	switch scenario {
	case "digest":
		helper.SHA256 = digest([]byte("wrong"))
	case "basename":
		copy := filepath.Join(filepath.Dir(helper.Path), "other")
		writeTool(t, copy, "synthetic native helper\n")
		helper.Path = copy
	case "source", "output":
		directory := locations.Source
		if scenario == "output" {
			directory = locations.Output
		}
		helper.Path = writeTool(t, filepath.Join(directory, canonicalToolName(helper.ID)), "synthetic native helper\n")
	case "tool-path":
		manifest.Tools[0].Path, manifest.Tools[0].SHA256 = helper.Path, helper.SHA256
	}
}

func preparedNativeHelper(t *testing.T) (*qualifiedTools, []string) {
	t.Helper()
	manifest := manifestWithNativeHelper(t)
	verified, err := LoadToolManifest(writeManifest(t, manifest), fixtureLocations(t))
	if err != nil {
		t.Fatal(err)
	}
	tools := newQualifiedTools(verified, nil)
	environment, err := tools.environment([]string{"Path=untrusted", "GoRoot=untrusted", "PATH=also-untrusted", "SAFE=kept"}, privateAssuranceDirectory(t, "helper-copies"))
	if err != nil {
		t.Fatal(err)
	}
	return tools, environment
}

func TestDeclaredNativeHelperUsesOnlyPrivatePATHCopy(t *testing.T) {
	tools, environment := preparedNativeHelper(t)
	assertPrivateHelperEnvironment(t, environment)
	path, err := exec.LookPath("ssh-keygen")
	if err != nil || path != tools.helperCopies["ssh-keygen"] || path == tools.manifest.Helpers[0].Path {
		t.Fatalf("resolved helper = %q, %v", path, err)
	}
	assertPreparedHelperReport(t, tools)
}

func assertPrivateHelperEnvironment(t *testing.T, environment []string) {
	t.Helper()
	var paths, roots int
	for _, entry := range environment {
		name, value, _ := strings.Cut(entry, "=")
		if strings.EqualFold(name, "PATH") {
			paths++
			t.Setenv("PATH", value)
		}
		if strings.EqualFold(name, "GOROOT") {
			roots++
		}
		if strings.Contains(entry, "untrusted") {
			t.Fatal("ambient helper environment survived")
		}
	}
	if paths != 1 || roots != 1 {
		t.Fatal("private PATH/GOROOT is not unique")
	}
}

func assertPreparedHelperReport(t *testing.T, tools *qualifiedTools) {
	t.Helper()
	report := tools.report()
	if len(report.Helpers) != 1 || !report.Helpers[0].Prepared || !report.Helpers[0].IntegrityCurrent {
		t.Fatalf("helper preparation report = %+v", report.Helpers)
	}
	encoded, err := json.Marshal(report.Helpers)
	if err != nil || strings.Contains(string(encoded), "probe") || strings.Contains(string(encoded), "version") {
		t.Fatal("helper report invented a version or execution probe")
	}
	if _, exists := tools.tool("ssh-keygen"); exists {
		t.Fatal("native helper became an analyzer command")
	}
}

func TestSelectedNativeHelperMustBePreparedBeforeCommands(t *testing.T) {
	manifest := manifestWithNativeHelper(t)
	verified, err := LoadToolManifest(writeManifest(t, manifest), fixtureLocations(t))
	if err != nil {
		t.Fatal(err)
	}
	calls := 0
	tools := newQualifiedTools(verified, func(context.Context, string, []string, string, []string) commandResult {
		calls++
		return commandResult{}
	})
	result := tools.run(context.Background(), "git", []string{"status"}, t.TempDir(), nil)
	if result.exitCode == 0 || calls != 0 || tools.probes["git"].Code != "helper_digest_mismatch" {
		t.Fatal("unprepared native dependency permitted execution")
	}
}

func TestNativeHelperDriftFencesEveryCommandBoundary(t *testing.T) {
	for _, location := range []string{"source", "copy"} {
		for _, phase := range []string{"before", "probe", "command"} {
			t.Run(location+"-"+phase, func(t *testing.T) { assertNativeHelperDriftDenied(t, location, phase) })
		}
	}
}

func assertNativeHelperDriftDenied(t *testing.T, location, phase string) {
	t.Helper()
	tools, environment := preparedNativeHelper(t)
	path := tools.manifest.Helpers[0].Path
	if location == "copy" {
		path = tools.helperCopies["ssh-keygen"]
	}
	mutate := func() {
		if err := os.WriteFile(path, []byte("changed"), 0700); err != nil {
			t.Fatal(err)
		}
	}
	calls := 0
	tools.command = nativeHelperDriftCommand(phase, &calls, mutate)
	wantCalls, wantCode := 0, "helper_digest_mismatch"
	if phase == "before" {
		mutate()
	} else if phase == "probe" {
		wantCalls, wantCode = 1, "helper_changed_after_probe"
	} else {
		wantCalls, wantCode = 2, "helper_changed_during_execution"
	}
	result := tools.run(context.Background(), "git", []string{"status"}, t.TempDir(), environment)
	if result.exitCode == 0 || calls != wantCalls || tools.probes["git"].Code != wantCode || tools.report().Helpers[0].IntegrityCurrent {
		t.Fatalf("drift accepted: calls=%d probe=%+v result=%+v", calls, tools.probes["git"], result)
	}
}

func nativeHelperDriftCommand(phase string, calls *int, mutate func()) commandFunc {
	return func(context.Context, string, []string, string, []string) commandResult {
		*calls++
		if phase == "probe" && *calls == 1 || phase == "command" && *calls == 2 {
			mutate()
		}
		return commandResult{output: []byte("version-output-git")}
	}
}
