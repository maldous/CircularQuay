package cli

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"testing/synctest"
	"time"

	"github.com/maldous/CircularQuay/internal/fault"
	cqmcp "github.com/maldous/CircularQuay/internal/mcp"
	"github.com/maldous/CircularQuay/internal/model"
	"github.com/maldous/CircularQuay/internal/resolve"
	"github.com/maldous/CircularQuay/internal/state"
)

// These are actual local policy/service boundaries, not transport authentication
// or executable qualification. The enrolled inert bytes are never launched.
type mcpCompletionFixture struct {
	t          *testing.T
	app        application
	policy     model.Policy
	backend    backend
	locks      int
	releases   int
	executions int
}

func newMCPCompletionFixture(t *testing.T) *mcpCompletionFixture {
	t.Helper()
	root := privateCLIDirectory(t)
	command := filepath.Join(root, "inert-command.exe")
	contents := []byte("owned identity bytes, never executed\n")
	writeProtectedFixture(t, command, contents)
	if err := os.Chmod(command, 0o700); err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(contents)
	values := filepath.Join(root, "values.json")
	writeProtectedFixture(t, values, []byte(`{"TOKEN":"owned","OTHER":"sibling"}`))
	f := &mcpCompletionFixture{t: t}
	f.policy = model.Policy{Version: 1,
		Sources: map[string]model.Source{"fixture": {Type: "json", Path: values}},
		Workloads: map[string]model.Workload{"demo": {Mode: "offline", Command: []string{command}, CommandSHA256: hex.EncodeToString(digest[:]), UntrustedCode: true,
			Bindings: []model.Binding{{Name: "TOKEN", Source: "fixture", Field: "TOKEN", AllowOffline: true}}}},
		Grants: []model.Grant{{Subject: "local:alice", Workload: "demo", Operations: []string{"status", "read", "run"}, Keys: []string{"TOKEN"}}}}
	f.app = application{config: filepath.Join(root, "policy.json"), directory: filepath.Join(root, "state"),
		paths: &applicationPathOperations{configurationDirectory: func() (string, error) {
			return filepath.Join(root, "config"), nil
		}}}
	f.save()
	f.backend = backend{app: f.app, acquire: f.acquire, execute: f.execute}
	return f
}

func (f *mcpCompletionFixture) save() {
	f.t.Helper()
	writeMCPTestPolicy(f.t, f.app.config, f.policy)
}

func (f *mcpCompletionFixture) acquire(engine *resolve.Engine, id string) (func() error, error) {
	f.t.Helper()
	if id != "demo" || engine.State.Directory != f.app.directory {
		f.t.Fatal("lock escaped owned state")
	}
	unlock, err := engine.State.Lock(id)
	if err != nil {
		return nil, err
	}
	f.locks++
	return func() error {
		f.releases++
		return unlock()
	}, nil
}

func (f *mcpCompletionFixture) execute(ctx context.Context, w model.Workload, generation model.Generation, override []string, input io.Reader, output, diagnostics io.Writer) (int, error) {
	f.t.Helper()
	deadline, ok := ctx.Deadline()
	if !ok || time.Until(deadline) > 60*time.Second || ctx.Err() != nil {
		f.t.Fatal("execution lost its bounded live context")
	}
	if !reflect.DeepEqual(w.Command, f.policy.Workloads["demo"].Command) || generation.Workload != "demo" || generation.Values["TOKEN"].Data != "owned" {
		f.t.Fatal("execution lost admitted workload/generation")
	}
	mcpCompletionExecutionIO(f.t, override, input, output, diagnostics, f.releases)
	f.executions++
	return 0, nil
}

func (f *mcpCompletionFixture) service(b cqmcp.Backend) *cqmcp.Service {
	f.t.Helper()
	s, err := cqmcp.New(f.policy, b, cqmcp.Options{Audit: mcpRepositoryAudit{}, PolicyProvider: f.app.load, ActionAllowed: f.app.actionAllowed})
	if err != nil {
		f.t.Fatal(err)
	}
	return s
}

func TestMCPCompletionStatusActualStateAndRefusals(t *testing.T) {
	for _, scenario := range []string{"absent", "record", "corrupt", "store"} {
		t.Run(scenario, func(t *testing.T) {
			mcpCompletionStatusCase(t, scenario)
		})
	}
}

