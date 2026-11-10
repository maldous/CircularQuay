package cli

import (
	"context"
	"flag"
	"io"
	"os"
	"path/filepath"

	"github.com/maldous/CircularQuay/internal/bootstrap"
	"github.com/maldous/CircularQuay/internal/fault"
	"github.com/maldous/CircularQuay/internal/platform"
)

const (
	envCQConfigSource = "CQ_CONFIG_SOURCE"
	envCQConfigRef    = "CQ_CONFIG_REF"
	envCQConfigPath   = "CQ_CONFIG_PATH"
	envCQConfigSHA256 = "CQ_CONFIG_SHA256"
)

type bootstrapFetch func(context.Context, bootstrap.Request) (bootstrap.Result, error)

type bootstrapOptions struct {
	sourceURL string
	ref       string
	path      string
	digest    string
	output    string
}

// bootstrapConfig materializes a verified immutable configuration. The CLI
// hook is deliberately separate so command routing can retain its own review.
func (a application) bootstrapConfig(ctx context.Context, args []string) (int, error) {
	fetcher := bootstrap.NewFetcher(nil)
	return a.bootstrapWithFetcher(ctx, args, fetcher.Fetch)
}

func (a application) bootstrapWithFetcher(ctx context.Context, args []string, fetch bootstrapFetch) (int, error) {
	options, err := a.bootstrapOptions(args)
	if err != nil {
		return 0, err
	}
	destination, err := filepath.Abs(options.output)
	if err != nil {
		return 0, fault.New(fault.Invalid, "bootstrap output")
	}
	if options.sourceURL == "" {
		return a.validateLocalBootstrap(ctx, destination)
	}
	if !pinnedBootstrapRequest(options.ref, options.digest) || fetch == nil {
		return 0, fault.New(fault.Invalid, "bootstrap pin")
	}
	if err := prepareBootstrapDestination(destination); err != nil {
		return 0, err
	}
	result, err := fetch(ctx, bootstrap.Request{SourceURL: options.sourceURL, ConfigRef: options.ref, ConfigPath: options.path, ExpectedSHA256: options.digest})
	if err != nil {
		return 0, err
	}
	if _, err = a.resolvePolicyBytes(ctx, result.Bytes, destination); err != nil {
		return 0, err
	}
	if err = platform.CreateProtectedExclusive(destination, result.Bytes); err != nil {
		return 0, err
	}
	return a.materializedBootstrapReceipt(destination, result)
}

func (a application) bootstrapOptions(args []string) (bootstrapOptions, error) {
	flags := flag.NewFlagSet("bootstrap", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	sourceURL := flags.String("config-source", os.Getenv(envCQConfigSource), "immutable GitHub configuration source")
	ref := flags.String("config-ref", os.Getenv(envCQConfigRef), "full immutable Git commit")
	path := flags.String("config-path", configPathDefault(), "configuration path in the repository")
	digest := flags.String("config-sha256", os.Getenv(envCQConfigSHA256), "required configuration SHA-256")
	output := flags.String("output", a.config, "new protected configuration output")
	_ = flags.Bool("non-interactive", false, "pinned retrieval never prompts")
	if flags.Parse(args) != nil || flags.NArg() != 0 {
		return bootstrapOptions{}, fault.New(fault.Invalid, "bootstrap arguments")
	}
	return bootstrapOptions{sourceURL: *sourceURL, ref: *ref, path: *path, digest: *digest, output: *output}, nil
}

func (a application) materializedBootstrapReceipt(destination string, result bootstrap.Result) (int, error) {
	return a.print(map[string]any{
		"state": "materialized", "source": result.CanonicalSourceURL, "ref": result.ConfigRef,
		"path": result.ConfigPath, "sha256": result.SHA256, "policy": destination,
	})
}

func configPathDefault() string {
	if path := os.Getenv(envCQConfigPath); path != "" {
		return path
	}
	return "cq.json"
}

func (a application) validateLocalBootstrap(ctx context.Context, destination string) (int, error) {
	policy, err := a.loadPolicy(ctx, destination)
	if err != nil {
		return 0, err
	}
	return a.print(map[string]any{"state": "validated", "source": "local", "policy": destination, "version": policy.Version})
}

func prepareBootstrapDestination(destination string) error {
	if err := platform.EnsurePrivateDirectory(filepath.Dir(destination)); err != nil {
		return err
	}
	if _, err := os.Lstat(destination); err == nil {
		return fault.New(fault.Trust, "bootstrap existing output")
	} else if !os.IsNotExist(err) {
		return fault.New(fault.Apply, "bootstrap output")
	}
	return nil
}

func pinnedBootstrapRequest(ref, digest string) bool {
	return lowerHex(ref, 40) && lowerHex(digest, 64)
}

func lowerHex(value string, length int) bool {
	if len(value) != length {
		return false
	}
	for _, character := range value {
		if !(character >= '0' && character <= '9' || character >= 'a' && character <= 'f') {
			return false
		}
	}
	return true
}
