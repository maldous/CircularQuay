package cq

import (
	"context"
	"encoding/json"
	"io"
	"reflect"
	"strings"
	"testing"

	"github.com/maldous/CircularQuay/internal/cli"
)

// localEventRecorder captures only the safe metadata that the LocalClient
// observer contract exposes.
type localEventRecorder struct{ events []Event }

func (r *localEventRecorder) Observe(event Event) { r.events = append(r.events, event) }

func TestLocalJSONOperationsKeepConfigurationAndTransportExplicit(t *testing.T) {
	options := localTestOptions(t)
	digest := strings.Repeat("a", 64)
	pinned := BootstrapRequest{
		SourceURL: "https://github.com/maldous/CircularQuay",
		Ref:       "0123456789abcdef0123456789abcdef01234567",
		Path:      "examples/cq.json",
		SHA256:    digest,
	}
	observer := &localEventRecorder{}
	client, err := NewLocal(options, nil, WithLocalObserver(observer))
	if err != nil {
		t.Fatal(err)
	}

	type operation struct {
		name  string
		input string
		args  []string
		call  func() (Result, error)
	}
	operations := []operation{
		{
			name: "update_check",
			args: []string{"update", "check", "--json"},
			call: func() (Result, error) { return client.UpdateCheck(context.Background()) },
		},
		{
			name: "config_init",
			args: []string{"config", "init", "--output", options.ConfigPath},
			call: func() (Result, error) { return client.ConfigInit(context.Background()) },
		},
		{
			name: "config_get",
			args: []string{"config", "get", pointerFlag, "/workloads/alpha"},
			call: func() (Result, error) { return client.ConfigGet(context.Background(), "/workloads/alpha") },
		},
		{
			name: "config_list",
			args: []string{"config", "list", pointerFlag, ""},
			call: func() (Result, error) { return client.ConfigList(context.Background(), "") },
		},
		{
			name:  "config_set",
			input: `{"enabled":true}`,
			args:  []string{"config", "set", pointerFlag, "/workloads/alpha", "--value-file", "-", expectedSHA256Flag, digest},
			call: func() (Result, error) {
				return client.ConfigSet(context.Background(), "/workloads/alpha", json.RawMessage(`{"enabled":true}`), digest)
			},
		},
		{
			name: "config_delete",
			args: []string{"config", "delete", pointerFlag, "/workloads/alpha", expectedSHA256Flag, digest},
			call: func() (Result, error) { return client.ConfigDelete(context.Background(), "/workloads/alpha", digest) },
		},
		{
			name:  "config_replace",
			input: `{"workloads":[]}`,
			args:  []string{"config", "replace", "--file", "-", expectedSHA256Flag, digest},
			call: func() (Result, error) {
				return client.ConfigReplace(context.Background(), json.RawMessage(`{"workloads":[]}`), digest)
			},
		},
		{
			name: "doctor",
			args: []string{"doctor", "--fix"},
			call: func() (Result, error) { return client.Doctor(context.Background(), true) },
		},
		{
			name: "plan",
			args: []string{"plan"},
			call: func() (Result, error) { return client.Plan(context.Background()) },
		},
		{
			name: "status",
			args: []string{"status", workloadFlag, "alpha", "--json"},
			call: func() (Result, error) { return client.Status(context.Background(), "alpha") },
		},
		{
			name: "snapshot",
			args: []string{"snapshot", workloadFlag, "alpha"},
			call: func() (Result, error) { return client.Snapshot(context.Background(), "alpha") },
		},
		{
			name: "sync",
			args: []string{"sync", workloadFlag, "alpha", "--target", "staging"},
			call: func() (Result, error) { return client.Sync(context.Background(), "alpha", "staging") },
		},
		{
			name: "apply",
			args: []string{"apply", workloadFlag, "alpha", "--target", "production"},
			call: func() (Result, error) { return client.Apply(context.Background(), "alpha", "production") },
		},
		{
			name: "bootstrap",
			args: []string{"bootstrap", "--config-source", pinned.SourceURL, "--config-ref", pinned.Ref, "--config-path", pinned.Path, "--config-sha256", pinned.SHA256, "--output", options.ConfigPath, "--non-interactive"},
			call: func() (Result, error) { return client.Bootstrap(context.Background(), pinned) },
		},
	}
	position := 0
	client.run = func(_ context.Context, runner cli.Runner, args []string) int {
		if position >= len(operations) {
			t.Fatalf("unexpected invocation: %q", args)
		}
		want := operations[position]
		position++
		assertLocalInput(t, runner.In, want.input)
		assertLocalCommand(t, options, args, want.args)
		_, _ = io.WriteString(runner.Out, `{"accepted":true}`)
		return 0
	}
	for _, operation := range operations {
		result, callErr := operation.call()
		if callErr != nil || result.ExitCode != 0 || string(result.JSON) != `{"accepted":true}` {
			t.Fatalf("%s result=%+v error=%v", operation.name, result, callErr)
		}
	}
	if position != len(operations) {
		t.Fatalf("invocations=%d want=%d", position, len(operations))
	}
	if len(observer.events) != len(operations) {
		t.Fatalf("observations=%d want=%d", len(observer.events), len(operations))
	}
	for index, event := range observer.events {
		assertLocalObservation(t, event, operations[index].name)
	}
}

