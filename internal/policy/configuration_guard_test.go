package policy

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/maldous/CircularQuay/internal/fault"
	"github.com/maldous/CircularQuay/internal/model"
)

func TestConfigurationGuardLeavesOrdinarySingleFilePolicyValid(t *testing.T) {
	if err := Validate(validPolicy()); err != nil {
		t.Fatalf("ordinary single-file policy rejected: %v", err)
	}
}

func TestConfigurationGuardRejectsUnresolvedComposition(t *testing.T) {
	policy := validPolicy()
	policy.Composition = &model.Composition{}
	requireConfigurationGuardReject(t, policy, "policy_structure")
}

func TestConfigurationGuardRetainsPrivateSourceBoundaries(t *testing.T) {
	tests := []struct {
		name      string
		operation string
		edit      func(*model.Policy, guardedConfigurationPaths)
	}{
		{
			name:      "root-owned source cannot escape its base",
			operation: "configuration_private_reference",
			edit: func(policy *model.Policy, paths guardedConfigurationPaths) {
				source := policy.Sources["corporate"]
				source.Path = filepath.Join(paths.base, "outside", "values.json")
				policy.Sources["corporate"] = source
			},
		},
		{
			name:      "corporate source cannot alias personal file",
			operation: "configuration_private_reference",
			edit: func(policy *model.Policy, paths guardedConfigurationPaths) {
				source := policy.Sources["corporate"]
				source.Path = filepath.Join(paths.personalRoot, "values.json")
				policy.Sources["corporate"] = source
			},
		},
		{
			name:      "source value cannot use read-only configuration root",
			operation: "configuration_source_cache",
			edit: func(policy *model.Policy, paths guardedConfigurationPaths) {
				source := policy.Sources["corporate"]
				source.Path = filepath.Join(paths.readOnlyRoot, "included-values.json")
				policy.Sources["corporate"] = source
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			policy, paths := guardedPrivatePolicy(t)
			test.edit(&policy, paths)
			requireConfigurationGuardReject(t, policy, test.operation)
		})
	}
}

func TestConfigurationGuardRejectsCrossDomainCredentials(t *testing.T) {
	tests := []struct {
		name   string
		policy func(*testing.T) model.Policy
	}{
		{
			name: "token",
			policy: func(t *testing.T) model.Policy {
				policy, shared := guardedTokenProviderPolicy(t)
				for _, name := range []string{"corporate", "personal"} {
					source := policy.Sources[name]
					source.TokenFile = shared
					policy.Sources[name] = source
				}
				return policy
			},
		},
		{
			name: "age identity",
			policy: func(t *testing.T) model.Policy {
				policy, shared := guardedAgeProviderPolicy(t)
				for _, name := range []string{"corporate", "personal"} {
					source := policy.Sources[name]
					source.AgeIdentityFile = shared
					policy.Sources[name] = source
				}
				return policy
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			requireConfigurationGuardReject(t, test.policy(t), "configuration_shared_credential")
		})
	}
}

func TestConfigurationGuardRejectsConfigurationOwnedProviderCredentials(t *testing.T) {
	tests := []struct {
		name   string
		policy func(*testing.T) model.Policy
	}{
		{
			name: "token file under a read-only repository root",
			policy: func(t *testing.T) model.Policy {
				policy, _ := guardedTokenProviderPolicy(t)
				cache := filepath.Join(t.TempDir(), "repository-cache")
				policy.ConfigurationGuard.ReadOnlyRoots = []string{cache}
				source := policy.Sources["corporate"]
				source.TokenFile = filepath.Join(cache, "token")
				policy.Sources["corporate"] = source
				return policy
			},
		},
		{
			name: "age identity equal to a resolved fragment input",
			policy: func(t *testing.T) model.Policy {
				policy, _ := guardedAgeProviderPolicy(t)
				input := filepath.Join(t.TempDir(), "fragment.json")
				policy.ConfigurationGuard.Inputs = []string{input}
				source := policy.Sources["personal"]
				source.AgeIdentityFile = input
				policy.Sources["personal"] = source
				return policy
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			requireConfigurationGuardReject(t, test.policy(t), "configuration_credential_cache")
		})
	}
}

