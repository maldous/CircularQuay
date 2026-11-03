package cli

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/maldous/CircularQuay/internal/fault"
	cqmcp "github.com/maldous/CircularQuay/internal/mcp"
	"github.com/maldous/CircularQuay/internal/model"
)

func TestParseMCPServeOptions(t *testing.T) {
	t.Parallel()

	options, err := parseMCPServeOptions([]string{"serve", "--transport", "http", "--tls-cert", "certificate.pem", "--tls-key", "private-key.pem"})
	if err != nil || options.transport != "http" || options.cert != "certificate.pem" || options.key != "private-key.pem" {
		t.Fatalf("MCP options = %+v, %v", options, err)
	}
	for _, args := range [][]string{{}, {"status"}, {"serve", "--unknown"}, {"serve", "unexpected"}} {
		if _, err := parseMCPServeOptions(args); !fault.Is(err, fault.Invalid) {
			t.Fatalf("arguments %q error = %v", args, err)
		}
	}
}

func TestLoopback(t *testing.T) {
	t.Parallel()

	for host, want := range map[string]bool{
		"localhost":   true,
		"LOCALHOST":   true,
		"127.0.0.1":   true,
		"::1":         true,
		"0.0.0.0":     false,
		"example.com": false,
	} {
		if got := loopback(host); got != want {
			t.Errorf("loopback(%q) = %t, want %t", host, got, want)
		}
	}
}

func TestMCPHTTPRejectsPublicPlaintextBeforeListening(t *testing.T) {
	t.Parallel()

	policy := model.Policy{
		Version: 1,
		Sources: map[string]model.Source{"fixture": {Type: "json", Path: "synthetic.json"}},
		Workloads: map[string]model.Workload{"demo": {
			Mode:     "offline",
			Bindings: []model.Binding{{Name: "TOKEN", Source: "fixture", Field: "TOKEN", AllowOffline: true}},
		}},
		MCP: model.MCP{Issuer: "https://issuer.example", Audience: "https://api.example/mcp", JWKSURL: "https://issuer.example/keys"},
	}
	service, err := cqmcp.New(policy, backend{}, cqmcp.Options{})
	if err != nil {
		t.Fatal(err)
	}
	_, err = runMCPHTTP(context.Background(), "0.0.0.0:0", service, mcpServeOptions{transport: "http"})
	if !fault.Is(err, fault.Trust) {
		t.Fatalf("public plaintext listener error = %v", err)
	}
}

