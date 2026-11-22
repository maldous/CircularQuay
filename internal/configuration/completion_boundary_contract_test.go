package configuration

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/maldous/CircularQuay/internal/fault"
	"github.com/maldous/CircularQuay/internal/model"
	"github.com/maldous/CircularQuay/internal/policy"
	"gopkg.in/yaml.v3"
)

func completionConfigurationFailure(t *testing.T, result Result, err error, kind fault.Kind, operation string) {
	t.Helper()
	if !reflect.DeepEqual(result, Result{}) || !fault.Is(err, kind) || err.Error() != string(kind)+": configuration_"+operation {
		t.Fatal("configuration boundary retained partial output or changed refusal")
	}
}

func TestCompletionConfigurationLocationFailuresBeforeEffects(t *testing.T) {
	calls := 0
	absolute := func(string) (string, error) {
		calls++
		return "", errors.New("synthetic cwd lookup failure")
	}
	result, err := loadWithAbsolute(context.Background(), "relative.json", absolute)
	completionConfigurationFailure(t, result, err, fault.Invalid, "master_location")
	result, err = parseWithAbsolute(context.Background(), nil, "relative.json", admissionNoEffects(t), absolute)
	completionConfigurationFailure(t, result, err, fault.Invalid, "master_location")
	root, err := readRootWithAbsolute(nil, "relative.json", absolute)
	if !reflect.DeepEqual(root, model.Policy{}) || !fault.Is(err, fault.Invalid) || err.Error() != "invalid_policy: configuration_master_location" || calls != 3 {
		t.Fatal("root location failure escaped its exact admission boundary")
	}
}

func TestCompletionConfigurationSinglePostValidationCancellation(t *testing.T) {
	path := fixturePath(t, "master", "policy.json")
	data := marshalPolicy(t, model.Policy{
		Version: 1,
		Sources: map[string]model.Source{
			"root": {Type: "json", Path: filepath.Join(filepath.Dir(path), "root.json")},
		},
		Workloads: map[string]model.Workload{
			"root": {Mode: "offline", Bindings: []model.Binding{
				{Name: "ROOT_VALUE", Source: "root", Field: "value", AllowOffline: true},
			}},
		},
	})
	if result, err := resolveSingle(context.Background(), data, path); err != nil || result.Policy.ConfigurationGuard == nil {
		t.Fatal("single-policy positive control failed")
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	calls := 0
	validate := func(candidate model.Policy, name string) error {
		calls++
		err := policy.ValidateResolvedReferences(candidate, name)
		cancel()
		return err
	}
	result, err := resolveSingleWithValidation(ctx, data, path, validate)
	completionConfigurationFailure(t, result, err, fault.Unavailable, "cancelled")
	if calls != 1 {
		t.Fatal("post-validation cancellation was not reached exactly once")
	}
}

func TestCompletionConfigurationComparablePathUsesNativeCaseContract(t *testing.T) {
	path := filepath.Join(t.TempDir(), "MixedCase")
	want := filepath.Clean(path)
	if runtime.GOOS == "windows" {
		want = strings.ToLower(want)
	}
	if comparablePath(path) != want {
		t.Fatal("native canonicalization changed")
	}
}

func completionYAMLMaster(t *testing.T, raw json.RawMessage) editBoundaryFixture {
	t.Helper()
	fixture := newEditBoundaryFixture(t, nil, nil)
	root, err := ReadRoot(fixture.data, fixture.path)
	if err != nil {
		t.Fatal(err)
	}
	root.Composition.Overrides = []model.PolicyEdit{{Pointer: "/sources/root/path", Value: raw}}
	fixture.data, err = yaml.Marshal(root)
	if err != nil {
		t.Fatal(err)
	}
	admitted, err := ReadRoot(fixture.data, fixture.path)
	if err != nil || admitted.Composition == nil || len(admitted.Composition.Overrides) != 1 {
		t.Fatal("serialized YAML master was not admitted", err)
	}
	if !bytes.Equal(admitted.Composition.Overrides[0].Value, raw) {
		t.Fatal("YAML byte sequence did not preserve the raw edit value")
	}
	return fixture
}

func TestCompletionConfigurationYAMLRawEditBoundaries(t *testing.T) {
	for _, raw := range []string{` "owned"`, `{malformed`, `null`} {
		t.Run(raw, func(t *testing.T) {
			fixture := completionYAMLMaster(t, json.RawMessage(raw))
			result, err := resolveEditBoundaryFixture(t, fixture)
			completionConfigurationFailure(t, result, err, fault.Invalid, "composition_edit")
		})
	}
}

func TestCompletionConfigurationYAMLRawEditPositiveControl(t *testing.T) {
	path := fixturePath(t, "replacement", "values.json")
	fixture := completionYAMLMaster(t, rawString(t, path))
	result, err := resolveEditBoundaryFixture(t, fixture)
	if err != nil || result.Policy.Sources["root"].Path != path || result.Policy.Composition != nil {
		t.Fatal("valid YAML byte-sequence edit was not resolved", err)
	}
}