func TestConfigurationGuardDeniesPrivateWorkloadEgress(t *testing.T) {
	tests := []struct {
		name      string
		operation string
		policy    func(*testing.T) model.Policy
	}{
		{
			name:      "allow publish",
			operation: "configuration_private_publish",
			policy: func(t *testing.T) model.Policy {
				policy, _ := guardedPrivatePolicy(t)
				changeWorkload(&policy, func(workload *model.Workload) {
					workload.Bindings[0].AllowPublish = true
				})
				return policy
			},
		},
		{
			name:      "native target",
			operation: "configuration_private_publish",
			policy: func(t *testing.T) model.Policy {
				policy, paths := guardedPrivatePolicy(t)
				setFileTarget(&policy, "sink", filepath.Join(paths.personalRoot, "export.json"))
				return policy
			},
		},
		{
			name:      "named remote authority grant",
			operation: "configuration_private_remote_grant",
			policy: func(t *testing.T) model.Policy {
				policy := namedIdentityPolicy()
				base := t.TempDir()
				root := filepath.Join(base, "private")
				source := policy.Sources["fixture"]
				source.Path = filepath.Join(root, "values.json")
				policy.Sources["fixture"] = source
				policy.ConfigurationGuard = &model.ConfigurationGuard{
					LocalSources:  []string{"fixture"},
					SourceDomains: map[string]string{"fixture": "private"},
					LocalRoots:    map[string]string{"fixture": root},
				}
				return policy
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			requireConfigurationGuardReject(t, test.policy(t), test.operation)
		})
	}
}

func TestConfigurationGuardSnapshotDomainBoundaries(t *testing.T) {
	t.Run("mixed domains cannot snapshot", func(t *testing.T) {
		policy, paths := guardedPrivatePolicy(t)
		changeWorkload(&policy, func(workload *model.Workload) {
			workload.Bindings = append(workload.Bindings, model.Binding{
				Name: "CORPORATE_MESSAGE", Source: "corporate", Field: "message", AllowOffline: true,
			})
			setSnapshot(workload, filepath.Join(paths.personalRoot, "snapshot.json"))
		})
		requireConfigurationGuardReject(t, policy, "configuration_mixed_snapshot")
	})

	t.Run("private snapshot remains inside its source root", func(t *testing.T) {
		policy, paths := guardedPrivatePolicy(t)
		changeWorkload(&policy, func(workload *model.Workload) {
			setSnapshot(workload, filepath.Join(paths.personalRoot, "snapshot.json"))
		})
		if err := Validate(policy); err != nil {
			t.Fatalf("bounded private snapshot rejected: %v", err)
		}
	})

	t.Run("private snapshot cannot escape its source root", func(t *testing.T) {
		policy, paths := guardedPrivatePolicy(t)
		changeWorkload(&policy, func(workload *model.Workload) {
			setSnapshot(workload, filepath.Join(paths.corporateRoot, "snapshot.json"))
		})
		requireConfigurationGuardReject(t, policy, "configuration_private_snapshot")
	})

	t.Run("corporate snapshot identity cannot alias personal root", func(t *testing.T) {
		policy, paths := corporateSharedPolicy(t)
		changeWorkload(&policy, func(workload *model.Workload) {
			workload.Bindings[0].Source = "corporate"
			setSnapshot(workload, filepath.Join(paths.corporateRoot, "snapshot.json"))
			workload.Snapshot.IdentityFile = filepath.Join(paths.personalRoot, "snapshot.age")
		})
		requireConfigurationGuardReject(t, policy, "configuration_private_reference")
	})

	t.Run("corporate target cannot alias personal root", func(t *testing.T) {
		policy, paths := corporateSharedPolicy(t)
		changeWorkload(&policy, func(workload *model.Workload) {
			workload.Bindings[0].Source = "corporate"
		})
		setFileTarget(&policy, "sink", filepath.Join(paths.personalRoot, "export.json"))
		requireConfigurationGuardReject(t, policy, "configuration_private_reference")
	})
}