func TestLocalEffectOperationsDiscardChildOutputAndKeepPinnedInputs(t *testing.T) {
	options := localTestOptions(t)
	digest := strings.Repeat("b", 64)
	client, err := NewLocal(options)
	if err != nil {
		t.Fatal(err)
	}
	flow := FlowRequest{
		Workload: "alpha",
		BootstrapRequest: BootstrapRequest{
			SourceURL: "https://github.com/maldous/CircularQuay",
			Ref:       "0123456789abcdef0123456789abcdef01234567",
			Path:      "examples/cq.json",
			SHA256:    digest,
		},
	}
	want := [][]string{
		{"run", workloadFlag, "alpha"},
		{"flow", "run", workloadFlag, flow.Workload, "--config-source", flow.SourceURL, "--config-ref", flow.Ref, "--config-path", flow.Path, "--config-sha256", flow.SHA256, "--non-interactive"},
	}
	position := 0
	client.run = func(_ context.Context, runner cli.Runner, args []string) int {
		if runner.Out != io.Discard || runner.Err != io.Discard {
			t.Fatal("effect operation retained child output")
		}
		if position >= len(want) {
			t.Fatalf("unexpected effect invocation: %q", args)
		}
		assertLocalCommand(t, options, args, want[position])
		position++
		return 20 + position
	}
	result, callErr := client.RunWorkload(context.Background(), "alpha")
	assertLocalEffectResult(t, result, callErr, 21)
	result, callErr = client.Flow(context.Background(), flow)
	assertLocalEffectResult(t, result, callErr, 22)
	if position != len(want) {
		t.Fatalf("effect invocations=%d want=%d", position, len(want))
	}
}

