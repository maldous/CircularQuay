package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/maldous/CircularQuay/internal/bootstrap"
	"github.com/maldous/CircularQuay/internal/fault"
	"github.com/maldous/CircularQuay/internal/model"
	"github.com/maldous/CircularQuay/internal/platform"
)

func TestDetectCIRunnerAcceptsExactContexts(t *testing.T) {
	for _, test := range []struct {
		key, value string
		want       ciRunner
	}{
		{"GITHUB_ACTIONS", "true", ciGitHub},
		{"GITLAB_CI", "true", ciGitLab},
		{"TF_BUILD", "True", ciAzure},
	} {
		t.Run(test.want.String(), func(t *testing.T) {
			clearFlowEnvironment(t)
			t.Setenv(test.key, test.value)
			got, err := detectCIRunner()
			if err != nil || got != test.want {
				t.Fatalf("runner = %q, %v", got, err)
			}
		})
	}
}

func (r ciRunner) String() string { return string(r) }

func TestDetectCIRunnerRejectsUntrustedAndAmbiguousMarkers(t *testing.T) {
	for _, test := range []map[string]string{
		{}, {"CI": "true"}, {"GITHUB_ACTIONS": "TRUE"},
		{"GITHUB_ACTIONS": "true", "GITLAB_CI": "true"},
	} {
		t.Run("invalid", func(t *testing.T) {
			clearFlowEnvironment(t)
			for key, value := range test {
				t.Setenv(key, value)
			}
			if _, err := detectCIRunner(); !fault.Is(err, fault.Invalid) {
				t.Fatalf("context error = %v", err)
			}
		})
	}
}

func TestFlowRejectsArgumentsAndPinsBeforeFetch(t *testing.T) {
	clearFlowEnvironment(t)
	t.Setenv("GITHUB_ACTIONS", "true")
	a := application{config: "missing.json", runner: Runner{Out: &bytes.Buffer{}}}
	for _, args := range [][]string{
		{"run", "--workload", "demo", "--", "ignored"},
		{"run", "--workload", "demo", "--config-source", "https://github.com/acme/config"},
		{"run", "--workload", "demo", "--config-source", "https://github.com/acme/config", "--config-ref", "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA", "--config-sha256", bootstrapDigest},
	} {
		calls := 0
		_, err := a.flowWithFetcher(context.Background(), args, func(context.Context, bootstrap.Request) (bootstrap.Result, error) {
			calls++
			return bootstrap.Result{}, nil
		})
		if !fault.Is(err, fault.Invalid) || calls != 0 {
			t.Fatalf("flow %q error=%v calls=%d", args, err, calls)
		}
	}
}

func TestFlowExplicitConfigWinsOverAmbientSource(t *testing.T) {
	clearFlowEnvironment(t)
	t.Setenv("GITHUB_ACTIONS", "true")
	t.Setenv(envCQUAYConfigSource, "https://github.com/acme/config/tree/"+bootstrapRef)
	t.Setenv(envCQUAYConfigRef, bootstrapRef)
	t.Setenv(envCQUAYConfigSHA256, bootstrapDigest)
	a := application{config: "missing-local.json", configExplicit: true, runner: Runner{Out: &bytes.Buffer{}}}
	calls := 0
	_, err := a.flowWithFetcher(context.Background(), []string{"run", "--workload", "demo"}, func(context.Context, bootstrap.Request) (bootstrap.Result, error) {
		calls++
		return bootstrap.Result{}, nil
	})
	if !fault.Is(err, fault.Denied) || calls != 0 {
		t.Fatalf("explicit local selection error=%v calls=%d", err, calls)
	}
}

func TestFlowOptionsOverrideEnvironmentPerField(t *testing.T) {
	clearFlowEnvironment(t)
	t.Setenv(envCQUAYConfigSource, "https://github.com/acme/config/tree/"+bootstrapRef)
	t.Setenv(envCQUAYConfigRef, bootstrapRef)
	t.Setenv(envCQUAYConfigPath, "from-environment.json")
	t.Setenv(envCQUAYConfigSHA256, bootstrapDigest)
	o, err := (application{}).flowOptions([]string{"run", "--workload", "demo", "--config-sha256", "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"})
	if err != nil || !o.explicit || o.source.sourceURL == "" || o.source.ref != bootstrapRef || o.source.path != "from-environment.json" || o.source.digest != "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc" {
		t.Fatalf("flow option precedence = %+v, %v", o, err)
	}
}

