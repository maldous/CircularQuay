//go:build !(aix || android || darwin || dragonfly || freebsd || illumos || ios || linux || netbsd || openbsd || solaris || windows)

package transport

// Unsupported platforms have no verified native socket-error mapping. They
// fail closed: unknown transport errors cannot authorize a fallback.
func knownConnectionErrno(error) bool { return false }

func nativeConnectionErrnos() []error { return nil }