func mcpCompletionStatusCase(t *testing.T, scenario string) {
	t.Helper()
	f := newMCPCompletionFixture(t)
	want := "not_provisioned"
	if scenario == "record" {
		mcpCompletionRecord(t, f, false)
		want = "ready"
	}
	if scenario == "corrupt" {
		mcpCompletionRecord(t, f, true)
	}
	if scenario == "store" {
		f.backend.app.directory = ""
		f.backend.app.paths = mcpCompletionUnavailableDirectory()
	}
	result, err := f.service(f.backend).GetStatus(context.Background(), cqmcp.Principal{Subject: "local:alice"}, "demo")
	mcpCompletionStatusResult(t, scenario, want, result, err)
	if f.locks != 0 || f.executions != 0 {
		t.Fatal("status reached action effects")
	}
}

func mcpCompletionRecord(t *testing.T, f *mcpCompletionFixture, corrupt bool) {
	t.Helper()
	s := state.Store{Directory: f.app.directory}
	if err := s.Write(state.Record{Workload: "demo", State: "ready", Mode: "offline"}); err != nil {
		t.Fatal(err)
	}
	if corrupt {
		writeProtectedFixture(t, filepath.Join(f.app.directory, "demo.json"), []byte("not-json"))
	}
}

func mcpCompletionUnavailableDirectory() *applicationPathOperations {
	return &applicationPathOperations{configurationDirectory: func() (string, error) {
		return "", errors.New("owned unavailable configuration root")
	}}
}

func TestMCPCompletionReadActualJSONAndAcquisitionRefusals(t *testing.T) {
	for _, scenario := range []string{"value", "engine", "lock", "source"} {
		t.Run(scenario, func(t *testing.T) {
			mcpCompletionReadCase(t, scenario)
		})
	}
}

func mcpCompletionReadCase(t *testing.T, scenario string) {
	t.Helper()
	f := newMCPCompletionFixture(t)
	mcpCompletionReadFailure(t, f, scenario)
	result, err := f.service(f.backend).ReadSecret(context.Background(), cqmcp.Principal{Subject: "local:alice"}, "demo", "TOKEN")
	if scenario == "value" {
		if err != nil || result.Value != "owned" {
			t.Fatalf("actual protected JSON read = %#v, %v", result, err)
		}
	} else if !fault.Is(err, mcpCompletionReadKind(scenario)) || !reflect.DeepEqual(result, cqmcp.SecretResult{}) {
		t.Fatal("read refusal returned a partial DTO")
	}
	if f.locks != f.releases || f.executions != 0 {
		t.Fatal("read leaked lock or launched an action")
	}
}

func TestMCPCompletionReadRechecksPolicyAfterOwnedLock(t *testing.T) {
	for _, scenario := range []string{"revoked", "binding", "policy-file"} {
		t.Run(scenario, func(t *testing.T) {
			f := newMCPCompletionFixture(t)
			f.backend.acquire = func(e *resolve.Engine, id string) (func() error, error) {
				release, err := f.acquire(e, id)
				if err != nil {
					return nil, err
				}
				mcpCompletionMutateRead(t, f, scenario)
				return release, nil
			}
			result, err := f.service(f.backend).ReadSecret(context.Background(), cqmcp.Principal{Subject: "local:alice"}, "demo", "TOKEN")
			if err == nil || !reflect.DeepEqual(result, cqmcp.SecretResult{}) || f.locks != 1 || f.releases != 1 {
				t.Fatal("post-lock policy refusal leaked value or lock")
			}
		})
	}
}

func mcpCompletionMutateRead(t *testing.T, f *mcpCompletionFixture, scenario string) {
	t.Helper()
	if scenario == "policy-file" {
		writeProtectedFixture(t, f.app.config, []byte("not-json"))
		return
	}
	if scenario == "revoked" {
		f.policy.Grants = nil
	} else {
		w := f.policy.Workloads["demo"]
		w.Bindings[0].Field = "OTHER"
		f.policy.Workloads["demo"] = w
	}
	f.save()
}

func TestMCPCompletionRunUsesActualResolverAndFiniteExecutor(t *testing.T) {
	for _, scenario := range []string{"success", "engine", "lock", "resolve", "execute", "unlock"} {
		t.Run(scenario, func(t *testing.T) {
			mcpCompletionRunCase(t, scenario)
		})
	}
}

