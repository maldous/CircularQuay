package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/maldous/CircularQuay/internal/fault"
	"github.com/maldous/CircularQuay/internal/model"
	"github.com/maldous/CircularQuay/internal/state"
)

func TestCLISafeStatusReturnsOnlyOwnedMetadata(t *testing.T) {
	config := configFixture(t)
	directory := filepath.Join(filepath.Dir(config), "state")
	want := state.Record{Workload: "demo", State: "not_provisioned"}
	cliSafeStatusRecord(t, config, directory, want)
	want = state.Record{Workload: "demo", State: "blocked", Mode: "offline", FallbackFence: true, Generation: "synthetic-generation"}
	if err := (state.Store{Directory: directory}).Write(want); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(directory, "demo.json")
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	cliSafeStatusRecord(t, config, directory, want)
	assertConfigUnchanged(t, path, before)
	requireMissingPath(t, filepath.Join(filepath.Dir(config), "fixture.json"))
}

func cliSafeStatusRecord(t *testing.T, config, directory string, want state.Record) {
	t.Helper()
	var out, diagnostics bytes.Buffer
	runner := Runner{In: bytes.NewReader(nil), Out: &out, Err: &diagnostics}
	code := runner.Run(context.Background(), []string{"--config", config, "--state-dir", directory, "status", "--workload", "demo", "--json"})
	if code != 0 || diagnostics.Len() != 0 {
		t.Fatalf("status = %d/%q", code, diagnostics.String())
	}
	var got state.Record
	if err := json.Unmarshal(out.Bytes(), &got); err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("status metadata = %#v/%v; want %#v", got, err, want)
	}
}

func TestCLISafeStatusRejectsMalformedOwnedRecord(t *testing.T) {
	root := privateCLIDirectory(t)
	path := filepath.Join(root, "demo.json")
	before := []byte(`{"workload":"other","state":"blocked"}`)
	writeProtectedFixture(t, path, before)
	var out bytes.Buffer
	a := application{directory: root, runner: Runner{Out: &out}}
	code, err := a.status([]string{"--workload", "demo"}, model.Policy{Workloads: map[string]model.Workload{"demo": {}}})
	requireCLISafeFault(t, err, fault.Trust, "state record")
	if code != 0 || out.Len() != 0 {
		t.Fatal("invalid record produced output")
	}
	assertConfigUnchanged(t, path, before)
}

func TestCLISafeLegacyStateFencePropagatesBeforeOperations(t *testing.T) {
	configDirectory := syntheticUserConfigDirectory(t)
	legacy := filepath.Join(configDirectory, "circularquay", "state")
	if err := os.MkdirAll(legacy, 0o700); err != nil {
		t.Fatal(err)
	}
	config := configFixture(t)
	var out bytes.Buffer
	a := application{config: config, runner: Runner{In: bytes.NewReader(nil), Out: &out, Err: &out}}
	policy, err := a.load()
	if err != nil {
		t.Fatal(err)
	}
	cliSafeLegacyCalls(t, a, policy)
	if out.Len() != 0 {
		t.Fatalf("legacy refusal emitted output: %q", out.String())
	}
	entries, err := os.ReadDir(legacy)
	if err != nil || len(entries) != 0 {
		t.Fatalf("legacy fixture mutated: %v/%v", entries, err)
	}
	requireMissingPath(t, filepath.Join(configDirectory, "cq", "state"))
	requireMissingPath(t, filepath.Join(filepath.Dir(config), "fixture.json"))
}

func cliSafeLegacyCalls(t *testing.T, a application, policy model.Policy) {
	t.Helper()
	code, err := a.status([]string{"--workload", "demo"}, policy)
	cliSafeLegacyCode(t, code, err)
	request := workloadRequest{command: "snapshot", id: "demo"}
	_, workload, engine, err := a.prepareWorkloadCommand(policy, request)
	requireCLISafeFault(t, err, fault.Trust, "legacy state directory")
	if engine != nil || !reflect.DeepEqual(workload, model.Workload{}) {
		t.Fatal("legacy refusal returned partial preparation")
	}
	code, err = a.serveMCP(context.Background(), []string{"serve"}, policy)
	cliSafeLegacyCode(t, code, err)
	code, err = a.updateCheckCommand(context.Background(), nil, policy)
	cliSafeLegacyCode(t, code, err)
}

func cliSafeLegacyCode(t *testing.T, code int, err error) {
	t.Helper()
	requireCLISafeFault(t, err, fault.Trust, "legacy state directory")
	if code != 0 {
		t.Fatalf("inner refusal status = %d", code)
	}
}
