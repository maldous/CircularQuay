package policy

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/maldous/CircularQuay/internal/fault"
	"github.com/maldous/CircularQuay/internal/model"
)

func TestSnapshotDeclarationContract(t *testing.T) {
	validRecipient := "age1" + strings.Repeat("q", 58)
	native := filepath.Join(t.TempDir(), "native")
	valid := model.Workload{
		Mode:     "hybrid",
		Bindings: []model.Binding{{AllowOffline: true}},
		Snapshot: model.Snapshot{
			Path:         filepath.Join(native, "snapshot.sops.json"),
			Recipients:   []string{validRecipient},
			IdentityFile: filepath.Join(native, "age.txt"),
			SOPS:         filepath.Join(native, "sops"),
		},
	}
	if err := validateSnapshotDeclaration(model.Workload{}); err != nil {
		t.Fatal("empty snapshot object rejected")
	}
	if err := validateSnapshotDeclaration(valid); err != nil {
		t.Fatal("complete static snapshot rejected")
	}
	for _, mutate := range []func(*model.Workload){
		func(w *model.Workload) { w.Snapshot.Path = "" },
		func(w *model.Workload) { w.Snapshot.Recipients = nil },
		func(w *model.Workload) { w.Snapshot.IdentityFile = "relative" },
		func(w *model.Workload) { w.Snapshot.SOPS = "sops" },
		func(w *model.Workload) { w.Snapshot.Recipients = []string{validRecipient, validRecipient} },
		func(w *model.Workload) { w.Snapshot.Recipients = []string{"age1INVALID"} },
		func(w *model.Workload) { w.Bindings[0].AllowOffline = false },
		func(w *model.Workload) { w.Mode = "online" },
		func(w *model.Workload) { w.AuditRequired = true },
	} {
		candidate := valid
		candidate.Snapshot.Recipients = append([]string(nil), valid.Snapshot.Recipients...)
		candidate.Bindings = append([]model.Binding(nil), valid.Bindings...)
		mutate(&candidate)
		if err := validateSnapshotDeclaration(candidate); !fault.Is(err, fault.Invalid) {
			t.Fatal("invalid snapshot declaration accepted")
		}
	}
}
