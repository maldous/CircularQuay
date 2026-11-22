//go:build !windows

package platform

import "github.com/maldous/CircularQuay/internal/model"

func nativeEnvironment() map[string]string { return map[string]string{} }

// launchDirectory preserves the Unix execution contract. Windows has an
// additional DLL search-path boundary for trusted secret-bearing workloads.
func launchDirectory(work model.Workload, _ model.Generation, _ string, _ []string) (string, error) {
	return work.Directory, nil
}
