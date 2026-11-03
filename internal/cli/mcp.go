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
	"github.com/maldous/CircularQuay/internal/resolve"
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
	// Listener errors must release the shutdown waiter even when the caller's
	// context remains live. The child also retains ordinary request cancellation.
	listenerContext, cancel := context.WithCancel(ctx)
	defer cancel()
	s := configuredMCPHTTPServer(listenerContext, address, handler)
	shutdownMCPServerOnCancel(listenerContext, s)
	err = listenMCPServer(s, options)
	if err != nil && err != http.ErrServerClosed {
		return 0, fault.New(fault.Apply, "mcp listener")
	}
	return 0, nil
}

func configuredMCPHTTPServer(listenerContext context.Context, address string, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:              address,
		Handler:           handler,
		BaseContext:       func(net.Listener) context.Context { return listenerContext },
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      65 * time.Second,
		IdleTimeout:       30 * time.Second,
		MaxHeaderBytes:    32 << 10,
	}
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
	if ctx.Err() != nil {
		return model.Policy{}, fault.New(fault.Denied, "MCP request cancelled")
	}
	p, err := b.app.load()
	if err != nil {
		return model.Policy{}, err
	}
	principal, ok := cqmcp.PrincipalFromContext(ctx)
	if !ok || !principal.Authorize(p, id, op, key, b.clock()) {
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
	readPolicy, workload, err := selectedReadPolicy(p, id, key)
	if err != nil {
		return model.Value{}, err
	}
	scope, err := actionScopeFingerprint(readPolicy, workload)
	if err != nil {
		return model.Value{}, err
	}
	engine, unlock, err := b.repositoryEngine(ctx, readPolicy, workload, id)
	if err != nil {
		return model.Value{}, err
	}
	defer func() { _ = unlock() }()
	if err := b.reauthorizeRead(ctx, id, key, scope); err != nil {
		return model.Value{}, err
	}
	return engine.ReadKey(ctx, id, key)
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
	engine, unlock, err := b.repositoryEngine(ctx, p, workload, id)
	if err != nil {
		return cqmcp.RunResult{}, err
	}
	defer func() {
		if unlock != nil {
			_ = unlock()
		}
	}()
	if _, err = b.authorizeResolvedAction(ctx, id, scope); err != nil {
		return cqmcp.RunResult{}, err
	}
	g, err := b.resolve(ctx, engine, id)
	if err != nil {
		return cqmcp.RunResult{}, err
	}
	workload, err = b.authorizeResolvedAction(ctx, id, scope)
	if err != nil {
		return cqmcp.RunResult{}, err
	}
	if err = unlock(); err != nil {
		return cqmcp.RunResult{}, err
	}
	unlock = nil
	runCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	code, err := platform.RunWithIO(runCtx, workload, g, nil, nil, io.Discard, io.Discard)
	if err != nil {
		return cqmcp.RunResult{}, err
	}
	return actionRunResult(code)
}

func (b backend) repositoryEngine(ctx context.Context, config model.Policy, workload model.Workload, id string) (*resolve.Engine, func() error, error) {
	engine, err := b.app.engine(config)
	if err != nil {
		return nil, nil, err
	}
	unlock, err := engine.State.Lock(id)
	if err != nil {
		return nil, nil, err
	}
	engine, err = b.app.withRepositorySources(ctx, workload, engine)
	if err != nil {
		_ = unlock()
		return nil, nil, err
	}
	return engine, unlock, nil
}

func (b backend) resolve(ctx context.Context, engine *resolve.Engine, id string) (model.Generation, error) {
	if b.resolveGeneration != nil {
		return b.resolveGeneration(ctx, engine.Policy, id)
	}
	return engine.Resolve(ctx, id)
}

func (b backend) reauthorizeRead(ctx context.Context, id, key string, before [sha256.Size]byte) error {
	p, err := b.authorized(ctx, id, "read", key)
	if err != nil {
		return err
	}
	readPolicy, workload, err := selectedReadPolicy(p, id, key)
	if err != nil {
		return err
	}
	after, err := actionScopeFingerprint(readPolicy, workload)
	if err != nil || before != after {
		return fault.New(fault.Denied, "MCP read policy changed")
	}
	return nil
}

func selectedReadPolicy(config model.Policy, id, key string) (model.Policy, model.Workload, error) {
	workload, found := config.Workloads[id]
	if !found {
		return model.Policy{}, model.Workload{}, fault.New(fault.Invalid, "workload")
	}
	for _, binding := range workload.Bindings {
		if binding.Name != key {
			continue
		}
		// Snapshot restoration verifies and decrypts one complete workload
		// generation. Keep every binding in its fingerprint and declaration;
		// ReadKey still returns only the authorized requested value. Other read
		// modes retain the narrow declaration so they cannot read sibling keys.
		if workload.Mode == "offline" && workload.Snapshot.Path != "" {
			return config, workload, nil
		}
		workload.Bindings = []model.Binding{binding}
		updated := config
		updated.Workloads = make(map[string]model.Workload, len(config.Workloads))
		for name, selected := range config.Workloads {
			updated.Workloads[name] = selected
		}
		updated.Workloads[id] = workload
		return updated, workload, nil
	}
	return model.Policy{}, model.Workload{}, fault.New(fault.Denied, "MCP binding")
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
	var repository *model.Repository
	for _, source := range sources {
		if strings.HasPrefix(source.Path, "repo:") {
			selected := p.Repository
			repository = &selected
			break
		}
	}
	payload, err := json.Marshal(struct {
		Workload   model.Workload          `json:"workload"`
		Sources    map[string]model.Source `json:"sources"`
		Repository *model.Repository       `json:"repository,omitempty"`
	}{Workload: workload, Sources: sources, Repository: repository})
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
