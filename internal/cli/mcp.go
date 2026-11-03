package cli

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/maldous/CircularQuay/internal/fault"
	cqmcp "github.com/maldous/CircularQuay/internal/mcp"
	"github.com/maldous/CircularQuay/internal/model"
	"github.com/maldous/CircularQuay/internal/platform"
	"github.com/maldous/CircularQuay/internal/policy"
)

func (a application) serveMCP(ctx context.Context, args []string) (int, error) {
	options, err := parseMCPServeOptions(args)
	if err != nil {
		return 0, err
	}
	p, err := a.load()
	if err != nil {
		return 0, err
	}
	b := backend{app: a}
	service, err := cqmcp.New(p, b, cqmcp.Options{PolicyProvider: a.load, ActionAllowed: a.actionAllowed})
	if err != nil {
		return 0, err
	}
	if options.transport == "stdio" {
		return 0, service.RunStdio(ctx)
	}
	if options.transport != "http" {
		return 0, fault.New(fault.Invalid, "mcp transport")
	}
	return runMCPHTTP(ctx, p.MCP.Address, service, options)
}

type mcpServeOptions struct {
	transport string
	cert      string
	key       string
}

func parseMCPServeOptions(args []string) (mcpServeOptions, error) {
	if len(args) == 0 || args[0] != "serve" {
		return mcpServeOptions{}, fault.New(fault.Invalid, "mcp command")
	}
	f := flag.NewFlagSet("mcp serve", flag.ContinueOnError)
	f.SetOutput(io.Discard)
	transport := f.String("transport", "stdio", "stdio or http")
	cert := f.String("tls-cert", "", "TLS certificate path")
	key := f.String("tls-key", "", "TLS private key path")
	if f.Parse(args[1:]) != nil || f.NArg() != 0 {
		return mcpServeOptions{}, fault.New(fault.Invalid, "mcp arguments")
	}
	return mcpServeOptions{transport: *transport, cert: *cert, key: *key}, nil
}

func runMCPHTTP(ctx context.Context, configuredAddress string, service *cqmcp.Service, options mcpServeOptions) (int, error) {
	handler, err := service.HTTPHandler()
	if err != nil {
		return 0, err
	}
	address := configuredAddress
	if address == "" {
		address = "127.0.0.1:8080"
	}
	host, _, err := net.SplitHostPort(address)
	if err != nil {
		return 0, fault.New(fault.Invalid, "mcp address")
	}
	if !loopback(host) && (options.cert == "" || options.key == "") {
		return 0, fault.New(fault.Trust, "public MCP requires TLS")
	}
	s := &http.Server{Addr: address, Handler: handler, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 30 * time.Second, WriteTimeout: 65 * time.Second, IdleTimeout: 30 * time.Second, MaxHeaderBytes: 32 << 10}
	shutdownMCPServerOnCancel(ctx, s)
	err = listenMCPServer(s, options)
	if err != nil && err != http.ErrServerClosed {
		return 0, fault.New(fault.Apply, "mcp listener")
	}
	return 0, nil
}

func shutdownMCPServerOnCancel(ctx context.Context, server *http.Server) {
	go func() {
		<-ctx.Done()
		shutdown, cancel := newMCPShutdownContext(ctx)
		defer cancel()
		_ = server.Shutdown(shutdown)
	}()
}

func newMCPShutdownContext(ctx context.Context) (context.Context, context.CancelFunc) {
	// The initiating request is canceled before shutdown starts. Preserve its
	// context values while removing that cancellation, then bound cleanup so a
	// stuck client cannot leave the shutdown goroutine alive indefinitely.
	return context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
}

func listenMCPServer(server *http.Server, options mcpServeOptions) error {
	if options.cert != "" && options.key != "" {
		return server.ListenAndServeTLS(options.cert, options.key)
	}
	return server.ListenAndServe()
}