func mcpCompletionRunCase(t *testing.T, scenario string) {
	t.Helper()
	f := newMCPCompletionFixture(t)
	mcpCompletionRunFailure(f, scenario)
	result, err := f.service(f.backend).RunWorkload(context.Background(), cqmcp.Principal{Subject: "local:alice"}, "demo")
	mcpCompletionRunResult(t, f, scenario, result, err)
	wantLocks := 1
	if scenario == "engine" || scenario == "lock" {
		wantLocks = 0
	}
	if f.locks != wantLocks || f.releases != wantLocks {
		t.Fatal("action did not release its real owned lock")
	}
	if scenario != "success" && f.executions != 0 {
		t.Fatal("refusal launched successful execution")
	}
}

func mcpCompletionRunFailure(f *mcpCompletionFixture, scenario string) {
	if scenario == "engine" {
		f.backend.app.directory = ""
		f.backend.app.paths = mcpCompletionUnavailableDirectory()
	}
	if scenario == "lock" {
		f.backend.acquire = func(*resolve.Engine, string) (func() error, error) {
			return nil, fault.New(fault.Unavailable, "owned lock refusal")
		}
	}
	if scenario == "resolve" {
		f.backend.resolveGeneration = func(context.Context, model.Policy, string) (model.Generation, error) {
			return model.Generation{}, fault.New(fault.Missing, "owned resolution refusal")
		}
	}
	if scenario == "execute" {
		f.backend.execute = func(context.Context, model.Workload, model.Generation, []string, io.Reader, io.Writer, io.Writer) (int, error) {
			return 0, fault.New(fault.Apply, "owned execution refusal")
		}
	}
	if scenario == "unlock" {
		f.backend.acquire = f.acquireWithReleaseRefusal
	}
}

func (f *mcpCompletionFixture) acquireWithReleaseRefusal(e *resolve.Engine, id string) (func() error, error) {
	release, err := f.acquire(e, id)
	if err != nil {
		return nil, err
	}
	return mcpCompletionReleaseRefusal(release), nil
}

func mcpCompletionReleaseRefusal(release func() error) func() error {
	closed := false
	return func() error {
		if closed {
			return nil
		}
		closed = true
		if err := release(); err != nil {
			return err
		}
		return fault.New(fault.Apply, "owned post-release refusal")
	}
}

// The real Service supplies its private principal context. This adapter changes
// only the owned policy at the already-authorized handoff; it cannot mint one.
type mcpCompletionHandoff struct {
	backend
	beforeRun func()
}

func (b mcpCompletionHandoff) Run(ctx context.Context, id string) (cqmcp.RunResult, error) {
	b.beforeRun()
	return b.backend.Run(ctx, id)
}

func TestMCPCompletionUntrustedActionRechecksEachDisclosureGrant(t *testing.T) {
	f := newMCPCompletionFixture(t)
	b := mcpCompletionHandoff{backend: f.backend, beforeRun: func() {
		f.policy.Grants[0].Operations = []string{"run"}
		f.policy.Grants[0].Keys = nil
		f.save()
	}}
	result, err := f.service(b).RunWorkload(context.Background(), cqmcp.Principal{Subject: "local:alice"}, "demo")
	if !fault.Is(err, fault.Denied) || result != (cqmcp.RunResult{}) || f.locks != 0 || f.executions != 0 {
		t.Fatal("lost read grant reached untrusted action effects")
	}
}

func TestMCPCompletionTrustedActionDoesNotRequireValueDisclosure(t *testing.T) {
	f := newMCPCompletionFixture(t)
	w := f.policy.Workloads["demo"]
	w.UntrustedCode = false
	f.policy.Workloads["demo"] = w
	f.policy.Grants[0].Operations = []string{"run"}
	f.policy.Grants[0].Keys = nil
	f.save()
	result, err := f.service(f.backend).RunWorkload(context.Background(), cqmcp.Principal{Subject: "local:alice"}, "demo")
	if err != nil || result.Status != "completed" || f.executions != 1 || f.releases != 1 {
		t.Fatal("trusted action failed its actual scoped resolution")
	}
}

