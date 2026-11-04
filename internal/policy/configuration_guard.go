package policy

import "github.com/maldous/CircularQuay/internal/model"

// validateConfigurationGuard checks boundaries retained by the trusted resolver
// after root-owned edits. Checking only the input fragments would allow an edit
// to turn a private source into a corporate export or a shared decryption key.
func validateConfigurationGuard(policy model.Policy) error {
	guard := policy.ConfigurationGuard
	if guard == nil {
		return nil
	}
	if err := validateConfigurationDomains(policy, guard); err != nil {
		return err
	}
	if err := validateConfigurationCredentials(policy, guard); err != nil {
		return err
	}
	if err := validatePrivateGlobalReferences(policy, guard); err != nil {
		return err
	}
	for name, workload := range policy.Workloads {
		if err := validateComposedWorkload(policy, name, workload, guard); err != nil {
			return err
		}
	}
	return nil
}

func validateConfigurationDomains(policy model.Policy, guard *model.ConfigurationGuard) error {
	if len(guard.SourceDomains) != len(policy.Sources) || len(guard.LocalSources) != len(guard.LocalRoots) {
		return invalid("configuration_domains")
	}
	for name := range policy.Sources {
		if !identifier.MatchString(guard.SourceDomains[name]) {
			return invalid("configuration_domains")
		}
	}
	seen := make(map[string]bool)
	for _, name := range guard.LocalSources {
		if _, exists := policy.Sources[name]; !exists || seen[name] || canonicalPath(guard.LocalRoots[name]) == "" {
			return invalid("configuration_local_domain")
		}
		seen[name] = true
	}
	return nil
}

func validateConfigurationCredentials(policy model.Policy, guard *model.ConfigurationGuard) error {
	owners := make(map[string]string)
	for name, source := range policy.Sources {
		if err := validateSourceCredentialDomain(name, source, guard, owners); err != nil {
			return err
		}
	}
	return nil
}

func validateSourceCredentialDomain(name string, source model.Source, guard *model.ConfigurationGuard, owners map[string]string) error {
	if localSourceInput(source) && insideAnyRoot(guard.ReadOnlyRoots, canonicalPath(source.Path)) {
		return invalid("configuration_source_cache")
	}
	if err := validatePrivateReferences(name, source, guard); err != nil {
		return err
	}
	for _, reference := range []string{source.TokenFile, source.AgeIdentityFile} {
		if configurationOwnsReference(guard, reference) {
			return invalid("configuration_credential_cache")
		}
		if err := recordCredentialDomain(owners, reference, guard.SourceDomains[name]); err != nil {
			return err
		}
	}
	return nil
}

func recordCredentialDomain(owners map[string]string, reference, domain string) error {
	path := canonicalPath(reference)
	if path == "" {
		return nil
	}
	if previous, exists := owners[path]; exists && previous != domain {
		return invalid("configuration_shared_credential")
	}
	owners[path] = domain
	return nil
}

func validatePrivateReferences(name string, source model.Source, guard *model.ConfigurationGuard) error {
	paths := []string{source.TokenFile, source.AgeIdentityFile}
	if localSourceInput(source) {
		paths = append(paths, source.Path)
	}
	for _, reference := range paths {
		if err := validatePrivateReference(name, reference, guard); err != nil {
			return err
		}
	}
	// Master-selected binaries may live outside a data reference base, but a
	// different department cannot execute a binary from a private domain.
	executables := []string{source.SOPS}
	if source.Type == "bitwarden" || source.Type == "onepassword" {
		executables = append(executables, source.Path)
	}
	return validateDomainReferences(executables, map[string]bool{guard.SourceDomains[name]: true}, guard)
}

func validatePrivateReference(name, reference string, guard *model.ConfigurationGuard) error {
	if reference == "" {
		return nil
	}
	path := canonicalPath(reference)
	if root, private := guard.LocalRoots[name]; private && !withinPath(canonicalPath(root), path) {
		return invalid("configuration_private_reference")
	}
	for owner, root := range guard.LocalRoots {
		if withinPath(canonicalPath(root), path) && guard.SourceDomains[name] != guard.SourceDomains[owner] {
			return invalid("configuration_private_reference")
		}
	}
	return nil
}

