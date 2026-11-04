// Package mcp exposes optional, policy-bound Model Context Protocol access.
// Its raw read response is an explicit disclosure; diagnostics never contain values.
package mcp

import (
	"context"
	"io"
	"log/slog"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/maldous/CircularQuay/internal/dpop"
	"github.com/maldous/CircularQuay/internal/fault"
	"github.com/maldous/CircularQuay/internal/model"
	"github.com/maldous/CircularQuay/internal/observability"
	"github.com/maldous/CircularQuay/internal/policy"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

const (
	mcpRequest    = "mcp request"
	mcpDisclosure = "mcp disclosure"
	mcpAction     = "mcp action"
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
// Authority and AuthorityFingerprint bind an HTTP principal to the configured
// issuer contract that verified it; they are never caller-supplied tool input.
type Principal struct {
	Subject, Client      string
	Authority            string
	AuthorityFingerprint string
	// HTTP scopes are verified token claims, never tool arguments.
	HTTP      bool
	Scopes    []string
	Roles     []string
	ExpiresAt time.Time
}

// ValidAt reports whether this principal remains authorized at now. Local OS
// principals have no token lifetime. HTTP principals require a verified,
// nonzero future expiration.
func (p Principal) ValidAt(now time.Time) bool {
	return !p.HTTP || !p.ExpiresAt.IsZero() && !now.IsZero() && now.Before(p.ExpiresAt)
}

// Authorize makes the exact current-policy decision for this principal. Named
// HTTP authorities require a matching current trust fingerprint, global claim
// filters, and an operation-specific permission before their local grant can
// apply. Local OS principals and legacy HTTP policies use only empty-authority
// grants, so they cannot consume a named-authority grant.
func (p Principal) Authorize(config model.Policy, workload, operation, key string, now time.Time) bool {
	if !p.ValidAt(now) {
		return false
	}
	if !p.HTTP {
		return p.localAuthorized(config, workload, operation, key, now)
	}
	if len(config.MCP.Authorities) == 0 {
		return p.legacyAuthorized(config, workload, operation, key, now)
	}
	return p.namedAuthorized(config, workload, operation, key, now)
}

func (p Principal) localAuthorized(config model.Policy, workload, operation, key string, now time.Time) bool {
	return p.Authority == "" && p.AuthorityFingerprint == "" && policy.AuthorizeIdentityAt(config, policy.Identity{Subject: p.Subject, Client: p.Client}, workload, operation, key, now)
}

func (p Principal) legacyAuthorized(config model.Policy, workload, operation, key string, now time.Time) bool {
	if p.Authority != "" || !legacyFingerprintMatches(config.MCP, p.AuthorityFingerprint) {
		return false
	}
	return legacyScopeAllows(p, operation) && policy.AuthorizeIdentityAt(config, policy.Identity{Subject: p.Subject, Client: p.Client}, workload, operation, key, now)
}

func legacyFingerprintMatches(config model.MCP, fingerprint string) bool {
	if config.Issuer == "" {
		return fingerprint == ""
	}
	current := policy.MCPLegacyFingerprint(config)
	return current != "" && fingerprint != "" && fingerprint == current
}

func (p Principal) namedAuthorized(config model.Policy, workload, operation, key string, now time.Time) bool {
	authority, found := config.MCP.Authorities[p.Authority]
	if !found || p.AuthorityFingerprint == "" || p.AuthorityFingerprint != policy.MCPAuthorityFingerprint(config.MCP, p.Authority) {
		return false
	}
	if !authorityAllows(authority, p, operation) {
		return false
	}
	return policy.AuthorizeIdentityAt(config, policy.Identity{Authority: p.Authority, Subject: p.Subject, Client: p.Client}, workload, operation, key, now)
}

func authorityAllows(authority model.IdentityAuthority, p Principal, operation string) bool {
	permission, found := authority.Permissions[operation]
	return found && len(permission.Scopes)+len(permission.Roles) > 0 && optionalClaimAllows(authority.Clients, p.Client) && optionalClaimAllows(authority.Subjects, p.Subject) && allClaimsPresent(p.Scopes, authority.RequiredScopes) && allClaimsPresent(p.Roles, authority.RequiredRoles) && allClaimsPresent(p.Scopes, permission.Scopes) && allClaimsPresent(p.Roles, permission.Roles)
}

func optionalClaimAllows(allowed []string, value string) bool {
	return len(allowed) == 0 || claimPresent(allowed, value)
}

func allClaimsPresent(actual, required []string) bool {
	for _, value := range required {
		if !claimPresent(actual, value) {
			return false
		}
	}
	return true
}

func claimPresent(values []string, wanted string) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
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
	// DPoPStateDirectory is the protected local state root for the optional
	// single-replica DPoP replay and nonce store.
	DPoPStateDirectory string
}

// Service is a bounded interface over an already enrolled resolver.
type Service struct {
	configured model.Policy
	backend    Backend
	options    Options
	now        func() time.Time
	dpopStore  dpop.Store
}

// New validates policy before exposing any protocol transport.
func New(p model.Policy, backend Backend, options Options) (*Service, error) {
	if backend == nil {
		return nil, fault.New(fault.Invalid, "mcp backend")
	}
	if err := policy.Validate(p); err != nil {
		return nil, err
	}
	if err := configureAudit(&options, p); err != nil {
		return nil, err
	}
	replay, err := createDPoPReplayStore(p, options.DPoPStateDirectory)
	if err != nil {
		return nil, err
	}
	return &Service{configured: p, backend: backend, options: options, now: time.Now, dpopStore: replay}, nil
}

func configureAudit(options *Options, policy model.Policy) error {
	if options.Audit != nil || policy.MCP.AuditPath == "" {
		return nil
	}
	audit, err := NewFileAuditor(policy.MCP.AuditPath)
	if err != nil {
		return err
	}
	options.Audit = audit
	return nil
}

func createDPoPReplayStore(policy model.Policy, stateDirectory string) (dpop.Store, error) {
	if !policyRequiresDPoP(policy) {
		return nil, nil
	}
	if protectedOperationalInputWithinDPoPState(policy, stateDirectory) {
		return nil, fault.New(fault.Invalid, "dpop protected input")
	}
	store, err := dpop.NewPersistentStore(stateDirectory)
	if err != nil {
		return nil, fault.New(fault.Trust, "dpop replay store")
	}
	return store, nil
}

func policyRequiresDPoP(p model.Policy) bool {
	for _, authority := range p.MCP.Authorities {
		if authority.SenderConstraint != nil {
			return true
		}
	}
	return false
}

// protectedOperationalInputWithinDPoPState prevents the replay-store writer
// from sharing a directory with any protected operational input. Policy
// validation has already established absolute, cleaned paths before this
// runtime-only comparison. In particular, the DPoP nonce store cannot become
// a writer to telemetry/update authorization or replacement-root files.
func protectedOperationalInputWithinDPoPState(policy model.Policy, stateDirectory string) bool {
	if !filepath.IsAbs(stateDirectory) {
		return false
	}
	stateDirectory = filepath.Clean(stateDirectory)
	for _, protected := range protectedOperationalInputs(policy) {
		if protected == "" {
			continue
		}
		relative, err := filepath.Rel(stateDirectory, protected)
		if err == nil && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
			return true
		}
	}
	return false
}

