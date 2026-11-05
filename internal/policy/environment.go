package policy

import "strings"

// ValidateEnvironment checks explicitly declared non-secret execution basics.
// Source-delivered values undergo the stricter forbiddenBindingName policy.
func ValidateEnvironment(values map[string]string) error {
	seen := make(map[string]bool)
	for name, value := range values {
		upper := strings.ToUpper(name)
		if !environmentName.MatchString(name) || seen[upper] || strings.IndexByte(value, 0) >= 0 {
			return invalid("environment_declaration")
		}
		if bootstrapName(upper) {
			return invalid("bootstrap_environment")
		}
		seen[upper] = true
	}
	return nil
}

// IsProtectedEnvironmentName reports whether name controls process bootstrap
// or execution behavior and therefore cannot be supplied by a secret source.
func IsProtectedEnvironmentName(name string) bool {
	return forbiddenBindingName(strings.ToUpper(name))
}

func forbiddenBindingName(name string) bool {
	if bootstrapName(name) || strings.HasPrefix(name, "LD_") || strings.HasPrefix(name, "DYLD_") {
		return true
	}
	switch name {
	case "PATH", "HOME", "USERPROFILE", "SYSTEMROOT", "COMSPEC", "PATHEXT", "SHELL", "BASH_ENV", "ENV", "IFS", "NODE_OPTIONS", "PYTHONPATH", "PYTHONHOME", "RUBYOPT", "PERL5OPT", "GOFLAGS", "GODEBUG", "GIT_CONFIG", "GIT_SSH", "GIT_SSH_COMMAND", "HTTP_PROXY", "HTTPS_PROXY", "ALL_PROXY", "SSL_CERT_FILE", "SSL_CERT_DIR", "NODE_EXTRA_CA_CERTS":
		return true
	default:
		return false
	}
}

func bootstrapName(name string) bool {
	return strings.HasPrefix(name, "SOPS_") || strings.HasPrefix(name, "CQ_") || name == "VAULT_TOKEN" || name == "BAO_TOKEN"
}
