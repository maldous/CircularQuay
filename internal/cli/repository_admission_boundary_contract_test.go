package cli

import (
	"context"
	"errors"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/maldous/CircularQuay/internal/fault"
	"github.com/maldous/CircularQuay/internal/model"
	"github.com/maldous/CircularQuay/internal/resolve"
)

func TestCLISafeRepositoryFailurePropagation(t *testing.T) {
	for _, name := range []string{"cancelled", "fetch", "current", "relative"} {
		t.Run(name, func(t *testing.T) { cliSafeRepositoryFailure(t, name) })
	}
}

func cliSafeRepositoryFailure(t *testing.T, name string) {
	t.Helper()
	workload := model.Workload{Mode: "online", Bindings: []model.Binding{{Name: "VALUE", Source: "fixture", Field: "VALUE"}}}
	policy := model.Policy{Sources: map[string]model.Source{"fixture": {Type: "json", Path: "repo:fixture.json"}}}
	engine := &resolve.Engine{Policy: policy}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if name == "cancelled" {
		cancel()
	}
	failure := fault.New(fault.Trust, "synthetic repository")
	fetches, currents := 0, 0
	operations := repositoryOperations{
		fetch: func(context.Context, model.Repository) (string, error) {
			fetches++
			if name == "fetch" {
				return "", failure
			}
			return "unused", nil
		},
		current: func(context.Context, model.Repository) (string, error) {
			currents++
			if name == "current" {
				return "", failure
			}
			return "relative-release", nil
		},
	}
	got, err := (application{repository: operations}).withRepositorySources(ctx, workload, engine)
	cliSafeRepositoryRefusal(t, name, err, failure)
	wantCalls := map[string][2]int{"cancelled": {0, 0}, "fetch": {1, 0}, "current": {1, 1}, "relative": {1, 1}}
	if [2]int{fetches, currents} != wantCalls[name] {
		t.Fatalf("callback counts = %d/%d", fetches, currents)
	}
	if got != nil || engine.Policy.Sources["fixture"].Path != "repo:fixture.json" {
		t.Fatalf("failure returned partial engine or changed source: %#v", got)
	}
}

func cliSafeRepositoryRefusal(t *testing.T, name string, err, failure error) {
	t.Helper()
	switch name {
	case "cancelled":
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("cancel refusal: %v", err)
		}
	case "fetch", "current":
		if err != failure {
			t.Fatalf("callback refusal: %v", err)
		}
	case "relative":
		requireCLISafeFault(t, err, fault.Trust, repositorySourceOperation)
	}
}

func TestCLISafeRepositoryExpansionPreservesOriginal(t *testing.T) {
	release := filepath.Join(privateCLIDirectory(t), "unmaterialized-release")
	source := model.Source{Type: "json", Path: "repo:fixture.json"}
	policy := model.Policy{Sources: map[string]model.Source{"fixture": source, "sibling": {Type: "json", Path: "untouched.json"}}}
	selected := map[string]repositorySource{"fixture": {source: source}}
	got, err := expandRepositorySources(policy, selected, release)
	if err != nil || got.Sources["fixture"].Path != filepath.Join(release, "fixture.json") {
		t.Fatalf("absolute expansion = %#v/%v", got, err)
	}
	if !reflect.DeepEqual(policy.Sources["fixture"], source) || !reflect.DeepEqual(policy.Sources["sibling"], got.Sources["sibling"]) {
		t.Fatal("expansion mutated original or sibling")
	}
	requireMissingPath(t, release)
}
