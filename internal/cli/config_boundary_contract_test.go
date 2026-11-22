package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"reflect"
	"testing"

	"github.com/maldous/CircularQuay/internal/fault"
)

func TestConfigBoundaryContractFormatterRespectsOwnedLock(t *testing.T) {
	path := configFixture(t)
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	release := configBoundaryOwnedLock(t, path)
	app := application{config: path}
	changed, err := app.formatConfigContext(context.Background())
	if changed {
		t.Fatal("contended formatter reported a committed change")
	}
	configBoundaryFault(t, err, fault.Unavailable, "reconciliation busy")
	assertRawConfigPreserved(t, path, before)
	release()
	configBoundaryOwnedLock(t, path)()

	configBoundaryFormattingControl(t, app, path, before)
}

func configBoundaryFormattingControl(t *testing.T, app application, path string, before []byte) {
	t.Helper()
	changed, err := app.formatConfigContext(context.Background())
	if err != nil || !changed {
		t.Fatalf("released formatter failed its positive control: changed=%v err=%v", changed, err)
	}
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(before, after) || !bytes.HasSuffix(after, []byte("\n")) {
		t.Fatal("formatter did not replace the compact declaration")
	}
	var original, formatted any
	if json.Unmarshal(before, &original) != nil || json.Unmarshal(after, &formatted) != nil || !reflect.DeepEqual(original, formatted) {
		t.Fatal("formatting changed the declaration instead of its representation")
	}
	configBoundaryOwnedLock(t, path)()
}

func TestConfigBoundaryContractFormatterRefusalsReleaseLock(t *testing.T) {
	for _, scenario := range []struct {
		name      string
		malformed bool
		kind      fault.Kind
		operation string
	}{
		{"malformed current bytes", true, fault.Invalid, "config JSON"},
		{"cancelled caller", false, fault.Unavailable, "configuration cancelled"},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			configBoundaryFormatterRefusal(t, scenario.malformed, scenario.kind, scenario.operation)
		})
	}
}

func configBoundaryFormatterRefusal(t *testing.T, malformed bool, kind fault.Kind, operation string) {
	t.Helper()
	path := configFixture(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if malformed {
		writeProtectedFixture(t, path, []byte(`{"SYNTHETIC_CONFIG_BOUNDARY_PRIVATE":`))
	} else {
		cancel()
	}
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	changed, err := (application{config: path}).formatConfigContext(ctx)
	if changed {
		t.Fatal("refused formatter reported a committed change")
	}
	configBoundaryFault(t, err, kind, operation)
	assertRawConfigPreserved(t, path, before)
	// Failure after locking must release the actual OS lock, not merely leave
	// the declaration untouched. A fresh acquisition is a cleanup oracle.
	configBoundaryOwnedLock(t, path)()
}

func TestConfigBoundaryContractCurrentBytesGuard(t *testing.T) {
	for _, scenario := range []struct {
		name    string
		change  string
		repair  bool
		matches bool
	}{
		{"unchanged declaration", "none", false, true},
		{"changed valid declaration", "valid", false, false},
		{"removed declaration", "removed", false, false},
		{"matching raw repair", "raw", true, true},
		{"stale raw repair", "raw", true, false},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			configBoundaryCurrentBytes(t, scenario.change, scenario.repair, scenario.matches)
		})
	}
}

func configBoundaryCurrentBytes(t *testing.T, change string, repair, matches bool) {
	t.Helper()
	path := configFixture(t)
	original, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	expected := rawConfigSHA(original)
	current := original
	switch change {
	case "valid":
		current = bytes.ReplaceAll(original, []byte("fixture.json"), []byte("changed.json"))
		writeProtectedFixture(t, path, current)
	case "raw":
		current = []byte(`{"SYNTHETIC_CONFIG_BOUNDARY_PRIVATE":`)
		writeProtectedFixture(t, path, current)
	case "removed":
		if err := os.Remove(path); err != nil {
			t.Fatal(err)
		}
		current = nil
	}
	if repair && matches {
		expected = rawConfigSHA(current)
	}
	err = (application{config: path}).ensureConfigUnchanged(path, expected, repair)
	configBoundaryGuardResult(t, err, matches)
	assertRawConfigPreserved(t, path, current)
	configBoundaryOwnedLock(t, path)()
}

func configBoundaryGuardResult(t *testing.T, err error, matches bool) {
	t.Helper()
	if matches {
		if err != nil {
			t.Fatalf("unchanged bytes were refused: %v", err)
		}
		return
	}
	configBoundaryFault(t, err, fault.Denied, "config changed")
}

func configBoundaryFault(t *testing.T, err error, kind fault.Kind, operation string) {
	t.Helper()
	failure, ok := err.(*fault.Error)
	if !ok || failure.Kind != kind || failure.Operation != operation || err.Error() != string(kind)+": "+operation {
		t.Fatalf("unexpected typed refusal: %v", err)
	}
}

func configBoundaryOwnedLock(t *testing.T, path string) func() {
	t.Helper()
	unlock, err := configLock(path)
	if err != nil {
		t.Fatalf("owned config lock acquisition failed: %v", err)
	}
	released := false
	release := func() {
		t.Helper()
		if !released {
			released = true
			if err := unlock(); err != nil {
				t.Errorf("owned config lock release failed: %v", err)
			}
		}
	}
	t.Cleanup(release)
	return release
}
