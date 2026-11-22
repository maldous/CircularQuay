package cli

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/maldous/CircularQuay/internal/fault"
	cqmcp "github.com/maldous/CircularQuay/internal/mcp"
	"github.com/maldous/CircularQuay/internal/model"
	"github.com/maldous/CircularQuay/internal/platform"
	"github.com/maldous/CircularQuay/internal/state"
)

func cliCompletionFault(t *testing.T, err error, kind fault.Kind, operation string) {
	t.Helper()
	if !fault.Is(err, kind) || err.Error() != string(kind)+": "+operation || errors.Unwrap(err) != nil {
		t.Fatalf("unexpected bounded CLI refusal: %v", err)
	}
}

func TestCLICompletionStandardStreamsArePerInvocation(t *testing.T) {
	var output, diagnostics bytes.Buffer
	if code := (Runner{}).runWithStandardStreams(context.Background(), []string{"version"}, bytes.NewReader(nil), &output, &diagnostics); code != 0 || output.Len() == 0 || diagnostics.Len() != 0 {
		t.Fatal("owned default I/O was not selected")
	}
	var selected bytes.Buffer
	if code := (Runner{In: bytes.NewReader(nil), Out: &selected, Err: io.Discard}).runWithStandardStreams(context.Background(), []string{"version"}, nil, &output, &diagnostics); code != 0 || selected.Len() == 0 {
		t.Fatal("explicit streams were overwritten")
	}
}

func TestCLICompletionPathFailuresWithholdEveryOutput(t *testing.T) {
	calls := 0
	app := application{config: "relative.json", runner: Runner{Out: io.Discard, Err: io.Discard}, paths: &applicationPathOperations{
		absolute: func(string) (string, error) { calls++; return "", os.ErrNotExist },
	}}
	ctx := context.Background()
	policy, err := app.loadContext(ctx)
	cliCompletionFault(t, err, fault.Invalid, policyPathDiagnostic)
	cliCompletionEmptyPolicy(t, policy)
	policy, err = app.resolvePolicyBytes(ctx, nil, "relative.json")
	cliCompletionFault(t, err, fault.Invalid, policyPathDiagnostic)
	cliCompletionEmptyPolicy(t, policy)
	data, name, policy, err := app.readRootPolicy(ctx, "relative.json")
	cliCompletionFault(t, err, fault.Invalid, policyPathDiagnostic)
	if data != nil || name != "" || !reflect.DeepEqual(policy, model.Policy{}) {
		t.Fatal("root retained declaration")
	}
	document, digest, err := app.readConfigDocument()
	cliCompletionFault(t, err, fault.Invalid, errorConfigPath)
	if document != nil || digest != "" {
		t.Fatal("read retained document")
	}
	code, err := app.mutateConfig(ctx, "", false, false, func(*any) error { t.Fatal("path refusal reached mutation"); return nil })
	cliCompletionFault(t, err, fault.Invalid, errorConfigPath)
	cliCompletionZeroCode(t, code)
	changed, err := app.formatConfigContext(ctx)
	cliCompletionFault(t, err, fault.Invalid, errorConfigPath)
	if changed {
		t.Fatal("format reported a change")
	}
	code, err = app.initialize(nil)
	cliCompletionFault(t, err, fault.Invalid, "init path")
	cliCompletionZeroCode(t, code)
	code, err = app.bootstrapWithFetcher(ctx, nil, nil)
	cliCompletionFault(t, err, fault.Invalid, "bootstrap output")
	cliCompletionZeroCode(t, code)
	code, err = app.doctorReadiness(ctx, &doctorReport{}, false, nil)
	cliCompletionFault(t, err, fault.Invalid, policyPathDiagnostic)
	if code != 0 || calls != 9 {
		t.Fatal("path admission count/order changed")
	}
}

func TestCLICompletionFreeInputAndNamespaceResolverFailures(t *testing.T) {
	fail := func(string) (string, error) { return "", os.ErrNotExist }
	data, err := readConfigInputWithResolver(bytes.NewReader([]byte("owned")), "relative.json", fail)
	cliCompletionFault(t, err, fault.Invalid, "config input")
	if data != nil {
		t.Fatal("failed input retained bytes")
	}
	for _, failAt := range []int{1, 2} {
		cliCompletionNamespaceFailure(t, failAt)
	}
}

func TestCLICompletionStateDirectoryErrorAndExplicitOverride(t *testing.T) {
	app := application{paths: &applicationPathOperations{configurationDirectory: func() (string, error) { return "", os.ErrNotExist }}}
	store, err := app.store()
	cliCompletionFault(t, err, fault.Apply, stateDirectoryDiagnostic)
	if store != (state.Store{}) {
		t.Fatal("failed directory retained store")
	}
	app.directory = filepath.Join(privateCLIDirectory(t), "state")
	store, err = app.store()
	if err != nil || store.Directory != app.directory {
		t.Fatal("explicit directory incorrectly depended on ambient configuration")
	}
}

