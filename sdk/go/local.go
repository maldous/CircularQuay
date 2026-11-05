package cq

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"math/big"
	"net/url"
	"path/filepath"
	"regexp"
	"runtime/debug"
	"strconv"
	"strings"
	"time"

	"github.com/maldous/CircularQuay/internal/cli"
	"github.com/maldous/CircularQuay/internal/jsondocument"
)

const (
	maxLocalJSONBytes  = 1 << 20
	maxIJSONDepth      = 32
	maxIJSONCollection = 1024
	maxIJSONNodes      = 16_384
)

const (
	circularQuayModule = "github.com/maldous/CircularQuay"
	developmentVersion = "1.0.0-dev"
	pointerFlag        = "--pointer"
	expectedSHA256Flag = "--expect-sha256"
	workloadFlag       = "--workload"
)

var (
	readBuildInfo   = debug.ReadBuildInfo
	officialV1Tag   = regexp.MustCompile(`^v1\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(?:-[0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*)?(?:\+[0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*)?$`)
	pseudoVersionV1 = regexp.MustCompile(`^v1\.[0-9]+\.[0-9]+-[0-9]+\.[0-9]{14}-[0-9a-f]{12}(?:\+[0-9A-Za-z.-]+)?$`)
	commitSetting   = regexp.MustCompile(`^[0-9a-fA-F]{7,64}$`)
	maxIJSONInteger = big.NewInt(9007199254740991)
)

// LocalOptions identifies one explicit local CQ installation. Both
// paths are required and absolute: the SDK never selects an ambient config or
// state directory.
type LocalOptions struct {
	ConfigPath     string
	StateDirectory string
}

// LocalClient calls the in-process CQ CLI runner. It is a bounded
// adapter, not a general command execution API. It never spawns a binary.
type LocalClient struct {
	configPath     string
	stateDirectory string
	observer       Observer
	run            localRun
}

type localRun func(context.Context, cli.Runner, []string) int

// LocalOption configures a LocalClient.
type LocalOption func(*LocalClient)

// WithLocalObserver installs a metadata-only observer for local operations.
func WithLocalObserver(observer Observer) LocalOption {
	return func(client *LocalClient) { client.observer = observer }
}

// NewLocal creates a local adapter with explicitly configured paths.
func NewLocal(options LocalOptions, clientOptions ...LocalOption) (*LocalClient, error) {
	if !absolutePath(options.ConfigPath) || !absolutePath(options.StateDirectory) {
		return nil, &Error{Category: CategoryInvalid}
	}
	client := &LocalClient{
		configPath:     filepath.Clean(options.ConfigPath),
		stateDirectory: filepath.Clean(options.StateDirectory),
		run: func(ctx context.Context, runner cli.Runner, arguments []string) int {
			return runner.Run(ctx, arguments)
		},
	}
	for _, option := range clientOptions {
		if option != nil {
			option(client)
		}
	}
	return client, nil
}

// Result is bounded JSON emitted by a non-interactive CQ CLI
// operation. It contains the CLI's public JSON DTO, never process output.
type Result struct {
	ExitCode int
	JSON     json.RawMessage
}

// EffectResult describes a command that might execute a workload. Output is
// discarded rather than captured, because it can contain child-process data.
type EffectResult struct{ ExitCode int }

// UpdateCheck performs the policy-configured, structured update-awareness
// check. It never downloads, installs, or activates software.
func (c *LocalClient) UpdateCheck(ctx context.Context) (Result, error) {
	return c.jsonCommand(ctx, "update_check", nil, "update", "check", "--json")
}

// ConfigInit creates the standard protected configuration and synthetic fixture
// at LocalOptions.ConfigPath. It fails if either destination already exists.
func (c *LocalClient) ConfigInit(ctx context.Context) (Result, error) {
	return c.jsonCommand(ctx, "config_init", nil, "config", "init", "--output", c.configPath)
}

// ConfigValidate validates the complete declared policy locally.
func (c *LocalClient) ConfigValidate(ctx context.Context) (Result, error) {
	return c.jsonCommand(ctx, "config_validate", nil, "config", "validate")
}

