package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/maldous/CircularQuay/internal/mcp"
	"github.com/maldous/CircularQuay/internal/model"
	"github.com/maldous/CircularQuay/internal/platform"
	"github.com/maldous/CircularQuay/internal/state"
)

func privateCLIDirectory(t *testing.T) string {
	t.Helper()
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	directory := filepath.Join(root, "cli")
	if err := platform.EnsurePrivateDirectory(directory); err != nil {
		t.Fatal(err)
	}
	return directory
}

func TestDoctorReportsGrantIdentity(t *testing.T) {
	var out, diagnostics bytes.Buffer
	code := (Runner{Out: &out, Err: &diagnostics}).Run(context.Background(), []string{"doctor", "--identity-only"})
	principal, err := mcp.LocalPrincipal()
	if err != nil {
		t.Fatal(err)
	}
	var report struct {
		Subject string `json:"local_subject"`
	}
	if code != 0 || json.Unmarshal(out.Bytes(), &report) != nil || report.Subject != principal.Subject {
		t.Fatalf("doctor did not report kernel grant identity: status %d, subject %q", code, report.Subject)
	}
}

func TestRunReleasesLockBeforeChild(t *testing.T) {
	dir := privateCLIDirectory(t)
	stateDir := filepath.Join(dir, "state")
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	fixture := filepath.Join(dir, "values.json")
	writeProtectedFixture(t, fixture, []byte(`{"message":"synthetic"}`))
	p := model.Policy{Version: 1, Sources: map[string]model.Source{"fixture": {Type: "json", Path: fixture}}, Workloads: map[string]model.Workload{"demo": {Mode: "offline", Bindings: []model.Binding{{Name: "MESSAGE", Source: "fixture", Field: "message", AllowOffline: true}}, Command: []string{executable, "-test.run=^TestLockProbeChild$", "--", "cquay-lock-probe", stateDir}}}}
	data, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	config := filepath.Join(dir, "policy.json")
	writeProtectedFixture(t, config, data)
	var out, diagnostics bytes.Buffer
	code := (Runner{Out: &out, Err: &diagnostics}).Run(context.Background(), []string{"--config", config, "--state-dir", stateDir, "run", "--workload", "demo"})
	if code != 0 {
		t.Fatalf("child could not reconcile while workload runs: status %d, %s", code, diagnostics.String())
	}
}

func TestLockProbeChild(t *testing.T) {
	if len(os.Args) < 3 || os.Args[len(os.Args)-2] != "cquay-lock-probe" {
		return
	}
	s := state.Store{Directory: os.Args[len(os.Args)-1]}
	record, err := s.Read("demo")
	if err != nil || record.Generation == "" {
		os.Exit(41)
	}
	release, err := s.Lock("demo")
	if err != nil {
		os.Exit(42)
	}
	if release() != nil {
		os.Exit(43)
	}
	os.Exit(0)
}

func TestSyncOncePublishesChangedBytesWithIdenticalMetadata(t *testing.T) {
	config, fixture, destination, modified := syncFixture(t)
	for _, value := range []string{"first", "later"} {
		assertSyncedValue(t, config, fixture, destination, modified, value)
	}
}

func syncFixture(t *testing.T) (config, fixture, destination string, modified time.Time) {
	t.Helper()
	dir := privateCLIDirectory(t)
	fixture = filepath.Join(dir, "values.json")
	destination = filepath.Join(dir, "published.json")
	writeProtectedFixture(t, fixture, []byte(`{"message":"first"}`))
	info, err := os.Stat(fixture)
	if err != nil {
		t.Fatal(err)
	}
	p := model.Policy{Version: 1, Sources: map[string]model.Source{"fixture": {Type: "json", Path: fixture}}, Workloads: map[string]model.Workload{"demo": {Mode: "offline", Bindings: []model.Binding{{Name: "MESSAGE", Source: "fixture", Field: "message", AllowOffline: true, AllowPublish: true}}, Targets: map[string]model.Target{"file": {Type: "file", Path: destination, AcceptSnapshot: true}}}}}
	data, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	config = filepath.Join(dir, "policy.json")
	writeProtectedFixture(t, config, data)
	return config, fixture, destination, info.ModTime()
}

func assertSyncedValue(t *testing.T, config, fixture, destination string, modified time.Time, value string) {
	t.Helper()
	writeProtectedFixture(t, fixture, []byte(`{"message":"`+value+`"}`))
	if err := os.Chtimes(fixture, modified, modified); err != nil {
		t.Fatal(err)
	}
	var out, diagnostics bytes.Buffer
	code := (Runner{Out: &out, Err: &diagnostics}).Run(context.Background(), []string{"--config", config, "--state-dir", filepath.Join(filepath.Dir(config), "state"), "sync", "--once", "--workload", "demo"})
	if code != 0 {
		t.Fatalf("sync failed: %s", diagnostics.String())
	}
	published, err := os.ReadFile(destination)
	if err != nil {
		t.Fatal(err)
	}
	var values map[string]string
	if err := json.Unmarshal(published, &values); err != nil {
		t.Fatal(err)
	}
	if values["MESSAGE"] != value {
		t.Fatal("metadata equality suppressed changed source bytes")
	}
	var record state.Record
	if err := json.Unmarshal(out.Bytes(), &record); err != nil {
		t.Fatal(err)
	}
	if record.State != "applied" || record.Health != "unknown" {
		t.Fatal("sync exaggerated activation/health")
	}
}

func writeProtectedFixture(t *testing.T, path string, data []byte) {
	t.Helper()
	if err := platform.WriteProtected(path, data); err != nil {
		t.Fatal(err)
	}
}
