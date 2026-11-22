package cli

import (
	"context"

	"github.com/maldous/CircularQuay/internal/model"
	"github.com/maldous/CircularQuay/internal/platform"
	"github.com/maldous/CircularQuay/internal/resolve"
)

type workloadCommandOperations struct {
	engine     func(model.Policy) (*resolve.Engine, error)
	lock       func(*resolve.Engine, string) (func() error, error)
	repository func(context.Context, model.Workload, *resolve.Engine) (*resolve.Engine, error)
	resolve    func(context.Context, *resolve.Engine, workloadRequest) (model.Generation, error)
	run        func(context.Context, *resolve.Engine, model.Workload, model.Generation, []string, *func() error) (int, error)
	complete   func(context.Context, workloadRequest, model.Workload, *resolve.Engine, model.Generation) (int, error)
}

func (a application) defaultWorkloadOperations() workloadCommandOperations {
	return workloadCommandOperations{a.engine, func(engine *resolve.Engine, id string) (func() error, error) { return engine.State.Lock(id) }, a.withRepositorySources, resolveWorkloadGeneration, a.runResolvedWorkload, a.completeWorkloadCommand}
}

type flowTemporaryOperations struct {
	parent  func() (string, error)
	mkdir   func(string, string) (string, error)
	protect func(string) error
	write   func(string, []byte) error
	invoke  func(application, context.Context, string, ciRunner, string, string, string) (int, error)
}

func defaultFlowOperations() flowTemporaryOperations {
	return flowTemporaryOperations{flowTemporaryParent, newFlowTempDir, platform.EnsurePrivateDirectory, platform.CreateProtectedExclusive, func(a application, ctx context.Context, workload string, runner ciRunner, kind, ref, digest string) (int, error) {
		return a.flowLocal(ctx, workload, runner, kind, ref, digest)
	}}
}
