package configuration

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/maldous/CircularQuay/internal/fault"
	"github.com/maldous/CircularQuay/internal/model"
	"github.com/maldous/CircularQuay/internal/platform"
)

func TestAdmissionMatrixContractRootRefusals(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	assertAdmissionParse(t, ctx, []byte(`{"version":1}`), resolver{}, fault.Unavailable, "cancelled")
	assertAdmissionParse(t, context.Background(), []byte(`{"version":`), resolver{}, fault.Invalid, "master_document")
}

func TestAdmissionMatrixContractDigestAdmission(t *testing.T) {
	for _, pin := range []string{"short", strings.Repeat("g", 64)} {
		fragment := fragmentDocument(t)
		include := localInclude(t, fragment)
		include.SHA256 = pin
		reader := admissionNoEffects(t)
		assertAdmissionParse(t, context.Background(), masterDocument(t, []model.Include{include}), reader, fault.Invalid, "include")
	}
}

func TestAdmissionMatrixContractLocalAndCacheAdmission(t *testing.T) {
	fragment := fragmentDocument(t)
	include := localInclude(t, fragment)
	include.File = fixturePath(t, "outside", "fragment.json")
	assertAdmissionParse(t, context.Background(), masterDocument(t, []model.Include{include}), admissionNoEffects(t), fault.Trust, "include_location")
	expires := time.Unix(100, 0).UTC()
	include = offlineRepositoryInclude(t, fragment, &expires)
	reader := admissionNoEffects(t)
	reader.now = func() time.Time { return expires }
	assertAdmissionParse(t, context.Background(), masterDocument(t, []model.Include{include}), reader, fault.Expired, "include_cache")
}

func TestAdmissionMatrixContractCancelBeforeNextInclude(t *testing.T) {
	fragment := fragmentDocument(t)
	first := localInclude(t, fragment)
	second := namedLocalInclude(t, "second", "second", fragment)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	reader := admissionNoEffects(t)
	reads := 0
	reader.readFile = func(path string, private bool) ([]byte, error) {
		reads++
		if path != first.File || !private || reads != 1 {
			t.Fatal("cancelled graph performed later read")
		}
		cancel()
		return fragment, nil
	}
	assertAdmissionParse(t, ctx, masterDocument(t, []model.Include{first, second}), reader, fault.Unavailable, "cancelled")
	if reads != 1 {
		t.Fatalf("initial read count=%d", reads)
	}
}

func TestAdmissionMatrixContractFragmentDecodeRefusals(t *testing.T) {
	cases := []struct {
		name string
		data []byte
		kind fault.Kind
		op   string
	}{
		{"yaml", []byte("version: 1\nsources: {}\nworkloads: {}\n"), fault.Unsupported, "include_yaml"},
		{"json", []byte(`{"version":`), fault.Invalid, "include_document"},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			include := localInclude(t, test.data)
			reader := admissionNoEffects(t)
			reader.readFile = func(path string, private bool) ([]byte, error) {
				if path != include.File || !private {
					t.Fatal("unexpected local read")
				}
				return test.data, nil
			}
			assertAdmissionParse(t, context.Background(), masterDocument(t, []model.Include{include}), reader, test.kind, test.op)
		})
	}
}

func TestAdmissionMatrixContractRepositoryContentRefusal(t *testing.T) {
	fragment := fragmentDocument(t)
	include := offlineRepositoryInclude(t, fragment, nil)
	include.Refresh = "online"
	reader := admissionNoEffects(t)
	release := filepath.Join(include.Repository.Directory, "releases", include.Commit)
	fetches, reads := 0, 0
	reader.fetch = func(_ context.Context, repository model.Repository, commit string) (string, error) {
		fetches++
		if !reflect.DeepEqual(repository, *include.Repository) || commit != include.Commit {
			t.Fatal("unexpected repository input")
		}
		return release, nil
	}
	reader.readFile = func(path string, private bool) ([]byte, error) {
		reads++
		if path != filepath.Join(release, include.Path) || private {
			t.Fatal("unexpected repository read")
		}
		return nil, errors.New("synthetic private read detail")
	}
	assertAdmissionParse(t, context.Background(), masterDocument(t, []model.Include{include}), reader, fault.Trust, "include_content")
	if reads != 1 || fetches != 1 {
		t.Fatalf("repository calls=%d/%d", reads, fetches)
	}
}

