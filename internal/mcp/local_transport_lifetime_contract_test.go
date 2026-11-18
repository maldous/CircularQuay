package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/maldous/CircularQuay/internal/fault"
	"github.com/maldous/CircularQuay/internal/model"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// These are in-process protocol and owned-path contracts. No real standard
// stream, listener, provider, child process or operational enrollment is used.
func TestLocalTransportLifetimeContractIdentityFailureNeverConnects(t *testing.T) {
	backend := &fakeBackend{}
	service := makeService(t, testPolicy(t), backend, Options{})
	wanted := fault.New(fault.Trust, "local mcp identity")
	principalCalls := 0
	transport := &localTransportRefusal{err: errors.New("UNREACHED_PRIVATE_TRANSPORT")}
	err := service.runStdioWith(context.Background(), func() (Principal, error) {
		principalCalls++
		return Principal{}, wanted
	}, transport)
	if err != wanted || principalCalls != 1 || transport.calls != 0 {
		t.Fatal("identity refusal changed or connected a transport")
	}
	assertMCPServiceRefusalNoBackendCalls(t, backend)
}

func TestLocalTransportLifetimeContractConnectionErrorIsReturnedExactly(t *testing.T) {
	principal := localTransportPrincipal(t)
	backend := &fakeBackend{}
	service := makeService(t, testPolicy(t), backend, Options{})
	wanted := errors.New("OWNED_TRANSPORT_REFUSAL")
	transport := &localTransportRefusal{err: wanted}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	err := service.runStdioWith(ctx, func() (Principal, error) { return principal, nil }, transport)
	if err != wanted || transport.calls != 1 || transport.context != ctx {
		t.Fatal("connection lost exact original error, context or call count")
	}
	assertMCPServiceRefusalNoBackendCalls(t, backend)
}

type localTransportRefusal struct {
	err     error
	calls   int
	context context.Context
}

func (p *localTransportRefusal) Connect(ctx context.Context) (sdk.Connection, error) {
	p.calls++
	p.context = ctx
	return nil, p.err
}

func localTransportPrincipal(t *testing.T) Principal {
	t.Helper()
	principal, err := LocalPrincipal()
	if err != nil || principal.Subject == "" || principal.HTTP || principal.Client != "" {
		t.Fatal("actual local metadata did not yield a local principal", err)
	}
	return principal
}

func TestLocalTransportLifetimeContractActualSDKRetainsKernelPrincipalAndStops(t *testing.T) {
	principal := localTransportPrincipal(t)
	config := testPolicy(t)
	config.Grants[0].Subject = principal.Subject
	backend := &fakeBackend{status: Status{State: "ready"}}
	service := makeService(t, config, backend, Options{})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	serverTransport, clientTransport := sdk.NewInMemoryTransports()
	finished := make(chan error, 1)
	go func() { finished <- service.runStdioWith(ctx, LocalPrincipal, serverTransport) }()
	t.Cleanup(func() { localTransportFinish(t, cancel, finished) })
	client, err := sdk.NewClient(&sdk.Implementation{Name: "owned-local-transport", Version: "1"}, nil).Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatal("owned in-memory client refused", err)
	}
	defer client.Close()
	localTransportStatus(t, ctx, client)
	backend.mu.Lock()
	actual, calls := backend.principal, backend.statuses
	backend.mu.Unlock()
	if !reflect.DeepEqual(actual, principal) || calls != 1 {
		t.Fatal("in-memory protocol changed the actual kernel principal")
	}
	localTransportDenied(t, ctx, client, backend)
}

func localTransportStatus(t *testing.T, ctx context.Context, client *sdk.ClientSession) {
	t.Helper()
	result, err := client.CallTool(ctx, &sdk.CallToolParams{Name: "get_status", Arguments: map[string]any{"workload": "alpha"}})
	if err != nil || result.IsError || len(result.Content) != 1 {
		t.Fatal("owned scoped status failed", err)
	}
	content, ok := result.Content[0].(*sdk.TextContent)
	if !ok {
		t.Fatal("status changed its text DTO")
	}
	var status Status
	if err := json.Unmarshal([]byte(content.Text), &status); err != nil || status != (Status{Workload: "alpha", Mode: "offline", State: "ready"}) {
		t.Fatal("status lost its complete safe public shape", err)
	}
}

func localTransportDenied(t *testing.T, ctx context.Context, client *sdk.ClientSession, backend *fakeBackend) {
	t.Helper()
	result, err := client.CallTool(ctx, &sdk.CallToolParams{Name: "get_status", Arguments: map[string]any{"workload": "beta"}})
	if err != nil || !result.IsError {
		t.Fatal("ungranted workload did not refuse", err)
	}
	backend.mu.Lock()
	defer backend.mu.Unlock()
	if backend.statuses != 1 || backend.reads != 0 || backend.runs != 0 {
		t.Fatal("ungranted workload reached a backend effect")
	}
}

func localTransportFinish(t *testing.T, cancel context.CancelFunc, finished <-chan error) {
	t.Helper()
	cancel()
	select {
	case err := <-finished:
		if err != nil && !errors.Is(err, context.Canceled) {
			t.Error("owned server returned an unexpected termination", err)
		}
	case <-time.After(time.Second):
		t.Error("owned in-memory server did not terminate after cancellation")
	}
}

func TestLocalTransportLifetimeContractMissingTokenNeverDispatches(t *testing.T) {
	calls := 0
	handler := boundTokenDeadline(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { calls++ }))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "https://example.test/mcp", nil))
	if calls != 0 || response.Code != http.StatusUnauthorized || response.Body.String() != "invalid access token\n" {
		t.Fatal("missing verified token reached dispatch or changed safe refusal")
	}
}

func TestLocalTransportLifetimeContractVerifierRedirectIsRefusedWithoutHTTP(t *testing.T) {
	client := accessTokenHTTPClient()
	if client.Timeout != 10*time.Second || client.Transport != nil || client.CheckRedirect == nil {
		t.Fatal("production verifier transport defaults changed")
	}
	request := httptest.NewRequest(http.MethodGet, "https://unrelated.example.test/keys", nil)
	if err := client.CheckRedirect(request, []*http.Request{httptest.NewRequest(http.MethodGet, "https://issuer.example.test/keys", nil)}); err != http.ErrUseLastResponse {
		t.Fatal("verifier permitted an authority redirect")
	}
	if issuers := configuredIssuers(model.MCP{}); issuers != nil {
		t.Fatal("absent issuer invented an authority")
	}
}

func TestLocalTransportLifetimeContractAuditPathFailurePreservesParent(t *testing.T) {
	path := privateAuditPath(t)
	parent := filepath.Dir(path)
	before, err := os.ReadDir(parent)
	if err != nil {
		t.Fatal(err)
	}
	// One component exceeds supported native filesystem component limits. This
	// exercises a real metadata error, not injected FileInfo or a permission claim.
	auditor, err := NewFileAuditor(filepath.Join(parent, strings.Repeat("a", 512)))
	requireMCPTransportBoundaryError(t, err, fault.Trust, "audit file")
	if auditor != nil {
		t.Fatal("invalid metadata lookup returned a partial auditor")
	}
	after, err := os.ReadDir(parent)
	if err != nil || !reflect.DeepEqual(before, after) {
		t.Fatal("audit constructor changed the owned parent", err)
	}
}
