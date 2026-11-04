package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/maldous/CircularQuay/internal/model"
)

func TestConfigSetGetListAndEscapedPointer(t *testing.T) {
	config := configFixture(t)
	out := runConfig(t, config, nil, "config", "set", "--pointer", "/sources/fixture/path", "--value-json", `"renamed.json"`)
	if outputState(t, out) != "updated" {
		t.Fatal("set did not report update")
	}
	out = runConfig(t, config, nil, "config", "get", "--pointer", "/sources/fixture/path")
	var response struct {
		Value string `json:"value"`
	}
	if json.Unmarshal(out, &response) != nil || response.Value != "renamed.json" {
		t.Fatalf("get response: %s", out)
	}
	out = runConfig(t, config, nil, "config", "list", "--pointer", "/sources")
	var listed struct {
		Entries []string `json:"entries"`
	}
	if json.Unmarshal(out, &listed) != nil || len(listed.Entries) != 1 || listed.Entries[0] != "fixture" {
		t.Fatalf("list response: %s", out)
	}
	value := map[string]any{"fixture/name": "escaped"}
	got, err := jsonPointerGet(value, "/fixture~1name")
	if err != nil || got != "escaped" {
		t.Fatalf("escaped JSON pointer: %v, %v", got, err)
	}

}

func TestConfigInvalidMutationsLeavePolicyUnchanged(t *testing.T) {
	config := configFixture(t)
	original, err := os.ReadFile(config)
	if err != nil {
		t.Fatal(err)
	}
	assertConfigFailure(t, config, nil, "config", "delete", "--pointer", "/sources/fixture")
	assertConfigUnchanged(t, config, original)
	assertConfigFailure(t, config, nil, "config", "set", "--pointer", "", "--value-json", `{"version":1}`)
	assertConfigUnchanged(t, config, original)
	assertConfigFailure(t, config, nil, "config", "set", "--pointer", "/sources/~2bad", "--value-json", `1`)
	assertConfigUnchanged(t, config, original)
}

func TestConfigRejectsInvalidMutationPointersWithoutChangingPolicy(t *testing.T) {
	config := configFixture(t)
	cases := []struct {
		name string
		args []string
	}{
		{name: "invalid escape", args: []string{"config", "set", "--pointer", "/sources/~", "--value-json", `1`}},
		{name: "plus array index", args: []string{"config", "set", "--pointer", "/workloads/demo/bindings/+0", "--value-json", `{}`}},
		{name: "negative zero array index", args: []string{"config", "set", "--pointer", "/workloads/demo/bindings/-0", "--value-json", `{}`}},
		{name: "leading zero array index", args: []string{"config", "set", "--pointer", "/workloads/demo/bindings/00", "--value-json", `{}`}},
		{name: "out of range array index", args: []string{"config", "set", "--pointer", "/workloads/demo/bindings/1", "--value-json", `{}`}},
		{name: "append array index", args: []string{"config", "set", "--pointer", "/workloads/demo/bindings/-", "--value-json", `{}`}},
		{name: "root delete", args: []string{"config", "delete", "--pointer", ""}},
		{name: "missing map item", args: []string{"config", "delete", "--pointer", "/sources/missing"}},
		{name: "scalar parent", args: []string{"config", "set", "--pointer", "/version/value", "--value-json", `1`}},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			original, digest := configSnapshot(t, config)
			assertConfigFailure(t, config, nil, test.args...)
			assertConfigSnapshotUnchanged(t, config, original, digest)
		})
	}
}

func TestConfigRejectsInvalidMutationInputsWithoutChangingPolicy(t *testing.T) {
	config := configFixture(t)
	valueFile := filepath.Join(filepath.Dir(config), "value.json")
	writeProtectedFixture(t, valueFile, []byte(`1`))
	largeValueFile := filepath.Join(filepath.Dir(config), "large-value.json")
	writeProtectedFixture(t, largeValueFile, bytes.Repeat([]byte("0"), maximumConfigInput+1))
	cases := []struct {
		name string
		args []string
	}{
		{name: "malformed expected digest", args: []string{"config", "set", "--pointer", "/version", "--value-json", `1`, "--expect-sha256", "not-a-digest"}},
		{name: "stale expected digest", args: []string{"config", "set", "--pointer", "/version", "--value-json", `1`, "--expect-sha256", stringsRepeat("0", 64)}},
		{name: "both value flags", args: []string{"config", "set", "--pointer", "/version", "--value-json", `1`, "--value-file", valueFile}},
		{name: "delete value flag", args: []string{"config", "delete", "--pointer", "/sources/fixture", "--value-json", `1`}},
		{name: "replace unknown value flag", args: []string{"config", "replace", "--file", valueFile, "--value-json", `1`}},
		{name: "oversized value file", args: []string{"config", "set", "--pointer", "/version", "--value-file", largeValueFile}},
	}
	if runtime.GOOS != "windows" {
		unsafeValueFile := filepath.Join(filepath.Dir(config), "unsafe-value.json")
		writeProtectedFixture(t, unsafeValueFile, []byte(`1`))
		if err := os.Chmod(unsafeValueFile, 0666); err != nil {
			t.Fatal(err)
		}
		cases = append(cases, struct {
			name string
			args []string
		}{name: "unsafe value file", args: []string{"config", "set", "--pointer", "/version", "--value-file", unsafeValueFile}})
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			original, digest := configSnapshot(t, config)
			assertConfigFailure(t, config, nil, test.args...)
			assertConfigSnapshotUnchanged(t, config, original, digest)
		})
	}
}

