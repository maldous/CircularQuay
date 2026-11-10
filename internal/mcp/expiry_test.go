package mcp

import (
	"context"
	"testing"
	"time"

	"github.com/maldous/CircularQuay/internal/fault"
	"github.com/maldous/CircularQuay/internal/model"
)

func TestPrincipalValidAtSeparatesHTTPAndLocalLifetimes(t *testing.T) {
	now := time.Date(2026, time.November, 4, 0, 0, 0, 0, time.UTC)
	local := Principal{Subject: "local:alice"}
	if !local.ValidAt(time.Time{}) {
		t.Fatal("local principal unexpectedly has a deadline")
	}
	for _, principal := range []Principal{
		{HTTP: true},
		{HTTP: true, ExpiresAt: now},
		{HTTP: true, ExpiresAt: now.Add(time.Second)},
	} {
		want := !principal.ExpiresAt.IsZero() && now.Before(principal.ExpiresAt)
		if got := principal.ValidAt(now); got != want {
			t.Fatalf("HTTP principal validity = %t, want %t", got, want)
		}
	}
}

func TestHTTPPrincipalExpiryFencesDisclosure(t *testing.T) {
	now := time.Date(2026, time.November, 4, 0, 0, 0, 0, time.UTC)
	expires := now.Add(time.Minute)
	principal := Principal{Subject: "local:alice", HTTP: true, Scopes: []string{"cq:read"}, ExpiresAt: expires}
	backend := &fakeBackend{value: model.Value{Data: "MUST-NOT-DISCLOSE"}, onRead: func() { now = expires }}
	audit := &memoryAudit{}
	service := makeService(t, testPolicy(t), backend, Options{Audit: audit})
	service.now = func() time.Time { return now }
	result, err := service.ReadSecret(context.Background(), principal, "alpha", "SERVICE_KEY")
	if !fault.Is(err, fault.Denied) || result.Value != "" || len(audit.events) != 0 || backend.reads != 1 {
		t.Fatalf("expired disclosure result = %#v, %v, reads=%d, audits=%d", result, err, backend.reads, len(audit.events))
	}
}

func TestHTTPPrincipalExpiryFencesDisclosureAfterAudit(t *testing.T) {
	now := time.Date(2026, time.November, 4, 0, 0, 0, 0, time.UTC)
	expires := now.Add(time.Minute)
	principal := Principal{Subject: "local:alice", HTTP: true, Scopes: []string{"cq:read"}, ExpiresAt: expires}
	backend := &fakeBackend{value: model.Value{Data: "MUST-NOT-DISCLOSE"}}
	audit := &memoryAudit{onRecord: func() { now = expires }}
	service := makeService(t, testPolicy(t), backend, Options{Audit: audit})
	service.now = func() time.Time { return now }
	result, err := service.ReadSecret(context.Background(), principal, "alpha", "SERVICE_KEY")
	if !fault.Is(err, fault.Denied) || result.Value != "" || len(audit.events) != 1 || backend.reads != 1 {
		t.Fatalf("post-audit expiry result = %#v, %v, reads=%d, audits=%d", result, err, backend.reads, len(audit.events))
	}
}

func TestExpiredHTTPPrincipalDoesNotReachProvider(t *testing.T) {
	now := time.Date(2026, time.November, 4, 0, 0, 0, 0, time.UTC)
	backend := &fakeBackend{value: model.Value{Data: "MUST-NOT-READ"}}
	service := makeService(t, testPolicy(t), backend, Options{Audit: &memoryAudit{}})
	service.now = func() time.Time { return now }
	principal := Principal{Subject: "local:alice", HTTP: true, Scopes: []string{"cq:read"}, ExpiresAt: now}
	if _, err := service.ReadSecret(context.Background(), principal, "alpha", "SERVICE_KEY"); !fault.Is(err, fault.Denied) || backend.reads != 0 {
		t.Fatalf("expired principal result = %v, reads=%d", err, backend.reads)
	}
}

func TestHTTPPrincipalExpiryFencesAction(t *testing.T) {
	now := time.Date(2026, time.November, 4, 0, 0, 0, 0, time.UTC)
	expires := now.Add(time.Minute)
	backend := &fakeBackend{run: RunResult{ID: "MUST-NOT-RUN"}}
	audit := &memoryAudit{onRecord: func() { now = expires }}
	service := makeService(t, testPolicy(t), backend, Options{Audit: audit, ActionAllowed: func(string) bool { return true }})
	service.now = func() time.Time { return now }
	principal := Principal{Subject: "local:alice", HTTP: true, Scopes: []string{"cq:run"}, ExpiresAt: expires}
	if _, err := service.RunWorkload(context.Background(), principal, "alpha"); !fault.Is(err, fault.Denied) || backend.runs != 0 {
		t.Fatalf("expired action result = %v, runs=%d", err, backend.runs)
	}
}

func TestHTTPPrincipalExpiryFencesActionAfterTrustCheck(t *testing.T) {
	now := time.Date(2026, time.November, 4, 0, 0, 0, 0, time.UTC)
	expires := now.Add(time.Minute)
	checks := 0
	backend := &fakeBackend{run: RunResult{ID: "MUST-NOT-RUN"}}
	service := makeService(t, testPolicy(t), backend, Options{Audit: &memoryAudit{}, ActionAllowed: func(string) bool {
		checks++
		if checks == 2 {
			now = expires
		}
		return true
	}})
	service.now = func() time.Time { return now }
	principal := Principal{Subject: "local:alice", HTTP: true, Scopes: []string{"cq:run"}, ExpiresAt: expires}
	if _, err := service.RunWorkload(context.Background(), principal, "alpha"); !fault.Is(err, fault.Denied) || backend.runs != 0 || checks != 2 {
		t.Fatalf("post-trust expiry result = %v, runs=%d, checks=%d", err, backend.runs, checks)
	}
}

func TestRunWorkloadChecksContextAfterAudit(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	backend := &fakeBackend{run: RunResult{ID: "MUST-NOT-RUN"}}
	service := makeService(t, testPolicy(t), backend, Options{Audit: &memoryAudit{onRecord: cancel}, ActionAllowed: func(string) bool { return true }})
	if _, err := service.RunWorkload(ctx, Principal{Subject: "local:alice"}, "alpha"); !fault.Is(err, fault.Unavailable) || backend.runs != 0 {
		t.Fatalf("post-audit context result = %v, runs=%d", err, backend.runs)
	}
}

func TestHTTPPrincipalExpiryFencesStatus(t *testing.T) {
	now := time.Date(2026, time.November, 4, 0, 0, 0, 0, time.UTC)
	expires := now.Add(time.Minute)
	backend := &fakeBackend{status: Status{State: "ready"}, onStatus: func() { now = expires }}
	service := makeService(t, testPolicy(t), backend, Options{})
	service.now = func() time.Time { return now }
	principal := Principal{Subject: "local:alice", HTTP: true, Scopes: []string{"cq:status"}, ExpiresAt: expires}
	if _, err := service.GetStatus(context.Background(), principal, "alpha"); !fault.Is(err, fault.Denied) || backend.statuses != 1 {
		t.Fatalf("expired status result = %v, statuses=%d", err, backend.statuses)
	}
}
