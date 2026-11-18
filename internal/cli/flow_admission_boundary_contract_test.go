package cli

import (
	"bytes"
	"context"
	"errors"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/maldous/CircularQuay/internal/bootstrap"
	"github.com/maldous/CircularQuay/internal/fault"
)

func requireCLISafeFault(t *testing.T, err error, kind fault.Kind, operation string) {
	t.Helper()
	got, ok := err.(*fault.Error)
	if !ok || got.Kind != kind || got.Operation != operation || errors.Unwrap(err) != nil {
		t.Fatalf("fault = %#v; want %s/%s", err, kind, operation)
	}
}

func TestCLISafeFlowCommandAndCIAdmission(t *testing.T) {
	clearFlowEnvironment(t)
	a := application{}
	for _, args := range [][]string{nil, {"unknown"}} {
		got, err := a.flowOptions(args)
		requireCLISafeFault(t, err, fault.Invalid, "flow command")
		if !reflect.DeepEqual(got, flowOptions{}) {
			t.Fatalf("partial options: %#v", got)
		}
	}
	calls := 0
	fetch := func(context.Context, bootstrap.Request) (bootstrap.Result, error) {
		calls++
		return bootstrap.Result{}, errors.New("unexpected retrieval")
	}
	code, err := a.flowWithFetcher(context.Background(), []string{"run", "--workload", "demo"}, fetch)
	requireCLISafeFault(t, err, fault.Invalid, "ci context")
	cliSafeNoFetch(t, code, calls)
	t.Setenv("GITHUB_ACTIONS", "true")
	t.Setenv(envCQConfigSource, "")
	code, err = a.flowWithFetcher(context.Background(), []string{"run", "--workload", "demo"}, fetch)
	requireCLISafeFault(t, err, fault.Invalid, "flow configuration pin")
	cliSafeNoFetch(t, code, calls)
}

func TestCLISafeFlowFetchStopsBeforeMaterialization(t *testing.T) {
	source := bootstrapOptions{sourceURL: "https://github.com/example/fixture", ref: strings.Repeat("a", 40), digest: strings.Repeat("b", 64), path: "cq.json"}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	calls := 0
	failure := fault.New(fault.Trust, "synthetic fetch")
	fetch := func(context.Context, bootstrap.Request) (bootstrap.Result, error) {
		calls++
		return bootstrap.Result{Bytes: []byte("must not escape")}, failure
	}
	got, err := fetchFlowPolicy(ctx, source, fetch)
	if !errors.Is(err, context.Canceled) || !reflect.DeepEqual(got, bootstrap.Result{}) || calls != 0 {
		t.Fatalf("cancelled fetch = %#v/%v; calls %d", got, err, calls)
	}
	got, err = fetchFlowPolicy(context.Background(), source, fetch)
	if err != failure || !reflect.DeepEqual(got, bootstrap.Result{}) || calls != 1 {
		t.Fatalf("failed fetch = %#v/%v; calls %d", got, err, calls)
	}
}

type cliSafeReceiptWriter struct {
	cancel context.CancelFunc
	fail   bool
	calls  int
}

func (w *cliSafeReceiptWriter) Write(data []byte) (int, error) {
	w.calls++
	if w.fail {
		return 0, errors.New("synthetic private writer failure")
	}
	w.cancel()
	return len(data), nil
}

func TestCLISafeFlowReceiptStopsDispatch(t *testing.T) {
	config := configFixture(t)
	var out bytes.Buffer
	a := application{config: config, directory: filepath.Join(filepath.Dir(config), "unused-state"), runner: Runner{Out: &out}}
	err := a.flowReceipt(ciGitHub, "local", "demo", "", "")
	requireCLISafeFault(t, err, fault.Apply, "flow receipt")
	writer := &cliSafeReceiptWriter{fail: true}
	a.runner.Err = writer
	code, err := a.flowLocal(context.Background(), "demo", ciGitHub, "local", "", "")
	requireCLISafeFault(t, err, fault.Apply, "flow receipt")
	if code != 0 || writer.calls != 1 || out.Len() != 0 {
		t.Fatalf("failed receipt dispatched: %d/%d/%q", code, writer.calls, out.String())
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	writer = &cliSafeReceiptWriter{cancel: cancel}
	a.runner.Err = writer
	code, err = a.flowLocal(ctx, "demo", ciGitHub, "local", "", "")
	if !errors.Is(err, context.Canceled) || code != 0 || writer.calls != 1 || out.Len() != 0 {
		t.Fatalf("cancelled receipt dispatched: %d/%v/%d", code, err, writer.calls)
	}
	requireMissingPath(t, a.directory)
	requireMissingPath(t, filepath.Join(filepath.Dir(config), "fixture.json"))
}

func TestCLISafeBootstrapRefusesBeforeFetch(t *testing.T) {
	clearFlowEnvironment(t)
	a := application{config: filepath.Join(privateCLIDirectory(t), "absent.json")}
	calls := 0
	fetch := func(context.Context, bootstrap.Request) (bootstrap.Result, error) {
		calls++
		return bootstrap.Result{}, errors.New("unexpected fetch")
	}
	code, err := a.bootstrapWithFetcher(context.Background(), []string{"--unknown"}, fetch)
	requireCLISafeFault(t, err, fault.Invalid, "bootstrap arguments")
	if code != 0 || calls != 0 {
		t.Fatal("malformed bootstrap reached fetch")
	}
	code, err = a.bootstrapWithFetcher(context.Background(), nil, fetch)
	requireCLISafeFault(t, err, fault.Trust, "policy file")
	if code != 0 || calls != 0 {
		t.Fatal("absent local policy reached fetch")
	}
	requireMissingPath(t, a.config)
}

func cliSafeNoFetch(t *testing.T, code, calls int) {
	t.Helper()
	if code != 0 || calls != 0 {
		t.Fatalf("inner refusal = %d; fetches = %d", code, calls)
	}
}
