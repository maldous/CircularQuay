package mcp

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestProtectedFileAudit(t *testing.T) {
	path := filepath.Join(t.TempDir(), "mcp-audit.jsonl")
	a, err := NewFileAuditor(path)
	if err != nil {
		t.Fatal(err)
	}
	event := Event{Time: time.Now().UTC(), Subject: "local:alice", Workload: "alpha", Operation: "read", Key: "SERVICE_KEY", Outcome: "released"}
	if err := a.Record(context.Background(), event); err != nil {
		t.Fatal(err)
	}
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var actual Event
	if err := json.Unmarshal(contents, &actual); err != nil {
		t.Fatal(err)
	}
	if actual.Subject != event.Subject || actual.Key != event.Key {
		t.Fatal("receipt lost actor")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := a.Record(ctx, event); err == nil {
		t.Fatal("cancelled audit recorded")
	}
	if runtime.GOOS != "windows" {
		info, err := os.Stat(path)
		if err != nil || info.Mode().Perm() != 0600 {
			t.Fatal("audit permissions")
		}
	}
}

func TestAuditRejectsUnsafePaths(t *testing.T) {
	if _, err := NewFileAuditor("relative.jsonl"); err == nil {
		t.Fatal("relative audit path accepted")
	}
	dir := t.TempDir()
	if _, err := NewFileAuditor(filepath.Join(dir, "missing", "audit.jsonl")); err == nil {
		t.Fatal("missing parent accepted")
	}
	if _, err := NewFileAuditor(dir); err == nil {
		t.Fatal("directory accepted")
	}
	if runtime.GOOS == "windows" {
		t.Skip("POSIX mode and symlink tests require native Windows ACL qualification")
	}
	path := filepath.Join(dir, "open.jsonl")
	if err := os.WriteFile(path, []byte(""), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := NewFileAuditor(path); err == nil {
		t.Fatal("public audit accepted")
	}
	if err := os.Chmod(path, 0600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "link.jsonl")
	if err := os.Symlink(path, link); err != nil {
		t.Fatal(err)
	}
	if _, err := NewFileAuditor(link); err == nil {
		t.Fatal("symlink accepted")
	}
	if err := os.Chmod(dir, 0777); err != nil {
		t.Fatal(err)
	}
	if _, err := NewFileAuditor(filepath.Join(dir, "new.jsonl")); err == nil {
		t.Fatal("writable parent accepted")
	}
}

func TestAuditFileReplacementAndConcurrency(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows storage needs ACL contract testing")
	}
	path := filepath.Join(t.TempDir(), "audit.jsonl")
	a, err := NewFileAuditor(path)
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 20)
	for range 20 {
		go func() {
			done <- a.Record(context.Background(), Event{Time: time.Now().UTC(), Operation: "read", Outcome: "released"})
		}()
	}
	for range 20 {
		if err := <-done; err != nil {
			t.Fatal(err)
		}
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(string(data), "\n") != 20 {
		t.Fatal("interleaved/truncated receipts")
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(path, 0700); err != nil {
		t.Fatal(err)
	}
	if err := a.Record(context.Background(), Event{}); err == nil {
		t.Fatal("changed audit destination accepted")
	}
}
