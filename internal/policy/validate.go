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
	if policy.Version != 1 || len(policy.Sources) == 0 || len(policy.Workloads) == 0 {
		return invalid("policy_structure")
	}
	if err := validateRepositoryReferences(policy); err != nil {
		return err
	}
	for name, source := range policy.Sources {
		if !identifier.MatchString(name) || validateSource(source) != nil {
			return invalid("source_declaration")
		}
	}
	if err := validateBootstrapSeparation(policy); err != nil {
		return err
	}
	for name, workload := range policy.Workloads {
		if !identifier.MatchString(name) || validateWorkload(policy, workload) != nil {
			return invalid("workload_declaration")
		}
	}
	if err := validateGrants(policy); err != nil {
		return err
	}
	return validateMCP(policy.MCP)
}

func validateSource(source model.Source) error {
	if !knownSource(source.Type) || !safeText(source.Path) || !safeText(source.TokenFile) || !safeText(source.AgeIdentityFile) || !safeText(source.SOPS) {
		return invalid("source_type")
	}
	if source.Endpoint != "" && !secureEndpoint(source.Endpoint, false) {
		return invalid("source_endpoint")
	}
	if requiresEndpoint(source.Type) && source.Endpoint == "" {
		return invalid("source_endpoint")
	}
	if localFileSource(source.Type) && source.Path == "" {
		return invalid("source_path")
	}
	if !localFileSource(source.Type) && len(source.Resources) == 0 {
		return invalid("source_resources")
	}
	if err := uniqueExact(source.Resources); err != nil {
		return err
	}
	return validateSourceIdentity(source)
}

func validateSourceIdentity(source model.Source) error {
	allowed := map[string]bool{"": true, "none": true, "token_file": true, "workload": true, "profile": true, "environment": true, "managed": true, "ec2": true, "ecs": true, "metadata": true}
	if !allowed[source.Identity] || source.Identity == "profile" && source.Profile == "" {
		return invalid("source_identity")
	}
	if source.Identity == "token_file" && source.TokenFile == "" {
		return invalid("source_identity")
	}
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
	if workload.Mode != "offline" && workload.Mode != "online" && workload.Mode != "hybrid" {
		return invalid("workload_mode")
	}
	if workload.Mode == "offline" && workload.AuditRequired {
		return invalid("offline_central_audit")
	}
	if len(workload.Bindings) == 0 || !safeText(workload.Directory) {
		return invalid("workload_bindings")
	}
	if workload.CommandSHA256 != "" && !validSHA256(workload.CommandSHA256) {
		return invalid("command_digest")
	}
	if err := ValidateEnvironment(workload.Environment); err != nil {
		return err
	}
	seen := make(map[string]bool)
	for key := range workload.Environment {
		seen[strings.ToUpper(key)] = true
	}
	for _, binding := range workload.Bindings {
		if err := validateBinding(policy, workload, binding, seen); err != nil {
			return err
		}
	}
	for _, arg := range workload.Command {
		if arg == "" || !safeText(arg) {
			return invalid("workload_command")
		}
	}
	return validateTargets(workload)
}

func validateBinding(policy model.Policy, workload model.Workload, binding model.Binding, seen map[string]bool) error {
	name := strings.ToUpper(binding.Name)
	if !environmentName.MatchString(binding.Name) || forbiddenBindingName(name) || seen[name] {
		return invalid("binding_name")
	}
	seen[name] = true
	source, exists := policy.Sources[binding.Source]
	if !exists || !resourceAllowed(source, binding.Resource) || !safeSelector(binding.Field) || !safeSelector(binding.Version) {
		return invalid("binding_scope")
	}
	if workload.AuditRequired && !source.AuditVerified {
		return invalid("audit_capability")
	}
	if workload.Mode == "offline" && (!binding.AllowOffline || !offlineSource(source.Type)) {
		return invalid("offline_scope")
	}
	if workload.Mode == "online" && (source.Type == "json" || source.Type == "env") {
		return invalid("online_source")
	}
	if workload.Mode == "online" && source.Type == "sops" && (source.AgeIdentityFile != "" || len(source.VaultAllowlist) == 0) {
		return invalid("online_decryption_authority")
	}
	return nil
}

func validateTargets(workload model.Workload) error {
	for name, target := range workload.Targets {
		if !identifier.MatchString(name) || !knownTarget(target.Type) || !safeText(target.Path) || !safeText(target.TokenFile) {
			return invalid("target_declaration")
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
		for _, binding := range workload.Bindings {
			if !binding.AllowPublish {
				return invalid("target_export_permission")
			}
		}
	}
	if workload.Mode == "online" && workload.Snapshot.Path != "" {
		return invalid("online_snapshot")
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
	if err != nil || parsed.Hostname() == "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" || strings.ContainsAny(raw, "\r\n\x00") {
		return false
	}
	if parsed.Scheme == "https" {
		return true
	}
	ip := net.ParseIP(parsed.Hostname())
	return allowLoopback && parsed.Scheme == "http" && (parsed.Hostname() == "localhost" || ip != nil && ip.IsLoopback())
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
