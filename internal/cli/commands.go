package cli

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"flag"
	"io"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/maldous/CircularQuay/internal/deliver"
	"github.com/maldous/CircularQuay/internal/fault"
	"github.com/maldous/CircularQuay/internal/model"
	"github.com/maldous/CircularQuay/internal/observability"
	"github.com/maldous/CircularQuay/internal/platform"
	"github.com/maldous/CircularQuay/internal/resolve"
	"github.com/maldous/CircularQuay/internal/state"
)

func (a application) initialize(args []string) (int, error) {
	f := flag.NewFlagSet("init", flag.ContinueOnError)
	f.SetOutput(io.Discard)
	out := f.String("output", a.config, "policy output")
	if f.Parse(args) != nil || f.NArg() != 0 {
		return 0, fault.New(fault.Invalid, "init arguments")
	}
	p, err := filepath.Abs(*out)
	if err != nil {
		return 0, fault.New(fault.Invalid, "init path")
	}
	fixture := filepath.Base(p) + ".fixture.json"
	config := model.Policy{Version: 1, Sources: map[string]model.Source{"demo": {Type: "json", Path: fixture}}, Workloads: map[string]model.Workload{"demo": {Mode: "offline", Bindings: []model.Binding{{Name: "DEMO_API_KEY", Source: "demo", Field: "DEMO_API_KEY", AllowOffline: true}}}}}
	b, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return 0, fault.New(fault.Apply, "init encode")
	}
	if _, err = os.Lstat(p); !os.IsNotExist(err) {
		return 0, fault.New(fault.Trust, "existing policy")
	}
	fixturePath := filepath.Join(filepath.Dir(p), fixture)
	if _, err = os.Lstat(fixturePath); !os.IsNotExist(err) {
		return 0, fault.New(fault.Trust, "existing fixture")
	}
	if err = os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
		return 0, fault.New(fault.Apply, "init directory")
	}
	if err = writeNew(fixturePath, []byte("{\"DEMO_API_KEY\":\"synthetic-only-not-a-credential\"}\n")); err != nil {
		return 0, err
	}
	if err = writeNew(p, b); err != nil {
		return 0, err
	}
	return a.print(map[string]string{"state": "initialized", "policy": p, "credentials": "synthetic fixture only"})
}

func writeNew(path string, data []byte) error {
	return platform.CreateProtectedExclusive(path, data)
}

func (a application) plan(p model.Policy) (int, error) {
	type entry struct {
		ID       string `json:"id"`
		Mode     string `json:"mode"`
		Bindings int    `json:"bindings"`
		Targets  int    `json:"targets"`
	}
	entries := []entry{}
	for id, w := range p.Workloads {
		entries = append(entries, entry{id, w.Mode, len(w.Bindings), len(w.Targets)})
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].ID < entries[j].ID })
	return a.print(map[string]any{"valid": true, "value_requests": 0, "workloads": entries})
}

func workloadFlags(command string, args []string) (string, string, []string, error) {
	f := flag.NewFlagSet(command, flag.ContinueOnError)
	f.SetOutput(io.Discard)
	id := f.String("workload", "", "workload ID")
	target := f.String("target", "", "target ID")
	if command == "sync" {
		_ = f.Bool("once", true, "perform one synchronization; schedule through native facilities")
	}
	if f.Parse(args) != nil || *id == "" {
		return "", "", nil, fault.New(fault.Invalid, "workload arguments")
	}
	return *id, *target, f.Args(), nil
}

func (a application) status(args []string, p model.Policy) (int, error) {
	f := flag.NewFlagSet("status", flag.ContinueOnError)
	f.SetOutput(io.Discard)
	id := f.String("workload", "", "workload ID")
	_ = f.Bool("json", false, "JSON output")
	if f.Parse(args) != nil || *id == "" || f.NArg() != 0 {
		return 0, fault.New(fault.Invalid, "status arguments")
	}
	if _, ok := p.Workloads[*id]; !ok {
		return 0, fault.New(fault.Invalid, "workload")
	}
	s, err := a.store()
	if err != nil {
		return 0, err
	}
	r, err := s.Read(*id)
	if err != nil {
		return 0, err
	}
	return a.print(r)
}

func (a application) workloadCommand(ctx context.Context, args []string, loaded ...model.Policy) (code int, err error) {
	ctx, complete := observability.Start(ctx, observability.Operation{Phase: observability.PhaseWorkload, ProviderKind: "native"})
	defer func() { complete(observability.OutcomeForError(err)) }()
	p, err := a.commandPolicy(ctx, loaded)
	if err != nil {
		return 0, err
	}
	request, err := parseWorkloadRequest(args)
	if err != nil {
		return 0, err
	}
	request, workload, engine, err := a.prepareWorkloadCommand(p, request)
	if err != nil {
		return 0, err
	}
	unlock, err := engine.State.Lock(request.id)
	if err != nil {
		return 0, err
	}
	defer func() {
		if unlock != nil {
			_ = unlock()
		}
	}()
	engine, err = a.withRepositorySources(ctx, workload, engine)
	if err != nil {
		return 0, err
	}
	g, err := resolveWorkloadGeneration(ctx, engine, request)
	if err != nil {
		return 0, err
	}
	if request.command == "run" {
		return a.runResolvedWorkload(ctx, engine, workload, g, request.override, &unlock)
	}
	return a.completeWorkloadCommand(ctx, request, workload, engine, g)
}

func (a application) commandPolicy(ctx context.Context, loaded []model.Policy) (model.Policy, error) {
	if len(loaded) == 1 {
		return loaded[0], nil
	}
	if len(loaded) != 0 {
		return model.Policy{}, fault.New(fault.Invalid, "policy snapshot")
	}
	return a.preflightOperationalPolicy(ctx)
}

