// Command cquay delivers only explicitly configured workload values.
package main

import (
	"context"
	"os"
	"os/signal"
	"runtime"
	"syscall"

	"github.com/maldous/CircularQuay/internal/cli"
)

func main() {
	ctx, cancel := rootSignalNotifyContext(context.Background())
	defer cancel()
	os.Exit(cli.Runner{}.Run(ctx, os.Args[1:]))
}

func rootSignalNotifyContext(parent context.Context) (context.Context, context.CancelFunc) {
	signals := []os.Signal{os.Interrupt}
	if runtime.GOOS != "windows" {
		signals = append(signals, syscall.SIGTERM)
	}
	return signal.NotifyContext(parent, signals...)
}
