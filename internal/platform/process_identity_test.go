package platform

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/maldous/CircularQuay/internal/fault"
	"github.com/maldous/CircularQuay/internal/model"
)

func TestPinnedLaunchRejectsOverrideIdentityChangesBeforeLaunchInputs(t *testing.T) {
	work, generation := pinnedProcessIdentityFixture(t)
	for _, test := range []struct {
		name     string
		override []string
	}{
		{
			name:     "executable",
			override: []string{filepath.Join(work.Directory, "override-executable"), "--platform-helper", "status"},
		},
		{
			name:     "argument",
			override: []string{work.Command[0], "--platform-helper", "exit"},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			args, environment, directory, err := launchInputs(work, generation, test.override)
			if !fault.Is(err, fault.Trust) {
				t.Fatalf("pinned override error category = %v", err)
			}
			if args != nil || environment != nil || directory != "" {
				t.Fatal("pinned override prepared launch inputs")
			}
		})
	}
}

func TestPinnedLaunchVerifiesMatchingOverrideAndMCPCommand(t *testing.T) {
	work, generation := pinnedProcessIdentityFixture(t)
	for _, test := range []struct {
		name     string
		digest   string
		override []string
	}{
		{
			name:     "matching local override",
			digest:   work.CommandSHA256,
			override: append([]string(nil), work.Command...),
		},
		{
			name:   "nil MCP override",
			digest: work.CommandSHA256,
		},
		{
			name:     "matching override with mismatched digest",
			digest:   "0000000000000000000000000000000000000000000000000000000000000000",
			override: append([]string(nil), work.Command...),
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			candidate := work
			candidate.CommandSHA256 = test.digest
			args, environment, directory, err := launchInputs(candidate, generation, test.override)
			if test.digest != work.CommandSHA256 {
				if !fault.Is(err, fault.Trust) || args != nil || environment != nil || directory != "" {
					t.Fatal("matching override bypassed executable verification")
				}
				return
			}
			if err != nil {
				t.Fatalf("matching pinned command denied: %v", err)
			}
			if !reflect.DeepEqual(args, work.Command) || directory != work.Directory || len(environment) == 0 {
				t.Fatal("matching pinned command inputs changed")
			}
		})
	}
}

func TestUnpinnedLocalOverrideRemainsPermitted(t *testing.T) {
	work, generation := pinnedProcessIdentityFixture(t)
	work.CommandSHA256 = ""
	override := []string{work.Command[0], "--platform-helper", "exit"}
	args, environment, directory, err := launchInputs(work, generation, override)
	if err != nil {
		t.Fatalf("unpinned local override denied: %v", err)
	}
	if !reflect.DeepEqual(args, override) || directory != work.Directory || len(environment) == 0 {
		t.Fatal("unpinned local override inputs changed")
	}
}

func TestPinnedLaunchWithMissingConfiguredCommandFailsClosed(t *testing.T) {
	work, generation := pinnedProcessIdentityFixture(t)
	work.Command = nil
	if _, environment, directory, err := launchInputs(work, generation, nil); !fault.Is(err, fault.Invalid) || environment != nil || directory != "" {
		t.Fatal("missing pinned command was accepted")
	}
	if _, environment, directory, err := launchInputs(work, generation, []string{work.CommandSHA256}); !fault.Is(err, fault.Trust) || environment != nil || directory != "" {
		t.Fatal("missing pinned command override was accepted")
	}
}

func pinnedProcessIdentityFixture(t *testing.T) (model.Workload, model.Generation) {
	t.Helper()
	executable, directory := protectedPlatformHelper(t)
	contents, err := os.ReadFile(executable)
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(contents)
	return model.Workload{
		Command:       []string{executable, "--platform-helper", "status"},
		CommandSHA256: hex.EncodeToString(digest[:]),
		Directory:     directory,
	}, fixtureGeneration()
}
