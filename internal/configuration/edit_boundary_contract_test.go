package configuration

import (
	"bytes"
	"context"
	"encoding/json"
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

const editBoundaryMarker = "synthetic-edit-value"

type editBoundaryFixture struct {
	data     []byte
	fragment []byte
	include  model.Include
	path     string
}

func TestEditBoundaryContractOverrideThenMatchingLock(t *testing.T) {
	replacement := fixturePath(t, "replacement", "values.json")
	edit := model.PolicyEdit{Pointer: "/sources/root/path", Value: rawString(t, replacement)}
	fixture := newEditBoundaryFixture(t, []model.PolicyEdit{edit}, []model.PolicyEdit{edit})
	result, err := resolveEditBoundaryFixture(t, fixture)
	if err != nil {
		t.Fatalf("valid override and final lock refused: %v", err)
	}
	if result.Policy.Sources["root"].Path != replacement {
		t.Fatal("matching lock did not observe the final overridden path")
	}
	if result.Policy.Sources["sales__data"].Path != filepath.Join(fixture.include.BaseDirectory, "values.json") {
		t.Fatal("root edit changed the independently bound fragment source")
	}
	expected := model.Workload{Mode: "offline", Bindings: []model.Binding{{Name: "ROOT_VALUE", Source: "root", Field: "value", AllowOffline: true}}, Environment: map[string]string{"SYNTHETIC_BASE": "unchanged"}}
	if !reflect.DeepEqual(result.Policy.Workloads["root"], expected) {
		t.Fatal("root source edit changed a sibling workload declaration")
	}
	if result.Policy.Composition != nil || result.Policy.ConfigurationGuard == nil || len(result.Provenance()) != 0 {
		t.Fatal("successful local edit lost its resolved-only authority boundary")
	}
}

func TestEditBoundaryContractPointerRefusals(t *testing.T) {
	prefix := "/sources/root/"
	cases := []struct {
		name    string
		pointer string
	}{
		{"missing_root_separator", "sources/root/path"},
		{"duplicate_separator", "/sources//root/path"},
		{"not_a_field", "/sources/root"},
		{"empty_leaf", "/sources/root/"},
		{"forbidden_authority", "/grants/root/name"},
		{"invalid_source_identifier", "/sources/_root/path"},
		{"invalid_escape", "/sources/root/~2"},
		{"unfinished_escape", "/sources/root/path~"},
		{"decoded_tilde_not_an_identifier", "/sources/ro~0ot/path"},
		{"decoded_slash_not_an_identifier", "/sources/ro~1ot/path"},
		{"pointer_byte_budget", prefix + strings.Repeat("x", 4097-len(prefix))},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			edit := model.PolicyEdit{Pointer: test.pointer, Value: rawString(t, editBoundaryMarker)}
			assertEditBoundaryRefusal(t, newEditBoundaryFixture(t, []model.PolicyEdit{edit}, nil), fault.Invalid, "composition_edit")
		})
	}
}

func TestEditBoundaryContractOverlappingOverrides(t *testing.T) {
	parent := model.PolicyEdit{Pointer: "/workloads/root/environment", Value: json.RawMessage(`{"SYNTHETIC_ONE":"first"}`)}
	child := model.PolicyEdit{Pointer: "/workloads/root/environment/SYNTHETIC_ONE", Value: rawString(t, "second")}
	cases := []struct {
		name  string
		edits []model.PolicyEdit
	}{
		{"same_field", []model.PolicyEdit{child, child}},
		{"ancestor_first", []model.PolicyEdit{parent, child}},
		{"descendant_first", []model.PolicyEdit{child, parent}},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			assertEditBoundaryRefusal(t, newEditBoundaryFixture(t, test.edits, nil), fault.Invalid, "composition_edit")
		})
	}
}

