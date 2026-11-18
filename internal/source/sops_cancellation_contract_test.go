package source

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/maldous/CircularQuay/internal/fault"
	"github.com/maldous/CircularQuay/internal/model"
)

func assertSOPSCancellationPrecedesAccess(t *testing.T, call func(context.Context) error, ordinary fault.Kind) {
	t.Helper()
	if err := call(context.Background()); !fault.Is(err, ordinary) {
		t.Fatalf("uncancelled control did not reach the selected boundary: %v", err)
	}
	for _, deadline := range []bool{false, true} {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		if deadline {
			ctx, cancel = context.WithDeadline(context.Background(), time.Unix(0, 0))
			defer cancel()
		}
		if err := call(ctx); !fault.Is(err, fault.Denied) {
			t.Fatalf("cancelled operation reached access instead of denying: %v", err)
		}
	}
}

func TestSOPSReaderCancellationBeforeSourceAccess(t *testing.T) {
	file := filepath.Join(privateSourceDirectory(t, "cancelled-sops-reader"), "missing-source.json")
	config := model.Source{Type: "sops", Path: file}
	assertSOPSCancellationPrecedesAccess(t, func(ctx context.Context) error {
		_, err := (SOPSReader{}).Read(ctx, config, model.Binding{Name: "KEY"})
		return err
	}, fault.Missing)
}

func TestSOPSSnapshotCancellationBeforeSourceAccess(t *testing.T) {
	file := filepath.Join(privateSourceDirectory(t, "cancelled-sops-snapshot"), "missing-source.json")
	assertSOPSCancellationPrecedesAccess(t, func(ctx context.Context) error {
		_, err := DecryptSnapshot(ctx, model.Snapshot{Path: file})
		return err
	}, fault.Missing)
}

func TestSOPSHelperCancellationBeforeIdentityAccess(t *testing.T) {
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	identity := filepath.Join(privateSourceDirectory(t, "cancelled-sops-identity"), "missing-identity")
	assertSOPSCancellationPrecedesAccess(t, func(ctx context.Context) error {
		_, err := runSOPS(ctx, executable, sopsDecryptArgs(), nil, identity, "", nil)
		return err
	}, fault.Missing)
}

func TestSOPSEncryptCancellationBeforeHelperAccess(t *testing.T) {
	directory := privateSourceDirectory(t, "cancelled-sops-encrypt")
	config := model.Snapshot{Path: filepath.Join(directory, "snapshot.json"), Recipients: []string{"age1synthetic"}, SOPS: filepath.Join(directory, "missing-helper")}
	assertSOPSCancellationPrecedesAccess(t, func(ctx context.Context) error {
		return EncryptSnapshot(ctx, config, map[string]string{"KEY": "synthetic"})
	}, fault.Unsupported)
}
