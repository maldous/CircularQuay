//go:build windows

package platform

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/maldous/CircularQuay/internal/fault"
	"github.com/maldous/CircularQuay/internal/model"
)

func nativeEnvironment() map[string]string {
	result := map[string]string{}
	for _, key := range []string{"SystemRoot", "WINDIR"} {
		if value := os.Getenv(key); value != "" {
			result[key] = value
		}
	}
	return result
}

// launchDirectory constrains Windows DLL search directories before a trusted
// workload receives values. It is not a loader sandbox: reviewed applications
// remain responsible for their plugin and explicit-load behavior.
func launchDirectory(work model.Workload, generation model.Generation, executable string, environment []string) (string, error) {
	if work.UntrustedCode || len(generation.Values) == 0 {
		return work.Directory, nil
	}
	directory, err := workingDirectory(work.Directory)
	if err != nil {
		return "", err
	}
	if _, err := trustedLaunchDirectory(filepath.Dir(executable)); err != nil {
		return "", err
	}
	if err := trustedPATHDirectories(environment); err != nil {
		return "", err
	}
	return directory, nil
}

func workingDirectory(configured string) (string, error) {
	if configured == "" {
		current, err := os.Getwd()
		if err != nil {
			return "", fault.New(fault.Trust, "working directory unavailable")
		}
		configured = current
	}
	return trustedLaunchDirectory(configured)
}

func trustedPATHDirectories(environment []string) error {
	for _, entry := range environment {
		name, value, found := strings.Cut(entry, "=")
		if found && strings.EqualFold(name, "PATH") {
			return trustedPATHValue(value)
		}
	}
	return nil
}

func trustedPATHValue(value string) error {
	if value == "" {
		return fault.New(fault.Trust, "empty PATH directory")
	}
	for _, directory := range filepath.SplitList(value) {
		if directory == "" || !filepath.IsAbs(directory) {
			return fault.New(fault.Trust, "absolute PATH directory required")
		}
		if _, err := trustedLaunchDirectory(directory); err != nil {
			return err
		}
	}
	return nil
}

func trustedLaunchDirectory(path string) (string, error) {
	if !filepath.IsAbs(path) || !resolvedPathIsOriginal(path) {
		return "", fault.New(fault.Trust, "absolute trusted directory required")
	}
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil || !filepath.IsAbs(resolved) {
		return "", fault.New(fault.Trust, "trusted directory unavailable")
	}
	info, err := os.Stat(resolved)
	if err != nil || !info.IsDir() || !resolvedPathIsOriginal(resolved) || !checkNativeSecurity(resolved, false) {
		return "", fault.New(fault.Trust, "trusted directory permissions")
	}
	return resolved, nil
}
