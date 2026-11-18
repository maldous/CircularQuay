package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/maldous/CircularQuay/internal/platform"
)

func TestRunnerRoutesBootstrapWithoutProviderAccess(t *testing.T) {
	path := filepath.Join(privateCLIDirectory(t), "cq.json")
	if err := platform.CreateProtectedExclusive(path, bootstrapPolicy()); err != nil {
		t.Fatal(err)
	}
	var output, diagnostics bytes.Buffer
	runner := Runner{Out: &output, Err: &diagnostics}
	if code := runner.Run(context.Background(), []string{"--config", path, "bootstrap", "--non-interactive"}); code != 0 {
		t.Fatalf("bootstrap command failed: %d, %s", code, diagnostics.String())
	}
	var receipt map[string]any
	if json.Unmarshal(output.Bytes(), &receipt) != nil || receipt["state"] != "validated" || receipt["policy"] != path {
		t.Fatal("runner did not execute the protected bootstrap command")
	}
	output.Reset()
	diagnostics.Reset()
	if code := runner.Run(context.Background(), []string{"bootstrap", "--config-source", "https://github.com/example/config/tree/main", "--non-interactive"}); code != 2 || output.Len() != 0 {
		t.Fatal("runner did not reject unpinned remote bootstrap before materialization")
	}
}
