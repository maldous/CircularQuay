package resolve

import (
	"context"
	"encoding/json"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/maldous/CircularQuay/internal/fault"
	"github.com/maldous/CircularQuay/internal/model"
	"github.com/maldous/CircularQuay/internal/source"
	"github.com/maldous/CircularQuay/internal/state"
)

type memorySnapshot struct {
	saved                 map[string]string
	encryptErr, errorRead error
}

func (m *memorySnapshot) Encrypt(_ context.Context, _ model.Snapshot, v map[string]string) error {
	if m.encryptErr != nil {
		return m.encryptErr
	}
	m.saved = v
	return nil
}
func (m *memorySnapshot) Decrypt(context.Context, model.Snapshot) (map[string]string, error) {
	return m.saved, m.errorRead
}

func fixture(t *testing.T) (*Engine, *memorySnapshot) {
	t.Helper()
	now := time.Date(2026, 11, 3, 12, 0, 0, 0, time.UTC)
	w := model.Workload{Mode: "hybrid", Bindings: []model.Binding{{Name: "TOKEN", Source: "a", Resource: "a", AllowOffline: true}, {Name: "OTHER", Source: "b", Resource: "b", AllowOffline: true}}, Snapshot: model.Snapshot{Path: filepath.Join(t.TempDir(), "snapshot"), Recipients: []string{"age1fixture"}}}
	p := model.Policy{Version: 1, Workloads: map[string]model.Workload{"demo": w}, Sources: map[string]model.Source{"a": {Type: "fake"}, "b": {Type: "fake"}}}
	m := &memorySnapshot{}
	e := &Engine{Policy: p, Reader: source.ReadFunc(func(_ context.Context, _ model.Source, b model.Binding) (model.Value, error) {
		return model.Value{Data: "synthetic-only-" + b.Name, Version: "v1"}, nil
	}), Snapshots: m, State: state.Store{Directory: filepath.Join(t.TempDir(), "state")}, Now: func() time.Time { return now }}
	return e, m
}

func TestCompleteGenerationAndScopedSnapshot(t *testing.T) {
	e, m := fixture(t)
	ctx := context.Background()
	g, err := e.Resolve(ctx, "demo")
	if err != nil || len(g.Values) != 2 || g.Offline {
		t.Fatalf("resolution failed: %v", err)
	}
	if err = e.Snapshot(ctx, g); err != nil {
		t.Fatal(err)
	}
	e.Reader = source.ReadFunc(func(context.Context, model.Source, model.Binding) (model.Value, error) {
		return model.Value{}, fault.New(fault.Unavailable, "fixture")
	})
	cached, err := e.Resolve(ctx, "demo")
	if err != nil || !cached.Offline || len(cached.Values) != 2 {
		t.Fatalf("eligible fallback failed: %v", err)
	}
	w := e.Policy.Workloads["demo"]
	w.Bindings[0].Resource = "different-account"
	e.Policy.Workloads["demo"] = w
	if _, err = e.Resolve(ctx, "demo"); !fault.Is(err, fault.Trust) {
		t.Fatalf("cross-scope cache accepted: %v", err)
	}
	if len(m.saved) != 1 {
		t.Fatal("unexpected unencrypted cache metadata")
	}
}

func TestPartialReadNeverActivatesAndDenialFences(t *testing.T) {
	e, _ := fixture(t)
	ctx := context.Background()
	g, _ := e.Resolve(ctx, "demo")
	if err := e.Snapshot(ctx, g); err != nil {
		t.Fatal(err)
	}
	calls := 0
	e.Reader = source.ReadFunc(func(context.Context, model.Source, model.Binding) (model.Value, error) {
		calls++
		if calls == 1 {
			return model.Value{Data: "new"}, nil
		}
		return model.Value{}, fault.New(fault.Denied, "fixture")
	})
	partial, err := e.Resolve(ctx, "demo")
	if !fault.Is(err, fault.Denied) || len(partial.Values) != 0 {
		t.Fatal("partial or denied generation escaped")
	}
	e.Reader = source.ReadFunc(func(context.Context, model.Source, model.Binding) (model.Value, error) {
		return model.Value{}, fault.New(fault.Unavailable, "fixture")
	})
	if _, err = e.Resolve(ctx, "demo"); !fault.Is(err, fault.Denied) {
		t.Fatal("observed denial did not fence cache")
	}
}

