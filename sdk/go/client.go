// Package cquay is the supported Go client for CircularQuay's optional MCP
// interface. It wraps a caller-owned official MCP SDK session; it does not
// discover credentials, start a process, or create a transport.
package cquay

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"regexp"
	"time"
	"unicode/utf8"

	mcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

const (
	maxStructuredResponseBytes = 1 << 20
	maxWorkloads               = 4096
	maxTextLength              = 256
	maxProviderVersionBytes    = 4096
	maxSecretValueBytes        = 65536
	maxExpiryTimestampBytes    = 64
)

var rfc3339Timestamp = regexp.MustCompile(`^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}(?:\.[0-9]+)?(?:Z|[+-][0-9]{2}:[0-9]{2})$`)

// Category is a safe, stable classification of an SDK failure. It deliberately
// does not retain server error content, request arguments, endpoints, tokens,
// or secret values.
type Category string

const (
	CategoryInvalid     Category = "invalid"
	CategoryDenied      Category = "denied"
	CategoryTrust       Category = "trust"
	CategoryUnavailable Category = "unavailable"
	CategoryExpired     Category = "expired"
	CategoryUnsupported Category = "unsupported"
	CategoryProtocol    Category = "protocol"
	CategoryCancelled   Category = "cancelled"
	CategoryIntegrity   Category = "integrity"
	CategoryProcess     Category = "process"
)

// Error reports a safe error category. It is intentionally not a wrapper for
// the underlying protocol error, because that error may contain sensitive
// response content.
type Error struct{ Category Category }

func (e *Error) Error() string { return "cquay: " + string(e.Category) }

// Event is metadata-only operation telemetry. It contains no tool arguments,
// identifiers, endpoint URLs, credentials, or values.
type Event struct {
	Operation string
	Outcome   string
	Duration  time.Duration
}

// Observer receives best-effort metadata-only SDK events. An observer cannot
// influence a completed operation; panics are contained by the SDK.
type Observer interface{ Observe(Event) }

// Option configures an SDK client.
type Option func(*Client)

// WithObserver installs a metadata-only observer. The observer is optional.
func WithObserver(observer Observer) Option {
	return func(c *Client) { c.observer = observer }
}

// Client exposes the current CircularQuay MCP tools. It owns neither the MCP
// session nor its lifecycle: callers create, authenticate, configure, and
// close the official MCP Go SDK session themselves.
type Client struct {
	caller   caller
	observer Observer
}

type caller interface {
	ListTools(context.Context, *mcp.ListToolsParams) (*mcp.ListToolsResult, error)
	CallTool(context.Context, *mcp.CallToolParams) (*mcp.CallToolResult, error)
}

type officialSession struct{ session *mcp.ClientSession }

func (s officialSession) ListTools(ctx context.Context, p *mcp.ListToolsParams) (*mcp.ListToolsResult, error) {
	return s.session.ListTools(ctx, p)
}

func (s officialSession) CallTool(ctx context.Context, p *mcp.CallToolParams) (*mcp.CallToolResult, error) {
	return s.session.CallTool(ctx, p)
}

// New adapts an existing official MCP Go SDK session. The session must already
// be connected over a transport chosen and configured by the caller.
func New(session *mcp.ClientSession, options ...Option) (*Client, error) {
	if session == nil {
		return nil, &Error{Category: CategoryInvalid}
	}
	return newClient(officialSession{session: session}, options...)
}

func newClient(call caller, options ...Option) (*Client, error) {
	if call == nil {
		return nil, &Error{Category: CategoryInvalid}
	}
	client := &Client{caller: call}
	for _, option := range options {
		if option != nil {
			option(client)
		}
	}
	return client, nil
}

// Tool is a bounded public description of a currently visible MCP tool.
type Tool struct {
	Name        string
	Description string
}

// DiscoverTools lists the CircularQuay MCP tools available to the caller's
// already-authenticated session. It neither caches nor follows pagination;
// current CircularQuay v1 responses are one bounded page.
func (c *Client) DiscoverTools(ctx context.Context) (tools []Tool, err error) {
	started := time.Now()
	defer func() { c.observe("discover_tools", outcome(err), time.Since(started)) }()
	result, err := c.listTools(ctx)
	if err != nil {
		return nil, err
	}
	return parseToolCatalog(result)
}

