package policy

import (
	"testing"

	"github.com/maldous/CircularQuay/internal/model"
)

func TestInfisicalPathContractAndSchema(t *testing.T) {
	for _, test := range []struct {
		name       string
		secretPath string
		valid      bool
	}{
		{name: "root", secretPath: "/", valid: true},
		{name: "application", secretPath: "/application", valid: true},
		{name: "nested_application", secretPath: "/application/production", valid: true},
		{name: "empty", secretPath: ""},
		{name: "relative", secretPath: "relative"},
		{name: "duplicate_separator", secretPath: "//duplicate"},
		{name: "current_directory", secretPath: "/./dot"},
		{name: "parent_directory", secretPath: "/parent/../escape"},
		{name: "trailing_separator", secretPath: "/trailing/"},
		{name: "query", secretPath: "/query?value"},
	} {
		t.Run(test.name, func(t *testing.T) {
			assertInfisicalPathContract(t, test.secretPath, test.valid)
		})
	}
}

func assertInfisicalPathContract(t *testing.T, secretPath string, wantValid bool) {
	t.Helper()
	policy := policyWithSource(infisicalPathSource(secretPath))
	semanticErr := Validate(policy)
	schemaErr := resolvedPolicySchema(t).Validate(policyJSONValue(t, policy))
	if wantValid {
		if semanticErr != nil {
			t.Fatalf("semantic policy rejected valid Infisical path %q: %v", secretPath, semanticErr)
		}
		if schemaErr != nil {
			t.Fatalf("schema rejected valid Infisical path %q: %v", secretPath, schemaErr)
		}
		return
	}
	if semanticErr == nil {
		t.Fatalf("semantic policy accepted ambiguous Infisical path %q", secretPath)
	}
	if schemaErr == nil {
		t.Fatalf("schema accepted ambiguous Infisical path %q", secretPath)
	}
}

func infisicalPathSource(secretPath string) model.Source {
	return model.Source{
		Type: "infisical", Project: "project", Environment: "test", Path: secretPath,
		Resources: []string{"KEY"}, Identity: "token_file", TokenFile: "/private/bootstrap",
	}
}
