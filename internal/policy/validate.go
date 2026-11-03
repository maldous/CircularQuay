package policy

import (
	"net"
	"net/url"
	"path/filepath"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/maldous/CircularQuay/internal/fault"
	"github.com/maldous/CircularQuay/internal/model"
)

var identifier = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{0,63}$`)
var environmentName = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// Validate is a pure check: it never authenticates, retrieves values or issues credentials.
func Validate(policy model.Policy) error {
	if !validPolicyStructure(policy) {
		return invalid("policy_structure")
	}
	if err := validateRepositoryReferences(policy); err != nil {
		return err
	}
	if err := validatePolicySources(policy.Sources); err != nil {
		return err
	}
	if err := validateBootstrapSeparation(policy); err != nil {
		return err
	}
	if err := validatePolicyWorkloads(policy); err != nil {
		return err
	}
	if err := validatePathOwnership(policy); err != nil {
		return err
	}
	if err := validateGrants(policy); err != nil {
		return err
	}
	return validateMCP(policy.MCP)
}

func validPolicyStructure(policy model.Policy) bool {
	return policy.Version == 1 && len(policy.Sources) > 0 && len(policy.Workloads) > 0
}

func validatePolicySources(sources map[string]model.Source) error {
	for name, source := range sources {
		if !identifier.MatchString(name) || validateSource(source) != nil {
			return invalid("source_declaration")
		}
	}
	return nil
}

func validatePolicyWorkloads(policy model.Policy) error {
	for name, workload := range policy.Workloads {
		if !identifier.MatchString(name) || validateWorkload(policy, workload) != nil {
			return invalid("workload_declaration")
		}
	}
	return nil
}

func validateSource(source model.Source) error {
	if !validSourceText(source) {
		return invalid("source_type")
	}
	if !validSourceEndpoint(source) {
		return invalid("source_endpoint")
	}
	if err := validateSourceLocation(source); err != nil {
		return err
	}
	if err := uniqueExact(source.Resources); err != nil {
		return err
	}
	if err := validateEnvironmentSourceResources(source); err != nil {
		return err
	}
	if err := validateSourceContract(source); err != nil {
		return err
	}
	return validateSourceIdentity(source)
}

func validateEnvironmentSourceResources(source model.Source) error {
	if source.Type != "env" {
		return nil
	}
	for _, resource := range source.Resources {
		if IsProtectedEnvironmentName(resource) {
			return invalid("source_environment_input")
		}
	}
	return nil
}

func validSourceText(source model.Source) bool {
	return knownSource(source.Type) && safeText(source.Path) && safeText(source.TokenFile) && safeText(source.AgeIdentityFile) && safeText(source.SOPS)
}

func validSourceEndpoint(source model.Source) bool {
	if source.Endpoint != "" && !secureEndpoint(source.Endpoint, false) {
		return false
	}
	return !requiresEndpoint(source.Type) || source.Endpoint != ""
}

func validateSourceLocation(source model.Source) error {
	if localFileSource(source.Type) && source.Path == "" {
		return invalid("source_path")
	}
	if !localFileSource(source.Type) && len(source.Resources) == 0 {
		return invalid("source_resources")
	}
	return nil
}

func validateSourceIdentity(source model.Source) error {
	if !validSourceIdentity(source) {
		return invalid("source_identity")
	}
	if source.Identity == "token_file" && source.TokenFile == "" {
		return invalid("source_identity")
	}
	if err := validateVaultAuthority(source); err != nil {
		return err
	}
	return nil
}

func validSourceIdentity(source model.Source) bool {
	allowed := map[string]bool{"": true, "none": true, "token_file": true, "workload": true, "profile": true, "environment": true, "managed": true, "ec2": true, "ecs": true, "metadata": true}
	return allowed[source.Identity] && (source.Identity != "profile" || source.Profile != "")
}

func validateVaultAuthority(source model.Source) error {
	if source.Type != "sops" && len(source.VaultAllowlist) > 0 {
		return invalid("vault_allowlist")
	}
	for _, endpoint := range source.VaultAllowlist {
		if !secureEndpoint(endpoint, false) {
			return invalid("vault_allowlist")
		}
	}
	if len(source.VaultAllowlist) > 0 && len(source.VaultKeys) == 0 {
		return invalid("vault_key_scope")
	}
	if err := uniqueExact(source.VaultKeys); err != nil {
		return err
	}
	if len(source.VaultAllowlist) > 0 && source.AgeIdentityFile != "" {
		return invalid("mixed_decryption_authority")
	}
	return nil
}

func validateWorkload(policy model.Policy, workload model.Workload) error {
	if !validWorkloadMode(workload) {
		return invalid("workload_mode")
	}
	if workload.Mode == "offline" && workload.AuditRequired {
		return invalid("offline_central_audit")
	}
	if !validWorkloadBindings(workload) {
		return invalid("workload_bindings")
	}
	if !validWorkloadDigest(workload) {
		return invalid("command_digest")
	}
	if err := ValidateEnvironment(workload.Environment); err != nil {
		return err
	}
	if err := validateWorkloadBindings(policy, workload); err != nil {
		return err
	}
	if err := validateSnapshotDeclaration(workload); err != nil {
		return err
	}
	if err := validateCommand(workload.Command); err != nil {
		return err
	}
	return validateTargets(workload)
}

func validWorkloadDigest(workload model.Workload) bool {
	return workload.CommandSHA256 == "" || validSHA256(workload.CommandSHA256)
}

func validateCommand(command []string) error {
	for _, arg := range command {
		if arg == "" || !safeText(arg) {
			return invalid("workload_command")
		}
	}
	return nil
}

func validWorkloadMode(workload model.Workload) bool {
	return workload.Mode == "offline" || workload.Mode == "online" || workload.Mode == "hybrid"
}

func validWorkloadBindings(workload model.Workload) bool {
	return len(workload.Bindings) > 0 && safeText(workload.Directory)
}

func validateWorkloadBindings(policy model.Policy, workload model.Workload) error {
	seen := environmentNames(workload.Environment)
	for _, binding := range workload.Bindings {
		if err := validateBinding(policy, workload, binding, seen); err != nil {
			return err
		}
	}
	return nil
}

func environmentNames(environment map[string]string) map[string]bool {
	seen := make(map[string]bool, len(environment))
	for key := range environment {
		seen[strings.ToUpper(key)] = true
	}
	return seen
}

func validateBinding(policy model.Policy, workload model.Workload, binding model.Binding, seen map[string]bool) error {
	name := strings.ToUpper(binding.Name)
	if !validBindingName(binding.Name, name, seen) {
		return invalid("binding_name")
	}
	seen[name] = true
	source, exists := policy.Sources[binding.Source]
	if !validBindingScope(source, exists, binding) {
		return invalid("binding_scope")
	}
	if binding.Version != "" && !model.SourceSupportsVersion(source.Type) {
		return invalid("binding_version")
	}
	if workload.AuditRequired && !source.AuditVerified {
		return invalid("audit_capability")
	}
	if err := validateBindingMode(workload, source, binding); err != nil {
		return err
	}
	return nil
}

func validateBindingMode(workload model.Workload, source model.Source, binding model.Binding) error {
	if workload.Mode == "offline" && (!binding.AllowOffline || !offlineSource(source.Type)) {
		return invalid("offline_scope")
	}
	if workload.Mode != "online" {
		return nil
	}
	if source.Type == "json" || source.Type == "env" {
		return invalid("online_source")
	}
	if source.Type == "sops" && (source.AgeIdentityFile != "" || len(source.VaultAllowlist) == 0) {
		return invalid("online_decryption_authority")
	}
	return nil
}

func validBindingName(name, upperName string, seen map[string]bool) bool {
	return environmentName.MatchString(name) && !IsProtectedEnvironmentName(upperName) && !seen[upperName]
}

func validBindingScope(source model.Source, exists bool, binding model.Binding) bool {
	return exists && resourceAllowed(source, binding.Resource) && safeSelector(binding.Field) && safeSelector(binding.Version)
}

func validateTargets(workload model.Workload) error {
	for name, target := range workload.Targets {
		if err := validateTargetDeclaration(name, target, workload.Bindings); err != nil {
			return err
		}
	}
	if workload.Mode == "online" && workload.Snapshot.Path != "" {
		return invalid("online_snapshot")
	}
	return nil
}

func validateTargetDeclaration(name string, target model.Target, bindings []model.Binding) error {
	if !validTarget(name, target) {
		return invalid("target_declaration")
	}
	if err := validateTargetContract(target); err != nil {
		return err
	}
	if target.Endpoint != "" && !secureEndpoint(target.Endpoint, false) {
		return invalid("target_endpoint")
	}
	if err := validateActivation(target); err != nil {
		return err
	}
	if !target.AcceptSnapshot {
		return invalid("target_snapshot_semantics")
	}
	return validateTargetExports(bindings)
}

func validTarget(name string, target model.Target) bool {
	return identifier.MatchString(name) && knownTarget(target.Type) && safeText(target.Path) && safeText(target.TokenFile)
}

func validateTargetExports(bindings []model.Binding) error {
	for _, binding := range bindings {
		if !binding.AllowPublish {
			return invalid("target_export_permission")
		}
	}
	return nil
}

func resourceAllowed(source model.Source, resource string) bool {
	if resource == "" {
		return localFileSource(source.Type)
	}
	for _, permitted := range source.Resources {
		if resource == permitted {
			return true
		}
	}
	return false
}

func knownSource(kind string) bool {
	switch kind {
	case "sops", "json", "env", "openbao", "vault", "aws-sm", "aws-ssm", "azure", "gcp", "doppler", "infisical", "bitwarden", "akeyless", "onepassword":
		return true
	default:
		return false
	}
}

func localFileSource(kind string) bool { return kind == "sops" || kind == "json" }

func offlineSource(kind string) bool { return localFileSource(kind) || kind == "env" }

func knownTarget(kind string) bool {
	switch kind {
	case "file", "systemd", "windows", "cloudflare", "netlify", "windmill":
		return true
	default:
		return false
	}
}

func requiresEndpoint(kind string) bool {
	return kind == "openbao" || kind == "vault" || kind == "azure"
}

func secureEndpoint(raw string, allowLoopback bool) bool {
	parsed, err := url.Parse(raw)
	if err != nil || !validEndpointURL(raw, parsed) {
		return false
	}
	if parsed.Scheme == "https" {
		return true
	}
	return allowLoopback && parsed.Scheme == "http" && loopbackHost(parsed.Hostname())
}

func validEndpointURL(raw string, parsed *url.URL) bool {
	return parsed.Hostname() != "" && parsed.User == nil && parsed.RawQuery == "" && parsed.Fragment == "" && !strings.ContainsAny(raw, "\r\n\x00")
}

func loopbackHost(host string) bool {
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func safeText(value string) bool {
	return utf8.ValidString(value) && !strings.ContainsAny(value, "\x00\r\n")
}

func safeSelector(value string) bool {
	if !safeText(value) || strings.ContainsAny(value, "*?[]\\") || len(value) > 4096 {
		return false
	}
	for _, segment := range strings.Split(value, "/") {
		if segment == "." || segment == ".." {
			return false
		}
	}
	return true
}

func uniqueExact(values []string) error {
	seen := make(map[string]bool)
	for _, value := range values {
		if value == "" || !safeSelector(value) || seen[value] {
			return invalid("resource_allowlist")
		}
		seen[value] = true
	}
	return nil
}

func invalid(operation string) error { return fault.New(fault.Invalid, operation) }

func validSHA256(value string) bool {
	if len(value) != 64 {
		return false
	}
	for _, digit := range value {
		if !strings.ContainsRune("0123456789abcdef", digit) {
			return false
		}
	}
	return true
}

func validateBootstrapSeparation(policy model.Policy) error {
	paths := make(map[string]bool)
	rememberPath(paths, policy.Repository.TokenFile)
	rememberPath(paths, policy.Repository.AllowedSignersFile)
	for _, source := range policy.Sources {
		rememberPath(paths, source.TokenFile)
		rememberPath(paths, source.AgeIdentityFile)
	}
	for _, workload := range policy.Workloads {
		rememberPath(paths, workload.Snapshot.IdentityFile)
		for _, target := range workload.Targets {
			rememberPath(paths, target.TokenFile)
		}
	}
	for _, source := range policy.Sources {
		if source.Path != "" && paths[pathKey(source.Path)] {
			return invalid("bootstrap_source_collision")
		}
	}
	return nil
}

func rememberPath(paths map[string]bool, path string) {
	if path != "" {
		paths[pathKey(path)] = true
	}
}

func pathKey(path string) string { return strings.ToLower(filepath.Clean(path)) }
