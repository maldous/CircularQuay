package resolve

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/maldous/CircularQuay/internal/fault"
	"github.com/maldous/CircularQuay/internal/model"
)

func TestCompletionResolveCancelledInnerBoundariesRetainGeneration(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	engine := &Engine{}
	generation := model.Generation{Workload: "sentinel"}
	before := generation
	for _, err := range []error{
		engine.readBinding(ctx, &generation, model.Binding{Name: "VALUE", Source: "absent"}, false),
		engine.restoreAllowed(ctx, "absent", model.Workload{}),
	} {
		if !fault.Is(err, fault.Apply) || err.Error() != "activation_failed: resolution canceled" {
			t.Fatal("cancellation crossed the no-collaborator boundary", err)
		}
	}
	if !reflect.DeepEqual(generation, before) {
		t.Fatal("canceled read changed generation")
	}
}

func TestCompletionResolveEqualCountsDoNotProveBindingMembership(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	workload := model.Workload{Bindings: []model.Binding{{Name: "VALUE", Source: "fixture"}}}
	value := model.Value{Data: "owned synthetic", Version: "v1"}
	generation := model.Generation{
		Values:      map[string]model.Value{"VALUE": value},
		Receipts:    []model.Receipt{{Source: "fixture", Version: "v1"}},
		RetrievedAt: now,
	}
	if err := validateFreshGeneration(workload, generation, now, false); err != nil {
		t.Fatal("complete live generation positive control failed", err)
	}
	generation.Values = map[string]model.Value{"OTHER": value}
	if !completeGeneration(workload, generation, now) {
		t.Fatal("fixture did not isolate membership after count admission")
	}
	err := validateFreshGeneration(workload, generation, now, false)
	if !fault.Is(err, fault.Trust) || err.Error() != "trust_failure: fresh generation" || len(generation.Values) != 1 || generation.Values["OTHER"] != value {
		t.Fatal("same-count wrong binding escaped or mutated result", err)
	}
}
