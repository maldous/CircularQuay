package cquay

import (
	"context"
	"encoding/json"
	"io"
	"path/filepath"
	"runtime/debug"
	"strings"
	"testing"

	"github.com/maldous/CircularQuay/internal/cli"
)

func TestNewLocalRequiresExplicitAbsolutePaths(t *testing.T) {
	options := localTestOptions(t)
	if _, err := NewLocal(LocalOptions{ConfigPath: "cquay.json", StateDirectory: options.StateDirectory}); category(err) != CategoryInvalid {
		t.Fatalf("relative config accepted: %v", err)
	}
	if _, err := NewLocal(LocalOptions{ConfigPath: options.ConfigPath, StateDirectory: "state"}); category(err) != CategoryInvalid {
		t.Fatalf("relative state accepted: %v", err)
	}
}

func TestLocalClientUsesBoundedOwnedIO(t *testing.T) {
	options := localTestOptions(t)
	client, err := NewLocal(options)
	if err != nil {
		t.Fatal(err)
	}
	var gotArgs []string
	client.run = func(_ context.Context, runner cli.Runner, args []string) int {
		gotArgs = append([]string(nil), args...)
		_, _ = io.WriteString(runner.Out, `{"valid":true}`)
		return 0
	}
	result, err := client.ConfigValidate(context.Background())
	if err != nil || result.ExitCode != 0 || string(result.JSON) != `{"valid":true}` {
		t.Fatalf("result=%+v err=%v", result, err)
	}
	if strings.Join(gotArgs[:6], " ") != strings.Join([]string{"--config", options.ConfigPath, "--state-dir", options.StateDirectory, "config", "validate"}, " ") {
		t.Fatalf("unexpected args: %q", gotArgs)
	}
}

func TestLocalRunDiscardsChildOutput(t *testing.T) {
	client, err := NewLocal(localTestOptions(t))
	if err != nil {
		t.Fatal(err)
	}
	client.run = func(_ context.Context, runner cli.Runner, _ []string) int {
		if _, captures := runner.Out.(*boundedBuffer); captures || runner.Err != io.Discard {
			t.Fatal("run received a capturing output writer")
		}
		_, _ = io.WriteString(runner.Out, "SYNTHETIC_CHILD_OUTPUT")
		return 17
	}
	result, err := client.RunWorkload(context.Background(), "alpha")
	if err != nil || result.ExitCode != 17 {
		t.Fatalf("result=%+v err=%v", result, err)
	}
}

func TestConfigSetRejectsOversizedOrInvalidInputBeforeRunner(t *testing.T) {
	client, err := NewLocal(localTestOptions(t))
	if err != nil {
		t.Fatal(err)
	}
	called := false
	client.run = func(context.Context, cli.Runner, []string) int { called = true; return 0 }
	_, err = client.ConfigSet(context.Background(), "/alpha", json.RawMessage("invalid"), "")
	if category(err) != CategoryInvalid || called {
		t.Fatalf("invalid value was dispatched: %v called=%v", err, called)
	}
	_, err = client.ConfigSet(context.Background(), "/alpha", json.RawMessage(`{"duplicate":1,"duplicate":2}`), "")
	if category(err) != CategoryInvalid || called {
		t.Fatalf("duplicate-key JSON was dispatched: %v called=%v", err, called)
	}
	_, err = client.ConfigSet(context.Background(), "/alpha", json.RawMessage(`9007199254740992`), "")
	if category(err) != CategoryInvalid || called {
		t.Fatalf("unsafe integer was dispatched: %v called=%v", err, called)
	}
}

func TestPortableIJSONValidationBoundsStructureAndNumbers(t *testing.T) {
	if !validJSON(json.RawMessage(`9007199254740991`)) {
		t.Fatal("safe I-JSON integer rejected")
	}
	for _, value := range []string{
		`-9007199254740992`,
		`1e400`,
		strings.Repeat("[", maxIJSONDepth+1) + `0` + strings.Repeat("]", maxIJSONDepth+1),
		`[` + strings.TrimSuffix(strings.Repeat(`0,`, maxIJSONCollection+1), ",") + `]`,
	} {
		if validJSON(json.RawMessage(value)) {
			t.Fatalf("non-portable JSON accepted: %q", value)
		}
	}
}

