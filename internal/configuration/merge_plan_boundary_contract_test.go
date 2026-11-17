package configuration

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/maldous/CircularQuay/internal/fault"
	"github.com/maldous/CircularQuay/internal/model"
)

// These fixtures enter the actual serialized master/fragment pipeline. All
// repository and file callbacks are explicit, finite, and synthetic.
func TestMergePlanBoundaryContractQualifiedWorkloads(t *testing.T) {
	fixture := newMergePlanFixture(t, 2)
	fixture.includes[1].DependsOn = []string{fixture.includes[0].ID}
	fixture.fragments[1] = mergePlanFragment(t, "data", "job", "n0__data")
	result, reads, fetches := fixture.run(t, nil, "", "")
	if reads != 2 || fetches != 2 || len(result.Policy.Sources) != 2 || len(result.Policy.Workloads) != 2 {
		t.Fatalf("unexpected complete graph: reads=%d fetches=%d result=%#v", reads, fetches, result)
	}
	for _, name := range []string{"n0__job", "n1__job"} {
		workload := result.Policy.Workloads[name]
		if len(workload.Bindings) != 1 || workload.Bindings[0].Source != "n0__data" || workload.Bindings[0].Name != "VALUE" {
			t.Fatalf("qualified binding %q = %#v", name, workload)
		}
	}
	assertMergePlanOrigins(t, fixture, result)
}

func TestMergePlanBoundaryContractCrossReferenceRefusals(t *testing.T) {
	for _, source := range []string{"n0__data", "unknown__data", "n0__", "__data"} {
		t.Run(source, func(t *testing.T) {
			assertMergePlanCrossRefusal(t, source)
		})
	}
}

func assertMergePlanCrossRefusal(t *testing.T, source string) {
	t.Helper()
	fixture := newMergePlanFixture(t, 2)
	if source != "n0__data" {
		fixture.includes[1].DependsOn = []string{fixture.includes[0].ID}
	}
	fixture.fragments[1] = mergePlanFragment(t, "data", "job", source)
	_, reads, fetches := fixture.run(t, nil, fault.Denied, "include_cross_reference")
	if reads != 2 || fetches != 2 {
		t.Fatalf("refusal before actual second fragment: reads=%d fetches=%d", reads, fetches)
	}
}

func TestMergePlanBoundaryContractQualifiedNameBounds(t *testing.T) {
	for _, workload := range []bool{false, true} {
		t.Run(fmt.Sprintf("workload_%t", workload), func(t *testing.T) {
			assertMergePlanNameBound(t, workload, 60, "")
			assertMergePlanNameBound(t, workload, 63, fault.Invalid)
		})
	}
}

func assertMergePlanNameBound(t *testing.T, workload bool, length int, kind fault.Kind) {
	t.Helper()
	fixture := newMergePlanFixture(t, 1)
	name := strings.Repeat("a", length)
	source, job := name, "job"
	if workload {
		source, job = "data", name
	}
	fixture.fragments[0] = mergePlanFragment(t, source, job, source)
	_, reads, fetches := fixture.run(t, nil, kind, "include_qualified_name")
	if reads != 1 || fetches != 1 {
		t.Fatalf("overflow did not reach child qualification: %d/%d", reads, fetches)
	}
}

func TestMergePlanBoundaryContractRootCollisions(t *testing.T) {
	for _, workload := range []bool{false, true} {
		t.Run(fmt.Sprintf("workload_%t", workload), func(t *testing.T) {
			assertMergePlanRootCollision(t, workload)
		})
	}
}

func assertMergePlanRootCollision(t *testing.T, workload bool) {
	t.Helper()
	fixture := newMergePlanFixture(t, 1)
	edit := func(root *model.Policy) {
		root.Sources["n0__data"] = model.Source{Type: "json", Path: filepath.Join(fixture.base, "root-values.json")}
	}
	if workload {
		edit = func(root *model.Policy) {
			root.Workloads["n0__job"] = model.Workload{Mode: "offline", Bindings: []model.Binding{{Name: "ROOT", Source: "n0__data", Field: "value", AllowOffline: true}}}
		}
	}
	_, reads, fetches := fixture.run(t, edit, fault.Invalid, "composition_conflict")
	if reads != 1 || fetches != 1 {
		t.Fatalf("collision did not reach merge: %d/%d", reads, fetches)
	}
}

