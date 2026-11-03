package platform

import (
	"context"
	"strings"

	"github.com/maldous/CircularQuay/internal/fault"
	"github.com/maldous/CircularQuay/internal/model"
)

// Activate requests a declared native service transition after an atomic
// projection. The caller must skip unchanged generations and authorize service
// control separately from reading or publishing configuration.
func Activate(ctx context.Context, target model.Target) error {
	if target.Activation == "" && target.Service == "" {
		return nil
	}
	if (target.Activation != "restart" && target.Activation != "reload") || !validServiceName(target.Service) {
		return fault.New(fault.Invalid, "native activation declaration")
	}
	return activateNative(ctx, target)
}

func validServiceName(name string) bool {
	if len(name) == 0 || len(name) > 200 || strings.HasPrefix(name, "-") {
		return false
	}
	for _, c := range name {
		if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || strings.ContainsRune("._@:-", c)) {
			return false
		}
	}
	return true
}
