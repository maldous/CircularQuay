// Package cli implements the non-secret command interface.
package cli

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/maldous/CircularQuay/internal/fault"
	"github.com/maldous/CircularQuay/internal/model"
	"github.com/maldous/CircularQuay/internal/observability"
	"github.com/maldous/CircularQuay/internal/policy"
	"github.com/maldous/CircularQuay/internal/resolve"
	"github.com/maldous/CircularQuay/internal/source"
	"github.com/maldous/CircularQuay/internal/state"
)

// Version and Commit are set by reproducible release builds.
var Version = "1.0.0-dev"
var Commit = "unknown"

// Runner explicitly owns command I/O; errors never include raw provider text.
type Runner struct {
	In       io.Reader
	Out, Err io.Writer
	// BuildVersion and BuildCommit identify the embedding artifact. Empty
	// values use the native binary's reproducible build values. SDK callers
	// must supply only trusted module build metadata, never environment input.
	BuildVersion string
	BuildCommit  string
}

type application struct {
	config         string
	configExplicit bool
	directory      string
	runner         Runner
	repository     repositoryOperations
	buildVersion   string
	buildCommit    string
}

// Run executes a command and returns a stable CLI or child status.
func (r Runner) Run(ctx context.Context, args []string) int {
	if r.In == nil {
		r.In = os.Stdin
	}
	if r.Out == nil {
		r.Out = os.Stdout
	}
	if r.Err == nil {
		r.Err = os.Stderr
	}
	f := flag.NewFlagSet("cquay", flag.ContinueOnError)
	f.SetOutput(io.Discard)
	config := f.String("config", "cquay.json", "protected policy path")
	stateDir := f.String("state-dir", "", "private metadata directory")
	if err := f.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return r.usage()
		}
		return r.failure(fault.New(fault.Invalid, "arguments"))
	}
	commands := f.Args()
	if len(commands) == 0 {
		return r.usage()
	}
	a := application{config: *config, configExplicit: flagWasSet(f, "config"), directory: *stateDir, runner: r, buildVersion: r.buildVersion(), buildCommit: r.buildCommit()}
	code, err := a.dispatch(ctx, commands)
	if err != nil {
		return r.failure(err)
	}
	return code
}

func (r Runner) buildVersion() string {
	if r.BuildVersion != "" {
		return r.BuildVersion
	}
	return Version
}

func (r Runner) buildCommit() string {
	if r.BuildCommit != "" {
		return r.BuildCommit
	}
	return Commit
}

func (a application) version() string {
	if a.buildVersion != "" {
		return a.buildVersion
	}
	return Version
}

func (a application) commit() string {
	if a.buildCommit != "" {
		return a.buildCommit
	}
	return Commit
}

func (r Runner) failure(err error) int {
	_ = json.NewEncoder(r.Err).Encode(map[string]string{"error": string(fault.Category(err))})
	if fault.Is(err, fault.Invalid) {
		return 2
	}
	if fault.Is(err, fault.Denied) || fault.Is(err, fault.Trust) {
		return 3
	}
	return 1
}

func (r Runner) usage() int {
	fmt.Fprintln(r.Out, "cquay [--config PATH] [--state-dir PATH] <bootstrap|flow|init|config|doctor|plan|run|sync|snapshot|apply|status|update|providers|mcp|version>")
	return 0
}

func (a application) dispatch(ctx context.Context, args []string) (int, error) {
	switch args[0] {
	case "help", "--help", "-h":
		return a.runner.usage(), nil
	case "version":
		return a.print(map[string]string{"version": a.version(), "commit": a.commit(), "go": "1.27", "maturity": "development"})
	case "providers":
		return a.print(source.Capabilities())
	case "bootstrap":
		return a.bootstrapConfig(ctx, args[1:])
	case "flow":
		return a.flow(ctx, args[1:])
	}
	return a.dispatchConfigured(ctx, args)
}

func (a application) dispatchConfigured(ctx context.Context, args []string) (int, error) {
	if args[0] == "init" || args[0] == "config" || args[0] == "doctor" || args[0] == "update" {
		return a.dispatchConfiguredRuntime(ctx, args)
	}
	p, err := a.preflightOperationalPolicy()
	if err != nil {
		return 0, err
	}
	return a.dispatchLoadedOperationalPolicy(ctx, args, p)
}

