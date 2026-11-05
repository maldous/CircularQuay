package cli

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/maldous/CircularQuay/internal/fault"
	cqmcp "github.com/maldous/CircularQuay/internal/mcp"
	"github.com/maldous/CircularQuay/internal/model"
)

func TestMCPRunRejectsExpiredPrincipalAfterResolution(t *testing.T) {
	expires := time.Now().Add(time.Minute)
	now := expires.Add(-time.Second)
	marker := filepath.Join(t.TempDir(), "started")
	app, _, policy := mcpRunTestApplication(t, marker)
	resolves := 0
	b := backend{
		app: app,
		now: func() time.Time { return now },
		resolveGeneration: func(context.Context, model.Policy, string) (model.Generation, error) {
			resolves++
			now = expires
			return model.Generation{Workload: "demo", Values: map[string]model.Value{"TOKEN": {Data: "synthetic"}}}, nil
		},
	}
	service, err := cqmcp.New(policy, b, cqmcp.Options{PolicyProvider: app.load, ActionAllowed: app.actionAllowed})
	if err != nil {
		t.Fatal(err)
	}
	principal := cqmcp.Principal{Subject: "local:alice", HTTP: true, Scopes: []string{"cq:run", "cq:read"}, ExpiresAt: expires}
	if _, err := service.RunWorkload(context.Background(), principal, "demo"); !fault.Is(err, fault.Denied) {
		t.Fatalf("expired action error = %v", err)
	}
	if resolves != 1 {
		t.Fatalf("resolution calls = %d, want 1", resolves)
	}
	if _, err := os.Stat(marker); !os.IsNotExist(err) {
		t.Fatalf("expired principal reached workload: %v", err)
	}
}