func TestMergePlanBoundaryContractPlanAdmission(t *testing.T) {
	for _, name := range []string{"conflicts", "empty", "id", "namespace", "origin", "short_commit", "nonhex_commit"} {
		t.Run(name, func(t *testing.T) {
			assertMergePlanAdmission(t, name)
		})
	}
}

func assertMergePlanAdmission(t *testing.T, name string) {
	t.Helper()
	fixture := newMergePlanFixture(t, 2)
	operation, edit := mergePlanAdmissionEdit(name)
	_, reads, fetches := fixture.run(t, edit, fault.Invalid, operation)
	if reads != 0 || fetches != 0 {
		t.Fatalf("plan refusal read or fetched: %d/%d", reads, fetches)
	}
}

func mergePlanAdmissionEdit(name string) (string, func(*model.Policy)) {
	switch name {
	case "conflicts":
		return "composition", func(root *model.Policy) { root.Composition.Conflicts = "replace" }
	case "empty":
		return "composition", func(root *model.Policy) { root.Composition.Includes = []model.Include{} }
	case "id":
		return "include_id", func(root *model.Policy) { root.Composition.Includes[1].ID = root.Composition.Includes[0].ID }
	case "namespace":
		return "include_namespace", func(root *model.Policy) {
			root.Composition.Includes[1].Namespace = root.Composition.Includes[0].Namespace
		}
	case "origin":
		return "include_origin", mergePlanDuplicateOrigin
	case "short_commit":
		return "include", func(root *model.Policy) { root.Composition.Includes[0].Commit = "abc" }
	default:
		return "include", func(root *model.Policy) { root.Composition.Includes[0].Commit = strings.Repeat("g", 40) }
	}
}

func mergePlanDuplicateOrigin(root *model.Policy) {
	first := root.Composition.Includes[0]
	root.Composition.Includes[1].Repository = first.Repository
	root.Composition.Includes[1].Commit = first.Commit
	root.Composition.Includes[1].Path = first.Path
}

func TestMergePlanBoundaryContractDependencyDepth(t *testing.T) {
	fixture := newMergePlanFixture(t, 8)
	mergePlanChain(fixture.includes)
	result, reads, fetches := fixture.run(t, nil, "", "")
	if reads != 8 || fetches != 8 || len(result.Policy.Workloads) != 8 {
		t.Fatalf("depth-eight control = %d/%d/%d", reads, fetches, len(result.Policy.Workloads))
	}
	fixture = newMergePlanFixture(t, 9)
	mergePlanChain(fixture.includes)
	_, reads, fetches = fixture.run(t, nil, fault.Invalid, "include_depth")
	if reads != 0 || fetches != 0 {
		t.Fatalf("depth refusal performed effects: %d/%d", reads, fetches)
	}
}

func mergePlanChain(includes []model.Include) {
	for index := 1; index < len(includes); index++ {
		includes[index].DependsOn = []string{includes[index-1].ID}
	}
}

type mergePlanFixture struct {
	base      string
	includes  []model.Include
	fragments [][]byte
}

func newMergePlanFixture(t *testing.T, count int) mergePlanFixture {
	t.Helper()
	fixture := mergePlanFixture{base: t.TempDir()}
	for index := 0; index < count; index++ {
		name := fmt.Sprintf("n%d", index)
		fixture.fragments = append(fixture.fragments, mergePlanFragment(t, "data", "job", "data"))
		fixture.includes = append(fixture.includes, model.Include{
			ID: name, Namespace: name, Commit: strings.Repeat("a", 40), Path: "fragment.json",
			BaseDirectory: filepath.Join(fixture.base, "data", name), Visibility: "shared", Refresh: "online",
			Repository: &model.Repository{
				URL: "https://github.com/example/configuration.git", Directory: filepath.Join(fixture.base, "cache", name),
				Ref: "refs/heads/main", AllowedSignersFile: filepath.Join(fixture.base, "trust", name), InitialCommit: strings.Repeat("b", 40),
			},
		})
	}
	return fixture
}

func mergePlanFragment(t *testing.T, source, job, binding string) []byte {
	t.Helper()
	return marshalValue(t, map[string]any{
		"version": 1,
		"sources": map[string]model.Source{source: {Type: "json", Path: "values.json"}},
		"workloads": map[string]any{job: map[string]any{
			"mode":     "offline",
			"bindings": []model.Binding{{Name: "VALUE", Source: binding, Field: "value", AllowOffline: true}},
		}},
	})
}