func TestAdmissionMatrixContractCredentialReferenceBase(t *testing.T) {
	fragment := marshalValue(t, map[string]any{
		"version":   1,
		"sources":   map[string]model.Source{"data": {Type: "vault", Endpoint: "https://vault.example.test", Mount: "kv", Resources: []string{"value"}, Identity: "token_file", TokenFile: fixturePath(t, "other", "token")}},
		"workloads": map[string]any{},
	})
	include := localInclude(t, fragment)
	reader := admissionNoEffects(t)
	reader.readFile = func(path string, private bool) ([]byte, error) {
		if path != include.File || !private {
			t.Fatal("unexpected include read")
		}
		return fragment, nil
	}
	assertAdmissionParse(t, context.Background(), masterDocument(t, []model.Include{include}), reader, fault.Invalid, "include_fragment")
}

func TestAdmissionMatrixContractAggregateSize(t *testing.T) {
	// Nine valid, individually bounded 950,000-byte fragments cross the 8 MiB
	// graph limit. One immutable buffer is shared; no large file is created.
	fragment := fragmentDocument(t)
	fragment = append(fragment, bytes.Repeat([]byte(" "), 950000-len(fragment))...)
	includes := make([]model.Include, 9)
	files := make(map[string][]byte)
	for index := range includes {
		name := fmt.Sprintf("part%d", index)
		includes[index] = namedLocalInclude(t, name, name, fragment)
		files[includes[index].File] = fragment
	}
	reader := admissionNoEffects(t)
	reads := 0
	reader.readFile = func(path string, private bool) ([]byte, error) {
		reads++
		data, exists := files[path]
		if !exists || !private {
			t.Fatal("unowned aggregate read")
		}
		return data, nil
	}
	before := digest(fragment)
	assertAdmissionParse(t, context.Background(), masterDocument(t, includes), reader, fault.Invalid, "composition_size")
	if reads != 9 || digest(fragment) != before {
		t.Fatalf("aggregate read count=%d or bytes changed", reads)
	}
}

func admissionNoEffects(t *testing.T) resolver {
	t.Helper()
	return resolver{
		readFile: func(string, bool) ([]byte, error) { t.Fatal("unexpected read"); return nil, nil },
		fetch: func(context.Context, model.Repository, string) (string, error) {
			t.Fatal("unexpected fetch")
			return "", nil
		},
		current: func(context.Context, model.Repository) (string, error) { t.Fatal("unexpected current"); return "", nil },
		now:     func() time.Time { return time.Unix(100, 0).UTC() },
	}
}

func assertAdmissionParse(t *testing.T, ctx context.Context, data []byte, reader resolver, kind fault.Kind, operation string) {
	t.Helper()
	before := append([]byte(nil), data...)
	result, err := parse(ctx, data, configurationPath(t), reader)
	actual, ok := err.(*fault.Error)
	operation = "configuration_" + operation
	if !ok || actual.Kind != kind || actual.Operation != operation || err.Error() != string(kind)+": "+operation || errors.Unwrap(err) != nil {
		t.Fatalf("unexpected bounded refusal: %v", err)
	}
	if !reflect.DeepEqual(result, Result{}) || !bytes.Equal(data, before) {
		t.Fatal("refusal exposed state or mutated input bytes")
	}
}

func TestAdmissionMatrixContractOwnedRootLoad(t *testing.T) {
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	root = filepath.Join(root, "private")
	if err := platform.EnsurePrivateDirectory(root); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(root, "policy.json")
	data := marshalPolicy(t, model.Policy{Version: 1,
		Sources:   map[string]model.Source{"data": {Type: "json", Path: filepath.Join(root, "absent-values.json")}},
		Workloads: map[string]model.Workload{"job": {Mode: "offline", Bindings: []model.Binding{{Name: "VALUE", Source: "data", Field: "value", AllowOffline: true}}}},
	})
	if err := os.WriteFile(file, data, 0o600); err != nil {
		t.Fatal(err)
	}
	assertAdmissionOwnedLoadControl(t, file)
	assertAdmissionOwnedLoadRefusal(t, context.Background(), file+".absent", fault.Trust, "master_file")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	assertAdmissionOwnedLoadRefusal(t, ctx, file, fault.Unavailable, "cancelled")
	assertAdmissionOwnedRootUnchanged(t, root, file, data)
}

func assertAdmissionOwnedLoadControl(t *testing.T, file string) {
	t.Helper()
	result, err := Load(context.Background(), file)
	if err != nil || len(result.Policy.Sources) != 1 || len(result.Policy.Workloads) != 1 || result.Policy.ConfigurationGuard == nil {
		t.Fatalf("owned root load failed: %v", err)
	}
	workload := result.Policy.Workloads["job"]
	if len(workload.Bindings) != 1 || workload.Bindings[0].Source != "data" || result.Policy.Sources["data"].Path != filepath.Join(filepath.Dir(file), "absent-values.json") {
		t.Fatal("owned load altered source/binding identity")
	}
}