func (c *Client) listTools(ctx context.Context) (*mcp.ListToolsResult, error) {
	if err := validContext(ctx); err != nil {
		return nil, err
	}
	result, err := c.caller.ListTools(ctx, &mcp.ListToolsParams{})
	if err != nil {
		return nil, normalize(err)
	}
	if err := ctx.Err(); err != nil {
		return nil, normalize(err)
	}
	return result, nil
}

func parseToolCatalog(result *mcp.ListToolsResult) ([]Tool, error) {
	if result == nil || result.NextCursor != "" || len(result.Tools) > 4 {
		return nil, &Error{Category: CategoryProtocol}
	}
	tools := make([]Tool, 0, len(result.Tools))
	seen := make(map[string]struct{}, len(result.Tools))
	for _, tool := range result.Tools {
		parsed, err := parseCatalogTool(tool, seen)
		if err != nil {
			return nil, err
		}
		tools = append(tools, parsed)
	}
	return tools, nil
}

func parseCatalogTool(tool *mcp.Tool, seen map[string]struct{}) (Tool, error) {
	if tool == nil || !knownTool(tool.Name) || !validHandle(tool.Name) || !boundedText(tool.Description) {
		return Tool{}, &Error{Category: CategoryProtocol}
	}
	if _, duplicate := seen[tool.Name]; duplicate {
		return Tool{}, &Error{Category: CategoryProtocol}
	}
	seen[tool.Name] = struct{}{}
	return Tool{Name: tool.Name, Description: tool.Description}, nil
}

// ListWorkloads returns the exact workload IDs visible to the session.
func (c *Client) ListWorkloads(ctx context.Context) (workloads []string, err error) {
	started := time.Now()
	defer func() { c.observe("list_workloads", outcome(err), time.Since(started)) }()
	var response struct {
		Workloads []string `json:"workloads"`
	}
	if err = c.call(ctx, "list_workloads", map[string]any{}, &response); err != nil {
		return nil, err
	}
	if len(response.Workloads) > maxWorkloads {
		return nil, &Error{Category: CategoryProtocol}
	}
	for index, workload := range response.Workloads {
		if !validHandle(workload) || index > 0 && response.Workloads[index-1] >= workload {
			return nil, &Error{Category: CategoryProtocol}
		}
	}
	return response.Workloads, nil
}

// Status is bounded, non-secret workload state returned by get_status.
type Status struct {
	Workload string `json:"workload"`
	Mode     string `json:"mode"`
	State    string `json:"state"`
}

// GetStatus returns safe state for one exact enrolled workload.
func (c *Client) GetStatus(ctx context.Context, workload string) (status Status, err error) {
	started := time.Now()
	defer func() { c.observe("get_status", outcome(err), time.Since(started)) }()
	if !validHandle(workload) {
		return Status{}, &Error{Category: CategoryInvalid}
	}
	if err = c.call(ctx, "get_status", map[string]any{"workload": workload}, &status); err != nil {
		return Status{}, err
	}
	if status.Workload != workload || !knownMode(status.Mode) || !knownState(status.State) {
		return Status{}, &Error{Category: CategoryProtocol}
	}
	return status, nil
}

// RunResult contains the bounded result of a locally enrolled action. It never
// exposes process output.
type RunResult struct {
	ID       string `json:"id"`
	Status   string `json:"status"`
	ExitCode int    `json:"exit_code"`
}

// RunWorkload invokes one exact locally enrolled workload action. It does not
// retry, because an action may already have taken effect.
func (c *Client) RunWorkload(ctx context.Context, workload string) (result RunResult, err error) {
	started := time.Now()
	defer func() { c.observe("run_workload", outcome(err), time.Since(started)) }()
	if !validHandle(workload) {
		return RunResult{}, &Error{Category: CategoryInvalid}
	}
	if err = c.call(ctx, "run_workload", map[string]any{"workload": workload}, &result); err != nil {
		return RunResult{}, err
	}
	if !validHandle(result.ID) || !knownRunStatus(result.Status) || result.ExitCode < 0 || result.ExitCode > 255 {
		return RunResult{}, &Error{Category: CategoryProtocol}
	}
	return result, nil
}

