package platform

import (
	"context"
	"github.com/maldous/CircularQuay/internal/fault"
	"github.com/maldous/CircularQuay/internal/model"
)

// Apply performs one explicitly authorized local projection. Authorization and
// snapshot acknowledgement belong to the calling policy/resolution boundary.
func Apply(ctx context.Context, target model.Target, gen model.Generation) error {
	if ctx.Err() != nil {
		return fault.New(fault.Apply, "projection canceled")
	}
	data, err := EncodeValues(gen)
	if err != nil {
		return err
	}
	switch target.Type {
	case "file", "json":
		return WriteProtected(target.Path, data)
	case "systemd":
		return applySystemd(ctx, target, data)
	case "windows":
		return applyDPAPI(target, data)
	default:
		return fault.New(fault.Unsupported, "native destination type")
	}
}