func TestConfigurationGuardRejectsForeignPrivateOperationalPaths(t *testing.T) {
	tests := []struct {
		name   string
		policy func(*testing.T) model.Policy
	}{
		{
			name: "corporate workload directory",
			policy: func(t *testing.T) model.Policy {
				policy, paths := corporateSharedPolicy(t)
				changeWorkload(&policy, func(workload *model.Workload) {
					workload.Bindings[0].Source = "corporate"
					workload.Directory = filepath.Join(paths.personalRoot, "work")
				})
				return policy
			},
		},
		{
			name: "corporate workload command",
			policy: func(t *testing.T) model.Policy {
				policy, paths := corporateSharedPolicy(t)
				changeWorkload(&policy, func(workload *model.Workload) {
					workload.Bindings[0].Source = "corporate"
					workload.Command = []string{filepath.Join(paths.personalRoot, "worker")}
					workload.CommandSHA256 = strings.Repeat("a", 64)
				})
				return policy
			},
		},
		{
			name: "corporate snapshot sops executable",
			policy: func(t *testing.T) model.Policy {
				policy, paths := corporateSharedPolicy(t)
				changeWorkload(&policy, func(workload *model.Workload) {
					workload.Bindings[0].Source = "corporate"
					setSnapshot(workload, filepath.Join(paths.corporateRoot, "snapshot.json"))
					workload.Snapshot.SOPS = filepath.Join(paths.personalRoot, "sops")
				})
				return policy
			},
		},
		{
			name: "corporate source sops executable",
			policy: func(t *testing.T) model.Policy {
				policy, paths := corporateSharedSOPSPolicy(t)
				source := policy.Sources["corporate"]
				source.SOPS = filepath.Join(paths.personalRoot, "sops")
				policy.Sources["corporate"] = source
				return policy
			},
		},
		{
			name: "corporate vendor executable",
			policy: func(t *testing.T) model.Policy {
				policy, paths := corporateSharedVendorPolicy(t)
				source := policy.Sources["corporate"]
				source.Path = filepath.Join(paths.personalRoot, "bw")
				policy.Sources["corporate"] = source
				return policy
			},
		},
		{
			name: "repository directory",
			policy: func(t *testing.T) model.Policy {
				policy, paths := corporateSharedPolicy(t)
				policy.Repository = testRepository(paths.base)
				requireValidConfigurationFixture(t, policy)
				policy.Repository.Directory = filepath.Join(paths.personalRoot, "repository")
				return policy
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			requireConfigurationGuardReject(t, test.policy(t), "configuration_private_reference")
		})
	}
}

func TestConfigurationGuardProtectsConfigurationAndRepositoryCredentials(t *testing.T) {
	tests := []struct {
		name      string
		operation string
		policy    func(*testing.T) model.Policy
	}{
		{
			name:      "configuration input cannot become a source value",
			operation: "bootstrap_source_collision",
			policy: func(t *testing.T) model.Policy {
				policy, input := configurationOwnedPolicy(t)
				source := policy.Sources["fixture"]
				source.Path = input
				policy.Sources["fixture"] = source
				return policy
			},
		},
		{
			name:      "configuration input cannot become a writer",
			operation: "path_ownership",
			policy: func(t *testing.T) model.Policy {
				policy, input := configurationOwnedPolicy(t)
				setFileTarget(&policy, "sink", input)
				return policy
			},
		},
		{
			name:      "configuration input cannot become a trust store",
			operation: "identity_trust_store_path",
			policy: func(t *testing.T) model.Policy {
				policy, input := configurationOwnedNamedIdentityPolicy(t)
				setAuthorityTrustStore(&policy, input)
				return policy
			},
		},
		{
			name:      "repository credential cannot become a source value",
			operation: "bootstrap_source_collision",
			policy: func(t *testing.T) model.Policy {
				policy, credential := repositoryCredentialPolicy(t, false)
				source := policy.Sources["fixture"]
				source.Path = credential
				policy.Sources["fixture"] = source
				return policy
			},
		},
		{
			name:      "repository credential cannot become a writer",
			operation: "path_ownership",
			policy: func(t *testing.T) model.Policy {
				policy, credential := repositoryCredentialPolicy(t, false)
				setFileTarget(&policy, "sink", credential)
				return policy
			},
		},
		{
			name:      "repository credential cannot become a trust store",
			operation: "trust_store_credential_path",
			policy: func(t *testing.T) model.Policy {
				policy, credential := repositoryCredentialPolicy(t, true)
				setAuthorityTrustStore(&policy, credential)
				return policy
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			requireConfigurationGuardReject(t, test.policy(t), test.operation)
		})
	}
}