// preflightOperationalPolicy loads the selected policy and applies its local
// runtime floor before any operational receipt, telemetry credential, provider
// credential, repository, child-process, or workload effect.
func (a application) preflightOperationalPolicy() (model.Policy, error) {
	p, err := a.load()
	if err != nil {
		return model.Policy{}, err
	}
	if err := enforceMinimumVersion(p, a.version()); err != nil {
		return model.Policy{}, err
	}
	return p, nil
}

func (a application) dispatchLoadedOperationalPolicy(ctx context.Context, args []string, p model.Policy) (int, error) {
	runtime, err := a.newObservability(ctx, p)
	if err != nil {
		return 0, err
	}
	defer func() {
		flush, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = runtime.Shutdown(flush)
	}()
	ctx = observability.WithRuntime(ctx, runtime)
	observability.Record(ctx, observability.Operation{Phase: observability.PhasePolicy, ProviderKind: "json"}, observability.OutcomeSuccess)
	if err := a.checkUpdates(ctx); err != nil {
		return 0, err
	}
	return a.dispatchWithObservability(ctx, args)
}

func (a application) dispatchWithObservability(ctx context.Context, args []string) (code int, err error) {
	ctx, complete := observability.Start(ctx, observability.Operation{Phase: observability.PhaseCLI, ProviderKind: "none"})
	defer func() { complete(observability.OutcomeForError(err)) }()
	return a.dispatchConfiguredRuntime(ctx, args)
}

func (a application) dispatchConfiguredRuntime(ctx context.Context, args []string) (int, error) {
	switch args[0] {
	case "init":
		return a.initialize(args[1:])
	case "doctor":
		return a.doctor(args[1:])
	case "config":
		return a.configCommand(args[1:])
	case "plan":
		return a.plan()
	case "update":
		return a.updateCommand(ctx, args[1:])
	case "status":
		return a.status(args[1:])
	case "run", "sync", "snapshot", "apply":
		return a.workloadCommand(ctx, args)
	case "mcp":
		return a.serveMCP(ctx, args[1:])
	default:
		return 0, fault.New(fault.Invalid, "command")
	}
}

func (a application) newObservability(ctx context.Context, policy model.Policy) (*observability.Runtime, error) {
	directory := ""
	if policy.Observability != nil && policy.Observability.DiagnosticLevel != "off" {
		store, err := a.store()
		if err != nil {
			return nil, err
		}
		directory = store.Directory
	}
	return observability.New(ctx, policy.Observability, observability.Options{DiagnosticDirectory: directory, Version: a.version()})
}

func (a application) print(value any) (int, error) {
	if json.NewEncoder(a.runner.Out).Encode(value) != nil {
		return 0, fault.New(fault.Apply, "output")
	}
	return 0, nil
}

func (a application) load() (model.Policy, error) {
	p, err := filepath.Abs(a.config)
	if err != nil {
		return model.Policy{}, fault.New(fault.Invalid, "policy path")
	}
	return policy.Load(p)
}

func (a application) store() (state.Store, error) {
	dir := a.directory
	if dir == "" {
		home, err := os.UserConfigDir()
		if err != nil {
			return state.Store{}, fault.New(fault.Apply, "state directory")
		}
		legacy := filepath.Join(home, "circularquay", "state")
		if _, err = os.Lstat(legacy); err == nil || !os.IsNotExist(err) {
			return state.Store{}, fault.New(fault.Trust, "legacy state directory")
		}
		dir = filepath.Join(home, "cquay", "state")
	}
	return state.Store{Directory: dir}, nil
}

type snapshotBackend struct{}

func (snapshotBackend) Encrypt(ctx context.Context, s model.Snapshot, v map[string]string) error {
	return source.EncryptSnapshot(ctx, s, v)
}
func (snapshotBackend) Decrypt(ctx context.Context, s model.Snapshot) (map[string]string, error) {
	return source.DecryptSnapshot(ctx, s)
}

func (a application) engine(p model.Policy) (*resolve.Engine, error) {
	s, err := a.store()
	if err != nil {
		return nil, err
	}
	r := source.NewRegistry()
	r.Register("sops", source.SOPSReader{})
	return &resolve.Engine{Policy: p, Reader: r, State: s, Snapshots: snapshotBackend{}}, nil
}
