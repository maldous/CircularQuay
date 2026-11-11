package policy

import (
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/maldous/CircularQuay/internal/model"
)

func TestPathOwnershipRejectsWritersOverInputsAndEachOther(t *testing.T) {
	base := t.TempDir()
	for name, change := range map[string]func(*model.Policy){
		"source": func(policy *model.Policy) {
			path := filepath.Join(base, "values.json")
			changeSource(policy, func(source *model.Source) { source.Path = path })
			setSnapshotPath(policy, path)
		},
		"bootstrap": func(policy *model.Policy) {
			path := filepath.Join(base, "token")
			changeSource(policy, func(source *model.Source) { source.TokenFile = path })
			setFileTarget(policy, "sink", path)
		},
		"command": func(policy *model.Policy) {
			path := filepath.Join(base, "worker")
			changeWorkload(policy, func(workload *model.Workload) { workload.Command = []string{path} })
			setFileTarget(policy, "sink", path)
		},
		"cross writer case fold": func(policy *model.Policy) {
			setSnapshotPath(policy, filepath.Join(base, "Output"))
			setFileTarget(policy, "sink", filepath.Join(base, "output"))
		},
	} {
		t.Run(name, func(t *testing.T) {
			policy := validPolicy()
			change(&policy)
			if Validate(policy) == nil {
				t.Fatal("conflicting local paths accepted")
			}
		})
	}
}

func TestPathOwnershipTreatsAuthorityTrustStoreAsInput(t *testing.T) {
	path := filepath.Join(t.TempDir(), "authority-roots.pem")
	policy := namedIdentityPolicy()
	authority := policy.MCP.Authorities["workforce"]
	authority.TrustStoreFile = path
	authority.TrustStoreSHA256 = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	policy.MCP.Authorities["workforce"] = authority
	setFileTarget(&policy, "sink", path)
	if Validate(policy) == nil {
		t.Fatal("trust-store writer collision accepted")
	}
}

func TestPathOwnershipAllowsOmittedProtectedReferences(t *testing.T) {
	if err := Validate(namedIdentityPolicy()); err != nil {
		t.Fatalf("named authority without a local trust store rejected: %v", err)
	}
	policy := validPolicy()
	policy.Observability = validObservability()
	policy.Updates = validUpdates()
	if err := Validate(policy); err != nil {
		t.Fatalf("omitted operational protected references rejected: %v", err)
	}
}

func TestPolicyFileOwnershipRejectsAuthorityTrustStoreUnderMutableInputs(t *testing.T) {
	base := t.TempDir()
	for name, path := range map[string]string{
		"policy":     filepath.Join(base, "policy.json"),
		"repository": filepath.Join(base, "repository", "authority-roots.pem"),
	} {
		t.Run(name, func(t *testing.T) {
			policy := namedIdentityPolicy()
			policy.Repository = testRepository(base)
			authority := policy.MCP.Authorities["workforce"]
			authority.TrustStoreFile = path
			authority.TrustStoreSHA256 = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
			policy.MCP.Authorities["workforce"] = authority
			if validatePolicyFileOwnership(policy, filepath.Join(base, "policy.json")) == nil {
				t.Fatal("authority trust store under mutable policy input accepted")
			}
		})
	}
}

func TestPathOwnershipRejectsRepositoryAndPolicyOutputs(t *testing.T) {
	base := t.TempDir()
	policy := validPolicy()
	policy.Repository = model.Repository{
		URL:                "https://repository.example.test/config",
		Directory:          filepath.Join(base, "repository"),
		Ref:                "refs/heads/main",
		AllowedSignersFile: filepath.Join(base, "signers"),
		InitialCommit:      "0123456789012345678901234567890123456789",
	}
	setSnapshotPath(&policy, filepath.Join(base, "repository", "snapshot"))
	if Validate(policy) == nil {
		t.Fatal("repository output accepted")
	}

	directory := t.TempDir()
	policy = validPolicy()
	setFileTarget(&policy, "sink", filepath.Join(directory, "policy.json"))
	if validatePolicyFileOwnership(policy, filepath.Join(directory, "policy.json")) == nil {
		t.Fatal("policy file output accepted")
	}
}

func TestPathOwnershipAllowsSeparateInputsAndWriters(t *testing.T) {
	base := t.TempDir()
	policy := validPolicy()
	policy.MCP.AuditPath = filepath.Join(base, "audit.jsonl")
	setFileTarget(&policy, "sink", filepath.Join(base, "output"))
	if err := Validate(policy); err != nil {
		t.Fatal(err)
	}
}

func TestAuditPathOwnershipRejectsInputsAndWriters(t *testing.T) {
	base := t.TempDir()
	for name, change := range map[string]func(*model.Policy){
		"source": func(policy *model.Policy) {
			path := filepath.Join(base, "values.json")
			changeSource(policy, func(source *model.Source) { source.Path = path })
			policy.MCP.AuditPath = path
		},
		"repository signer": func(policy *model.Policy) {
			policy.Repository = testRepository(base)
			policy.MCP.AuditPath = policy.Repository.AllowedSignersFile
		},
		"snapshot": func(policy *model.Policy) {
			path := filepath.Join(base, "snapshot")
			setSnapshotPath(policy, path)
			policy.MCP.AuditPath = path
		},
		"target": func(policy *model.Policy) {
			path := filepath.Join(base, "output")
			setFileTarget(policy, "sink", path)
			policy.MCP.AuditPath = path
		},
	} {
		t.Run(name, func(t *testing.T) {
			policy := validPolicy()
			change(&policy)
			if Validate(policy) == nil {
				t.Fatal("audit writer collision accepted")
			}
		})
	}
}

