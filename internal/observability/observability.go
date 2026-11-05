// Package observability provides bounded, configuration-owned runtime signals.
//
// It deliberately records only a closed operational vocabulary. Secret values,
// resource names, file paths, identities, credentials, request bodies, and raw
// errors are excluded before a signal reaches an exporter or local diagnostic
// sink. Telemetry is advisory: an exporter outage must never change a workload
// authorization, resolution, projection, activation, or recovery result.
package observability

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"time"

	"github.com/maldous/CircularQuay/internal/fault"
)

type runtimeContextKey struct{}

// Runtime owns one process-local telemetry pipeline. It is intentionally not a
// global OpenTelemetry provider: CQ can run beside an SDK consumer
// without replacing that consumer's telemetry ownership.
type Runtime struct {
	backend backend
}

type backend interface {
	begin(context.Context, Operation) (context.Context, span)
	event(context.Context, Event)
	health() Health
	shutdown(context.Context) error
}

type span interface{ end(Outcome) }

type noopBackend struct{}
type noopSpan struct{}

func (noopBackend) begin(ctx context.Context, _ Operation) (context.Context, span) {
	return ctx, noopSpan{}
}
func (noopBackend) event(context.Context, Event) {
	// Disabled telemetry deliberately discards events without opening a sink.
}
func (noopBackend) health() Health                 { return Health{} }
func (noopBackend) shutdown(context.Context) error { return nil }
func (noopSpan) end(Outcome) {
	// A disabled span owns no exporter or diagnostic state to finalize.
}

// Disabled returns a runtime that performs no network work or local logging.
// It is safe to share between callers.
func Disabled() *Runtime { return &Runtime{backend: noopBackend{}} }

// WithRuntime attaches the configured process-local runtime to a request or
// operation context. An absent runtime is equivalent to Disabled.
func WithRuntime(ctx context.Context, runtime *Runtime) context.Context {
	if runtime == nil {
		runtime = Disabled()
	}
	return context.WithValue(ctx, runtimeContextKey{}, runtime)
}

// From returns the runtime scoped to ctx, or a no-op runtime when none exists.
func From(ctx context.Context) *Runtime {
	if runtime, ok := ctx.Value(runtimeContextKey{}).(*Runtime); ok && runtime != nil {
		return runtime
	}
	return Disabled()
}

// Phase is a closed operational phase label. It never accepts a workload,
// secret, provider resource, principal, filesystem path, or user input.
type Phase string

const (
	PhaseCLI           Phase = "cli"
	PhasePolicy        Phase = "policy"
	PhaseRepository    Phase = "repository"
	PhaseAuthorization Phase = "authorization"
	PhaseResolve       Phase = "resolve"
	PhaseRefresh       Phase = "refresh"
	PhaseSource        Phase = "source"
	PhaseSnapshot      Phase = "snapshot"
	PhaseProjection    Phase = "projection"
	PhaseActivation    Phase = "activation"
	PhaseWorkload      Phase = "workload"
	PhaseMCP           Phase = "mcp"
	PhaseAudit         Phase = "audit"
	PhaseRecovery      Phase = "recovery"
)

// Outcome is a closed result label suitable for bounded metric dimensions.
type Outcome string

const (
	OutcomeSuccess     Outcome = "success"
	OutcomeDenied      Outcome = "denied"
	OutcomeFailed      Outcome = "failed"
	OutcomeUnavailable Outcome = "unavailable"
	OutcomeCancelled   Outcome = "cancelled"
	OutcomeExpired     Outcome = "expired"
	OutcomeFallback    Outcome = "fallback"
)

// Operation contains only bounded catalog dimensions. ProviderKind is an
// implementation category such as "openbao" or "aws-sm", never a provider
// account, endpoint, resource, secret name, or profile.
type Operation struct {
	Phase        Phase
	ProviderKind string
}

// Event records an explicit bounded event without a payload. Correlation is an
// opaque random identifier; it aids joining records but is never authorization.
type Event struct {
	Operation
	Outcome     Outcome
	Correlation string
	At          time.Time
}

// Start begins a bounded operation span and returns a context carrying its
// trace correlation. The returned completion function must be called once;
// it does not return an error and cannot alter the caller's operation result.
func Start(ctx context.Context, operation Operation) (context.Context, func(Outcome)) {
	operation = normalizeOperation(operation)
	ctx = withCorrelation(ctx)
	ctx, current := From(ctx).backend.begin(ctx, operation)
	return ctx, func(outcome Outcome) { current.end(normalizeOutcome(outcome)) }
}