func TestSnapshotEligibilityAndLifecycle(t *testing.T) {
	for _, test := range []struct {
		name   string
		change func(*model.Workload, *model.Generation)
	}{
		{"online", func(w *model.Workload, _ *model.Generation) { w.Mode = "online" }},
		{"no_path", func(w *model.Workload, _ *model.Generation) { w.Snapshot.Path = "" }},
		{"export_denied", func(w *model.Workload, _ *model.Generation) { w.Bindings[0].AllowOffline = false }},
		{"missing", func(_ *model.Workload, g *model.Generation) { delete(g.Values, "TOKEN") }},
		{"leased", func(_ *model.Workload, g *model.Generation) {
			v := g.Values["TOKEN"]
			v.Leased = true
			g.Values["TOKEN"] = v
		}},
		{"renewable", func(_ *model.Workload, g *model.Generation) {
			v := g.Values["TOKEN"]
			v.Renewable = true
			g.Values["TOKEN"] = v
		}},
		{"expired", func(_ *model.Workload, g *model.Generation) {
			v := g.Values["TOKEN"]
			v.ExpiresAt = time.Time{}.Add(time.Second)
			g.Values["TOKEN"] = v
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			e, _ := fixture(t)
			g, _ := e.Resolve(context.Background(), "demo")
			w := e.Policy.Workloads["demo"]
			test.change(&w, &g)
			e.Policy.Workloads["demo"] = w
			if e.Snapshot(context.Background(), g) == nil {
				t.Fatal("ineligible snapshot accepted")
			}
		})
	}
}

func TestModesAndMissingCapabilities(t *testing.T) {
	e, m := fixture(t)
	ctx := context.Background()
	if _, err := e.Resolve(ctx, "unknown"); !fault.Is(err, fault.Invalid) {
		t.Fatal("unknown workload")
	}
	r := e.Reader
	e.Reader = nil
	if _, err := e.Resolve(ctx, "demo"); !fault.Is(err, fault.Unsupported) {
		t.Fatal("missing reader")
	}
	e.Reader = r
	g, _ := e.Resolve(ctx, "demo")
	if err := e.Snapshot(ctx, g); err != nil {
		t.Fatal(err)
	}
	w := e.Policy.Workloads["demo"]
	w.Mode = "offline"
	e.Policy.Workloads["demo"] = w
	if _, err := e.Resolve(ctx, "demo"); !fault.Is(err, fault.Trust) {
		t.Fatal("mode change failed to invalidate snapshot")
	}
	g, _ = e.ResolveFresh(ctx, "demo")
	if err := e.Snapshot(ctx, g); err != nil {
		t.Fatal(err)
	}
	if _, err := e.ReadKey(ctx, "demo", "TOKEN"); err != nil {
		t.Fatal(err)
	}
	if _, err := e.ReadKey(ctx, "demo", "unknown"); !fault.Is(err, fault.Missing) {
		t.Fatal("unknown key")
	}
	e.Snapshots = nil
	if _, err := e.Resolve(ctx, "demo"); !fault.Is(err, fault.Unsupported) {
		t.Fatal("missing cache backend")
	}
	e.Snapshots = m
	m.errorRead = errors.New("synthetic read failure")
	if _, err := e.Resolve(ctx, "demo"); err == nil {
		t.Fatal("cache read failure ignored")
	}
}