type guardedConfigurationPaths struct {
	base, corporateRoot, personalRoot, readOnlyRoot string
}

func guardedPrivatePolicy(t *testing.T) (model.Policy, guardedConfigurationPaths) {
	t.Helper()
	base := t.TempDir()
	paths := guardedConfigurationPaths{
		base:          base,
		corporateRoot: filepath.Join(base, "corporate"),
		personalRoot:  filepath.Join(base, "personal"),
		readOnlyRoot:  filepath.Join(base, "configuration"),
	}
	policy := model.Policy{
		Version: 1,
		Sources: map[string]model.Source{
			"corporate": {Type: "json", Path: filepath.Join(paths.corporateRoot, "values.json")},
			"personal":  {Type: "json", Path: filepath.Join(paths.personalRoot, "values.json")},
		},
		Workloads: map[string]model.Workload{"demo": {
			Mode:     "offline",
			Bindings: []model.Binding{{Name: "DEMO_MESSAGE", Source: "personal", Field: "message", AllowOffline: true}},
		}},
		ConfigurationGuard: &model.ConfigurationGuard{
			Inputs:        []string{filepath.Join(paths.readOnlyRoot, "master.json")},
			ReadOnlyRoots: []string{paths.readOnlyRoot},
			LocalSources:  []string{"corporate", "personal"},
			SourceDomains: map[string]string{"corporate": "corporate", "personal": "personal"},
			LocalRoots:    map[string]string{"corporate": paths.corporateRoot, "personal": paths.personalRoot},
		},
	}
	requireValidConfigurationFixture(t, policy)
	return policy, paths
}

func guardedTokenProviderPolicy(t *testing.T) (model.Policy, string) {
	t.Helper()
	base := t.TempDir()
	policy := model.Policy{
		Version: 1,
		Sources: map[string]model.Source{
			"corporate": openBAOTokenSource(filepath.Join(base, "corporate", "token")),
			"personal":  openBAOTokenSource(filepath.Join(base, "personal", "token")),
		},
		Workloads: map[string]model.Workload{"demo": {
			Mode:     "online",
			Bindings: []model.Binding{{Name: "DEMO_MESSAGE", Source: "corporate", Resource: "app", Field: "message"}},
		}},
		ConfigurationGuard: &model.ConfigurationGuard{
			SourceDomains: map[string]string{"corporate": "corporate", "personal": "personal"},
		},
	}
	requireValidConfigurationFixture(t, policy)
	return policy, filepath.Join(base, "shared", "token")
}

func corporateSharedPolicy(t *testing.T) (model.Policy, guardedConfigurationPaths) {
	t.Helper()
	policy, paths := guardedPrivatePolicy(t)
	policy.ConfigurationGuard.LocalSources = []string{"personal"}
	delete(policy.ConfigurationGuard.LocalRoots, "corporate")
	requireValidConfigurationFixture(t, policy)
	return policy, paths
}

func corporateSharedSOPSPolicy(t *testing.T) (model.Policy, guardedConfigurationPaths) {
	t.Helper()
	policy, paths := corporateSharedPolicy(t)
	policy.Sources["corporate"] = sopsAgeSource(paths.corporateRoot)
	changeWorkload(&policy, func(workload *model.Workload) {
		workload.Bindings[0].Source = "corporate"
	})
	requireValidConfigurationFixture(t, policy)
	return policy, paths
}

func corporateSharedVendorPolicy(t *testing.T) (model.Policy, guardedConfigurationPaths) {
	t.Helper()
	policy, paths := corporateSharedPolicy(t)
	policy.Sources["corporate"] = model.Source{
		Type: "bitwarden", Path: filepath.Join(paths.corporateRoot, "bw"), Project: "vault",
		Resources: []string{"item"}, Identity: "token_file", TokenFile: filepath.Join(paths.corporateRoot, "token"),
	}
	changeWorkload(&policy, func(workload *model.Workload) {
		workload.Mode = "online"
		workload.Bindings[0] = model.Binding{Name: "DEMO_MESSAGE", Source: "corporate", Resource: "item", Field: "value"}
	})
	requireValidConfigurationFixture(t, policy)
	return policy, paths
}

