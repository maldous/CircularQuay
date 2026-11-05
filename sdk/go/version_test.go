package cq

import (
	"context"
	"io"
	"strings"
	"testing"

	"github.com/maldous/CircularQuay/internal/cli"
)

type runtimeVersionCase struct {
	name     string
	version  string
	category Category
}

func TestLocalVersionClassifiesStrictRuntimeVersionDTO(t *testing.T) {
	for _, test := range []runtimeVersionCase{
		{name: "valid prerelease and build", version: "1.0.0-alpha.1+build.7"},
		{name: "valid v prefix", version: "v1.0.0"},
		{name: "maximum UTF-8 bytes", version: "1.0.0-" + strings.Repeat("a", 250)},
		{name: "malformed string", version: "1.invalid", category: CategoryProtocol},
		{name: "over maximum UTF-8 bytes", version: "1.0.0-" + strings.Repeat("a", 251), category: CategoryProtocol},
		{name: "abbreviated major", version: "1", category: CategoryProtocol},
		{name: "abbreviated minor", version: "1.2", category: CategoryProtocol},
		{name: "abbreviated incompatible major", version: "2", category: CategoryProtocol},
		{name: "leading zero minor", version: "1.01.0", category: CategoryProtocol},
		{name: "empty prerelease identifier", version: "1.0.0-alpha..1", category: CategoryProtocol},
		{name: "leading zero prerelease identifier", version: "1.0.0-01", category: CategoryProtocol},
		{name: "incompatible major", version: "2.0.0", category: CategoryUnsupported},
	} {
		t.Run(test.name, func(t *testing.T) {
			assertRuntimeVersionCase(t, test)
		})
	}
}

func assertRuntimeVersionCase(t *testing.T, test runtimeVersionCase) {
	t.Helper()
	client := versionFixtureClient(t, `{"version":"`+test.version+`","commit":"synthetic","go":"go1.27","maturity":"development"}`)
	version, err := client.Version(context.Background())
	if test.category != "" {
		if category(err) != test.category {
			t.Fatalf("category=%q error=%v", category(err), err)
		}
		return
	}
	if err != nil || version.Version != test.version {
		t.Fatalf("version=%+v error=%v", version, err)
	}
}

func TestLocalVersionRejectsMalformedDTO(t *testing.T) {
	for _, output := range []string{
		`{"version":1,"commit":"synthetic","go":"go1.27","maturity":"development"}`,
		`{"version":"1.0.0","commit":"synthetic","go":"go1.27"}`,
		`{"version":"1.0.0","commit":null,"go":"go1.27","maturity":"development"}`,
	} {
		client := versionFixtureClient(t, output)
		if _, err := client.Version(context.Background()); category(err) != CategoryProtocol {
			t.Fatalf("output=%s category=%q error=%v", output, category(err), err)
		}
	}
}

func TestAssertRuntimeCompatibilityUsesVersionContract(t *testing.T) {
	client := versionFixtureClient(t, `{"version":"1.0.0","commit":"synthetic","go":"go1.27","maturity":"development"}`)
	version, err := client.AssertRuntimeCompatibility(context.Background())
	if err != nil || version.Version != "1.0.0" {
		t.Fatalf("version=%+v error=%v", version, err)
	}
}

func versionFixtureClient(t *testing.T, output string) *LocalClient {
	t.Helper()
	client, err := NewLocal(localTestOptions(t))
	if err != nil {
		t.Fatal(err)
	}
	client.run = func(_ context.Context, runner cli.Runner, arguments []string) int {
		if len(arguments) != 5 || arguments[4] != "version" {
			t.Fatalf("arguments=%q", arguments)
		}
		_, _ = io.WriteString(runner.Out, output)
		return 0
	}
	return client
}