func TestFreshReadAndValueChecks(t *testing.T) {
	e, _ := fixture(t)
	ctx := context.Background()
	if _, err := e.ReadKey(ctx, "unknown", "TOKEN"); !fault.Is(err, fault.Invalid) {
		t.Fatal("unknown workload")
	}
	if _, err := e.ReadKey(ctx, "demo", "absent"); !fault.Is(err, fault.Missing) {
		t.Fatal("unknown field")
	}
	for _, v := range []model.Value{{Data: "bad\x00value"}, {Data: "old", ExpiresAt: e.now()}, {Data: "leased", Leased: true}, {Data: "renewable", Renewable: true}} {
		e.Reader = source.ReadFunc(func(context.Context, model.Source, model.Binding) (model.Value, error) { return v, nil })
		if _, err := e.ReadKey(ctx, "demo", "TOKEN"); err == nil {
			t.Fatal("unsafe value accepted")
		}
	}
	e.Reader = source.ReadFunc(func(context.Context, model.Source, model.Binding) (model.Value, error) {
		return model.Value{}, fault.New(fault.Denied, "fixture")
	})
	if _, err := e.ReadKey(ctx, "demo", "TOKEN"); !fault.Is(err, fault.Denied) {
		t.Fatal("raw read denial")
	}
	w := e.Policy.Workloads["demo"]
	w.AuditRequired = true
	e.Policy.Workloads["demo"] = w
	e.Reader = source.ReadFunc(func(context.Context, model.Source, model.Binding) (model.Value, error) {
		return model.Value{Data: "fixture"}, nil
	})
	if _, err := e.Resolve(ctx, "demo"); !fault.Is(err, fault.Unsupported) {
		t.Fatal("audit capability not enforced")
	}
	e.Reader = source.ReadFunc(func(context.Context, model.Source, model.Binding) (model.Value, error) {
		return model.Value{Data: "fixture", AuditVerified: true}, nil
	})
	if _, err := e.ResolveFresh(ctx, "demo"); err != nil {
		t.Fatal(err)
	}
}

func TestReadKeyRejectsMissingReaderAndSource(t *testing.T) {
	e, _ := fixture(t)
	e.Reader = nil
	if _, err := e.ReadKey(context.Background(), "demo", "TOKEN"); !fault.Is(err, fault.Unsupported) {
		t.Fatalf("missing reader: %v", err)
	}

	e, _ = fixture(t)
	policy := e.Policy
	policy.Sources = map[string]model.Source{}
	e.Policy = policy
	if _, err := e.ReadKey(context.Background(), "demo", "TOKEN"); !fault.Is(err, fault.Invalid) {
		t.Fatalf("missing binding source: %v", err)
	}
}

func TestMalformedSnapshot(t *testing.T) {
	e, m := fixture(t)
	ctx := context.Background()
	for _, data := range []string{"", `{"version":2}`, `{"version":1,"workload":"other"}`} {
		m.saved = map[string]string{"circularquay_snapshot": data}
		if _, err := e.restore(ctx, "demo", e.Policy.Workloads["demo"]); err == nil {
			t.Fatal("invalid snapshot accepted")
		}
	}
	g, _ := e.Resolve(ctx, "demo")
	if err := e.Snapshot(ctx, g); err != nil {
		t.Fatal(err)
	}
	var p snapshotPayload
	if err := json.Unmarshal([]byte(m.saved["circularquay_snapshot"]), &p); err != nil {
		t.Fatal(err)
	}
	v := p.Generation.Values["TOKEN"]
	v.Leased = true
	p.Generation.Values["TOKEN"] = v
	b, _ := json.Marshal(p)
	m.saved["circularquay_snapshot"] = string(b)
	if _, err := e.restore(ctx, "demo", e.Policy.Workloads["demo"]); !fault.Is(err, fault.Denied) {
		t.Fatal("leased cache accepted")
	}
}

func TestSnapshotRejectsMismatchedPolicyFingerprint(t *testing.T) {
	e, snapshots := fixture(t)
	ctx := context.Background()
	generation, err := e.Resolve(ctx, "demo")
	if err != nil {
		t.Fatal(err)
	}
	if err = e.Snapshot(ctx, generation); err != nil {
		t.Fatal(err)
	}

	var payload snapshotPayload
	if err = json.Unmarshal([]byte(snapshots.saved["circularquay_snapshot"]), &payload); err != nil {
		t.Fatal(err)
	}
	payload.Policy = "different-valid-policy-fingerprint"
	encoded, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	snapshots.saved["circularquay_snapshot"] = string(encoded)

	if _, err = e.restore(ctx, "demo", e.Policy.Workloads["demo"]); !fault.Is(err, fault.Trust) {
		t.Fatalf("mismatched policy fingerprint accepted: %v", err)
	}
}

