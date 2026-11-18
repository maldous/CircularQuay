package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/maldous/CircularQuay/internal/fault"
	"github.com/maldous/CircularQuay/internal/model"
)

func TestCommandsBoundaryContractInitRejectsBeforePathEffects(t *testing.T) {
	for _, args := range [][]string{
		{"--SYNTHETIC_PRIVATE_FLAG"},
		{"--output"},
		{"SYNTHETIC_PRIVATE_ARGUMENT"},
	} {
		var output, diagnostics bytes.Buffer
		app := application{config: "SYNTHETIC_PRIVATE_POLICY", runner: Runner{Out: &output, Err: &diagnostics}}
		code, err := app.initialize(args)
		requireCommandsBoundaryRefusal(t, code, err, fault.Invalid, "init arguments", &output, &diagnostics)
	}
}

func TestCommandsBoundaryContractStatusRejectsBeforeStore(t *testing.T) {
	for _, tc := range []struct {
		name, operation string
		args            []string
	}{
		{"missing workload", "status arguments", nil},
		{"unknown flag", "status arguments", []string{"--SYNTHETIC_PRIVATE_FLAG"}},
		{"extra argument", "status arguments", []string{"--workload", "demo", "SYNTHETIC_PRIVATE_ARGUMENT"}},
		{"unknown workload", "workload", []string{"--workload", "SYNTHETIC_PRIVATE_WORKLOAD"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var output, diagnostics bytes.Buffer
			app := application{runner: Runner{Out: &output, Err: &diagnostics}}
			policy := model.Policy{Workloads: map[string]model.Workload{"demo": {Mode: "offline"}}}
			code, err := app.status(tc.args, policy)
			requireCommandsBoundaryRefusal(t, code, err, fault.Invalid, tc.operation, &output, &diagnostics)
		})
	}
}

func TestCommandsBoundaryContractWorkloadFlagsHaveClosedResults(t *testing.T) {
	for _, args := range [][]string{
		{"--SYNTHETIC_PRIVATE_FLAG"},
		{"--workload"},
		nil,
	} {
		id, target, rest, err := workloadFlags("run", args)
		requireCommandsBoundaryFault(t, err, fault.Invalid, "workload arguments")
		if id != "" || target != "" || rest != nil {
			t.Fatal("rejected workload flags retained parsed selectors or arguments")
		}
	}
	// This is flag parsing only: it starts no synchronization or target effect.
	id, target, rest, err := workloadFlags("sync", []string{"--workload", "demo", "--target", "chosen", "--once=false"})
	if err != nil || id != "demo" || target != "chosen" || len(rest) != 0 {
		t.Fatal("valid explicit sync flag values changed")
	}
}

func TestCommandsBoundaryContractLoadedWorkloadRefusesBeforeResolution(t *testing.T) {
	for _, tc := range []struct {
		name, operation string
		kind            fault.Kind
		args            []string
	}{
		{"unknown workload", "workload", fault.Invalid, []string{"sync", "--workload", "missing"}},
		{"apply requires target", "explicit target required", fault.Invalid, []string{"apply", "--workload", "demo"}},
		{"apply unknown target", "target mapping", fault.Denied, []string{"apply", "--workload", "demo", "--target", "missing"}},
		{"sync unknown target", "target mapping", fault.Denied, []string{"sync", "--workload", "demo", "--target", "missing"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			app, policy, probe := commandsBoundaryLoadedFixture(t)
			code, err := app.workloadCommand(context.Background(), tc.args, policy)
			requireCommandsBoundaryRefusal(t, code, err, tc.kind, tc.operation, app.runner.Out.(*bytes.Buffer), app.runner.Err.(*bytes.Buffer))
			assertCommandPreflightNoResolutionEffects(t, app.directory, probe)
		})
	}
}

