package cq

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/maldous/CircularQuay/internal/cli"
)

func TestLocalInputContractDefaultVersionNeedsNoPolicyOrState(t *testing.T) {
	options := localInputOptions(t)
	client, err := NewLocal(options)
	if err != nil {
		t.Fatal("explicit owned SDK paths were rejected")
	}
	version, err := client.Version(context.Background())
	if err != nil || version.Go != "1.27" || version.Maturity != "development" {
		t.Fatal("linked metadata-only version lost its declared shape")
	}
	if !strings.HasPrefix(strings.TrimPrefix(version.Version, "v"), "1.") || version.Commit == "" {
		t.Fatal("linked metadata lost its v1 build identity")
	}
	for _, path := range []string{options.ConfigPath, options.StateDirectory} {
		if _, err := os.Lstat(path); !errors.Is(err, os.ErrNotExist) {
			t.Fatal("metadata-only Version created an operational path")
		}
	}
}

func TestLocalInputContractInvalidWorkloadNeverDispatches(t *testing.T) {
	client := newLocalInputClient(t)
	client.run = localInputTripwire(t)
	result, err := client.RunWorkload(context.Background(), "")
	if result != (EffectResult{}) || category(err) != CategoryInvalid {
		t.Fatal("invalid workload returned a result or unexpected category")
	}
}

func TestLocalInputContractJSONFailuresReturnNoPartialResult(t *testing.T) {
	for _, scenario := range []struct {
		name, stdout, stderr string
		exit                 int
		category             Category
	}{
		{"stderr refusal", "", `{"error":"access_denied"}`, 1, CategoryDenied},
		{"unknown code", `{"error":"SYNTHETIC_PRIVATE_CODE"}`, "", 1, CategoryProtocol},
		{"malformed success", `{"accepted":`, "", 0, CategoryProtocol},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			observer := &localEventRecorder{}
			client := newLocalInputClient(t)
			client.observer = observer
			calls := 0
			client.run = func(_ context.Context, runner cli.Runner, _ []string) int {
				calls++
				localInputWrite(t, runner.Out, scenario.stdout)
				localInputWrite(t, runner.Err, scenario.stderr)
				return scenario.exit
			}
			result, err := client.ConfigValidate(context.Background())
			requireLocalInputRefusal(t, result, err, scenario.category)
			if calls != 1 || len(observer.events) != 1 {
				t.Fatal("JSON command failed to dispatch and observe exactly once")
			}
			event := observer.events[0]
			if event.Operation != "config_validate" || event.Outcome != string(scenario.category) || event.Duration < 0 {
				t.Fatal("JSON refusal leaked or changed its bounded observation")
			}
		})
	}
}

func TestLocalInputContractFractionalJSONRemainsOwnedNonTTYInput(t *testing.T) {
	for _, raw := range []string{`1.25`, `1e-2`} {
		t.Run(raw, func(t *testing.T) {
			options := localInputOptions(t)
			client, err := NewLocal(options)
			if err != nil {
				t.Fatal(err)
			}
			input := json.RawMessage(raw)
			before := bytes.Clone(input)
			calls := 0
			client.run = func(_ context.Context, runner cli.Runner, args []string) int {
				calls++
				assertLocalCommandArguments(t, options, args, []string{"config", "set", pointerFlag, "/synthetic", "--value-file", "-"})
				assertLocalJSONTransport(t, runner, raw)
				return 0
			}
			result, err := client.ConfigSet(context.Background(), "/synthetic", input, "")
			if err != nil || result.ExitCode != 0 || string(result.JSON) != `{"accepted":true}` || calls != 1 {
				t.Fatal("portable fractional JSON was changed or not dispatched exactly once")
			}
			if !bytes.Equal(input, before) {
				t.Fatal("SDK mutated caller-owned fractional input")
			}
		})
	}
}

func TestLocalInputContractInvalidObjectChildRejectsWholeDocument(t *testing.T) {
	client := newLocalInputClient(t)
	client.run = localInputTripwire(t)
	input := json.RawMessage(`{"valid":null,"nonportable":9007199254740992}`)
	before := bytes.Clone(input)
	result, err := client.ConfigReplace(context.Background(), input, "")
	requireLocalInputRefusal(t, result, err, CategoryInvalid)
	if !bytes.Equal(input, before) {
		t.Fatal("invalid object child changed caller JSON")
	}
}