func TestAuditPathOwnershipRejectsPolicyFile(t *testing.T) {
	directory := t.TempDir()
	policy := validPolicy()
	policy.MCP.AuditPath = filepath.Join(directory, "policy.json")
	if validatePolicyFileOwnership(policy, policy.MCP.AuditPath) == nil {
		t.Fatal("policy file audit writer accepted")
	}
}

func TestParseRejectsConfigLockNamespaceReferences(t *testing.T) {
	directory := t.TempDir()
	config := filepath.Join(directory, "policy.json")
	lock := filepath.Join(directory, configLockDirectory)
	for name, change := range map[string]func(*model.Policy){
		"target": func(policy *model.Policy) {
			setFileTarget(policy, "sink", filepath.Join(lock, "policy.json.lock"))
		},
		"snapshot": func(policy *model.Policy) {
			setSnapshotPath(policy, filepath.Join(lock, "snapshot"))
		},
		"audit": func(policy *model.Policy) {
			policy.MCP.AuditPath = filepath.Join(lock, "audit.jsonl")
		},
	} {
		t.Run(name, func(t *testing.T) {
			policy := validPolicy()
			change(&policy)
			data, err := json.Marshal(policy)
			if err != nil {
				t.Fatal(err)
			}
			if _, err = Parse(data, config); err == nil {
				t.Fatal("reserved config metadata path accepted")
			}
		})
	}
}

func TestParseAllowsSeparateConfigLockAndPolicyPaths(t *testing.T) {
	directory := t.TempDir()
	policy := validPolicy()
	setFileTarget(&policy, "sink", filepath.Join(directory, "output"))
	policy.MCP.AuditPath = filepath.Join(directory, "audit.jsonl")
	data, err := json.Marshal(policy)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = Parse(data, filepath.Join(directory, "policy.json")); err != nil {
		t.Fatal(err)
	}
}

func TestRemoteTargetOwnershipRejectsWholeNetlifyVariable(t *testing.T) {
	base := t.TempDir()
	policy := validPolicy()
	changeWorkload(&policy, func(workload *model.Workload) {
		workload.Bindings[0].AllowPublish = true
		workload.Targets = map[string]model.Target{
			"first":  netlifyTarget(base, "production"),
			"second": netlifyTarget(base, "deploy-preview"),
		}
	})
	if Validate(policy) == nil {
		t.Fatal("Netlify variable contexts accepted as separate owners")
	}
}

func TestRemoteTargetOwnershipNormalizesImplementedDestinations(t *testing.T) {
	base := t.TempDir()
	for name, targets := range map[string]map[string]model.Target{
		"cloudflare": {
			"first":  {Type: "cloudflare", Account: "account", Resource: "worker", Path: "CQ_CONFIG", TokenFile: filepath.Join(base, "first"), AcceptSnapshot: true},
			"second": {Type: "cloudflare", Endpoint: "https://api.cloudflare.com/", Account: "account", Resource: "worker", Path: "CQ_CONFIG", TokenFile: filepath.Join(base, "second"), AcceptSnapshot: true},
		},
		"windmill": {
			"first":  {Type: "windmill", Endpoint: "https://windmill.example", Project: "workspace", Resource: "variable", TokenFile: filepath.Join(base, "first"), AcceptSnapshot: true},
			"second": {Type: "windmill", Endpoint: "https://windmill.example/", Project: "workspace", Resource: "variable", TokenFile: filepath.Join(base, "second"), AcceptSnapshot: true},
		},
	} {
		t.Run(name, func(t *testing.T) {
			policy := validPolicy()
			changeWorkload(&policy, func(workload *model.Workload) {
				workload.Bindings[0].AllowPublish = true
				workload.Targets = targets
			})
			if Validate(policy) == nil {
				t.Fatal("equivalent remote destinations accepted")
			}
		})
	}
}

func testRepository(base string) model.Repository {
	return model.Repository{
		URL:                "https://repository.example.test/config",
		Directory:          filepath.Join(base, "repository"),
		Ref:                "refs/heads/main",
		AllowedSignersFile: filepath.Join(base, "signers"),
		InitialCommit:      "0123456789012345678901234567890123456789",
	}
}

func netlifyTarget(base, environment string) model.Target {
	return model.Target{
		Type: "netlify", Account: "account", Project: "site", Environment: environment,
		Path: "CQ_CONFIG", TokenFile: filepath.Join(base, "netlify-token"), AcceptSnapshot: true,
	}
}

func setSnapshotPath(policy *model.Policy, path string) {
	changeWorkload(policy, func(workload *model.Workload) {
		workload.Snapshot = model.Snapshot{
			Path:         path,
			Recipients:   []string{"age1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqq"},
			IdentityFile: filepath.Join(filepath.Dir(path), "snapshot.age"),
			SOPS:         filepath.Join(filepath.Dir(path), "sops"),
		}
	})
}

func setFileTarget(policy *model.Policy, name, path string) {
	changeWorkload(policy, func(workload *model.Workload) {
		workload.Bindings[0].AllowPublish = true
		workload.Targets = map[string]model.Target{name: {Type: "file", Path: path, AcceptSnapshot: true}}
	})
}
