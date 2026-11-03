package policy

import (
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/maldous/CircularQuay/internal/model"
)

func TestPathOwnershipRejectsWritersOverInputsAndEachOther(t *testing.T) {
	for name, change := range map[string]func(*model.Policy){
		"source": func(policy *model.Policy) {
			setSnapshotPath(policy, "/fixtures/values.json")
		},
		"bootstrap": func(policy *model.Policy) {
			changeSource(policy, func(source *model.Source) { source.TokenFile = "/private/token" })
			setFileTarget(policy, "sink", "/private/token")
		},
		"command": func(policy *model.Policy) {
			changeWorkload(policy, func(workload *model.Workload) { workload.Command = []string{"/private/worker"} })
			setFileTarget(policy, "sink", "/private/worker")
		},
		"cross writer case fold": func(policy *model.Policy) {
			setSnapshotPath(policy, "/private/Output")
			setFileTarget(policy, "sink", "/private/output")
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

func TestPathOwnershipRejectsRepositoryAndPolicyOutputs(t *testing.T) {
	policy := validPolicy()
	policy.Repository = model.Repository{
		URL:                "https://repository.example.test/config",
		Directory:          "/private/repository",
		Ref:                "refs/heads/main",
		AllowedSignersFile: "/private/signers",
		InitialCommit:      "0123456789012345678901234567890123456789",
	}
	setSnapshotPath(&policy, "/private/repository/snapshot")
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
	policy := validPolicy()
	policy.MCP.AuditPath = "/private/audit.jsonl"
	setFileTarget(&policy, "sink", "/private/output")
	if err := Validate(policy); err != nil {
		t.Fatal(err)
	}
}

func TestAuditPathOwnershipRejectsInputsAndWriters(t *testing.T) {
	for name, change := range map[string]func(*model.Policy){
		"source": func(policy *model.Policy) {
			policy.MCP.AuditPath = "/fixtures/values.json"
		},
		"repository signer": func(policy *model.Policy) {
			policy.Repository = testRepository()
			policy.MCP.AuditPath = policy.Repository.AllowedSignersFile
		},
		"snapshot": func(policy *model.Policy) {
			setSnapshotPath(policy, "/private/snapshot")
			policy.MCP.AuditPath = "/private/snapshot"
		},
		"target": func(policy *model.Policy) {
			setFileTarget(policy, "sink", "/private/output")
			policy.MCP.AuditPath = "/private/output"
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
	policy := validPolicy()
	changeWorkload(&policy, func(workload *model.Workload) {
		workload.Bindings[0].AllowPublish = true
		workload.Targets = map[string]model.Target{
			"first":  netlifyTarget("production"),
			"second": netlifyTarget("deploy-preview"),
		}
	})
	if Validate(policy) == nil {
		t.Fatal("Netlify variable contexts accepted as separate owners")
	}
}

func TestRemoteTargetOwnershipNormalizesImplementedDestinations(t *testing.T) {
	for name, targets := range map[string]map[string]model.Target{
		"cloudflare": {
			"first":  {Type: "cloudflare", Account: "account", Resource: "worker", Path: "CQUAY_CONFIG", TokenFile: "/private/first", AcceptSnapshot: true},
			"second": {Type: "cloudflare", Endpoint: "https://api.cloudflare.com/", Account: "account", Resource: "worker", Path: "CQUAY_CONFIG", TokenFile: "/private/second", AcceptSnapshot: true},
		},
		"windmill": {
			"first":  {Type: "windmill", Endpoint: "https://windmill.example", Project: "workspace", Resource: "variable", TokenFile: "/private/first", AcceptSnapshot: true},
			"second": {Type: "windmill", Endpoint: "https://windmill.example/", Project: "workspace", Resource: "variable", TokenFile: "/private/second", AcceptSnapshot: true},
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

func testRepository() model.Repository {
	return model.Repository{
		URL:                "https://repository.example.test/config",
		Directory:          "/private/repository",
		Ref:                "refs/heads/main",
		AllowedSignersFile: "/private/signers",
		InitialCommit:      "0123456789012345678901234567890123456789",
	}
}

func netlifyTarget(environment string) model.Target {
	return model.Target{
		Type: "netlify", Account: "account", Project: "site", Environment: environment,
		Path: "CQUAY_CONFIG", TokenFile: "/private/netlify-token", AcceptSnapshot: true,
	}
}

func setSnapshotPath(policy *model.Policy, path string) {
	changeWorkload(policy, func(workload *model.Workload) {
		workload.Snapshot = model.Snapshot{
			Path:         path,
			Recipients:   []string{"age1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqq"},
			IdentityFile: "/private/snapshot.age",
			SOPS:         "/usr/bin/sops",
		}
	})
}

func setFileTarget(policy *model.Policy, name, path string) {
	changeWorkload(policy, func(workload *model.Workload) {
		workload.Bindings[0].AllowPublish = true
		workload.Targets = map[string]model.Target{name: {Type: "file", Path: path, AcceptSnapshot: true}}
	})
}