func TestConfigCommandsDoNotReadDeclaredSources(t *testing.T) {
	config := configFixture(t)
	missingSource := filepath.Join(filepath.Dir(config), "provider-input-does-not-exist.json")
	pathJSON, err := json.Marshal(missingSource)
	if err != nil {
		t.Fatal(err)
	}
	out := runConfig(t, config, nil, "config", "set", "--pointer", "/sources/fixture/path", "--value-json", string(pathJSON))
	if outputState(t, out) != "updated" {
		t.Fatalf("set response: %s", out)
	}
	validation := runConfig(t, config, nil, "config", "validate")
	var response struct {
		Valid bool `json:"valid"`
	}
	if err := json.Unmarshal(validation, &response); err != nil || !response.Valid {
		t.Fatalf("validate response: %s", validation)
	}
	if len(runConfig(t, config, nil, "config", "get", "--pointer", "/sources/fixture")) == 0 {
		t.Fatal("get returned no source declaration")
	}
}

func TestConfigReadOnlyCommandsExposeOnlyTheRootMaster(t *testing.T) {
	directory := privateCLIDirectory(t)
	config := filepath.Join(directory, "policy.json")
	master := map[string]any{
		"version": 1,
		"sources": map[string]any{
			"root": map[string]any{"type": "json", "path": "root.json"},
		},
		"workloads": map[string]any{
			"demo": map[string]any{"mode": "offline", "bindings": []any{map[string]any{"name": "VALUE", "source": "root", "field": "VALUE", "allow_offline": true}}},
		},
		"composition": map[string]any{
			"conflicts": "reject",
			"includes": []any{map[string]any{
				"id": "personal", "namespace": "personal", "sha256": strings.Repeat("a", 64),
				"file": "missing-personal-fragment.json", "base_directory": directory, "visibility": "local",
			}},
		},
	}
	data, err := json.Marshal(master)
	if err != nil {
		t.Fatal(err)
	}
	writeProtectedFixture(t, config, data)

	listed := runConfig(t, config, nil, "config", "list", "--pointer", "/sources")
	var response struct {
		Entries []string `json:"entries"`
	}
	if err = json.Unmarshal(listed, &response); err != nil || strings.Join(response.Entries, ",") != "root" {
		t.Fatalf("root source declarations = %s", listed)
	}

	original, digest := configSnapshot(t, config)
	assertConfigFailure(t, config, nil, "config", "validate")
	assertConfigSnapshotUnchanged(t, config, original, digest)
	assertConfigFailure(t, config, nil, "config", "set", "--pointer", "/sources/root/path", "--value-json", `"changed.json"`)
	assertConfigSnapshotUnchanged(t, config, original, digest)
}

func TestConfigRejectsDuplicateOrOversizedInput(t *testing.T) {
	config := configFixture(t)
	assertConfigFailure(t, config, nil, "config", "set", "--pointer", "/version", "--value-json", `1`+string(make([]byte, 1<<20)))
	assertConfigFailure(t, config, bytes.NewReader([]byte(`{"version":1,"version":1}`)), "config", "replace", "--file", "-")
}

func TestConfigExpectedDigestAndStdinReplace(t *testing.T) {
	config := configFixture(t)
	out := runConfig(t, config, nil, "config", "get", "--pointer", "")
	var current struct {
		SHA string `json:"sha256"`
	}
	if json.Unmarshal(out, &current) != nil || current.SHA == "" {
		t.Fatalf("missing digest: %s", out)
	}
	assertConfigFailure(t, config, nil, "config", "set", "--pointer", "/version", "--value-json", `1`, "--expect-sha256", stringsRepeat("0", 64))
	replacement, err := os.ReadFile(config)
	if err != nil {
		t.Fatal(err)
	}
	out = runConfig(t, config, bytes.NewReader(replacement), "config", "replace", "--file", "-", "--expect-sha256", current.SHA)
	if outputState(t, out) != "updated" {
		t.Fatalf("replace response: %s", out)
	}
}

