package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
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
	if err = os.WriteFile(config, data, 0600); err != nil {
		t.Fatal(err)
	}
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

func TestConfigFormatAndUnsafePathRejection(t *testing.T) {
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
	if err = os.Chmod(config, 0666); err != nil {
		t.Fatal(err)
	}
	assertConfigFailure(t, config, nil, "config", "validate")
	if err = os.Chmod(config, 0600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(filepath.Dir(config), "link.json")
	if err = os.Symlink(config, link); err == nil {
		assertConfigFailure(t, link, nil, "config", "validate")
	}
	if len(before) == 0 {
		t.Fatal("empty fixture")
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
	if err := os.WriteFile(config, invalid, 0600); err != nil {
		t.Fatal(err)
	}
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
