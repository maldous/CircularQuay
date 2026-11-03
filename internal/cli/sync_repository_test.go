package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/maldous/CircularQuay/internal/fault"
	"github.com/maldous/CircularQuay/internal/model"
	"github.com/maldous/CircularQuay/internal/platform"
)

func TestOfflineRepositoryUsesCurrentWithoutFetch(t *testing.T) {
	called := 0
	operations := repositoryOperations{
		fetch:   func(context.Context, model.Repository) (string, error) { called++; return "", nil },
		current: func(model.Repository) (string, error) { return "/private/release", nil },
	}
	release, err := repositoryRelease(context.Background(), "offline", model.Repository{}, literalRepositorySource(true), operations)
	if err != nil || release != "/private/release" || called != 0 {
		t.Fatalf("offline repository transport = %q, %v, fetches=%d", release, err, called)
	}
}

func TestHybridRepositoryFallsBackOnlyForAllowedAvailabilityFailure(t *testing.T) {
	current := 0
	operations := repositoryOperations{
		fetch: func(context.Context, model.Repository) (string, error) {
			return "", fault.New(fault.Unavailable, "synthetic transport")
		},
		current: func(model.Repository) (string, error) { current++; return "/private/release", nil },
	}
	release, err := repositoryRelease(context.Background(), "hybrid", model.Repository{}, literalRepositorySource(true), operations)
	if err != nil || release != "/private/release" || current != 1 {
		t.Fatalf("allowed hybrid fallback = %q, %v, current=%d", release, err, current)
	}
	if _, err = repositoryRelease(context.Background(), "hybrid", model.Repository{}, literalRepositorySource(false), operations); !fault.Is(err, fault.Unavailable) || current != 1 {
		t.Fatalf("ineligible hybrid fallback = %v, current=%d", err, current)
	}
}

func TestRepositoryDoesNotFallbackOnRejectedFetch(t *testing.T) {
	for _, kind := range []fault.Kind{fault.Denied, fault.Trust, fault.Invalid} {
		t.Run(string(kind), func(t *testing.T) {
			current := 0
			operations := repositoryOperations{
				fetch: func(context.Context, model.Repository) (string, error) {
					return "", fault.New(kind, "synthetic rejection")
				},
				current: func(model.Repository) (string, error) { current++; return "/private/release", nil },
			}
			_, err := repositoryRelease(context.Background(), "hybrid", model.Repository{}, literalRepositorySource(true), operations)
			if !fault.Is(err, kind) || current != 0 {
				t.Fatalf("rejected fetch error = %v, current=%d", err, current)
			}
		})
	}
}

func TestRepositorySourceExpansionSelectsOnlyWorkloadSources(t *testing.T) {
	config, workload := repositoryPolicyFixture(t, "hybrid", true)
	config.Sources["unused"] = model.Source{Type: "json", Path: "repo:unused.json"}
	config.Grants = []model.Grant{{Subject: "operator", Workload: "demo", Operations: []string{"status"}}}
	selected, err := repositorySources(config, workload)
	if err != nil {
		t.Fatal(err)
	}
	updated, err := expandRepositorySources(config, selected, filepath.Join(t.TempDir(), "release"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.HasPrefix(updated.Sources["selected"].Path, "repo:") || updated.Sources["unused"].Path != "repo:unused.json" {
		t.Fatalf("repository source selection = %+v", updated.Sources)
	}
	if !reflect.DeepEqual(updated.Grants, config.Grants) || !reflect.DeepEqual(updated.Repository, config.Repository) {
		t.Fatal("repository materialization changed local policy authority")
	}
}

func TestWorkloadCommandRejectsInvalidTargetBeforeRepositoryFetch(t *testing.T) {
	config, _ := repositoryPolicyFixture(t, "offline", true)
	path := writeRepositoryPolicy(t, config)
	fetches := 0
	app := application{config: path, directory: filepath.Join(filepath.Dir(path), "state"), repository: repositoryOperations{
		fetch:   func(context.Context, model.Repository) (string, error) { fetches++; return "", nil },
		current: func(model.Repository) (string, error) { return "", nil },
	}}
	if _, err := app.workloadCommand(context.Background(), []string{"sync", "--workload", "demo", "--target", "missing"}); !fault.Is(err, fault.Invalid) && !fault.Is(err, fault.Denied) {
		t.Fatalf("invalid target error = %v", err)
	}
	if fetches != 0 {
		t.Fatalf("invalid target fetched repository %d times", fetches)
	}
}

func TestWorkloadCommandResolvesOnlyAcceptedOfflineRepositoryRelease(t *testing.T) {
	config, _ := repositoryPolicyFixture(t, "offline", true)
	release := filepath.Join(filepath.Dir(config.Repository.Directory), "release")
	if err := platform.EnsurePrivateDirectory(release); err != nil {
		t.Fatal(err)
	}
	writeProtectedFixture(t, filepath.Join(release, "values.json"), []byte(`{"value":"synthetic"}`))
	path := writeRepositoryPolicy(t, config)
	var output bytes.Buffer
	fetches, currents := 0, 0
	app := application{config: path, directory: filepath.Join(filepath.Dir(path), "state"), runner: Runner{Out: &output}, repository: repositoryOperations{
		fetch:   func(context.Context, model.Repository) (string, error) { fetches++; return "", nil },
		current: func(model.Repository) (string, error) { currents++; return release, nil },
	}}
	if _, err := app.workloadCommand(context.Background(), []string{"sync", "--once", "--workload", "demo"}); err != nil {
		t.Fatal(err)
	}
	var record struct {
		State string `json:"state"`
	}
	if err := json.Unmarshal(output.Bytes(), &record); err != nil || record.State != "resolved" || fetches != 0 || currents != 1 {
		t.Fatalf("offline repository sync state=%q fetches=%d currents=%d err=%v", record.State, fetches, currents, err)
	}
}

func literalRepositorySource(allowOffline bool) map[string]repositorySource {
	return map[string]repositorySource{"selected": {source: model.Source{Type: "json"}, allowOffline: allowOffline}}
}

func repositoryPolicyFixture(t *testing.T, mode string, allowOffline bool) (model.Policy, model.Workload) {
	t.Helper()
	directory := privateCLIDirectory(t)
	repository := model.Repository{
		URL: "https://example.test/synthetic.git", Directory: filepath.Join(directory, "repository"), Ref: "refs/heads/data",
		AllowedSignersFile: filepath.Join(directory, "signers"), InitialCommit: strings.Repeat("a", 40),
	}
	workload := model.Workload{Mode: mode, Bindings: []model.Binding{{Name: "VALUE", Source: "selected", Field: "value", AllowOffline: allowOffline}}}
	return model.Policy{Version: 1, Repository: repository, Sources: map[string]model.Source{"selected": {Type: "json", Path: "repo:values.json"}}, Workloads: map[string]model.Workload{"demo": workload}}, workload
}

func writeRepositoryPolicy(t *testing.T, config model.Policy) string {
	t.Helper()
	data, err := json.Marshal(config)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(filepath.Dir(config.Repository.Directory), "policy.json")
	if err = platform.WriteProtected(path, data); err != nil {
		t.Fatal(err)
	}
	return path
}
