package cq

import (
	"context"
	"encoding/json"
	"io"
	"strconv"
	"strings"
	"testing"

	"github.com/maldous/CircularQuay/internal/cli"
)

func TestLocalBoundaryContractCancellationAfterDispatchWithholdsResults(t *testing.T) {
	for _, operation := range []string{"json", "run", "flow"} {
		t.Run(operation, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			observer := &localEventRecorder{}
			client, err := NewLocal(localTestOptions(t), WithLocalObserver(observer))
			if err != nil {
				t.Fatal(err)
			}
			calls := 0
			client.run = func(_ context.Context, runner cli.Runner, _ []string) int {
				calls++
				_, _ = io.WriteString(runner.Out, `{"accepted":true}`)
				cancel()
				return 23
			}
			assertLocalBoundaryCanceledResult(t, client, ctx, operation)
			if calls != 1 || len(observer.events) != 1 {
				t.Fatal("canceled operation was retried or lost its observation")
			}
			if observer.events[0].Outcome != string(CategoryCancelled) {
				t.Fatal("late cancellation was reported as an ordinary process result")
			}
		})
	}
}

func assertLocalBoundaryCanceledResult(t *testing.T, client *LocalClient, ctx context.Context, operation string) {
	t.Helper()
	if operation == "json" {
		result, err := client.ConfigValidate(ctx)
		if category(err) != CategoryCancelled || result.ExitCode != 0 || result.JSON != nil {
			t.Fatal("canceled JSON operation returned data or a non-cancellation result")
		}
		return
	}
	var result EffectResult
	var err error
	if operation == "run" {
		result, err = client.RunWorkload(ctx, "synthetic-workload")
	} else {
		result, err = client.Flow(ctx, FlowRequest{
			Workload: "synthetic-workload",
			BootstrapRequest: BootstrapRequest{
				SourceURL: "https://github.com/example/synthetic-config",
				Ref:       strings.Repeat("a", 40),
				Path:      "policy.json",
				SHA256:    strings.Repeat("b", 64),
			},
		})
	}
	if category(err) != CategoryCancelled || result.ExitCode != 0 {
		t.Fatal("canceled effect operation returned a process result")
	}
}

func TestLocalBoundaryContractMissingRunnerIsSafeProcessFailure(t *testing.T) {
	for _, client := range []*LocalClient{nil, {}} {
		result, err := client.ConfigValidate(context.Background())
		if category(err) != CategoryProcess || result.ExitCode != 0 || result.JSON != nil {
			t.Fatal("missing runner returned data or a non-process failure")
		}
	}
}

func TestLocalBoundaryContractInvalidDeclarationsNeverDispatch(t *testing.T) {
	client, err := NewLocal(localTestOptions(t))
	if err != nil {
		t.Fatal(err)
	}
	calls := 0
	client.run = func(context.Context, cli.Runner, []string) int {
		calls++
		return 0
	}
	checks := map[string]func() (Result, error){
		"invalid list pointer": func() (Result, error) {
			return client.ConfigList(context.Background(), "/bad~escape")
		},
		"empty replacement": func() (Result, error) {
			return client.ConfigReplace(context.Background(), nil, "")
		},
		"invalid replacement digest": func() (Result, error) {
			return client.ConfigReplace(context.Background(), json.RawMessage(`{"synthetic":true}`), strings.Repeat("g", 64))
		},
	}
	for name, check := range checks {
		t.Run(name, func(t *testing.T) {
			result, callErr := check()
			if category(callErr) != CategoryInvalid || result.ExitCode != 0 || result.JSON != nil {
				t.Fatal("invalid declaration returned data or an unexpected category")
			}
		})
	}
	if calls != 0 {
		t.Fatal("invalid declaration reached the linked runner")
	}
}

func TestLocalBoundaryContractIJSONAggregateAndObjectBounds(t *testing.T) {
	for _, size := range []int{maxIJSONCollection, maxIJSONCollection + 1} {
		t.Run("object-"+strconv.Itoa(size), func(t *testing.T) {
			object := make(map[string]any, size)
			for index := range size {
				object["synthetic-"+strconv.Itoa(index)] = nil
			}
			assertLocalBoundaryJSONAdmission(t, object, size == maxIJSONCollection)
		})
	}
	// One root plus fifteen 1,024-node child arrays and the final child
	// reaches exactly 16,384 nodes at length 1,022, then exceeds it by one.
	// Individual collections and nesting remain within their independent limits.
	for _, finalLength := range []int{1022, 1023} {
		t.Run("aggregate-"+strconv.Itoa(finalLength), func(t *testing.T) {
			groups := make([]any, 16)
			for index := range groups {
				groups[index] = make([]any, 1023)
			}
			groups[len(groups)-1] = make([]any, finalLength)
			assertLocalBoundaryJSONAdmission(t, groups, finalLength == 1022)
		})
	}
}

