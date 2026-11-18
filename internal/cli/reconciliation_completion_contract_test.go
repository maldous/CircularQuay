package cli

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/maldous/CircularQuay/internal/fault"
	"github.com/maldous/CircularQuay/internal/model"
	"github.com/maldous/CircularQuay/internal/resolve"
	"github.com/maldous/CircularQuay/internal/state"
)

func TestCLICompletionReconciliationReadAndInitialWriteRefusals(t *testing.T) {
	for _, phase := range []string{"read", "initial-write"} {
		t.Run(phase, func(t *testing.T) {
			cliCompletionReconciliationCompletionCase1(t, phase)
		})
	}
}

func TestCLICompletionReconciliationWriteFailureFencesActivation(t *testing.T) {
	for _, phase := range []string{"project-receipt", "activation-receipt", "failure-receipt"} {
		t.Run(phase, func(t *testing.T) {
			cliCompletionReconciliationCompletionCase2(t, phase)
		})
	}
}

func TestCLICompletionCancellationAfterDurablePendingDoesNotActivate(t *testing.T) {
	store, workload, generation := reconciliationFixture(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	write := func(record state.Record) error {
		if err := store.Write(record); err != nil { return err }
		if record.State == "partial_pending_activation" { cancel() }
		return nil
	}
	result, err := reconcileTargetsWithState(ctx, store, workload, generation, []string{"a"}, targetOperations{
		project: func(context.Context, model.Workload, string, model.Generation) error { return nil },
		activate: func(context.Context, model.Target) error { t.Fatal("canceled pending receipt reached native activation"); return nil },
	}, store.Read, write)
	cliCompletionFault(t, err, fault.Apply, "activation canceled")
	stored, readErr := store.Read("demo")
	if readErr != nil || result.State != "partial_pending_activation" || stored.State != result.State || stored.Targets["a"].Error != fault.Apply || !stored.Targets["a"].ActivatedAt.IsZero() { t.Fatal("pending failure was not durable") }
}

func TestCLICompletionObservedActivationAndReconcileFailuresAreForwarded(t *testing.T) {
	ctx := context.Background()
	target := model.Target{Type: "systemd", Service: "synthetic.service", Activation: "restart"}
	refusal := fault.New(fault.Apply, "owned activation")
	calls := 0
	err := activateObservedTargetWithOperation(ctx, target, func(got context.Context, value model.Target) error { calls++; if got == nil || value != target { t.Fatal("activation inputs changed") }; return refusal })
	if err != refusal || calls != 1 { t.Fatal("observed activation failure was hidden") }
	app := application{runner: Runner{Out: io.Discard}}
	code, err := app.reconcileWorkloadWithOperations(ctx, model.Workload{}, &resolve.Engine{}, model.Generation{}, nil, func(context.Context, state.Store, model.Workload, model.Generation, []string, targetOperations) (state.Record, error) { return state.Record{}, refusal })
	if code != 0 || err != refusal { t.Fatal("reconciliation refusal was hidden") }
	if completeSelection(model.Workload{Targets: map[string]model.Target{"a":{}}}, []string{"other"}) { t.Fatal("same-sized foreign selection was complete") }
}

func TestCLICompletionMetadataConstructorsAndRepositoryAdmission(t *testing.T) {
	for _, app := range []application{{}, {repository: repositoryOperations{fetch: func(context.Context, model.Repository) (string, error) { t.Fatal("partial override selected"); return "", nil }}}} {
		operations := app.repositoryOperations()
		if operations.fetch == nil || operations.current == nil { t.Fatal("default operations missing") }
	}
	cases := []struct{ policy model.Policy; work model.Workload; kind fault.Kind }{
		{model.Policy{}, model.Workload{Bindings: []model.Binding{{Source:"absent"}}}, fault.Invalid},
		{model.Policy{Sources:map[string]model.Source{"a":{Path:"repo:../escape"}}}, model.Workload{Bindings:[]model.Binding{{Source:"a"}}}, fault.Trust},
	}
	for _, test := range cases {
		selected, err := repositorySources(test.policy, test.work)
		cliCompletionFault(t, err, test.kind, repositorySourceOperation)
		if selected != nil { t.Fatal("invalid binding retained source selection") }
	}
	path, err := repositorySourcePath("relative-release", "repo:values.json")
	cliCompletionFault(t, err, fault.Trust, repositorySourceOperation)
	if path != "" { t.Fatal("relative release produced path") }
}

func TestCLICompletionDispatchAndRecoverySnapshotsKeepRefusals(t *testing.T) {
	app, _ := cliCompletionConfig(t)
	for _, method := range []func(context.Context, []model.Policy) (model.Policy,error){app.commandPolicy, app.recoveryPolicy, app.updatePolicy} {
		policy, err := method(context.Background(), []model.Policy{{}, {}})
		cliCompletionFault(t, err, fault.Invalid, "policy snapshot")
		if !reflect.DeepEqual(policy, model.Policy{}) { t.Fatal("ambiguous snapshot retained policy") }
	}
	policy := model.Policy{Version:1}
	loaded, err := app.updatePolicy(context.Background(), []model.Policy{policy})
	if err != nil || !reflect.DeepEqual(loaded,policy) { t.Fatal("selected recovery policy changed") }
	for _, args := range [][]string{{"doctor", "--not-a-flag"}, {"update", "invalid"}, {"mcp", "invalid"}, {"unrecognized"}} {
		code, err := app.dispatchLoadedRuntime(context.Background(), args, policy)
		if code != 0 || !fault.Is(err, fault.Invalid) { t.Fatal("runtime dispatch hid inner refusal") }
	}

}

func cliCompletionReconciliationCompletionCase1(t *testing.T, phase string) {
	t.Helper()
	store, workload, generation := reconciliationFixture(t)
	refusal := fault.New(fault.Apply, "owned "+phase)
	var calls []string
	result, err := reconcileTargetsWithState(context.Background(), store, workload, generation, []string{"a", "b"}, targetOperations{
		project: func(context.Context, model.Workload, string, model.Generation) error { t.Fatal("state admission reached projection"); return nil },
		activate: func(context.Context, model.Target) error { t.Fatal("state admission reached activation"); return nil },
	}, func(id string) (state.Record, error) {
		calls = append(calls, "read")
		if id != "demo" { t.Fatal("state scope changed") }
		if phase == "read" { return state.Record{}, refusal }
		return state.Record{}, nil
	}, func(record state.Record) error {
		calls = append(calls, "write")
		cliCompletionInitialRecord(t,record)
		return refusal
	})
	want := []string{"read"}
	if phase != "read" { want = append(want, "write") }
	if err != refusal || result.Workload != "demo" || !reflect.DeepEqual(calls, want) { t.Fatal("refused state was hidden or later effects ran") }
	if _, err := os.Lstat(store.Directory); !os.IsNotExist(err) { t.Fatal("fake state acknowledgement created durable state") }

}

func cliCompletionReconciliationCompletionCase2(t *testing.T, phase string) {
	t.Helper()
	store, workload, generation := reconciliationFixture(t)
	var calls []string
	writes := 0
	refusal := fault.New(fault.Apply, "owned state write")
	project := func(context.Context, model.Workload, string, model.Generation) error { calls = append(calls, "project"); return nil }
	if phase == "failure-receipt" { project = func(context.Context, model.Workload, string, model.Generation) error { calls = append(calls, "project"); return fault.New(fault.Denied, "owned projection") } }
	write := func(record state.Record) error {
		writes++
		calls = append(calls, "write:"+record.State)
		if writes == map[string]int{"activation-receipt":3,"project-receipt":2,"failure-receipt":2}[phase] { return refusal }
		return nil
	}
	_, err := reconcileTargetsWithState(context.Background(), store, workload, generation, []string{"a"}, targetOperations{project: project, activate: func(context.Context, model.Target) error { t.Fatal("uncommitted receipt reached activation"); return nil }}, func(string) (state.Record, error) { return state.Record{}, nil }, write)
	want := map[string][]string{
 "project-receipt":{"write:partial_resolved","project","write:partial_resolved"},
 "failure-receipt":{"write:partial_resolved","project","write:partial_apply_failed"},
 "activation-receipt":{"write:partial_resolved","project","write:partial_resolved","write:partial_pending_activation"},
 }[phase]
	if err != refusal || !reflect.DeepEqual(calls, want) { t.Fatalf("receipt refusal ordering changed: %v %v", calls, err) }

}

func cliCompletionInitialRecord(t *testing.T,record state.Record) {
 t.Helper()
 if record.Workload!="demo" || len(record.Targets)!=2 || record.State!="resolved" {t.Fatal("initial state changed")}
}

func TestCLICompletionUnsupportedCompletionAndMissingBootstrap(t *testing.T) {
 app:=application{runner:Runner{Out:io.Discard}}
	if code, err := app.validateLocalBootstrap(context.Background(), filepath.Join(privateCLIDirectory(t), "absent.json")); code != 0 || err == nil { t.Fatal("local bootstrap admitted missing policy") }
	if code, err := app.completeWorkloadCommand(context.Background(), workloadRequest{command:"unknown"}, model.Workload{}, &resolve.Engine{}, model.Generation{}); code != 0 || !fault.Is(err, fault.Invalid) { t.Fatal("unknown completion selected effect") }
}
