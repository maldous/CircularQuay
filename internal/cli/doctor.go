package cli

import (
	"flag"
	"io"
	"os"
	"path/filepath"
	"runtime"

	"github.com/maldous/CircularQuay/internal/fault"
	"github.com/maldous/CircularQuay/internal/mcp"
	"github.com/maldous/CircularQuay/internal/model"
	"github.com/maldous/CircularQuay/internal/platform"
)

// doctor reports local declaration and executable readiness without invoking a
// provider, reading credential contents, or changing service configuration.
func (a application) doctor(args []string) (int, error) {
	flags := flag.NewFlagSet("doctor", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	fix := flags.Bool("fix", false, "normalize a protected JSON config")
	identityOnly := flags.Bool("identity-only", false, "report local identity without config readiness")
	if flags.Parse(args) != nil || flags.NArg() != 0 {
		return 0, fault.New(fault.Invalid, "doctor arguments")
	}
	report := doctorReport{OS: runtime.GOOS, Arch: runtime.GOARCH, LocalSubject: doctorPrincipal()}
	if *identityOnly {
		report.Config = "not_checked"
		return a.print(report)
	}
	return a.doctorReadiness(&report, *fix)
}

func (a application) doctorReadiness(report *doctorReport, fix bool) (int, error) {
	path, err := filepath.Abs(a.config)
	if err != nil {
		return 0, fault.New(fault.Invalid, "policy path")
	}
	if _, err := os.Lstat(path); os.IsNotExist(err) {
		report.Config, report.Fix = "missing_default", "not_applicable"
		return printDoctor(a, *report, 1)
	}
	policy, err := a.load()
	if err != nil {
		return 0, err
	}
	report.Config, report.Dependencies = "valid", declaredDependencies(policy)
	if fix {
		changed, err := a.formatConfig()
		if err != nil {
			return 0, err
		}
		if changed {
			report.Fix = "formatted"
		} else {
			report.Fix = "already_formatted"
		}
	} else {
		report.Fix = "not_requested"
	}
	if doctorDependenciesReady(report.Dependencies) {
		return a.print(*report)
	}
	return printDoctor(a, *report, 1)
}

func doctorDependenciesReady(dependencies []doctorDependency) bool {
	for _, dependency := range dependencies {
		if dependency.State != "ready" {
			return false
		}
	}
	return true
}

type doctorReport struct {
	OS           string             `json:"os"`
	Arch         string             `json:"arch"`
	LocalSubject string             `json:"local_subject"`
	Config       string             `json:"config"`
	Dependencies []doctorDependency `json:"dependencies,omitempty"`
	Fix          string             `json:"fix,omitempty"`
}

type doctorDependency struct {
	Kind  string `json:"kind"`
	Path  string `json:"path"`
	State string `json:"state"`
}

func doctorPrincipal() string {
	principal, err := mcp.LocalPrincipal()
	if err != nil {
		return "unavailable"
	}
	return principal.Subject
}

func declaredDependencies(policy model.Policy) []doctorDependency {
	paths := make([]string, 0)
	for _, source := range policy.Sources {
		if source.SOPS != "" {
			paths = append(paths, source.SOPS)
		}
		if source.Type == "bitwarden" || source.Type == "onepassword" {
			paths = append(paths, source.Path)
		}
	}
	for _, workload := range policy.Workloads {
		if workload.Snapshot.SOPS != "" {
			paths = append(paths, workload.Snapshot.SOPS)
		}
		if len(workload.Command) > 0 {
			paths = append(paths, workload.Command[0])
		}
		for _, target := range workload.Targets {
			if target.Type == "systemd" {
				paths = append(paths, "/usr/bin/systemd-creds")
			}
		}
	}
	return append(doctorDependencies(paths), doctorPolicyMetadata(policy)...)
}

func doctorPolicyMetadata(policy model.Policy) []doctorDependency {
	paths := []string{policy.Repository.AllowedSignersFile, policy.Repository.TokenFile}
	for _, source := range policy.Sources {
		paths = append(paths, source.TokenFile, source.AgeIdentityFile)
		if source.Type == "json" || source.Type == "env" || source.Type == "sops" {
			paths = append(paths, source.Path)
		}
	}
	for _, workload := range policy.Workloads {
		paths = append(paths, workload.Snapshot.IdentityFile)
		for _, target := range workload.Targets {
			paths = append(paths, target.TokenFile)
		}
	}
	return doctorMetadata(paths)
}

func doctorDependencies(paths []string) []doctorDependency {
	seen := make(map[string]bool)
	result := make([]doctorDependency, 0, len(paths))
	for _, path := range paths {
		if path == "" || seen[path] {
			continue
		}
		seen[path] = true
		state := "ready"
		if !filepath.IsAbs(path) {
			state = "invalid_name"
		} else if err := platform.ValidateExecutable(path); err != nil {
			if _, statErr := os.Stat(path); os.IsNotExist(statErr) {
				state = "missing"
			} else {
				state = "untrusted"
			}
		}
		result = append(result, doctorDependency{Kind: "executable", Path: path, State: state})
	}
	return result
}

func doctorMetadata(paths []string) []doctorDependency {
	seen := make(map[string]bool)
	result := make([]doctorDependency, 0, len(paths))
	for _, path := range paths {
		if path == "" || seen[path] {
			continue
		}
		seen[path] = true
		state := "ready"
		if err := platform.ValidateFileMetadata(path, true); err != nil {
			if fault.Is(err, fault.Missing) {
				state = "missing"
			} else {
				state = "untrusted"
			}
		}
		result = append(result, doctorDependency{Kind: "protected_file", Path: path, State: state})
	}
	return result
}

func printDoctor(a application, report doctorReport, code int) (int, error) {
	if _, err := a.print(report); err != nil {
		return 0, err
	}
	return code, nil
}
