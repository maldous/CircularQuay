// Command cquay-audit generates independently reproducible assurance evidence.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/maldous/CircularQuay/internal/assurance"
)

func main() { os.Exit(run(os.Args[1:], os.Stdout, os.Stderr)) }

func run(arguments []string, stdout, stderr io.Writer) int {
	if len(arguments) > 0 && (arguments[0] == "sonar-init" || arguments[0] == "sonar-bootstrap") {
		return runSonar(arguments, stdout, stderr)
	}
	if len(arguments) == 0 || arguments[0] != "run" {
		_, _ = fmt.Fprintln(stderr, "usage: cquay-audit run --profile source|enterprise --output DIR")
		return 2
	}
	flags := flag.NewFlagSet("run", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	var options assurance.Options
	flags.StringVar(&options.Source, "source", ".", "source checkout")
	flags.StringVar(&options.Output, "output", "audit-results", "private evidence directory")
	flags.StringVar(&options.Profile, "profile", "enterprise", "source or enterprise")
	flags.StringVar(&options.GoBinary, "go", "go", "qualified Go executable")
	flags.StringVar(&options.SonarURL, "sonar-url", "", "qualified SonarQube URL")
	flags.StringVar(&options.SonarTokenFile, "sonar-token-file", "", "protected scanner token file")
	flags.StringVar(&options.SonarProjectKey, "sonar-project-key", "circularquay", "SonarQube project key")
	if flags.Parse(arguments[1:]) != nil || flags.NArg() != 0 {
		_, _ = fmt.Fprintln(stderr, "invalid audit arguments")
		return 2
	}
	report, err := assurance.Run(context.Background(), options)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, "audit could not complete; verify source, output and prerequisites")
		return 2
	}
	if err = json.NewEncoder(stdout).Encode(report); err != nil {
		return 2
	}
	if !report.Passed {
		return 1
	}
	return 0
}

func runSonar(arguments []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet(arguments[0], flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	directory := flags.String("directory", ".local/sonar", "private local bootstrap directory")
	endpoint := flags.String("url", "http://127.0.0.1:9000", "dedicated loopback SonarQube server")
	project := flags.String("project", "circularquay", "fresh audit project")
	if flags.Parse(arguments[1:]) != nil || flags.NArg() != 0 {
		_, _ = fmt.Fprintln(stderr, "invalid bootstrap arguments")
		return 2
	}
	var err error
	if arguments[0] == "sonar-init" {
		err = assurance.InitSonar(*directory)
	} else {
		err = assurance.BootstrapSonar(context.Background(), *directory, *endpoint, *project)
	}
	if err != nil {
		_, _ = fmt.Fprintln(stderr, "Sonar bootstrap failed; verify a fresh isolated server and private state")
		return 2
	}
	_, _ = fmt.Fprintln(stdout, "local Sonar bootstrap step completed; credential values are stored only in private state")
	return 0
}
