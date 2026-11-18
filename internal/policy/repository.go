package policy

import (
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/maldous/CircularQuay/internal/model"
)

var commitID = regexp.MustCompile(`^[0-9a-f]{40}$`)
var refName = regexp.MustCompile(`^refs/(?:heads|tags)/[A-Za-z0-9][A-Za-z0-9._/-]*$`)
var serviceName = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.@-]{0,127}$`)

// ValidateRepository checks the trust declaration without network or filesystem I/O.
func ValidateRepository(repo model.Repository) error {
	if repo.URL == "" {
		return validateEmptyRepository(repo)
	}
	if !validRepositoryTrust(repo) {
		return invalid("repository_trust")
	}
	if !validRepositoryPaths(repo) {
		return invalid("repository_paths")
	}
	if !validRepositoryToken(repo.TokenFile) {
		return invalid("repository_token")
	}
	if repositoryBootstrapCycle(repo) {
		return invalid("repository_bootstrap_cycle")
	}
	return nil
}

func validateEmptyRepository(repo model.Repository) error {
	if repo.Directory != "" || repo.Ref != "" || repo.AllowedSignersFile != "" || repo.InitialCommit != "" || repo.TokenFile != "" {
		return invalid("repository_declaration")
	}
	return nil
}

func validRepositoryTrust(repo model.Repository) bool {
	return secureEndpoint(repo.URL, false) && commitID.MatchString(repo.InitialCommit) && validRef(repo.Ref)
}

func validRepositoryPaths(repo model.Repository) bool {
	return filepath.IsAbs(repo.Directory) && filepath.IsAbs(repo.AllowedSignersFile) && safeText(repo.Directory) && safeText(repo.AllowedSignersFile)
}

func validRepositoryToken(tokenFile string) bool {
	return tokenFile == "" || filepath.IsAbs(tokenFile) && safeText(tokenFile)
}

func repositoryBootstrapCycle(repo model.Repository) bool {
	return within(repo.Directory, repo.AllowedSignersFile) || repo.TokenFile != "" && within(repo.Directory, repo.TokenFile)
}

func validRef(value string) bool {
	if !refName.MatchString(value) || strings.Contains(value, "..") || strings.Contains(value, "//") || strings.HasSuffix(value, "/") || strings.HasSuffix(value, ".") {
		return false
	}
	for _, segment := range strings.Split(value, "/") {
		if strings.HasPrefix(segment, ".") || strings.HasSuffix(segment, ".lock") {
			return false
		}
	}
	return true
}

func within(directory, file string) bool {
	parent := pathKey(directory)
	child := pathKey(file)
	return child == parent || strings.HasPrefix(child, parent+string(filepath.Separator))
}

// ValidRepositoryPath permits a portable data path, never traversal or Git policy.
func ValidRepositoryPath(value string) bool {
	if value == "" || value != path.Clean(value) || strings.HasPrefix(value, "/") || !safeSelector(value) || strings.Contains(value, ":") {
		return false
	}
	for _, segment := range strings.Split(value, "/") {
		if !portableSegment(segment) {
			return false
		}
	}
	return true
}

func portableSegment(segment string) bool {
	if invalidPortableSuffix(segment) {
		return false
	}
	upper := strings.ToUpper(segment)
	if gitControlFile(upper) {
		return false
	}
	return !windowsReservedName(upper)
}

func invalidPortableSuffix(segment string) bool {
	return segment == "" || segment == "." || segment == ".." || strings.HasSuffix(segment, ".") || strings.HasSuffix(segment, " ") || strings.ContainsAny(segment, `<>|"`)
}

func gitControlFile(upper string) bool {
	return upper == ".GIT" || upper == ".GITATTRIBUTES" || upper == ".GITMODULES"
}

func windowsReservedName(upper string) bool {
	base := strings.SplitN(upper, ".", 2)[0]
	reserved := map[string]bool{"CON": true, "PRN": true, "AUX": true, "NUL": true, "COM1": true, "COM2": true, "COM3": true, "COM4": true, "COM5": true, "COM6": true, "COM7": true, "COM8": true, "COM9": true, "LPT1": true, "LPT2": true, "LPT3": true, "LPT4": true, "LPT5": true, "LPT6": true, "LPT7": true, "LPT8": true, "LPT9": true}
	return reserved[base]
}

func validateRepositoryReferences(config model.Policy) error {
	if err := ValidateRepository(config.Repository); err != nil {
		return err
	}
	for _, source := range config.Sources {
		if err := validateRepositorySource(config.Repository, source); err != nil {
			return err
		}
	}
	for _, workload := range config.Workloads {
		if err := validateLocalWorkloadPaths(workload); err != nil {
			return err
		}
	}
	return nil
}

func validateRepositorySource(repository model.Repository, source model.Source) error {
	if strings.HasPrefix(source.Path, "repo:") && (repository.URL == "" || !localFileSource(source.Type) || !ValidRepositoryPath(strings.TrimPrefix(source.Path, "repo:"))) {
		return invalid("repository_source")
	}
	for _, bootstrap := range []string{source.TokenFile, source.AgeIdentityFile, source.SOPS} {
		if strings.HasPrefix(bootstrap, "repo:") {
			return invalid("repository_bootstrap")
		}
	}
	return nil
}

func validateLocalWorkloadPaths(workload model.Workload) error {
	for _, value := range []string{workload.Directory, workload.Snapshot.Path, workload.Snapshot.IdentityFile, workload.Snapshot.SOPS} {
		if strings.HasPrefix(value, "repo:") {
			return invalid("repository_execution_or_cache")
		}
	}
	if len(workload.Command) > 0 && strings.HasPrefix(workload.Command[0], "repo:") {
		return invalid("repository_execution")
	}
	for _, target := range workload.Targets {
		if strings.HasPrefix(target.TokenFile, "repo:") || strings.HasPrefix(target.Path, "repo:") {
			return invalid("repository_destination")
		}
	}
	return nil
}

func validateActivation(target model.Target) error {
	if !knownActivation(target.Activation) {
		return invalid("activation_action")
	}
	if target.Service != "" && !serviceName.MatchString(target.Service) {
		return invalid("activation_service")
	}
	if activationNeedsService(target) && (target.Service == "" || !nativeActivationTarget(target.Type)) {
		return invalid("activation_scope")
	}
	return nil
}

func knownActivation(action string) bool {
	return action == "" || action == "none" || action == "restart" || action == "reload"
}

func activationNeedsService(target model.Target) bool {
	return target.Activation != "" && target.Activation != "none"
}

func nativeActivationTarget(targetType string) bool {
	return targetType == "systemd" || targetType == "windows"
}
