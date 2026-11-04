package cli

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/maldous/CircularQuay/internal/fault"
	"github.com/maldous/CircularQuay/internal/model"
	"github.com/maldous/CircularQuay/internal/update"
	"golang.org/x/term"
)

// checkUpdates performs only the advisory, interactive release check. The
// caller has already applied the minimum-version preflight before telemetry or
// other operational effects. MCP startup is deliberately excluded: a server
// must not make an ambient release-feed request when a terminal launches it.
func (a application) checkUpdates(ctx context.Context, command string, config *model.Updates) error {
	if !shouldCheckUpdates(command, config, interactiveTerminal(a.runner.In, a.runner.Err)) {
		return nil
	}
	store, err := a.store()
	if err != nil {
		return err
	}
	result, err := update.Check(ctx, config, update.Options{Version: a.version(), StateDirectory: store.Directory})
	if err != nil {
		_, _ = fmt.Fprintln(a.runner.Err, "CircularQuay could not check for updates; continuing.")
		return nil
	}
	if result.Status == "available" {
		_, _ = fmt.Fprintf(a.runner.Err, "A newer CircularQuay release is available: %s\nUpgrade instructions: %s\nContinuing with this invocation.\n", result.Available, result.InstructionURL)
	}
	return nil
}

func shouldCheckUpdates(command string, config *model.Updates, terminal bool) bool {
	if !terminal || config == nil || config.Check != "interactive" {
		return false
	}
	switch command {
	case "plan", "run", "sync", "snapshot", "apply", "status":
		return true
	default:
		return false
	}
}

// enforceMinimumVersion is deliberately usable before telemetry initialization:
// the policy minimum must be checked before any credential file or workload
// effect is reached. Doctor, config repair and version do not call it.
func enforceMinimumVersion(policy model.Policy, version string) error {
	return update.EnforceMinimum(version, minimumVersion(policy))
}

func minimumVersion(policy model.Policy) string {
	if policy.Updates == nil {
		return ""
	}
	return policy.Updates.MinimumVersion
}

func (a application) updateCommand(ctx context.Context, args []string) (int, error) {
	if len(args) == 0 || args[0] != "check" {
		return 0, fault.New(fault.Invalid, "update command")
	}
	return a.updateCheckCommand(ctx, args[1:])
}

// updateCheckCommand exposes the same bounded check as structured JSON. It
// deliberately does not prompt, download, install, or execute software.
func (a application) updateCheckCommand(ctx context.Context, args []string) (int, error) {
	flags := flag.NewFlagSet("update check", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	jsonOutput := flags.Bool("json", true, "write structured JSON")
	if flags.Parse(args) != nil || flags.NArg() != 0 || !*jsonOutput {
		return 0, fault.New(fault.Invalid, "update check arguments")
	}
	policy, err := a.load()
	if err != nil {
		return 0, err
	}
	store, err := a.store()
	if err != nil {
		return 0, err
	}
	result, err := update.Check(ctx, policy.Updates, update.Options{Version: a.version(), StateDirectory: store.Directory})
	if err != nil {
		return 0, err
	}
	return a.print(result)
}

func interactiveTerminal(in io.Reader, diagnostics io.Writer) bool {
	input, inputOK := in.(*os.File)
	output, outputOK := diagnostics.(*os.File)
	return inputOK && outputOK && term.IsTerminal(int(input.Fd())) && term.IsTerminal(int(output.Fd()))
}
