package mcp

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/maldous/CircularQuay/internal/fault"
	"github.com/maldous/CircularQuay/internal/model"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

type fakeBackend struct {
	mu                    sync.Mutex
	reads, runs, statuses int
	value                 model.Value
	err                   error
	status                Status
	run                   RunResult
	principal             Principal
	deadline              time.Time
	hasDeadline           bool
	onRead                func()
	onStatus              func()
}

func (b *fakeBackend) Read(ctx context.Context, _, _ string) (model.Value, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.reads++
	b.principal, _ = PrincipalFromContext(ctx)
	b.deadline, b.hasDeadline = ctx.Deadline()
	if b.onRead != nil {
		b.onRead()
	}
	return b.value, b.err
}
func (b *fakeBackend) Run(ctx context.Context, _ string) (RunResult, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.runs++
	b.principal, _ = PrincipalFromContext(ctx)
	return b.run, b.err
}
func (b *fakeBackend) Status(ctx context.Context, _ string) (Status, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.statuses++
	b.principal, _ = PrincipalFromContext(ctx)
	if b.onStatus != nil {
		b.onStatus()
	}
	return b.status, b.err
}

type memoryAudit struct {
	mu       sync.Mutex
	events   []Event
	fail     bool
	onRecord func()
}

func (a *memoryAudit) Record(_ context.Context, e Event) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.fail {
		return errors.New("internal details must not escape")
	}
	a.events = append(a.events, e)
	if a.onRecord != nil {
		a.onRecord()
	}
	return nil
}

func testPolicy(t *testing.T) model.Policy {
	t.Helper()
	return model.Policy{Version: 1,
		Sources: map[string]model.Source{"fixture": {Type: "json", Path: filepath.Join(t.TempDir(), "fixture.json")}},
		Workloads: map[string]model.Workload{
			"alpha": {Mode: "offline", Bindings: []model.Binding{{Name: "SERVICE_KEY", Source: "fixture", AllowOffline: true}}, Command: []string{"/bin/true"}},
			"beta":  {Mode: "offline", Bindings: []model.Binding{{Name: "SERVICE_KEY", Source: "fixture", AllowOffline: true}}},
		},
		Grants: []model.Grant{{Subject: "local:alice", Workload: "alpha", Operations: []string{"list", "status", "run", "read"}, Keys: []string{"SERVICE_KEY"}}},
	}
}

