package resolve

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/maldous/CircularQuay/internal/fault"
	"github.com/maldous/CircularQuay/internal/model"
	"github.com/maldous/CircularQuay/internal/source"
	"github.com/maldous/CircularQuay/internal/state"
)

func TestProviderSnapshotMatrixContractMissingSource(t *testing.T) {
	engine, _ := fixture(t)
	engine.State = state.Store{}
	delete(engine.Policy.Sources, "a")
	engine.Reader = source.ReadFunc(func(context.Context, model.Source, model.Binding) (model.Value, error) {
		t.Fatal("missing declaration reached provider")
		return model.Value{}, nil
	})
	result, err := engine.Resolve(context.Background(), "demo")
	assertMappedResolveError(t, err, fault.Invalid, "source")
	assertMappedZeroGeneration(t, result)
}

func TestProviderSnapshotMatrixContractProviderBounds(t *testing.T) {
	for _, version := range []string{strings.Repeat("v", 1025), string([]byte{0xff})} {
		engine, _ := fixture(t)
		engine.State = state.Store{}
		calls := 0
		engine.Reader = source.ReadFunc(func(context.Context, model.Source, model.Binding) (model.Value, error) {
			calls++
			return model.Value{Data: "synthetic", Version: version}, nil
		})
		result, err := engine.Resolve(context.Background(), "demo")
		assertMappedResolveError(t, err, fault.Trust, "source value bounds")
		assertMappedZeroGeneration(t, result)
		if calls != 1 {
			t.Fatalf("invalid response allowed later read: %d", calls)
		}
	}
}

func TestProviderSnapshotMatrixContractCancellation(t *testing.T) {
	for _, key := range []bool{false, true} {
		assertMappedProviderCancellation(t, key)
	}
	engine, snapshots := fixture(t)
	engine.State = state.Store{}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	engine.Reader = source.ReadFunc(func(context.Context, model.Source, model.Binding) (model.Value, error) {
		t.Fatal("pre-cancelled key read called provider")
		return model.Value{}, nil
	})
	value, err := engine.ReadKey(ctx, "demo", "TOKEN")
	assertMappedResolveError(t, err, fault.Apply, "resolution canceled")
	assertMappedZeroValue(t, value)
	assertMappedResolveError(t, engine.Snapshot(ctx, model.Generation{}), fault.Apply, "resolution canceled")
	if snapshots.saved != nil {
		t.Fatal("cancelled snapshot wrote data")
	}
}

func assertMappedProviderCancellation(t *testing.T, key bool) {
	t.Helper()
	engine, _ := fixture(t)
	engine.State = state.Store{}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	calls := 0
	engine.Reader = source.ReadFunc(func(context.Context, model.Source, model.Binding) (model.Value, error) {
		calls++
		cancel()
		return model.Value{Data: "must-not-escape", Version: "v1"}, nil
	})
	if key {
		value, err := engine.ReadKey(ctx, "demo", "TOKEN")
		assertMappedResolveError(t, err, fault.Apply, "resolution canceled")
		assertMappedZeroValue(t, value)
	} else {
		generation, err := engine.Resolve(ctx, "demo")
		assertMappedResolveError(t, err, fault.Apply, "resolution canceled")
		assertMappedZeroGeneration(t, generation)
	}
	if calls != 1 {
		t.Fatalf("cancellation calls=%d", calls)
	}
}

func TestProviderSnapshotMatrixContractExactKeyUnavailable(t *testing.T) {
	engine, snapshots := fixture(t)
	engine.State = state.Store{}
	control, err := engine.ReadKey(context.Background(), "demo", "TOKEN")
	if err != nil || control.Data != "synthetic-only-TOKEN" || control.Version != "v1" {
		t.Fatalf("valid exact-key control: %v", err)
	}
	cause := fault.New(fault.Unavailable, "synthetic unavailable")
	engine.Reader = source.ReadFunc(func(context.Context, model.Source, model.Binding) (model.Value, error) { return model.Value{}, cause })
	snapshots.errorRead = errors.New("snapshot must not be selected")
	value, err := engine.ReadKey(context.Background(), "demo", "TOKEN")
	if err != cause {
		t.Fatalf("unavailable exact read changed cause or used fallback: %v", err)
	}
	assertMappedZeroValue(t, value)
}

func TestProviderSnapshotMatrixContractOfflineReadRefusal(t *testing.T) {
	engine, snapshots := fixture(t)
	engine.State = state.Store{}
	workload := engine.Policy.Workloads["demo"]
	workload.Mode = "offline"
	engine.Policy.Workloads["demo"] = workload
	snapshots.errorRead = fault.New(fault.Trust, "synthetic snapshot refusal")
	value, err := engine.ReadKey(context.Background(), "demo", "TOKEN")
	assertMappedResolveError(t, err, fault.Trust, "synthetic snapshot refusal")
	assertMappedZeroValue(t, value)
	workload.AuditRequired = true
	engine.Policy.Workloads["demo"] = workload
	result, err := engine.Resolve(context.Background(), "demo")
	assertMappedResolveError(t, err, fault.Denied, "snapshot audit policy")
	assertMappedZeroGeneration(t, result)
}

