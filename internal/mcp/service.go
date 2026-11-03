// Package mcp exposes optional, policy-bound Model Context Protocol access.
// Its raw read response is an explicit disclosure; diagnostics never contain values.
package mcp

import (
	"context"
	"io"
	"log/slog"
	"os"
	"os/user"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/maldous/CircularQuay/internal/fault"
	"github.com/maldous/CircularQuay/internal/model"
	"github.com/maldous/CircularQuay/internal/policy"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// Backend is the enrolled workload interface. It must not return process output.
type Backend interface {
	Status(context.Context, string) (Status, error)
	Run(context.Context, string) (RunResult, error)
	Read(context.Context, string, string) (model.Value, error)
}

// Status contains only non-secret, allowlisted state.
type Status struct {
	Workload string `json:"workload"`
	Mode     string `json:"mode"`
	State    string `json:"state"`
}

// RunResult deliberately has no stdout, stderr or arbitrary result field.
type RunResult struct {
	ID       string `json:"id"`
	Status   string `json:"status"`
	ExitCode int    `json:"exit_code"`
}

// Principal is authoritative identity from the OS or a verified access token.
type Principal struct {
	Subject, Client string
	// HTTP scopes are verified token claims, never tool arguments.
	HTTP   bool
	Scopes []string
}

type principalContextKey struct{}

// PrincipalFromContext lets an enrolled backend recheck freshly loaded policy.
// The context is set only after transport authentication, never from tool input.
func PrincipalFromContext(ctx context.Context) (Principal, bool) {
	p, ok := ctx.Value(principalContextKey{}).(Principal)
	return p, ok
}

func withPrincipal(ctx context.Context, p Principal) context.Context {
	return context.WithValue(ctx, principalContextKey{}, p)
}

// Options controls trusted integration hooks, not caller-provided arguments.
type Options struct {
	Audit Auditor
	// ActionAllowed must validate the enrolled executable identity and trust grant.
	// An omitted hook denies invocation; it does not affect explicit secret reads.
	ActionAllowed func(string) bool
	// PolicyProvider permits revocation without trusting a cached discovery result.
	// When absent, the immutable configuration supplied to New is used.
	PolicyProvider func() (model.Policy, error)
}

// Service is a bounded interface over an already enrolled resolver.
type Service struct {
	configured model.Policy
	backend    Backend
	options    Options
}

// New validates policy before exposing any protocol transport.
func New(p model.Policy, backend Backend, options Options) (*Service, error) {
	if backend == nil {
		return nil, fault.New(fault.Invalid, "mcp backend")
	}
	if err := policy.Validate(p); err != nil {
		return nil, err
	}
	if options.Audit == nil && p.MCP.AuditPath != "" {
		audit, err := NewFileAuditor(p.MCP.AuditPath)
		if err != nil {
			return nil, err
		}
		options.Audit = audit
	}
	return &Service{configured: p, backend: backend, options: options}, nil
}

func (s *Service) current() model.Policy {
	if s.options.PolicyProvider != nil {
		p, err := s.options.PolicyProvider()
		if err != nil || policy.Validate(p) != nil {
			return model.Policy{}
		}
		return p
	}
	return s.configured
}

func (s *Service) authorized(p Principal, workload, operation, key string) bool {
	return validHandle(workload) && scopeAllows(p, operation) && policy.Authorize(s.current(), p.Subject, p.Client, workload, operation, key)
}

func scopeAllows(p Principal, operation string) bool {
	if !p.HTTP {
		return true
	}
	for _, scope := range p.Scopes {
		if scope == "cquay:"+operation {
			return true
		}
	}
	return false
}

func validHandle(value string) bool {
	if value == "" || len(value) > 128 {
		return false
	}
	for _, r := range value {
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '-' || r == '.') {
			return false
		}
	}
	return value != "." && value != ".."
}

// ListWorkloads returns no source references or secret names.
func (s *Service) ListWorkloads(ctx context.Context, p Principal) ([]string, error) {
	if err := ctx.Err(); err != nil {
		return nil, fault.New(fault.Unavailable, "mcp request")
	}
	ids := []string{}
	for id := range s.current().Workloads {
		if s.authorized(p, id, "list", "") {
			ids = append(ids, id)
		}
	}
	sort.Strings(ids)
	return ids, nil
}

// GetStatus checks authorization before asking the backend for even metadata.
func (s *Service) GetStatus(ctx context.Context, p Principal, workload string) (Status, error) {
	if ctx.Err() != nil {
		return Status{}, fault.New(fault.Unavailable, "mcp request")
	}
	if !s.authorized(p, workload, "status", "") {
		return Status{}, fault.New(fault.Denied, "mcp status")
	}
	status, err := s.backend.Status(withPrincipal(ctx, p), workload)
	if err != nil {
		return Status{}, safeError(err, "mcp status")
	}
	status.Workload = workload
	status.Mode = s.current().Workloads[workload].Mode
	if !oneOf(status.State, "unprovisioned", "ready", "pending", "error", "running", "stopped", "unknown") {
		status.State = "unknown"
	}
	return status, nil
}

