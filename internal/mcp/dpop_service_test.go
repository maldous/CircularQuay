package mcp

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/maldous/CircularQuay/internal/fault"
	"github.com/maldous/CircularQuay/internal/model"
)

func senderConstrainedPolicy(t *testing.T) model.Policy {
	t.Helper()
	config := namedAuthorityServicePolicy(t)
	authority := config.MCP.Authorities["workforce"]
	authority.SenderConstraint = &model.SenderConstraint{
		Type: "dpop", ProofMaxAgeSeconds: 60, Nonce: "required", ReplayStore: "local_persistent",
	}
	config.MCP.Authorities["workforce"] = authority
	return config
}

func TestDPoPServiceRequiresProtectedPersistentState(t *testing.T) {
	config := senderConstrainedPolicy(t)
	for _, directory := range []string{"", "relative-state"} {
		service, err := New(config, &fakeBackend{}, Options{Audit: &memoryAudit{}, DPoPStateDirectory: directory})
		if service != nil || !fault.Is(err, fault.Trust) {
			t.Fatalf("unprotected state accepted: service=%v err=%v", service != nil, err)
		}
	}
	directory := filepath.Dir(privateAuditPath(t))
	service := makeService(t, config, &fakeBackend{}, Options{Audit: &memoryAudit{}, DPoPStateDirectory: directory})
	if service.dpopStore == nil {
		t.Fatal("sender-constrained service has no replay store")
	}
	if info, err := os.Lstat(filepath.Join(directory, "dpop")); err != nil || !info.IsDir() {
		t.Fatalf("persistent replay directory unavailable: %v", err)
	}
}

func TestBearerServiceDoesNotCreateDPoPState(t *testing.T) {
	directory := filepath.Join(filepath.Dir(privateAuditPath(t)), "unused-state")
	service := makeService(t, namedAuthorityServicePolicy(t), &fakeBackend{}, Options{Audit: &memoryAudit{}, DPoPStateDirectory: directory})
	if service.dpopStore != nil {
		t.Fatal("ordinary bearer authority acquired a replay store")
	}
	if _, err := os.Lstat(directory); !os.IsNotExist(err) {
		t.Fatalf("optional replay state was created: %v", err)
	}
}

func TestSenderConstraintPolicyChangeRevokesExistingPrincipal(t *testing.T) {
	now := time.Date(2026, time.November, 4, 1, 0, 0, 0, time.UTC)
	config := senderConstrainedPolicy(t)
	principal := namedAuthorityPrincipal(config, now.Add(time.Minute))
	current := config
	backend := &fakeBackend{value: model.Value{Data: "SYNTHETIC-MUST-NOT-DISCLOSE"}}
	service := makeService(t, config, backend, Options{
		Audit: &memoryAudit{}, DPoPStateDirectory: filepath.Dir(privateAuditPath(t)),
		PolicyProvider: func() (model.Policy, error) { return current, nil },
	})
	service.now = func() time.Time { return now }
	authority := current.MCP.Authorities["workforce"]
	authority.SenderConstraint = nil
	current.MCP.Authorities["workforce"] = authority
	if _, err := service.ReadSecret(context.Background(), principal, "alpha", "SERVICE_KEY"); !fault.Is(err, fault.Denied) || backend.reads != 0 {
		t.Fatalf("changed sender constraint released a value: err=%v reads=%d", err, backend.reads)
	}
}