func TestConfigCRUDPreservesPolicyContractAndCompareAndSwap(t *testing.T) {
	config := configFixtureWithBindings(t)
	_, digest := configSnapshot(t, config)

	created := runConfig(t, config, bytes.NewReader([]byte(`{"type":"json","path":"spare.json"}`)),
		"config", "set", "--pointer", "/sources/spare", "--value-file", "-", "--expect-sha256", digest)
	if outputState(t, created) != "updated" {
		t.Fatalf("create source response: %s", created)
	}

	listedSources := runConfig(t, config, nil, "config", "list", "--pointer", "/sources")
	var sourceListing struct {
		Entries []string `json:"entries"`
	}
	if err := json.Unmarshal(listedSources, &sourceListing); err != nil || strings.Join(sourceListing.Entries, ",") != "fixture,spare" {
		t.Fatalf("source listing: %s", listedSources)
	}

	_, digest = configSnapshot(t, config)
	deleted := runConfig(t, config, nil, "config", "delete", "--pointer", "/sources/spare", "--expect-sha256", digest)
	if outputState(t, deleted) != "updated" {
		t.Fatalf("delete source response: %s", deleted)
	}

	listedBindings := runConfig(t, config, nil, "config", "list", "--pointer", "/workloads/demo/bindings")
	var bindingListing struct {
		Entries []string `json:"entries"`
	}
	if err := json.Unmarshal(listedBindings, &bindingListing); err != nil || strings.Join(bindingListing.Entries, ",") != "0,1" {
		t.Fatalf("binding listing: %s", listedBindings)
	}

	_, digest = configSnapshot(t, config)
	deleted = runConfig(t, config, nil, "config", "delete", "--pointer", "/workloads/demo/bindings/0", "--expect-sha256", digest)
	if outputState(t, deleted) != "updated" {
		t.Fatalf("delete binding response: %s", deleted)
	}
	remaining := runConfig(t, config, nil, "config", "get", "--pointer", "/workloads/demo/bindings/0")
	var binding struct {
		Value struct {
			Name string `json:"name"`
		} `json:"value"`
	}
	if err := json.Unmarshal(remaining, &binding); err != nil || binding.Value.Name != "SECOND" {
		t.Fatalf("remaining binding: %s", remaining)
	}
	runConfig(t, config, nil, "config", "validate")
}

func TestConfigReadOnlyOperationsLeavePolicyUntouchedAndRedactValidationFailure(t *testing.T) {
	config := configFixture(t)
	before, digest := configSnapshot(t, config)
	for _, args := range [][]string{
		{"config", "validate"},
		{"config", "get", "--pointer", "/workloads/demo"},
		{"config", "list", "--pointer", "/workloads"},
	} {
		runConfig(t, config, nil, args...)
	}
	assertConfigSnapshotUnchanged(t, config, before, digest)
	if _, err := os.Stat(filepath.Join(filepath.Dir(config), ".cquay-config-lock")); !os.IsNotExist(err) {
		t.Fatalf("read-only command created a mutation lock: %v", err)
	}

	marker := "SYNTHETIC_NEVER_ECHO_CONFIG_MARKER"
	data, err := os.ReadFile(config)
	if err != nil {
		t.Fatal(err)
	}
	var document map[string]any
	if err = json.Unmarshal(data, &document); err != nil {
		t.Fatal(err)
	}
	sources := document["sources"].(map[string]any)
	source := sources["fixture"].(map[string]any)
	source["endpoint"] = "https://user:" + marker + "@example.test"
	invalid, err := json.Marshal(document)
	if err != nil {
		t.Fatal(err)
	}
	writeProtectedFixture(t, config, invalid)

	var output, diagnostics bytes.Buffer
	code := (Runner{Out: &output, Err: &diagnostics}).Run(context.Background(), []string{"--config", config, "config", "validate"})
	if code != 2 || output.Len() != 0 || strings.Contains(diagnostics.String(), marker) {
		t.Fatalf("validation failure leaked policy content or had unexpected result: code=%d output=%q diagnostics=%q", code, output.String(), diagnostics.String())
	}
}

