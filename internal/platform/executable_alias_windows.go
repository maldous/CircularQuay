//go:build windows

package platform

// checkNativeSecurity already rejects Windows reparse components while
// preserving the supported case and short-name aliases.
func trustedExecutableAliases(_ string) bool { return true }