func TestMCPCompletionRepositoryRefusalReleasesOwnedLock(t *testing.T) {
	for _, operation := range []string{"read", "run"} {
		t.Run(operation, func(t *testing.T) {
			f := newMCPCompletionFixture(t)
			s := f.policy.Sources["fixture"]
			s.Path = "repo:values.json"
			f.policy.Sources["fixture"] = s
			root := filepath.Dir(f.app.config)
			f.policy.Repository = model.Repository{URL: "https://example.test/owned.git", Directory: filepath.Join(root, "repo"), Ref: "refs/heads/data", AllowedSignersFile: filepath.Join(root, "signers"), InitialCommit: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}
			f.save()
			calls := 0
			f.backend.app.repository = repositoryOperations{fetch: func(context.Context, model.Repository) (string, error) {
				t.Fatal("offline read fetched")
				return "", nil
			}, current: func(context.Context, model.Repository) (string, error) {
				calls++
				return "", fault.New(fault.Trust, "owned current refusal")
			}}
			svc := f.service(f.backend)
			mcpCompletionRepositoryResult(t, svc, operation)
			if calls != 1 || f.locks != 1 || f.releases != 1 || f.executions != 0 {
				t.Fatal("repository failure escaped exact owned lifetime")
			}
		})
	}
}

func TestMCPCompletionServeDispatchKeepsConstructedService(t *testing.T) {
	for _, transport := range []string{"stdio", "http", "unsupported", "constructor"} {
		t.Run(transport, func(t *testing.T) {
			f := newMCPCompletionFixture(t)
			probe := &mcpCompletionServeProbe{fixture: f, transport: transport, wanted: fault.New(fault.Apply, "owned transport refusal")}
			ops := mcpServeOperations{construct: probe.construct, stdio: probe.stdio, http: probe.http}
			code, err := f.app.serveMCPWithOperations(context.Background(), []string{"serve", "--transport", transport}, []model.Policy{f.policy}, ops)
			probe.assertResult(code, err)
		})
	}
}

type mcpCompletionServeProbe struct {
	fixture     *mcpCompletionFixture
	transport   string
	wanted      error
	constructed int
	served      int
	actual      *cqmcp.Service
}

func (p *mcpCompletionServeProbe) construct(policy model.Policy, b cqmcp.Backend, options cqmcp.Options) (*cqmcp.Service, error) {
	p.fixture.t.Helper()
	p.constructed++
	if options.PolicyProvider == nil || options.ActionAllowed == nil || options.DPoPStateDirectory != p.fixture.app.directory {
		p.fixture.t.Fatal("constructor lost owned policy/state hooks")
	}
	if p.transport == "constructor" {
		return nil, p.wanted
	}
	service, err := cqmcp.New(policy, b, options)
	p.actual = service
	return service, err
}

func (p *mcpCompletionServeProbe) stdio(ctx context.Context, s *cqmcp.Service) error {
	p.fixture.t.Helper()
	if ctx.Err() != nil || s == nil || s != p.actual {
		p.fixture.t.Fatal("stdio received a different service")
	}
	p.served++
	return p.wanted
}

func (p *mcpCompletionServeProbe) http(ctx context.Context, address string, s *cqmcp.Service, options mcpServeOptions) (int, error) {
	p.fixture.t.Helper()
	if ctx.Err() != nil || address != p.fixture.policy.MCP.Address || s != p.actual || options.transport != "http" {
		p.fixture.t.Fatal("HTTP dispatch lost selected values")
	}
	p.served++
	return 7, p.wanted
}

func (p *mcpCompletionServeProbe) assertResult(code int, err error) {
	p.fixture.t.Helper()
	if p.constructed != 1 {
		p.fixture.t.Fatal("service construction count changed")
	}
	if p.transport == "unsupported" {
		if !fault.Is(err, fault.Invalid) || p.served != 0 {
			p.fixture.t.Fatal("unknown transport reached I/O")
		}
		return
	}
	p.assertSelectedResult(code, err)
}

func (p *mcpCompletionServeProbe) assertSelectedResult(code int, err error) {
	p.fixture.t.Helper()
	wantCode, wantServed := 0, 1
	if p.transport == "http" {
		wantCode = 7
	}
	if p.transport == "constructor" {
		wantServed = 0
	}
	if err != p.wanted || code != wantCode || p.served != wantServed {
		p.fixture.t.Fatal("dispatch lost exact inner code/error/lifetime")
	}
}

func TestMCPCompletionHTTPAdmissionAndListenerLifecycle(t *testing.T) {
	for _, scenario := range []string{"handler", "address", "tls", "default", "closed", "failure", "public-tls"} {
		t.Run(scenario, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				mcpCompletionHTTPCase(t, scenario)
			})
		})
	}
}