type workloadRequest struct {
	command  string
	id       string
	target   string
	override []string
	targets  []string
}

func parseWorkloadRequest(args []string) (workloadRequest, error) {
	if len(args) == 0 {
		return workloadRequest{}, fault.New(fault.Invalid, "workload command")
	}
	command := args[0]
	id, target, override, err := workloadFlags(command, args[1:])
	if err != nil {
		return workloadRequest{}, err
	}
	if command != "run" && len(override) > 0 {
		return workloadRequest{}, fault.New(fault.Invalid, "unexpected arguments")
	}
	return workloadRequest{command: command, id: id, target: target, override: override}, nil
}

func (a application) prepareWorkloadCommand(p model.Policy, request workloadRequest) (workloadRequest, model.Workload, *resolve.Engine, error) {
	workload, ok := p.Workloads[request.id]
	if !ok {
		return request, model.Workload{}, nil, fault.New(fault.Invalid, "workload")
	}
	if request.command == "run" {
		// Reject command-identity changes before state, repository, or provider
		// resolution. launchInputs repeats this immediately before child launch.
		if _, err := platform.ValidateCommandIdentity(workload, request.override); err != nil {
			return request, model.Workload{}, nil, err
		}
	}
	if request.command == "apply" && request.target == "" {
		return request, model.Workload{}, nil, fault.New(fault.Invalid, "explicit target required")
	}
	if request.command == "apply" || request.command == "sync" {
		var err error
		request.targets, err = selectedTargets(workload, request.target)
		if err != nil {
			return request, model.Workload{}, nil, err
		}
	}
	engine, err := a.engine(p)
	if err != nil {
		return request, model.Workload{}, nil, err
	}
	return request, workload, engine, nil
}

func resolveWorkloadGeneration(ctx context.Context, engine *resolve.Engine, request workloadRequest) (model.Generation, error) {
	if request.command == "snapshot" || request.command == "sync" {
		return engine.ResolveFresh(ctx, request.id)
	}
	return engine.Resolve(ctx, request.id)
}

func (a application) runResolvedWorkload(ctx context.Context, engine *resolve.Engine, workload model.Workload, generation model.Generation, override []string, unlock *func() error) (int, error) {
	if err := recordPhase(engine.State, workload, generation, "resolved"); err != nil {
		return 0, err
	}
	// Only reconciliation is serialized. A child may run indefinitely while
	// subsequent checks and rotations need to acquire the same workload lock.
	if err := (*unlock)(); err != nil {
		return 0, err
	}
	*unlock = nil
	return platform.RunWithIO(ctx, workload, generation, override, a.runner.In, a.runner.Out, a.runner.Err)
}

func (a application) completeWorkloadCommand(ctx context.Context, request workloadRequest, workload model.Workload, engine *resolve.Engine, generation model.Generation) (int, error) {
	switch request.command {
	case "snapshot":
		return a.storeSnapshot(ctx, request, workload, engine, generation)
	case "apply":
		return a.reconcileWorkload(ctx, workload, engine, generation, request.targets)
	case "sync":
		return a.syncWorkload(ctx, request, workload, engine, generation)
	default:
		return 0, fault.New(fault.Invalid, "workload command")
	}
}

func (a application) storeSnapshot(ctx context.Context, request workloadRequest, workload model.Workload, engine *resolve.Engine, generation model.Generation) (int, error) {
	if err := engine.Snapshot(ctx, generation); err != nil {
		return 0, err
	}
	if err := recordPhase(engine.State, workload, generation, "snapshotted"); err != nil {
		return 0, err
	}
	return a.print(map[string]any{"workload": request.id, "state": "snapshotted", "operation": request.command, "offline": generation.Offline, "bindings": len(generation.Values)})
}

func (a application) reconcileWorkload(ctx context.Context, workload model.Workload, engine *resolve.Engine, generation model.Generation, targets []string) (int, error) {
	record, err := reconcileTargets(ctx, engine.State, workload, generation, targets, nativeOperations())
	if err != nil {
		return 0, err
	}
	return a.print(record)
}

func (a application) syncWorkload(ctx context.Context, request workloadRequest, workload model.Workload, engine *resolve.Engine, generation model.Generation) (int, error) {
	if workload.Snapshot.Path != "" && workload.Mode != "online" {
		if err := engine.Snapshot(ctx, generation); err != nil {
			return 0, err
		}
	}
	return a.reconcileWorkload(ctx, workload, engine, generation, request.targets)
}

func recordPhase(s state.Store, w model.Workload, g model.Generation, phase string) error {
	record, err := generationRecord(w, g, phase)
	if err != nil {
		return err
	}
	return s.Write(record)
}

func generationRecord(w model.Workload, g model.Generation, phase string) (state.Record, error) {
	id := make([]byte, 16)
	if _, err := rand.Read(id); err != nil {
		return state.Record{}, fault.New(fault.Apply, "generation identifier")
	}
	return state.Record{Workload: g.Workload, Generation: hex.EncodeToString(id), State: phase, Health: "unknown", Mode: w.Mode, Offline: g.Offline, LastChecked: time.Now().UTC(), LastSuccess: g.RetrievedAt, Receipts: g.Receipts}, nil
}

func apply(ctx context.Context, w model.Workload, id string, g model.Generation) error {
	t, ok := w.Targets[id]
	if !ok {
		return fault.New(fault.Denied, "target mapping")
	}
	for _, b := range w.Bindings {
		if !b.AllowPublish {
			return fault.New(fault.Denied, "publication permission")
		}
	}
	switch t.Type {
	case "file", "systemd", "windows":
		return platform.Apply(ctx, t, g)
	default:
		return deliver.Apply(ctx, t, g)
	}
}
