package cq

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	mcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

// These default-adapter contracts use the real pinned SDK over owned memory.
// They do not authenticate a CQ server or replace the tagged executable lane.
func TestOfficialSessionBoundaryContractConnectedPublicAdapterForwardsExactly(t *testing.T) {
	probe := &officialSessionBoundaryProbe{}
	client, raw, ctx, observer := officialSessionBoundaryClient(t, probe)
	tools, err := client.DiscoverTools(ctx)
	if err != nil || !reflect.DeepEqual(tools, []Tool{{Name: "get_status", Description: "Synthetic safe status"}}) {
		t.Fatal("official catalog adapter changed its fixed public tool", err)
	}
	status, err := client.GetStatus(ctx, "alpha")
	if err != nil || status != (Status{Workload: "alpha", Mode: "offline", State: "ready"}) {
		t.Fatal("official call adapter changed the complete safe status", err)
	}
	probe.assertRequest(t)
	officialSessionBoundaryEvents(t, observer, []Event{{Operation: "discover_tools", Outcome: "success"}, {Operation: "get_status", Outcome: "success"}})
	// Adapting and calling the client must not take ownership of the session.
	if catalog, err := raw.ListTools(ctx, nil); err != nil || len(catalog.Tools) != 1 {
		t.Fatal("CQ adapter consumed or closed the caller-owned session", err)
	}
}

func TestOfficialSessionBoundaryContractRemoteToolErrorStaysSanitized(t *testing.T) {
	probe := &officialSessionBoundaryProbe{deny: true}
	client, _, ctx, observer := officialSessionBoundaryClient(t, probe)
	status, err := client.GetStatus(ctx, "alpha")
	assertSDKResponseRefusal(t, err, CategoryDenied)
	if status != (Status{}) || strings.Contains(err.Error(), "SYNTHETIC_PRIVATE_TOOL_DETAIL") || errors.Unwrap(err) != nil {
		t.Fatal("official adapter exposed a remote cause or partial status")
	}
	probe.assertRequest(t)
	officialSessionBoundaryEvents(t, observer, []Event{{Operation: "get_status", Outcome: "denied"}})
}

type officialSessionBoundaryInput struct {
	Workload string `json:"workload"`
}

type officialSessionBoundaryProbe struct {
	mu        sync.Mutex
	deny      bool
	calls     int
	name      string
	arguments json.RawMessage
	workload  string
}

func (p *officialSessionBoundaryProbe) status(_ context.Context, request *mcp.CallToolRequest, input officialSessionBoundaryInput) (*mcp.CallToolResult, Status, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.calls++
	p.name = request.Params.Name
	p.arguments = append(json.RawMessage(nil), request.Params.Arguments...)
	p.workload = input.Workload
	if p.deny {
		return nil, Status{}, errors.New("SYNTHETIC_PRIVATE_TOOL_DETAIL")
	}
	return nil, Status{Workload: input.Workload, Mode: "offline", State: "ready"}, nil
}

func (p *officialSessionBoundaryProbe) assertRequest(t *testing.T) {
	t.Helper()
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.calls != 1 || p.name != "get_status" || p.workload != "alpha" || string(p.arguments) != `{"workload":"alpha"}` {
		t.Fatal("official forwarding changed tool, exact arguments or call count")
	}
}

func officialSessionBoundaryClient(t *testing.T, probe *officialSessionBoundaryProbe) (*Client, *mcp.ClientSession, context.Context, *clientBoundaryObserver) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)
	server := mcp.NewServer(&mcp.Implementation{Name: "synthetic-adapter-server", Version: "1"}, &mcp.ServerOptions{Logger: slog.New(slog.NewTextHandler(io.Discard, nil))})
	mcp.AddTool(server, &mcp.Tool{Name: "get_status", Description: "Synthetic safe status"}, probe.status)
	serverTransport, clientTransport := mcp.NewInMemoryTransports()
	serverSession, err := server.Connect(ctx, serverTransport, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { officialSessionBoundaryClose(t, ctx, serverSession.Close) })
	raw, err := mcp.NewClient(&mcp.Implementation{Name: "synthetic-adapter-client", Version: "1"}, nil).Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { officialSessionBoundaryClose(t, ctx, raw.Close) })
	observer := &clientBoundaryObserver{}
	client, err := New(raw, WithObserver(observer))
	if err != nil {
		t.Fatal("connected official session was refused", err)
	}
	return client, raw, ctx, observer
}

func officialSessionBoundaryClose(t *testing.T, ctx context.Context, closeSession func() error) {
	t.Helper()
	if err := closeSession(); err != nil && ctx.Err() == nil {
		t.Error("owned official session did not close", err)
	}
}

func officialSessionBoundaryEvents(t *testing.T, observer *clientBoundaryObserver, wanted []Event) {
	t.Helper()
	if len(observer.events) != len(wanted) {
		t.Fatal("official adapter changed metadata event count")
	}
	for i, event := range observer.events {
		if event.Operation != wanted[i].Operation || event.Outcome != wanted[i].Outcome || event.Duration < 0 {
			t.Fatal("official adapter changed safe metadata-only telemetry")
		}
	}
}
