package main

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/maldous/CircularQuay/internal/platform"
)

func TestInvalidAuditArguments(t *testing.T) {
	for _, args := range [][]string{nil, {"version"}, {"run", "--unknown"}, {"run", "extra"}, {"run", "--profile", "unknown"}} {
		var out, err bytes.Buffer
		if code := run(context.Background(), args, &out, &err); code != 2 || err.Len() == 0 {
			t.Fatal("invalid arguments accepted")
		}
		if strings.Contains(err.String(), "--unknown") {
			t.Fatal("untrusted arguments reflected")
		}
	}
}

func TestCancelledBootstrapDoesNotContactServer(t *testing.T) {
	directory, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := platform.EnsurePrivateDirectory(directory); err != nil {
		t.Fatal(err)
	}
	const marker = "synthetic-audit-password"
	if err := platform.CreateProtectedExclusive(filepath.Join(directory, "admin.password"), []byte(marker)); err != nil {
		t.Fatal(err)
	}
	var contacted atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		contacted.Add(1)
		writer.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	var stdout, stderr bytes.Buffer
	code := run(ctx, []string{"sonar-bootstrap", "--directory", directory, "--url", server.URL}, &stdout, &stderr)
	if code != 2 || contacted.Load() != 0 || stdout.Len() != 0 || strings.Contains(stderr.String(), marker) {
		t.Fatal("cancelled bootstrap contacted its server or disclosed private input")
	}
}

func TestBootstrapTokenValidityFlagIsBootstrapOnlyAndBounded(t *testing.T) {
	for _, arguments := range [][]string{
		{"sonar-init", "--token-validity-days", "30"},
		{"sonar-bootstrap", "--token-validity-days", "6"},
		{"sonar-bootstrap", "--token-validity-days", "31"},
		{"sonar-bootstrap", "--token-validity-days", "not-a-number"},
	} {
		var stdout, stderr bytes.Buffer
		if code := run(context.Background(), arguments, &stdout, &stderr); code != 2 || stdout.Len() != 0 || stderr.Len() == 0 {
			t.Fatal("invalid bootstrap token validity accepted or disclosed output")
		}
	}
}

func TestAuditSignalContextHonorsParentCancellation(t *testing.T) {
	parent, cancelParent := context.WithCancel(context.Background())
	ctx, stop := auditSignalContext(parent)
	defer stop()
	cancelParent()
	select {
	case <-ctx.Done():
	case <-time.After(time.Second):
		t.Fatal("audit context lost its parent cancellation")
	}
}