func commandsBoundaryLoadedFixture(t *testing.T) (application, model.Policy, *commandPreflightProbe) {
	t.Helper()
	root := t.TempDir()
	probe := &commandPreflightProbe{}
	app := application{
		directory: filepath.Join(root, "unused-state"),
		runner:    Runner{In: bytes.NewReader(nil), Out: &bytes.Buffer{}, Err: &bytes.Buffer{}},
		repository: repositoryOperations{
			fetch: func(context.Context, model.Repository) (string, error) {
				probe.fetches++
				return "", errors.New("synthetic repository must not be reached")
			},
			current: func(context.Context, model.Repository) (string, error) {
				probe.currents++
				return "", errors.New("synthetic repository must not be reached")
			},
		},
	}
	policy := model.Policy{
		Version:    1,
		Repository: model.Repository{URL: "https://github.com/synthetic/config", Directory: filepath.Join(root, "repository"), Ref: "refs/heads/main", AllowedSignersFile: filepath.Join(root, "signers"), InitialCommit: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"},
		Sources:    map[string]model.Source{"repository": {Type: "json", Path: "repo:values.json"}},
		Workloads: map[string]model.Workload{"demo": {
			Mode:      "offline",
			Directory: filepath.Join(root, "unused-work"),
			Bindings:  []model.Binding{{Name: "VALUE", Source: "repository", Field: "value", AllowOffline: true, AllowPublish: true}},
			Targets:   map[string]model.Target{"chosen": {Type: "file", Path: filepath.Join(root, "unused-output"), AcceptSnapshot: true}},
		}},
	}
	// This exercises the existing explicit loaded-policy seam. It does not
	// claim to validate persisted policy enrollment or enter an operational engine.
	return app, policy, probe
}

type commandsBoundaryPlanEntry struct {
	ID       string `json:"id"`
	Mode     string `json:"mode"`
	Bindings int    `json:"bindings"`
	Targets  int    `json:"targets"`
}

type commandsBoundaryPlanReport struct {
	Valid         bool                        `json:"valid"`
	ValueRequests int                         `json:"value_requests"`
	Workloads     []commandsBoundaryPlanEntry `json:"workloads"`
}

func TestCommandsBoundaryContractPlanSortsMetadataWithoutDisclosure(t *testing.T) {
	private := "SYNTHETIC_PRIVATE_DETAIL"
	root := t.TempDir()
	first := model.Binding{Name: private + "_FIRST", Source: "synthetic", Field: "first", AllowOffline: true, AllowPublish: true}
	second := model.Binding{Name: private + "_SECOND", Source: "synthetic", Field: "second", AllowOffline: true, AllowPublish: true}
	policy := model.Policy{
		Version: 1,
		Sources: map[string]model.Source{"synthetic": {Type: "json", Path: filepath.Join(root, private+".json")}},
		Workloads: map[string]model.Workload{
			"zulu":   {Mode: "hybrid", Directory: filepath.Join(root, "zulu"), Bindings: []model.Binding{first, second}, Targets: map[string]model.Target{"last": {Type: "file", Path: filepath.Join(root, private+"-last"), AcceptSnapshot: true}}},
			"alpha":  {Mode: "online", Directory: filepath.Join(root, "alpha"), Bindings: []model.Binding{first}, Targets: map[string]model.Target{"first": {Type: "file", Path: filepath.Join(root, private+"-first")}, "second": {Type: "file", Path: filepath.Join(root, private+"-second")}}},
			"middle": {Mode: "offline", Directory: filepath.Join(root, "middle"), Bindings: []model.Binding{first}},
		},
	}
	var output, diagnostics bytes.Buffer
	code, err := (application{runner: Runner{Out: &output, Err: &diagnostics}}).plan(policy)
	if code != 0 || err != nil || diagnostics.Len() != 0 {
		t.Fatal("metadata-only plan did not return ordinary output")
	}
	report := decodeCommandsBoundaryPlan(t, output.Bytes())
	if !report.Valid || report.ValueRequests != 0 {
		t.Fatal("plan changed its validation or zero-request declaration")
	}
	want := []commandsBoundaryPlanEntry{{"alpha", "online", 1, 2}, {"middle", "offline", 1, 0}, {"zulu", "hybrid", 2, 1}}
	if !reflect.DeepEqual(report.Workloads, want) || bytes.Contains(output.Bytes(), []byte(private)) {
		t.Fatal("plan lost sorted workload metadata or disclosed declarations")
	}
}

func decodeCommandsBoundaryPlan(t *testing.T, data []byte) commandsBoundaryPlanReport {
	t.Helper()
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var report commandsBoundaryPlanReport
	if decoder.Decode(&report) != nil {
		t.Fatal("plan did not emit only its documented metadata fields")
	}
	if decoder.Decode(&struct{}{}) != io.EOF {
		t.Fatal("plan emitted another result or trailing non-JSON output")
	}
	return report
}

func requireCommandsBoundaryRefusal(t *testing.T, code int, err error, kind fault.Kind, operation string, output, diagnostics *bytes.Buffer) {
	t.Helper()
	requireCommandsBoundaryFault(t, err, kind, operation)
	if code != 0 || output.Len() != 0 || diagnostics.Len() != 0 {
		t.Fatal("application refusal retained success output or diagnostics")
	}
}

func requireCommandsBoundaryFault(t *testing.T, err error, kind fault.Kind, operation string) {
	t.Helper()
	typed, ok := err.(*fault.Error)
	if !ok {
		t.Fatal("application refusal did not return a direct typed fault")
	}
	if typed.Kind != kind || typed.Operation != operation || err.Error() != string(kind)+": "+operation {
		t.Fatal("application refusal changed its safe category or operation")
	}
}