// AllowPlaintext is an explicit marker required before a client can request a
// plaintext secret through MCP. It does not grant access; the server policy
// still decides whether a request may be disclosed.
type AllowPlaintext struct{}

// Disclosure returns a client capable of requesting plaintext values. Calling
// it makes the disclosure decision visible in source review.
func (c *Client) Disclosure(AllowPlaintext) *DisclosureClient { return &DisclosureClient{client: c} }

// DisclosureClient is the explicit plaintext-disclosure surface.
type DisclosureClient struct{ client *Client }

// Secret is a deliberately redacted secret result. Value exposes plaintext only
// to the direct caller. The SDK makes no claim that Go memory is zeroized.
type Secret struct {
	Alias     string
	Version   string
	ExpiresAt time.Time
	value     string
}

// Value returns the plaintext delivered by the explicitly requested MCP tool.
func (s Secret) Value() string { return s.value }

// String never returns the plaintext secret.
func (Secret) String() string { return "[cquay secret redacted]" }

// GoString never returns the plaintext secret.
func (Secret) GoString() string { return "cquay.Secret{Value:[redacted]}" }

// MarshalJSON serializes secret metadata and a redaction marker, never the
// plaintext value.
func (s Secret) MarshalJSON() ([]byte, error) {
	var expiresAt *time.Time
	if !s.ExpiresAt.IsZero() {
		value := s.ExpiresAt
		expiresAt = &value
	}
	return json.Marshal(struct {
		Alias     string     `json:"alias"`
		Version   string     `json:"version,omitempty"`
		ExpiresAt *time.Time `json:"expires_at,omitempty"`
		Value     string     `json:"value"`
	}{Alias: s.Alias, Version: s.Version, ExpiresAt: expiresAt, Value: "[redacted]"})
}

// ReadSecret explicitly requests one policy-authorized plaintext value.
func (d *DisclosureClient) ReadSecret(ctx context.Context, workload, key string) (secret Secret, err error) {
	started := time.Now()
	if d == nil || d.client == nil {
		return Secret{}, &Error{Category: CategoryInvalid}
	}
	defer func() { d.client.observe("read_secret", outcome(err), time.Since(started)) }()
	if !validHandle(workload) || !validHandle(key) {
		return Secret{}, &Error{Category: CategoryInvalid}
	}
	response, err := d.requestSecret(ctx, workload, key)
	if err != nil {
		return Secret{}, err
	}
	return secretFromResponse(response, key)
}

type secretResponse struct {
	Alias     string          `json:"alias"`
	Value     string          `json:"value"`
	Version   string          `json:"version,omitempty"`
	ExpiresAt json.RawMessage `json:"expires_at,omitempty"`
}

func (d *DisclosureClient) requestSecret(ctx context.Context, workload, key string) (secretResponse, error) {
	var response secretResponse
	if err := d.client.call(ctx, "read_secret", map[string]any{"workload": workload, "key": key}, &response); err != nil {
		return secretResponse{}, err
	}
	return response, nil
}

func secretFromResponse(response secretResponse, key string) (Secret, error) {
	expiresAt, expiryErr := parseSecretExpiry(response.ExpiresAt)
	if expiryErr != nil {
		return Secret{}, &Error{Category: CategoryProtocol}
	}
	if response.Alias != key || !validSecretValue(response.Value) || !validProviderVersion(response.Version) {
		return Secret{}, &Error{Category: CategoryProtocol}
	}
	if !expiresAt.IsZero() && !time.Now().Before(expiresAt) {
		return Secret{}, &Error{Category: CategoryExpired}
	}
	return Secret{Alias: response.Alias, Version: response.Version, ExpiresAt: expiresAt, value: response.Value}, nil
}