func TestRunnerFlowExplicitConfigSuppressesAmbientRemote(t *testing.T) {
	clearFlowEnvironment(t)
	t.Setenv("GITHUB_ACTIONS", "true")
	t.Setenv(envCQUAYConfigSource, "https://github.com/acme/ignored/tree/"+bootstrapRef)
	parent := privateCLIDirectory(t)
	values := filepath.Join(parent, "values.json")
	writeProtectedFixture(t, values, []byte(`{"message":"synthetic-flow-value"}`))
	working := filepath.Join(parent, "work")
	if err := platform.EnsurePrivateDirectory(working); err != nil {
		t.Fatal(err)
	}
	config := filepath.Join(parent, "local.json")
	if err := platform.CreateProtectedExclusive(config, flowPolicy(t, copyProtectedTestExecutable(t, parent), values, working, "success")); err != nil {
		t.Fatal(err)
	}
	var output, diagnostics bytes.Buffer
	code := (Runner{Out: &output, Err: &diagnostics}).Run(context.Background(), []string{"--config", config, "--state-dir", filepath.Join(parent, "state"), "flow", "run", "--workload", "demo"})
	if code != 0 || !bytes.Contains(output.Bytes(), []byte("flow-child-ack")) || !bytes.Contains(diagnostics.Bytes(), []byte(`"source_kind":"local"`)) || bytes.Contains(diagnostics.Bytes(), []byte("synthetic-flow-value")) {
		t.Fatalf("explicit local runner flow code=%d stdout=%q stderr=%q", code, output.Bytes(), diagnostics.Bytes())
	}
}

func TestFlowCancellationPreventsFetchAndTemporaryPolicy(t *testing.T) {
	clearFlowEnvironment(t)
	t.Setenv("GITHUB_ACTIONS", "true")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	calls := 0
	a := application{config: "missing.json", runner: Runner{Err: &bytes.Buffer{}}}
	_, err := a.flowWithFetcher(ctx, []string{"run", "--workload", "demo", "--config-source", "https://github.com/acme/config/tree/" + bootstrapRef, "--config-ref", bootstrapRef, "--config-sha256", bootstrapDigest}, func(context.Context, bootstrap.Request) (bootstrap.Result, error) {
		calls++
		return bootstrap.Result{}, nil
	})
	if err == nil || calls != 0 {
		t.Fatalf("pre-cancel flow error=%v fetches=%d", err, calls)
	}
}

func TestFlowCancellationAfterFetchLeavesNoTemporaryPolicy(t *testing.T) {
	clearFlowEnvironment(t)
	t.Setenv("GITHUB_ACTIONS", "true")
	parent := privateCLIDirectory(t)
	root := filepath.Join(parent, "cancelled")
	newFlowTempDir = func(string, string) (string, error) { return root, nil }
	t.Cleanup(func() { newFlowTempDir = os.MkdirTemp })
	ctx, cancel := context.WithCancel(context.Background())
	a := application{config: "missing.json", runner: Runner{Err: &bytes.Buffer{}}}
	_, err := a.flowWithFetcher(ctx, []string{"run", "--workload", "demo", "--config-source", "https://github.com/acme/config/tree/" + bootstrapRef, "--config-ref", bootstrapRef, "--config-sha256", bootstrapDigest}, func(context.Context, bootstrap.Request) (bootstrap.Result, error) {
		cancel()
		return bootstrap.Result{Bytes: bootstrapPolicy()}, nil
	})
	if err == nil {
		t.Fatal("fetch-triggered cancellation was accepted")
	}
	if _, err := os.Lstat(root); !os.IsNotExist(err) {
		t.Fatalf("canceled flow created temporary policy state: %v", err)
	}
}

func TestFlowRemovesTemporaryPolicyAfterSuccessAndChildFailure(t *testing.T) {
	clearFlowEnvironment(t)
	t.Setenv("GITHUB_ACTIONS", "true")
	fixture := newFlowExecutionFixture(t)
	for _, outcome := range []string{"success", "failure"} {
		t.Run(outcome, func(t *testing.T) {
			assertFlowExecutionOutcome(t, fixture, outcome)
		})
	}
}

type flowExecutionFixture struct {
	parent, state, executable, values, working string
}

func newFlowExecutionFixture(t *testing.T) flowExecutionFixture {
	t.Helper()
	parent := privateCLIDirectory(t)
	fixture := flowExecutionFixture{
		parent: parent, state: filepath.Join(parent, "state"),
		executable: copyProtectedTestExecutable(t, parent),
		values:     filepath.Join(parent, "values.json"), working: filepath.Join(parent, "work"),
	}
	writeProtectedFixture(t, fixture.values, []byte(`{"message":"synthetic-flow-value"}`))
	if err := platform.EnsurePrivateDirectory(fixture.working); err != nil {
		t.Fatal(err)
	}
	return fixture
}

