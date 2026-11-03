package policy

import (
	"sort"
	"strings"
	"time"

	"github.com/maldous/CircularQuay/internal/model"
)

var operations = map[string]bool{
	"list": true, "status": true, "run": true, "read": true,
}

func validateGrants(policy model.Policy) error {
	for _, grant := range policy.Grants {
		if err := validateGrant(policy, grant); err != nil {
			return err
		}
	}
	return nil
}

func validateGrant(policy model.Policy, grant model.Grant) error {
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
	if !validGrantWindow(grant) {
		return invalid("grant_window")
	}
	return nil
}

func validGrantWindow(grant model.Grant) bool {
	if grant.NotBefore != nil && grant.NotBefore.IsZero() {
		return false
	}
	if grant.ExpiresAt != nil && grant.ExpiresAt.IsZero() {
		return false
	}
	return grant.NotBefore == nil || grant.ExpiresAt == nil || grant.NotBefore.Before(*grant.ExpiresAt)
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

// Authorize requires an exact identity, client, workload and operation match
// at the current local time. Remote identity claims remain independently
// validated by their caller before this local policy decision.
func Authorize(policy model.Policy, subject, client, workload, operation, key string) bool {
	return AuthorizeAt(policy, subject, client, workload, operation, key, time.Now())
}

// AuthorizeAt makes the same exact local-policy decision at now. Grant starts
// are inclusive and expiry is exclusive, preventing an expired local grant
// from widening an already validated remote identity or credential grant.
func AuthorizeAt(policy model.Policy, subject, client, workload, operation, key string, now time.Time) bool {
	if now.IsZero() || !safePrincipal(subject) || !operations[operation] {
		return false
	}
	for _, grant := range policy.Grants {
		if matchingGrant(grant, subject, client, workload, operation, key, now) {
			return true
		}
	}
	return false
}

func matchingGrant(grant model.Grant, subject, client, workload, operation, key string, now time.Time) bool {
	if grant.Subject != subject || grant.Client != client || grant.Workload != workload || !contains(grant.Operations, operation) {
		return false
	}
	return grantWindowAllows(grant, now) && (!rawRead(operation) || key != "" && contains(grant.Keys, key))
}

func grantWindowAllows(grant model.Grant, now time.Time) bool {
	if !validGrantWindow(grant) {
		return false
	}
	if grant.NotBefore != nil && now.Before(*grant.NotBefore) {
		return false
	}
	return grant.ExpiresAt == nil || now.Before(*grant.ExpiresAt)
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
