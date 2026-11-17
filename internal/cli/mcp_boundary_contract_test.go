package cli

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/maldous/CircularQuay/internal/fault"
	cqmcp "github.com/maldous/CircularQuay/internal/mcp"
	"github.com/maldous/CircularQuay/internal/model"
)

func TestMCPBoundaryContractMalformedServeStopsBeforePolicy(t *testing.T) {
	for _, tc := range []struct {
		name, operation string
		args            []string
	}{
		{"missing command", "mcp command", nil},
		{"unknown command", "mcp command", []string{"SYNTHETIC_PRIVATE_COMMAND"}},
		{"unknown flag", "mcp arguments", []string{"serve", "--SYNTHETIC_PRIVATE_FLAG"}},
		{"missing flag value", "mcp arguments", []string{"serve", "--transport"}},
		{"extra argument", "mcp arguments", []string{"serve", "SYNTHETIC_PRIVATE_ARGUMENT"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			app, probe := mcpBoundaryUnusedApplication(t)
			code, err := app.serveMCP(context.Background(), tc.args)
			// This is the inner method's code/error pair, not Runner.Run's exit
			// status, a successful command or an authorization decision.
			if code != 0 {
				t.Fatal("malformed serve changed the inner return code")
			}
			requireMCPBoundaryFault(t, err, fault.Invalid, tc.operation)
			requireMCPBoundaryNoEffects(t, app, probe)
		})
	}
}

func TestMCPBoundaryContractCanceledBackendHasWholeZeroResults(t *testing.T) {
	for _, operation := range []string{"status", "read", "run"} {
		t.Run(operation, func(t *testing.T) {
			app, probe := mcpBoundaryUnusedApplication(t)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			cancel()
			b := backend{
				app: app,
				resolveGeneration: func(context.Context, model.Policy, string) (model.Generation, error) {
					probe.resolutions++
					return model.Generation{}, errors.New("synthetic resolver must not be reached")
				},
			}
			// Public Service cancellation has its own contract. These direct
			// backend calls prove only the inner pre-load cancellation fence.
			requireMCPBoundaryCanceledResult(t, ctx, b, operation)
			requireMCPBoundaryNoEffects(t, app, probe)
		})
	}
}

func requireMCPBoundaryCanceledResult(t *testing.T, ctx context.Context, b backend, operation string) {
	t.Helper()
	switch operation {
	case "status":
		result, err := b.Status(ctx, "SYNTHETIC_PRIVATE_WORKLOAD")
		requireMCPBoundaryFault(t, err, fault.Denied, "MCP request cancelled")
		if result != (cqmcp.Status{}) {
			t.Fatal("canceled status returned metadata")
		}
	case "read":
		result, err := b.Read(ctx, "SYNTHETIC_PRIVATE_WORKLOAD", "SYNTHETIC_PRIVATE_KEY")
		requireMCPBoundaryFault(t, err, fault.Denied, "MCP request cancelled")
		if !reflect.DeepEqual(result, model.Value{}) {
			t.Fatal("canceled read returned value metadata")
		}
	case "run":
		result, err := b.Run(ctx, "SYNTHETIC_PRIVATE_WORKLOAD")
		requireMCPBoundaryFault(t, err, fault.Denied, "MCP request cancelled")
		if result != (cqmcp.RunResult{}) {
			t.Fatal("canceled run returned a job result")
		}
	default:
		t.Fatal("unsupported synthetic operation")
	}
}

