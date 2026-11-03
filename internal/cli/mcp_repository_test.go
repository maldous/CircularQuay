package cli

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/maldous/CircularQuay/internal/fault"
	cqmcp "github.com/maldous/CircularQuay/internal/mcp"
	"github.com/maldous/CircularQuay/internal/model"
	"github.com/maldous/CircularQuay/internal/platform"
	"github.com/maldous/CircularQuay/internal/resolve"
	"github.com/maldous/CircularQuay/internal/source"
)

type mcpRepositoryAudit struct{}

func (mcpRepositoryAudit) Record(context.Context, cqmcp.Event) error { return nil }

func TestMCPRepositoryReadUsesSelectedBindingMode(t *testing.T) {
	for _, test := range []struct {
		name       string
		mode       string
		fetchError error
		wantFetch  int
	}{
		{name: "hybrid", mode: "hybrid", wantFetch: 1},
		{name: "offline", mode: "offline", wantFetch: 0},
	} {
		t.Run(test.name, func(t *testing.T) {
			app, policy, release := mcpRepositoryApplication(t, test.mode)
			fetches, currents := 0, 0
			app.repository = repositoryOperations{
				fetch: func(context.Context, model.Repository) (string, error) {
					fetches++
					return release, test.fetchError
				},
				current: func(context.Context, model.Repository) (string, error) {
					currents++
					return release, nil
				},
			}
			service := mcpRepositoryService(t, app, policy, backend{app: app})
			result, err := service.ReadSecret(context.Background(), cqmcp.Principal{Subject: "local:alice"}, "demo", "TOKEN")
			if err != nil || result.Value != "synthetic" {
				t.Fatalf("read result=%+v err=%v", result, err)
			}
			if fetches != test.wantFetch || currents != 1 {
				t.Fatalf("repository operations fetch=%d current=%d", fetches, currents)
			}
		})
	}
}

func TestMCPRepositoryOnlineRunFetchesBeforeResolution(t *testing.T) {
	app, policy, release := mcpRepositoryApplication(t, "online")
	workload := policy.Workloads["demo"]
	workload.Bindings[0].AllowOffline = false
	policy.Workloads["demo"] = workload
	policy.Sources["fixture"] = model.Source{
		Type: "sops", Path: "repo:values.sops.json", SOPS: filepath.Join(filepath.Dir(app.config), "sops"),
		Identity: "token_file", TokenFile: filepath.Join(filepath.Dir(app.config), "token"),
		VaultAllowlist: []string{"https://vault.example.test"}, VaultKeys: []string{"key"},
	}
	writeMCPTestPolicy(t, app.config, policy)
	fetches, resolutions := 0, 0
	app.repository = repositoryOperations{
		fetch:   func(context.Context, model.Repository) (string, error) { fetches++; return release, nil },
		current: func(context.Context, model.Repository) (string, error) { return release, nil },
	}
	service := mcpRepositoryService(t, app, policy, backend{app: app,
		resolveGeneration: func(context.Context, model.Policy, string) (model.Generation, error) {
			resolutions++
			return model.Generation{Workload: "demo", Values: map[string]model.Value{"TOKEN": {Data: "synthetic"}}}, nil
		},
	})
	if _, err := service.RunWorkload(context.Background(), cqmcp.Principal{Subject: "local:alice"}, "demo"); err != nil {
		t.Fatal(err)
	}
	if fetches != 1 || resolutions != 1 {
		t.Fatalf("online repository fetches=%d resolutions=%d", fetches, resolutions)
	}
}

func TestMCPRepositoryUnauthorizedReadNeverFetches(t *testing.T) {
	app, policy, _ := mcpRepositoryApplication(t, "hybrid")
	fetches := 0
	app.repository = repositoryOperations{
		fetch:   func(context.Context, model.Repository) (string, error) { fetches++; return "", nil },
		current: func(context.Context, model.Repository) (string, error) { return "", nil },
	}
	service := mcpRepositoryService(t, app, policy, backend{app: app})
	_, err := service.ReadSecret(context.Background(), cqmcp.Principal{Subject: "local:unapproved"}, "demo", "TOKEN")
	if !fault.Is(err, fault.Denied) || fetches != 0 {
		t.Fatalf("unauthorized repository read error=%v fetches=%d", err, fetches)
	}
}

func TestMCPRepositoryRevocationAfterFetchStopsActionBeforeResolution(t *testing.T) {
	app, policy, release := mcpRepositoryApplication(t, "hybrid")
	fetches, resolutions := 0, 0
	app.repository = repositoryOperations{
		fetch: func(context.Context, model.Repository) (string, error) {
			fetches++
			policy.Grants = nil
			writeMCPTestPolicy(t, app.config, policy)
			return release, nil
		},
		current: func(context.Context, model.Repository) (string, error) { return release, nil },
	}
	service := mcpRepositoryService(t, app, policy, backend{
		app: app,
		resolveGeneration: func(context.Context, model.Policy, string) (model.Generation, error) {
			resolutions++
			return model.Generation{}, nil
		},
	})
	_, err := service.RunWorkload(context.Background(), cqmcp.Principal{Subject: "local:alice"}, "demo")
	if !fault.Is(err, fault.Denied) || fetches != 1 || resolutions != 0 {
		t.Fatalf("revoked repository action error=%v fetches=%d resolutions=%d", err, fetches, resolutions)
	}
}

type mcpSnapshotMemory struct{ values map[string]string }

func (m *mcpSnapshotMemory) Encrypt(_ context.Context, _ model.Snapshot, values map[string]string) error {
	m.values = values
	return nil
}

