package cli

import (
	"context"
	"encoding/json"
	"flag"
	"io"
	"os"
	"path/filepath"

	"github.com/maldous/CircularQuay/internal/bootstrap"
	"github.com/maldous/CircularQuay/internal/fault"
	"github.com/maldous/CircularQuay/internal/platform"
	"github.com/maldous/CircularQuay/internal/policy"
)

const temporaryFlowPolicy = "policy.json"

var newFlowTempDir = os.MkdirTemp

type flowOptions struct {
	workload string
	source   bootstrapOptions
	explicit bool
}

type ciRunner string

const (
	ciGitHub ciRunner = "github-actions"
	ciGitLab ciRunner = "gitlab-ci"
	ciAzure  ciRunner = "azure-pipelines"
)

// flow runs one declared workload in an exact supported CI context. Context
// markers identify an execution environment only: they are never treated as a
// verified workload identity or an authorization grant.
func (a application) flow(ctx context.Context, args []string) (int, error) {
	return a.flowWithFetcher(ctx, args, bootstrap.NewFetcher(nil).Fetch)
}

func (a application) flowWithFetcher(ctx context.Context, args []string, fetch bootstrapFetch) (int, error) {
	options, err := a.flowOptions(args)
	if err != nil {
		return 0, err
	}
	runner, err := detectCIRunner()
	if err != nil {
		return 0, err
	}
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	if options.explicit || (!a.configExplicit && flowEnvironmentSourcePresent()) {
		return a.flowRemote(ctx, options, runner, fetch)
	}
	if localPolicyExists(a.config) {
		return a.flowLocal(ctx, options.workload, runner, "local", "", "")
	}
	// A missing enrolled configuration fails closed. A runner's ambient
	// variables are not an enrollment or a configuration trust anchor.
	return 0, fault.New(fault.Denied, "configuration discovery unavailable")
}

func (a application) flowOptions(args []string) (flowOptions, error) {
	if len(args) == 0 || args[0] != "run" {
		return flowOptions{}, fault.New(fault.Invalid, "flow command")
	}
	f := flag.NewFlagSet("flow run", flag.ContinueOnError)
	f.SetOutput(io.Discard)
	workload := f.String("workload", "", "declared workload ID")
	source := f.String("config-source", os.Getenv(envCQUAYConfigSource), "immutable GitHub configuration source")
	ref := f.String("config-ref", os.Getenv(envCQUAYConfigRef), "full immutable Git commit")
	path := f.String("config-path", configPathDefault(), "configuration path in the repository")
	digest := f.String("config-sha256", os.Getenv(envCQUAYConfigSHA256), "required configuration SHA-256")
	_ = f.Bool("non-interactive", false, "flow never prompts")
	if f.Parse(args[1:]) != nil || f.NArg() != 0 || *workload == "" {
		return flowOptions{}, fault.New(fault.Invalid, "flow arguments")
	}
	explicit := flagWasSet(f, "config-source") || flagWasSet(f, "config-ref") || flagWasSet(f, "config-path") || flagWasSet(f, "config-sha256")
	return flowOptions{workload: *workload, source: bootstrapOptions{sourceURL: *source, ref: *ref, path: *path, digest: *digest}, explicit: explicit}, nil
}

func (a application) flowRemote(ctx context.Context, options flowOptions, runner ciRunner, fetch bootstrapFetch) (int, error) {
	source := options.source
	result, err := fetchFlowPolicy(ctx, source, fetch)
	if err != nil {
		return 0, err
	}
	parent, err := flowTemporaryParent()
	if err != nil {
		return 0, err
	}
	directory, err := newFlowTempDir(parent, "cquay-flow-")
	if err != nil {
		return 0, fault.New(fault.Apply, "flow temporary policy")
	}
	defer os.RemoveAll(directory)
	if err = platform.EnsurePrivateDirectory(directory); err != nil {
		return 0, err
	}
	config := filepath.Join(directory, temporaryFlowPolicy)
	if _, err = policy.Parse(result.Bytes, config); err != nil {
		return 0, err
	}
	if err = platform.CreateProtectedExclusive(config, result.Bytes); err != nil {
		return 0, err
	}
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	return (application{config: config, directory: a.directory, runner: a.runner, repository: a.repository}).flowLocal(ctx, options.workload, runner, "pinned", source.ref, source.digest)
}

