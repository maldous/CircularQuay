package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/maldous/CircularQuay/internal/mcp"
	"github.com/maldous/CircularQuay/internal/model"
	"github.com/maldous/CircularQuay/internal/state"
)

func TestDoctorReportsGrantIdentity(t *testing.T) {
	var out, diagnostics bytes.Buffer
	code := (Runner{Out: &out, Err: &diagnostics}).Run(context.Background(), []string{"doctor"})
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
	dir := t.TempDir()
	stateDir := filepath.Join(dir, "state")
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	fixture := filepath.Join(dir, "values.json")
	if err := os.WriteFile(fixture, []byte(`{"message":"synthetic"}`), 0600); err != nil {
		t.Fatal(err)
	}
	p := model.Policy{Version: 1, Sources: map[string]model.Source{"fixture": {Type: "json", Path: fixture}}, Workloads: map[string]model.Workload{"demo": {Mode: "offline", Bindings: []model.Binding{{Name: "MESSAGE", Source: "fixture", Field: "message", AllowOffline: true}}, Command: []string{executable, "-test.run=^TestLockProbeChild$", "--", "cquay-lock-probe", stateDir}}}}
	data, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	config := filepath.Join(dir, "policy.json")
	if err := os.WriteFile(config, data, 0600); err != nil {
		t.Fatal(err)
	}
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
	dir := t.TempDir()
	if err := os.Chmod(dir, 0700); err != nil {
		t.Fatal(err)
	}
	fixture := filepath.Join(dir, "values.json")
	destination := filepath.Join(dir, "published.json")
	if err := os.WriteFile(fixture, []byte(`{"message":"first"}`), 0600); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(fixture)
	if err != nil {
		t.Fatal(err)
	}
	p := model.Policy{Version: 1, Sources: map[string]model.Source{"fixture": {Type: "json", Path: fixture}}, Workloads: map[string]model.Workload{"demo": {Mode: "offline", Bindings: []model.Binding{{Name: "MESSAGE", Source: "fixture", Field: "message", AllowOffline: true, AllowPublish: true}}, Targets: map[string]model.Target{"file": {Type: "file", Path: destination, AcceptSnapshot: true}}}}}
	data, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	config := filepath.Join(dir, "policy.json")
	if err = os.WriteFile(config, data, 0600); err != nil {
		t.Fatal(err)
	}
	for _, value := range []string{"first", "later"} {
		if err = os.WriteFile(fixture, []byte(`{"message":"`+value+`"}`), 0600); err != nil {
			t.Fatal(err)
		}
		if err = os.Chtimes(fixture, info.ModTime(), info.ModTime()); err != nil {
			t.Fatal(err)
		}
		var out, diagnostics bytes.Buffer
		code := (Runner{Out: &out, Err: &diagnostics}).Run(context.Background(), []string{"--config", config, "--state-dir", filepath.Join(dir, "state"), "sync", "--once", "--workload", "demo"})
		if code != 0 {
			t.Fatalf("sync failed: %s", diagnostics.String())
		}
		published, err := os.ReadFile(destination)
		if err != nil {
			t.Fatal(err)
		}
		var values map[string]string
		if json.Unmarshal(published, &values) != nil || values["MESSAGE"] != value {
			t.Fatal("metadata equality suppressed changed source bytes")
		}
		var record state.Record
		if json.Unmarshal(out.Bytes(), &record) != nil || record.State != "applied" || record.Health != "unknown" {
			t.Fatal("sync exaggerated activation/health")
		}
	}
}