func TestConfigInitAndValidate(t *testing.T) {
	dir := privateCLIDirectory(t)
	config := filepath.Join(dir, "cquay.json")
	out := runConfig(t, config, nil, "config", "init")
	if outputState(t, out) != "initialized" {
		t.Fatalf("init response: %s", out)
	}
	out = runConfig(t, config, nil, "config", "validate")
	var response struct {
		Valid bool `json:"valid"`
	}
	if json.Unmarshal(out, &response) != nil || !response.Valid {
		t.Fatalf("validate response: %s", out)
	}
}

func configFixture(t *testing.T) string {
	t.Helper()
	dir := privateCLIDirectory(t)
	config := filepath.Join(dir, "policy.json")
	policy := model.Policy{Version: 1, Sources: map[string]model.Source{"fixture": {Type: "json", Path: "fixture.json"}}, Workloads: map[string]model.Workload{"demo": {Mode: "offline", Bindings: []model.Binding{{Name: "VALUE", Source: "fixture", Field: "VALUE", AllowOffline: true}}}}}
	data, err := json.Marshal(policy)
	if err != nil {
		t.Fatal(err)
	}
	writeProtectedFixture(t, config, data)
	return config
}

func configFixtureWithBindings(t *testing.T) string {
	t.Helper()
	dir := privateCLIDirectory(t)
	config := filepath.Join(dir, "policy.json")
	policy := model.Policy{
		Version: 1,
		Sources: map[string]model.Source{"fixture": {Type: "json", Path: "fixture.json"}},
		Workloads: map[string]model.Workload{"demo": {
			Mode: "offline",
			Bindings: []model.Binding{
				{Name: "FIRST", Source: "fixture", Field: "FIRST", AllowOffline: true},
				{Name: "SECOND", Source: "fixture", Field: "SECOND", AllowOffline: true},
			},
		}},
	}
	data, err := json.Marshal(policy)
	if err != nil {
		t.Fatal(err)
	}
	writeProtectedFixture(t, config, data)
	return config
}

func runConfig(t *testing.T, config string, in *bytes.Reader, args ...string) []byte {
	t.Helper()
	var out, diagnostics bytes.Buffer
	runner := Runner{Out: &out, Err: &diagnostics}
	if in != nil {
		runner.In = in
	}
	code := runner.Run(context.Background(), append([]string{"--config", config}, args...))
	if code != 0 {
		t.Fatalf("config command %v: code %d, error %s", args, code, diagnostics.String())
	}
	return out.Bytes()
}

func assertConfigFailure(t *testing.T, config string, in *bytes.Reader, args ...string) {
	t.Helper()
	var out, diagnostics bytes.Buffer
	runner := Runner{Out: &out, Err: &diagnostics}
	if in != nil {
		runner.In = in
	}
	if code := runner.Run(context.Background(), append([]string{"--config", config}, args...)); code == 0 {
		t.Fatalf("config command unexpectedly succeeded: %v", args)
	}
}
func assertConfigUnchanged(t *testing.T, config string, original []byte) {
	t.Helper()
	now, err := os.ReadFile(config)
	if err != nil || !bytes.Equal(now, original) {
		t.Fatal("invalid mutation changed policy")
	}
}

func configSnapshot(t *testing.T, config string) ([]byte, string) {
	t.Helper()
	data, err := os.ReadFile(config)
	if err != nil {
		t.Fatal(err)
	}
	out := runConfig(t, config, nil, "config", "get", "--pointer", "")
	var response struct {
		SHA string `json:"sha256"`
	}
	if err := json.Unmarshal(out, &response); err != nil || response.SHA == "" {
		t.Fatalf("config digest response: %s", out)
	}
	return data, response.SHA
}

func assertConfigSnapshotUnchanged(t *testing.T, config string, original []byte, digest string) {
	t.Helper()
	assertConfigUnchanged(t, config, original)
	_, now := configSnapshot(t, config)
	if now != digest {
		t.Fatalf("failed mutation changed config digest: got %s, want %s", now, digest)
	}
}

func outputState(t *testing.T, data []byte) string {
	t.Helper()
	var response struct {
		State string `json:"state"`
	}
	if json.Unmarshal(data, &response) != nil {
		return ""
	}
	return response.State
}
func stringsRepeat(s string, n int) string {
	result := ""
	for range n {
		result += s
	}
	return result
}

func TestConfigFormatIsIdempotent(t *testing.T) {
	config := configFixture(t)
	before, err := os.ReadFile(config)
	if err != nil {
		t.Fatal(err)
	}
	formatted, err := (application{config: config}).formatConfig()
	if err != nil || !formatted {
		t.Fatalf("format: %v, %v", formatted, err)
	}
	formatted, err = (application{config: config}).formatConfig()
	if err != nil || formatted {
		t.Fatalf("idempotent format: %v, %v", formatted, err)
	}
	if len(before) == 0 {
		t.Fatal("empty fixture")
	}
}

