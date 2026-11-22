package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"path/filepath"
	"testing"

	"github.com/maldous/CircularQuay/internal/model"
	"github.com/maldous/CircularQuay/internal/platform"
)

func TestUpdateCheckRemainsAvailableBelowMinimum(t *testing.T) {
	directory := privateCLIDirectory(t)
	fixture := filepath.Join(directory, "fixture.json")
	writeProtectedFixture(t, fixture, []byte(`{"VALUE":"synthetic"}`))
	config := filepath.Join(directory, "policy.json")
	policy := model.Policy{
		Version:   1,
		Sources:   map[string]model.Source{"fixture": {Type: "json", Path: fixture}},
		Workloads: map[string]model.Workload{"demo": {Mode: "offline", Bindings: []model.Binding{{Name: "VALUE", Source: "fixture", Field: "VALUE", AllowOffline: true}}}},
		Updates:   &model.Updates{Check: "never", MinimumVersion: "v999.0.0"},
	}
	data, err := json.Marshal(policy)
	if err != nil {
		t.Fatal(err)
	}
	if err = platform.CreateProtectedExclusive(config, data); err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	app := application{config: config, directory: filepath.Join(directory, "state"), runner: Runner{In: bytes.NewReader(nil), Out: &output, Err: io.Discard}}
	code, err := app.updateCommand(context.Background(), []string{"check"})
	var result struct {
		Status string `json:"status"`
	}
	if err != nil || code != 0 || json.Unmarshal(output.Bytes(), &result) != nil || result.Status != "disabled" {
		t.Fatalf("update recovery check = code %d, output %q, error %v", code, output.String(), err)
	}
}

func TestUpdateInteractiveGateRejectsSDKStyleStreams(t *testing.T) {
	if interactiveTerminal(bytes.NewReader(nil), io.Discard) {
		t.Fatal("non-terminal SDK streams enabled an interactive update prompt")
	}
}

func TestUpdateCheckEligibilityNeverChecksMCP(t *testing.T) {
	config := &model.Updates{Check: "interactive"}
	for command, expected := range map[string]bool{
		"plan": true, "run": true, "sync": true, "snapshot": true, "apply": true, "status": true,
		"mcp": false, "doctor": false, "config": false, "init": false, "update": false,
	} {
		if actual := shouldCheckUpdates(command, config, true); actual != expected {
			t.Fatalf("update eligibility for %q = %t, want %t", command, actual, expected)
		}
	}
	if shouldCheckUpdates("run", config, false) || shouldCheckUpdates("run", nil, true) || shouldCheckUpdates("run", &model.Updates{Check: "never"}, true) {
		t.Fatal("advisory check enabled without an interactive update declaration and terminal")
	}
}

func TestRunnerBuildVersionControlsMinimumRuntimeGate(t *testing.T) {
	directory := privateCLIDirectory(t)
	fixture := filepath.Join(directory, "fixture.json")
	writeProtectedFixture(t, fixture, []byte(`{"VALUE":"synthetic"}`))
	config := filepath.Join(directory, "policy.json")
	policy := model.Policy{
		Version:   1,
		Sources:   map[string]model.Source{"fixture": {Type: "json", Path: fixture}},
		Workloads: map[string]model.Workload{"demo": {Mode: "offline", Bindings: []model.Binding{{Name: "VALUE", Source: "fixture", Field: "VALUE", AllowOffline: true}}}},
		Updates:   &model.Updates{Check: "never", MinimumVersion: "v1.0.0"},
	}
	data, err := json.Marshal(policy)
	if err != nil {
		t.Fatal(err)
	}
	if err = platform.CreateProtectedExclusive(config, data); err != nil {
		t.Fatal(err)
	}
	arguments := []string{"--config", config, "--state-dir", filepath.Join(directory, "state"), "plan"}
	var sdkOut, sdkErr bytes.Buffer
	sdkCode := (Runner{In: bytes.NewReader(nil), Out: &sdkOut, Err: &sdkErr, BuildVersion: "1.0.0"}).Run(context.Background(), arguments)
	if sdkCode != 0 || sdkErr.Len() != 0 {
		t.Fatalf("SDK build version did not satisfy policy: code=%d diagnostics=%q", sdkCode, sdkErr.String())
	}
	var nativeOut, nativeErr bytes.Buffer
	nativeCode := (Runner{In: bytes.NewReader(nil), Out: &nativeOut, Err: &nativeErr}).Run(context.Background(), arguments)
	if nativeCode != 3 || nativeErr.String() != "{\"error\":\"access_denied\"}\n" {
		t.Fatalf("native development version bypassed minimum: code=%d diagnostics=%q", nativeCode, nativeErr.String())
	}
}
