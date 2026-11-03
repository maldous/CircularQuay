package cli

import (
	"context"
	"errors"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/maldous/CircularQuay/internal/fault"
	"github.com/maldous/CircularQuay/internal/model"
	"github.com/maldous/CircularQuay/internal/state"
)

func reconciliationFixture(t *testing.T) (state.Store, model.Workload, model.Generation) {
	t.Helper()
	dir := privateCLIDirectory(t)
	w := model.Workload{Mode: "offline", Targets: map[string]model.Target{"a": {Type: "systemd", Service: "synthetic-a.service", Activation: "restart"}, "b": {Type: "systemd", Service: "synthetic-b.service", Activation: "reload"}}}
	g := model.Generation{Workload: "demo", RetrievedAt: time.Now().UTC(), Values: map[string]model.Value{"TOKEN": {Data: "synthetic"}}}
	return state.Store{Directory: filepath.Join(dir, "state")}, w, g
}

func TestReconciliationProjectsBeforeActivation(t *testing.T) {
	s, w, g := reconciliationFixture(t)
	var calls []string
	operations := targetOperations{
		project: func(_ context.Context, _ model.Workload, id string, _ model.Generation) error {
			calls = append(calls, "project:"+id)
			return nil
		},
		activate: func(_ context.Context, target model.Target) error {
			calls = append(calls, "activate:"+target.Service)
			record, err := s.Read("demo")
			if err != nil || record.State != "pending_activation" {
				t.Fatal("activation was not durably marked pending")
			}
			return nil
		},
	}
	ids, err := selectedTargets(w, "")
	if err != nil {
		t.Fatal(err)
	}
	result, err := reconcileTargets(context.Background(), s, w, g, ids, operations)
	expected := []string{"project:a", "project:b", "activate:synthetic-a.service", "activate:synthetic-b.service"}
	if err != nil || !reflect.DeepEqual(calls, expected) || result.State != "activated" || result.Health != "unknown" {
		t.Fatalf("incorrect transition order/status: %v, %v", calls, err)
	}
}

func TestProjectionFailurePreventsAllActivation(t *testing.T) {
	s, w, g := reconciliationFixture(t)
	activationCalls := 0
	operations := targetOperations{
		project: func(_ context.Context, _ model.Workload, id string, _ model.Generation) error {
			if id == "b" {
				return fault.New(fault.Apply, "synthetic failure")
			}
			return nil
		},
		activate: func(context.Context, model.Target) error { activationCalls++; return nil },
	}
	result, err := reconcileTargets(context.Background(), s, w, g, []string{"a", "b"}, operations)
	if err == nil || activationCalls != 0 || result.State != "apply_failed" || result.Targets["a"].State != "applied" {
		t.Fatal("partial projection was activated or hidden")
	}
	saved, readErr := s.Read("demo")
	if readErr != nil || saved.State != "apply_failed" {
		t.Fatal("projection failure not durable")
	}
}

func TestFailedActivationRemainsPendingAndRetries(t *testing.T) {
	s, w, g := reconciliationFixture(t)
	attempts := 0
	operations := targetOperations{project: func(context.Context, model.Workload, string, model.Generation) error { return nil }, activate: func(context.Context, model.Target) error {
		attempts++
		if attempts == 1 {
			return errors.New("synthetic failure")
		}
		return nil
	}}
	first, err := reconcileTargets(context.Background(), s, w, g, []string{"a"}, operations)
	if err == nil || first.State != "partial_pending_activation" || !first.Targets["a"].ActivatedAt.IsZero() {
		t.Fatal("failed activation reported successful")
	}
	second, err := reconcileTargets(context.Background(), s, w, g, []string{"a"}, operations)
	if err != nil || second.State != "partial_activated" || attempts != 2 {
		t.Fatal("failed activation was not retried")
	}
}

func TestTargetedReconciliationRetainsPriorTargetsWithoutGlobalActivation(t *testing.T) {
	s, w, g := reconciliationFixture(t)
	failActivation := targetOperations{
		project:  func(context.Context, model.Workload, string, model.Generation) error { return nil },
		activate: func(context.Context, model.Target) error { return errors.New("synthetic activation failure") },
	}
	if _, err := reconcileTargets(context.Background(), s, w, g, []string{"b"}, failActivation); err == nil {
		t.Fatal("fixture activation unexpectedly succeeded")
	}
	result, err := reconcileTargets(context.Background(), s, w, g, []string{"a"}, targetOperations{
		project:  func(context.Context, model.Workload, string, model.Generation) error { return nil },
		activate: func(context.Context, model.Target) error { return nil },
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.State != "partial_activated" {
		t.Fatalf("targeted activation claimed a complete workload: state=%q error=%v", result.State, err)
	}
	assertCurrentActivated(t, result.Targets["a"], result.Generation)
	assertRetainedPending(t, result.Targets["b"])
	saved, err := s.Read("demo")
	if err != nil {
		t.Fatal(err)
	}
	if saved.State != "partial_activated" || saved.Targets["b"].Current {
		t.Fatalf("partial state was not durable: %+v, %v", saved, err)
	}
}

func assertCurrentActivated(t *testing.T, target state.TargetRecord, generation string) {
	t.Helper()
	if !target.Current || target.Generation != generation || target.State != "activated" {
		t.Fatalf("selected target lacks current generation record: %+v", target)
	}
}

func assertRetainedPending(t *testing.T, target state.TargetRecord) {
	t.Helper()
	if target.Current || target.State != "pending_activation" || target.Generation == "" {
		t.Fatalf("prior pending target was hidden or relabeled: %+v", target)
	}
}

func TestCanceledReconciliationDoesNotPublish(t *testing.T) {
	s, w, g := reconciliationFixture(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	calls := 0
	operations := targetOperations{project: func(context.Context, model.Workload, string, model.Generation) error { calls++; return nil }}
	if _, err := reconcileTargets(ctx, s, w, g, []string{"a"}, operations); err == nil || calls != 0 {
		t.Fatal("canceled generation published")
	}
}