// RunWorkload invokes only a locally enrolled and independently approved action.
func (s *Service) RunWorkload(ctx context.Context, p Principal, workload string) (RunResult, error) {
	if ctx.Err() != nil {
		return RunResult{}, fault.New(fault.Unavailable, "mcp request")
	}
	if !s.authorized(p, workload, "run", "") || s.options.ActionAllowed == nil || !s.options.ActionAllowed(workload) {
		return RunResult{}, fault.New(fault.Denied, "mcp action")
	}
	w := s.current().Workloads[workload]
	if w.UntrustedCode && !s.mayReadAll(p, workload, w) {
		return RunResult{}, fault.New(fault.Denied, "mcp code disclosure")
	}
	if err := s.record(ctx, p, workload, "run", "", "authorized", w.AuditRequired); err != nil {
		return RunResult{}, err
	}
	result, err := s.backend.Run(withPrincipal(ctx, p), workload)
	if err != nil {
		return RunResult{}, safeError(err, "mcp action")
	}
	// The backend is trusted, but its strings must not become an output channel.
	if !validHandle(result.ID) {
		result.ID = "redacted"
	}
	if !oneOf(result.Status, "completed", "failed", "running", "cancelled") {
		result.Status = "unknown"
	}
	if result.ExitCode < 0 || result.ExitCode > 255 {
		result.ExitCode = 1
	}
	return result, nil
}

func (s *Service) mayReadAll(p Principal, id string, w model.Workload) bool {
	for _, b := range w.Bindings {
		if !s.authorized(p, id, "read", b.Name) {
			return false
		}
	}
	return true
}

// ReadSecret deliberately discloses one declared static value to the MCP client.
// Revocation cannot recall a value already delivered into client/model context.
func (s *Service) ReadSecret(ctx context.Context, p Principal, workload, key string) (SecretResult, error) {
	if ctx.Err() != nil {
		return SecretResult{}, fault.New(fault.Unavailable, "mcp request")
	}
	if !validHandle(key) || !s.authorized(p, workload, "read", key) {
		return SecretResult{}, fault.New(fault.Denied, "mcp disclosure")
	}
	if s.options.Audit == nil {
		return SecretResult{}, fault.New(fault.Apply, "mcp disclosure audit")
	}
	value, err := s.backend.Read(withPrincipal(ctx, p), workload, key)
	if err != nil {
		return SecretResult{}, safeError(err, "mcp disclosure")
	}
	if value.Leased || value.Renewable {
		return SecretResult{}, fault.New(fault.Unsupported, "mcp leased disclosure")
	}
	if !value.ExpiresAt.IsZero() && !time.Now().Before(value.ExpiresAt) {
		return SecretResult{}, fault.New(fault.Expired, "mcp disclosure")
	}
	if len(value.Data) > 65536 {
		return SecretResult{}, fault.New(fault.Unsupported, "mcp value size")
	}
	if s.current().Workloads[workload].AuditRequired && !value.AuditVerified {
		return SecretResult{}, fault.New(fault.Denied, "mcp source audit")
	}
	// The provider call may have blocked while an operator revoked the grant.
	// Recheck current policy before recording a disclosure and releasing data.
	if !s.authorized(p, workload, "read", key) {
		return SecretResult{}, fault.New(fault.Denied, "mcp disclosure revoked")
	}
	if ctx.Err() != nil {
		return SecretResult{}, fault.New(fault.Unavailable, "mcp request")
	}
	if err := s.record(ctx, p, workload, "read", key, "released", true); err != nil {
		return SecretResult{}, err
	}
	return SecretResult{Alias: key, Value: value.Data, Version: value.Version, ExpiresAt: value.ExpiresAt}, nil
}

// SecretResult is the sole protocol result allowed to carry plaintext.
type SecretResult struct {
	Alias     string    `json:"alias"`
	Value     string    `json:"value"`
	Version   string    `json:"version,omitempty"`
	ExpiresAt time.Time `json:"expires_at,omitempty"`
}

func safeError(err error, operation string) error { return fault.New(fault.Category(err), operation) }
func oneOf(value string, allowed ...string) bool {
	for _, item := range allowed {
		if value == item {
			return true
		}
	}
	return false
}

