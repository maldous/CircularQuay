package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/maldous/CircularQuay/internal/fault"
	"github.com/maldous/CircularQuay/internal/model"
)

func TestCLISafeConfigReplaceCreatesMissingDeclaration(t *testing.T) {
	source := configFixture(t)
	before, err := os.ReadFile(source)
	if err != nil {
		t.Fatal(err)
	}
	destination := filepath.Join(filepath.Dir(source), "created.json")
	out := runConfig(t, destination, nil, "config", "replace", "--file", source)
	if outputState(t, out) != "updated" {
		t.Fatal("missing declaration was not created")
	}
	var got, want model.Policy
	created, err := os.ReadFile(destination)
	if err != nil {
		t.Fatal(err)
	}
	if json.Unmarshal(created, &got) != nil || json.Unmarshal(before, &want) != nil || !reflect.DeepEqual(got, want) {
		t.Fatal("created declaration differs from replacement input")
	}
	assertConfigUnchanged(t, source, before)
	requireMissingPath(t, filepath.Join(filepath.Dir(source), "fixture.json"))
}

func TestCLISafeConfigRejectsOversizedSerializedCandidate(t *testing.T) {
	config := configFixture(t)
	before, digest := configSnapshot(t, config)
	var out bytes.Buffer
	a := application{config: config, runner: Runner{Out: &out}}
	value := `"` + strings.Repeat("x", maximumConfigInput-2) + `"`
	code, err := a.configCommand(context.Background(), []string{"set", "--pointer", "/sources/fixture/path", "--value-json", value})
	requireCLISafeFault(t, err, fault.Invalid, "config document")
	if code != 0 || out.Len() != 0 {
		t.Fatal("oversized candidate produced success output")
	}
	assertConfigSnapshotUnchanged(t, config, before, digest)
}

func TestCLISafeConfigArrayReplacementAndTildeLookup(t *testing.T) {
	config := configFixtureWithBindings(t)
	before, err := os.ReadFile(config)
	if err != nil {
		t.Fatal(err)
	}
	var original model.Policy
	if err := json.Unmarshal(before, &original); err != nil {
		t.Fatal(err)
	}
	out := runConfig(t, config, nil, "config", "set", "--pointer", "/workloads/demo/bindings/0", "--value-json", `{"name":"REPLACED","source":"fixture","field":"FIRST","allow_offline":true}`)
	if outputState(t, out) != "updated" {
		t.Fatal("array element not replaced")
	}
	cliSafeBindingReplacement(t, config, original)
	var output bytes.Buffer
	a := application{config: config, runner: Runner{Out: &output}}
	code, err := a.configCommand(context.Background(), []string{"get", "--pointer", "/sources/missing~0name"})
	requireCLISafeFault(t, err, fault.Missing, errorConfigPointer)
	if code != 0 || output.Len() != 0 {
		t.Fatal("absent escaped property produced output")
	}
}

func cliSafeBindingReplacement(t *testing.T, config string, original model.Policy) {
	t.Helper()
	data, err := os.ReadFile(config)
	if err != nil {
		t.Fatal(err)
	}
	var got model.Policy
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}
	want := original
	workload := want.Workloads["demo"]
	workload.Bindings[0].Name = "REPLACED"
	want.Workloads["demo"] = workload
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("array replacement changed siblings: %#v", got)
	}
}
