package cli

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
)

func TestConfigReplaceRepairsMalformedPolicyWithRawDigest(t *testing.T) {
	const malformed = `{"SYNTHETIC_PRIVATE_INPUT":`
	for _, stale := range []bool{false, true} {
		name := "matching raw digest"
		if stale {
			name = "stale raw digest"
		}
		t.Run(name, func(t *testing.T) {
			config := filepath.Join(privateCLIDirectory(t), "policy.json")
			writeProtectedFixture(t, config, []byte(malformed))
			expected := fmt.Sprintf("%x", sha256.Sum256([]byte(malformed)))
			if stale {
				expected = strings.Repeat("0", 64)
			}
			var out, diagnostics bytes.Buffer
			runner := Runner{In: bytes.NewReader(configFixtureData(t)), Out: &out, Err: &diagnostics}
			code := runner.Run(context.Background(), []string{"--config", config, "config", "replace", "--file", "-", "--expect-sha256", expected})
			if stale {
				assertConfigMutationRefused(t, code, &out, &diagnostics, 3, "access_denied", "SYNTHETIC_PRIVATE_INPUT")
				assertConfigUnchanged(t, config, []byte(malformed))
				return
			}
			assertConfigRepairPublished(t, config, code, &out, &diagnostics)
		})
	}
}

func assertConfigRepairPublished(t *testing.T, config string, code int, out, diagnostics *bytes.Buffer) {
	t.Helper()
	if code != 0 || diagnostics.Len() != 0 {
		t.Fatalf("repair failed: exit %d, diagnostics: %s", code, diagnostics)
	}
	if outputState(t, out.Bytes()) != "updated" {
		t.Fatalf("repair did not publish an updated receipt: %s", out)
	}
	// Read and validate through the same public CLI after recovery, so a success
	// receipt alone cannot stand in for a usable replacement policy.
	runConfig(t, config, nil, "config", "validate")
	got := runConfig(t, config, nil, "config", "get", "--pointer", "/sources/fixture/path")
	if !bytes.Contains(got, []byte(`"value":"fixture.json"`)) {
		t.Fatalf("replacement policy was not persisted: %s", got)
	}
}
