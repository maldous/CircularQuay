package cli

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/maldous/CircularQuay/internal/fault"
	cqmcp "github.com/maldous/CircularQuay/internal/mcp"
	"github.com/maldous/CircularQuay/internal/model"
)

// These cases stop before a transport, provider, credential read or workload
// execution. Policy values and paths are synthetic and private to each case.
func TestMCPRefusalContractServePreconditions(t *testing.T) {
	for _, entry := range []struct {
		name      string
		args      []string
		snapshots int
		invalid   bool
		operation string
	}{
		{name: "command", snapshots: 1, operation: "mcp command"},
		{name: "snapshot_count", args: []string{"serve"}, snapshots: 2, operation: "policy snapshot"},
		{name: "policy", args: []string{"serve"}, snapshots: 1, invalid: true, operation: "policy_structure"},
		{name: "transport", args: []string{"serve", "--transport", "unsupported"}, snapshots: 1, operation: "mcp transport"},
	} {
		t.Run(entry.name, func(t *testing.T) {
			app, policy := mcpRefusalFixture(t)
			if entry.invalid {
				policy.Version = 0
			}
			loaded := make([]model.Policy, entry.snapshots)
			for index := range loaded {
				loaded[index] = policy
			}
			code, err := app.serveMCP(context.Background(), entry.args, loaded...)
			assertMCPRefusalFault(t, err, fault.Invalid, entry.operation)
			if code != 0 {
				t.Fatalf("refused setup returned action exit code %d", code)
			}
			assertMCPRefusalStateAbsent(t, app.directory)
		})
	}
}

func TestMCPRefusalContractCanceledBackendNeverLoadsPolicy(t *testing.T) {
	for _, entry := range []struct {
		name string
		call func(backend, context.Context) (any, error)
		zero any
	}{
		{"status", func(b backend, ctx context.Context) (any, error) { return b.Status(ctx, "demo") }, cqmcp.Status{}},
		{"read", func(b backend, ctx context.Context) (any, error) { return b.Read(ctx, "demo", "TOKEN") }, model.Value{}},
		{"run", func(b backend, ctx context.Context) (any, error) { return b.Run(ctx, "demo") }, cqmcp.RunResult{}},
	} {
		t.Run(entry.name, func(t *testing.T) {
			app, _ := mcpRefusalFixture(t)
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			result, err := entry.call(backend{app: app}, ctx)
			assertMCPRefusalFault(t, err, fault.Denied, "MCP request cancelled")
			if !reflect.DeepEqual(result, entry.zero) {
				t.Fatal("canceled backend returned a nonzero result")
			}
			// The deliberately nonexistent policy would fail with Trust if the
			// canceled request reached loading instead of the cancellation guard.
			assertMCPRefusalStateAbsent(t, app.directory)
		})
	}
}

func TestMCPRefusalContractReadSelectionRejectsUnknownReferences(t *testing.T) {
	for _, entry := range []struct {
		name      string
		workload  string
		key       string
		kind      fault.Kind
		operation string
	}{
		{"workload", "missing", "TOKEN", fault.Invalid, "workload"},
		{"binding", "demo", "MISSING", fault.Denied, "MCP binding"},
		{"empty_binding", "demo", "", fault.Denied, "MCP binding"},
	} {
		t.Run(entry.name, func(t *testing.T) {
			_, policy := mcpRefusalFixture(t)
			before := mcpRefusalPolicyBytes(t, policy)
			selected, workload, err := selectedReadPolicy(policy, entry.workload, entry.key)
			assertMCPRefusalFault(t, err, entry.kind, entry.operation)
			if !reflect.DeepEqual(selected, model.Policy{}) || !reflect.DeepEqual(workload, model.Workload{}) {
				t.Fatal("refused selection exposed a partial declaration")
			}
			assertMCPRefusalPolicyUnchanged(t, policy, before)
		})
	}
}

func TestMCPRefusalContractNarrowReadViewDoesNotMutateOriginal(t *testing.T) {
	for _, mode := range []string{"online", "hybrid"} {
		t.Run(mode, func(t *testing.T) {
			_, policy := mcpRefusalFixture(t)
			original := policy.Workloads["demo"]
			original.Mode = mode
			original.Bindings = append(original.Bindings, model.Binding{Name: "OTHER", Source: "fixture", Field: "OTHER"})
			policy.Workloads["demo"] = original
			policy.Workloads["untouched"] = original
			before := mcpRefusalPolicyBytes(t, policy)
			selected, workload, err := selectedReadPolicy(policy, "demo", "TOKEN")
			if err != nil || len(workload.Bindings) != 1 || workload.Bindings[0].Name != "TOKEN" {
				t.Fatalf("narrow selection failed: %v", err)
			}
			if !reflect.DeepEqual(selected.Workloads["demo"], workload) {
				t.Fatal("returned policy and selected workload disagree")
			}
			workload.Bindings[0].Field = "CHANGED"
			delete(selected.Workloads, "untouched")
			assertMCPRefusalPolicyUnchanged(t, policy, before)
		})
	}
}

func TestMCPRefusalContractMissingActionSourceHasNoFingerprint(t *testing.T) {
	_, policy := mcpRefusalFixture(t)
	workload := policy.Workloads["demo"]
	workload.Bindings[0].Source = "missing"
	fingerprint, err := actionScopeFingerprint(policy, workload)
	assertMCPRefusalFault(t, err, fault.Invalid, "action source")
	if fingerprint != ([sha256.Size]byte{}) {
		t.Fatal("missing source produced a usable scope fingerprint")
	}
}

func mcpRefusalFixture(t *testing.T) (application, model.Policy) {
	t.Helper()
	root := t.TempDir()
	return application{config: filepath.Join(root, "never-created-policy.json"), directory: filepath.Join(root, "never-created-state")}, model.Policy{
		Version:   1,
		Sources:   map[string]model.Source{"fixture": {Type: "json", Path: filepath.Join(root, "never-created-values.json")}},
		Workloads: map[string]model.Workload{"demo": {
			Mode: "offline", Bindings: []model.Binding{{Name: "TOKEN", Source: "fixture", Field: "TOKEN", AllowOffline: true}},
		}},
	}
}

func assertMCPRefusalFault(t *testing.T, err error, kind fault.Kind, operation string) {
	t.Helper()
	var safe *fault.Error
	if !errors.As(err, &safe) || safe.Kind != kind || safe.Operation != operation {
		t.Fatalf("refusal = %v, want %s/%s", err, kind, operation)
	}
}

func assertMCPRefusalStateAbsent(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Lstat(path); !os.IsNotExist(err) {
		t.Fatalf("refused request touched state: %v", err)
	}
}

func mcpRefusalPolicyBytes(t *testing.T, policy model.Policy) []byte {
	t.Helper()
	value, err := json.Marshal(policy)
	if err != nil {
		t.Fatal(err)
	}
	return value
}

func assertMCPRefusalPolicyUnchanged(t *testing.T, policy model.Policy, before []byte) {
	t.Helper()
	if !bytes.Equal(mcpRefusalPolicyBytes(t, policy), before) {
		t.Fatal("selection changed the preceding policy declaration")
	}
}