func TestLocalInputContractEscapedPointerIsForwardedVerbatim(t *testing.T) {
	options := localInputOptions(t)
	client, err := NewLocal(options)
	if err != nil {
		t.Fatal(err)
	}
	calls := 0
	client.run = func(_ context.Context, runner cli.Runner, args []string) int {
		calls++
		assertLocalCommandArguments(t, options, args, []string{"config", "get", pointerFlag, "/a~1b/~0"})
		localInputWrite(t, runner.Out, `{"selected":"SYNTHETIC"}`)
		return 0
	}
	result, err := client.ConfigGet(context.Background(), "/a~1b/~0")
	if err != nil || result.ExitCode != 0 || string(result.JSON) != `{"selected":"SYNTHETIC"}` || calls != 1 {
		t.Fatal("valid escaped pointer was decoded, changed or dispatched more than once")
	}
}

func TestLocalInputContractShortDigestRefusesBeforeDispatch(t *testing.T) {
	client := newLocalInputClient(t)
	client.run = localInputTripwire(t)
	result, err := client.ConfigReplace(context.Background(), json.RawMessage(`{"synthetic":true}`), strings.Repeat("a", 63))
	requireLocalInputRefusal(t, result, err, CategoryInvalid)
}

func TestLocalInputContractMalformedOrHostlessSourceNeverDispatches(t *testing.T) {
	client := newLocalInputClient(t)
	client.run = localInputTripwire(t)
	for _, source := range []string{"https://fixture.invalid/%zz", "https:"} {
		t.Run(source, func(t *testing.T) {
			request := BootstrapRequest{SourceURL: source, Ref: strings.Repeat("a", 40), Path: "policy.json", SHA256: strings.Repeat("b", 64)}
			result, err := client.Bootstrap(context.Background(), request)
			requireLocalInputRefusal(t, result, err, CategoryInvalid)
			flow, err := client.Flow(context.Background(), FlowRequest{Workload: "synthetic-workload", BootstrapRequest: request})
			if flow != (EffectResult{}) || category(err) != CategoryInvalid {
				t.Fatal("invalid source returned a flow result or wrong category")
			}
		})
	}
}

func TestLocalInputContractVersionPreservesContextRefusal(t *testing.T) {
	client := newLocalInputClient(t)
	client.run = localInputTripwire(t)
	version, err := client.Version(nil)
	if version != (RuntimeVersion{}) || category(err) != CategoryInvalid {
		t.Fatal("Version returned metadata after invalid-context refusal")
	}
}

func localInputOptions(t *testing.T) LocalOptions {
	t.Helper()
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return LocalOptions{ConfigPath: filepath.Join(root, "missing", "cq.json"), StateDirectory: filepath.Join(root, "missing", "state")}
}

func newLocalInputClient(t *testing.T) *LocalClient {
	t.Helper()
	client, err := NewLocal(localInputOptions(t))
	if err != nil {
		t.Fatal("explicit SDK options were rejected")
	}
	return client
}

func localInputTripwire(t *testing.T) localRun {
	t.Helper()
	return func(context.Context, cli.Runner, []string) int {
		t.Fatal("refused public input reached the linked runner")
		return 1
	}
}

func localInputWrite(t *testing.T, writer io.Writer, value string) {
	t.Helper()
	if written, err := io.WriteString(writer, value); err != nil || written != len(value) {
		t.Fatal("synthetic response write failed")
	}
}

func requireLocalInputRefusal(t *testing.T, result Result, err error, want Category) {
	t.Helper()
	var safe *Error
	if !errors.As(err, &safe) || safe.Category != want || err.Error() != "cq: "+string(want) {
		t.Fatal("public refusal lost its exact safe SDK category")
	}
	if result.ExitCode != 0 || result.JSON != nil || errors.Unwrap(err) != nil {
		t.Fatal("public refusal returned partial JSON or private error cause")
	}
}