// ConfigGet retrieves one RFC 6901 configuration pointer.
func (c *LocalClient) ConfigGet(ctx context.Context, pointer string) (Result, error) {
	if !validPointer(pointer) {
		return Result{}, &Error{Category: CategoryInvalid}
	}
	return c.jsonCommand(ctx, "config_get", nil, "config", "get", pointerFlag, pointer)
}

// ConfigList lists entries at one RFC 6901 configuration pointer.
func (c *LocalClient) ConfigList(ctx context.Context, pointer string) (Result, error) {
	if !validPointer(pointer) {
		return Result{}, &Error{Category: CategoryInvalid}
	}
	return c.jsonCommand(ctx, "config_list", nil, "config", "list", pointerFlag, pointer)
}

// ConfigSet transactionally assigns one JSON value. ExpectedSHA256 is optional
// optimistic-concurrency protection supplied by the caller.
func (c *LocalClient) ConfigSet(ctx context.Context, pointer string, value json.RawMessage, expectedSHA256 string) (Result, error) {
	if !validPointer(pointer) || !validJSON(value) || !optionalDigest(expectedSHA256) {
		return Result{}, &Error{Category: CategoryInvalid}
	}
	// Keep JSON out of argv, where process listings and telemetry collectors may
	// expose it. bytes.NewReader below is deliberately a bounded non-TTY input.
	args := []string{"config", "set", pointerFlag, pointer, "--value-file", "-"}
	if expectedSHA256 != "" {
		args = append(args, expectedSHA256Flag, expectedSHA256)
	}
	return c.jsonCommand(ctx, "config_set", value, args...)
}

// ConfigDelete transactionally removes one JSON pointer.
func (c *LocalClient) ConfigDelete(ctx context.Context, pointer, expectedSHA256 string) (Result, error) {
	if pointer == "" || !validPointer(pointer) || !optionalDigest(expectedSHA256) {
		return Result{}, &Error{Category: CategoryInvalid}
	}
	args := []string{"config", "delete", pointerFlag, pointer}
	if expectedSHA256 != "" {
		args = append(args, expectedSHA256Flag, expectedSHA256)
	}
	return c.jsonCommand(ctx, "config_delete", nil, args...)
}

// ConfigReplace transactionally replaces the complete configuration document.
func (c *LocalClient) ConfigReplace(ctx context.Context, document json.RawMessage, expectedSHA256 string) (Result, error) {
	if !validJSON(document) || !optionalDigest(expectedSHA256) {
		return Result{}, &Error{Category: CategoryInvalid}
	}
	args := []string{"config", "replace", "--file", "-"}
	if expectedSHA256 != "" {
		args = append(args, expectedSHA256Flag, expectedSHA256)
	}
	return c.jsonCommand(ctx, "config_replace", document, args...)
}

// Doctor evaluates readiness without contacting a provider. Fix permits only
// the CLI's protected JSON formatting correction.
func (c *LocalClient) Doctor(ctx context.Context, fix bool) (Result, error) {
	args := []string{"doctor"}
	if fix {
		args = append(args, "--fix")
	}
	return c.jsonCommand(ctx, "doctor", nil, args...)
}

// Plan returns the local, non-secret declared workload plan.
func (c *LocalClient) Plan(ctx context.Context) (Result, error) {
	return c.jsonCommand(ctx, "plan", nil, "plan")
}

// Status returns the current redacted local workload state record.
func (c *LocalClient) Status(ctx context.Context, workload string) (Result, error) {
	if !validHandle(workload) {
		return Result{}, &Error{Category: CategoryInvalid}
	}
	return c.jsonCommand(ctx, "status", nil, "status", workloadFlag, workload, "--json")
}

// Snapshot resolves and stores a protected offline snapshot for one workload.
func (c *LocalClient) Snapshot(ctx context.Context, workload string) (Result, error) {
	return c.workloadJSON(ctx, "snapshot", workload, "")
}

// Sync resolves and synchronizes one workload once through its declared targets.
func (c *LocalClient) Sync(ctx context.Context, workload, target string) (Result, error) {
	return c.workloadJSON(ctx, "sync", workload, target)
}