func TestConfigMethodsRejectMalformedJSONPointerBeforeRunner(t *testing.T) {
	client, err := NewLocal(localTestOptions(t))
	if err != nil {
		t.Fatal(err)
	}
	called := false
	client.run = func(context.Context, cli.Runner, []string) int { called = true; return 0 }
	for _, pointer := range []string{"missing-leading-slash", "/bad~escape", "/bad~"} {
		if _, err := client.ConfigGet(context.Background(), pointer); category(err) != CategoryInvalid {
			t.Fatalf("pointer=%q err=%v", pointer, err)
		}
	}
	if called {
		t.Fatal("invalid pointer was dispatched")
	}
}

func TestBootstrapAndFlowRejectUnsafeConfigurationSources(t *testing.T) {
	client, err := NewLocal(localTestOptions(t))
	if err != nil {
		t.Fatal(err)
	}
	called := false
	client.run = func(context.Context, cli.Runner, []string) int { called = true; return 0 }
	request := BootstrapRequest{
		SourceURL: "https://github.com/maldous/CircularQuay",
		Ref:       "0123456789abcdef0123456789abcdef01234567",
		Path:      "examples/cquay.json",
		SHA256:    strings.Repeat("a", 64),
	}
	if !validConfigSource(request.SourceURL) {
		t.Fatal("valid HTTPS source rejected")
	}
	for _, source := range []string{
		"http://github.com/maldous/CircularQuay",
		"https://user:password@github.com/maldous/CircularQuay",
		"https://github.com/maldous/CircularQuay?token=synthetic",
		"https://github.com/maldous/CircularQuay?",
		"https://github.com/maldous/CircularQuay#fragment",
		"https://github.com/maldous/CircularQuay#",
		"relative/config.json",
		"https://" + strings.Repeat("a", 4090),
	} {
		request.SourceURL = source
		if _, err := client.Bootstrap(context.Background(), request); category(err) != CategoryInvalid {
			t.Fatalf("bootstrap source=%q category=%q", source, category(err))
		}
		if _, err := client.Flow(context.Background(), FlowRequest{Workload: "alpha", BootstrapRequest: request}); category(err) != CategoryInvalid {
			t.Fatalf("flow source=%q category=%q", source, category(err))
		}
	}
	if called {
		t.Fatal("unsafe configuration source was dispatched")
	}
}

func TestConfigSetUsesBoundedNonTTYInputInsteadOfArguments(t *testing.T) {
	client, err := NewLocal(localTestOptions(t))
	if err != nil {
		t.Fatal(err)
	}
	client.run = func(_ context.Context, runner cli.Runner, args []string) int {
		if strings.Join(args, " ") == "" || strings.Contains(strings.Join(args, " "), `{"safe":true}`) || !strings.Contains(strings.Join(args, " "), "--value-file -") {
			t.Fatalf("unsafe arguments: %q", args)
		}
		input, readErr := io.ReadAll(runner.In)
		if readErr != nil || string(input) != `{"safe":true}` {
			t.Fatalf("input=%q error=%v", input, readErr)
		}
		_, _ = io.WriteString(runner.Out, `{"sha256":"updated","state":"updated"}`)
		return 0
	}
	if _, err := client.ConfigSet(context.Background(), "/alpha", json.RawMessage(`{"safe":true}`), ""); err != nil {
		t.Fatal(err)
	}
}