func mcpCompletionHTTPCase(t *testing.T, scenario string) {
	t.Helper()
	address, options := mcpCompletionHTTPOptions(scenario)
	calls := 0
	var child context.Context
	listen := func(server *http.Server, got mcpServeOptions) error {
		calls++
		mcpCompletionServer(t, server, got, options, address)
		child = server.BaseContext(nil)
		return mcpCompletionListenerResult(scenario)
	}
	handle := func(*cqmcp.Service) (http.Handler, error) { return mcpCompletionHTTPHandler(t, scenario) }
	code, err := runMCPHTTPWithOperations(context.Background(), address, nil, options, handle, listen)
	mcpCompletionHTTPResult(t, scenario, code, err, calls, child)
}

func mcpCompletionHTTPOptions(scenario string) (string, mcpServeOptions) {
	address := ""
	options := mcpServeOptions{}
	if scenario == "address" {
		address = "not-an-address"
	}
	if scenario == "tls" || scenario == "public-tls" {
		address = "0.0.0.0:8443"
	}
	if scenario == "public-tls" {
		options.cert, options.key = "owned-cert", "owned-key"
	}
	return address, options
}

func mcpCompletionListenerResult(scenario string) error {
	if scenario == "closed" {
		return http.ErrServerClosed
	}
	if scenario == "failure" {
		return errors.New("PRIVATE_LISTENER_DETAIL")
	}
	return nil
}

func mcpCompletionHTTPHandler(t *testing.T, scenario string) (http.Handler, error) {
	t.Helper()
	if scenario == "handler" {
		return nil, fault.New(fault.Trust, "owned handler refusal")
	}
	return http.HandlerFunc(func(http.ResponseWriter, *http.Request) { t.Fatal("no request is issued") }), nil
}

func mcpCompletionHTTPResult(t *testing.T, scenario string, code int, err error, calls int, child context.Context) {
	t.Helper()
	if code != 0 {
		t.Fatal("listener changed inner exit code")
	}
	if scenario == "handler" || scenario == "address" || scenario == "tls" {
		if err == nil || calls != 0 {
			t.Fatal("HTTP refusal reached listener")
		}
		return
	}
	mcpCompletionCompletedListener(t, scenario, err, calls, child)
}

func mcpCompletionCompletedListener(t *testing.T, scenario string, err error, calls int, child context.Context) {
	t.Helper()
	if scenario == "failure" {
		requireMCPBoundaryFault(t, err, fault.Apply, "mcp listener")
	} else if err != nil {
		t.Fatal(err)
	}
	if calls != 1 || child == nil || !errors.Is(child.Err(), context.Canceled) {
		t.Fatal("listener did not cancel its owned child")
	}
	synctest.Wait()
}

func mcpCompletionServer(t *testing.T, s *http.Server, got, want mcpServeOptions, address string) {
	t.Helper()
	if address == "" {
		address = "127.0.0.1:8080"
	}
	if s.Addr != address || got != want || s.Handler == nil || s.BaseContext(nil).Err() != nil {
		t.Fatal("server lost configuration or live parent")
	}
	mcpCompletionServerBounds(t, s)
}

func TestMCPCompletionTLSSelectionNeverOpensSocket(t *testing.T) {
	for _, options := range []mcpServeOptions{{}, {cert: "cert"}, {key: "key"}, {cert: "cert", key: "key"}} {
		mcpCompletionTLSCase(t, options)
	}
}

func mcpCompletionTLSCase(t *testing.T, options mcpServeOptions) {
	t.Helper()
	server := &http.Server{}
	plain, encrypted := 0, 0
	wanted := errors.New("owned listener result")
	err := listenMCPServerWithOperations(server, options,
		func(s *http.Server) error {
			if s != server {
				t.Fatal("plain server changed")
			}
			plain++
			return wanted
		},
		func(s *http.Server, cert, key string) error {
			if s != server || cert != options.cert || key != options.key {
				t.Fatal("TLS arguments changed")
			}
			encrypted++
			return wanted
		})
	if err != wanted || plain+encrypted != 1 {
		t.Fatal("listener result/count changed")
	}
	if (options.cert != "" && options.key != "") != (encrypted == 1) {
		t.Fatal("TLS pair selection changed")
	}
}

