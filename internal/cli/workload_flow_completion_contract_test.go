package cli

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/maldous/CircularQuay/internal/bootstrap"
	"github.com/maldous/CircularQuay/internal/fault"
	"github.com/maldous/CircularQuay/internal/model"
	"github.com/maldous/CircularQuay/internal/observability"
	"github.com/maldous/CircularQuay/internal/platform"
	"github.com/maldous/CircularQuay/internal/resolve"
	"github.com/maldous/CircularQuay/internal/state"
)

func TestCLICompletionWorkloadPipelineStopsAndUnlocksInOrder(t *testing.T) {
	for _, phase := range []string{"engine", "lock", "repository", "resolve", "complete"} {
		t.Run(phase, func(t *testing.T) { cliCompletionWorkloadFailure(t, phase) })
	}
}

func cliCompletionWorkloadFailure(t *testing.T, phase string) {
	t.Helper()
	p := &cliCompletionPipelineProbe{t: t, phase: phase, refusal: fault.New(fault.Apply, "owned workload "+phase)}
	p.policy = model.Policy{Version: 1, Workloads: map[string]model.Workload{"demo": {Mode: "offline"}}}
	p.engine = &resolve.Engine{Policy: p.policy}
	p.generation = model.Generation{Workload: "demo"}
	ops := workloadCommandOperations{p.construct, p.lock, p.repository, p.resolve, p.run, p.complete}
	code, err := (application{}).workloadCommandWithOperations(context.Background(), []string{"snapshot", "--workload", "demo"}, []model.Policy{p.policy}, ops)
	expected := map[string][]string{
		"engine": {"engine"}, "lock": {"engine", "lock"},
		"repository": {"engine", "lock", "repository", "unlock"},
		"resolve":    {"engine", "lock", "repository", "resolve", "unlock"},
		"complete":   {"engine", "lock", "repository", "resolve", "complete", "unlock"},
	}[phase]
	if code != 0 || err != p.refusal || !reflect.DeepEqual(p.calls, expected) {
		t.Fatalf("pipeline order/refusal changed: %v %v", p.calls, err)
	}
}

func TestCLICompletionRunReleasesBeforeFiniteChildAndRetainsFailure(t *testing.T) {
	for _, phase := range []string{"record", "unlock", "run"} {
		t.Run(phase, func(t *testing.T) {
			cliCompletionWorkloadFlowCompletionCase1(t, phase)
		})
	}
}

func TestCLICompletionSnapshotCommitAndSyncFailuresStopLaterEffects(t *testing.T) {
	for _, phase := range []string{"snapshot", "record", "output"} {
		t.Run(phase, func(t *testing.T) {
			cliCompletionWorkloadFlowCompletionCase2(t, phase)
		})
	}
	refusal := fault.New(fault.Apply, "owned snapshot")
	code, err := (application{}).syncWorkloadWithOperations(context.Background(), workloadRequest{}, model.Workload{Mode: "hybrid", Snapshot: model.Snapshot{Path: "owned"}}, &resolve.Engine{}, model.Generation{}, func(context.Context, model.Generation) error { return refusal }, func(context.Context, model.Workload, *resolve.Engine, model.Generation, []string) (int, error) {
		t.Fatal("failed snapshot reached reconciliation")
		return 0, nil
	})
	if code != 0 || err != refusal {
		t.Fatal("sync snapshot failure changed")
	}
}

type cliCompletionFailWriter struct{}

func (cliCompletionFailWriter) Write([]byte) (int, error) { return 0, os.ErrPermission }

func TestCLICompletionObservabilityUsesOnlySelectedDiagnosticStore(t *testing.T) {
	for _, phase := range []string{"store", "construct", "off"} {
		t.Run(phase, func(t *testing.T) {
			cliCompletionWorkloadFlowCompletionCase3(t, phase)
		})
	}
}

func TestCLICompletionFlowTemporaryRefusalsCleanOnlyOwnedDirectory(t *testing.T) {
	for _, phase := range []string{"parent", "mkdir", "protect", "write", "cancel", "invoke"} {
		t.Run(phase, func(t *testing.T) { cliCompletionFlowFailure(t, phase) })
	}
}