func (m *mcpSnapshotMemory) Decrypt(context.Context, model.Snapshot) (map[string]string, error) {
	return m.values, nil
}

func TestMCPReadOfflineSnapshotRestoresCompleteBindingSet(t *testing.T) {
	workload := model.Workload{
		Mode: "offline",
		Bindings: []model.Binding{
			{Name: "TOKEN", Source: "first", AllowOffline: true},
			{Name: "OTHER", Source: "second", AllowOffline: true},
		},
		Snapshot: model.Snapshot{Path: filepath.Join(t.TempDir(), "snapshot")},
	}
	policy := model.Policy{
		Version:   1,
		Sources:   map[string]model.Source{"first": {Type: "fixture"}, "second": {Type: "fixture"}},
		Workloads: map[string]model.Workload{"demo": workload},
	}
	selectedPolicy, selected, err := selectedReadPolicy(policy, "demo", "TOKEN")
	if err != nil || len(selected.Bindings) != 2 {
		t.Fatalf("snapshot read selection bindings=%d err=%v", len(selected.Bindings), err)
	}
	completeScope, err := actionScopeFingerprint(policy, workload)
	if err != nil {
		t.Fatal(err)
	}
	selectedScope, err := actionScopeFingerprint(selectedPolicy, selected)
	if err != nil || completeScope != selectedScope {
		t.Fatalf("complete snapshot scope err=%v equal=%t", err, completeScope == selectedScope)
	}
	snapshots := &mcpSnapshotMemory{}
	reads := 0
	engine := &resolve.Engine{
		Policy: selectedPolicy,
		Reader: source.ReadFunc(func(context.Context, model.Source, model.Binding) (model.Value, error) {
			reads++
			return model.Value{}, fault.New(fault.Apply, "provider read")
		}),
		Snapshots: snapshots,
		Now:       func() time.Time { return time.Date(2026, 11, 4, 0, 0, 0, 0, time.UTC) },
	}
	generation := model.Generation{Workload: "demo", RetrievedAt: engine.Now(), Values: map[string]model.Value{
		"TOKEN": {Data: "synthetic-token", Version: "v1"},
		"OTHER": {Data: "synthetic-other", Version: "v1"},
	}, Receipts: []model.Receipt{{Source: "first", Version: "v1"}, {Source: "second", Version: "v1"}}}
	if err = engine.Snapshot(context.Background(), generation); err != nil {
		t.Fatal(err)
	}
	value, err := engine.ReadKey(context.Background(), "demo", "TOKEN")
	if err != nil || value.Data != "synthetic-token" || reads != 0 {
		t.Fatalf("offline snapshot read value=%q err=%v provider_reads=%d", value.Data, err, reads)
	}
}

func TestActionScopeFingerprintIncludesSelectedRepositoryDeclaration(t *testing.T) {
	_, policy, _ := mcpRepositoryApplication(t, "offline")
	workload := policy.Workloads["demo"]
	before, err := actionScopeFingerprint(policy, workload)
	if err != nil {
		t.Fatal(err)
	}
	policy.Repository.AllowedSignersFile = "/different/trust"
	after, err := actionScopeFingerprint(policy, workload)
	if err != nil || before == after {
		t.Fatalf("repository trust change fingerprint err=%v equal=%t", err, before == after)
	}
	policy.Sources["fixture"] = model.Source{Type: "json", Path: filepath.Join(t.TempDir(), "values.json")}
	withoutRepository, err := actionScopeFingerprint(policy, workload)
	if err != nil {
		t.Fatal(err)
	}
	policy.Repository.Ref = "refs/heads/other"
	withoutRepositoryChanged, err := actionScopeFingerprint(policy, workload)
	if err != nil || withoutRepository != withoutRepositoryChanged {
		t.Fatalf("unselected repository fingerprint err=%v equal=%t", err, withoutRepository == withoutRepositoryChanged)
	}
}

func mcpRepositoryApplication(t *testing.T, mode string) (application, model.Policy, string) {
	t.Helper()
	app, _, policy := mcpRunTestApplication(t, filepath.Join(t.TempDir(), "started"))
	release := filepath.Join(filepath.Dir(app.config), "release")
	if err := platform.EnsurePrivateDirectory(release); err != nil {
		t.Fatal(err)
	}
	writeProtectedFixture(t, filepath.Join(release, "values.json"), []byte(`{"TOKEN":"synthetic","OTHER":"other"}`))
	workload := policy.Workloads["demo"]
	workload.Mode = mode
	workload.Bindings[0].Source = "fixture"
	workload.Bindings[0].AllowOffline = true
	policy.Workloads["demo"] = workload
	policy.Sources["fixture"] = model.Source{Type: "json", Path: "repo:values.json"}
	policy.Repository = model.Repository{
		URL:                "https://example.test/synthetic.git",
		Directory:          filepath.Join(filepath.Dir(app.config), "repository"),
		Ref:                "refs/heads/data",
		AllowedSignersFile: filepath.Join(filepath.Dir(app.config), "signers"),
		InitialCommit:      "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
	}
	writeMCPTestPolicy(t, app.config, policy)
	return app, policy, release
}

func mcpRepositoryService(t *testing.T, app application, policy model.Policy, backend backend) *cqmcp.Service {
	t.Helper()
	service, err := cqmcp.New(policy, backend, cqmcp.Options{
		Audit:          mcpRepositoryAudit{},
		PolicyProvider: app.load,
		ActionAllowed:  app.actionAllowed,
	})
	if err != nil {
		t.Fatal(err)
	}
	return service
}
