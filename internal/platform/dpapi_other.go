//go:build !windows

package platform

import (
	"github.com/maldous/CircularQuay/internal/fault"
	"github.com/maldous/CircularQuay/internal/model"
)

func applyDPAPI(_ model.Target, _ []byte) error {
	return fault.New(fault.Unsupported, "DPAPI on this platform")
}

// ReadDPAPI is available only on the machine that produced a DPAPI projection.
func ReadDPAPI(_ string) ([]byte, error) {
	return nil, fault.New(fault.Unsupported, "DPAPI on this platform")
}
