package cli

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/maldous/CircularQuay/internal/fault"
	"github.com/maldous/CircularQuay/internal/model"
	"github.com/maldous/CircularQuay/internal/platform"
)

type commandPreflightProbe struct {
	fetches  int
	currents int
}

func TestWorkloadCommandRejectsInvalidCommandBeforeResolutionEffects(t *testing.T) {
	for _, test := range []struct {
		name      string
		mutate    func(*model.Workload)
		override  []string
		wantFault fault.Kind
	}{
		{
			name:      "pinned executable override",
			mutate:    func(work *model.Workload) {},
			override:  []string{"/synthetic/other-executable", "--platform-helper", "status"},
			wantFault: fault.Trust,
		},
		{
			name:      "pinned argument override",
			mutate:    func(work *model.Workload) {},
			override:  []string{"configured-at-runtime", "--platform-helper", "different"},
			wantFault: fault.Trust,
		},
		{
			name:      "malformed pin",
			mutate:    func(work *model.Workload) { work.CommandSHA256 = "not-a-digest" },
			wantFault: fault.Trust,
		},
		{
			name:      "missing pinned command",
			mutate:    func(work *model.Workload) { work.Command = nil },
			wantFault: fault.Invalid,
		},
		{
			name:      "invalid unpinned executable",
			mutate:    func(work *model.Workload) { work.CommandSHA256 = "" },
			override:  []string{"relative-executable"},
			wantFault: fault.Invalid,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			app, policy, work, probe := commandPreflightFixture(t)
			test.mutate(&work)
			if test.name == "pinned argument override" {
				test.override[0] = work.Command[0]
			}
			policy.Workloads["demo"] = work
			args := commandPreflightArgs(test.override)
			_, err := app.workloadCommand(context.Background(), args, policy)
			if !fault.Is(err, test.wantFault) {
				t.Fatalf("preflight category = %v", err)
			}
			assertCommandPreflightNoResolutionEffects(t, app.directory, probe)
		})
	}
}

func TestPrepareWorkloadCommandRetainsMatchingPinnedAndUnpinnedOverrides(t *testing.T) {
	t.Run("matching pinned override", func(t *testing.T) {
		testPrepareWorkloadCommandAcceptsOverride(t, func(*model.Workload) {}, func(work model.Workload) []string {
			return append([]string(nil), work.Command...)
		})
	})
	t.Run("unpinned local override", func(t *testing.T) {
		testPrepareWorkloadCommandAcceptsOverride(t, func(work *model.Workload) {
			work.CommandSHA256 = ""
		}, func(work model.Workload) []string {
			return append(append([]string(nil), work.Command[:2]...), "exit")
		})
	})
}

func testPrepareWorkloadCommandAcceptsOverride(t *testing.T, mutate func(*model.Workload), override func(model.Workload) []string) {
	t.Helper()
	app, policy, work, probe := commandPreflightFixture(t)
	mutate(&work)
	policy.Workloads["demo"] = work
	request, err := parseWorkloadRequest(commandPreflightArgs(override(work)))
	if err != nil {
		t.Fatal(err)
	}
	_, accepted, engine, err := app.prepareWorkloadCommand(policy, request)
	if err != nil || engine == nil || accepted.CommandSHA256 != work.CommandSHA256 {
		t.Fatal("valid command preflight contract changed")
	}
	if probe.fetches != 0 || probe.currents != 0 {
		t.Fatal("command preflight contacted repository")
	}
}

func commandPreflightArgs(override []string) []string {
	args := []string{"run", "--workload", "demo"}
	if len(override) != 0 {
		args = append(args, "--")
		args = append(args, override...)
	}
	return args
}

func commandPreflightFixture(t *testing.T) (application, model.Policy, model.Workload, *commandPreflightProbe) {
	t.Helper()
	directory := privateCLIDirectory(t)
	binaryDirectory := filepath.Join(directory, "bin")
	if err := platform.EnsurePrivateDirectory(binaryDirectory); err != nil {
		t.Fatal(err)
	}
	executable := copyProtectedTestExecutable(t, binaryDirectory)
	contents, err := os.ReadFile(executable)
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(contents)
	probe := &commandPreflightProbe{}
	work := model.Workload{
		Mode:          "offline",
		Bindings:      []model.Binding{{Name: "VALUE", Source: "repository", Field: "value", AllowOffline: true}},
		Command:       []string{executable, "--platform-helper", "status"},
		CommandSHA256: hex.EncodeToString(digest[:]),
		Directory:     filepath.Join(directory, "work"),
	}
	policy := model.Policy{
		Version:    1,
		Repository: model.Repository{URL: "https://example.test/synthetic.git", Directory: filepath.Join(directory, "repository"), Ref: "refs/heads/main", AllowedSignersFile: filepath.Join(directory, "signers"), InitialCommit: strings.Repeat("a", 40)},
		Sources:    map[string]model.Source{"repository": {Type: "json", Path: "repo:values.json"}},
		Workloads:  map[string]model.Workload{"demo": work},
	}
	app := application{
		directory: filepath.Join(directory, "state"),
		repository: repositoryOperations{
			fetch: func(context.Context, model.Repository) (string, error) {
				probe.fetches++
				return "", errors.New("repository must not be contacted")
			},
			current: func(context.Context, model.Repository) (string, error) {
				probe.currents++
				return "", errors.New("repository must not be contacted")
			},
		},
	}
	return app, policy, work, probe
}

func assertCommandPreflightNoResolutionEffects(t *testing.T, stateDirectory string, probe *commandPreflightProbe) {
	t.Helper()
	if probe.fetches != 0 || probe.currents != 0 {
		t.Fatal("invalid command reached repository callbacks")
	}
	if _, err := os.Stat(stateDirectory); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("invalid command reached workload state")
	}
}
