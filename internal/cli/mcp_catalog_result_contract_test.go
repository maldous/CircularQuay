package cli

import (
	"encoding/hex"
	"path/filepath"
	"testing"
)

func TestCLISafeMCPUnpinnedWorkloadStaysHidden(t *testing.T) {
	config := configFixture(t)
	a := application{config: config}
	if a.actionAllowed("demo") {
		t.Fatal("unconfigured executable appeared in action catalog")
	}
	requireMissingPath(t, filepath.Join(filepath.Dir(config), "fixture.json"))
}

func TestCLISafeMCPFailedResultRetainsExitStatus(t *testing.T) {
	got, err := actionRunResult(17)
	if err != nil || got.Status != "failed" || got.ExitCode != 17 {
		t.Fatalf("failed result = %#v/%v", got, err)
	}
	id, err := hex.DecodeString(got.ID)
	if err != nil || len(id) != 16 {
		t.Fatalf("result identifier shape = %q/%v", got.ID, err)
	}
}
