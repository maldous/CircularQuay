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
	return activateNativeWithRunner(ctx, target, runSystemdCommand)
}

type systemCommand struct {
	path   string
	args   []string
	env    []string
	stdin  io.Reader
	stdout io.Writer
	stderr io.Writer
}

type systemCommandRunner func(context.Context, systemCommand) error

const systemdWaitDelay = time.Second

func activateNativeWithRunner(ctx context.Context, target model.Target, run systemCommandRunner) error {
	if !strings.HasSuffix(target.Service, ".service") {
		return fault.New(fault.Invalid, "systemd service name required")
	}
	if target.Activation != "restart" && target.Activation != "reload" {
		return fault.New(fault.Invalid, "systemd activation")
	}
	if run == nil {
		return fault.New(fault.Apply, "systemd service activation")
	}
	verb := "try-restart"
	if target.Activation == "reload" {
		verb = "reload"
	}
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	// No shell, inherited identity, password agent, or caller-controlled flags.
	if err := run(ctx, systemCommand{
		path: "/usr/bin/systemctl", args: []string{"--no-ask-password", verb, "--", target.Service},
		env: []string{}, stdout: io.Discard, stderr: io.Discard,
	}); err != nil {
		return fault.New(fault.Apply, "systemd service activation")
	}
	return nil
}

func runSystemdCommand(ctx context.Context, command systemCommand) error {
	cmd := exec.CommandContext(ctx, command.path, command.args...)
	// Both fixed commands use non-file I/O writers, so os/exec starts copier
	// goroutines. Bound an exited child that leaves those pipes open.
	cmd.WaitDelay = systemdWaitDelay
	cmd.Env = command.env
	cmd.Stdin, cmd.Stdout, cmd.Stderr = command.stdin, command.stdout, command.stderr
	return cmd.Run()
}
