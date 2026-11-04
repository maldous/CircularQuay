package configuration

import (
	"path/filepath"
	"strings"
	"time"

	"github.com/maldous/CircularQuay/internal/fault"
	"github.com/maldous/CircularQuay/internal/model"
	"github.com/maldous/CircularQuay/internal/policy"
)

type includePlan struct {
	includes []model.Include
	byID     map[string]model.Include
	index    map[string]int
}

func validMaster(composition *model.Composition) bool {
	return composition != nil && composition.Conflicts == "reject" && len(composition.Includes) > 0 && len(composition.Includes) <= maximumIncludes
}

func newPlan(includes []model.Include) (includePlan, error) {
	result := includePlan{includes: includes, byID: make(map[string]model.Include, len(includes)), index: make(map[string]int, len(includes))}
	namespaces := make(map[string]struct{}, len(includes))
	for index, include := range includes {
		if !validInclude(include) {
			return includePlan{}, failure(fault.Invalid, "include")
		}
		if _, exists := result.byID[include.ID]; exists {
			return includePlan{}, failure(fault.Invalid, "include_id")
		}
		if _, exists := namespaces[include.Namespace]; exists {
			return includePlan{}, failure(fault.Invalid, "include_namespace")
		}
		result.byID[include.ID] = include
		result.index[include.ID] = index
		namespaces[include.Namespace] = struct{}{}
	}
	if err := result.validateDependencies(); err != nil {
		return includePlan{}, err
	}
	if !result.uniqueOrigins() {
		return includePlan{}, failure(fault.Invalid, "include_origin")
	}
	if !result.validPrivateBoundaries() {
		return includePlan{}, failure(fault.Trust, "include_boundary")
	}
	return result, nil
}

func validInclude(include model.Include) bool {
	return include.ID != "root" && validName(include.ID) && validNamespace(include.Namespace) && validDigest(include.SHA256) && validLocation(include) && validRefresh(include)
}

func validNamespace(value string) bool { return validName(value) && !strings.Contains(value, "__") }

func validName(value string) bool {
	if value == "" || len(value) > maximumNamespaceBytes {
		return false
	}
	for index, c := range value {
		if !(c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '_' || c == '-') || index == 0 && !(c >= 'a' && c <= 'z' || c >= '0' && c <= '9') {
			return false
		}
	}
	return true
}

func validLocation(include model.Include) bool {
	return isAbsoluteClean(include.BaseDirectory) && includeLocationValid(include)
}

func includeLocationValid(include model.Include) bool {
	if include.Repository == nil {
		return validLocalLocation(include)
	}
	return validRepositoryLocation(include)
}

func validLocalLocation(include model.Include) bool {
	return include.Visibility == "local" && include.File != "" && isAbsoluteClean(include.File) && include.Path == "" && include.Commit == ""
}

func validRepositoryLocation(include model.Include) bool {
	return include.Visibility == "shared" && include.File == "" && policy.ValidateRepository(*include.Repository) == nil && validRepositoryPath(include.Path) && validCommit(include.Commit)
}

func isAbsoluteClean(value string) bool {
	return value != "" && len(value) <= 4096 && !strings.ContainsAny(value, "\x00\r\n") && filepath.IsAbs(value) && filepath.Clean(value) == value
}

func validRefresh(include model.Include) bool {
	if include.Repository == nil {
		return include.Refresh == "" && include.CacheExpiresAt == nil
	}
	if include.Refresh == "online" {
		return include.CacheExpiresAt == nil
	}
	return include.Refresh == "offline" && include.CacheExpiresAt != nil && !include.CacheExpiresAt.IsZero()
}

func (plan includePlan) validateDependencies() error {
	for index, include := range plan.includes {
		seen := make(map[string]struct{}, len(include.DependsOn))
		for _, dependency := range include.DependsOn {
			if _, duplicate := seen[dependency]; duplicate || dependency == include.ID || plan.index[dependency] >= index {
				return failure(fault.Invalid, "include_dependency")
			}
			if _, found := plan.byID[dependency]; !found {
				return failure(fault.Invalid, "include_dependency")
			}
			seen[dependency] = struct{}{}
		}
		if plan.depth(include.ID) > maximumDepth {
			return failure(fault.Invalid, "include_depth")
		}
	}
	return nil
}

func (plan includePlan) depth(id string) int {
	include := plan.byID[id]
	depth := 1
	for _, dependency := range include.DependsOn {
		if candidate := plan.depth(dependency) + 1; candidate > depth {
			depth = candidate
		}
	}
	return depth
}

func (plan includePlan) cacheFresh(now func() time.Time) error {
	for _, include := range plan.includes {
		if include.Refresh == "offline" && !now().Before(*include.CacheExpiresAt) {
			return failure(fault.Expired, "include_cache")
		}
	}
	return nil
}

func (plan includePlan) permits(namespace, source string, include model.Include) bool {
	for _, dependency := range include.DependsOn {
		if plan.byID[dependency].Namespace == namespace && source != "" {
			return true
		}
	}
	return false
}

func (plan includePlan) validPrivateBoundaries() bool {
	for _, include := range plan.includes {
		if include.Visibility == "local" && plan.localBaseOverlapsOther(include) {
			return false
		}
	}
	return true
}

func (plan includePlan) localBaseOverlapsOther(local model.Include) bool {
	for _, other := range plan.includes {
		if other.ID == local.ID {
			continue
		}
		if pathsOverlap(local.BaseDirectory, other.BaseDirectory) || repositoryOverlaps(local.BaseDirectory, other.Repository) {
			return true
		}
	}
	return false
}

func (plan includePlan) uniqueOrigins() bool {
	for index, include := range plan.includes {
		for _, other := range plan.includes[:index] {
			if sameOrigin(include, other) {
				return false
			}
		}
	}
	return true
}

func sameOrigin(first, second model.Include) bool {
	if first.Repository == nil || second.Repository == nil {
		return first.Repository == nil && second.Repository == nil && samePath(first.File, second.File)
	}
	return sameRepositoryOrigin(*first.Repository, *second.Repository) && first.Commit == second.Commit && first.Path == second.Path
}

func sameRepositoryOrigin(first, second model.Repository) bool {
	return first.URL == second.URL && samePath(first.Directory, second.Directory) && first.Ref == second.Ref && samePath(first.AllowedSignersFile, second.AllowedSignersFile) && first.InitialCommit == second.InitialCommit && samePath(first.TokenFile, second.TokenFile)
}

func repositoryOverlaps(base string, repository *model.Repository) bool {
	return repository != nil && (pathsOverlap(base, repository.Directory) || pathsOverlap(base, repository.AllowedSignersFile) || pathsOverlap(base, repository.TokenFile))
}

func pathsOverlap(first, second string) bool {
	return first != "" && second != "" && (within(first, second) || within(second, first))
}

func validCommit(value string) bool {
	if len(value) != 40 {
		return false
	}
	for _, c := range value {
		if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f') {
			return false
		}
	}
	return true
}

func validRepositoryPath(value string) bool { return policy.ValidRepositoryPath(value) }
