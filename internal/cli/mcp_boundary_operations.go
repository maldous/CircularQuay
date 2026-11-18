package cli

import (
	"context"
	"io"

	cqmcp "github.com/maldous/CircularQuay/internal/mcp"
	"github.com/maldous/CircularQuay/internal/model"
	"github.com/maldous/CircularQuay/internal/platform"
	"github.com/maldous/CircularQuay/internal/resolve"
)

type mcpServeOperations struct {
	construct func(model.Policy, cqmcp.Backend, cqmcp.Options) (*cqmcp.Service, error)
	stdio     func(context.Context, *cqmcp.Service) error
	http      func(context.Context, string, *cqmcp.Service, mcpServeOptions) (int, error)
}

func defaultMCPServeOperations() mcpServeOperations {
	return mcpServeOperations{cqmcp.New, func(ctx context.Context, service *cqmcp.Service) error { return service.RunStdio(ctx) }, runMCPHTTP}
}

func (b backend) acquireWorkload(engine *resolve.Engine, id string) (func() error, error) {
	if b.acquire != nil {
		return b.acquire(engine, id)
	}
	return engine.State.Lock(id)
}

func (b backend) executeWorkload(ctx context.Context, workload model.Workload, generation model.Generation, override []string, input io.Reader, output, diagnostics io.Writer) (int, error) {
	if b.execute != nil {
		return b.execute(ctx, workload, generation, override, input, output, diagnostics)
	}
	return platform.RunWithIO(ctx, workload, generation, override, input, output, diagnostics)
}