// Apply resolves and applies one workload to one explicit target.
func (c *LocalClient) Apply(ctx context.Context, workload, target string) (Result, error) {
	if target == "" {
		return Result{}, &Error{Category: CategoryInvalid}
	}
	return c.workloadJSON(ctx, "apply", workload, target)
}

// RunWorkload runs one declared workload. The adapter deliberately discards
// child stdout and stderr and returns only its exit code.
func (c *LocalClient) RunWorkload(ctx context.Context, workload string) (result EffectResult, err error) {
	started := now()
	defer func() { c.observe("run_workload", outcome(err), since(started)) }()
	if !validHandle(workload) {
		return EffectResult{}, &Error{Category: CategoryInvalid}
	}
	if err = validContext(ctx); err != nil {
		return EffectResult{}, err
	}
	code := c.invoke(ctx, nil, io.Discard, io.Discard, "run", workloadFlag, workload)
	if err = ctx.Err(); err != nil {
		return EffectResult{}, normalize(err)
	}
	return EffectResult{ExitCode: code}, nil
}

// BootstrapRequest pins a configuration document before it is materialized.
// Every field is explicit so local environment defaults are not consulted.
type BootstrapRequest struct {
	SourceURL string
	Ref       string
	Path      string
	SHA256    string
}

// Bootstrap materializes a pinned configuration at LocalOptions.ConfigPath.
func (c *LocalClient) Bootstrap(ctx context.Context, request BootstrapRequest) (Result, error) {
	if !validConfigSource(request.SourceURL) || request.Path == "" || !exactDigest(request.Ref, 40) || !exactDigest(request.SHA256, 64) {
		return Result{}, &Error{Category: CategoryInvalid}
	}
	return c.jsonCommand(ctx, "bootstrap", nil, "bootstrap", "--config-source", request.SourceURL, "--config-ref", request.Ref, "--config-path", request.Path, "--config-sha256", request.SHA256, "--output", c.configPath, "--non-interactive")
}

// FlowRequest is an explicit pinned configuration invocation for a supported
// CI runner context. The runner itself remains responsible for its identity.
type FlowRequest struct {
	Workload string
	BootstrapRequest
}

// Flow runs one workload under the CLI's supported CI context rules. Child
// output is discarded and only the exit code is returned.
func (c *LocalClient) Flow(ctx context.Context, request FlowRequest) (result EffectResult, err error) {
	started := now()
	defer func() { c.observe("flow", outcome(err), since(started)) }()
	if !validHandle(request.Workload) || !validConfigSource(request.SourceURL) || request.Path == "" || !exactDigest(request.Ref, 40) || !exactDigest(request.SHA256, 64) {
		return EffectResult{}, &Error{Category: CategoryInvalid}
	}
	if err = validContext(ctx); err != nil {
		return EffectResult{}, err
	}
	args := []string{"flow", "run", workloadFlag, request.Workload, "--config-source", request.SourceURL, "--config-ref", request.Ref, "--config-path", request.Path, "--config-sha256", request.SHA256, "--non-interactive"}
	code := c.invoke(ctx, nil, io.Discard, io.Discard, args...)
	if err = ctx.Err(); err != nil {
		return EffectResult{}, normalize(err)
	}
	return EffectResult{ExitCode: code}, nil
}

func (c *LocalClient) workloadJSON(ctx context.Context, command, workload, target string) (Result, error) {
	if !validHandle(workload) || (target != "" && !validHandle(target)) {
		return Result{}, &Error{Category: CategoryInvalid}
	}
	args := []string{command, workloadFlag, workload}
	if target != "" {
		args = append(args, "--target", target)
	}
	return c.jsonCommand(ctx, command, nil, args...)
}

