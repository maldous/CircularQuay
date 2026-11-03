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

	"github.com/maldous/CircularQuay/internal/platform"
)

func privateAuditPath(t *testing.T) string {
	t.Helper()
	directory := filepath.Join(t.TempDir(), "audit")
	if err := platform.EnsurePrivateDirectory(directory); err != nil {
		t.Fatal(err)
	}
	return filepath.Join(directory, "mcp-audit.jsonl")
}

func TestProtectedFileAudit(t *testing.T) {
	path := privateAuditPath(t)
	a := newTestAuditor(t, path)
	event := Event{Time: time.Now().UTC(), Subject: "local:alice", Workload: "alpha", Operation: "read", Key: "SERVICE_KEY", Outcome: "released"}
	assertAuditRecord(t, a, path, event)
	assertCancelledAuditRejected(t, a, event)
	assertPrivateAuditPermissions(t, path)
}

func newTestAuditor(t *testing.T, path string) *FileAuditor {
	t.Helper()
	auditor, err := NewFileAuditor(path)
	if err != nil {
		t.Fatal(err)
	}
	return auditor
}

func assertAuditRecord(t *testing.T, auditor *FileAuditor, path string, event Event) {
	t.Helper()
	if err := auditor.Record(context.Background(), event); err != nil {
		t.Fatal(err)
	}
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var actual Event
	if err := json.Unmarshal(contents, &actual); err != nil || actual.Subject != event.Subject || actual.Key != event.Key {
		t.Fatal("receipt lost actor", err)
	}
}

func assertCancelledAuditRejected(t *testing.T, auditor *FileAuditor, event Event) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := auditor.Record(ctx, event); err == nil {
		t.Fatal("cancelled audit recorded")
	}
}

func assertPrivateAuditPermissions(t *testing.T, path string) {
	t.Helper()
	if runtime.GOOS == "windows" {
		return
	}
	info, err := os.Stat(path)
	if err != nil || info.Mode().Perm() != 0600 {
		t.Fatal("audit permissions")
	}
}

func TestAuditRejectsUnsafePaths(t *testing.T) {
	assertUnsafeAuditPath(t, "relative.jsonl", "relative audit path accepted")
	dir := t.TempDir()
	assertUnsafeAuditPath(t, filepath.Join(dir, "missing", "audit.jsonl"), "missing parent accepted")
	assertUnsafeAuditPath(t, dir, "directory accepted")
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
	assertUnsafeAuditPath(t, path, "public audit accepted")
	if err := os.Chmod(path, 0600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "link.jsonl")
	if err := os.Symlink(path, link); err != nil {
		t.Fatal(err)
	}
	assertUnsafeAuditPath(t, link, "symlink accepted")
	if err := os.Chmod(dir, 0777); err != nil {
		t.Fatal(err)
	}
	assertUnsafeAuditPath(t, filepath.Join(dir, "new.jsonl"), "writable parent accepted")
}

func assertUnsafeAuditPath(t *testing.T, path, message string) {
	t.Helper()
	if _, err := NewFileAuditor(path); err == nil {
		t.Fatal(message)
	}
}

func TestAuditFileReplacementAndConcurrency(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows storage needs ACL contract testing")
	}
	path := privateAuditPath(t)
	a := newTestAuditor(t, path)
	done := make(chan error, 20)
	recordAuditEvents(a, done, 20)
	awaitAuditEvents(t, done, 20)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(string(data), "\n") != 20 {
		t.Fatal("interleaved/truncated receipts")
	}
	replaceAuditFileWithDirectory(t, path)
	if err := a.Record(context.Background(), Event{}); err == nil {
		t.Fatal("changed audit destination accepted")
	}
}

func recordAuditEvents(auditor *FileAuditor, done chan<- error, count int) {
	for range count {
		go func() {
			done <- auditor.Record(context.Background(), Event{Time: time.Now().UTC(), Operation: "read", Outcome: "released"})
		}()
	}
}

func awaitAuditEvents(t *testing.T, done <-chan error, count int) {
	t.Helper()
	for range count {
		if err := <-done; err != nil {
			t.Fatal(err)
		}
	}
}

func replaceAuditFileWithDirectory(t *testing.T, path string) {
	t.Helper()
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(path, 0700); err != nil {
		t.Fatal(err)
	}
}