func assertLocalBoundaryJSONAdmission(t *testing.T, document any, accepted bool) {
	t.Helper()
	encoded, err := json.Marshal(document)
	if err != nil {
		t.Fatal(err)
	}
	client, err := NewLocal(localTestOptions(t))
	if err != nil {
		t.Fatal(err)
	}
	calls := 0
	client.run = func(_ context.Context, runner cli.Runner, _ []string) int {
		calls++
		input, readErr := io.ReadAll(runner.In)
		if readErr != nil || string(input) != string(encoded) {
			t.Fatal("accepted JSON changed before the linked runner")
		}
		_, _ = io.WriteString(runner.Out, `{"accepted":true}`)
		return 0
	}
	result, callErr := client.ConfigReplace(context.Background(), encoded, "")
	assertLocalBoundaryJSONResult(t, result, callErr, calls, accepted)
}

func assertLocalBoundaryJSONResult(t *testing.T, result Result, callErr error, calls int, accepted bool) {
	t.Helper()
	if accepted {
		if callErr != nil || calls != 1 || string(result.JSON) != `{"accepted":true}` {
			t.Fatal("within-bound JSON was refused or changed")
		}
		return
	}
	if category(callErr) != CategoryInvalid || calls != 0 || result.JSON != nil {
		t.Fatal("over-bound JSON reached the runner or returned data")
	}
}

func TestLocalBoundaryContractOverflowCannotResumeWithValidSuffix(t *testing.T) {
	for _, route := range []string{"write-string", "copy"} {
		t.Run(route, func(t *testing.T) {
			assertLocalBoundaryOverflow(t, route)
		})
	}
}

func assertLocalBoundaryOverflow(t *testing.T, route string) {
	t.Helper()
	client, err := NewLocal(localTestOptions(t))
	if err != nil {
		t.Fatal(err)
	}
	client.run = func(_ context.Context, runner cli.Runner, _ []string) int {
		writeLocalBoundaryOverflow(t, runner.Out, route)
		return 0
	}
	result, callErr := client.ConfigValidate(context.Background())
	if category(callErr) != CategoryProtocol || result.JSON != nil {
		t.Fatal("overflow was cleared by a later write or returned partial data")
	}
}

func writeLocalBoundaryOverflow(t *testing.T, output io.Writer, route string) {
	t.Helper()
	_, _ = io.WriteString(output, `{"accepted":`)
	padding := strings.Repeat(" ", maxLocalJSONBytes)
	if route == "copy" {
		// LimitReader has no WriterTo fast path, so this also exercises a
		// destination's ReaderFrom when one is unintentionally exposed.
		_, _ = io.Copy(output, io.LimitReader(strings.NewReader(padding), int64(len(padding))))
	} else {
		_, _ = io.WriteString(output, padding)
	}
	buffer, ok := output.(*boundedBuffer)
	if !ok {
		t.Fatal("linked JSON output did not use the bounded writer")
	}
	retained := len(buffer.Bytes())
	_, _ = io.WriteString(output, `true}`)
	if !buffer.overflow || retained > maxLocalJSONBytes || len(buffer.Bytes()) != retained {
		t.Fatal("overflowed output buffer resumed accepting data")
	}
	if route == "write-string" && string(buffer.Bytes()) != `{"accepted":` {
		t.Fatal("single rejected write changed the retained prefix")
	}
}

func TestLocalBoundaryContractCopyWithinLimitReturnsOwnedJSON(t *testing.T) {
	client, err := NewLocal(localTestOptions(t))
	if err != nil {
		t.Fatal(err)
	}
	var captured *boundedBuffer
	client.run = func(_ context.Context, runner cli.Runner, _ []string) int {
		const response = `{"accepted":true}`
		_, _ = io.Copy(runner.Out, io.LimitReader(strings.NewReader(response), int64(len(response))))
		var ok bool
		captured, ok = runner.Out.(*boundedBuffer)
		if !ok {
			t.Fatal("linked JSON output did not use the bounded writer")
		}
		return 0
	}
	result, callErr := client.ConfigValidate(context.Background())
	if callErr != nil || string(result.JSON) != `{"accepted":true}` {
		t.Fatal("within-bound copying lost its success result")
	}
	captured.Bytes()[0] = 'X'
	if string(result.JSON) != `{"accepted":true}` {
		t.Fatal("returned JSON retained a mutable alias into the output buffer")
	}
}

func TestLocalBoundaryContractObserverPanicCannotChangeResult(t *testing.T) {
	client, err := NewLocal(localTestOptions(t), WithLocalObserver(panicObserver{}))
	if err != nil {
		t.Fatal(err)
	}
	calls := 0
	client.run = func(_ context.Context, runner cli.Runner, _ []string) int {
		calls++
		_, _ = io.WriteString(runner.Out, `{"accepted":true}`)
		return 0
	}
	result, callErr := client.ConfigValidate(context.Background())
	if callErr != nil || calls != 1 || string(result.JSON) != `{"accepted":true}` {
		t.Fatal("best-effort observer changed a completed local operation")
	}
}