func assertAdmissionOwnedRootUnchanged(t *testing.T, root, file string, data []byte) {
	t.Helper()
	entries, err := os.ReadDir(root)
	if err != nil || len(entries) != 1 || entries[0].Name() != "policy.json" {
		t.Fatal("root load created a source, credential, or child path")
	}
	actual, err := os.ReadFile(file)
	if err != nil || !bytes.Equal(actual, data) {
		t.Fatal("root load changed its declaration")
	}
}

func assertAdmissionOwnedLoadRefusal(t *testing.T, ctx context.Context, path string, kind fault.Kind, operation string) {
	t.Helper()
	result, err := Load(ctx, path)
	actual, ok := err.(*fault.Error)
	operation = "configuration_" + operation
	if !ok || actual.Kind != kind || actual.Operation != operation || err.Error() != string(kind)+": "+operation || errors.Unwrap(err) != nil {
		t.Fatalf("owned load refusal: %v", err)
	}
	if !reflect.DeepEqual(result, Result{}) {
		t.Fatal("owned load refusal exposed partial result")
	}
}

func TestAdmissionMatrixContractFinalOperationalReferences(t *testing.T) {
	for _, composed := range []bool{false, true} {
		assertAdmissionFinalReferences(t, composed)
	}
}

func assertAdmissionFinalReferences(t *testing.T, composed bool) {
	t.Helper()
	root := t.TempDir()
	candidate := model.Policy{Version: 1,
		Sources:   map[string]model.Source{"root": {Type: "json", Path: filepath.Join(root, "root-values.json")}},
		Workloads: map[string]model.Workload{"job": {Mode: "offline", Bindings: []model.Binding{{Name: "VALUE", Source: "root", Field: "value", AllowOffline: true}}}},
		Updates:   &model.Updates{Check: "interactive", Channel: "stable", IntervalHours: 24, TimeoutSeconds: 10, TokenFile: filepath.Join(root, "absent-token")},
	}
	reader := admissionNoEffects(t)
	if composed {
		fragment := fragmentDocument(t)
		include := localInclude(t, fragment)
		candidate.Composition = &model.Composition{Conflicts: "reject", Includes: []model.Include{include}}
		reader.readFile = func(path string, private bool) ([]byte, error) {
			if path != include.File || !private {
				t.Fatal("unexpected composed reference read")
			}
			return fragment, nil
		}
	}
	data := marshalPolicy(t, candidate)
	before := append([]byte(nil), data...)
	result, err := parse(context.Background(), data, filepath.Join(root, "policy.json"), reader)
	assertAdmissionReferenceResult(t, result, err, "updates_token_file")
	entries, readErr := os.ReadDir(root)
	if readErr != nil || len(entries) != 0 || !bytes.Equal(data, before) {
		t.Fatal("final reference refusal changed input or created operational paths")
	}
}

func TestAdmissionMatrixContractForeignPrivateReference(t *testing.T) {
	fragment := fragmentDocument(t)
	include := localInclude(t, fragment)
	data := masterDocumentWithRoot(t, []model.Include{include}, nil, filepath.Join(include.BaseDirectory, "other-values.json"))
	before := append([]byte(nil), data...)
	reader := admissionNoEffects(t)
	reader.readFile = func(path string, private bool) ([]byte, error) {
		if path != include.File || !private {
			t.Fatal("unexpected private include read")
		}
		return fragment, nil
	}
	result, err := parse(context.Background(), data, configurationPath(t), reader)
	assertAdmissionReferenceResult(t, result, err, "configuration_private_reference")
	if !bytes.Equal(data, before) {
		t.Fatal("cross-domain refusal mutated master bytes")
	}
}

func assertAdmissionReferenceResult(t *testing.T, result Result, err error, operation string) {
	t.Helper()
	actual, ok := err.(*fault.Error)
	if !ok || actual.Kind != fault.Invalid || actual.Operation != operation || err.Error() != "invalid_policy: "+operation || errors.Unwrap(err) != nil {
		t.Fatalf("unexpected final reference refusal: %v", err)
	}
	if !reflect.DeepEqual(result, Result{}) {
		t.Fatal("final reference refusal exposed a compiled result")
	}
}
