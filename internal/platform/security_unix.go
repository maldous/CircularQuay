//go:build !windows

package platform

func checkNativeSecurity(_ string, _ bool) bool { return true }
