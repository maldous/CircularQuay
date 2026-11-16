package state

import (
	"strings"
	"testing"
	"time"

	"github.com/maldous/CircularQuay/internal/fault"
)

func TestFailedMetadataEncodingPreservesPreviousGeneration(t *testing.T) {
	store := Store{Directory: privateStateDirectory(t)}
	previous := Record{Workload: "demo", Generation: "previous", State: "ready"}
	if err := store.Write(previous); err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name   string
		record Record
		kind   fault.Kind
	}{
		{"unencodable time", Record{Workload: "demo", LastChecked: time.Date(10000, 1, 1, 0, 0, 0, 0, time.UTC)}, fault.Apply},
		{"oversized record", Record{Workload: "demo", Generation: strings.Repeat("x", 1<<20)}, fault.Invalid},
		{"invalid workload", Record{Workload: "../escape"}, fault.Invalid},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			if err := store.Write(test.record); !fault.Is(err, test.kind) {
				t.Fatalf("failed record was not rejected with %s: %v", test.kind, err)
			}
			actual, err := store.Read("demo")
			if err != nil || actual.Generation != previous.Generation || actual.State != previous.State {
				t.Fatalf("failed write replaced the previous generation: %#v err=%v", actual, err)
			}
		})
	}
}

func TestInvalidLockAndRepeatedReleaseDoNotStrandReconciliation(t *testing.T) {
	store := Store{Directory: privateStateDirectory(t)}
	if release, err := store.Lock("../escape"); release != nil || !fault.Is(err, fault.Invalid) {
		t.Fatalf("invalid workload acquired a lock: release=%v err=%v", release != nil, err)
	}
	release := requireLock(t, store)
	requireUnlock(t, release)
	if err := release(); !fault.Is(err, fault.Apply) {
		t.Fatalf("repeated release falsely reported success: %v", err)
	}
	// A failed second cleanup must not retain the original OS lock.
	requireUnlock(t, requireLock(t, store))
}