func (s *Service) record(ctx context.Context, p Principal, workload, operation, key, outcome string, required bool) error {
	if s.options.Audit == nil {
		if required {
			return fault.New(fault.Apply, "mcp audit")
		}
		return nil
	}
	event := Event{Time: time.Now().UTC(), Subject: p.Subject, Client: p.Client, Workload: workload, Operation: operation, Key: key, Outcome: outcome}
	if err := s.options.Audit.Record(ctx, event); err != nil {
		return fault.New(fault.Apply, "mcp audit")
	}
	return nil
}

func (s *Service) server(p Principal) *sdk.Server {
	server := sdk.NewServer(&sdk.Implementation{Name: "CircularQuay", Version: "0.1.0"}, &sdk.ServerOptions{
		Capabilities: &sdk.ServerCapabilities{},
		Logger:       slog.New(slog.NewTextHandler(io.Discard, nil)),
		// Leave protocol negotiation to the official SDK's maintained versions.
		SetCacheable: func(_ context.Context, _ sdk.Request, c *sdk.Cacheable) { c.CacheScope = "private"; c.TTLMs = 0 },
	})
	if s.hasOperation(p, "list") {
		sdk.AddTool(server, &sdk.Tool{Name: "list_workloads", Description: "List workload IDs explicitly visible to the authenticated principal."}, func(ctx context.Context, _ *sdk.CallToolRequest, _ emptyInput) (*sdk.CallToolResult, workloadList, error) {
			v, e := s.ListWorkloads(ctx, p)
			return nil, workloadList{Workloads: v}, e
		})
	}
	if s.hasOperation(p, "status") {
		sdk.AddTool(server, &sdk.Tool{Name: "get_status", Description: "Get safe state for an authorized workload; no values or process logs."}, func(ctx context.Context, _ *sdk.CallToolRequest, in workloadInput) (*sdk.CallToolResult, Status, error) {
			v, e := s.GetStatus(ctx, p, in.Workload)
			return nil, v, e
		})
	}
	if s.hasOperation(p, "run") {
		sdk.AddTool(server, &sdk.Tool{Name: "run_workload", Description: "Invoke one locally registered action. Accepts no executable, shell or provider URI."}, func(ctx context.Context, _ *sdk.CallToolRequest, in workloadInput) (*sdk.CallToolResult, RunResult, error) {
			v, e := s.RunWorkload(ctx, p, in.Workload)
			return nil, v, e
		})
	}
	if s.hasOperation(p, "read") {
		sdk.AddTool(server, &sdk.Tool{Name: "read_secret", Description: "Deliberately disclose one permitted static value to this client and potentially model context. Requires a separate disclosure grant."}, func(ctx context.Context, _ *sdk.CallToolRequest, in secretInput) (*sdk.CallToolResult, SecretResult, error) {
			v, e := s.ReadSecret(ctx, p, in.Workload, in.Key)
			return nil, v, e
		})
	}
	return server
}

type emptyInput struct{}
type workloadInput struct {
	Workload string `json:"workload" jsonschema:"An exact enrolled workload ID."`
}
type secretInput struct {
	Workload string `json:"workload"`
	Key      string `json:"key" jsonschema:"One exact permitted output alias, never a provider URI."`
}
type workloadList struct {
	Workloads []string `json:"workloads"`
}

func (s *Service) hasOperation(p Principal, operation string) bool {
	for id, w := range s.current().Workloads {
		if operation != "read" && s.authorized(p, id, operation, "") {
			return true
		}
		if operation == "read" {
			for _, b := range w.Bindings {
				if s.authorized(p, id, "read", b.Name) {
					return true
				}
			}
		}
	}
	return false
}

// LocalPrincipal binds local grants to a kernel UID or Windows SID. Username
// environment variables are not authority, including static Android builds.
func LocalPrincipal() (Principal, error) {
	if runtime.GOOS != "windows" {
		uid := os.Getuid()
		if uid < 0 {
			return Principal{}, fault.New(fault.Trust, "local mcp identity")
		}
		return Principal{Subject: "local:uid:" + strconv.Itoa(uid)}, nil
	}
	u, err := user.Current()
	if err != nil || !validSID(u.Uid) {
		return Principal{}, fault.New(fault.Trust, "local mcp identity")
	}
	return Principal{Subject: "local:sid:" + u.Uid}, nil
}

func validSID(sid string) bool {
	if !strings.HasPrefix(sid, "S-1-") || len(sid) > 256 {
		return false
	}
	for _, part := range strings.Split(sid[4:], "-") {
		if part == "" {
			return false
		}
		for _, r := range part {
			if r < '0' || r > '9' {
				return false
			}
		}
	}
	return true
}

// RunStdio uses the actual OS identity, not a caller-supplied identity label.
func (s *Service) RunStdio(ctx context.Context) error {
	p, err := LocalPrincipal()
	if err != nil {
		return err
	}
	return s.server(p).Run(ctx, &sdk.StdioTransport{})
}
