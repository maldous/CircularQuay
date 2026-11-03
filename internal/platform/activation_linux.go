//go:build linux && !android

package platform

import (
	"context"
	"io"
	"os/exec"
	"strings"
	"time"

	"github.com/maldous/CircularQuay/internal/fault"
	"github.com/maldous/CircularQuay/internal/model"
)

func activateNative(ctx context.Context, target model.Target) error {
	if !strings.HasSuffix(target.Service, ".service") {
		return fault.New(fault.Invalid, "systemd service name required")
	}
	verb := "try-restart"
	if target.Activation == "reload" {
		verb = "reload"
	}
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	// No shell, inherited identity, password agent, or caller-controlled flags.
	cmd := exec.CommandContext(ctx, "/usr/bin/systemctl", "--no-ask-password", verb, "--", target.Service)
	cmd.Env = []string{}
	cmd.Stdout, cmd.Stderr = io.Discard, io.Discard
	if err := cmd.Run(); err != nil {
		return fault.New(fault.Apply, "systemd service activation")
	}
	return nil
}
