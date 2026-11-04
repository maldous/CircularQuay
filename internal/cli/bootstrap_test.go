package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/maldous/CircularQuay/internal/bootstrap"
	"github.com/maldous/CircularQuay/internal/fault"
	"github.com/maldous/CircularQuay/internal/platform"
)

const bootstrapRef = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
const bootstrapDigest = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"

func TestBootstrapMaterializesValidatedPinnedPolicy(t *testing.T) {
	directory := privateCLIDirectory(t)
	output := filepath.Join(directory, "fetched.json")
	data := bootstrapPolicy()
	var stdout bytes.Buffer
	application := application{config: filepath.Join(directory, "default.json"), runner: Runner{Out: &stdout}}
	called := false
	code, err := application.bootstrapWithFetcher(context.Background(), []string{
		"--config-source", "https://github.com/acme/cquay/tree/" + bootstrapRef,
		"--config-ref", bootstrapRef, "--config-path", "config/policy.json", "--config-sha256", bootstrapDigest,
		"--output", output, "--non-interactive",
	}, func(_ context.Context, request bootstrap.Request) (bootstrap.Result, error) {
		called = true
		if request.ConfigRef != bootstrapRef || request.ExpectedSHA256 != bootstrapDigest || request.ConfigPath != "config/policy.json" {
			t.Fatalf("unexpected fetch request: %#v", request)
		}
		return bootstrap.Result{CanonicalSourceURL: "https://raw.githubusercontent.com/acme/cquay/" + bootstrapRef + "/config/policy.json", ConfigRef: bootstrapRef, ConfigPath: request.ConfigPath, SHA256: bootstrapDigest, Bytes: data}, nil
	})
	if err != nil || code != 0 || !called {
		t.Fatalf("bootstrap result = code %d, error %v", code, err)
	}
	stored, err := platform.ReadProtected(output, true)
	if err != nil || !bytes.Equal(stored, data) {
		t.Fatalf("materialized policy = %q, %v", stored, err)
	}
	assertBootstrapReceipt(t, stdout.Bytes(), output)
}

func TestBootstrapLocalValidatesExistingPolicy(t *testing.T) {
	directory := privateCLIDirectory(t)
	output := filepath.Join(directory, "local.json")
	if err := platform.CreateProtectedExclusive(output, bootstrapPolicy()); err != nil {
		t.Fatal(err)
	}
	var stdout bytes.Buffer
	application := application{config: filepath.Join(directory, "default.json"), runner: Runner{Out: &stdout}}
	code, err := application.bootstrapConfig(context.Background(), []string{"--output", output})
	if err != nil || code != 0 {
		t.Fatalf("local validation = code %d, error %v", code, err)
	}
	var receipt map[string]any
	if json.Unmarshal(stdout.Bytes(), &receipt) != nil || receipt["state"] != "validated" || receipt["source"] != "local" || receipt["version"] != float64(1) {
		t.Fatalf("local receipt = %s", stdout.Bytes())
	}
}

func TestBootstrapRejectsUnsafeMaterializationBeforeWrite(t *testing.T) {
	for _, test := range bootstrapRejectionCases() {
		t.Run(test.name, func(t *testing.T) {
			runBootstrapRejection(t, test)
		})
	}
}

func TestBootstrapRejectsRootMinimumBeforeMaterialization(t *testing.T) {
	directory := privateCLIDirectory(t)
	output := filepath.Join(directory, "minimum.json")
	app := application{config: filepath.Join(directory, "default.json"), runner: Runner{Out: &bytes.Buffer{}}}
	policy := []byte(`{"version":1,"sources":{"fixture":{"type":"json","path":"values.json"}},"workloads":{"demo":{"mode":"offline","bindings":[{"name":"DEMO","source":"fixture","field":"value","allow_offline":true}]}},"updates":{"check":"never","minimum_version":"v999.0.0"}}`)
	_, err := app.bootstrapWithFetcher(context.Background(), append(pinnedBootstrapArgs(), "--output", output), func(context.Context, bootstrap.Request) (bootstrap.Result, error) {
		return bootstrap.Result{Bytes: policy}, nil
	})
	if !fault.Is(err, fault.Denied) {
		t.Fatalf("root minimum bootstrap error = %v", err)
	}
	assertBootstrapAbsent(t, output)
}

type bootstrapRejectionCase struct {
	name         string
	args         []string
	result       bootstrap.Result
	prepare      func(*testing.T, string)
	wantFetches  int
	assertOutput func(*testing.T, string)
}