// fetchFlowPolicy validates every remote pin before any retrieval. A canceled
// request cannot turn a late fetch response into a materialized policy.
func fetchFlowPolicy(ctx context.Context, source bootstrapOptions, fetch bootstrapFetch) (bootstrap.Result, error) {
	if source.sourceURL == "" || !pinnedBootstrapRequest(source.ref, source.digest) || fetch == nil {
		return bootstrap.Result{}, fault.New(fault.Invalid, "flow configuration pin")
	}
	if err := ctx.Err(); err != nil {
		return bootstrap.Result{}, err
	}
	result, err := fetch(ctx, bootstrap.Request{SourceURL: source.sourceURL, ConfigRef: source.ref, ConfigPath: source.path, ExpectedSHA256: source.digest})
	if err != nil {
		return bootstrap.Result{}, err
	}
	if err := ctx.Err(); err != nil {
		return bootstrap.Result{}, err
	}
	return result, nil
}

func (a application) flowLocal(ctx context.Context, workload string, runner ciRunner, sourceKind, ref, digest string) (int, error) {
	if err := a.flowReceipt(runner, sourceKind, workload, ref, digest); err != nil {
		return 0, err
	}
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	return a.workloadCommand(ctx, []string{"run", "--workload", workload})
}

// flowReceipt records non-secret provenance before child dispatch. The runner
// context is an environment hint, never an authenticated workload identity.
func (a application) flowReceipt(runner ciRunner, sourceKind, workload, ref, digest string) error {
	if a.runner.Err == nil {
		return fault.New(fault.Apply, "flow receipt")
	}
	receipt := map[string]string{"runner_context": string(runner), "runner_context_trust": "hint_not_identity", "source_kind": sourceKind, "workload": workload}
	if sourceKind == "pinned" {
		receipt["config_ref"] = ref
		receipt["config_sha256"] = digest
	}
	if err := json.NewEncoder(a.runner.Err).Encode(receipt); err != nil {
		return fault.New(fault.Apply, "flow receipt")
	}
	return nil
}

func flowEnvironmentSourcePresent() bool {
	for _, key := range []string{envCQUAYConfigSource, envCQUAYConfigRef, envCQUAYConfigPath, envCQUAYConfigSHA256} {
		if _, found := os.LookupEnv(key); found {
			return true
		}
	}
	return false
}

func flowTemporaryParent() (string, error) {
	parent, err := filepath.EvalSymlinks(os.TempDir())
	if err != nil {
		return "", fault.New(fault.Apply, "flow temporary policy")
	}
	parent, err = filepath.Abs(parent)
	if err != nil {
		return "", fault.New(fault.Apply, "flow temporary policy")
	}
	return parent, nil
}

func localPolicyExists(path string) bool {
	_, err := os.Lstat(path)
	return err == nil
}

func detectCIRunner() (ciRunner, error) {
	markers := []struct {
		key, value string
		runner     ciRunner
	}{
		{"GITHUB_ACTIONS", "true", ciGitHub},
		{"GITLAB_CI", "true", ciGitLab},
		{"TF_BUILD", "True", ciAzure},
	}
	var detected ciRunner
	for _, marker := range markers {
		value, present := os.LookupEnv(marker.key)
		if !present {
			continue
		}
		if value != marker.value || detected != "" {
			return "", fault.New(fault.Invalid, "ci context")
		}
		detected = marker.runner
	}
	if detected == "" {
		return "", fault.New(fault.Invalid, "ci context")
	}
	return detected, nil
}
