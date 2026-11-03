package cli

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
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
	if len(args) == 0 || args[0] != "serve" {
		return 0, fault.New(fault.Invalid, "mcp command")
	}
	f := flag.NewFlagSet("mcp serve", flag.ContinueOnError)
	f.SetOutput(io.Discard)
	transport := f.String("transport", "stdio", "stdio or http")
	cert := f.String("tls-cert", "", "TLS certificate path")
	key := f.String("tls-key", "", "TLS private key path")
	if f.Parse(args[1:]) != nil || f.NArg() != 0 {
		return 0, fault.New(fault.Invalid, "mcp arguments")
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
	if *transport == "stdio" {
		return 0, service.RunStdio(ctx)
	}
	if *transport != "http" {
		return 0, fault.New(fault.Invalid, "mcp transport")
	}
	handler, err := service.HTTPHandler()
	if err != nil {
		return 0, err
	}
	address := p.MCP.Address
	if address == "" {
		address = "127.0.0.1:8080"
	}
	host, _, err := net.SplitHostPort(address)
	if err != nil {
		return 0, fault.New(fault.Invalid, "mcp address")
	}
	if !loopback(host) && (*cert == "" || *key == "") {
		return 0, fault.New(fault.Trust, "public MCP requires TLS")
	}
	s := &http.Server{Addr: address, Handler: handler, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 30 * time.Second, WriteTimeout: 65 * time.Second, IdleTimeout: 30 * time.Second, MaxHeaderBytes: 32 << 10}
	go func() {
		<-ctx.Done()
		shutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = s.Shutdown(shutdown)
	}()
	if *cert != "" && *key != "" {
		err = s.ListenAndServeTLS(*cert, *key)
	} else {
		err = s.ListenAndServe()
	}
	if err != nil && err != http.ErrServerClosed {
		return 0, fault.New(fault.Apply, "mcp listener")
	}
	return 0, nil
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
	if !ok || len(w.Command) == 0 || len(w.CommandSHA256) != 64 {
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

type backend struct{ app application }

func (b backend) authorized(ctx context.Context, id, op, key string) (model.Policy, error) {
	p, err := b.app.load()
	if err != nil {
		return model.Policy{}, err
	}
	principal, ok := cqmcp.PrincipalFromContext(ctx)
	if !ok || !policy.Authorize(p, principal.Subject, principal.Client, id, op, key) {
		return model.Policy{}, fault.New(fault.Denied, "current MCP grant")
	}
	return p, nil
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
	p, err := b.authorized(ctx, id, "run", "")
	if err != nil {
		return cqmcp.RunResult{}, err
	}
	if !b.app.actionAllowed(id) {
		return cqmcp.RunResult{}, fault.New(fault.Denied, "action identity")
	}
	w := p.Workloads[id]
	if w.UntrustedCode {
		for _, binding := range w.Bindings {
			if _, err = b.authorized(ctx, id, "read", binding.Name); err != nil {
				return cqmcp.RunResult{}, err
			}
		}
	}
	e, err := b.app.engine(p)
	if err != nil {
		return cqmcp.RunResult{}, err
	}
	g, err := e.Resolve(ctx, id)
	if err != nil {
		return cqmcp.RunResult{}, err
	}
	runCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	code, err := platform.RunWithIO(runCtx, w, g, nil, nil, io.Discard, io.Discard)
	if err != nil {
		return cqmcp.RunResult{}, err
	}
	job := make([]byte, 16)
	if _, err = rand.Read(job); err != nil {
		return cqmcp.RunResult{}, fault.New(fault.Apply, "run identifier")
	}
	status := "completed"
	if code != 0 {
		status = "failed"
	}
	return cqmcp.RunResult{ID: hex.EncodeToString(job), Status: status, ExitCode: code}, nil
}