func cliCompletionFlowFailure(t *testing.T, phase string) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	p := &cliCompletionFlowProbe{t: t, phase: phase, ctx: ctx, cancel: cancel, parentPath: privateCLIDirectory(t), refusal: fault.New(fault.Apply, "owned flow "+phase)}
	p.directory = filepath.Join(p.parentPath, "flow-owned")
	ops := flowTemporaryOperations{p.parent, p.mkdir, p.protect, p.write, p.invoke}
	options := flowOptions{workload: "demo", source: bootstrapOptions{sourceURL: "https://github.com/acme/cq/tree/" + bootstrapRef, ref: bootstrapRef, path: "cq.json", digest: bootstrapDigest}}
	app := application{buildVersion: "1.0.0", runner: Runner{Out: io.Discard, Err: io.Discard}}
	code, err := app.flowRemoteWithOperations(ctx, options, ciGitHub, p.fetch, ops)
	p.assertResult(code, err)
}

func TestCLICompletionFlowParentResolutionAndAbsoluteFailures(t *testing.T) {
	for _, phase := range []string{"resolve", "absolute"} {
		cliCompletionFlowParentFailure(t, phase)
	}
}
func cliCompletionFlowParentFailure(t *testing.T, phase string) {
	t.Helper()

	var calls []string
	path, err := flowTemporaryParentWithOperations("owned-temp", func(got string) (string, error) {
		calls = append(calls, "resolve")
		if got != "owned-temp" {
			t.Fatal("temporary input changed")
		}
		if phase == "resolve" {
			return "", os.ErrPermission
		}
		return "owned-resolved", nil
	}, func(got string) (string, error) {
		calls = append(calls, "absolute")
		if got != "owned-resolved" {
			t.Fatal("canonical input changed")
		}
		return "", os.ErrPermission
	})
	cliCompletionFault(t, err, fault.Apply, flowTemporaryPolicyOperation)
	want := []string{"resolve"}
	if phase == "absolute" {
		want = append(want, "absolute")
	}
	if path != "" || !reflect.DeepEqual(calls, want) {
		t.Fatal("parent resolver continued after failure")
	}
}

// Current-generation timestamps are a real input, not an injected clock failure.
func cliCompletionGeneration() model.Generation {
	return model.Generation{Workload: "demo", RetrievedAt: time.Now().UTC(), Values: map[string]model.Value{}}
}

func cliCompletionWorkloadFlowCompletionCase1(t *testing.T, phase string) {
	t.Helper()
	p := &cliCompletionLaunchProbe{t: t, phase: phase, refusal: fault.New(fault.Apply, "owned "+phase), app: application{runner: Runner{In: bytes.NewReader(nil), Out: io.Discard, Err: io.Discard}}}
	p.release = p.unlock
	code, err := p.app.runResolvedWorkloadWithOperations(context.Background(), &resolve.Engine{}, model.Workload{}, model.Generation{}, nil, &p.release, p.record, p.run)
	expected := map[string][]string{"record": {"record"}, "unlock": {"record", "unlock"}, "run": {"record", "unlock", "run"}}[phase]
	wantCode := 0
	if phase == "run" {
		wantCode = 17
	}
	if code != wantCode || err != p.refusal || !reflect.DeepEqual(p.calls, expected) || (p.release == nil) != (phase == "run") {
		t.Fatal("record/unlock/child failure changed")
	}
}

func cliCompletionWorkloadFlowCompletionCase2(t *testing.T, phase string) {
	t.Helper()
	p := &cliCompletionSnapshotProbe{phase: phase, refusal: fault.New(fault.Apply, "owned "+phase)}
	app := application{runner: Runner{Out: io.Discard}}
	if phase == "output" {
		app.runner.Out = cliCompletionFailWriter{}
	}
	code, err := app.storeSnapshotWithOperations(context.Background(), workloadRequest{id: "demo", command: "snapshot"}, model.Workload{}, &resolve.Engine{}, model.Generation{Workload: "demo"}, p.snapshot, p.record)
	if phase == "output" {
		cliCompletionFault(t, err, fault.Apply, "output")
	} else if err != p.refusal {
		t.Fatal("snapshot failure changed")
	}
	expected := map[string][]string{"snapshot": {"snapshot"}, "record": {"snapshot", "snapshotted"}, "output": {"snapshot", "snapshotted"}}[phase]
	if code != 0 || !reflect.DeepEqual(p.calls, expected) {
		t.Fatal("snapshot persistence ordering changed")
	}
}

func cliCompletionWorkloadFlowCompletionCase3(t *testing.T, phase string) {
	t.Helper()
	p := &cliCompletionTelemetryProbe{t: t, phase: phase, refusal: fault.New(fault.Apply, "owned "+phase), config: &model.Observability{DiagnosticLevel: "error"}}
	if phase == "off" {
		p.config.DiagnosticLevel = "off"
	}
	app := application{buildVersion: "1.2.3"}
	runtime, err := app.newObservabilityWithOperations(context.Background(), model.Policy{Observability: p.config}, p.store, p.construct)
	expected := map[string][]string{"store": {"store"}, "construct": {"store", "construct"}, "off": {"construct"}}[phase]
	if runtime != nil || err != p.refusal || !reflect.DeepEqual(p.calls, expected) {
		t.Fatal("telemetry admitted after store refusal")
	}
}

