package repository

import (
	"context"
	"errors"
	"io"
	"reflect"
	"strings"
	"testing"

	"github.com/maldous/CircularQuay/internal/fault"
)

func TestGitRuntimeBoundaryContractEveryPrelaunchRefusalIsOrderedAndSanitized(t *testing.T) {
	for _, test := range []struct {
		stage string
		kind  fault.Kind
		steps []string
	}{
		{"find", fault.Unsupported, []string{"find"}},
		{"validate", fault.Trust, []string{"find", "validate"}},
		{"quota", fault.Unsupported, []string{"find", "validate", "quota"}},
		{"run", fault.Unavailable, []string{"find", "validate", "quota", "run"}},
	} {
		t.Run(test.stage, func(t *testing.T) {
			assertGitRuntimeRefusal(t, test.stage, test.kind, test.steps)
		})
	}
}

func assertGitRuntimeRefusal(t *testing.T, stage string, kind fault.Kind, wanted []string) {
	t.Helper()
	steps := []string{}
	operations := syntheticGitRuntime(t, stage, &steps, "owned")
	data, err := executeGitWithRuntime(context.Background(), []string{"SYNTHETIC_BOUNDARY=1"}, 8, operations, "version")
	if data != nil || !fault.Is(err, kind) || strings.Contains(err.Error(), "PRIVATE_DATA") {
		t.Fatalf("prelaunch refusal released data or upstream details: %q, %v", data, err)
	}
	if !reflect.DeepEqual(steps, wanted) {
		t.Fatalf("guard order = %v, want %v", steps, wanted)
	}
}

func TestGitRuntimeBoundaryContractOwnedWriterEnforcesExactBound(t *testing.T) {
	steps := []string{}
	operations := syntheticGitRuntime(t, "", &steps, "owned")
	data, err := executeGitWithRuntime(context.Background(), []string{"SYNTHETIC_BOUNDARY=1"}, 5, operations, "version")
	if err != nil || string(data) != "owned" {
		t.Fatalf("exact-bound synthetic output was refused: %q, %v", data, err)
	}
	steps = nil
	data, err = executeGitWithRuntime(context.Background(), []string{"SYNTHETIC_BOUNDARY=1"}, 4, operations, "version")
	if data != nil || !fault.Is(err, fault.Unavailable) {
		t.Fatalf("overflow published partial stdout: %q, %v", data, err)
	}
}

func syntheticGitRuntime(t *testing.T, fail string, steps *[]string, output string) gitRuntime {
	t.Helper()
	step := func(name string) error {
		*steps = append(*steps, name)
		if fail == name {
			return errors.New("PRIVATE_DATA")
		}
		return nil
	}
	return gitRuntime{
		find: func(name string) (string, error) {
			if name != "git" {
				t.Fatal("lookup selected an undeclared program")
			}
			return "synthetic-program-not-executed", step("find")
		},
		validate: func(program string) error {
			assertSyntheticGitProgram(t, program)
			return step("validate")
		},
		quota: func(program string) error {
			assertSyntheticGitProgram(t, program)
			return step("quota")
		},
		run: func(ctx context.Context, program string, environment, arguments []string, writer io.Writer) error {
			assertSyntheticGitProgram(t, program)
			assertSyntheticGitInvocation(t, ctx, environment, arguments)
			if err := step("run"); err != nil {
				return err
			}
			_, err := io.WriteString(writer, output)
			return err
		},
	}
}

func assertSyntheticGitProgram(t *testing.T, program string) {
	t.Helper()
	if program != "synthetic-program-not-executed" {
		t.Fatal("native program selection escaped the explicit fake")
	}
}

func assertSyntheticGitInvocation(t *testing.T, ctx context.Context, environment, arguments []string) {
	t.Helper()
	if ctx.Err() != nil || !reflect.DeepEqual(environment, []string{"SYNTHETIC_BOUNDARY=1"}) || !reflect.DeepEqual(arguments, []string{"version"}) {
		t.Fatal("fake runner received unexpected context, environment or argv")
	}
}
