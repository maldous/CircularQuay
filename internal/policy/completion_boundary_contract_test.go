package policy

import (
	"encoding/json"
	"errors"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/maldous/CircularQuay/internal/fault"
	"github.com/maldous/CircularQuay/internal/model"
)

func TestCompletionPolicySecondLocationFailurePreservesZeroPolicy(t *testing.T) {
	candidate := declarationPolicy(t)
	data, err := json.Marshal(candidate)
	if err != nil {
		t.Fatal(err)
	}
	name := filepath.Join(t.TempDir(), "policy.json")
	if _, err := Parse(data, name); err != nil {
		t.Fatal("real parse positive control failed", err)
	}
	calls := 0
	relative := "synthetic-policy.json"
	result, err := parseWithAbsolute(data, relative, func(got string) (string, error) {
		calls++
		if got != relative {
			t.Fatal("location input changed")
		}
		return "", errors.New("synthetic absolute location failure")
	})
	if calls != 1 || !reflect.DeepEqual(result, model.Policy{}) || !fault.Is(err, fault.Invalid) || err.Error() != "invalid_policy: policy_location" {
		t.Fatal("location failure changed admission result")
	}
}

func TestCompletionPolicyGuardDomainAdmission(t *testing.T) {
	cases := []struct {
		name string
		op   string
		edit func(*model.ConfigurationGuard)
	}{
		{"missing_source", "configuration_domains", func(g *model.ConfigurationGuard) { delete(g.SourceDomains, "fixture") }},
		{"invalid_domain", "configuration_domains", func(g *model.ConfigurationGuard) { g.SourceDomains["fixture"] = "bad domain" }},
		{"unmatched_local", "configuration_domains", func(g *model.ConfigurationGuard) { g.LocalSources = []string{"fixture"} }},
		{"unknown_local", "configuration_local_domain", func(g *model.ConfigurationGuard) {
			g.LocalSources = []string{"missing"}
			g.LocalRoots = map[string]string{"missing": ""}
		}},
		{"empty_root", "configuration_local_domain", func(g *model.ConfigurationGuard) {
			g.LocalSources = []string{"fixture"}
			g.LocalRoots = map[string]string{"fixture": ""}
		}},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			candidate, _ := configurationOwnedPolicy(t)
			test.edit(candidate.ConfigurationGuard)
			assertCompletionGuardPolicy(t, candidate, test.op)
		})
	}
}

func TestCompletionPolicyNamedClaimFiltersRejectWildcard(t *testing.T) {
	for _, field := range []string{"clients", "subjects", "roles"} {
		t.Run(field, func(t *testing.T) {
			candidate := namedIdentityPolicy()
			candidate.Grants = nil
			authority := candidate.MCP.Authorities["workforce"]
			switch field {
			case "clients":
				authority.Clients = []string{"client*"}
			case "subjects":
				authority.Subjects = []string{"subject?"}
			case "roles":
				authority.RequiredRoles = []string{"[Operator]"}
			}
			candidate.MCP.Authorities["workforce"] = authority
			// The verifier permits literal wildcard text; policy's narrower
			// claim-filter declaration must reject it without loading keys.
			assertDeclarationPolicy(t, candidate, "mcp_named_authority")
			assertDeclarationError(t, validateIdentityAuthority("workforce", authority), "identity_claim_filter")
		})
	}
}

func TestCompletionPolicyPrivateSnapshotCannotLeaveOwnedDomain(t *testing.T) {
	candidate := declarationPolicy(t)
	root := filepath.Dir(candidate.Sources["fixture"].Path)
	candidate.ConfigurationGuard = &model.ConfigurationGuard{
		SourceDomains: map[string]string{"fixture": "private"},
		LocalSources:  []string{"fixture"},
		LocalRoots:    map[string]string{"fixture": root},
	}
	changeWorkload(&candidate, func(w *model.Workload) { setSnapshot(w, filepath.Join(root, "snapshot")) })
	assertDeclarationPolicy(t, candidate, "")
	outside := filepath.Join(t.TempDir(), "snapshot")
	changeWorkload(&candidate, func(w *model.Workload) { setSnapshot(w, outside) })
	assertCompletionGuardPolicy(t, candidate, "configuration_private_snapshot")
}

func assertCompletionGuardPolicy(t *testing.T, candidate model.Policy, operation string) {
	t.Helper()
	before, err := json.Marshal(candidate.ConfigurationGuard)
	if err != nil {
		t.Fatal(err)
	}
	assertDeclarationPolicy(t, candidate, operation)
	after, err := json.Marshal(candidate.ConfigurationGuard)
	if err != nil || string(before) != string(after) {
		t.Fatal("domain validation mutated the resolver-only guard")
	}
}