type cliCompletionPipelineProbe struct {
	t          *testing.T
	phase      string
	refusal    error
	calls      []string
	policy     model.Policy
	engine     *resolve.Engine
	generation model.Generation
}

func (p *cliCompletionPipelineProbe) step(name string) error {
	p.calls = append(p.calls, name)
	if p.phase == name {
		return p.refusal
	}
	return nil
}
func (p *cliCompletionPipelineProbe) construct(policy model.Policy) (*resolve.Engine, error) {
	if !reflect.DeepEqual(policy, p.policy) {
		p.t.Fatal("policy changed")
	}
	return p.engine, p.step("engine")
}
func (p *cliCompletionPipelineProbe) lock(engine *resolve.Engine, id string) (func() error, error) {
	if engine != p.engine || id != "demo" {
		p.t.Fatal("lock scope changed")
	}
	return p.unlock, p.step("lock")
}
func (p *cliCompletionPipelineProbe) unlock() error { p.calls = append(p.calls, "unlock"); return nil }
func (p *cliCompletionPipelineProbe) repository(_ context.Context, work model.Workload, engine *resolve.Engine) (*resolve.Engine, error) {
	if work.Mode != "offline" || engine != p.engine {
		p.t.Fatal("repository scope changed")
	}
	return engine, p.step("repository")
}
func (p *cliCompletionPipelineProbe) resolve(_ context.Context, engine *resolve.Engine, request workloadRequest) (model.Generation, error) {
	if engine != p.engine || request.command != "snapshot" || request.id != "demo" {
		p.t.Fatal("resolution scope changed")
	}
	return p.generation, p.step("resolve")
}
func (p *cliCompletionPipelineProbe) run(context.Context, *resolve.Engine, model.Workload, model.Generation, []string, *func() error) (int, error) {
	p.t.Fatal("snapshot selected launch")
	return 0, nil
}
func (p *cliCompletionPipelineProbe) complete(_ context.Context, request workloadRequest, _ model.Workload, engine *resolve.Engine, generation model.Generation) (int, error) {
	if request.command != "snapshot" || engine != p.engine || !reflect.DeepEqual(generation, p.generation) {
		p.t.Fatal("completion scope changed")
	}
	return 0, p.step("complete")
}

type cliCompletionFlowProbe struct {
	t                     *testing.T
	phase                 string
	ctx                   context.Context
	cancel                context.CancelFunc
	parentPath, directory string
	refusal               error
	calls                 []string
	fetches               int
}