func makeService(t *testing.T, p model.Policy, b *fakeBackend, options Options) *Service {
	t.Helper()
	s, err := New(p, b, options)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestAuthorizationPrecedesBackend(t *testing.T) {
	b := &fakeBackend{value: model.Value{Data: "synthetic-only"}}
	s := makeService(t, testPolicy(t), b, Options{Audit: &memoryAudit{}, ActionAllowed: func(string) bool { return true }})
	ctx := context.Background()
	bob := Principal{Subject: "local:bob"}
	if ids, err := s.ListWorkloads(ctx, bob); err != nil || len(ids) != 0 {
		t.Fatal("unauthorized discovery")
	}
	if _, err := s.GetStatus(ctx, bob, "alpha"); !fault.Is(err, fault.Denied) {
		t.Fatal(err)
	}
	if _, err := s.RunWorkload(ctx, bob, "alpha"); !fault.Is(err, fault.Denied) {
		t.Fatal(err)
	}
	if _, err := s.ReadSecret(ctx, bob, "alpha", "SERVICE_KEY"); !fault.Is(err, fault.Denied) {
		t.Fatal(err)
	}
	if _, err := s.ReadSecret(ctx, Principal{Subject: "local:alice"}, "beta", "SERVICE_KEY"); !fault.Is(err, fault.Denied) {
		t.Fatal(err)
	}
	if b.reads+b.runs+b.statuses != 0 {
		t.Fatal("denied operation reached backend")
	}
}

func TestDisclosureReceiptAndIsolation(t *testing.T) {
	marker := "SYNTHETIC_DISCLOSURE_FIXTURE"
	b := &fakeBackend{value: model.Value{Data: marker, Version: "v1"}}
	a := &memoryAudit{}
	s := makeService(t, testPolicy(t), b, Options{Audit: a})
	p := Principal{Subject: "local:alice"}
	result, err := s.ReadSecret(context.Background(), p, "alpha", "SERVICE_KEY")
	if err != nil || result.Value != marker || result.Alias != "SERVICE_KEY" {
		t.Fatal("authorized direct disclosure failed", err)
	}
	if b.principal.Subject != p.Subject {
		t.Fatal("backend lost verified actor")
	}
	if len(a.events) != 1 || a.events[0].Outcome != "release_authorized" {
		t.Fatal("missing pre-response receipt")
	}
	if strings.Contains(a.events[0].Subject+a.events[0].Workload+a.events[0].Key+a.events[0].Outcome, marker) {
		t.Fatal("audit leaked value")
	}
	a.fail = true
	if _, err := s.ReadSecret(context.Background(), p, "alpha", "SERVICE_KEY"); !fault.Is(err, fault.Apply) {
		t.Fatal("audit failure permitted response", err)
	}
}

func TestDisclosureFailures(t *testing.T) {
	for _, tc := range []struct {
		name       string
		value      model.Value
		backendErr error
		audit      bool
		key        string
		want       fault.Kind
	}{
		{"no audit", model.Value{Data: "v"}, nil, false, "SERVICE_KEY", fault.Apply},
		{"wrong key", model.Value{Data: "v"}, nil, true, "OTHER", fault.Denied},
		{"NUL alias", model.Value{Data: "v"}, nil, true, "SERVICE_KEY\x00", fault.Denied},
		{"URI alias", model.Value{Data: "v"}, nil, true, "https://host/key", fault.Denied},
		{"leased", model.Value{Data: "v", Leased: true}, nil, true, "SERVICE_KEY", fault.Unsupported},
		{"renewable", model.Value{Data: "v", Renewable: true}, nil, true, "SERVICE_KEY", fault.Unsupported},
		{"expired", model.Value{Data: "v", ExpiresAt: time.Now().Add(-time.Second)}, nil, true, "SERVICE_KEY", fault.Expired},
		{"oversized", model.Value{Data: strings.Repeat("x", 65537)}, nil, true, "SERVICE_KEY", fault.Unsupported},
		{"safe upstream", model.Value{}, errors.New("DO_NOT_ECHO_SECRET"), true, "SERVICE_KEY", fault.Apply},
	} {
		t.Run(tc.name, func(t *testing.T) {
			b := &fakeBackend{value: tc.value, err: tc.backendErr}
			opts := Options{}
			if tc.audit {
				opts.Audit = &memoryAudit{}
			}
			s := makeService(t, testPolicy(t), b, opts)
			_, err := s.ReadSecret(context.Background(), Principal{Subject: "local:alice"}, "alpha", tc.key)
			if !fault.Is(err, tc.want) || strings.Contains(err.Error(), "DO_NOT_ECHO_SECRET") {
				t.Fatal("unsafe classification", err)
			}
		})
	}
}

func TestMandatorySourceAudit(t *testing.T) {
	p := testPolicy(t)
	src := model.Source{Type: "vault", Endpoint: "https://vault.example", Mount: "kv", Resources: []string{"app"}, Identity: "token_file", TokenFile: filepath.Join(t.TempDir(), "synthetic-token"), AuditVerified: true}
	p.Sources["audit"] = src
	w := p.Workloads["alpha"]
	w.Mode = "hybrid"
	w.Bindings[0].Source = "audit"
	w.Bindings[0].Resource = "app"
	w.AuditRequired = true
	p.Workloads["alpha"] = w
	b := &fakeBackend{value: model.Value{Data: "value"}}
	s := makeService(t, p, b, Options{Audit: &memoryAudit{}})
	if _, err := s.ReadSecret(context.Background(), Principal{Subject: "local:alice"}, "alpha", "SERVICE_KEY"); !fault.Is(err, fault.Denied) {
		t.Fatal(err)
	}
	b.value.AuditVerified = true
	if _, err := s.ReadSecret(context.Background(), Principal{Subject: "local:alice"}, "alpha", "SERVICE_KEY"); err != nil {
		t.Fatal(err)
	}
}

func TestRevocationAndInvalidReloadFailClosed(t *testing.T) {
	p := testPolicy(t)
	current := p
	b := &fakeBackend{value: model.Value{Data: "value"}}
	s := makeService(t, p, b, Options{Audit: &memoryAudit{}, PolicyProvider: func() (model.Policy, error) { return current, nil }})
	actor := Principal{Subject: "local:alice"}
	if _, err := s.ReadSecret(context.Background(), actor, "alpha", "SERVICE_KEY"); err != nil {
		t.Fatal(err)
	}
	current.Grants = nil
	if _, err := s.ReadSecret(context.Background(), actor, "alpha", "SERVICE_KEY"); !fault.Is(err, fault.Denied) {
		t.Fatal("revocation bypassed", err)
	}
	current = p
	current.Version = 99
	if _, err := s.ReadSecret(context.Background(), actor, "alpha", "SERVICE_KEY"); !fault.Is(err, fault.Denied) {
		t.Fatal("invalid reload used stale grants", err)
	}
	s.options.PolicyProvider = func() (model.Policy, error) { return model.Policy{}, errors.New("private parser details") }
	if ids, err := s.ListWorkloads(context.Background(), actor); err != nil || len(ids) != 0 {
		t.Fatal("reload failure exposed metadata")
	}
	if b.reads != 1 {
		t.Fatal("revoked calls reached source")
	}
}

func TestScopesAndClientAreIndependent(t *testing.T) {
	p := testPolicy(t)
	p.Grants = []model.Grant{{Subject: "https://issuer|alice", Client: "agent-a", Workload: "alpha", Operations: []string{"read"}, Keys: []string{"SERVICE_KEY"}}}
	b := &fakeBackend{value: model.Value{Data: "value"}}
	s := makeService(t, p, b, Options{Audit: &memoryAudit{}})
	actor := Principal{Subject: "https://issuer|alice", Client: "agent-a", HTTP: true, Scopes: []string{"cquay:mcp"}, ExpiresAt: time.Now().Add(time.Minute)}
	if _, err := s.ReadSecret(context.Background(), actor, "alpha", "SERVICE_KEY"); !fault.Is(err, fault.Denied) {
		t.Fatal("scope bypass")
	}
	actor.Scopes = append(actor.Scopes, "cquay:read")
	actor.Client = "agent-b"
	if _, err := s.ReadSecret(context.Background(), actor, "alpha", "SERVICE_KEY"); !fault.Is(err, fault.Denied) {
		t.Fatal("client bypass")
	}
	actor.Client = "agent-a"
	if _, err := s.ReadSecret(context.Background(), actor, "alpha", "SERVICE_KEY"); err != nil {
		t.Fatal(err)
	}
}

func TestActionsCannotBecomeUnreviewedDisclosure(t *testing.T) {
	p := testPolicy(t)
	actor := Principal{Subject: "local:alice"}
	b := &fakeBackend{run: RunResult{ID: "run-1", Status: "completed", ExitCode: 0}}
	s := makeService(t, p, b, Options{})
	if _, err := s.RunWorkload(context.Background(), actor, "alpha"); !fault.Is(err, fault.Denied) {
		t.Fatal("missing action approval", err)
	}
	s.options.ActionAllowed = func(string) bool { return true }
	w := p.Workloads["alpha"]
	w.UntrustedCode = true
	p.Workloads["alpha"] = w
	p.Grants[0].Operations = []string{"run"}
	s.configured = p
	if _, err := s.RunWorkload(context.Background(), actor, "alpha"); !fault.Is(err, fault.Denied) {
		t.Fatal("use-only grant executed mutable code", err)
	}
	p.Grants[0].Operations = []string{"run", "read"}
	s.configured = p
	result, err := s.RunWorkload(context.Background(), actor, "alpha")
	assertSafeRunResult(t, result, err, "run-1", "completed", 0)
	b.run = RunResult{ID: "raw output with spaces", Status: "PRIVATE_PAYLOAD", ExitCode: 999}
	result, err = s.RunWorkload(context.Background(), actor, "alpha")
	assertSafeRunResult(t, result, err, "redacted", "unknown", 1)
	b.err = errors.New("PRIVATE_PAYLOAD")
	if _, err = s.RunWorkload(context.Background(), actor, "alpha"); err == nil || strings.Contains(err.Error(), "PRIVATE_PAYLOAD") {
		t.Fatal(err)
	}
}

func assertSafeRunResult(t *testing.T, result RunResult, err error, id, status string, exitCode int) {
	t.Helper()
	if err != nil || result.ID != id || result.Status != status || result.ExitCode != exitCode {
		t.Fatal("unsafe action result", result, err)
	}
}

func TestSafeStatus(t *testing.T) {
	b := &fakeBackend{status: Status{Workload: "OTHER", Mode: "private", State: "PRIVATE_PAYLOAD"}}
	s := makeService(t, testPolicy(t), b, Options{})
	status, err := s.GetStatus(context.Background(), Principal{Subject: "local:alice"}, "alpha")
	if err != nil || status.Workload != "alpha" || status.Mode != "offline" || status.State != "unknown" {
		t.Fatal(status, err)
	}
	b.err = errors.New("PRIVATE_PAYLOAD")
	if _, err = s.GetStatus(context.Background(), Principal{Subject: "local:alice"}, "alpha"); err == nil || strings.Contains(err.Error(), "PRIVATE_PAYLOAD") {
		t.Fatal(err)
	}
}

func TestProtocolDiscoveryAndExplicitRead(t *testing.T) {
	b := &fakeBackend{value: model.Value{Data: "PROTOCOL_SYNTHETIC_MARKER"}}
	s := makeService(t, testPolicy(t), b, Options{Audit: &memoryAudit{}})
	client, ctx := inMemoryMCPClient(t, s)
	tools, err := client.ListTools(ctx, nil)
	if err != nil || len(tools.Tools) != 4 || tools.CacheScope != "private" || tools.TTLMs != 0 {
		t.Fatal("unqualified discovery cache", tools, err)
	}
	assertProtocolRead(t, client, ctx)
}

func inMemoryMCPClient(t *testing.T, s *Service) (*sdk.ClientSession, context.Context) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	serverTransport, clientTransport := sdk.NewInMemoryTransports()
	serverSession, err := s.server(Principal{Subject: "local:alice"}).Connect(ctx, serverTransport, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { serverSession.Close() })
	client, err := sdk.NewClient(&sdk.Implementation{Name: "synthetic-client", Version: "1"}, nil).Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { client.Close() })
	return client, ctx
}