func (c *LocalClient) jsonCommand(ctx context.Context, operation string, input []byte, command ...string) (result Result, err error) {
	started := now()
	defer func() { c.observe(operation, outcome(err), since(started)) }()
	if err = validContext(ctx); err != nil {
		return Result{}, err
	}
	output := &boundedBuffer{limit: maxLocalJSONBytes}
	failure := &boundedBuffer{limit: 512}
	code := c.invoke(ctx, input, output, failure, command...)
	if err = ctx.Err(); err != nil {
		return Result{}, normalize(err)
	}
	if output.overflow || failure.overflow {
		return Result{}, &Error{Category: CategoryProtocol}
	}
	if code != 0 {
		if len(output.Bytes()) != 0 {
			return Result{}, localFailure(output.Bytes())
		}
		if len(failure.Bytes()) == 0 {
			return Result{}, &Error{Category: CategoryProcess}
		}
		return Result{}, localFailure(failure.Bytes())
	}
	if !json.Valid(output.Bytes()) {
		return Result{}, &Error{Category: CategoryProtocol}
	}
	return Result{ExitCode: code, JSON: append(json.RawMessage(nil), output.Bytes()...)}, nil
}

func (c *LocalClient) invoke(ctx context.Context, input []byte, output, errors io.Writer, command ...string) int {
	if c == nil || c.run == nil {
		return 1
	}
	args := make([]string, 0, len(command)+4)
	args = append(args, "--config", c.configPath, "--state-dir", c.stateDirectory)
	args = append(args, command...)
	identity := sdkBuildIdentity()
	// bytes.Reader is never a terminal. A CLI invocation therefore cannot
	// enter an interactive credential or confirmation prompt through this SDK.
	return c.run(ctx, cli.Runner{In: bytes.NewReader(input), Out: output, Err: errors, BuildVersion: identity.version, BuildCommit: identity.commit}, args)
}

type buildIdentity struct{ version, commit string }

// sdkBuildIdentity examines only Go's embedded build metadata for this exact
// module. It is version-routing data, not a signature or provenance claim.
func sdkBuildIdentity() buildIdentity {
	identity := buildIdentity{version: developmentVersion}
	info, ok := readBuildInfo()
	if !ok || info == nil {
		return identity
	}
	identity.version = sdkVersion(info)
	identity.commit = sdkCommit(info.Settings)
	return identity
}

// sdkVersion accepts only the SDK's own tagged module metadata; a pseudo or
// unrelated dependency version must not become the runtime policy identity.
func sdkVersion(info *debug.BuildInfo) string {
	if validSDKModule(info.Main) {
		return strings.TrimPrefix(info.Main.Version, "v")
	}
	for _, module := range info.Deps {
		if module != nil && validSDKModule(*module) {
			return strings.TrimPrefix(module.Version, "v")
		}
	}
	return developmentVersion
}

func validSDKModule(module debug.Module) bool {
	return module.Path == circularQuayModule && validOfficialV1Tag(module.Version)
}

func sdkCommit(settings []debug.BuildSetting) string {
	for _, setting := range settings {
		if setting.Key == "vcs.revision" && commitSetting.MatchString(setting.Value) {
			return setting.Value
		}
	}
	return ""
}

func validOfficialV1Tag(version string) bool {
	return officialV1Tag.MatchString(version) && !pseudoVersionV1.MatchString(version)
}

func localFailure(data []byte) error {
	var response map[string]string
	if jsondocument.Decode(data, &response) != nil || len(response) != 1 {
		return &Error{Category: CategoryProtocol}
	}
	value, found := response["error"]
	if !found {
		return &Error{Category: CategoryProtocol}
	}
	switch value {
	case "invalid_policy":
		return &Error{Category: CategoryInvalid}
	case "access_denied":
		return &Error{Category: CategoryDenied}
	case "trust_failure":
		return &Error{Category: CategoryTrust}
	case "expired_value":
		return &Error{Category: CategoryExpired}
	case "unsupported_capability":
		return &Error{Category: CategoryUnsupported}
	case "unavailable", "activation_failed", "missing_value":
		return &Error{Category: CategoryUnavailable}
	default:
		return &Error{Category: CategoryProtocol}
	}
}

func (c *LocalClient) observe(operation, result string, duration time.Duration) {
	if c == nil || c.observer == nil {
		return
	}
	defer func() { _ = recover() }()
	c.observer.Observe(Event{Operation: operation, Outcome: result, Duration: duration})
}

