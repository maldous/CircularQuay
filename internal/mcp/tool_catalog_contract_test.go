package mcp

import (
	"context"
	"testing"

	"github.com/maldous/CircularQuay/internal/model"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestSDKToolCatalogOmitsRawReadWithoutGrantAndPreservesPeerRead(t *testing.T) {
	policy := testPolicy(t)
	policy.Grants = []model.Grant{
		{Subject: "local:status-only", Workload: "beta", Operations: []string{"status"}},
		{Subject: "local:reader", Workload: "alpha", Operations: []string{"read"}, Keys: []string{"SERVICE_KEY"}},
	}
	backend := &fakeBackend{
		status: Status{Workload: "beta", Mode: "offline", State: "ready"},
		value:  model.Value{Data: "synthetic-peer-value", Version: "v1"},
	}
	service := makeService(t, policy, backend, Options{Audit: &memoryAudit{}})

	statusClient, statusContext := inMemoryMCPClientForPrincipal(t, service, Principal{Subject: "local:status-only"})
	assertToolCatalog(t, statusClient, statusContext, "get_status")
	assertToolAbsent(t, statusClient, statusContext, "read_secret")
	status, err := statusClient.CallTool(statusContext, &sdk.CallToolParams{Name: "get_status", Arguments: map[string]any{"workload": "beta"}})
	if err != nil || status.IsError || backend.statuses != 1 {
		t.Fatalf("status-only MCP operation result=%+v err=%v calls=%d", status, err, backend.statuses)
	}

	readerClient, readerContext := inMemoryMCPClientForPrincipal(t, service, Principal{Subject: "local:reader"})
	assertToolCatalog(t, readerClient, readerContext, "read_secret")
	read, err := readerClient.CallTool(readerContext, &sdk.CallToolParams{Name: "read_secret", Arguments: map[string]any{"workload": "alpha", "key": "SERVICE_KEY"}})
	if err != nil || read.IsError || backend.reads != 1 {
		t.Fatalf("peer read MCP operation result=%+v err=%v calls=%d", read, err, backend.reads)
	}
}

func inMemoryMCPClientForPrincipal(t *testing.T, service *Service, principal Principal) (*sdk.ClientSession, context.Context) {
	t.Helper()
	context, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	serverTransport, clientTransport := sdk.NewInMemoryTransports()
	server, err := service.server(principal).Connect(context, serverTransport, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { server.Close() })
	client, err := sdk.NewClient(&sdk.Implementation{Name: "catalog-contract", Version: "1"}, nil).Connect(context, clientTransport, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { client.Close() })
	return client, context
}

func assertToolCatalog(t *testing.T, client *sdk.ClientSession, ctx context.Context, wanted string) {
	t.Helper()
	tools, err := client.ListTools(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, tool := range tools.Tools {
		if tool.Name == wanted {
			return
		}
	}
	t.Fatalf("tool catalog does not contain %q: %+v", wanted, tools.Tools)
}

func assertToolAbsent(t *testing.T, client *sdk.ClientSession, ctx context.Context, unwanted string) {
	t.Helper()
	tools, err := client.ListTools(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, tool := range tools.Tools {
		if tool.Name == unwanted {
			t.Fatalf("tool catalog exposed %q: %+v", unwanted, tools.Tools)
		}
	}
}