func TestLocalClientRejectsInvalidOrCancelledOperationsBeforeDispatch(t *testing.T) {
	client, err := NewLocal(localTestOptions(t))
	if err != nil {
		t.Fatal(err)
	}
	called := false
	client.run = func(context.Context, cli.Runner, []string) int {
		called = true
		return 0
	}
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	digest := strings.Repeat("c", 64)
	pinned := BootstrapRequest{
		SourceURL: "https://github.com/maldous/CircularQuay",
		Ref:       "0123456789abcdef0123456789abcdef01234567",
		Path:      "examples/cq.json",
		SHA256:    digest,
	}
	checks := []struct {
		name string
		err  error
		want Category
	}{
		{"empty delete pointer", func() error { _, err := client.ConfigDelete(context.Background(), "", digest); return err }(), CategoryInvalid},
		{"empty workload", func() error { _, err := client.Status(context.Background(), ""); return err }(), CategoryInvalid},
		{"unsafe workload", func() error { _, err := client.Snapshot(context.Background(), ".."); return err }(), CategoryInvalid},
		{"unsafe target", func() error { _, err := client.Sync(context.Background(), "alpha", "unsafe/path"); return err }(), CategoryInvalid},
		{"missing apply target", func() error { _, err := client.Apply(context.Background(), "alpha", ""); return err }(), CategoryInvalid},
		{"cancelled run", func() error { _, err := client.RunWorkload(cancelled, "alpha"); return err }(), CategoryCancelled},
		{"cancelled flow", func() error {
			_, err := client.Flow(cancelled, FlowRequest{Workload: "alpha", BootstrapRequest: pinned})
			return err
		}(), CategoryCancelled},
	}
	for _, check := range checks {
		if actual := category(check.err); actual != check.want {
			t.Fatalf("%s category=%q error=%v", check.name, actual, check.err)
		}
	}
	if called {
		t.Fatal("invalid or cancelled operation was dispatched")
	}
}

// Nil is an invalid input under test, rather than an application context.
func TestLocalClientRejectsNilContextBeforeDispatch(t *testing.T) {
	client, err := NewLocal(localTestOptions(t))
	if err != nil {
		t.Fatal(err)
	}
	client.run = func(context.Context, cli.Runner, []string) int {
		t.Fatal("nil context was dispatched")
		return 0
	}
	for _, ctx := range []context.Context{nil} {
		if _, err := client.ConfigValidate(ctx); category(err) != CategoryInvalid {
			t.Fatalf("nil context category=%q error=%v", category(err), err)
		}
	}
}

func assertLocalEffectResult(t *testing.T, result EffectResult, err error, exitCode int) {
	t.Helper()
	if err != nil || result.ExitCode != exitCode {
		t.Fatalf("effect result=%+v want exit code=%d error=%v", result, exitCode, err)
	}
}

func assertLocalInput(t *testing.T, input io.Reader, want string) {
	t.Helper()
	actual, err := io.ReadAll(input)
	if err != nil || string(actual) != want {
		t.Fatalf("input=%q want=%q error=%v", actual, want, err)
	}
}

func assertLocalCommand(t *testing.T, options LocalOptions, args, want []string) {
	t.Helper()
	if len(args) < 4 {
		t.Fatalf("incomplete configuration args=%q", args)
	}
	if !reflect.DeepEqual(args[:4], []string{"--config", options.ConfigPath, "--state-dir", options.StateDirectory}) {
		t.Fatalf("configuration args=%q", args)
	}
	if !reflect.DeepEqual(args[4:], want) {
		t.Fatalf("args=%q want command=%q", args, want)
	}
}

func assertLocalObservation(t *testing.T, event Event, operation string) {
	t.Helper()
	if event.Operation != operation || event.Outcome != "success" || event.Duration < 0 {
		t.Fatalf("operation=%s event=%+v", operation, event)
	}
}

func TestLocalJSONOperationsBoundUntrustedCLIDiagnostics(t *testing.T) {
	client, err := NewLocal(localTestOptions(t))
	if err != nil {
		t.Fatal(err)
	}
	client.run = func(_ context.Context, runner cli.Runner, _ []string) int {
		_, _ = io.WriteString(runner.Err, strings.Repeat("x", 513))
		return 1
	}
	if _, callErr := client.ConfigValidate(context.Background()); category(callErr) != CategoryProtocol {
		t.Fatalf("overflow diagnostic category=%q error=%v", category(callErr), callErr)
	}
	client.run = func(_ context.Context, runner cli.Runner, _ []string) int {
		_, _ = io.WriteString(runner.Out, strings.Repeat("x", maxLocalJSONBytes+1))
		return 0
	}
	if _, callErr := client.ConfigValidate(context.Background()); category(callErr) != CategoryProtocol {
		t.Fatalf("overflow response category=%q error=%v", category(callErr), callErr)
	}
}
