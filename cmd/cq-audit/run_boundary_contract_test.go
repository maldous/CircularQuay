package main

import (
	"bytes"
	"context"
	"errors"
	"io"
	"path/filepath"
	"testing"

	"github.com/maldous/CircularQuay/internal/platform"
)

func TestRunBoundaryContractMissingManifestIsSanitized(t *testing.T) {
	root := auditRunBoundaryRoot(t)
	var stdout, stderr bytes.Buffer
	code := run(context.Background(), []string{"run", "--source", root, "--output", filepath.Join(root, "audit-results"), "--tool-manifest", filepath.Join(root, "missing-manifest"), "--profile", "source"}, &stdout, &stderr)
	if code != 1 || stderr.Len() != 0 || !bytes.Contains(stdout.Bytes(), []byte(`"tool_manifest_invalid"`)) {
		t.Fatal("missing manifest lost its serialized failed-report outcome")
	}
}

func TestRunBoundaryContractReportEncoderFailureDoesNotBecomePass(t *testing.T) {
	root := auditRunBoundaryRoot(t)
	var stderr bytes.Buffer
	writer := auditRunBoundaryWriter{}
	code := run(context.Background(), []string{"run", "--source", root, "--output", filepath.Join(root, "audit-results"), "--tool-manifest", filepath.Join(root, "missing-manifest"), "--profile", "source"}, writer, &stderr)
	if code != 2 {
		t.Fatal("failed report output was promoted to a successful audit command")
	}
}

type auditRunBoundaryWriter struct{}

func (auditRunBoundaryWriter) Write([]byte) (int, error) {
	return 0, errors.New("synthetic output refusal")
}

func auditRunBoundaryRoot(t *testing.T) string {
	t.Helper()
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := platform.EnsurePrivateDirectory(root); err != nil {
		t.Fatal(err)
	}
	return root
}

func TestRunBoundaryContractInitCreatesOnlyPrivateLocalState(t *testing.T) {
	root := filepath.Join(auditRunBoundaryRoot(t), "sonar")
	var stdout, stderr bytes.Buffer
	code := runSonar(context.Background(), []string{"sonar-init", "--directory", root}, &stdout, &stderr)
	if code != 0 || stderr.Len() != 0 || stdout.String() != "local Sonar bootstrap step completed; credential values are stored only in private state\n" {
		t.Fatal("local initialization lost its sanitized completion contract")
	}
	file, err := platform.OpenProtected(filepath.Join(root, "admin.password"), true)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	if _, err := io.Copy(io.Discard, file); err != nil {
		t.Fatal("initialized private state was not readable through the protected boundary")
	}
}