func (p *cliCompletionFlowProbe) fetch(ctx context.Context, request bootstrap.Request) (bootstrap.Result, error) {
	p.fetches++
	if ctx != p.ctx || request.ConfigRef != bootstrapRef || request.ExpectedSHA256 != bootstrapDigest {
		p.t.Fatal("fetch pin changed")
	}
	return bootstrap.Result{Bytes: bootstrapPolicy()}, nil
}
func (p *cliCompletionFlowProbe) parent() (string, error) {
	p.calls = append(p.calls, "parent")
	if p.phase == "parent" {
		return "", p.refusal
	}
	return p.parentPath, nil
}
func (p *cliCompletionFlowProbe) mkdir(parent, pattern string) (string, error) {
	p.calls = append(p.calls, "mkdir")
	if parent != p.parentPath || pattern != "cq-flow-" {
		p.t.Fatal("temporary scope changed")
	}
	if p.phase == "mkdir" {
		return "", os.ErrPermission
	}
	return p.directory, os.Mkdir(p.directory, 0o700)
}
func (p *cliCompletionFlowProbe) protect(directory string) error {
	p.calls = append(p.calls, "protect")
	if directory != p.directory {
		p.t.Fatal("protection path changed")
	}
	if p.phase == "protect" {
		return p.refusal
	}
	return platform.EnsurePrivateDirectory(directory)
}
func (p *cliCompletionFlowProbe) write(path string, data []byte) error {
	p.calls = append(p.calls, "write")
	if path != filepath.Join(p.directory, temporaryFlowPolicy) || !bytes.Equal(data, bootstrapPolicy()) {
		p.t.Fatal("policy write changed")
	}
	if p.phase == "write" {
		return p.refusal
	}
	err := platform.CreateProtectedExclusive(path, data)
	if p.phase == "cancel" {
		p.cancel()
	}
	return err
}
func (p *cliCompletionFlowProbe) invoke(app application, ctx context.Context, workload string, runner ciRunner, kind, ref, digest string) (int, error) {
	p.calls = append(p.calls, "invoke")
	if ctx != p.ctx || app.config != filepath.Join(p.directory, temporaryFlowPolicy) || workload != "demo" || runner != ciGitHub || kind != "pinned" || ref != bootstrapRef || digest != bootstrapDigest {
		p.t.Fatal("temporary dispatch identity changed")
	}
	return 19, p.refusal
}
func (p *cliCompletionFlowProbe) assertResult(code int, err error) {
	expected := map[string][]string{
		"parent": {"parent"}, "mkdir": {"parent", "mkdir"}, "protect": {"parent", "mkdir", "protect"},
		"write": {"parent", "mkdir", "protect", "write"}, "cancel": {"parent", "mkdir", "protect", "write"},
		"invoke": {"parent", "mkdir", "protect", "write", "invoke"},
	}[p.phase]
	wantCode := 0
	if p.phase == "invoke" {
		wantCode = 19
	}
	p.assertError(err)
	if code != wantCode || p.fetches != 1 || !reflect.DeepEqual(p.calls, expected) {
		p.t.Fatalf("flow effect ordering changed: %v", p.calls)
	}
	if _, err := os.Lstat(p.directory); !os.IsNotExist(err) {
		p.t.Fatal("owned temporary directory survived cleanup")
	}
	if _, err := os.Stat(p.parentPath); err != nil {
		p.t.Fatal("cleanup removed parent")
	}
}
func (p *cliCompletionFlowProbe) assertError(err error) {
	switch p.phase {
	case "mkdir":
		cliCompletionFault(p.t, err, fault.Apply, flowTemporaryPolicyOperation)
	case "cancel":
		if err != context.Canceled {
			p.t.Fatal("late cancellation changed")
		}
	default:
		if err != p.refusal {
			p.t.Fatalf("flow refusal changed: %v", err)
		}
	}
}

type cliCompletionLaunchProbe struct {
	t       *testing.T
	phase   string
	refusal error
	calls   []string
	app     application
	release func() error
}

func (p *cliCompletionLaunchProbe) record(state.Store, model.Workload, model.Generation, string) error {
	p.calls = append(p.calls, "record")
	if p.phase == "record" {
		return p.refusal
	}
	return nil
}
func (p *cliCompletionLaunchProbe) unlock() error {
	p.calls = append(p.calls, "unlock")
	if p.phase == "unlock" {
		return p.refusal
	}
	return nil
}
func (p *cliCompletionLaunchProbe) run(_ context.Context, _ model.Workload, _ model.Generation, _ []string, input io.Reader, output, diagnostics io.Writer) (int, error) {
	p.calls = append(p.calls, "run")
	if p.release != nil || input != p.app.runner.In || output != p.app.runner.Out || diagnostics != p.app.runner.Err {
		p.t.Fatal("launch retained lock or changed I/O")
	}
	return 17, p.refusal
}

type cliCompletionSnapshotProbe struct {
	phase   string
	refusal error
	calls   []string
}

func (p *cliCompletionSnapshotProbe) snapshot(context.Context, model.Generation) error {
	p.calls = append(p.calls, "snapshot")
	if p.phase == "snapshot" {
		return p.refusal
	}
	return nil
}
func (p *cliCompletionSnapshotProbe) record(_ state.Store, _ model.Workload, _ model.Generation, phase string) error {
	p.calls = append(p.calls, phase)
	if p.phase == "record" {
		return p.refusal
	}
	return nil
}

type cliCompletionTelemetryProbe struct {
	t       *testing.T
	phase   string
	refusal error
	calls   []string
	config  *model.Observability
}

func (p *cliCompletionTelemetryProbe) store() (state.Store, error) {
	p.calls = append(p.calls, "store")
	if p.phase == "store" {
		return state.Store{}, p.refusal
	}
	return state.Store{Directory: "owned-directory"}, nil
}
func (p *cliCompletionTelemetryProbe) construct(_ context.Context, config *model.Observability, options observability.Options) (*observability.Runtime, error) {
	p.calls = append(p.calls, "construct")
	directory := "owned-directory"
	if p.phase == "off" {
		directory = ""
	}
	if config != p.config || options.Version != "1.2.3" || options.DiagnosticDirectory != directory {
		p.t.Fatal("telemetry inputs changed")
	}
	return nil, p.refusal
}
