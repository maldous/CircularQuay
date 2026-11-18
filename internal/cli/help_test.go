package cli

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestRunnerHelpDoesNotLoadConfiguration(t *testing.T) {
	t.Parallel()

	for _, arguments := range [][]string{{"help"}, {"-h"}, {"--help"}} {
		arguments := arguments
		t.Run(strings.Join(arguments, " "), func(t *testing.T) {
			t.Parallel()
			var out, diagnostics bytes.Buffer

			code := (Runner{Out: &out, Err: &diagnostics}).Run(context.Background(), arguments)

			if code != 0 {
				t.Fatalf("help returned status %d", code)
			}
			if !strings.HasPrefix(out.String(), "cq [--config PATH]") {
				t.Fatalf("help did not print usage: %q", out.String())
			}
			if diagnostics.Len() != 0 {
				t.Fatalf("help wrote diagnostics: %q", diagnostics.String())
			}
		})
	}
}

func TestRunnerRejectsInvalidGlobalFlag(t *testing.T) {
	t.Parallel()

	var out, diagnostics bytes.Buffer
	code := (Runner{Out: &out, Err: &diagnostics}).Run(context.Background(), []string{"--unknown"})

	if code != 2 {
		t.Fatalf("invalid flag returned status %d", code)
	}
	if out.Len() != 0 {
		t.Fatalf("invalid flag wrote normal output: %q", out.String())
	}
	if diagnostics.String() != "{\"error\":\"invalid_policy\"}\n" {
		t.Fatalf("invalid flag diagnostics = %q", diagnostics.String())
	}
}