func TestProviderSnapshotMatrixContractRestoreProvenance(t *testing.T) {
	engine, snapshots := mappedOfflineSnapshot(t)
	control, err := engine.Resolve(context.Background(), "demo")
	if err != nil || !control.Offline || len(control.Values) != 2 {
		t.Fatalf("valid offline control: %v", err)
	}
	var payload snapshotPayload
	if err := json.Unmarshal([]byte(snapshots.saved["cq_snapshot"]), &payload); err != nil {
		t.Fatal(err)
	}
	payload.Generation.Receipts[0].Version = "different-version"
	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	snapshots.saved["cq_snapshot"] = string(data)
	result, err := engine.Resolve(context.Background(), "demo")
	assertMappedResolveError(t, err, fault.Trust, "snapshot provenance")
	assertMappedZeroGeneration(t, result)
	if snapshots.saved["cq_snapshot"] != string(data) {
		t.Fatal("refused snapshot mutated")
	}
}

func TestProviderSnapshotMatrixContractCancelAfterDecrypt(t *testing.T) {
	engine, snapshots := mappedOfflineSnapshot(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	calls := 0
	engine.Snapshots = mappedSnapshotCallbacks{
		encrypt: func(context.Context, model.Snapshot, map[string]string) error { t.Fatal("unexpected encrypt"); return nil },
		decrypt: func(context.Context, model.Snapshot) (map[string]string, error) {
			calls++
			cancel()
			return snapshots.saved, nil
		},
	}
	result, err := engine.Resolve(ctx, "demo")
	assertMappedResolveError(t, err, fault.Apply, "resolution canceled")
	assertMappedZeroGeneration(t, result)
	if calls != 1 {
		t.Fatalf("decrypt calls=%d", calls)
	}
}

func TestProviderSnapshotMatrixContractEncodedSize(t *testing.T) {
	engine, snapshots := fixture(t)
	engine.State = state.Store{}
	data := strings.Repeat("x", 1<<20)
	engine.Reader = source.ReadFunc(func(context.Context, model.Source, model.Binding) (model.Value, error) {
		return model.Value{Data: data, Version: "v1"}, nil
	})
	generation, err := engine.ResolveFresh(context.Background(), "demo")
	if err != nil || len(generation.Values) != 2 {
		t.Fatalf("individually bounded values refused: %v", err)
	}
	assertMappedResolveError(t, engine.Snapshot(context.Background(), generation), fault.Apply, "snapshot encode")
	if snapshots.saved != nil || generation.Values["TOKEN"].Data != data || generation.Values["OTHER"].Data != data {
		t.Fatal("oversize encoded snapshot was emitted or generation mutated")
	}
}

func mappedOfflineSnapshot(t *testing.T) (*Engine, *memorySnapshot) {
	t.Helper()
	engine, snapshots := fixture(t)
	engine.State = state.Store{}
	workload := engine.Policy.Workloads["demo"]
	workload.Mode = "offline"
	engine.Policy.Workloads["demo"] = workload
	generation, err := engine.ResolveFresh(context.Background(), "demo")
	if err != nil {
		t.Fatal(err)
	}
	if err := engine.Snapshot(context.Background(), generation); err != nil {
		t.Fatal(err)
	}
	return engine, snapshots
}

type mappedSnapshotCallbacks struct {
	encrypt func(context.Context, model.Snapshot, map[string]string) error
	decrypt func(context.Context, model.Snapshot) (map[string]string, error)
}

func (callbacks mappedSnapshotCallbacks) Encrypt(ctx context.Context, snapshot model.Snapshot, values map[string]string) error {
	return callbacks.encrypt(ctx, snapshot, values)
}

func (callbacks mappedSnapshotCallbacks) Decrypt(ctx context.Context, snapshot model.Snapshot) (map[string]string, error) {
	return callbacks.decrypt(ctx, snapshot)
}

func assertMappedResolveError(t *testing.T, err error, kind fault.Kind, operation string) {
	t.Helper()
	actual, ok := err.(*fault.Error)
	if !ok || actual.Kind != kind || actual.Operation != operation || err.Error() != string(kind)+": "+operation || errors.Unwrap(err) != nil {
		t.Fatalf("unexpected bounded resolution refusal: %v", err)
	}
}

func assertMappedZeroValue(t *testing.T, value model.Value) {
	t.Helper()
	if !reflect.DeepEqual(value, model.Value{}) {
		t.Fatal("refusal exposed a partial value")
	}
}

func assertMappedZeroGeneration(t *testing.T, generation model.Generation) {
	t.Helper()
	if !reflect.DeepEqual(generation, model.Generation{}) {
		t.Fatal("refusal exposed a partial generation")
	}
}
