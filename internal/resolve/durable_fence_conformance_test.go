package resolve

import (
	"context"
	"os"
	"runtime"
	"testing"

	"github.com/maldous/CircularQuay/internal/fault"
	"github.com/maldous/CircularQuay/internal/model"
	"github.com/maldous/CircularQuay/internal/source"
)

func TestCrashAfterPersistedProviderFenceDeniesSnapshotAfterRestart(t *testing.T) {
	engine, snapshots := fixture(t)
	seedFallbackSnapshot(t, engine)
	workload := engine.Policy.Workloads["demo"]
	if _, err := engine.prepareProviderRead("demo", workload); err != nil {
		t.Fatal(err)
	}
	if record := readStateRecord(t, engine); !record.FallbackFence {
		t.Fatalf("pre-read fence was not persisted: %+v", record)
	}

	restarted := restartEngine(engine, snapshots, unavailableReader())
	if _, err := restarted.Resolve(context.Background(), "demo"); !fault.Is(err, fault.Denied) {
		t.Fatalf("restart after pre-read crash restored snapshot: %v", err)
	}
}

func TestUnavailableRestoreWriteFailureRetainsFenceAfterRestart(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX directory write permissions")
	}
	engine, snapshots := fixture(t)
	seedFallbackSnapshot(t, engine)
	calls := 0
	engine.Reader = source.ReadFunc(func(context.Context, model.Source, model.Binding) (model.Value, error) {
		calls++
		if err := os.Chmod(engine.State.Directory, 0o500); err != nil {
			t.Fatal(err)
		}
		return model.Value{}, fault.New(fault.Unavailable, "synthetic transport")
	})
	t.Cleanup(func() { _ = os.Chmod(engine.State.Directory, 0o700) })
	if _, err := engine.Resolve(context.Background(), "demo"); err == nil {
		t.Fatal("availability result with failed fence restoration succeeded")
	} else if fault.Is(err, fault.Unavailable) {
		t.Fatalf("fence restoration unexpectedly succeeded: %v", err)
	}
	if calls != 1 {
		t.Fatalf("provider calls = %d, want one", calls)
	}
	if err := os.Chmod(engine.State.Directory, 0o700); err != nil {
		t.Fatal(err)
	}
	if record := readStateRecord(t, engine); !record.FallbackFence {
		t.Fatalf("failed fence restoration cleared durable fence: %+v", record)
	}

	restarted := restartEngine(engine, snapshots, unavailableReader())
	if _, err := restarted.Resolve(context.Background(), "demo"); !fault.Is(err, fault.Denied) {
		t.Fatalf("restart after failed fence restoration restored snapshot: %v", err)
	}
}
