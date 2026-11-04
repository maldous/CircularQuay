//go:build integration

package cquay

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	mcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

const officialMCPConformanceTimeout = 30 * time.Second

// TestOfficialMCPClientSessionAgainstSyntheticCquay exercises the exported Go
// SDK against a repository-built cquay stdio server through the official MCP
// Go SDK v1.8.0 CommandTransport and ClientSession. The fixture is an explicit
// hosted qualification gate: it has no skip path and requires a synthetic,
// scoped policy prepared by sdk/conformance/go-mcp.sh's calling workflow.
func TestOfficialMCPClientSessionAgainstSyntheticCquay(t *testing.T) {
	fixture := officialMCPFixtureFromEnvironment(t)
	ctx, cancel := context.WithTimeout(context.Background(), officialMCPConformanceTimeout)
	defer cancel()
	raw := connectOfficialMCPFixture(t, ctx, fixture)
	defer closeOfficialMCPFixture(t, ctx, raw)
	client, err := New(raw)
	if err != nil {
		t.Fatalf("adapt official MCP client session: %v", err)
	}
	assertSyntheticCatalog(t, ctx, client)
	assertSyntheticWorkloads(t, ctx, client)
	assertSyntheticStatus(t, ctx, client)
	assertSyntheticDisclosure(t, ctx, client)
	assertSyntheticDeniedStatus(t, ctx, client)
}

func connectOfficialMCPFixture(t *testing.T, ctx context.Context, fixture officialMCPFixture) *mcp.ClientSession {
	t.Helper()
	command := exec.CommandContext(ctx, fixture.executable, "--config", fixture.config, "--state-dir", fixture.stateDirectory, "mcp", "serve", "--transport", "stdio")
	// The synthetic server receives its paths as explicit arguments. Do not pass
	// the hosted job's environment, which could contain provider credentials.
	command.Env = []string{}
	transport := &mcp.CommandTransport{Command: command, TerminateDuration: 2 * time.Second}
	raw, err := mcp.NewClient(&mcp.Implementation{Name: "cquay-sdk-conformance", Version: "1.0.0"}, nil).Connect(ctx, transport, nil)
	if err != nil {
		t.Fatalf("connect official MCP client session: %v", err)
	}
	return raw
}

func closeOfficialMCPFixture(t *testing.T, ctx context.Context, raw *mcp.ClientSession) {
	t.Helper()
	if err := raw.Close(); err != nil && ctx.Err() == nil {
		t.Errorf("close official MCP client session: %v", err)
	}
}

func assertSyntheticCatalog(t *testing.T, ctx context.Context, client *Client) {
	t.Helper()
	tools, err := client.DiscoverTools(ctx)
	if err != nil {
		t.Fatalf("discover MCP tools: %v", err)
	}
	toolNames := make([]string, 0, len(tools))
	for _, tool := range tools {
		toolNames = append(toolNames, tool.Name)
	}
	slices.Sort(toolNames)
	if !slices.Equal(toolNames, []string{"get_status", "list_workloads", "read_secret"}) {
		t.Fatalf("unexpected authorized tool catalog: %q", toolNames)
	}
}

func assertSyntheticWorkloads(t *testing.T, ctx context.Context, client *Client) {
	t.Helper()
	workloads, err := client.ListWorkloads(ctx)
	if err != nil || !slices.Equal(workloads, []string{"demo"}) {
		t.Fatalf("list workloads: workloads=%q error=%v", workloads, err)
	}
}

func assertSyntheticStatus(t *testing.T, ctx context.Context, client *Client) {
	t.Helper()
	status, err := client.GetStatus(ctx, "demo")
	if err != nil || status.Workload != "demo" {
		t.Fatalf("get status: status=%+v error=%v", status, err)
	}
}

func assertSyntheticDisclosure(t *testing.T, ctx context.Context, client *Client) {
	t.Helper()
	secret, err := client.Disclosure(AllowPlaintext{}).ReadSecret(ctx, "demo", "DEMO_API_KEY")
	if err != nil || secret.Value() != "synthetic-only-not-a-credential" {
		t.Fatalf("read synthetic secret: error=%v", err)
	}
	encoded, marshalErr := json.Marshal(secret)
	if marshalErr != nil || strings.Contains(string(encoded), secret.Value()) || strings.Contains(secret.String(), secret.Value()) || strings.Contains(secret.GoString(), secret.Value()) {
		t.Fatalf("synthetic secret redaction failed: marshal error=%v", marshalErr)
	}
}

func assertSyntheticDeniedStatus(t *testing.T, ctx context.Context, client *Client) {
	t.Helper()
	_, err := client.GetStatus(ctx, "not-granted")
	if category(err) != CategoryDenied {
		t.Fatalf("status without scoped grant: category=%q error=%v", category(err), err)
	}
}

type officialMCPFixture struct {
	executable     string
	config         string
	stateDirectory string
}

func officialMCPFixtureFromEnvironment(t *testing.T) officialMCPFixture {
	t.Helper()
	if os.Getenv("CQUAY_GO_MCP_CONFORMANCE") != "1" {
		t.Fatal("CQUAY_GO_MCP_CONFORMANCE=1 is required for official MCP conformance")
	}
	fixture := officialMCPFixture{
		executable:     os.Getenv("CQUAY_EXECUTABLE"),
		config:         os.Getenv("CQUAY_CONFIG"),
		stateDirectory: os.Getenv("CQUAY_STATE_DIRECTORY"),
	}
	for _, path := range []string{fixture.executable, fixture.config, fixture.stateDirectory} {
		if path == "" || !filepath.IsAbs(path) {
			t.Fatal("official MCP conformance requires explicit absolute synthetic paths")
		}
	}
	info, err := os.Stat(fixture.executable)
	if err != nil || !info.Mode().IsRegular() || info.Mode().Perm()&0o111 == 0 {
		t.Fatal("official MCP conformance executable is unavailable")
	}
	return fixture
}