func validateComposedWorkload(policy model.Policy, name string, workload model.Workload, guard *model.ConfigurationGuard) error {
	domains := make(map[string]bool)
	privateRoot := ""
	for _, binding := range workload.Bindings {
		domains[guard.SourceDomains[binding.Source]] = true
		if root, private := guard.LocalRoots[binding.Source]; private {
			if binding.AllowPublish {
				return invalid("configuration_private_publish")
			}
			privateRoot = canonicalPath(root)
		}
	}
	if workload.Snapshot.Path != "" && len(domains) > 1 {
		return invalid("configuration_mixed_snapshot")
	}
	if err := validateWorkloadDomainReferences(workload, domains, guard); err != nil {
		return err
	}
	if privateRoot != "" {
		return validatePrivateWorkload(policy, name, workload, privateRoot)
	}
	return nil
}

func validateWorkloadDomainReferences(workload model.Workload, domains map[string]bool, guard *model.ConfigurationGuard) error {
	paths := []string{workload.Directory, workload.Snapshot.Path, workload.Snapshot.IdentityFile, workload.Snapshot.SOPS}
	if len(workload.Command) != 0 {
		paths = append(paths, workload.Command[0])
	}
	for _, target := range workload.Targets {
		paths = append(paths, target.TokenFile)
		if nativeTarget(target.Type) {
			paths = append(paths, target.Path)
		}
	}
	return validateDomainReferences(paths, domains, guard)
}

func validatePrivateGlobalReferences(policy model.Policy, guard *model.ConfigurationGuard) error {
	paths := []string{policy.Repository.Directory, policy.Repository.TokenFile, policy.Repository.AllowedSignersFile, policy.MCP.AuditPath}
	paths = append(paths, operationalProtectedReferences(policy)...)
	paths = append(paths, trustStorePaths(policy)...)
	return validateDomainReferences(paths, nil, guard)
}

func validateDomainReferences(paths []string, domains map[string]bool, guard *model.ConfigurationGuard) error {
	for _, path := range paths {
		if err := validateDomainReference(path, domains, guard); err != nil {
			return err
		}
	}
	return nil
}

func validateDomainReference(reference string, domains map[string]bool, guard *model.ConfigurationGuard) error {
	if reference == "" {
		return nil
	}
	for owner, root := range guard.LocalRoots {
		if withinPath(canonicalPath(root), canonicalPath(reference)) && (len(domains) != 1 || !domains[guard.SourceDomains[owner]]) {
			return invalid("configuration_private_reference")
		}
	}
	return nil
}

func validatePrivateWorkload(policy model.Policy, name string, workload model.Workload, root string) error {
	if len(workload.Targets) != 0 {
		return invalid("configuration_private_target")
	}
	for _, grant := range policy.Grants {
		if grant.Workload == name && grant.Authority != "" {
			return invalid("configuration_private_remote_grant")
		}
	}
	if workload.Snapshot.Path == "" {
		return nil
	}
	if !withinPath(root, canonicalPath(workload.Snapshot.Path)) || !withinPath(root, canonicalPath(workload.Snapshot.IdentityFile)) {
		return invalid("configuration_private_snapshot")
	}
	return nil
}

func configurationOwnsReference(guard *model.ConfigurationGuard, reference string) bool {
	if guard == nil || reference == "" {
		return false
	}
	path := canonicalPath(reference)
	for _, input := range guard.Inputs {
		if path == canonicalPath(input) {
			return true
		}
	}
	return insideAnyRoot(guard.ReadOnlyRoots, path)
}

func insideAnyRoot(roots []string, path string) bool {
	for _, root := range roots {
		if withinPath(canonicalPath(root), path) {
			return true
		}
	}
	return false
}