type boundedBuffer struct {
	bytes.Buffer
	limit    int
	overflow bool
}

func (b *boundedBuffer) Write(value []byte) (int, error) {
	if b.overflow || len(value)+b.Len() > b.limit {
		b.overflow = true
		return len(value), nil
	}
	return b.Buffer.Write(value)
}

func absolutePath(path string) bool { return path != "" && filepath.IsAbs(path) }

func validJSON(value []byte) bool {
	if len(value) == 0 || len(value) > maxLocalJSONBytes {
		return false
	}
	var document any
	if jsondocument.Decode(value, &document) != nil {
		return false
	}
	decoder := json.NewDecoder(bytes.NewReader(value))
	decoder.UseNumber()
	if decoder.Decode(&document) != nil {
		return false
	}
	return validIJSON(document, 0, new(int))
}

func validIJSON(value any, depth int, nodes *int) bool {
	if depth > maxIJSONDepth || nodes == nil {
		return false
	}
	*nodes = *nodes + 1
	if *nodes > maxIJSONNodes {
		return false
	}
	switch candidate := value.(type) {
	case nil, string, bool:
		return true
	case json.Number:
		return validIJSONNumber(candidate)
	case []any:
		return validIJSONArray(candidate, depth, nodes)
	case map[string]any:
		return validIJSONObject(candidate, depth, nodes)
	default:
		return false
	}
}

// validIJSONNumber preserves cross-SDK numeric portability without accepting
// rounded integers outside I-JSON's exact interoperable range.
func validIJSONNumber(value json.Number) bool {
	if _, err := strconv.ParseFloat(value.String(), 64); err != nil {
		return false
	}
	rational, ok := new(big.Rat).SetString(value.String())
	if !ok || !rational.IsInt() {
		return ok
	}
	return new(big.Int).Abs(rational.Num()).Cmp(maxIJSONInteger) <= 0
}

func validIJSONArray(values []any, depth int, nodes *int) bool {
	if len(values) > maxIJSONCollection {
		return false
	}
	for _, value := range values {
		if !validIJSON(value, depth+1, nodes) {
			return false
		}
	}
	return true
}

func validIJSONObject(values map[string]any, depth int, nodes *int) bool {
	if len(values) > maxIJSONCollection {
		return false
	}
	for _, value := range values {
		if !validIJSON(value, depth+1, nodes) {
			return false
		}
	}
	return true
}

func validPointer(pointer string) bool {
	if pointer == "" {
		return true
	}
	if !strings.HasPrefix(pointer, "/") || len(pointer) > 4096 {
		return false
	}
	for index := 0; index < len(pointer); index++ {
		if pointer[index] != '~' {
			continue
		}
		if index+1 >= len(pointer) || pointer[index+1] != '0' && pointer[index+1] != '1' {
			return false
		}
		index++
	}
	return true
}

func optionalDigest(value string) bool { return value == "" || exactDigest(value, 64) }

func exactDigest(value string, length int) bool {
	if len(value) != length {
		return false
	}
	for _, r := range value {
		if !(r >= '0' && r <= '9' || r >= 'a' && r <= 'f') {
			return false
		}
	}
	return true
}

func validConfigSource(value string) bool {
	if len(value) == 0 || len(value) > 4096 {
		return false
	}
	parsed, err := url.Parse(value)
	if err != nil {
		return false
	}
	return validConfigSourceURL(parsed, value)
}

func validConfigSourceURL(parsed *url.URL, original string) bool {
	if !strings.EqualFold(parsed.Scheme, "https") {
		return false
	}
	if !parsed.IsAbs() {
		return false
	}
	if parsed.Host == "" {
		return false
	}
	if parsed.User != nil {
		return false
	}
	if parsed.RawQuery != "" {
		return false
	}
	if parsed.ForceQuery {
		return false
	}
	if parsed.Fragment != "" {
		return false
	}
	if strings.Contains(original, "#") {
		return false
	}
	return parsed.Opaque == ""
}

var now = func() time.Time { return time.Now() }
var since = time.Since