func guardedAgeProviderPolicy(t *testing.T) (model.Policy, string) {
	t.Helper()
	base := t.TempDir()
	policy := model.Policy{
		Version: 1,
		Sources: map[string]model.Source{
			"corporate": sopsAgeSource(filepath.Join(base, "corporate")),
			"personal":  sopsAgeSource(filepath.Join(base, "personal")),
		},
		Workloads: map[string]model.Workload{"demo": {
			Mode:     "offline",
			Bindings: []model.Binding{{Name: "DEMO_MESSAGE", Source: "personal", Field: "message", AllowOffline: true}},
		}},
		ConfigurationGuard: &model.ConfigurationGuard{
			SourceDomains: map[string]string{"corporate": "corporate", "personal": "personal"},
		},
	}
	requireValidConfigurationFixture(t, policy)
	return policy, filepath.Join(base, "shared", "age-identity")
}

func configurationOwnedPolicy(t *testing.T) (model.Policy, string) {
	t.Helper()
	base := t.TempDir()
	input := filepath.Join(base, "configuration", "included.json")
	policy := validPolicy()
	policy.Sources["fixture"] = model.Source{Type: "json", Path: filepath.Join(base, "values.json")}
	policy.ConfigurationGuard = &model.ConfigurationGuard{
		Inputs:        []string{input},
		ReadOnlyRoots: []string{filepath.Join(base, "configuration")},
		SourceDomains: map[string]string{"fixture": "corporate"},
	}
	requireValidConfigurationFixture(t, policy)
	return policy, input
}

func configurationOwnedNamedIdentityPolicy(t *testing.T) (model.Policy, string) {
	t.Helper()
	policy, input := configurationOwnedPolicy(t)
	named := namedIdentityPolicy()
	named.Sources = policy.Sources
	named.ConfigurationGuard = policy.ConfigurationGuard
	requireValidConfigurationFixture(t, named)
	return named, input
}

func repositoryCredentialPolicy(t *testing.T, named bool) (model.Policy, string) {
	t.Helper()
	base := t.TempDir()
	credential := filepath.Join(base, "credential", "repository-token")
	var policy model.Policy
	if named {
		policy = namedIdentityPolicy()
	} else {
		policy = validPolicy()
	}
	policy.Repository = testRepository(base)
	policy.Repository.TokenFile = credential
	policy.Sources["fixture"] = model.Source{Type: "json", Path: filepath.Join(base, "values.json")}
	requireValidConfigurationFixture(t, policy)
	return policy, credential
}

func openBAOTokenSource(token string) model.Source {
	return model.Source{
		Type: "openbao", Endpoint: "https://bao.example.test", Mount: "kv", Resources: []string{"app"},
		Identity: "token_file", TokenFile: token,
	}
}

func sopsAgeSource(root string) model.Source {
	return model.Source{
		Type: "sops", Path: filepath.Join(root, "values.sops.json"), SOPS: filepath.Join(root, "sops"),
		AgeIdentityFile: filepath.Join(root, "age-identity"), Identity: "none",
	}
}

func setSnapshot(workload *model.Workload, path string) {
	workload.Snapshot = model.Snapshot{
		Path:         path,
		Recipients:   []string{"age1" + strings.Repeat("q", 58)},
		IdentityFile: filepath.Join(filepath.Dir(path), "snapshot.age"),
		SOPS:         filepath.Join(filepath.Dir(path), "sops"),
	}
}

func setAuthorityTrustStore(policy *model.Policy, path string) {
	authority := policy.MCP.Authorities["workforce"]
	authority.TrustStoreFile = path
	authority.TrustStoreSHA256 = strings.Repeat("a", 64)
	policy.MCP.Authorities["workforce"] = authority
}

func requireValidConfigurationFixture(t *testing.T, policy model.Policy) {
	t.Helper()
	if err := Validate(policy); err != nil {
		t.Fatalf("valid configuration fixture rejected: %v", err)
	}
}

func requireConfigurationGuardReject(t *testing.T, policy model.Policy, operation string) {
	t.Helper()
	err := Validate(policy)
	var sanitized *fault.Error
	if !errors.As(err, &sanitized) || sanitized.Kind != fault.Invalid || sanitized.Operation != operation {
		t.Fatalf("expected invalid policy %q, got %v", operation, err)
	}
}
