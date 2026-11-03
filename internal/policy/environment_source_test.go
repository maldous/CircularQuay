package policy

import (
	"testing"

	"github.com/maldous/CircularQuay/internal/fault"
	"github.com/maldous/CircularQuay/internal/model"
)

func TestEnvironmentSourcesRejectProtectedInputAliases(t *testing.T) {
	for _, resource := range []string{"VAULT_TOKEN", "vault_token", "Cquay_Bootstrap", "Ld_Preload"} {
		t.Run(resource, func(t *testing.T) {
			policy := validPolicy()
			policy.Sources["fixture"] = model.Source{Type: "env", Resources: []string{resource}}
			changeWorkload(&policy, func(workload *model.Workload) {
				workload.Bindings[0].Resource = resource
				workload.Bindings[0].Field = ""
			})
			if err := Validate(policy); !fault.Is(err, fault.Invalid) {
				t.Fatalf("protected environment resource accepted: %v", err)
			}
		})
	}
}

func TestProtectedEnvironmentNamesAreCaseInsensitive(t *testing.T) {
	for _, name := range []string{"vault_token", "cQuAy_bootstrap", "lD_preload", "Path"} {
		if !IsProtectedEnvironmentName(name) {
			t.Fatalf("protected environment name %q was accepted", name)
		}
	}
}