func mcpCompletionExecutionIO(t *testing.T, override []string, input io.Reader, output, diagnostics io.Writer, releases int) {
	t.Helper()
	if override != nil || input != nil || output != io.Discard || diagnostics != io.Discard || releases != 1 {
		t.Fatal("execution preceded unlock or gained an I/O capability")
	}
}

func mcpCompletionStatusResult(t *testing.T, scenario, want string, result cqmcp.Status, err error) {
	t.Helper()
	if scenario == "corrupt" || scenario == "store" {
		if !fault.Is(err, mcpCompletionStatusKind(scenario)) || result != (cqmcp.Status{}) {
			t.Fatal("state refusal released a partial DTO")
		}
	} else if err != nil || result != (cqmcp.Status{Workload: "demo", Mode: "offline", State: want}) {
		t.Fatalf("actual state result = %#v, %v", result, err)
	}
}

func mcpCompletionReadFailure(t *testing.T, f *mcpCompletionFixture, scenario string) {
	t.Helper()
	if scenario == "engine" {
		f.backend.app.directory = ""
		f.backend.app.paths = mcpCompletionUnavailableDirectory()
	}
	if scenario == "lock" {
		f.backend.acquire = func(*resolve.Engine, string) (func() error, error) {
			return nil, fault.New(fault.Unavailable, "owned lock refusal")
		}
	}
	if scenario == "source" {
		if err := os.Remove(f.policy.Sources["fixture"].Path); err != nil {
			t.Fatal(err)
		}
	}
}

func mcpCompletionRunResult(t *testing.T, f *mcpCompletionFixture, scenario string, result cqmcp.RunResult, err error) {
	t.Helper()
	if scenario == "success" {
		if err != nil || result.Status != "completed" || len(result.ID) != 32 || f.executions != 1 {
			t.Fatalf("finite execution = %#v, %v", result, err)
		}
	} else if !fault.Is(err, mcpCompletionRunKind(scenario)) || result != (cqmcp.RunResult{}) {
		t.Fatal("failed action released a job DTO")
	}
}

func mcpCompletionRepositoryResult(t *testing.T, svc *cqmcp.Service, operation string) {
	t.Helper()
	if operation == "read" {
		result, err := svc.ReadSecret(context.Background(), cqmcp.Principal{Subject: "local:alice"}, "demo", "TOKEN")
		if !fault.Is(err, fault.Trust) || !reflect.DeepEqual(result, cqmcp.SecretResult{}) {
			t.Fatal("repository refusal released value")
		}
	} else {
		result, err := svc.RunWorkload(context.Background(), cqmcp.Principal{Subject: "local:alice"}, "demo")
		if !fault.Is(err, fault.Trust) || result != (cqmcp.RunResult{}) {
			t.Fatal("repository refusal released action")
		}
	}
}

func mcpCompletionServerBounds(t *testing.T, s *http.Server) {
	t.Helper()
	if s.ReadHeaderTimeout != 5*time.Second || s.ReadTimeout != 30*time.Second || s.WriteTimeout != 65*time.Second || s.IdleTimeout != 30*time.Second || s.MaxHeaderBytes != 32<<10 {
		t.Fatal("HTTP bounds changed")
	}
}

func TestMCPCompletionDefaultServeRejectsUnknownTransportBeforeIO(t *testing.T) {
	f := newMCPCompletionFixture(t)
	code, err := f.app.serveMCP(context.Background(), []string{"serve", "--transport", "unsupported"}, f.policy)
	requireMCPBoundaryFault(t, err, fault.Invalid, "mcp transport")
	if code != 0 {
		t.Fatal("inner refusal changed its code")
	}
	if _, err := os.Lstat(f.app.directory); !os.IsNotExist(err) {
		t.Fatal("default construction created state despite no replay authority")
	}
}

func mcpCompletionReadKind(scenario string) fault.Kind {
	if scenario == "engine" {
		return fault.Apply
	}
	if scenario == "lock" {
		return fault.Unavailable
	}
	return fault.Missing
}

func mcpCompletionStatusKind(scenario string) fault.Kind {
	if scenario == "store" {
		return fault.Apply
	}
	return fault.Trust
}

func mcpCompletionRunKind(scenario string) fault.Kind {
	if scenario == "lock" {
		return fault.Unavailable
	}
	if scenario == "resolve" {
		return fault.Missing
	}
	return fault.Apply
}
