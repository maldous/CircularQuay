package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCancelledInitializationHasNoFilesystemEffects(t *testing.T) {
	for _, command := range [][]string{{"init"}, {"config", "init"}} {
		t.Run(strings.Join(command, "/"), func(t *testing.T) {
			assertCancelledInitialization(t, command)
		})
	}
}

func assertCancelledInitialization(t *testing.T, command []string) {
	t.Helper()
	directory := filepath.Join(t.TempDir(), "not-created")
	policy := filepath.Join(directory, "cq.json")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	var out, diagnostics bytes.Buffer
	args := append(append([]string{}, command...), "--output", policy)
	status := (Runner{In: strings.NewReader(""), Out: &out, Err: &diagnostics}).Run(ctx, args)
	if status != 1 || out.Len() != 0 {
		t.Fatalf("cancelled initialization status=%d, stdout bytes=%d", status, out.Len())
	}
	if _, err := os.Lstat(directory); !os.IsNotExist(err) {
		t.Fatal("cancelled initialization created or inspected an unexpected output directory")
	}
	var diagnostic map[string]string
	if err := json.Unmarshal(diagnostics.Bytes(), &diagnostic); err != nil {
		t.Fatal("cancelled initialization did not return a category-only diagnostic")
	}
	if len(diagnostic) != 1 || diagnostic["error"] != "unavailable" {
		t.Fatal("cancelled initialization returned an unexpected diagnostic")
	}
}
