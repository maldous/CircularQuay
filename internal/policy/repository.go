package policy

import (
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/maldous/CircularQuay/internal/model"
)

var commitID = regexp.MustCompile(`^(?:[0-9a-f]{40}|[0-9a-f]{64})$`)
var refName = regexp.MustCompile(`^refs/(?:heads|tags)/[A-Za-z0-9][A-Za-z0-9._/-]*$`)
var serviceName = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.@-]{0,127}$`)

// ValidateRepository checks the trust declaration without network or filesystem I/O.
func ValidateRepository(repo model.Repository) error {
	if repo.URL == "" {
		if repo.Directory != "" || repo.Ref != "" || repo.AllowedSignersFile != "" || repo.InitialCommit != "" || repo.TokenFile != "" {
			return invalid("repository_declaration")
		}
		return nil
	}
	if !secureEndpoint(repo.URL, false) || !commitID.MatchString(repo.InitialCommit) || !validRef(repo.Ref) {
		return invalid("repository_trust")
	}
	if !filepath.IsAbs(repo.Directory) || !filepath.IsAbs(repo.AllowedSignersFile) || !safeText(repo.Directory) || !safeText(repo.AllowedSignersFile) {
		return invalid("repository_paths")
	}
	if repo.TokenFile != "" && (!filepath.IsAbs(repo.TokenFile) || !safeText(repo.TokenFile)) {
		return invalid("repository_token")
	}
	if within(repo.Directory, repo.AllowedSignersFile) || repo.TokenFile != "" && within(repo.Directory, repo.TokenFile) {
		return invalid("repository_bootstrap_cycle")
	}
	return nil
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
	if segment == "" || strings.HasSuffix(segment, ".") || strings.HasSuffix(segment, " ") || strings.ContainsAny(segment, `<>|"`) {
		return false
	}
	upper := strings.ToUpper(segment)
	if upper == ".GIT" || upper == ".GITATTRIBUTES" || upper == ".GITMODULES" {
		return false
	}
	base := strings.SplitN(upper, ".", 2)[0]
	switch base {
	case "CON", "PRN", "AUX", "NUL", "COM1", "COM2", "COM3", "COM4", "COM5", "COM6", "COM7", "COM8", "COM9", "LPT1", "LPT2", "LPT3", "LPT4", "LPT5", "LPT6", "LPT7", "LPT8", "LPT9":
		return false
	default:
		return true
	}
}

func validateRepositoryReferences(config model.Policy) error {
	if err := ValidateRepository(config.Repository); err != nil {
		return err
	}
	for _, source := range config.Sources {
		if strings.HasPrefix(source.Path, "repo:") && (config.Repository.URL == "" || !localFileSource(source.Type) || !ValidRepositoryPath(strings.TrimPrefix(source.Path, "repo:"))) {
			return invalid("repository_source")
		}
		for _, bootstrap := range []string{source.TokenFile, source.AgeIdentityFile, source.SOPS} {
			if strings.HasPrefix(bootstrap, "repo:") {
				return invalid("repository_bootstrap")
			}
		}
	}
	for _, workload := range config.Workloads {
		if err := validateLocalWorkloadPaths(workload); err != nil {
			return err
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
	if target.Activation != "" && target.Activation != "none" && target.Activation != "restart" && target.Activation != "reload" {
		return invalid("activation_action")
	}
	if target.Service != "" && !serviceName.MatchString(target.Service) {
		return invalid("activation_service")
	}
	if target.Activation != "" && target.Activation != "none" && (target.Service == "" || target.Type != "systemd" && target.Type != "windows") {
		return invalid("activation_scope")
	}
	return nil
}
