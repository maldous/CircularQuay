//go:build !windows

package assurance

import (
	"os"

	"github.com/maldous/CircularQuay/internal/platform"
)

func openFixtureRuleInputForReplacement(path string) (*os.File, error) {
	return platform.OpenProtected(path, true)
}