func (fixture mergePlanFixture) run(t *testing.T, edit func(*model.Policy), kind fault.Kind, operation string) (Result, int, int) {
	t.Helper()
	files := make(map[string][]byte)
	for index := range fixture.includes {
		include := &fixture.includes[index]
		include.SHA256 = digest(fixture.fragments[index])
		file := filepath.Join(include.Repository.Directory, "releases", include.Commit, include.Path)
		files[file] = append([]byte(nil), fixture.fragments[index]...)
	}
	root := model.Policy{Version: 1, Sources: map[string]model.Source{}, Workloads: map[string]model.Workload{}, Composition: &model.Composition{Includes: append([]model.Include(nil), fixture.includes...), Conflicts: "reject"}}
	if edit != nil {
		edit(&root)
	}
	data := marshalPolicy(t, root)
	before := append([]byte(nil), data...)
	reads, fetches := 0, 0
	reader := fixture.reader(t, files, &reads, &fetches)
	result, err := parse(context.Background(), data, filepath.Join(fixture.base, "master.json"), reader)
	if !bytes.Equal(data, before) {
		t.Fatal("master bytes mutated")
	}
	fixture.assertFragments(t, files)
	assertMergePlanOutcome(t, result, err, kind, operation)
	return result, reads, fetches
}

func (fixture mergePlanFixture) reader(t *testing.T, files map[string][]byte, reads, fetches *int) resolver {
	t.Helper()
	return resolver{
		readFile: func(path string, private bool) ([]byte, error) {
			*reads++
			data, found := files[path]
			if !found || private {
				t.Fatalf("unexpected fragment read %q private=%t", path, private)
			}
			return data, nil
		},
		fetch: func(_ context.Context, repository model.Repository, commit string) (string, error) {
			*fetches++
			return fixture.release(t, repository, commit), nil
		},
		current: func(context.Context, model.Repository) (string, error) {
			t.Fatal("offline fallback called")
			return "", nil
		},
		now: func() time.Time { return time.Unix(100, 0).UTC() },
	}
}

func (fixture mergePlanFixture) release(t *testing.T, repository model.Repository, commit string) string {
	t.Helper()
	for _, include := range fixture.includes {
		if reflect.DeepEqual(repository, *include.Repository) && commit == include.Commit {
			return filepath.Join(repository.Directory, "releases", commit)
		}
	}
	t.Fatal("unexpected repository or commit")
	return ""
}

func (fixture mergePlanFixture) assertFragments(t *testing.T, files map[string][]byte) {
	t.Helper()
	for index, include := range fixture.includes {
		file := filepath.Join(include.Repository.Directory, "releases", include.Commit, include.Path)
		if !bytes.Equal(files[file], fixture.fragments[index]) {
			t.Fatal("fragment bytes mutated")
		}
	}
}

func assertMergePlanOutcome(t *testing.T, result Result, err error, kind fault.Kind, operation string) {
	t.Helper()
	if kind == "" {
		if err != nil {
			t.Fatalf("valid composed graph: %v", err)
		}
		return
	}
	actual, ok := err.(*fault.Error)
	expected := "configuration_" + operation
	if !ok || actual.Kind != kind || actual.Operation != expected || err.Error() != string(kind)+": "+expected || errors.Unwrap(err) != nil {
		t.Fatalf("unexpected refusal: %v", err)
	}
	if !reflect.DeepEqual(result, Result{}) {
		t.Fatalf("partial result after refusal: %#v", result)
	}
}

func assertMergePlanOrigins(t *testing.T, fixture mergePlanFixture, result Result) {
	t.Helper()
	expected := make([]Provenance, 0, len(fixture.includes))
	for _, include := range fixture.includes {
		expected = append(expected, Provenance{ID: include.ID, Namespace: include.Namespace, SHA256: include.SHA256})
	}
	if !reflect.DeepEqual(result.Provenance(), expected) || result.Policy.ConfigurationGuard == nil {
		t.Fatal("complete shared provenance or guard missing")
	}
	for _, include := range fixture.includes {
		source := result.Policy.Sources[include.Namespace+"__data"]
		if source.Path != filepath.Join(include.BaseDirectory, "values.json") || result.Policy.ConfigurationGuard.SourceDomains[include.Namespace+"__data"] != include.ID {
			t.Fatalf("source origin changed: %#v", source)
		}
	}
}
