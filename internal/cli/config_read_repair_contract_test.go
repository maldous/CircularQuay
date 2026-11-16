package cli

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestConfigReadFailuresPreserveOriginalBytesAndDoNotCreateMutationState(t *testing.T) {
	valid := configFixtureData(t)
	oversized := append(append([]byte(nil), valid...), bytes.Repeat([]byte(" "), maximumConfigInput+1-len(valid))...)
	cases := []struct {
		name     string
		data     []byte
		code     int
		category string
	}{
		{"missing", nil, 1, "missing_value"},
		{"oversized valid JSON", oversized, 2, "invalid_policy"},
		{"malformed JSON", []byte(`{"SYNTHETIC_CONFIG_PRIVATE":`), 2, "invalid_policy"},
		{"invalid declaration", []byte(`{"version":1,"sources":{},"workloads":{}}`), 2, "invalid_policy"},
	}
	for _, scenario := range cases {
		t.Run(scenario.name, func(t *testing.T) {
			assertConfigReadFailure(t, scenario.data, scenario.code, scenario.category)
		})
	}
}

func assertConfigReadFailure(t *testing.T, data []byte, expectedCode int, category string) {
	t.Helper()
	for _, command := range []string{"get", "list"} {
		t.Run(command, func(t *testing.T) {
			path := filepath.Join(privateCLIDirectory(t), "policy.json")
			if data != nil {
				writeProtectedFixture(t, path, data)
			}
			var output, diagnostics bytes.Buffer
			runner := Runner{Out: &output, Err: &diagnostics}
			code := runner.Run(context.Background(), []string{"--config", path, "config", command, "--pointer", ""})
			assertConfigMutationRefused(t, code, &output, &diagnostics, expectedCode, category, "SYNTHETIC_CONFIG_PRIVATE")
			assertRawConfigPreserved(t, path, data)
			if _, err := os.Stat(filepath.Join(filepath.Dir(path), ".cq-config-lock")); !os.IsNotExist(err) {
				t.Fatalf("refused read created mutation state: %v", err)
			}
		})
	}
}

func TestConfigSetCommitsSelectedProtectedValueFileAndReceipt(t *testing.T) {
	path := configFixture(t)
	valuePath := filepath.Join(filepath.Dir(path), "value.json")
	writeProtectedFixture(t, valuePath, []byte(`"from-protected-file.json"`))
	output := runConfig(t, path, nil, "config", "set", "--pointer", "/sources/fixture/path", "--value-file", valuePath)
	assertStoredConfigReceipt(t, path, output)
	var value struct {
		Value string `json:"value"`
	}
	readback := runConfig(t, path, nil, "config", "get", "--pointer", "/sources/fixture/path")
	if json.Unmarshal(readback, &value) != nil || value.Value != "from-protected-file.json" {
		t.Fatal("protected input was not committed as the selected declaration")
	}
	if _, err := os.Stat(filepath.Join(filepath.Dir(path), value.Value)); !os.IsNotExist(err) {
		t.Fatal("declaration editing created the provider file")
	}
}

func TestConfigReplaceRepairUsesRawDigestForCompareAndSwap(t *testing.T) {
	path := configFixture(t)
	replacement := configFixtureData(t)
	broken := []byte(`{"SYNTHETIC_CONFIG_PRIVATE":`)
	writeProtectedFixture(t, path, broken)
	var output, diagnostics bytes.Buffer
	runner := Runner{In: bytes.NewReader(replacement), Out: &output, Err: &diagnostics}
	code := runner.Run(context.Background(), []string{"--config", path, "config", "replace", "--file", "-", "--expect-sha256", rawConfigSHA(broken)})
	if code != 0 || diagnostics.Len() != 0 {
		t.Fatalf("matching raw digest did not permit explicit policy repair: code %d, %s", code, &diagnostics)
	}
	assertStoredConfigReceipt(t, path, output.Bytes())
	runConfig(t, path, nil, "config", "validate")
	if bytes.Contains(output.Bytes(), broken) {
		t.Fatal("repair receipt included the old declaration")
	}
}

func TestConfigReplaceRefusesStaleRawDigestWithoutRepairingBrokenPolicy(t *testing.T) {
	path := configFixture(t)
	replacement := configFixtureData(t)
	broken := []byte(`{"SYNTHETIC_CONFIG_PRIVATE":`)
	writeProtectedFixture(t, path, broken)
	var output, diagnostics bytes.Buffer
	runner := Runner{In: bytes.NewReader(replacement), Out: &output, Err: &diagnostics}
	code := runner.Run(context.Background(), []string{"--config", path, "config", "replace", "--file", "-", "--expect-sha256", rawConfigSHA(replacement)})
	assertConfigMutationRefused(t, code, &output, &diagnostics, 3, "access_denied", "SYNTHETIC_CONFIG_PRIVATE")
	assertRawConfigPreserved(t, path, broken)
}

func assertRawConfigPreserved(t *testing.T, path string, before []byte) {
	t.Helper()
	after, err := os.ReadFile(path)
	if before == nil {
		if !os.IsNotExist(err) {
			t.Fatalf("missing declaration was created or changed type: %v", err)
		}
		return
	}
	if err != nil || !bytes.Equal(after, before) || rawConfigSHA(after) != rawConfigSHA(before) {
		t.Fatal("refused command changed the original declaration")
	}
}

func assertStoredConfigReceipt(t *testing.T, path string, output []byte) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var receipt struct {
		SHA   string `json:"sha256"`
		State string `json:"state"`
	}
	if json.Unmarshal(output, &receipt) != nil || receipt.State != "updated" || receipt.SHA != rawConfigSHA(data) {
		t.Fatal("receipt does not identify the committed policy bytes")
	}
}

func rawConfigSHA(data []byte) string {
	digest := sha256.Sum256(data)
	return hex.EncodeToString(digest[:])
}
