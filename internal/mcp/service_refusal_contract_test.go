package mcp

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/maldous/CircularQuay/internal/fault"
	"github.com/maldous/CircularQuay/internal/model"
)

func TestMCPServiceRefusalContractMissingRequiredAuditPreventsAction(t *testing.T) {
	backend := &fakeBackend{run: RunResult{ID: "synthetic-run", Status: "completed"}}
	service := makeService(t, mcpServiceRefusalAuditedPolicy(t), backend, Options{
		ActionAllowed: func(workload string) bool { return workload == "alpha" },
	})
	result, err := service.RunWorkload(context.Background(), Principal{Subject: "local:alice"}, "alpha")
	assertMCPServiceRefusalAuditFailure(t, result, err)
	assertMCPServiceRefusalNoBackendCalls(t, backend)
}

func TestMCPServiceRefusalContractFailedRequiredAuditPreventsAction(t *testing.T) {
	backend := &fakeBackend{run: RunResult{ID: "synthetic-run", Status: "completed"}}
	audit := &mcpServiceRefusalAuditor{err: errors.New("SYNTHETIC_PRIVATE_AUDIT_DETAIL")}
	service := makeService(t, mcpServiceRefusalAuditedPolicy(t), backend, Options{
		Audit:         audit,
		ActionAllowed: func(workload string) bool { return workload == "alpha" },
	})
	result, err := service.RunWorkload(context.Background(), Principal{Subject: "local:alice"}, "alpha")
	assertMCPServiceRefusalAuditFailure(t, result, err)
	assertMCPServiceRefusalNoBackendCalls(t, backend)
	assertMCPServiceRefusalAuditEvent(t, audit, "run", "", "authorized")
}

func TestMCPServiceRefusalContractRecordedRequiredAuditAllowsSyntheticAction(t *testing.T) {
	want := RunResult{ID: "synthetic-run", Status: "completed", ExitCode: 0}
	backend := &fakeBackend{run: want}
	audit := &mcpServiceRefusalAuditor{}
	service := makeService(t, mcpServiceRefusalAuditedPolicy(t), backend, Options{
		Audit:         audit,
		ActionAllowed: func(workload string) bool { return workload == "alpha" },
	})
	result, err := service.RunWorkload(context.Background(), Principal{Subject: "local:alice"}, "alpha")
	if err != nil || result != want {
		t.Fatalf("admitted synthetic action result=%+v err=%v", result, err)
	}
	if backend.runs != 1 || backend.reads != 0 || backend.statuses != 0 {
		t.Fatalf("unexpected backend calls: runs=%d reads=%d statuses=%d", backend.runs, backend.reads, backend.statuses)
	}
	assertMCPServiceRefusalAuditEvent(t, audit, "run", "", "authorized")
}

func TestMCPServiceRefusalContractFiniteExpiryReachesOnlyAuditedReadResult(t *testing.T) {
	expires := time.Now().UTC().Add(time.Hour)
	backend := &fakeBackend{value: model.Value{Data: "SYNTHETIC_EXPIRING_VALUE", Version: "synthetic-v1", ExpiresAt: expires}}
	audit := &mcpServiceRefusalAuditor{}
	service := makeService(t, testPolicy(t), backend, Options{Audit: audit})
	result, err := service.ReadSecret(context.Background(), Principal{Subject: "local:alice"}, "alpha", "SERVICE_KEY")
	want := SecretResult{Alias: "SERVICE_KEY", Value: "SYNTHETIC_EXPIRING_VALUE", Version: "synthetic-v1", ExpiresAt: &expires}
	if err != nil || !reflect.DeepEqual(result, want) {
		t.Fatalf("finite expiry result=%+v err=%v", result, err)
	}
	if backend.reads != 1 || backend.runs != 0 || backend.statuses != 0 {
		t.Fatalf("unexpected backend calls: reads=%d runs=%d statuses=%d", backend.reads, backend.runs, backend.statuses)
	}
	assertMCPServiceRefusalAuditEvent(t, audit, "read", "SERVICE_KEY", "release_authorized")
}

func TestMCPServiceRefusalContractClockFallbackAndInjectedClockRemainBounded(t *testing.T) {
	service := &Service{}
	before := time.Now()
	got := service.clock()
	after := time.Now()
	if got.IsZero() || got.Before(before) || got.After(after) {
		t.Fatalf("fallback clock escaped its observation bounds: before=%v got=%v after=%v", before, got, after)
	}
	want := time.Date(2030, time.January, 2, 3, 4, 5, 0, time.UTC)
	service.now = func() time.Time { return want }
	if got := service.clock(); !got.Equal(want) {
		t.Fatalf("injected clock ignored: got=%v want=%v", got, want)
	}
}

func mcpServiceRefusalAuditedPolicy(t *testing.T) model.Policy {
	t.Helper()
	config := testPolicy(t)
	// This is only a declared capability; all source/backend work stays synthetic.
	source := config.Sources["fixture"]
	source.AuditVerified = true
	config.Sources["fixture"] = source
	workload := config.Workloads["alpha"]
	workload.Mode = "hybrid"
	workload.AuditRequired = true
	config.Workloads["alpha"] = workload
	return config
}

type mcpServiceRefusalAuditor struct {
	calls int
	event Event
	err   error
}

func (a *mcpServiceRefusalAuditor) Record(_ context.Context, event Event) error {
	a.calls++
	a.event = event
	return a.err
}

func assertMCPServiceRefusalAuditFailure(t *testing.T, result RunResult, err error) {
	t.Helper()
	if result != (RunResult{}) {
		t.Fatalf("refused action returned a partial result: %+v", result)
	}
	var typed *fault.Error
	if !errors.As(err, &typed) {
		t.Fatalf("missing typed audit refusal: %v", err)
	}
	if err != typed || typed.Kind != fault.Apply || typed.Operation != "mcp audit" || err.Error() != "activation_failed: mcp audit" {
		t.Fatalf("audit refusal changed category or leaked upstream detail: %v", err)
	}
}

func assertMCPServiceRefusalNoBackendCalls(t *testing.T, backend *fakeBackend) {
	t.Helper()
	if backend.runs != 0 || backend.reads != 0 || backend.statuses != 0 {
		t.Fatalf("audit refusal reached backend: runs=%d reads=%d statuses=%d", backend.runs, backend.reads, backend.statuses)
	}
}

func assertMCPServiceRefusalAuditEvent(t *testing.T, audit *mcpServiceRefusalAuditor, operation, key, outcome string) {
	t.Helper()
	if audit.calls != 1 {
		t.Fatalf("unexpected audit attempts: %d", audit.calls)
	}
	got := audit.event
	if got.Time.IsZero() || got.Time.Location() != time.UTC {
		t.Fatal("audit attempt omitted its UTC timestamp")
	}
	got.Time = time.Time{}
	want := Event{Subject: "local:alice", Workload: "alpha", Operation: operation, Key: key, Outcome: outcome}
	if got != want {
		t.Fatalf("audit metadata changed: got=%+v want=%+v", got, want)
	}
}