func assertProtocolRead(t *testing.T, client *sdk.ClientSession, ctx context.Context) {
	t.Helper()
	result, err := client.CallTool(ctx, &sdk.CallToolParams{Name: "read_secret", Arguments: map[string]any{"workload": "alpha", "key": "SERVICE_KEY"}})
	if err != nil || result.IsError {
		t.Fatal(result, err)
	}
	text := result.Content[0].(*sdk.TextContent).Text
	if !strings.Contains(text, "PROTOCOL_SYNTHETIC_MARKER") {
		t.Fatal("authorized response omitted requested value")
	}
	denied, err := client.CallTool(ctx, &sdk.CallToolParams{Name: "read_secret", Arguments: map[string]any{"workload": "beta", "key": "SERVICE_KEY"}})
	if err != nil || !denied.IsError || strings.Contains(denied.Content[0].(*sdk.TextContent).Text, "PROTOCOL_SYNTHETIC_MARKER") {
		t.Fatal("cross-workload response", denied, err)
	}
}

func TestInvalidConfigurationAndCancellation(t *testing.T) {
	if _, err := New(model.Policy{}, &fakeBackend{}, Options{}); err == nil {
		t.Fatal("invalid policy accepted")
	}
	if _, err := New(testPolicy(t), nil, Options{}); err == nil {
		t.Fatal("nil backend accepted")
	}
	s := makeService(t, testPolicy(t), &fakeBackend{}, Options{})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := s.ListWorkloads(ctx, Principal{Subject: "local:alice"}); !fault.Is(err, fault.Unavailable) {
		t.Fatal(err)
	}
	if _, ok := PrincipalFromContext(context.Background()); ok {
		t.Fatal("invented principal")
	}
	for _, handle := range []string{"", ".", "..", "../alpha", "alpha\x00", "alpha/other", strings.Repeat("x", 129)} {
		if validHandle(handle) {
			t.Fatal("unsafe handle", handle)
		}
	}
}

