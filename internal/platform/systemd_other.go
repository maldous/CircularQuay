//go:build !linux || android

package platform

import (
	"context"
	"github.com/maldous/CircularQuay/internal/fault"
	"github.com/maldous/CircularQuay/internal/model"
)

func applySystemd(_ context.Context, _ model.Target, _ []byte) error {
	return fault.New(fault.Unsupported, "systemd credentials on this platform")
}
