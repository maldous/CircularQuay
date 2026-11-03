package policy

import (
	"net/url"
	"path/filepath"
	"strings"

	"github.com/maldous/CircularQuay/internal/model"
)

// configLockDirectory must match the CLI's private sibling coordination store.
const configLockDirectory = ".cquay-config-lock"

// validatePathOwnership prevents one policy-owned writer from replacing another
// local input or writer. It operates on canonical paths after Load binds paths.
func validatePathOwnership(policy model.Policy) error {
	return validatePathOwnershipWithPolicyFile(policy, "")
}

func validatePolicyFileOwnership(policy model.Policy, policyFile string) error {
	if err := validateReservedPolicyPaths(policy, policyFile); err != nil {
		return err
	}
	return validatePathOwnershipWithPolicyFile(policy, policyFile)
}

func validateReservedPolicyPaths(policy model.Policy, policyFile string) error {
	reserved := canonicalPath(filepath.Join(filepath.Dir(policyFile), configLockDirectory))
	return rejectReservedMetadataPaths(policyLocalPaths(policy), reserved)
}

func rejectReservedMetadataPaths(paths []string, reserved string) error {
	for _, path := range paths {
		if withinPath(reserved, canonicalPath(path)) {
			return invalid("reserved_metadata_path")
		}
	}
	return nil
}

func policyLocalPaths(policy model.Policy) []string {
	paths := []string{policy.Repository.Directory, policy.Repository.AllowedSignersFile, policy.Repository.TokenFile, policy.MCP.AuditPath}
	for _, source := range policy.Sources {
		paths = append(paths, source.Path, source.TokenFile, source.AgeIdentityFile, source.SOPS)
	}
	for _, workload := range policy.Workloads {
		paths = append(paths, workload.Directory, workload.Snapshot.Path, workload.Snapshot.IdentityFile, workload.Snapshot.SOPS)
		paths = append(paths, workload.Command...)
		for _, target := range workload.Targets {
			paths = append(paths, target.Path, target.TokenFile)
		}
	}
	return paths
}

func validatePathOwnershipWithPolicyFile(policy model.Policy, policyFile string) error {
	owners := pathOwners{inputs: make(map[string]bool), writers: make(map[string]bool), remote: make(map[string]bool), repository: canonicalPath(policy.Repository.Directory)}
	owners.addPolicyInputs(policy)
	owners.addInput(policyFile)
	if err := owners.addWriter(policy.MCP.AuditPath); err != nil {
		return err
	}
	for _, workload := range policy.Workloads {
		if err := owners.addWorkloadWriters(workload); err != nil {
			return err
		}
	}
	return nil
}

type pathOwners struct {
	inputs, writers, remote map[string]bool
	repository              string
}

func (owners *pathOwners) addPolicyInputs(policy model.Policy) {
	owners.addInput(policy.Repository.AllowedSignersFile)
	owners.addInput(policy.Repository.TokenFile)
	for _, source := range policy.Sources {
		owners.addSourceInputs(source)
	}
	for _, workload := range policy.Workloads {
		owners.addWorkloadInputs(workload)
	}
}

func (owners *pathOwners) addSourceInputs(source model.Source) {
	if localSourceInput(source) {
		owners.addInput(source.Path)
	}
	owners.addInput(source.TokenFile)
	owners.addInput(source.AgeIdentityFile)
	owners.addInput(source.SOPS)
	if source.Type == "bitwarden" || source.Type == "onepassword" {
		owners.addInput(source.Path)
	}
}

func localSourceInput(source model.Source) bool {
	return source.Type == "json" || source.Type == "sops" || source.Type == "env" && source.Path != ""
}

func (owners *pathOwners) addWorkloadInputs(workload model.Workload) {
	if len(workload.Command) > 0 {
		owners.addInput(workload.Command[0])
	}
	owners.addInput(workload.Snapshot.IdentityFile)
	owners.addInput(workload.Snapshot.SOPS)
	for _, target := range workload.Targets {
		owners.addInput(target.TokenFile)
	}
}

func (owners *pathOwners) addWorkloadWriters(workload model.Workload) error {
	if err := owners.addWriter(workload.Snapshot.Path); err != nil {
		return err
	}
	for _, target := range workload.Targets {
		if nativeTarget(target.Type) {
			if err := owners.addWriter(target.Path); err != nil {
				return err
			}
		} else if err := owners.addRemoteWriter(target); err != nil {
			return err
		}
	}
	return nil
}

func nativeTarget(targetType string) bool {
	return targetType == "file" || targetType == "systemd" || targetType == "windows"
}

func (owners *pathOwners) addInput(path string) {
	if canonical := canonicalPath(path); canonical != "" {
		owners.inputs[canonical] = true
	}
}

func (owners *pathOwners) addWriter(path string) error {
	canonical := canonicalPath(path)
	if canonical == "" {
		return nil
	}
	if owners.inputs[canonical] || owners.writers[canonical] || withinPath(owners.repository, canonical) {
		return invalid("path_ownership")
	}
	owners.writers[canonical] = true
	return nil
}

func (owners *pathOwners) addRemoteWriter(target model.Target) error {
	key := remoteOwnershipKey(target)
	if key == "" || owners.remote[key] {
		return invalid("target_ownership")
	}
	owners.remote[key] = true
	return nil
}

func remoteOwnershipKey(target model.Target) string {
	switch target.Type {
	case "cloudflare":
		return remoteKey(target.Type, endpointKey(target.Endpoint, "https://api.cloudflare.com"), target.Account, target.Resource, targetBinding(target))
	case "netlify":
		return remoteKey(target.Type, endpointKey(target.Endpoint, "https://api.netlify.com"), target.Account, target.Project, targetBinding(target))
	case "windmill":
		return remoteKey(target.Type, endpointKey(target.Endpoint, ""), target.Project, target.Resource)
	default:
		return ""
	}
}

func remoteKey(parts ...string) string { return strings.Join(parts, "\x00") }

func targetBinding(target model.Target) string {
	if target.Path != "" {
		return target.Path
	}
	return "CQUAY_CONFIG"
}

func endpointKey(endpoint, fallback string) string {
	if endpoint == "" {
		endpoint = fallback
	}
	parsed, err := url.Parse(endpoint)
	if err != nil {
		return ""
	}
	parsed.Scheme = strings.ToLower(parsed.Scheme)
	parsed.Host = strings.ToLower(parsed.Host)
	parsed.Path = strings.TrimRight(parsed.Path, "/")
	return parsed.String()
}

func canonicalPath(path string) string {
	if path == "" || !filepath.IsAbs(path) || strings.HasPrefix(path, "repo:") {
		return ""
	}
	return pathKey(path)
}

func withinPath(directory, path string) bool {
	return directory != "" && (path == directory || strings.HasPrefix(path, directory+string(filepath.Separator)))
}