func TestNewCreatesConfiguredProtectedAuditor(t *testing.T) {
	p := testPolicy(t)
	p.MCP.AuditPath = privateAuditPath(t)
	s, err := New(p, &fakeBackend{}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := s.options.Audit.(*FileAuditor); !ok {
		t.Fatal("configured audit path did not install protected auditor")
	}
	if _, err := New(p, &fakeBackend{}, Options{Audit: &memoryAudit{}}); err != nil {
		t.Fatal("explicit auditor should take precedence", err)
	}
	p.MCP.AuditPath = "relative.jsonl"
	if _, err := New(p, &fakeBackend{}, Options{}); !fault.Is(err, fault.Invalid) {
		t.Fatal("unsafe configured audit path accepted", err)
	}
}

func TestLocalPrincipalUsesKernelIdentity(t *testing.T) {
	t.Setenv("USER", "untrusted-agent-label")
	t.Setenv("USERNAME", "untrusted-agent-label")
	p, err := LocalPrincipal()
	if err != nil || untrustedLocalPrincipal(p) {
		t.Fatal("environment became identity", p, err)
	}
	if wrongKernelPrincipal(p) {
		t.Fatal("principal is not kernel UID")
	}
	assertSIDCases(t)
}

func untrustedLocalPrincipal(p Principal) bool {
	return strings.Contains(p.Subject, "untrusted-agent-label") || p.Client != "" || p.HTTP
}

func wrongKernelPrincipal(p Principal) bool {
	return runtime.GOOS != "windows" && p.Subject != "local:uid:"+strconv.Itoa(os.Getuid())
}

func assertSIDCases(t *testing.T) {
	t.Helper()
	assertValidSIDs(t, []string{"S-1-5-18", "S-1-5-21-123-456-789-1001"})
	assertInvalidSIDs(t, []string{"", "S-1-", "S-1--5", "S-1-5-", "S-2-5-18", "S-1-5-alice", strings.Repeat("a", 257)})
}

func assertValidSIDs(t *testing.T, sids []string) {
	t.Helper()
	for _, sid := range sids {
		if !validSID(sid) {
			t.Fatal("valid SID rejected")
		}
	}
}

func assertInvalidSIDs(t *testing.T, sids []string) {
	t.Helper()
	for _, sid := range sids {
		if validSID(sid) {
			t.Fatal("malformed SID accepted")
		}
	}
}

func TestCancellationPrecedesProviderAndDisclosure(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	b := &fakeBackend{value: model.Value{Data: "NEVER_RETURN_CANCELLED"}}
	s := makeService(t, testPolicy(t), b, Options{Audit: &memoryAudit{}, ActionAllowed: func(string) bool { return true }})
	p := Principal{Subject: "local:alice"}
	if _, err := s.GetStatus(ctx, p, "alpha"); !fault.Is(err, fault.Unavailable) {
		t.Fatal(err)
	}
	if _, err := s.RunWorkload(ctx, p, "alpha"); !fault.Is(err, fault.Unavailable) {
		t.Fatal(err)
	}
	if _, err := s.ReadSecret(ctx, p, "alpha", "SERVICE_KEY"); !fault.Is(err, fault.Unavailable) {
		t.Fatal(err)
	}
	if b.reads+b.runs+b.statuses != 0 {
		t.Fatal("cancelled request reached backend")
	}
	ctx, cancel = context.WithCancel(context.Background())
	b.onRead = cancel
	result, err := s.ReadSecret(ctx, p, "alpha", "SERVICE_KEY")
	if !fault.Is(err, fault.Unavailable) || result.Value != "" {
		t.Fatal("cancelled backend result was disclosed")
	}
}

func TestRevocationDuringReadPreventsDisclosure(t *testing.T) {
	p := testPolicy(t)
	b := &fakeBackend{value: model.Value{Data: "NEVER_RETURN_REVOKED"}, onRead: func() { p.Grants = nil }}
	a := &memoryAudit{}
	s := makeService(t, p, b, Options{Audit: a, PolicyProvider: func() (model.Policy, error) { return p, nil }})
	result, err := s.ReadSecret(context.Background(), Principal{Subject: "local:alice"}, "alpha", "SERVICE_KEY")
	if !fault.Is(err, fault.Denied) || result.Value != "" || len(a.events) != 0 {
		t.Fatal("revoked blocked read was disclosed")
	}
}

func TestProtocolAllowlistedOperationsAndNoArbitraryArguments(t *testing.T) {
	b := &fakeBackend{status: Status{State: "ready"}, run: RunResult{ID: "test-run", Status: "completed"}}
	s := makeService(t, testPolicy(t), b, Options{Audit: &memoryAudit{}, ActionAllowed: func(string) bool { return true }})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	serverTransport, clientTransport := sdk.NewInMemoryTransports()
	server, err := s.server(Principal{Subject: "local:alice"}).Connect(ctx, serverTransport, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer server.Close()
	client, err := sdk.NewClient(&sdk.Implementation{Name: "fixture", Version: "1"}, nil).Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	for _, op := range []string{"list_workloads", "get_status", "run_workload"} {
		args := map[string]any{}
		if op != "list_workloads" {
			args["workload"] = "alpha"
		}
		result, err := client.CallTool(ctx, &sdk.CallToolParams{Name: op, Arguments: args})
		if err != nil || result.IsError {
			t.Fatal(op, result, err)
		}
	}
	result, err := client.CallTool(ctx, &sdk.CallToolParams{Name: "run_workload", Arguments: map[string]any{"workload": "alpha", "command": "/bin/unapproved"}})
	if err != nil || !result.IsError || b.runs != 1 {
		t.Fatal("unregistered tool arguments accepted", result, err)
	}
}