func assertFlowExecutionOutcome(t *testing.T, fixture flowExecutionFixture, outcome string) {
	t.Helper()
	root := filepath.Join(fixture.parent, "flow-"+outcome)
	useFlowTemporaryDirectory(t, root)
	data := flowPolicy(t, fixture.executable, fixture.values, fixture.working, outcome)
	var output, diagnostics bytes.Buffer
	a := application{config: "missing.json", directory: fixture.state, runner: Runner{Out: &output, Err: &diagnostics}}
	code, err := a.flowWithFetcher(context.Background(), []string{"run", "--workload", "demo", "--config-source", "https://github.com/acme/config/tree/" + bootstrapRef, "--config-ref", bootstrapRef, "--config-sha256", bootstrapDigest}, flowFixtureFetch(data))
	if err != nil || code != map[string]int{"success": 0, "failure": 17}[outcome] {
		t.Fatalf("flow %s = code %d, error %v", outcome, code, err)
	}
	if bytes.Contains(append(output.Bytes(), diagnostics.Bytes()...), []byte("synthetic-flow-value")) {
		t.Fatal("wrapper output exposed a resolved value")
	}
	if _, err := os.Lstat(root); !os.IsNotExist(err) {
		t.Fatalf("temporary policy directory remained: %v", err)
	}
}

func useFlowTemporaryDirectory(t *testing.T, root string) {
	t.Helper()
	newFlowTempDir = func(parent, _ string) (string, error) {
		if parent == "" {
			t.Fatal("flow used an uncanonicalized temporary parent")
		}
		if err := platform.EnsurePrivateDirectory(root); err != nil {
			return "", err
		}
		return root, nil
	}
	t.Cleanup(func() { newFlowTempDir = os.MkdirTemp })
}

func TestFlowRemovesTemporaryPolicyAfterParseFailure(t *testing.T) {
	clearFlowEnvironment(t)
	t.Setenv("GITHUB_ACTIONS", "true")
	parent := privateCLIDirectory(t)
	root := filepath.Join(parent, "parse-failure")
	newFlowTempDir = func(parent, _ string) (string, error) {
		if parent == "" {
			t.Fatal("flow used an uncanonicalized temporary parent")
		}
		if err := platform.EnsurePrivateDirectory(root); err != nil {
			return "", err
		}
		return root, nil
	}
	t.Cleanup(func() { newFlowTempDir = os.MkdirTemp })
	a := application{config: "missing.json", runner: Runner{Out: &bytes.Buffer{}, Err: &bytes.Buffer{}}}
	_, err := a.flowWithFetcher(context.Background(), []string{"run", "--workload", "demo", "--config-source", "https://github.com/acme/config/tree/" + bootstrapRef, "--config-ref", bootstrapRef, "--config-sha256", bootstrapDigest}, flowFixtureFetch([]byte(`{"version":1}`)))
	if err == nil {
		t.Fatal("invalid remote policy was accepted")
	}
	if _, err := os.Lstat(root); !os.IsNotExist(err) {
		t.Fatalf("parse failure retained temporary policy directory: %v", err)
	}
}

func flowPolicy(t *testing.T, executable, values, working, outcome string) []byte {
	t.Helper()
	p := model.Policy{Version: 1, Sources: map[string]model.Source{"fixture": {Type: "json", Path: values}}, Workloads: map[string]model.Workload{"demo": {Mode: "offline", Directory: working, Bindings: []model.Binding{{Name: "MESSAGE", Source: "fixture", Field: "message", AllowOffline: true}}, Command: []string{executable, "-test.run=^TestFlowChild$", "--", "cquay-flow-child", outcome}}}}
	data, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func flowFixtureFetch(data []byte) bootstrapFetch {
	return func(_ context.Context, request bootstrap.Request) (bootstrap.Result, error) {
		return bootstrap.Result{CanonicalSourceURL: request.SourceURL, ConfigRef: request.ConfigRef, ConfigPath: request.ConfigPath, SHA256: request.ExpectedSHA256, Bytes: data}, nil
	}
}

func TestFlowChild(t *testing.T) {
	if len(os.Args) < 3 || os.Args[len(os.Args)-2] != "cquay-flow-child" {
		return
	}
	if os.Getenv("MESSAGE") != "synthetic-flow-value" {
		os.Exit(18)
	}
	fmt.Fprintln(os.Stdout, "flow-child-ack")
	if os.Args[len(os.Args)-1] == "failure" {
		os.Exit(17)
	}
}

func clearFlowEnvironment(t *testing.T) {
	t.Helper()
	for _, key := range []string{"GITHUB_ACTIONS", "GITLAB_CI", "TF_BUILD", envCQUAYConfigSource, envCQUAYConfigRef, envCQUAYConfigPath, envCQUAYConfigSHA256} {
		value, present := os.LookupEnv(key)
		if err := os.Unsetenv(key); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() {
			if present {
				_ = os.Setenv(key, value)
				return
			}
			_ = os.Unsetenv(key)
		})
	}
}
