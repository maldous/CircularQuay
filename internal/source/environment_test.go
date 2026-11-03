package source

import (
	"context"
	"strings"
	"testing"

	"github.com/maldous/CircularQuay/internal/fault"
	"github.com/maldous/CircularQuay/internal/model"
	"github.com/maldous/CircularQuay/internal/platform"
)

func TestEnvironmentRejectsProtectedInputsBeforeRead(t *testing.T) {
	for _, name := range []string{"vAuLt_ToKeN", "cQuAy_bootstrap", "ld_preload"} {
		t.Run(name, func(t *testing.T) {
			t.Setenv(name, "SYNTHETIC_PROTECTED_MARKER")
			assertProtectedEnvironmentInput(t, model.Source{Type: "env", Resources: []string{name}}, name)
			assertProtectedEnvironmentInput(t, model.Source{Type: "env", Path: privateLiteralFile(t, "absent.env"), Resources: []string{name}}, name)
		})
	}
}

func assertProtectedEnvironmentInput(t *testing.T, source model.Source, resource string) {
	t.Helper()
	_, err := readEnvironment(context.Background(), source, model.Binding{Resource: resource})
	if !fault.Is(err, fault.Invalid) || strings.Contains(err.Error(), "SYNTHETIC_PROTECTED_MARKER") {
		t.Fatalf("protected environment input was not rejected safely: %v", err)
	}
}

func TestEnvironmentFieldSelectsOnlyTopLevelString(t *testing.T) {
	const resource = "TEST_ENVIRONMENT_DOCUMENT"
	const selected = "SYNTHETIC_SELECTED_VALUE"
	document := `{"PRIMARY":"` + selected + `","SECONDARY":"SYNTHETIC_UNSELECTED_VALUE"}`
	binding := model.Binding{Resource: resource, Field: "PRIMARY"}
	t.Setenv(resource, document)

	registry := NewRegistry()
	source := model.Source{Type: "env", Resources: []string{resource}}
	value, err := registry.Read(context.Background(), source, binding)
	if err != nil || value.Data != selected {
		t.Fatalf("environment field selection = %#v, %v", value, err)
	}
	if strings.Contains(value.Data, "UNSELECTED") {
		t.Fatal("environment field selection returned a sibling value")
	}

	file := privateLiteralFile(t, "selected.env")
	if err := platform.WriteProtected(file, []byte(resource+"="+document+"\n")); err != nil {
		t.Fatal(err)
	}
	source.Path = file
	value, err = registry.Read(context.Background(), source, binding)
	if err != nil || value.Data != selected {
		t.Fatalf("dotenv field selection = %#v, %v", value, err)
	}
}

func TestEnvironmentFieldRejectsTrailingJSON(t *testing.T) {
	const resource = "TEST_ENVIRONMENT_TRAILING"
	binding := model.Binding{Resource: resource, Field: "PRIMARY"}
	registry := NewRegistry()
	source := model.Source{Type: "env", Resources: []string{resource}}
	t.Setenv(resource, `{"PRIMARY":"SYNTHETIC"} trailing`)
	if _, err := registry.Read(context.Background(), source, binding); !fault.Is(err, fault.Trust) {
		t.Fatalf("environment trailing JSON error = %v", err)
	}

	file := privateLiteralFile(t, "trailing.env")
	if err := platform.WriteProtected(file, []byte(resource+`={"PRIMARY":"SYNTHETIC"} trailing`+"\n")); err != nil {
		t.Fatal(err)
	}
	source.Path = file
	if _, err := registry.Read(context.Background(), source, binding); !fault.Is(err, fault.Trust) {
		t.Fatalf("dotenv trailing JSON error = %v", err)
	}
}