func TestMCPBoundaryContractNonFirstSelectionCopiesBindingAndMap(t *testing.T) {
	first := model.Binding{Name: "FIRST", Source: "first", AllowOffline: true}
	second := model.Binding{Name: "SECOND", Source: "second", AllowOffline: true}
	workload := model.Workload{Mode: "offline", Bindings: []model.Binding{first, second}}
	unrelated := model.Workload{Mode: "offline", Bindings: []model.Binding{first}}
	policy := model.Policy{
		Version: 1,
		Sources: map[string]model.Source{
			"first":  {Type: "json", Path: "synthetic-first.json"},
			"second": {Type: "json", Path: "synthetic-second.json"},
		},
		Workloads: map[string]model.Workload{"demo": workload, "unrelated": unrelated},
	}
	// This pure selection is not policy validation, a grant, a provider read
	// or snapshot restoration. Empty Snapshot keeps the narrow read branch.
	selectedPolicy, selected, err := selectedReadPolicy(policy, "demo", "SECOND")
	if err != nil {
		t.Fatal("valid non-first binding selection failed")
	}
	requireMCPBoundarySelection(t, policy, selectedPolicy, selected, second, unrelated)
	wantOriginal := model.Workload{Mode: "offline", Bindings: []model.Binding{first, second}}
	if !reflect.DeepEqual(policy.Workloads["demo"], wantOriginal) {
		t.Fatal("selection mutated the input binding list")
	}
	selected.Bindings[0].Name = "CHANGED_RETURNED_COPY"
	delete(selectedPolicy.Workloads, "unrelated")
	selectedPolicy.Workloads["new"] = model.Workload{}
	if !reflect.DeepEqual(policy.Workloads, map[string]model.Workload{"demo": wantOriginal, "unrelated": unrelated}) {
		t.Fatal("returned selection aliases the input workload map or bindings")
	}
}

func requireMCPBoundarySelection(t *testing.T, original, selectedPolicy model.Policy, selected model.Workload, binding model.Binding, unrelated model.Workload) {
	t.Helper()
	want := model.Workload{Mode: "offline", Bindings: []model.Binding{binding}}
	if !reflect.DeepEqual(selected, want) || !reflect.DeepEqual(selectedPolicy.Workloads["demo"], want) {
		t.Fatal("selection retained a sibling or changed the requested declaration")
	}
	if len(selectedPolicy.Workloads) != 2 || !reflect.DeepEqual(selectedPolicy.Workloads["unrelated"], unrelated) {
		t.Fatal("selection lost an unrelated workload")
	}
	if selectedPolicy.Version != original.Version || !reflect.DeepEqual(selectedPolicy.Sources, original.Sources) {
		t.Fatal("selection changed version or source declarations")
	}
}

type mcpBoundaryProbe struct {
	fetches, currents, resolutions int
}

func mcpBoundaryUnusedApplication(t *testing.T) (application, *mcpBoundaryProbe) {
	t.Helper()
	root := t.TempDir()
	probe := &mcpBoundaryProbe{}
	app := application{
		config:         filepath.Join(root, "absent-policy.json"),
		configExplicit: true,
		directory:      filepath.Join(root, "absent-state"),
		runner:         Runner{In: bytes.NewReader(nil), Out: &bytes.Buffer{}, Err: &bytes.Buffer{}},
		repository: repositoryOperations{
			fetch: func(context.Context, model.Repository) (string, error) {
				probe.fetches++
				return "", errors.New("synthetic fetch must not be reached")
			},
			current: func(context.Context, model.Repository) (string, error) {
				probe.currents++
				return "", errors.New("synthetic current must not be reached")
			},
		},
	}
	return app, probe
}

func requireMCPBoundaryFault(t *testing.T, err error, kind fault.Kind, operation string) {
	t.Helper()
	var typed *fault.Error
	if !errors.As(err, &typed) || typed.Kind != kind || typed.Operation != operation {
		t.Fatal("refusal lost its exact typed category or sanitized operation")
	}
	if err.Error() != string(kind)+": "+operation {
		t.Fatal("refusal included an unexpected diagnostic")
	}
}

func requireMCPBoundaryNoEffects(t *testing.T, app application, probe *mcpBoundaryProbe) {
	t.Helper()
	if *probe != (mcpBoundaryProbe{}) {
		t.Fatal("refused operation reached repository or resolution callbacks")
	}
	if app.runner.Out.(*bytes.Buffer).Len() != 0 || app.runner.Err.(*bytes.Buffer).Len() != 0 {
		t.Fatal("inner refusal emitted output or diagnostics")
	}
	for _, path := range []string{app.config, app.directory} {
		if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
			t.Fatal("refused operation created policy or state paths")
		}
	}
}
