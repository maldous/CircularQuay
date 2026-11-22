//go:build (!linux && !windows) || android

package platform

import (
	"context"
	"github.com/maldous/CircularQuay/internal/fault"
	"github.com/maldous/CircularQuay/internal/model"
)

func activateNative(context.Context, model.Target) error {
	return fault.New(fault.Unsupported, "native service activation on this platform")
}
