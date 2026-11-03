package policy

import (
	"sort"
	"strings"

	"github.com/maldous/CircularQuay/internal/model"
)

var operations = map[string]bool{
	"list": true, "status": true, "run": true, "read": true, "cancel": true,
	"list_workloads": true, "get_status": true, "run_workload": true,
	"read_secret": true, "cancel_workload": true, "get_result": true,
}

func validateGrants(policy model.Policy) error {
	for _, grant := range policy.Grants {
		if !safePrincipal(grant.Subject) || grant.Client != "" && !safePrincipal(grant.Client) || len(grant.Operations) == 0 {
			return invalid("grant_identity")
		}
		workload, found := policy.Workloads[grant.Workload]
		if !found {
			return invalid("grant_workload")
		}
		if err := validateGrantOperations(grant); err != nil {
			return err
		}
		for _, key := range grant.Keys {
			if !bindingExists(workload, key) || forbiddenBindingName(strings.ToUpper(key)) {
				return invalid("grant_key")
			}
		}
	}
	return nil
}

func validateGrantOperations(grant model.Grant) error {
	seen := make(map[string]bool)
	for _, operation := range grant.Operations {
		if !operations[operation] || seen[operation] {
			return invalid("grant_operation")
		}
		if rawRead(operation) && len(grant.Keys) == 0 {
			return invalid("disclosure_scope")
		}
		seen[operation] = true
	}
	return nil
}

func safePrincipal(value string) bool {
	return value != "" && len(value) <= 1024 && safeText(value) && !strings.ContainsAny(value, "*?[]")
}

// Authorize requires an exact identity, client, workload and operation match.
// Empty client is a local-client binding, never a wildcard for hosted clients.
func Authorize(policy model.Policy, subject, client, workload, operation, key string) bool {
	if !safePrincipal(subject) || !operations[operation] {
		return false
	}
	for _, grant := range policy.Grants {
		if grant.Subject != subject || grant.Client != client || grant.Workload != workload {
			continue
		}
		if !contains(grant.Operations, operation) {
			continue
		}
		if !rawRead(operation) || key != "" && contains(grant.Keys, key) {
			return true
		}
	}
	return false
}

// VisibleWorkloads lists only workloads for which discovery was granted.
func VisibleWorkloads(policy model.Policy, subject, client string) []string {
	result := make([]string, 0)
	for name := range policy.Workloads {
		if Authorize(policy, subject, client, name, "list", "") || Authorize(policy, subject, client, name, "list_workloads", "") {
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

func rawRead(operation string) bool { return operation == "read" || operation == "read_secret" }

func bindingExists(workload model.Workload, key string) bool {
	for _, binding := range workload.Bindings {
		if binding.Name == key {
			return true
		}
	}
	return false
}
