package mcp

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
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

func TestDPoPServiceRejectsTrustStoreInsideReplayState(t *testing.T) {
	directory := filepath.Join(t.TempDir(), "state")
	config := senderConstrainedPolicy(t)
	authority := config.MCP.Authorities["workforce"]
	authority.TrustStoreFile = filepath.Join(directory, "authority-roots.pem")
	authority.TrustStoreSHA256 = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	config.MCP.Authorities["workforce"] = authority
	service, err := New(config, &fakeBackend{}, Options{Audit: &memoryAudit{}, DPoPStateDirectory: directory})
	if service != nil || !fault.Is(err, fault.Invalid) {
		t.Fatalf("replay-state trust store accepted: service=%v err=%v", service != nil, err)
	}
	if _, err := os.Lstat(filepath.Join(directory, "dpop")); !os.IsNotExist(err) {
		t.Fatalf("replay store created before trust-store rejection: %v", err)
	}
}

func TestProtectedOperationalInputWithinDPoPStateUsesFilesystemBoundaries(t *testing.T) {
	root := filepath.VolumeName(os.TempDir()) + string(filepath.Separator)
	state := filepath.Join(root, "cq-state")
	for name, test := range map[string]struct {
		state, protected string
		kind             string
		want             bool
	}{
		"authority same path":       {state, state, "authority", true},
		"observability token child": {state, filepath.Join(state, "telemetry-token"), "telemetry-token", true},
		"observability CA child":    {state, filepath.Join(state, "collector-roots.pem"), "telemetry-trust", true},
		"update token child":        {state, filepath.Join(state, "update-token"), "update-token", true},
		"update CA child":           {state, filepath.Join(state, "update-roots.pem"), "update-trust", true},
		"prefix sibling":            {state, filepath.Join(root, "cq-state-other", "roots.pem"), "authority", false},
		"parent directory":          {state, filepath.Dir(state), "authority", false},
		"root directory":            {root, filepath.Join(root, "authority-roots.pem"), "authority", true},
	} {
		t.Run(name, func(t *testing.T) {
			config := model.Policy{MCP: model.MCP{Authorities: map[string]model.IdentityAuthority{"authority": {}}}}
			switch test.kind {
			case "authority":
				authority := config.MCP.Authorities["authority"]
				authority.TrustStoreFile = test.protected
				config.MCP.Authorities["authority"] = authority
			case "telemetry-token":
				config.Observability = &model.Observability{TokenFile: test.protected}
			case "telemetry-trust":
				config.Observability = &model.Observability{TrustStoreFile: test.protected}
			case "update-token":
				config.Updates = &model.Updates{TokenFile: test.protected}
			case "update-trust":
				config.Updates = &model.Updates{TrustStoreFile: test.protected}
			}
			if got := protectedOperationalInputWithinDPoPState(config, test.state); got != test.want {
				t.Fatalf("protectedOperationalInputWithinDPoPState() = %t, want %t", got, test.want)
			}
		})
	}
	if runtime.GOOS == "windows" {
		config := model.Policy{MCP: model.MCP{Authorities: map[string]model.IdentityAuthority{"authority": {TrustStoreFile: strings.ToLower(filepath.Join(state, "authority-roots.pem"))}}}}
		if !protectedOperationalInputWithinDPoPState(config, strings.ToUpper(state)) {
			t.Fatal("case-equivalent Windows paths were treated as separate directories")
		}
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
