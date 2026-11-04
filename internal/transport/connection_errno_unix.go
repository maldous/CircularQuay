//go:build aix || android || darwin || dragonfly || freebsd || illumos || ios || linux || netbsd || openbsd || solaris

package transport

import (
	"errors"
	"syscall"
)

func knownConnectionErrno(err error) bool {
	for _, value := range nativeConnectionErrnos() {
		if errors.Is(err, value) {
			return true
		}
	}
	return false
}

func nativeConnectionErrnos() []error {
	return []error{syscall.ECONNREFUSED, syscall.ECONNRESET, syscall.ETIMEDOUT, syscall.ENETUNREACH, syscall.EHOSTUNREACH}
}