func TestFallbackRequiresAuditCompatibility(t *testing.T) {
	e, _ := fixture(t)
	ctx := context.Background()
	g, _ := e.Resolve(ctx, "demo")
	if err := e.Snapshot(ctx, g); err != nil {
		t.Fatal(err)
	}
	w := e.Policy.Workloads["demo"]
	w.AuditRequired = true
	e.Policy.Workloads["demo"] = w
	e.Reader = source.ReadFunc(func(context.Context, model.Source, model.Binding) (model.Value, error) {
		return model.Value{}, fault.New(fault.Unavailable, "fixture")
	})
	if _, err := e.Resolve(ctx, "demo"); !fault.Is(err, fault.Unavailable) {
		t.Fatal("required audit bypassed")
	}
}

func TestSnapshotRejectsAmbiguityAndExtraScope(t *testing.T) {
	for _, change := range []struct {
		name  string
		alter func(map[string]string)
	}{
		{"outer_fields", func(m map[string]string) { m["extra"] = "synthetic" }},
		{"unknown", func(m map[string]string) {
			m["circularquay_snapshot"] = strings.Replace(m["circularquay_snapshot"], `"version":1`, `"version":1,"extra":true`, 1)
		}},
		{"duplicate", func(m map[string]string) {
			m["circularquay_snapshot"] = strings.Replace(m["circularquay_snapshot"], `"version":1`, `"version":1,"version":1`, 1)
		}},
		{"case_alias", func(m map[string]string) {
			m["circularquay_snapshot"] = strings.Replace(m["circularquay_snapshot"], `"version":1`, `"Version":1`, 1)
		}},
		{"extra_receipt", func(m map[string]string) {
			var p snapshotPayload
			_ = json.Unmarshal([]byte(m["circularquay_snapshot"]), &p)
			p.Generation.Receipts = append(p.Generation.Receipts, model.Receipt{Source: "unrelated"})
			b, _ := json.Marshal(p)
			m["circularquay_snapshot"] = string(b)
		}},
		{"missing_timestamp", func(m map[string]string) {
			var p snapshotPayload
			_ = json.Unmarshal([]byte(m["circularquay_snapshot"]), &p)
			p.Generation.RetrievedAt = time.Time{}
			b, _ := json.Marshal(p)
			m["circularquay_snapshot"] = string(b)
		}},
	} {
		t.Run(change.name, func(t *testing.T) {
			e, m := fixture(t)
			ctx := context.Background()
			g, err := e.Resolve(ctx, "demo")
			if err != nil {
				t.Fatal(err)
			}
			if err = e.Snapshot(ctx, g); err != nil {
				t.Fatal(err)
			}
			change.alter(m.saved)
			if _, err = e.restore(ctx, "demo", e.Policy.Workloads["demo"]); err == nil {
				t.Fatal("invalid snapshot accepted")
			}
		})
	}
}

func TestFreshDenialFencesLaterFallback(t *testing.T) {
	e, _ := fixture(t)
	ctx := context.Background()
	g, _ := e.Resolve(ctx, "demo")
	if err := e.Snapshot(ctx, g); err != nil {
		t.Fatal(err)
	}
	e.Reader = source.ReadFunc(func(context.Context, model.Source, model.Binding) (model.Value, error) {
		return model.Value{}, fault.New(fault.Denied, "fixture")
	})
	if _, err := e.ResolveFresh(ctx, "demo"); !fault.Is(err, fault.Denied) {
		t.Fatal("fresh denial missing")
	}
	e.Reader = source.ReadFunc(func(context.Context, model.Source, model.Binding) (model.Value, error) {
		return model.Value{}, fault.New(fault.Unavailable, "fixture")
	})
	if _, err := e.Resolve(ctx, "demo"); !fault.Is(err, fault.Denied) {
		t.Fatal("fresh denial did not fence fallback")
	}
}

func TestResolutionHonorsCancellation(t *testing.T) {
	e, _ := fixture(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	calls := 0
	e.Reader = source.ReadFunc(func(context.Context, model.Source, model.Binding) (model.Value, error) {
		calls++
		return model.Value{Data: "synthetic"}, nil
	})
	if _, err := e.Resolve(ctx, "demo"); err == nil || calls != 0 {
		t.Fatal("canceled resolution read sources")
	}
}
