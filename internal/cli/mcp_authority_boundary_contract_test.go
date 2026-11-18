package cli

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/maldous/CircularQuay/internal/fault"
	cqmcp "github.com/maldous/CircularQuay/internal/mcp"
	"github.com/maldous/CircularQuay/internal/model"
)

func TestMCPAuthorityBoundaryContractServeRequiresReadablePolicy(t *testing.T) {
	app, probe := mcpBoundaryUnusedApplication(t)
	// Valid command syntax reaches the policy read, unlike the malformed
	// command tests. The inner code is not a successful CLI exit status.
	code, err := app.serveMCP(context.Background(), []string{"serve"})
	if code != 0 {
		t.Fatal("policy refusal changed the inner return code")
	}
	requireMCPBoundaryFault(t, err, fault.Trust, "policy file")
	requireMCPBoundaryNoEffects(t, app, probe)
}

func TestMCPAuthorityBoundaryContractBackendPreservesPolicyRefusal(t *testing.T) {
	app, probe := mcpBoundaryUnusedApplication(t)
	b := backend{
		app: app,
		resolveGeneration: func(context.Context, model.Policy, string) (model.Generation, error) {
			probe.resolutions++
			return model.Generation{}, errors.New("synthetic resolver must not be reached")
		},
	}
	// A live context reaches policy loading before principal admission. These
	// inner calls establish neither a public service grant nor cancellation.
	ctx := context.Background()
	status, err := b.Status(ctx, "SYNTHETIC_PRIVATE_WORKLOAD")
	requireMCPBoundaryFault(t, err, fault.Trust, "policy file")
	if status != (cqmcp.Status{}) {
		t.Fatal("policy refusal returned status metadata")
	}
	requireMCPBoundaryNoEffects(t, app, probe)

	value, err := b.Read(ctx, "SYNTHETIC_PRIVATE_WORKLOAD", "SYNTHETIC_PRIVATE_KEY")
	requireMCPBoundaryFault(t, err, fault.Trust, "policy file")
	if !reflect.DeepEqual(value, model.Value{}) {
		t.Fatal("policy refusal returned a value or its metadata")
	}
	requireMCPBoundaryNoEffects(t, app, probe)

	result, err := b.Run(ctx, "SYNTHETIC_PRIVATE_WORKLOAD")
	requireMCPBoundaryFault(t, err, fault.Trust, "policy file")
	if result != (cqmcp.RunResult{}) {
		t.Fatal("policy refusal returned a job result")
	}
	requireMCPBoundaryNoEffects(t, app, probe)
}

func TestMCPAuthorityBoundaryContractActionCatalogDeniesUnreadablePolicy(t *testing.T) {
	app, probe := mcpBoundaryUnusedApplication(t)
	// The catalog callback must fail closed before inspecting an executable
	// identity when its current policy cannot be read.
	if app.actionAllowed("SYNTHETIC_PRIVATE_WORKLOAD") {
		t.Fatal("unreadable policy admitted an action")
	}
	requireMCPBoundaryNoEffects(t, app, probe)
}
