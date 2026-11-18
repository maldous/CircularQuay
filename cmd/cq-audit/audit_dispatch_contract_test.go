package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"reflect"
	"testing"

	"github.com/maldous/CircularQuay/internal/assurance"
)

// These controls qualify dispatch and serialization, not a real audit tool,
// source, coverage report or production release.
func TestAuditDispatchContractPassedAndFailedReportsRetainExactPayload(t *testing.T) {
	for _, passed := range []bool{false, true} {
		report := assurance.Report{Version: 1, Profile: "source", Passed: passed, Checks: []assurance.Check{{Name: "synthetic-audit-control", Required: true, Status: "passed"}}}
		var stdout, stderr bytes.Buffer
		code, calls := auditDispatchFixture(t, report, nil, &stdout, &stderr)
		want := 1
		if passed {
			want = 0
		}
		var decoded assurance.Report
		if json.Unmarshal(stdout.Bytes(), &decoded) != nil || !reflect.DeepEqual(decoded, report) || code != want || calls != 1 || stderr.Len() != 0 {
			t.Fatal("audit dispatch changed the exact report or completion status")
		}
	}
}

func TestAuditDispatchContractErrorNeverEncodesPartialReportOrPrivateCause(t *testing.T) {
	var stdout, stderr bytes.Buffer
	partial := assurance.Report{Commit: "private-source-marker", Passed: true}
	code, calls := auditDispatchFixture(t, partial, errors.New("private upstream marker"), &stdout, &stderr)
	if code != 2 || calls != 1 || stdout.Len() != 0 || stderr.String() != "audit could not complete; verify source, output and prerequisites\n" {
		t.Fatal("audit failure leaked partial data or lost its sanitized status")
	}
}

func TestAuditDispatchContractEncoderFailureIsNotSuccess(t *testing.T) {
	var stderr bytes.Buffer
	code, calls := auditDispatchFixture(t, assurance.Report{Passed: true}, nil, auditRunBoundaryWriter{}, &stderr)
	if code != 2 || calls != 1 || stderr.Len() != 0 {
		t.Fatal("failed report encoding became successful dispatch")
	}
}

func auditDispatchFixture(t *testing.T, report assurance.Report, failure error, stdout, stderr io.Writer) (int, int) {
	t.Helper()
	ctx := context.Background()
	want := assurance.Options{Source: "owned-source", Output: "owned-output", Profile: "source", ToolManifest: "owned-manifest", SonarProjectKey: "cq"}
	calls := 0
	audit := func(gotContext context.Context, options assurance.Options) (assurance.Report, error) {
		calls++
		if gotContext != ctx || !reflect.DeepEqual(options, want) {
			t.Fatal("audit dispatch changed explicit parsed inputs or context")
		}
		return report, failure
	}
	arguments := []string{"run", "--source", want.Source, "--output", want.Output, "--profile", want.Profile, "--tool-manifest", want.ToolManifest}
	code := runWithAudit(ctx, arguments, stdout, stderr, audit)
	return code, calls
}