func TestEditBoundaryContractSiblingNamesDoNotOverlap(t *testing.T) {
	edits := []model.PolicyEdit{
		{Pointer: "/workloads/root/environment/SYNTHETIC_ONE", Value: rawString(t, "first")},
		{Pointer: "/workloads/root/environment/SYNTHETIC_ONE_MORE", Value: rawString(t, "second")},
	}
	result, err := resolveEditBoundaryFixture(t, newEditBoundaryFixture(t, edits, nil))
	if err != nil {
		t.Fatalf("distinct sibling fields refused: %v", err)
	}
	expected := map[string]string{"SYNTHETIC_BASE": "unchanged", "SYNTHETIC_ONE": "first", "SYNTHETIC_ONE_MORE": "second"}
	if !reflect.DeepEqual(result.Policy.Workloads["root"].Environment, expected) {
		t.Fatal("distinct sibling override values were lost or conflated")
	}
}

func TestEditBoundaryContractEditCountLimit(t *testing.T) {
	edits, environment := editBoundaryEnvironmentEdits(t, 128)
	result, err := resolveEditBoundaryFixture(t, newEditBoundaryFixture(t, edits, nil))
	if err != nil {
		t.Fatalf("128 disjoint field overrides refused: %v", err)
	}
	if !reflect.DeepEqual(result.Policy.Workloads["root"].Environment, environment) {
		t.Fatal("the maximum accepted edit list was truncated or changed")
	}
	excess, _ := editBoundaryEnvironmentEdits(t, 129)
	assertEditBoundaryRefusal(t, newEditBoundaryFixture(t, excess, nil), fault.Invalid, "composition_edit")
}

func TestEditBoundaryContractMissingAndNonMapParents(t *testing.T) {
	for _, pointer := range []string{"/sources/missing/path", "/sources/root/path/child", "/workloads/root/bindings/0/name"} {
		t.Run(pointer, func(t *testing.T) {
			edit := model.PolicyEdit{Pointer: pointer, Value: rawString(t, editBoundaryMarker)}
			assertEditBoundaryRefusal(t, newEditBoundaryFixture(t, []model.PolicyEdit{edit}, nil), fault.Invalid, "composition_override")
		})
	}
}

func TestEditBoundaryContractTypedResultRefusals(t *testing.T) {
	cases := []struct {
		name    string
		pointer string
		value   json.RawMessage
	}{
		{"path_is_not_a_number", "/sources/root/path", json.RawMessage(`123`)},
		{"unknown_source_field", "/sources/root/unknown", json.RawMessage(`"synthetic-edit-value"`)},
		{"bindings_are_not_an_object", "/workloads/root/bindings", json.RawMessage(`{"unexpected":true}`)},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			edit := model.PolicyEdit{Pointer: test.pointer, Value: test.value}
			assertEditBoundaryRefusal(t, newEditBoundaryFixture(t, []model.PolicyEdit{edit}, nil), fault.Invalid, "composition_result")
		})
	}
}

func TestEditBoundaryContractMissingLockTargets(t *testing.T) {
	for _, pointer := range []string{"/sources/missing/path", "/sources/root/profile"} {
		t.Run(pointer, func(t *testing.T) {
			lock := model.PolicyEdit{Pointer: pointer, Value: rawString(t, editBoundaryMarker)}
			assertEditBoundaryRefusal(t, newEditBoundaryFixture(t, nil, []model.PolicyEdit{lock}), fault.Denied, "composition_lock")
		})
	}
}

func TestEditBoundaryContractLaterFailureExposesNoPartialOverride(t *testing.T) {
	edits := []model.PolicyEdit{
		{Pointer: "/sources/root/path", Value: rawString(t, fixturePath(t, "replacement.json"))},
		{Pointer: "/sources/missing/path", Value: rawString(t, editBoundaryMarker)},
	}
	assertEditBoundaryRefusal(t, newEditBoundaryFixture(t, edits, nil), fault.Invalid, "composition_override")
}

