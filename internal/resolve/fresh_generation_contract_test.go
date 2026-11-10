package resolve

import (
	"context"
	"testing"

	"github.com/maldous/CircularQuay/internal/fault"
	"github.com/maldous/CircularQuay/internal/model"
	"github.com/maldous/CircularQuay/internal/source"
)

func TestFreshGenerationAllowsLiveOnlyBindingsAndDynamicValues(t *testing.T) {
	for _, test := range []struct {
		name  string
		setup func(*model.Workload)
		value model.Value
	}{
		{
			name: "online explicit non-offline binding",
			setup: func(workload *model.Workload) {
				workload.Mode = "online"
				workload.Snapshot = model.Snapshot{}
				setAllowOffline(workload, false)
			},
		},
		{
			name: "hybrid explicit non-offline binding",
			setup: func(workload *model.Workload) {
				workload.Snapshot = model.Snapshot{}
				setAllowOffline(workload, false)
			},
		},
		{
			name:  "leased provider value",
			setup: func(*model.Workload) {},
			value: model.Value{Leased: true},
		},
		{
			name:  "renewable provider value",
			setup: func(*model.Workload) {},
			value: model.Value{Renewable: true},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			engine, _ := fixture(t)
			workload := engine.Policy.Workloads["demo"]
			test.setup(&workload)
			engine.Policy.Workloads["demo"] = workload
			engine.Reader = freshGenerationReader(test.value)

			generation, err := engine.Resolve(context.Background(), "demo")
			if err != nil || len(generation.Values) != len(workload.Bindings) || generation.Offline {
				t.Fatalf("fresh resolution generation=%+v err=%v", generation, err)
			}
		})
	}
}

func TestSnapshotRejectsLiveOnlyAndDynamicFreshGeneration(t *testing.T) {
	for _, test := range []struct {
		name  string
		setup func(*model.Workload)
		value model.Value
	}{
		{
			name: "non-offline binding",
			setup: func(workload *model.Workload) {
				setAllowOffline(workload, false)
			},
		},
		{name: "leased value", setup: func(*model.Workload) {}, value: model.Value{Leased: true}},
		{name: "renewable value", setup: func(*model.Workload) {}, value: model.Value{Renewable: true}},
	} {
		t.Run(test.name, func(t *testing.T) {
			engine, _ := fixture(t)
			workload := engine.Policy.Workloads["demo"]
			test.setup(&workload)
			engine.Policy.Workloads["demo"] = workload
			engine.Reader = freshGenerationReader(test.value)

			generation, err := engine.Resolve(context.Background(), "demo")
			if err != nil {
				t.Fatalf("fresh generation error = %v", err)
			}
			if err = engine.Snapshot(context.Background(), generation); !fault.Is(err, fault.Denied) {
				t.Fatalf("ineligible snapshot error = %v", err)
			}
		})
	}
}

func TestFreshDenialStillPersistsFallbackFence(t *testing.T) {
	engine, _ := fixture(t)
	engine.Reader = source.ReadFunc(func(context.Context, model.Source, model.Binding) (model.Value, error) {
		return model.Value{}, fault.New(fault.Denied, "synthetic denial")
	})
	if _, err := engine.ResolveFresh(context.Background(), "demo"); !fault.Is(err, fault.Denied) {
		t.Fatalf("fresh denial = %v", err)
	}
	record := readStateRecord(t, engine)
	if record.State != "blocked" || !record.FallbackFence {
		t.Fatalf("fresh denial did not persist fence: %+v", record)
	}
}

func freshGenerationReader(options model.Value) source.Reader {
	return source.ReadFunc(func(_ context.Context, _ model.Source, binding model.Binding) (model.Value, error) {
		value := options
		value.Data = "synthetic-" + binding.Name
		value.Version = "v1"
		return value, nil
	})
}

func setAllowOffline(workload *model.Workload, allowed bool) {
	for index := range workload.Bindings {
		workload.Bindings[index].AllowOffline = allowed
	}
}

func TestFreshGenerationRejectsIncompleteAndMismatchedProvenance(t *testing.T) {
	for _, test := range []struct {
		name  string
		alter func(*model.Generation)
	}{
		{
			name: "fewer values",
			alter: func(generation *model.Generation) {
				delete(generation.Values, "OTHER")
			},
		},
		{
			name: "more values",
			alter: func(generation *model.Generation) {
				generation.Values["EXTRA"] = model.Value{Data: "synthetic-extra", Version: "v1"}
			},
		},
		{
			name: "fewer receipts",
			alter: func(generation *model.Generation) {
				generation.Receipts = generation.Receipts[:1]
			},
		},
		{
			name: "more receipts",
			alter: func(generation *model.Generation) {
				generation.Receipts = append(generation.Receipts, model.Receipt{Source: "extra", Version: "v1"})
			},
		},
		{
			name: "out of order receipts",
			alter: func(generation *model.Generation) {
				generation.Receipts[0], generation.Receipts[1] = generation.Receipts[1], generation.Receipts[0]
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			engine, _ := fixture(t)
			generation, err := engine.Resolve(context.Background(), "demo")
			if err != nil {
				t.Fatal(err)
			}
			test.alter(&generation)
			if err = validateFreshGeneration(engine.Policy.Workloads["demo"], generation, engine.now(), false); !fault.Is(err, fault.Trust) {
				t.Fatalf("invalid fresh generation error = %v", err)
			}
		})
	}
}