// Record emits a standalone bounded lifecycle event. It is suitable for a
// branch such as fallback or recovery that has no single parent operation.
func Record(ctx context.Context, operation Operation, outcome Outcome) {
	From(ctx).backend.event(ctx, Event{
		Operation:   normalizeOperation(operation),
		Outcome:     normalizeOutcome(outcome),
		Correlation: correlation(ctx),
		At:          time.Now().UTC(),
	})
}

// Shutdown flushes the bounded pipeline within the caller's deadline. It is
// intended for process teardown and must not be called from a workload path.
func (r *Runtime) Shutdown(ctx context.Context) error {
	if r == nil || r.backend == nil {
		return nil
	}
	return r.backend.shutdown(ctx)
}

// Health describes local pipeline conditions without exporting an endpoint,
// credential, error body, or other sensitive implementation detail. Operators
// can query it through an embedding application or future managed service.
type Health struct {
	DiagnosticWriteFailures uint64 `json:"diagnostic_write_failures"`
	ExportFailures          uint64 `json:"export_failures"`
	ShutdownExportFailure   bool   `json:"shutdown_export_failure"`
}

// Health returns a bounded view of local telemetry health. It is advisory and
// deliberately separate from mandatory security audit evidence.
func (r *Runtime) Health() Health {
	if r == nil || r.backend == nil {
		return Health{}
	}
	return r.backend.health()
}

func normalizeOperation(operation Operation) Operation {
	if !validPhase(operation.Phase) {
		operation.Phase = PhaseCLI
	}
	if !validProviderKind(operation.ProviderKind) {
		operation.ProviderKind = "none"
	}
	return operation
}

func normalizeOutcome(outcome Outcome) Outcome {
	if validOutcome(outcome) {
		return outcome
	}
	return OutcomeFailed
}

func validPhase(phase Phase) bool {
	switch phase {
	case PhaseCLI, PhasePolicy, PhaseRepository, PhaseAuthorization, PhaseResolve, PhaseRefresh, PhaseSource, PhaseSnapshot, PhaseProjection, PhaseActivation, PhaseWorkload, PhaseMCP, PhaseAudit, PhaseRecovery:
		return true
	default:
		return false
	}
}

func validProviderKind(kind string) bool {
	switch kind {
	case "none", "json", "env", "sops", "openbao", "vault", "doppler", "infisical", "akeyless", "aws-sm", "aws-ssm", "azure", "gcp", "bitwarden", "onepassword", "repository", "mcp", "native", "http", "file", "systemd", "windows", "cloudflare", "netlify", "windmill":
		return true
	default:
		return false
	}
}

func validOutcome(outcome Outcome) bool {
	switch outcome {
	case OutcomeSuccess, OutcomeDenied, OutcomeFailed, OutcomeUnavailable, OutcomeCancelled, OutcomeExpired, OutcomeFallback:
		return true
	default:
		return false
	}
}

func correlation(ctx context.Context) string {
	if current, ok := ctx.Value(correlationKey{}).(string); ok && len(current) == 32 {
		return current
	}
	var bytes [16]byte
	if _, err := rand.Read(bytes[:]); err != nil {
		return "unavailable"
	}
	return hex.EncodeToString(bytes[:])
}

type correlationKey struct{}

func withCorrelation(ctx context.Context) context.Context {
	if current, ok := ctx.Value(correlationKey{}).(string); ok && len(current) == 32 {
		return ctx
	}
	return context.WithValue(ctx, correlationKey{}, correlation(ctx))
}

// OutcomeForError maps only CQ's public error category. It never
// serializes Error(), which may contain provider-controlled sensitive text.
func OutcomeForError(err error) Outcome {
	if err == nil {
		return OutcomeSuccess
	}
	if errorsIsCanceled(err) {
		return OutcomeCancelled
	}
	switch fault.Category(err) {
	case fault.Denied, fault.Trust, fault.Invalid:
		return OutcomeDenied
	case fault.Unavailable:
		return OutcomeUnavailable
	case fault.Expired:
		return OutcomeExpired
	default:
		return OutcomeFailed
	}
}

func errorsIsCanceled(err error) bool {
	return errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded)
}