func (c *Client) call(ctx context.Context, name string, arguments map[string]any, target any) error {
	if err := validContext(ctx); err != nil {
		return err
	}
	result, err := c.caller.CallTool(ctx, &mcp.CallToolParams{Name: name, Arguments: arguments})
	if err != nil {
		return normalize(err)
	}
	if err := ctx.Err(); err != nil {
		return normalize(err)
	}
	if result == nil || result.IsError || result.StructuredContent == nil {
		return &Error{Category: CategoryDenied}
	}
	encoded, err := json.Marshal(result.StructuredContent)
	if err != nil || len(encoded) > maxStructuredResponseBytes {
		return &Error{Category: CategoryProtocol}
	}
	decoder := json.NewDecoder(bytes.NewReader(encoded))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return &Error{Category: CategoryProtocol}
	}
	return nil
}

func (c *Client) observe(operation, result string, duration time.Duration) {
	if c == nil || c.observer == nil {
		return
	}
	defer func() { _ = recover() }()
	c.observer.Observe(Event{Operation: operation, Outcome: result, Duration: duration})
}

func validContext(ctx context.Context) error {
	if ctx == nil {
		return &Error{Category: CategoryInvalid}
	}
	if err := ctx.Err(); err != nil {
		return normalize(err)
	}
	return nil
}

func normalize(err error) error {
	if errors.Is(err, context.Canceled) {
		return &Error{Category: CategoryCancelled}
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return &Error{Category: CategoryUnavailable}
	}
	var networkError net.Error
	if errors.As(err, &networkError) {
		return &Error{Category: CategoryUnavailable}
	}
	return &Error{Category: CategoryProtocol}
}

func outcome(err error) string {
	if err == nil {
		return "success"
	}
	var safe *Error
	if errors.As(err, &safe) {
		return string(safe.Category)
	}
	return "unavailable"
}

func knownTool(name string) bool {
	return name == "list_workloads" || name == "get_status" || name == "run_workload" || name == "read_secret"
}

func validHandle(value string) bool {
	if value == "" || len(value) > 128 || value == "." || value == ".." {
		return false
	}
	for _, r := range value {
		if !validHandleRune(r) {
			return false
		}
	}
	return true
}

// validHandleRune keeps handle grammar separate from the bounded whole-value
// checks so every accepted identifier is assessed by the same character rule.
func validHandleRune(r rune) bool {
	return r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '_' || r == '-' || r == '.'
}

func boundedText(value string) bool { return len(value) <= maxTextLength }

// validProviderVersion accepts the provider's opaque version identifier. It is
// deliberately not a CircularQuay handle: providers use paths, timestamps,
// and other provider-specific syntaxes. C0 controls and DEL are excluded so
// the metadata remains safe for bounded structured use.
func validProviderVersion(value string) bool {
	if len(value) > maxProviderVersionBytes || !utf8.ValidString(value) {
		return false
	}
	for _, r := range value {
		if r <= 0x1f || r == 0x7f {
			return false
		}
	}
	return true
}

func validSecretValue(value string) bool {
	return len(value) <= maxSecretValueBytes && utf8.ValidString(value)
}

func parseSecretExpiry(raw json.RawMessage) (time.Time, error) {
	if len(raw) == 0 || bytes.Equal(raw, []byte("null")) {
		return time.Time{}, nil
	}
	var value string
	if json.Unmarshal(raw, &value) != nil || len(value) == 0 || len(value) > maxExpiryTimestampBytes || !rfc3339Timestamp.MatchString(value) {
		return time.Time{}, errors.New("invalid timestamp")
	}
	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return time.Time{}, errors.New("invalid timestamp")
	}
	return parsed, nil
}

func knownMode(value string) bool {
	return value == "online" || value == "offline" || value == "hybrid"
}

func knownState(value string) bool {
	return map[string]bool{"unprovisioned": true, "ready": true, "pending": true, "error": true, "running": true, "stopped": true, "unknown": true, "not_provisioned": true, "blocked": true, "resolved": true, "snapshotted": true, "applied": true, "activated": true, "apply_failed": true, "pending_activation": true, "partial_resolved": true, "partial_applied": true, "partial_activated": true, "partial_apply_failed": true, "partial_pending_activation": true}[value]
}

func knownRunStatus(value string) bool {
	return value == "completed" || value == "failed" || value == "running" || value == "cancelled"
}

var _ fmt.Stringer = Secret{}
