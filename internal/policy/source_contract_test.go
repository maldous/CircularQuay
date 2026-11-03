package policy

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/maldous/CircularQuay/internal/model"
)

func TestSourceContractsRejectUnsupportedIdentityAndFields(t *testing.T) {
	for name, source := range map[string]model.Source{
		"json ghost identity":        {Type: "json", Path: "/values.json", Identity: "token_file", TokenFile: "/bootstrap"},
		"vault missing bootstrap":    {Type: "vault", Endpoint: "https://vault.example", Resources: []string{"app"}, Mount: "kv"},
		"aws unsupported identity":   {Type: "aws-sm", Account: "123456789012", Region: "ap-southeast-2", Resources: []string{"arn:aws:secretsmanager:ap-southeast-2:123456789012:secret:app"}, Identity: "managed"},
		"azure missing client":       {Type: "azure", Endpoint: "https://vault.example", Account: "tenant", Resources: []string{"secret"}, Identity: "managed"},
		"gcp ambient identity":       {Type: "gcp", Account: "service@example", Project: "project", Resources: []string{"projects/project/secrets/value"}, Identity: "none"},
		"vendor relative executable": {Type: "onepassword", Path: "op", Project: "vault", Resources: []string{"op://vault/item/field"}, Identity: "token_file", TokenFile: "/bootstrap"},
	} {
		t.Run(name, func(t *testing.T) {
			if err := Validate(policyWithSource(source)); err == nil {
				t.Fatal("unsupported source declaration accepted")
			}
		})
	}
}

func TestSourceContractsAcceptSupportedConfigurations(t *testing.T) {
	native := filepath.Join(t.TempDir(), "native")
	for name, source := range map[string]model.Source{
		"literal json":  {Type: "json", Path: filepath.Join(native, "values.json")},
		"process env":   {Type: "env", Resources: []string{"VALUE"}},
		"age sops":      {Type: "sops", Path: filepath.Join(native, "values.sops.json"), SOPS: filepath.Join(native, "sops"), AgeIdentityFile: filepath.Join(native, "age"), Identity: "none"},
		"vault":         {Type: "vault", Endpoint: "https://vault.example", Mount: "kv", Resources: []string{"app"}, Identity: "token_file", TokenFile: filepath.Join(native, "bootstrap")},
		"azure managed": {Type: "azure", Endpoint: "https://vault.example", Account: "tenant", Profile: "client", Resources: []string{"secret"}, Identity: "managed"},
		"gcp metadata":  {Type: "gcp", Account: "service@example", Project: "project", Resources: []string{"projects/project/secrets/value"}, Identity: "metadata"},
	} {
		t.Run(name, func(t *testing.T) {
			if err := Validate(policyWithSource(source)); err != nil {
				t.Fatalf("supported source declaration rejected: %v", err)
			}
		})
	}
}

func TestSourceContractsRejectControlText(t *testing.T) {
	source := model.Source{Type: "aws-sm", Account: "123\n456", Region: "ap-southeast-2", Resources: []string{"arn:aws:secretsmanager:ap-southeast-2:123456789012:secret:app"}, Identity: "ec2"}
	err := Validate(policyWithSource(source))
	if err == nil || strings.Contains(err.Error(), "123") {
		t.Fatalf("unsafe source text error = %v", err)
	}
}

func policyWithSource(source model.Source) model.Policy {
	resource := ""
	if !localFileSource(source.Type) {
		resource = source.Resources[0]
	}
	return model.Policy{
		Version: 1,
		Sources: map[string]model.Source{"source": source},
		Workloads: map[string]model.Workload{"workload": {
			Mode: "hybrid", Bindings: []model.Binding{{Name: "VALUE", Source: "source", Resource: resource, AllowOffline: true}},
		}},
	}
}
