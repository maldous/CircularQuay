//go:build !windows

package platform

func nativeEnvironment() map[string]string { return map[string]string{} }