func TestCLICompletionInitializationFailurePreservesRealPartialEffects(t *testing.T) {
	for _, phase := range []string{"directory", "fixture", "policy"} {
		t.Run(phase, func(t *testing.T) {
			cliCompletionPathInitializationCompletionCase1(t, phase)
		})
	}
}

func TestCLICompletionBootstrapDestinationAndPrincipalRefusals(t *testing.T) {
	path := filepath.Join(privateCLIDirectory(t), "new.json")
	calls := 0
	err := prepareBootstrapDestinationWithOperations(path, func(string) error { calls++; return os.ErrPermission }, func(string) (os.FileInfo, error) { t.Fatal("directory refusal reached stat"); return nil, nil })
	if !errors.Is(err, os.ErrPermission) || calls != 1 {
		t.Fatal("directory error changed")
	}
	err = prepareBootstrapDestinationWithOperations(path, platform.EnsurePrivateDirectory, func(string) (os.FileInfo, error) { return nil, os.ErrPermission })
	cliCompletionFault(t, err, fault.Apply, "bootstrap output")
	if got := doctorPrincipalWithIdentity(func() (cqmcp.Principal, error) { return cqmcp.Principal{}, os.ErrPermission }); got != "unavailable" {
		t.Fatal("private identity failure leaked")
	}
	if got := doctorPrincipalWithIdentity(func() (cqmcp.Principal, error) { return cqmcp.Principal{Subject: "local:synthetic"}, nil }); got != "local:synthetic" {
		t.Fatal("local identity forwarding changed")
	}
}

func TestCLICompletionSnapshotDelegationRetainsCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := (snapshotBackend{}).Encrypt(ctx, model.Snapshot{}, map[string]string{"VALUE": "synthetic"})
	cliCompletionFault(t, err, fault.Denied, "provider_cancelled")
	values, err := (snapshotBackend{}).Decrypt(ctx, model.Snapshot{})
	cliCompletionFault(t, err, fault.Denied, "provider_cancelled")
	if values != nil {
		t.Fatal("canceled decrypt returned values")
	}
}

func cliCompletionPathInitializationCompletionCase1(t *testing.T, phase string) {
	t.Helper()
	dir := privateCLIDirectory(t)
	path := filepath.Join(dir, "cq.json")
	app := application{config: path, runner: Runner{Out: io.Discard, Err: io.Discard}}
	writes := 0
	ops := initializationOperations{mkdir: os.MkdirAll, write: func(name string, data []byte) error {
		writes++
		if phase == "fixture" || writes == 2 {
			return fault.New(fault.Apply, "owned initializer write")
		}
		return platform.CreateProtectedExclusive(name, data)
	}}
	if phase == "directory" {
		ops.mkdir = func(string, os.FileMode) error { return os.ErrPermission }
	}
	code, err := app.initializeWithOperations(nil, ops)
	operation := "owned initializer write"
	if phase == "directory" {
		operation = "init directory"
	}
	cliCompletionFault(t, err, fault.Apply, operation)
	if code != 0 {
		t.Fatal("initializer inner code changed")
	}
	if _, err := os.Lstat(path); !os.IsNotExist(err) {
		t.Fatal("failed init wrote policy")
	}
	cliCompletionInitializerOutput(t, phase, path, writes)

}

func cliCompletionEmptyPolicy(t *testing.T, policy model.Policy) {
	t.Helper()
	if !reflect.DeepEqual(policy, model.Policy{}) {
		t.Fatal("refusal retained policy")
	}
}
func cliCompletionZeroCode(t *testing.T, code int) {
	t.Helper()
	if code != 0 {
		t.Fatal("inner refusal code changed")
	}
}

func cliCompletionNamespaceFailure(t *testing.T, failAt int) {
	t.Helper()
	calls := 0
	resolve := func(name string) (string, error) {
		calls++
		if calls == failAt {
			return "", os.ErrNotExist
		}
		return filepath.Abs(name)
	}
	err := checkDefaultStateNamespaceWithResolver("relative-home", "relative-state", resolve)
	cliCompletionFault(t, err, fault.Apply, stateDirectoryDiagnostic)
	if calls != failAt {
		t.Fatal("namespace resolver continued after refusal")
	}
}

func cliCompletionInitializerOutput(t *testing.T, phase, path string, writes int) {
	t.Helper()
	fixture := path + ".fixture.json"
	if phase == "policy" {
		data, err := platform.ReadProtected(fixture, true)
		if err != nil || string(data) != "{\"DEMO_API_KEY\":\"synthetic-only-not-a-credential\"}\n" || writes != 2 {
			t.Fatal("real earlier fixture write was hidden")
		}
	} else if _, err := os.Lstat(fixture); !os.IsNotExist(err) {
		t.Fatal("failure wrote fixture")
	}
}
