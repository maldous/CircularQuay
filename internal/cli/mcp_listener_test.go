package cli

import (
	"context"
	"testing"
	"testing/synctest"

	"github.com/maldous/CircularQuay/internal/fault"
	cqmcp "github.com/maldous/CircularQuay/internal/mcp"
	"github.com/maldous/CircularQuay/internal/model"
)

func TestMCPListenerFailureReleasesShutdownWaiter(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		config := model.Policy{
			Version: 1,
			Sources: map[string]model.Source{"fixture": {Type: "json", Path: "synthetic.json"}},
			Workloads: map[string]model.Workload{"demo": {
				Mode:     "offline",
				Bindings: []model.Binding{{Name: "TOKEN", Source: "fixture", Field: "TOKEN", AllowOffline: true}},
			}},
			MCP: model.MCP{Issuer: "https://issuer.example", Audience: "https://api.example/mcp", JWKSURL: "https://issuer.example/keys"},
		}
		service, err := cqmcp.New(config, backend{}, cqmcp.Options{})
		if err != nil {
			t.Fatal(err)
		}
		// A negative port fails before a socket is opened. The background parent
		// deliberately stays live: a failed listener must own its own cleanup.
		if _, err = runMCPHTTP(context.Background(), "127.0.0.1:-1", service, mcpServeOptions{}); !fault.Is(err, fault.Apply) {
			t.Fatalf("listener failure = %v", err)
		}
		// synctest requires every owned goroutine to exit. An orphaned shutdown
		// waiter cannot be hidden by a timer, process exit, or parent cancellation.
	})
}
