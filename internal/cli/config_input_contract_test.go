package cli

import (
	"bytes"
	"context"
	"errors"
	"io"
	"path/filepath"
	"strings"
	"testing"
)

func TestConfigInputBoundaryRefusesInvalidCommandsWithoutReceiptsOrMutation(t *testing.T) {
	cases := []struct {
		name string
		args []string
	}{
		{"missing subcommand", []string{}},
		{"unknown subcommand", []string{"SYNTHETIC_CONFIG_INPUT"}},
		{"validate extra argument", []string{"validate", "SYNTHETIC_CONFIG_INPUT"}},
		{"get unknown flag", []string{"get", "--SYNTHETIC_CONFIG_INPUT"}},
		{"get missing pointer", []string{"get"}},
		{"get invalid pointer", []string{"get", "--pointer", "SYNTHETIC_CONFIG_INPUT"}},
		{"get scalar descendant", []string{"get", "--pointer", "/version/SYNTHETIC_CONFIG_INPUT"}},
		{"list unknown flag", []string{"list", "--SYNTHETIC_CONFIG_INPUT"}},
		{"list invalid pointer", []string{"list", "--pointer", "SYNTHETIC_CONFIG_INPUT"}},
		{"list scalar target", []string{"list", "--pointer", "/version"}},
		{"set unknown flag", []string{"set", "--SYNTHETIC_CONFIG_INPUT"}},
		{"set missing pointer", []string{"set", "--value-json", `"SYNTHETIC_CONFIG_INPUT"`}},
		{"set missing value", []string{"set", "--pointer", "/version"}},
		{"set non JSON value", []string{"set", "--pointer", "/version", "--value-json", "SYNTHETIC_CONFIG_INPUT"}},
		{"delete unknown flag", []string{"delete", "--SYNTHETIC_CONFIG_INPUT"}},
		{"delete scalar descendant", []string{"delete", "--pointer", "/version/SYNTHETIC_CONFIG_INPUT"}},
		{"replace missing file flag", []string{"replace"}},
	}
	for _, scenario := range cases {
		t.Run(scenario.name, func(t *testing.T) {
			assertConfigInputRefusal(t, scenario.args, nil, 2, "invalid_policy")
		})
	}
}

func TestConfigInputBoundaryDistinguishesMissingTargetsWithoutMutation(t *testing.T) {
	for _, args := range [][]string{
		{"get", "--pointer", "/sources/SYNTHETIC_CONFIG_INPUT"},
		{"get", "--pointer", "/workloads/demo/bindings/1"},
		{"list", "--pointer", "/sources/SYNTHETIC_CONFIG_INPUT"},
		{"delete", "--pointer", "/SYNTHETIC_CONFIG_INPUT/child"},
	} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			assertConfigInputRefusal(t, args, nil, 1, "missing_value")
		})
	}
}

func TestConfigInputBoundaryRejectsUnavailableValueFilesWithoutMutation(t *testing.T) {
	missing := filepath.Join(privateCLIDirectory(t), "SYNTHETIC_CONFIG_INPUT.json")
	for _, args := range [][]string{
		{"set", "--pointer", "/version", "--value-file", missing},
		{"replace", "--file", missing},
	} {
		t.Run(args[0], func(t *testing.T) {
			assertConfigInputRefusal(t, args, nil, 2, "invalid_policy")
		})
	}
}

func TestConfigInputBoundaryRejectsFailedStdinEvenWithValidJSON(t *testing.T) {
	for _, command := range []string{"set", "replace"} {
		t.Run(command, func(t *testing.T) {
			data := configFixtureData(t)
			args := []string{"replace", "--file", "-"}
			if command == "set" {
				data = []byte(`"SYNTHETIC_CONFIG_INPUT"`)
				args = []string{"set", "--pointer", "/sources/fixture/path", "--value-file", "-"}
			}
			input := &failingConfigInput{data: data}
			assertConfigInputRefusal(t, args, input, 2, "invalid_policy")
			if !input.delivered {
				t.Fatal("stdin failure was not exercised")
			}
		})
	}
}

// A read may return valid bytes and a non-EOF failure together. Neither a
// complete JSON value nor a sanitized public error authorizes committing it.
type failingConfigInput struct {
	data      []byte
	delivered bool
}

func (input *failingConfigInput) Read(destination []byte) (int, error) {
	if input.delivered {
		return 0, io.EOF
	}
	input.delivered = true
	return copy(destination, input.data), errors.New("SYNTHETIC_CONFIG_INPUT read failure")
}

func assertConfigInputRefusal(t *testing.T, args []string, input io.Reader, code int, category string) {
	t.Helper()
	config := configFixture(t)
	original, digest := configSnapshot(t, config)
	var output, diagnostics bytes.Buffer
	runner := Runner{In: input, Out: &output, Err: &diagnostics}
	status := runner.Run(context.Background(), append([]string{"--config", config, "config"}, args...))
	assertConfigMutationRefused(t, status, &output, &diagnostics, code, category, "SYNTHETIC_CONFIG_INPUT")
	assertConfigSnapshotUnchanged(t, config, original, digest)
}