func TestConfigRejectsUnsafePaths(t *testing.T) {
	config := configFixture(t)
	if runtime.GOOS != "windows" {
		if err := os.Chmod(config, 0666); err != nil {
			t.Fatal(err)
		}
		assertConfigFailure(t, config, nil, "config", "validate")
		if err := os.Chmod(config, 0600); err != nil {
			t.Fatal(err)
		}
	}
	link := filepath.Join(filepath.Dir(config), "link.json")
	if err := os.Symlink(config, link); err == nil {
		assertConfigFailure(t, link, nil, "config", "validate")
	}
}

func TestConfigCooperativeLockPreventsConflictingMutation(t *testing.T) {
	config := configFixture(t)
	path, err := filepath.Abs(config)
	if err != nil {
		t.Fatal(err)
	}
	unlock, err := configLock(path)
	if err != nil {
		t.Fatal(err)
	}
	original, err := os.ReadFile(config)
	if err != nil {
		t.Fatal(err)
	}
	assertConfigFailure(t, config, nil, "config", "set", "--pointer", "/version", "--value-json", `1`)
	assertConfigUnchanged(t, config, original)
	if err = unlock(); err != nil {
		t.Fatal(err)
	}
	if outputState(t, runConfig(t, config, nil, "config", "set", "--pointer", "/version", "--value-json", `1`)) != "updated" {
		t.Fatal("fresh mutation was not applied")
	}
}

func TestConfigReadCommandsRejectWriteFlags(t *testing.T) {
	config := configFixture(t)
	assertConfigFailure(t, config, nil, "config", "get", "--pointer", "", "--expect-sha256", stringsRepeat("0", 64))
	assertConfigFailure(t, config, nil, "config", "list", "--value-json", `1`)
}

func TestConfigReplaceRepairsInvalidExistingPolicy(t *testing.T) {
	config := configFixture(t)
	invalid := []byte(`{"version":1,"sources":{},"workloads":{}}`)
	writeProtectedFixture(t, config, invalid)
	replacement := configFixtureData(t)
	out := runConfig(t, config, bytes.NewReader(replacement), "config", "replace", "--file", "-")
	if outputState(t, out) != "updated" {
		t.Fatalf("repair response: %s", out)
	}
	if got, err := os.ReadFile(config); err != nil || bytes.Equal(got, invalid) {
		t.Fatal("valid replacement did not repair invalid policy")
	}
	before, _ := os.ReadFile(config)
	assertConfigFailure(t, config, bytes.NewReader([]byte(`{"version":1}`)), "config", "replace", "--file", "-")
	assertConfigUnchanged(t, config, before)
}

func TestConfigReplaceRepairsRelativePolicy(t *testing.T) {
	for _, name := range []string{"cquay.json", "relative-policy.json"} {
		t.Run(name, func(t *testing.T) {
			directory := filepath.Dir(configFixture(t))
			t.Chdir(directory)
			path := filepath.Join(directory, name)
			writeProtectedFixture(t, path, []byte(`{"version":1,"sources":{},"workloads":{}}`))
			args := []string{"config", "replace", "--file", "-"}
			if name != "cquay.json" {
				args = append([]string{"--config", name}, args...)
			}
			var out, diagnostics bytes.Buffer
			status := (Runner{In: bytes.NewReader(configFixtureData(t)), Out: &out, Err: &diagnostics}).Run(context.Background(), args)
			if status != 0 || outputState(t, out.Bytes()) != "updated" {
				t.Fatalf("relative repair failed: status %d, %s", status, diagnostics.Bytes())
			}
			runConfig(t, path, nil, "config", "validate")
		})
	}
}

func TestPointerIndexRejectsNonDigits(t *testing.T) {
	for _, pointer := range []string{"/+0", "/-0", "/ 0", "/0x1"} {
		if _, err := jsonPointerGet([]any{"item"}, pointer); err == nil {
			t.Fatalf("accepted non-canonical index %q", pointer)
		}
	}
}

func configFixtureData(t *testing.T) []byte {
	t.Helper()
	policy := model.Policy{Version: 1, Sources: map[string]model.Source{"fixture": {Type: "json", Path: "fixture.json"}}, Workloads: map[string]model.Workload{"demo": {Mode: "offline", Bindings: []model.Binding{{Name: "VALUE", Source: "fixture", Field: "VALUE", AllowOffline: true}}}}}
	data, err := json.Marshal(policy)
	if err != nil {
		t.Fatal(err)
	}
	return data
}