func TestActionAllowedRequiresExactExecutableDigest(t *testing.T) {
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	digest, err := executableDigest(executable)
	if err != nil {
		t.Fatal(err)
	}
	dir := privateCLIDirectory(t)
	fixture := filepath.Join(dir, "synthetic.json")
	if err = os.WriteFile(fixture, []byte(`{"TOKEN":"synthetic"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	config := filepath.Join(dir, "policy.json")
	policy := model.Policy{
		Version: 1,
		Sources: map[string]model.Source{"fixture": {Type: "json", Path: fixture}},
		Workloads: map[string]model.Workload{"demo": {
			Mode:          "offline",
			Bindings:      []model.Binding{{Name: "TOKEN", Source: "fixture", Field: "TOKEN", AllowOffline: true}},
			Command:       []string{executable},
			CommandSHA256: digest,
		}},
	}
	writeMCPTestPolicy(t, config, policy)

	app := application{config: config}
	if !app.actionAllowed("demo") {
		t.Fatal("matching executable digest was denied")
	}
	policy.Workloads["demo"] = model.Workload{Mode: "offline", Bindings: []model.Binding{{Name: "TOKEN", Source: "fixture", Field: "TOKEN", AllowOffline: true}}, Command: []string{executable}, CommandSHA256: strings.Repeat("0", 64)}
	writeMCPTestPolicy(t, config, policy)
	if app.actionAllowed("demo") {
		t.Fatal("mismatched executable digest was allowed")
	}
}

func TestMCPRunRechecksAuthorizationAfterResolution(t *testing.T) {
	for name, mutate := range map[string]func(*model.Policy){
		"grant revoked": func(p *model.Policy) {
			p.Grants = nil
		},
		"executable changed": func(p *model.Policy) {
			w := p.Workloads["demo"]
			w.CommandSHA256 = strings.Repeat("0", 64)
			p.Workloads["demo"] = w
		},
		"binding changed": func(p *model.Policy) {
			w := p.Workloads["demo"]
			w.Bindings[0].Field = "OTHER"
			p.Workloads["demo"] = w
		},
	} {
		t.Run(name, func(t *testing.T) {
			started := make(chan struct{})
			release := make(chan struct{})
			marker := filepath.Join(t.TempDir(), "started")
			app, config, policy := mcpRunTestApplication(t, marker)
			service, err := cqmcp.New(policy, backend{
				app: app,
				resolveGeneration: func(context.Context, model.Policy, string) (model.Generation, error) {
					close(started)
					<-release
					return model.Generation{Workload: "demo", Values: map[string]model.Value{"TOKEN": {Data: "synthetic"}}}, nil
				},
			}, cqmcp.Options{PolicyProvider: app.load, ActionAllowed: app.actionAllowed})
			if err != nil {
				t.Fatal(err)
			}
			done := make(chan error, 1)
			go func() {
				_, runErr := service.RunWorkload(context.Background(), cqmcp.Principal{Subject: "local:alice"}, "demo")
				done <- runErr
			}()
			<-started
			mutate(&policy)
			writeMCPTestPolicy(t, config, policy)
			close(release)
			if err := <-done; !fault.Is(err, fault.Denied) {
				t.Fatalf("revoked action error = %v", err)
			}
			if _, err := os.Stat(marker); !os.IsNotExist(err) {
				t.Fatalf("child process started: marker error = %v", err)
			}
		})
	}
}

func mcpRunTestApplication(t *testing.T, marker string) (application, string, model.Policy) {
	t.Helper()
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	digest, err := executableDigest(executable)
	if err != nil {
		t.Fatal(err)
	}
	dir := privateCLIDirectory(t)
	fixture := filepath.Join(dir, "synthetic.json")
	if err := os.WriteFile(fixture, []byte(`{"TOKEN":"synthetic","OTHER":"synthetic"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	policy := model.Policy{
		Version: 1,
		Sources: map[string]model.Source{"fixture": {Type: "json", Path: fixture}},
		Workloads: map[string]model.Workload{"demo": {
			Mode:          "offline",
			Bindings:      []model.Binding{{Name: "TOKEN", Source: "fixture", Field: "TOKEN", AllowOffline: true}},
			Environment:   map[string]string{"CQTEST_HELPER": "1", "CQTEST_MARKER": marker},
			Command:       []string{executable, "-test.run=^TestMCPRunHelper$"},
			CommandSHA256: digest,
			UntrustedCode: true,
		}},
		Grants: []model.Grant{{Subject: "local:alice", Workload: "demo", Operations: []string{"run", "read"}, Keys: []string{"TOKEN"}}},
	}
	config := filepath.Join(dir, "policy.json")
	writeMCPTestPolicy(t, config, policy)
	return application{config: config}, config, policy
}

func TestMCPRunHelper(t *testing.T) {
	if os.Getenv("CQTEST_HELPER") != "1" {
		return
	}
	if err := os.WriteFile(os.Getenv("CQTEST_MARKER"), []byte("started"), 0o600); err != nil {
		os.Exit(2)
	}
	os.Exit(0)
}

func executableDigest(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256(data)
	return hex.EncodeToString(digest[:]), nil
}

func writeMCPTestPolicy(t *testing.T, path string, policy model.Policy) {
	t.Helper()
	data, err := json.Marshal(policy)
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
}
