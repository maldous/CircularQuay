//go:build linux && !android

package platform

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/maldous/CircularQuay/internal/fault"
	"github.com/maldous/CircularQuay/internal/model"
)

func TestCompletionLinuxProjectionAndActivationAdmission(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	completionFault(t, Apply(ctx, model.Target{}, model.Generation{}), fault.Apply, "projection canceled")
	completionFault(t, Activate(ctx, model.Target{}), fault.Apply, "activation canceled")
	completionFault(t, Apply(context.Background(), model.Target{Type: "systemd", Path: "relative"}, model.Generation{}), fault.Invalid, "absolute systemd credential path required")
	completionFault(t, Activate(context.Background(), model.Target{Activation: "restart", Service: "owned"}), fault.Invalid, "systemd service name required")
	completionFault(t, applySystemdWithRunner(context.Background(), model.Target{Path: filepath.Join(secureTempDir(t), "credential")}, []byte("owned"), nil), fault.Apply, "systemd credential encryption")
	completionFault(t, activateNativeWithRunner(context.Background(), model.Target{Activation: "reload", Service: "owned.service"}, nil), fault.Apply, "systemd service activation")
	if validServiceName("owned/service") || !validServiceName("owned.service") {
		t.Fatal("valid closed service alphabet refused")
	}
	gen := model.Generation{Values: map[string]model.Value{"bad\x00": {Data: "owned"}}}
	completionFault(t, Apply(context.Background(), model.Target{}, gen), fault.Invalid, "invalid delivery value")
}
