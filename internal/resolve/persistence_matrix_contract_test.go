package resolve

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/maldous/CircularQuay/internal/fault"
	"github.com/maldous/CircularQuay/internal/model"
	"github.com/maldous/CircularQuay/internal/source"
)

func TestPersistenceMatrixContractCorruptOwnedState(t *testing.T) {
	for _, offline := range []bool{false, true} {
		assertMappedCorruptState(t, offline)
	}
}

func assertMappedCorruptState(t *testing.T, offline bool) {
	t.Helper()
	engine, _ := fixture(t)
	path := filepath.Join(engine.State.Directory, "demo.json")
	data := []byte(`{"workload":`)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	engine.Reader = source.ReadFunc(func(context.Context, model.Source, model.Binding) (model.Value, error) {
		t.Fatal("corrupt state reached provider")
		return model.Value{}, nil
	})
	engine.Snapshots = mappedSnapshotCallbacks{
		encrypt: func(context.Context, model.Snapshot, map[string]string) error {
			t.Fatal("unexpected encrypt")
			return nil
		},
		decrypt: func(context.Context, model.Snapshot) (map[string]string, error) {
			t.Fatal("corrupt state reached decrypt")
			return nil, nil
		},
	}
	if offline {
		workload := engine.Policy.Workloads["demo"]
		workload.Mode = "offline"
		engine.Policy.Workloads["demo"] = workload
		generation, err := engine.Resolve(context.Background(), "demo")
		assertMappedResolveError(t, err, fault.Trust, "state record")
		assertMappedZeroGeneration(t, generation)
	} else {
		value, err := engine.ReadKey(context.Background(), "demo", "TOKEN")
		assertMappedResolveError(t, err, fault.Trust, "state record")
		assertMappedZeroValue(t, value)
	}
	actual, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(actual, data) {
		t.Fatal("refusal replaced corrupt owned state")
	}
}

func TestPersistenceMatrixContractCompletionWriteRefusals(t *testing.T) {
	for _, mode := range []string{"generation", "key_success", "key_unavailable", "key_denied"} {
		t.Run(mode, func(t *testing.T) { assertMappedCompletionWriteRefusal(t, mode) })
	}
}

func assertMappedCompletionWriteRefusal(t *testing.T, mode string) {
	t.Helper()
	engine, _ := fixture(t)
	path := filepath.Join(engine.State.Directory, "demo.json")
	backup := path + ".retained"
	var fence []byte
	calls := 0
	engine.Reader = source.ReadFunc(func(context.Context, model.Source, model.Binding) (model.Value, error) {
		calls++
		if calls == 1 {
			fence = replaceMappedStateDestination(t, path, backup)
		}
		return mappedCompletionValue(mode)
	})
	if mode == "generation" {
		generation, err := engine.ResolveFresh(context.Background(), "demo")
		assertMappedResolveError(t, err, fault.Trust, "nonregular destination")
		assertMappedZeroGeneration(t, generation)
		if calls != 2 {
			t.Fatalf("complete generation read count=%d", calls)
		}
	} else {
		value, err := engine.ReadKey(context.Background(), "demo", "TOKEN")
		assertMappedResolveError(t, err, fault.Trust, "nonregular destination")
		assertMappedZeroValue(t, value)
		if calls != 1 {
			t.Fatalf("exact-key read count=%d", calls)
		}
	}
	restoreMappedStateDestination(t, engine, path, backup, fence)
}

func mappedCompletionValue(mode string) (model.Value, error) {
	switch mode {
	case "key_unavailable":
		return model.Value{}, fault.New(fault.Unavailable, "synthetic unavailable")
	case "key_denied":
		return model.Value{}, fault.New(fault.Denied, "synthetic denied")
	default:
		return model.Value{Data: "synthetic", Version: "v1"}, nil
	}
}

func replaceMappedStateDestination(t *testing.T, path, backup string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(path, backup); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(path, 0o700); err != nil {
		t.Fatal(err)
	}
	return data
}

func restoreMappedStateDestination(t *testing.T, engine *Engine, path, backup string, fence []byte) {
	t.Helper()
	entries, err := os.ReadDir(path)
	if err != nil || len(entries) != 0 {
		t.Fatal("failed write altered destination directory")
	}
	actual, err := os.ReadFile(backup)
	if err != nil || !bytes.Equal(actual, fence) {
		t.Fatal("failed write altered retained pre-read fence")
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(backup, path); err != nil {
		t.Fatal(err)
	}
	assertMappedRetainedFence(t, engine)
}

func assertMappedRetainedFence(t *testing.T, engine *Engine) {
	t.Helper()
	record, err := engine.State.Read("demo")
	if err != nil || !record.FallbackFence || record.Workload != "demo" {
		t.Fatalf("owned pre-read fence not preserved: %v", err)
	}
}
