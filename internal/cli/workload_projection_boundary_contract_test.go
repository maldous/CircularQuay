package cli

import (
	"bytes"
	"context"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/maldous/CircularQuay/internal/fault"
	"github.com/maldous/CircularQuay/internal/model"
)

func TestCLISafeWorkloadAdmissionBeforeState(t *testing.T) {
	root := privateCLIDirectory(t)
	var out bytes.Buffer
	a := application{config: filepath.Join(root, "absent.json"), directory: filepath.Join(root, "unused-state"), runner: Runner{Out: &out}}
	code, err := a.workloadCommand(context.Background(), []string{"snapshot", "--workload", "demo"})
	requireCLISafeFault(t, err, fault.Trust, "policy file")
	if code != 0 || out.Len() != 0 {
		t.Fatal("missing policy returned success output")
	}
	code, err = a.workloadCommand(context.Background(), []string{"snapshot", "--unknown"}, model.Policy{})
	requireCLISafeFault(t, err, fault.Invalid, "workload arguments")
	if code != 0 || out.Len() != 0 {
		t.Fatal("malformed request returned success output")
	}
	requireMissingPath(t, a.directory)
}

func TestCLISafeWorkloadRejectsNonRunTrailingArguments(t *testing.T) {
	got, err := parseWorkloadRequest([]string{"snapshot", "--workload", "demo", "unexpected"})
	requireCLISafeFault(t, err, fault.Invalid, "unexpected arguments")
	if !reflect.DeepEqual(got, workloadRequest{}) {
		t.Fatalf("trailing argument returned partial request: %#v", got)
	}
}

func TestCLISafeTargetSelectionAndPublicationRefusal(t *testing.T) {
	workload := model.Workload{Bindings: []model.Binding{{Name: "VALUE", AllowPublish: false}}, Targets: map[string]model.Target{"first": {Type: "file"}, "second": {Type: "file"}}}
	selected, err := selectedTargets(workload, "second")
	if err != nil || !reflect.DeepEqual(selected, []string{"second"}) {
		t.Fatalf("selected targets = %v/%v", selected, err)
	}
	err = apply(context.Background(), workload, "second", model.Generation{})
	requireCLISafeFault(t, err, fault.Denied, "publication permission")
}

func TestCLISafeProjectionRejectsDynamicValuesBeforeCallback(t *testing.T) {
	values := []model.Value{{Leased: true}, {Renewable: true}, {ExpiresAt: time.Unix(2000000000, 0)}}
	for _, value := range values {
		cliSafeProjectionRefusal(t, value)
	}
	calls := 0
	operations := targetOperations{project: func(context.Context, model.Workload, string, model.Generation) error {
		calls++
		return nil
	}}
	if err := projectTarget(context.Background(), model.Workload{}, model.Generation{}, "target", operations); err != nil || calls != 1 {
		t.Fatalf("explicit empty-generation callback control = %v/%d", err, calls)
	}
}

func cliSafeProjectionRefusal(t *testing.T, value model.Value) {
	t.Helper()
	calls := 0
	operations := targetOperations{project: func(context.Context, model.Workload, string, model.Generation) error {
		calls++
		return nil
	}}
	generation := model.Generation{Workload: "demo", Values: map[string]model.Value{"VALUE": value}}
	err := projectTarget(context.Background(), model.Workload{}, generation, "target", operations)
	requireCLISafeFault(t, err, fault.Denied, "static projection required")
	if calls != 0 || !reflect.DeepEqual(generation.Values["VALUE"], value) {
		t.Fatal("dynamic value reached projection or changed input")
	}
}
