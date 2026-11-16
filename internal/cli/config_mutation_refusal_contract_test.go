package cli

import (
	"bytes"
	"context"
	"path/filepath"
	"testing"
)

func TestConfigOrdinaryMutationCannotImplicitlyCreateOrRepairAnUnreadablePolicy(t *testing.T) {
	valid := configFixtureData(t)
	oversized := append(append([]byte(nil), valid...), bytes.Repeat([]byte(" "), maximumConfigInput+1-len(valid))...)
	cases := []struct {
		name     string
		data     []byte
		code     int
		category string
	}{
		{"missing", nil, 1, "missing_value"},
		{"malformed", []byte(`{"SYNTHETIC_MUTATION_PRIVATE":`), 2, "invalid_policy"},
		{"oversized valid JSON", oversized, 2, "invalid_policy"},
	}
	for _, scenario := range cases {
		t.Run(scenario.name, func(t *testing.T) {
			for _, command := range []string{"set", "delete"} {
				t.Run(command, func(t *testing.T) {
					args := []string{command, "--pointer", "/sources/fixture/path"}
					if command == "set" {
						args = append(args, "--value-json", `"SYNTHETIC_MUTATION_PRIVATE.json"`)
					}
					assertUnreadableConfigMutationRefused(t, scenario.data, args, nil, scenario.code, scenario.category)
				})
			}
		})
	}
}

func TestConfigExplicitReplacementCannotRepairAnOversizedExistingPolicy(t *testing.T) {
	valid := configFixtureData(t)
	oversized := append(append([]byte(nil), valid...), bytes.Repeat([]byte(" "), maximumConfigInput+1-len(valid))...)
	assertUnreadableConfigMutationRefused(t, oversized, []string{"replace", "--file", "-"}, valid, 2, "invalid_policy")
}

func assertUnreadableConfigMutationRefused(t *testing.T, original []byte, args []string, input []byte, code int, category string) {
	t.Helper()
	path := filepath.Join(privateCLIDirectory(t), "policy.json")
	if original != nil {
		writeProtectedFixture(t, path, original)
	}
	var output, diagnostics bytes.Buffer
	runner := Runner{In: bytes.NewReader(input), Out: &output, Err: &diagnostics}
	status := runner.Run(context.Background(), append([]string{"--config", path, "config"}, args...))
	assertConfigMutationRefused(t, status, &output, &diagnostics, code, category, "SYNTHETIC_MUTATION_PRIVATE")
	assertRawConfigPreserved(t, path, original)
}

func TestConfigMutationPointerFailuresCannotCommitAReplacement(t *testing.T) {
	cases := []struct {
		name     string
		args     []string
		code     int
		category string
	}{
		{"set missing parent", []string{"set", "--pointer", "/sources/SYNTHETIC_MUTATION_PRIVATE/path", "--value-json", `"ignored.json"`}, 1, "missing_value"},
		{"delete malformed pointer", []string{"delete", "--pointer", "SYNTHETIC_MUTATION_PRIVATE"}, 2, "invalid_policy"},
		{"delete noncanonical array index", []string{"delete", "--pointer", "/workloads/demo/bindings/00"}, 2, "invalid_policy"},
		{"delete absent array index", []string{"delete", "--pointer", "/workloads/demo/bindings/1"}, 1, "missing_value"},
	}
	for _, scenario := range cases {
		t.Run(scenario.name, func(t *testing.T) {
			path := configFixture(t)
			original, digest := configSnapshot(t, path)
			var output, diagnostics bytes.Buffer
			runner := Runner{Out: &output, Err: &diagnostics}
			status := runner.Run(context.Background(), append([]string{"--config", path, "config"}, scenario.args...))
			assertConfigMutationRefused(t, status, &output, &diagnostics, scenario.code, scenario.category, "SYNTHETIC_MUTATION_PRIVATE")
			assertConfigSnapshotUnchanged(t, path, original, digest)
		})
	}
}
