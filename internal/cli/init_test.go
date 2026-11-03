package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/maldous/CircularQuay/internal/platform"
	"github.com/maldous/CircularQuay/internal/policy"
)

func TestInitCreatesUsableProtectedPolicyAndFixture(t *testing.T) {
	config := filepath.Join(privateCLIDirectory(t), "cquay.json")
	var output, diagnostics bytes.Buffer
	runner := Runner{Out: &output, Err: &diagnostics}
	if code := runner.Run(context.Background(), []string{"init", "--output", config}); code != 0 {
		t.Fatalf("init failed: status %d, %s", code, diagnostics.String())
	}
	loaded, err := policy.Load(config)
	if err != nil {
		t.Fatal("initialized policy cannot be loaded with native protections")
	}
	fixture, err := platform.ReadProtected(loaded.Sources["demo"].Path, true)
	if err != nil {
		t.Fatal("initialized synthetic fixture is not protected")
	}
	var values map[string]string
	if json.Unmarshal(fixture, &values) != nil || values["DEMO_API_KEY"] != "synthetic-only-not-a-credential" {
		t.Fatal("init did not produce the declared synthetic example")
	}
	output.Reset()
	if code := runner.Run(context.Background(), []string{"--config", config, "plan"}); code != 0 {
		t.Fatalf("initialized policy cannot be planned: status %d, %s", code, diagnostics.String())
	}
}

func TestInitPreservesEitherExistingOutput(t *testing.T) {
	for _, existing := range []string{"cquay.json", "cquay.json.fixture.json"} {
		t.Run(existing, func(t *testing.T) {
			assertInitPreservesExistingOutput(t, existing)
		})
	}
}

func assertInitPreservesExistingOutput(t *testing.T, existing string) {
	t.Helper()
	directory := privateCLIDirectory(t)
	path := filepath.Join(directory, existing)
	original := []byte("existing synthetic bytes")
	if err := platform.CreateProtectedExclusive(path, original); err != nil {
		t.Fatal(err)
	}
	var output, diagnostics bytes.Buffer
	code := (Runner{Out: &output, Err: &diagnostics}).Run(context.Background(), []string{"init", "--output", filepath.Join(directory, "cquay.json")})
	if code == 0 || output.Len() != 0 {
		t.Fatal("init reported success for an existing destination")
	}
	assertInitLeftOneOriginalOutput(t, directory, path, original)
}

func assertInitLeftOneOriginalOutput(t *testing.T, directory, path string, original []byte) {
	t.Helper()
	actual, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(actual, original) {
		t.Fatal("init modified existing output")
	}
	entries, err := os.ReadDir(directory)
	if err != nil || len(entries) != 1 {
		t.Fatal("rejected init created another output")
	}
}
