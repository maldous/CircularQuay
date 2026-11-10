//go:build windows

package transport

import (
	"errors"

	"golang.org/x/sys/windows"
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
	return []error{windows.WSAECONNREFUSED, windows.WSAECONNRESET, windows.WSAETIMEDOUT, windows.WSAENETUNREACH, windows.WSAEHOSTUNREACH}
}
