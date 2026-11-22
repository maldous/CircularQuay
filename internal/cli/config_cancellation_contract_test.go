package cli

import (
	"bytes"
	"context"
	"io"
	"strings"
	"testing"
)

func TestConfigMutationCancellationPreservesPolicy(t *testing.T) {
	const privateInput = "SYNTHETIC_PRIVATE_INPUT.json"
	replacement := strings.ReplaceAll(string(configFixtureData(t)), "fixture.json", privateInput)
	cases := []struct {
		name        string
		args        []string
		input       string
		preCanceled bool
	}{
		{
			name:        "set with canceled caller",
			args:        []string{"config", "set", "--pointer", "/sources/fixture/path", "--value-json", `"` + privateInput + `"`},
			preCanceled: true,
		},
		{
			name:  "set interrupted while reading stdin",
			args:  []string{"config", "set", "--pointer", "/sources/fixture/path", "--value-file", "-"},
			input: `"` + privateInput + `"`,
		},
		{
			name:  "replace interrupted while reading stdin",
			args:  []string{"config", "replace", "--file", "-"},
			input: replacement,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			config := configFixture(t)
			original, digest := configSnapshot(t, config)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			var input io.Reader = strings.NewReader(tc.input)
			if tc.preCanceled {
				cancel()
			} else {
				input = cancelingConfigInput{reader: input, cancel: cancel}
			}
			var out, diagnostics bytes.Buffer
			runner := Runner{In: input, Out: &out, Err: &diagnostics}
			args := append([]string{"--config", config}, tc.args...)
			args = append(args, "--expect-sha256", digest)
			code := runner.Run(ctx, args)
			if ctx.Err() != context.Canceled {
				t.Fatal("caller cancellation was not delivered")
			}
			assertConfigMutationRefused(t, code, &out, &diagnostics, 1, "unavailable", privateInput)
			assertConfigSnapshotUnchanged(t, config, original, digest)
		})
	}
}

// Deliver valid input while interrupting its caller, rather than manufacturing
// a parse failure or depending on scheduler timing.
type cancelingConfigInput struct {
	reader io.Reader
	cancel context.CancelFunc
}

func (r cancelingConfigInput) Read(data []byte) (int, error) {
	n, err := r.reader.Read(data)
	if n > 0 {
		r.cancel()
	}
	return n, err
}

func assertConfigMutationRefused(t *testing.T, code int, out, diagnostics *bytes.Buffer, wantCode int, category, privateInput string) {
	t.Helper()
	if code != wantCode {
		t.Fatalf("exit code = %d, want %d; diagnostics: %s", code, wantCode, diagnostics)
	}
	if out.Len() != 0 {
		t.Fatalf("refused mutation emitted an output receipt: %s", out)
	}
	if diagnostics.String() != `{"error":"`+category+`"}`+"\n" {
		t.Fatalf("unexpected public error category: %s", diagnostics)
	}
	if strings.Contains(diagnostics.String(), privateInput) {
		t.Fatal("diagnostics exposed private input")
	}
}
