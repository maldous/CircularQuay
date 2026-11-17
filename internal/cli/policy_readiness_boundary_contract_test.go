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
)

func TestCLISafeRootCancellationAndParsedMinimum(t *testing.T) {
	config := configFixture(t)
	before, err := os.ReadFile(config)
	if err != nil {
		t.Fatal(err)
	}
	a := application{config: config, buildVersion: "1.0.0"}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	data, path, policy, err := a.readRootPolicy(ctx, config)
	requireCLISafeFault(t, err, fault.Unavailable, "configuration cancelled")
	if data != nil || path != "" || !reflect.DeepEqual(policy, model.Policy{}) {
		t.Fatal("cancelled root read returned partial policy")
	}
	var candidate model.Policy
	if err := json.Unmarshal(before, &candidate); err != nil {
		t.Fatal(err)
	}
	candidate.Updates = &model.Updates{Check: "never", MinimumVersion: "2.0.0"}
	encoded, err := json.Marshal(candidate)
	if err != nil {
		t.Fatal(err)
	}
	got, err := a.resolveParsedPolicy(context.Background(), encoded, config)
	requireCLISafeFault(t, err, fault.Denied, "minimum runtime version")
	if !reflect.DeepEqual(got, model.Policy{}) {
		t.Fatal("minimum refusal returned partial policy")
	}
	assertConfigUnchanged(t, config, before)
}

func TestCLISafeDoctorRereadRefusesChangedInvalidDocument(t *testing.T) {
	config := configFixture(t)
	var out bytes.Buffer
	a := application{config: config, runner: Runner{Out: &out}}
	loaded, err := a.load()
	if err != nil {
		t.Fatal(err)
	}
	invalid := []byte("{")
	writeProtectedFixture(t, config, invalid)
	state, err := a.doctorFormatting(context.Background(), true)
	requireCLISafeFault(t, err, fault.Invalid, errorConfigJSON)
	if state != "" {
		t.Fatal("formatter caller reported success")
	}
	report := doctorReport{}
	code, err := a.doctorReadiness(context.Background(), &report, true, []model.Policy{loaded})
	requireCLISafeFault(t, err, fault.Invalid, errorConfigJSON)
	if code != 0 || out.Len() != 0 || report.Fix != "" {
		t.Fatal("reread failure produced readiness success")
	}
	assertConfigUnchanged(t, config, invalid)
}

func TestCLISafeDoctorTargetMetadataIsDeduplicated(t *testing.T) {
	path := filepath.Join(privateCLIDirectory(t), "token-metadata")
	before := []byte("synthetic-only")
	writeProtectedFixture(t, path, before)
	policy := model.Policy{Workloads: map[string]model.Workload{"demo": {Targets: map[string]model.Target{"first": {TokenFile: path}, "second": {TokenFile: path}}}}}
	got := doctorPolicyMetadata(policy)
	want := []doctorDependency{{Kind: "protected_file", Path: path, State: "ready"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("target metadata = %#v; want %#v", got, want)
	}
	assertConfigUnchanged(t, path, before)
}