func loopback(host string) bool {
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func (a application) actionAllowed(id string) bool {
	p, err := a.load()
	if err != nil {
		return false
	}
	w, ok := p.Workloads[id]
	return ok && actionIdentityAllowed(w)
}

func actionIdentityAllowed(w model.Workload) bool {
	if len(w.Command) == 0 || len(w.CommandSHA256) != 64 {
		return false
	}
	f, err := os.Open(w.Command[0])
	if err != nil {
		return false
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Size() > 128<<20 {
		return false
	}
	h := sha256.New()
	if _, err = io.Copy(h, f); err != nil {
		return false
	}
	return hex.EncodeToString(h.Sum(nil)) == w.CommandSHA256
}

type backend struct {
	app               application
	resolveGeneration func(context.Context, model.Policy, string) (model.Generation, error)
	now               func() time.Time
}

func (b backend) authorized(ctx context.Context, id, op, key string) (model.Policy, error) {
	p, err := b.app.load()
	if err != nil {
		return model.Policy{}, err
	}
	principal, ok := cqmcp.PrincipalFromContext(ctx)
	if !ok || !principal.ValidAt(b.clock()) || !policy.Authorize(p, principal.Subject, principal.Client, id, op, key) {
		return model.Policy{}, fault.New(fault.Denied, "current MCP grant")
	}
	return p, nil
}

func (b backend) clock() time.Time {
	if b.now == nil {
		return time.Now()
	}
	return b.now()
}

func (b backend) Status(ctx context.Context, id string) (cqmcp.Status, error) {
	p, err := b.authorized(ctx, id, "status", "")
	if err != nil {
		return cqmcp.Status{}, err
	}
	s, err := b.app.store()
	if err != nil {
		return cqmcp.Status{}, err
	}
	r, err := s.Read(id)
	if err != nil {
		return cqmcp.Status{}, err
	}
	return cqmcp.Status{Workload: id, Mode: p.Workloads[id].Mode, State: r.State}, nil
}

func (b backend) Read(ctx context.Context, id, key string) (model.Value, error) {
	p, err := b.authorized(ctx, id, "read", key)
	if err != nil {
		return model.Value{}, err
	}
	e, err := b.app.engine(p)
	if err != nil {
		return model.Value{}, err
	}
	return e.ReadKey(ctx, id, key)
}

func (b backend) Run(ctx context.Context, id string) (cqmcp.RunResult, error) {
	p, workload, err := b.authorizeAction(ctx, id)
	if err != nil {
		return cqmcp.RunResult{}, err
	}
	scope, err := actionScopeFingerprint(p, workload)
	if err != nil {
		return cqmcp.RunResult{}, err
	}
	g, err := b.resolve(ctx, p, id)
	if err != nil {
		return cqmcp.RunResult{}, err
	}
	workload, err = b.authorizeResolvedAction(ctx, id, scope)
	if err != nil {
		return cqmcp.RunResult{}, err
	}
	runCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	code, err := platform.RunWithIO(runCtx, workload, g, nil, nil, io.Discard, io.Discard)
	if err != nil {
		return cqmcp.RunResult{}, err
	}
	return actionRunResult(code)
}

func (b backend) resolve(ctx context.Context, p model.Policy, id string) (model.Generation, error) {
	if b.resolveGeneration != nil {
		return b.resolveGeneration(ctx, p, id)
	}
	e, err := b.app.engine(p)
	if err != nil {
		return model.Generation{}, err
	}
	return e.Resolve(ctx, id)
}

func (b backend) authorizeResolvedAction(ctx context.Context, id string, before [sha256.Size]byte) (model.Workload, error) {
	p, workload, err := b.authorizeAction(ctx, id)
	if err != nil {
		return model.Workload{}, err
	}
	after, err := actionScopeFingerprint(p, workload)
	if err != nil || before != after {
		return model.Workload{}, fault.New(fault.Denied, "MCP action policy changed")
	}
	return workload, nil
}

func (b backend) authorizeAction(ctx context.Context, id string) (model.Policy, model.Workload, error) {
	p, err := b.authorized(ctx, id, "run", "")
	if err != nil {
		return model.Policy{}, model.Workload{}, err
	}
	workload, ok := p.Workloads[id]
	if !ok || !actionIdentityAllowed(workload) {
		return model.Policy{}, model.Workload{}, fault.New(fault.Denied, "action identity")
	}
	if err := b.authorizeUntrustedCode(ctx, id, workload); err != nil {
		return model.Policy{}, model.Workload{}, err
	}
	return p, workload, nil
}

// actionScopeFingerprint binds a resolved generation to the workload and only
// the source declarations from which its bindings can be resolved. Grants are
// reauthorized separately so unrelated policy changes do not cancel a run.
func actionScopeFingerprint(p model.Policy, workload model.Workload) ([sha256.Size]byte, error) {
	sources := make(map[string]model.Source, len(workload.Bindings))
	for _, binding := range workload.Bindings {
		source, ok := p.Sources[binding.Source]
		if !ok {
			return [sha256.Size]byte{}, fault.New(fault.Invalid, "action source")
		}
		sources[binding.Source] = source
	}
	payload, err := json.Marshal(struct {
		Workload model.Workload          `json:"workload"`
		Sources  map[string]model.Source `json:"sources"`
	}{Workload: workload, Sources: sources})
	if err != nil {
		return [sha256.Size]byte{}, fault.New(fault.Invalid, "action scope")
	}
	return sha256.Sum256(payload), nil
}

func (b backend) authorizeUntrustedCode(ctx context.Context, id string, workload model.Workload) error {
	if !workload.UntrustedCode {
		return nil
	}
	for _, binding := range workload.Bindings {
		if _, err := b.authorized(ctx, id, "read", binding.Name); err != nil {
			return err
		}
	}
	return nil
}

func actionRunResult(code int) (cqmcp.RunResult, error) {
	job := make([]byte, 16)
	if _, err := rand.Read(job); err != nil {
		return cqmcp.RunResult{}, fault.New(fault.Apply, "run identifier")
	}
	status := "completed"
	if code != 0 {
		status = "failed"
	}
	return cqmcp.RunResult{ID: hex.EncodeToString(job), Status: status, ExitCode: code}, nil
}
