package policy

import (
	"sort"
	"strings"

	"github.com/maldous/CircularQuay/internal/model"
)

var operations = map[string]bool{
	"list": true, "status": true, "run": true, "read": true,
}

func validateGrants(policy model.Policy) error {
	for _, grant := range policy.Grants {
		if !validGrantIdentity(grant) {
			return invalid("grant_identity")
		}
		workload, found := policy.Workloads[grant.Workload]
		if !found {
			return invalid("grant_workload")
		}
		if err := validateGrantOperations(grant); err != nil {
			return err
		}
		if err := validateGrantKeys(workload, grant.Keys); err != nil {
			return err
		}
	}
	return nil
}

func validGrantIdentity(grant model.Grant) bool {
	return safePrincipal(grant.Subject) && (grant.Client == "" || safePrincipal(grant.Client)) && len(grant.Operations) > 0
}

func validateGrantKeys(workload model.Workload, keys []string) error {
	seen := make(map[string]bool)
	for _, key := range keys {
		if seen[key] || !bindingExists(workload, key) || forbiddenBindingName(strings.ToUpper(key)) {
			return invalid("grant_key")
		}
		seen[key] = true
	}
	return nil
}

func validateGrantOperations(grant model.Grant) error {
	seen := make(map[string]bool)
	hasRead := false
	for _, operation := range grant.Operations {
		if !operations[operation] || seen[operation] {
			return invalid("grant_operation")
		}
		hasRead = hasRead || rawRead(operation)
		seen[operation] = true
	}
	if hasRead && len(grant.Keys) == 0 {
		return invalid("disclosure_scope")
	}
	if !hasRead && len(grant.Keys) != 0 {
		return invalid("grant_key_scope")
	}
	return nil
}

func safePrincipal(value string) bool {
	return value != "" && len(value) <= 1024 && safeText(value) && !strings.ContainsAny(value, "*?[]")
}

// Authorize requires an exact identity, client, workload and operation match.
// A run grant authorizes the complete enrolled workload; it cannot restrict
// resolution to selected bindings. Keys apply only to read in a mixed read/run
// grant. Empty client is a local-client binding, never a hosted-client wildcard.
func Authorize(policy model.Policy, subject, client, workload, operation, key string) bool {
	if !safePrincipal(subject) || !operations[operation] {
		return false
	}
	for _, grant := range policy.Grants {
		if matchingGrant(grant, subject, client, workload, operation, key) {
			return true
		}
	}
	return false
}

func matchingGrant(grant model.Grant, subject, client, workload, operation, key string) bool {
	if grant.Subject != subject || grant.Client != client || grant.Workload != workload || !contains(grant.Operations, operation) {
		return false
	}
	return !rawRead(operation) || key != "" && contains(grant.Keys, key)
}

// VisibleWorkloads lists only workloads for which discovery was granted.
func VisibleWorkloads(policy model.Policy, subject, client string) []string {
	result := make([]string, 0)
	for name := range policy.Workloads {
		if Authorize(policy, subject, client, name, "list", "") {
			result = append(result, name)
		}
	}
	sort.Strings(result)
	return result
}

func contains(values []string, selected string) bool {
	for _, value := range values {
		if value == selected {
			return true
		}
	}
	return false
}

func rawRead(operation string) bool { return operation == "read" }

func bindingExists(workload model.Workload, key string) bool {
	for _, binding := range workload.Bindings {
		if binding.Name == key {
			return true
		}
	}
	return false
}