func bootstrapRejectionCases() []bootstrapRejectionCase {
	return []bootstrapRejectionCase{
		{"missing_pins", []string{"--config-source", "https://github.com/acme/cquay/tree/" + bootstrapRef}, bootstrap.Result{}, noBootstrapPreparation, 0, assertBootstrapAbsent},
		{"invalid_policy", pinnedBootstrapArgs(), bootstrap.Result{Bytes: []byte(`{"version":1}`)}, noBootstrapPreparation, 1, assertBootstrapAbsent},
		{"existing_output", pinnedBootstrapArgs(), bootstrap.Result{Bytes: bootstrapPolicy()}, createBootstrapExistingOutput, 0, assertBootstrapExisting},
	}
}

func runBootstrapRejection(t *testing.T, test bootstrapRejectionCase) {
	t.Helper()
	directory := privateCLIDirectory(t)
	output := filepath.Join(directory, "output.json")
	test.prepare(t, output)
	args := append(append([]string{}, test.args...), "--output", output)
	calls := 0
	application := application{config: filepath.Join(directory, "default.json"), runner: Runner{Out: &bytes.Buffer{}}}
	_, err := application.bootstrapWithFetcher(context.Background(), args, func(context.Context, bootstrap.Request) (bootstrap.Result, error) {
		calls++
		return test.result, nil
	})
	if err == nil || calls != test.wantFetches {
		t.Fatalf("rejection error=%v fetches=%d", err, calls)
	}
	test.assertOutput(t, output)
}

func noBootstrapPreparation(*testing.T, string) {}

func createBootstrapExistingOutput(t *testing.T, output string) {
	t.Helper()
	if err := platform.CreateProtectedExclusive(output, []byte("SYNTHETIC_EXISTING")); err != nil {
		t.Fatal(err)
	}
}

func TestBootstrapUsesEnvironmentDefaults(t *testing.T) {
	directory := privateCLIDirectory(t)
	output := filepath.Join(directory, "environment.json")
	t.Setenv(envCQUAYConfigSource, "https://github.com/acme/cquay/tree/"+bootstrapRef)
	t.Setenv(envCQUAYConfigRef, bootstrapRef)
	t.Setenv(envCQUAYConfigPath, "config/policy.json")
	t.Setenv(envCQUAYConfigSHA256, bootstrapDigest)
	application := application{config: filepath.Join(directory, "default.json"), runner: Runner{Out: &bytes.Buffer{}}}
	_, err := application.bootstrapWithFetcher(context.Background(), []string{"--output", output}, func(_ context.Context, request bootstrap.Request) (bootstrap.Result, error) {
		return bootstrap.Result{CanonicalSourceURL: request.SourceURL, ConfigRef: request.ConfigRef, ConfigPath: request.ConfigPath, SHA256: request.ExpectedSHA256, Bytes: bootstrapPolicy()}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestBootstrapDigestFailureDoesNotMaterialize(t *testing.T) {
	directory := privateCLIDirectory(t)
	output := filepath.Join(directory, "digest.json")
	application := application{config: filepath.Join(directory, "default.json"), runner: Runner{Out: &bytes.Buffer{}}}
	_, err := application.bootstrapWithFetcher(context.Background(), append(pinnedBootstrapArgs(), "--output", output), func(context.Context, bootstrap.Request) (bootstrap.Result, error) {
		return bootstrap.Result{}, fault.New(fault.Trust, "synthetic_digest_failure")
	})
	if !fault.Is(err, fault.Trust) {
		t.Fatalf("digest failure = %v", err)
	}
	assertBootstrapAbsent(t, output)
}

func pinnedBootstrapArgs() []string {
	return []string{
		"--config-source", "https://github.com/acme/cquay/tree/" + bootstrapRef,
		"--config-ref", bootstrapRef, "--config-path", "config/policy.json", "--config-sha256", bootstrapDigest,
	}
}

func bootstrapPolicy() []byte {
	return []byte(`{"version":1,"sources":{"fixture":{"type":"json","path":"values.json"}},"workloads":{"demo":{"mode":"offline","bindings":[{"name":"DEMO","source":"fixture","field":"value","allow_offline":true}]}}}`)
}

func assertBootstrapReceipt(t *testing.T, data []byte, output string) {
	t.Helper()
	var receipt map[string]any
	if json.Unmarshal(data, &receipt) != nil || receipt["state"] != "materialized" || receipt["policy"] != output || receipt["sha256"] != bootstrapDigest {
		t.Fatalf("invalid bootstrap receipt: %s", data)
	}
	if bytes.Contains(data, bootstrapPolicy()) {
		t.Fatal("bootstrap receipt included policy bytes")
	}
}

func assertBootstrapAbsent(t *testing.T, output string) {
	t.Helper()
	data, err := os.ReadFile(output)
	if !os.IsNotExist(err) {
		t.Fatalf("rejected bootstrap wrote output: %q, %v", data, err)
	}
}

func assertBootstrapExisting(t *testing.T, output string) {
	t.Helper()
	data, err := os.ReadFile(output)
	if err != nil || string(data) != "SYNTHETIC_EXISTING" {
		t.Fatalf("existing output changed: %q, %v", data, err)
	}
}
