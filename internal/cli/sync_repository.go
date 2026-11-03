package cli

import (
	"context"
	"path/filepath"
	"strings"

	"github.com/maldous/CircularQuay/internal/fault"
	"github.com/maldous/CircularQuay/internal/model"
	"github.com/maldous/CircularQuay/internal/policy"
	"github.com/maldous/CircularQuay/internal/repository"
	"github.com/maldous/CircularQuay/internal/resolve"
)

const repositorySourceOperation = "repository source"

type repositoryOperations struct {
	fetch   func(context.Context, model.Repository) (string, error)
	current func(model.Repository) (string, error)
}

func nativeRepositoryOperations() repositoryOperations {
	return repositoryOperations{fetch: repository.Fetch, current: repository.Current}
}

func (a application) withRepositorySources(ctx context.Context, workload model.Workload, engine *resolve.Engine) (*resolve.Engine, error) {
	sources, err := repositorySources(engine.Policy, workload)
	if err != nil || len(sources) == 0 {
		return engine, err
	}
	operations := a.repositoryOperations()
	release, err := repositoryRelease(ctx, workload.Mode, engine.Policy.Repository, sources, operations)
	if err != nil {
		return nil, err
	}
	updated, err := expandRepositorySources(engine.Policy, sources, release)
	if err != nil {
		return nil, err
	}
	return &resolve.Engine{
		Policy: updated, Reader: engine.Reader, Snapshots: engine.Snapshots,
		State: engine.State, Now: engine.Now,
	}, nil
}

func (a application) repositoryOperations() repositoryOperations {
	operations := a.repository
	if operations.fetch == nil || operations.current == nil {
		return nativeRepositoryOperations()
	}
	return operations
}

type repositorySource struct {
	source       model.Source
	allowOffline bool
}

func repositorySources(config model.Policy, workload model.Workload) (map[string]repositorySource, error) {
	result := make(map[string]repositorySource)
	for _, binding := range workload.Bindings {
		source, found := config.Sources[binding.Source]
		if !found {
			return nil, fault.New(fault.Invalid, repositorySourceOperation)
		}
		if !strings.HasPrefix(source.Path, "repo:") {
			continue
		}
		if !policy.ValidRepositoryPath(strings.TrimPrefix(source.Path, "repo:")) {
			return nil, fault.New(fault.Trust, repositorySourceOperation)
		}
		selected, exists := result[binding.Source]
		if !exists {
			selected = repositorySource{source: source, allowOffline: true}
		}
		selected.allowOffline = selected.allowOffline && binding.AllowOffline
		result[binding.Source] = selected
	}
	return result, nil
}

func repositoryRelease(ctx context.Context, mode string, repo model.Repository, sources map[string]repositorySource, operations repositoryOperations) (string, error) {
	if mode == "offline" {
		return operations.current(repo)
	}
	if _, err := operations.fetch(ctx, repo); err == nil {
		return operations.current(repo)
	} else if mode != "hybrid" || !fault.Is(err, fault.Unavailable) || !cachedRepositoryValuesAllowed(sources) {
		return "", err
	}
	return operations.current(repo)
}

// cachedRepositoryValuesAllowed limits a hybrid transport fallback to bindings
// that explicitly permit offline use. Transit-backed SOPS ciphertext is not a
// value cache: its configured online decryption authority still runs later.
func cachedRepositoryValuesAllowed(sources map[string]repositorySource) bool {
	for _, selected := range sources {
		if localRepositoryValue(selected.source) && !selected.allowOffline {
			return false
		}
	}
	return true
}

func localRepositoryValue(source model.Source) bool {
	return source.Type == "json" || source.Type == "sops" && source.AgeIdentityFile != ""
}

func expandRepositorySources(config model.Policy, sources map[string]repositorySource, release string) (model.Policy, error) {
	updated := config
	updated.Sources = make(map[string]model.Source, len(config.Sources))
	for name, source := range config.Sources {
		updated.Sources[name] = source
	}
	for name, selected := range sources {
		source := selected.source
		path, err := repositorySourcePath(release, source.Path)
		if err != nil {
			return model.Policy{}, err
		}
		source.Path = path
		updated.Sources[name] = source
	}
	return updated, nil
}

func repositorySourcePath(release, declared string) (string, error) {
	path := strings.TrimPrefix(declared, "repo:")
	if !policy.ValidRepositoryPath(path) || !filepath.IsAbs(release) {
		return "", fault.New(fault.Trust, repositorySourceOperation)
	}
	candidate := filepath.Join(release, filepath.FromSlash(path))
	relative, err := filepath.Rel(release, candidate)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return "", fault.New(fault.Trust, repositorySourceOperation)
	}
	return candidate, nil
}