func newEditBoundaryFixture(t *testing.T, overrides, locks []model.PolicyEdit) editBoundaryFixture {
	t.Helper()
	fragment := fragmentDocument(t)
	include := localInclude(t, fragment)
	path := configurationPath(t)
	master := model.Policy{
		Version:   1,
		Sources:   map[string]model.Source{"root": {Type: "json", Path: filepath.Join(filepath.Dir(path), "root.json")}},
		Workloads: map[string]model.Workload{"root": {Mode: "offline", Bindings: []model.Binding{{Name: "ROOT_VALUE", Source: "root", Field: "value", AllowOffline: true}}, Environment: map[string]string{"SYNTHETIC_BASE": "unchanged"}}},
		Composition: &model.Composition{
			Includes: []model.Include{include}, Conflicts: "reject", Overrides: overrides, Locks: locks,
		},
	}
	return editBoundaryFixture{data: marshalPolicy(t, master), fragment: fragment, include: include, path: path}
}

func resolveEditBoundaryFixture(t *testing.T, fixture editBoundaryFixture) (Result, error) {
	t.Helper()
	dataCopy := append([]byte(nil), fixture.data...)
	fragmentCopy := append([]byte(nil), fixture.fragment...)
	// Exercise the real byte decoder, include admission, edit resolver and final
	// policy checks. Only the declared synthetic include is read by the fake;
	// no global nativeResolver is changed or selected.
	result, err := parse(context.Background(), fixture.data, fixture.path, editBoundaryResolver(t, fixture))
	if !bytes.Equal(fixture.data, dataCopy) || !bytes.Equal(fixture.fragment, fragmentCopy) {
		t.Fatal("edit resolution modified caller-owned serialized input")
	}
	return result, err
}

func editBoundaryResolver(t *testing.T, fixture editBoundaryFixture) resolver {
	t.Helper()
	return resolver{
		readFile: func(path string, private bool) ([]byte, error) {
			if path != fixture.include.File || !private {
				t.Fatal("edit fixture requested a different input or custody scope")
			}
			return fixture.fragment, nil
		},
		fetch: func(context.Context, model.Repository, string) (string, error) {
			t.Fatal("edit fixture attempted a repository fetch")
			return "", errors.New("unexpected synthetic repository fetch")
		},
		current: func(context.Context, model.Repository) (string, error) {
			t.Fatal("edit fixture attempted a repository cache read")
			return "", errors.New("unexpected synthetic repository cache read")
		},
		now: func() time.Time { return time.Unix(100, 0).UTC() },
	}
}

func assertEditBoundaryRefusal(t *testing.T, fixture editBoundaryFixture, kind fault.Kind, operation string) {
	t.Helper()
	result, err := resolveEditBoundaryFixture(t, fixture)
	var refusal *fault.Error
	if !errors.As(err, &refusal) {
		t.Fatalf("edit did not return a typed refusal: %v", err)
	}
	if refusal.Kind != kind || refusal.Operation != "configuration_"+operation {
		t.Fatalf("refusal = %q/%q, want %q/%q", refusal.Kind, refusal.Operation, kind, "configuration_"+operation)
	}
	if !reflect.DeepEqual(result, Result{}) {
		t.Fatal("failed edit exposed a partial policy or provenance")
	}
	if strings.Contains(err.Error(), editBoundaryMarker) || strings.Contains(err.Error(), fixture.path) || strings.Contains(err.Error(), fixture.include.BaseDirectory) {
		t.Fatal("edit refusal disclosed caller input or private synthetic paths")
	}
}

func editBoundaryEnvironmentEdits(t *testing.T, count int) ([]model.PolicyEdit, map[string]string) {
	t.Helper()
	edits := make([]model.PolicyEdit, 0, count)
	environment := map[string]string{"SYNTHETIC_BASE": "unchanged"}
	for index := 0; index < count; index++ {
		name := fmt.Sprintf("SYNTHETIC_%03d", index)
		value := fmt.Sprintf("value-%03d", index)
		edits = append(edits, model.PolicyEdit{Pointer: "/workloads/root/environment/" + name, Value: rawString(t, value)})
		environment[name] = value
	}
	return edits, environment
}
