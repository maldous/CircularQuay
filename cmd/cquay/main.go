// Command cquay delivers only explicitly configured workload values.
package main

import (
	"context"
	"github.com/maldous/CircularQuay/internal/cli"
	"os"
	"os/signal"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()
	os.Exit(cli.Runner{}.Run(ctx, os.Args[1:]))
}
