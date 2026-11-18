package source

import (
	"context"
	"reflect"
	"testing"

	"github.com/maldous/CircularQuay/internal/model"
)

func TestAmbientProjectionBoundaryContractSOPSAllowlistIsExplicitAndOrdered(t *testing.T) {
	allowed := []string{"SystemRoot", "WINDIR", "HTTP_PROXY", "HTTPS_PROXY", "NO_PROXY", "SSL_CERT_FILE", "SSL_CERT_DIR"}
	visited := []string{}
	lookup := func(name string) string {
		visited = append(visited, name)
		return "SYNTHETIC_" + name
	}
	environment, err := sopsEnvironmentWithLookup("owned-home", "", "", nil, lookup)
	if err != nil || !reflect.DeepEqual(visited, allowed) {
		t.Fatal("environment projection read undeclared names or changed order")
	}
	expected := []string{"HOME=owned-home", "USERPROFILE=owned-home", "SOPS_HC_VAULT_ALLOWLIST=none"}
	for _, name := range allowed {
		expected = append(expected, name+"=SYNTHETIC_"+name)
	}
	if !reflect.DeepEqual(environment, expected) {
		t.Fatal("environment projection changed fixed custody values or retained extra input")
	}
}

func TestAmbientProjectionBoundaryContractSOPSExplicitFieldOverridesBindingName(t *testing.T) {
	config := sopsOrchestrationSource(t)
	run := func(context.Context, string, []string, []byte, string, string, []string) ([]byte, error) {
		return []byte(`{"KEY":"SELECTED","OTHER":"NOT_SELECTED"}`), nil
	}
	value, err := readSOPSWithRunner(context.Background(), config, model.Binding{Name: "OTHER", Field: "KEY"}, run)
	if err != nil || value.Data != "SELECTED" {
		t.Fatal("explicit field did not override the binding name")
	}
}