func TestLocalFailureMapsSafeCLICategories(t *testing.T) {
	for raw, expected := range map[string]Category{
		`{"error":"invalid_policy"}`:         CategoryInvalid,
		`{"error":"access_denied"}`:          CategoryDenied,
		`{"error":"trust_failure"}`:          CategoryTrust,
		`{"error":"expired_value"}`:          CategoryExpired,
		`{"error":"unsupported_capability"}`: CategoryUnsupported,
		`{"error":"activation_failed"}`:      CategoryUnavailable,
		`{"error":"missing_value"}`:          CategoryUnavailable,
		`{"error":"unavailable"}`:            CategoryUnavailable,
	} {
		if actual := category(localFailure([]byte(raw))); actual != expected {
			t.Fatalf("raw=%s actual=%q expected=%q", raw, actual, expected)
		}
	}
	for _, raw := range []string{
		`{"error":"access_denied","extra":"ignored"}`,
		`{"error":"access_denied","error":"trust_failure"}`,
		`{"other":"access_denied"}`,
	} {
		if actual := category(localFailure([]byte(raw))); actual != CategoryProtocol {
			t.Fatalf("raw=%s actual=%q", raw, actual)
		}
	}
}

func TestNonzeroJSONOutputDoesNotReturnSuccessResult(t *testing.T) {
	client, err := NewLocal(localTestOptions(t))
	if err != nil {
		t.Fatal(err)
	}
	client.run = func(_ context.Context, runner cli.Runner, _ []string) int {
		_, _ = io.WriteString(runner.Out, `{"valid":false}`)
		return 1
	}
	if _, err := client.ConfigValidate(context.Background()); category(err) != CategoryProtocol {
		t.Fatalf("nonzero stdout category=%q error=%v", category(err), err)
	}
}

func TestNonzeroCommandWithoutDiagnosticIsProcessFailure(t *testing.T) {
	client, err := NewLocal(localTestOptions(t))
	if err != nil {
		t.Fatal(err)
	}
	client.run = func(context.Context, cli.Runner, []string) int { return 1 }
	if _, err := client.ConfigValidate(context.Background()); category(err) != CategoryProcess {
		t.Fatalf("empty diagnostic category=%q error=%v", category(err), err)
	}
}

func TestSDKBuildIdentityUsesOnlyOfficialEmbeddedModuleMetadata(t *testing.T) {
	original := readBuildInfo
	t.Cleanup(func() { readBuildInfo = original })
	readBuildInfo = func() (*debug.BuildInfo, bool) {
		return &debug.BuildInfo{
			Main:     debug.Module{Path: "example.invalid/host", Version: "v1.2.3"},
			Deps:     []*debug.Module{{Path: circularQuayModule, Version: "v1.0.0"}},
			Settings: []debug.BuildSetting{{Key: "vcs.revision", Value: "abcdef0123456789"}},
		}, true
	}
	identity := sdkBuildIdentity()
	if identity.version != "1.0.0" || identity.commit != "abcdef0123456789" {
		t.Fatalf("identity=%+v", identity)
	}
	client, err := NewLocal(localTestOptions(t))
	if err != nil {
		t.Fatal(err)
	}
	client.run = func(_ context.Context, runner cli.Runner, _ []string) int {
		if runner.BuildVersion != "1.0.0" || runner.BuildCommit != "abcdef0123456789" {
			t.Fatalf("runner identity version=%q commit=%q", runner.BuildVersion, runner.BuildCommit)
		}
		_, _ = io.WriteString(runner.Out, `{"valid":true}`)
		return 0
	}
	if _, err = client.ConfigValidate(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func localTestOptions(t *testing.T) LocalOptions {
	t.Helper()
	root, err := filepath.Abs(".")
	if err != nil {
		t.Fatal(err)
	}
	return LocalOptions{
		ConfigPath:     filepath.Join(root, "policy", "cquay.json"),
		StateDirectory: filepath.Join(root, "state", "cquay"),
	}
}

func TestSDKBuildIdentityKeepsDevelopmentFallbackForUntrustedVersions(t *testing.T) {
	original := readBuildInfo
	t.Cleanup(func() { readBuildInfo = original })
	for _, version := range []string{"(devel)", "v1.0.1-0.20260101010101-abcdef012345", "v2.0.0", ""} {
		readBuildInfo = func() (*debug.BuildInfo, bool) {
			return &debug.BuildInfo{Main: debug.Module{Path: circularQuayModule, Version: version}}, true
		}
		if identity := sdkBuildIdentity(); identity.version != developmentVersion {
			t.Fatalf("version=%q identity=%+v", version, identity)
		}
	}
}