func protectedOperationalInputs(policy model.Policy) []string {
	paths := make([]string, 0, len(policy.MCP.Authorities)+4)
	for _, authority := range policy.MCP.Authorities {
		paths = append(paths, authority.TrustStoreFile)
	}
	if config := policy.Observability; config != nil {
		paths = append(paths, config.TokenFile, config.TrustStoreFile)
	}
	if config := policy.Updates; config != nil {
		paths = append(paths, config.TokenFile, config.TrustStoreFile)
	}
	return paths
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
	config := s.current()
	return validHandle(workload) && p.Authorize(config, workload, operation, key, s.clock())
}

func (s *Service) clock() time.Time {
	if s.now == nil {
		return time.Now()
	}
	return s.now()
}

func legacyScopeAllows(p Principal, operation string) bool {
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
	if value == "" || len(value) > 128 || value == "." || value == ".." {
		return false
	}
	for _, r := range value {
		if !handleRune(r) {
			return false
		}
	}
	return true
}

func handleRune(r rune) bool {
	return r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '_' || r == '-' || r == '.'
}

// ListWorkloads returns no source references or secret names.
func (s *Service) ListWorkloads(ctx context.Context, p Principal) ([]string, error) {
	if err := ctx.Err(); err != nil {
		return nil, fault.New(fault.Unavailable, mcpRequest)
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
func (s *Service) GetStatus(ctx context.Context, p Principal, workload string) (status Status, err error) {
	ctx, complete := observability.Start(ctx, observability.Operation{Phase: observability.PhaseMCP, ProviderKind: "mcp"})
	defer func() { complete(observability.OutcomeForError(err)) }()
	if ctx.Err() != nil {
		return Status{}, fault.New(fault.Unavailable, mcpRequest)
	}
	if !s.authorized(p, workload, "status", "") {
		return Status{}, fault.New(fault.Denied, "mcp status")
	}
	status, err := s.backend.Status(withPrincipal(ctx, p), workload)
	if err != nil {
		return Status{}, safeError(err, "mcp status")
	}
	if err := s.recheck(ctx, p, workload, "status", "", "mcp status revoked"); err != nil {
		return Status{}, err
	}
	status.Workload = workload
	status.Mode = s.current().Workloads[workload].Mode
	if !oneOf(status.State, "unprovisioned", "ready", "pending", "error", "running", "stopped", "unknown", "not_provisioned", "blocked", "resolved", "snapshotted", "applied", "activated", "apply_failed", "pending_activation", "partial_resolved", "partial_applied", "partial_activated", "partial_apply_failed", "partial_pending_activation") {
		status.State = "unknown"
	}
	return status, nil
}

// RunWorkload invokes only a locally enrolled and independently approved action.
func (s *Service) RunWorkload(ctx context.Context, p Principal, workload string) (result RunResult, err error) {
	ctx, complete := observability.Start(ctx, observability.Operation{Phase: observability.PhaseMCP, ProviderKind: "mcp"})
	defer func() { complete(observability.OutcomeForError(err)) }()
	if ctx.Err() != nil {
		return RunResult{}, fault.New(fault.Unavailable, mcpRequest)
	}
	w, err := s.authorizeRun(p, workload)
	if err != nil {
		return RunResult{}, err
	}
	if err := s.record(ctx, p, workload, "run", "", "authorized", w.AuditRequired); err != nil {
		return RunResult{}, err
	}
	if _, err := s.authorizeRun(p, workload); err != nil {
		return RunResult{}, err
	}
	if ctx.Err() != nil {
		return RunResult{}, fault.New(fault.Unavailable, mcpRequest)
	}
	result, err := s.backend.Run(withPrincipal(ctx, p), workload)
	if err != nil {
		return RunResult{}, safeError(err, mcpAction)
	}
	return sanitizeRunResult(result), nil
}

func (s *Service) authorizeRun(p Principal, workload string) (model.Workload, error) {
	if !s.authorized(p, workload, "run", "") || s.options.ActionAllowed == nil || !s.options.ActionAllowed(workload) {
		return model.Workload{}, fault.New(fault.Denied, mcpAction)
	}
	w := s.current().Workloads[workload]
	if w.UntrustedCode && !s.mayReadAll(p, workload, w) {
		return model.Workload{}, fault.New(fault.Denied, "mcp code disclosure")
	}
	// ActionAllowed may perform enrolled executable trust work. Recheck the
	// identity and current grant after it finishes, immediately before effect.
	if !s.authorized(p, workload, "run", "") {
		return model.Workload{}, fault.New(fault.Denied, mcpAction)
	}
	return w, nil
}

func sanitizeRunResult(result RunResult) RunResult {
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
	return result
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
func (s *Service) ReadSecret(ctx context.Context, p Principal, workload, key string) (result SecretResult, err error) {
	ctx, complete := observability.Start(ctx, observability.Operation{Phase: observability.PhaseMCP, ProviderKind: "mcp"})
	defer func() { complete(observability.OutcomeForError(err)) }()
	if ctx.Err() != nil {
		return SecretResult{}, fault.New(fault.Unavailable, mcpRequest)
	}
	if !validHandle(key) || !s.authorized(p, workload, "read", key) {
		return SecretResult{}, fault.New(fault.Denied, mcpDisclosure)
	}
	if s.options.Audit == nil {
		return SecretResult{}, fault.New(fault.Apply, "mcp disclosure audit")
	}
	value, err := s.backend.Read(withPrincipal(ctx, p), workload, key)
	if err != nil {
		return SecretResult{}, safeError(err, mcpDisclosure)
	}
	if err := s.validateDisclosure(workload, value); err != nil {
		return SecretResult{}, err
	}
	// The provider call may have blocked while an operator revoked the grant.
	// Recheck current policy before recording authorization to release plaintext.
	if err := s.recheck(ctx, p, workload, "read", key, "mcp disclosure revoked"); err != nil {
		return SecretResult{}, err
	}
	// This receipt proves durable release authorization, not client delivery.
	if err := s.record(ctx, p, workload, "read", key, "release_authorized", true); err != nil {
		return SecretResult{}, err
	}
	if err := s.recheck(ctx, p, workload, "read", key, "mcp disclosure revoked"); err != nil {
		return SecretResult{}, err
	}
	result = SecretResult{Alias: key, Value: value.Data, Version: value.Version}
	if !value.ExpiresAt.IsZero() {
		expiresAt := value.ExpiresAt
		result.ExpiresAt = &expiresAt
	}
	return result, nil
}

func (s *Service) recheck(ctx context.Context, p Principal, workload, operation, key, denied string) error {
	if !s.authorized(p, workload, operation, key) {
		return fault.New(fault.Denied, denied)
	}
	if ctx.Err() != nil {
		return fault.New(fault.Unavailable, mcpRequest)
	}
	return nil
}

func (s *Service) validateDisclosure(workload string, value model.Value) error {
	if value.Leased || value.Renewable {
		return fault.New(fault.Unsupported, "mcp leased disclosure")
	}
	if !value.ExpiresAt.IsZero() && !time.Now().Before(value.ExpiresAt) {
		return fault.New(fault.Expired, mcpDisclosure)
	}
	if len(value.Data) > 65536 {
		return fault.New(fault.Unsupported, "mcp value size")
	}
	if s.current().Workloads[workload].AuditRequired && !value.AuditVerified {
		return fault.New(fault.Denied, "mcp source audit")
	}
	return nil
}

// SecretResult is the sole protocol result allowed to carry plaintext.
type SecretResult struct {
	Alias     string     `json:"alias"`
	Value     string     `json:"value"`
	Version   string     `json:"version,omitempty"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
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
	server := sdk.NewServer(&sdk.Implementation{Name: "CircularQuay", Version: "1.0.0"}, &sdk.ServerOptions{
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
		if operation == "read" && s.canReadBinding(p, id, w) {
			return true
		}
	}
	return false
}

func (s *Service) canReadBinding(p Principal, workload string, w model.Workload) bool {
	for _, binding := range w.Bindings {
		if s.authorized(p, workload, "read", binding.Name) {
			return true
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
		if !decimalSIDPart(part) {
			return false
		}
	}
	return true
}

func decimalSIDPart(part string) bool {
	if part == "" {
		return false
	}
	for _, r := range part {
		if r < '0' || r > '9' {
			return false
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
