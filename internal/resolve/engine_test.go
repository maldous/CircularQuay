package resolve

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/maldous/CircularQuay/internal/fault"
	"github.com/maldous/CircularQuay/internal/model"
	"github.com/maldous/CircularQuay/internal/platform"
	"github.com/maldous/CircularQuay/internal/source"
	"github.com/maldous/CircularQuay/internal/state"
)

type memorySnapshot struct {
	saved                 map[string]string
	encryptErr, errorRead error
}

func privateResolveDirectory(t *testing.T, name string) string {
	t.Helper()
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	directory := filepath.Join(root, name)
	if err := platform.EnsurePrivateDirectory(directory); err != nil {
		t.Fatal(err)
	}
	return directory
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
	w := model.Workload{Mode: "hybrid", Bindings: []model.Binding{{Name: "TOKEN", Source: "a", Resource: "a", AllowOffline: true}, {Name: "OTHER", Source: "b", Resource: "b", AllowOffline: true}}, Snapshot: model.Snapshot{Path: filepath.Join(privateResolveDirectory(t, "snapshot"), "snapshot"), Recipients: []string{"age1fixture"}}}
	p := model.Policy{Version: 1, Workloads: map[string]model.Workload{"demo": w}, Sources: map[string]model.Source{"a": {Type: "fake"}, "b": {Type: "fake"}}}
	m := &memorySnapshot{}
	e := &Engine{Policy: p, Reader: source.ReadFunc(func(_ context.Context, _ model.Source, b model.Binding) (model.Value, error) {
		return model.Value{Data: "synthetic-only-" + b.Name, Version: "v1"}, nil
	}), Snapshots: m, State: state.Store{Directory: privateResolveDirectory(t, "state")}, Now: func() time.Time { return now }}
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

func TestResolveRejectsValueExpiredBeforeGenerationCompletes(t *testing.T) {
	e, snapshots := fixture(t)
	ctx := context.Background()
	seed, err := e.Resolve(ctx, "demo")
	if err != nil {
		t.Fatal(err)
	}
	if err = e.Snapshot(ctx, seed); err != nil {
		t.Fatal(err)
	}

	now := e.now()
	expires := now.Add(time.Minute)
	e.Now = func() time.Time { return now }
	e.Reader = source.ReadFunc(func(_ context.Context, _ model.Source, binding model.Binding) (model.Value, error) {
		if binding.Name == "TOKEN" {
			return model.Value{Data: "synthetic-first", Version: "v2", ExpiresAt: expires}, nil
		}
		now = expires
		return model.Value{Data: "synthetic-later", Version: "v2"}, nil
	})

	generation, err := e.Resolve(ctx, "demo")
	if !fault.Is(err, fault.Expired) || generation.Offline || len(generation.Values) != 0 {
		t.Fatalf("expired generation or fallback accepted: generation=%+v error=%v", generation, err)
	}
	if len(snapshots.saved) != 1 {
		t.Fatal("test fixture lost its eligible snapshot")
	}
}

func TestResolveRejectsIncompleteGenerationFromInvalidPolicy(t *testing.T) {
	e, _ := fixture(t)
	workload := e.Policy.Workloads["demo"]
	workload.Bindings[1].Name = workload.Bindings[0].Name
	e.Policy.Workloads["demo"] = workload
	if _, err := e.Resolve(context.Background(), "demo"); !fault.Is(err, fault.Trust) {
		t.Fatalf("incomplete generation accepted: %v", err)
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
		m.saved = map[string]string{"cq_snapshot": data}
		if _, err := e.restore(ctx, "demo", e.Policy.Workloads["demo"]); err == nil {
			t.Fatal("invalid snapshot accepted")
		}
	}
	g, _ := e.Resolve(ctx, "demo")
	if err := e.Snapshot(ctx, g); err != nil {
		t.Fatal(err)
	}
	var p snapshotPayload
	if err := json.Unmarshal([]byte(m.saved["cq_snapshot"]), &p); err != nil {
		t.Fatal(err)
	}
	v := p.Generation.Values["TOKEN"]
	v.Leased = true
	p.Generation.Values["TOKEN"] = v
	b, _ := json.Marshal(p)
	m.saved["cq_snapshot"] = string(b)
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
	if err = json.Unmarshal([]byte(snapshots.saved["cq_snapshot"]), &payload); err != nil {
		t.Fatal(err)
	}
	payload.Policy = "different-valid-policy-fingerprint"
	encoded, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	snapshots.saved["cq_snapshot"] = string(encoded)

	if _, err = e.restore(ctx, "demo", e.Policy.Workloads["demo"]); !fault.Is(err, fault.Trust) {
		t.Fatalf("mismatched policy fingerprint accepted: %v", err)
	}
}

func TestSnapshotRejectsPolicyVersionAndPathChanges(t *testing.T) {
	for _, change := range []struct {
		name  string
		apply func(*Engine)
	}{
		{"version", func(engine *Engine) { engine.Policy.Version++ }},
		{"path", func(engine *Engine) {
			workload := engine.Policy.Workloads["demo"]
			workload.Snapshot.Path += ".other"
			engine.Policy.Workloads["demo"] = workload
		}},
	} {
		t.Run(change.name, func(t *testing.T) {
			e, _ := fixture(t)
			generation, err := e.Resolve(context.Background(), "demo")
			if err != nil {
				t.Fatal(err)
			}
			if err = e.Snapshot(context.Background(), generation); err != nil {
				t.Fatal(err)
			}
			change.apply(e)
			e.Reader = unavailableReader()
			if _, err = e.Resolve(context.Background(), "demo"); !fault.Is(err, fault.Trust) {
				t.Fatalf("changed policy accepted snapshot: %v", err)
			}
		})
	}
}

func TestSnapshotRejectsFutureAndExpiredValues(t *testing.T) {
	for _, change := range []struct {
		name  string
		apply func(*snapshotPayload, time.Time)
		want  fault.Kind
	}{
		{"future_generation_clock", func(payload *snapshotPayload, now time.Time) {
			payload.Generation.RetrievedAt = now.Add(time.Nanosecond)
		}, fault.Trust},
		{"expired_value", func(payload *snapshotPayload, now time.Time) {
			value := payload.Generation.Values["TOKEN"]
			value.ExpiresAt = now
			payload.Generation.Values["TOKEN"] = value
		}, fault.Expired},
	} {
		t.Run(change.name, func(t *testing.T) {
			assertSnapshotValueRejected(t, change.apply, change.want)
		})
	}
}

func assertSnapshotValueRejected(t *testing.T, change func(*snapshotPayload, time.Time), want fault.Kind) {
	t.Helper()
	engine, snapshots := fixture(t)
	generation, err := engine.Resolve(context.Background(), "demo")
	if err != nil {
		t.Fatal(err)
	}
	if err = engine.Snapshot(context.Background(), generation); err != nil {
		t.Fatal(err)
	}
	payload := decodeSnapshotPayload(t, snapshots)
	change(&payload, engine.now())
	snapshots.saved["cq_snapshot"] = encodeSnapshotPayload(t, payload)
	engine.Reader = unavailableReader()
	if _, err = engine.Resolve(context.Background(), "demo"); !fault.Is(err, want) {
		t.Fatalf("invalid snapshot result = %v, want %s", err, want)
	}
}

func decodeSnapshotPayload(t *testing.T, snapshots *memorySnapshot) snapshotPayload {
	t.Helper()
	var payload snapshotPayload
	if err := json.Unmarshal([]byte(snapshots.saved["cq_snapshot"]), &payload); err != nil {
		t.Fatal(err)
	}
	return payload
}

func encodeSnapshotPayload(t *testing.T, payload snapshotPayload) string {
	t.Helper()
	encoded, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	return string(encoded)
}

func unavailableReader() source.Reader {
	return source.ReadFunc(func(context.Context, model.Source, model.Binding) (model.Value, error) {
		return model.Value{}, fault.New(fault.Unavailable, "fixture")
	})
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
			m["cq_snapshot"] = strings.Replace(m["cq_snapshot"], `"version":1`, `"version":1,"extra":true`, 1)
		}},
		{"duplicate", func(m map[string]string) {
			m["cq_snapshot"] = strings.Replace(m["cq_snapshot"], `"version":1`, `"version":1,"version":1`, 1)
		}},
		{"case_alias", func(m map[string]string) {
			m["cq_snapshot"] = strings.Replace(m["cq_snapshot"], `"version":1`, `"Version":1`, 1)
		}},
		{"extra_receipt", func(m map[string]string) {
			var p snapshotPayload
			_ = json.Unmarshal([]byte(m["cq_snapshot"]), &p)
			p.Generation.Receipts = append(p.Generation.Receipts, model.Receipt{Source: "unrelated"})
			b, _ := json.Marshal(p)
			m["cq_snapshot"] = string(b)
		}},
		{"missing_timestamp", func(m map[string]string) {
			var p snapshotPayload
			_ = json.Unmarshal([]byte(m["cq_snapshot"]), &p)
			p.Generation.RetrievedAt = time.Time{}
			b, _ := json.Marshal(p)
			m["cq_snapshot"] = string(b)
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

func TestFreshDenialFencesFallbackWithoutStateStore(t *testing.T) {
	e, _ := fixture(t)
	e.State = state.Store{}
	ctx := context.Background()
	generation, err := e.Resolve(ctx, "demo")
	if err != nil {
		t.Fatal(err)
	}
	if err = e.Snapshot(ctx, generation); err != nil {
		t.Fatal(err)
	}
	e.Reader = source.ReadFunc(func(context.Context, model.Source, model.Binding) (model.Value, error) {
		return model.Value{}, fault.New(fault.Denied, "fixture")
	})
	if _, err = e.ResolveFresh(ctx, "demo"); !fault.Is(err, fault.Denied) {
		t.Fatalf("fresh denial result = %v", err)
	}
	e.Reader = unavailableReader()
	if _, err = e.Resolve(ctx, "demo"); !fault.Is(err, fault.Denied) {
		t.Fatalf("fresh denial allowed in-memory fallback: %v", err)
	}
}

func TestProviderFailureFenceSurvivesRestart(t *testing.T) {
	e, snapshots := fixture(t)
	seedFallbackSnapshot(t, e)
	e.Reader = source.ReadFunc(func(context.Context, model.Source, model.Binding) (model.Value, error) {
		return model.Value{}, fault.New(fault.Denied, "fixture")
	})
	if _, err := e.ResolveFresh(context.Background(), "demo"); !fault.Is(err, fault.Denied) {
		t.Fatalf("fresh denial = %v, want denied", err)
	}

	restarted := restartEngine(e, snapshots, unavailableReader())
	if _, err := restarted.Resolve(context.Background(), "demo"); !fault.Is(err, fault.Denied) {
		t.Fatalf("restarted fallback = %v, want denied", err)
	}
	record := readStateRecord(t, restarted)
	if record.State != "blocked" || !record.FallbackFence {
		t.Fatalf("restart cleared denial fence: %+v", record)
	}
}

func TestProviderFenceSurvivesFailureWriteAndRestart(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX directory write permissions")
	}
	e, snapshots := fixture(t)
	seedFallbackSnapshot(t, e)
	calls := 0
	e.Reader = source.ReadFunc(func(context.Context, model.Source, model.Binding) (model.Value, error) {
		calls++
		if err := os.Chmod(e.State.Directory, 0o500); err != nil {
			t.Fatal(err)
		}
		return model.Value{}, fault.New(fault.Denied, "fixture")
	})
	t.Cleanup(func() { _ = os.Chmod(e.State.Directory, 0o700) })
	if _, err := e.ResolveFresh(context.Background(), "demo"); err == nil {
		t.Fatal("provider failure with an unwritable final state write succeeded")
	} else if fault.Is(err, fault.Denied) {
		t.Fatalf("final state write unexpectedly succeeded: %v", err)
	}
	if calls != 1 {
		t.Fatalf("provider calls = %d, want one", calls)
	}
	if record := readStateRecord(t, e); !record.FallbackFence {
		t.Fatalf("pre-read fence was not durable after final write failure: %+v", record)
	}
	if err := os.Chmod(e.State.Directory, 0o700); err != nil {
		t.Fatal(err)
	}

	restarted := restartEngine(e, snapshots, unavailableReader())
	if _, err := restarted.Resolve(context.Background(), "demo"); !fault.Is(err, fault.Denied) {
		t.Fatalf("restart restored an unfenced snapshot: %v", err)
	}
	record := readStateRecord(t, restarted)
	if !record.FallbackFence {
		t.Fatalf("interrupted failure lost durable fence: %+v", record)
	}
}

func TestProviderReadRequiresDurableFenceBeforeReader(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX directory write permissions")
	}
	e, _ := fixture(t)
	prior := state.Record{Workload: "demo", State: "ready", Mode: "hybrid"}
	if err := e.State.Write(prior); err != nil {
		t.Fatal(err)
	}
	if _, err := e.State.Read("demo"); err != nil {
		t.Fatalf("readable state before permission change: %v", err)
	}
	if err := os.Chmod(e.State.Directory, 0o500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(e.State.Directory, 0o700) })
	if _, err := e.State.Read("demo"); err != nil {
		t.Fatalf("readable state rejected after permission change: %v", err)
	}
	calls := 0
	e.Reader = source.ReadFunc(func(context.Context, model.Source, model.Binding) (model.Value, error) {
		calls++
		return model.Value{Data: "must-not-be-read"}, nil
	})
	if _, err := e.ResolveFresh(context.Background(), "demo"); err == nil {
		t.Fatal("provider read proceeded without a durable fence")
	}
	if calls != 0 {
		t.Fatalf("provider calls = %d, want zero", calls)
	}
}

func TestProviderFenceRestoresUnfencedRecordBeforeFallback(t *testing.T) {
	e, _ := fixture(t)
	seedFallbackSnapshot(t, e)
	prior := state.Record{
		Workload:    "demo",
		Generation:  "previous",
		State:       "ready",
		Mode:        "hybrid",
		LastChecked: e.now().Add(-time.Hour),
		LastSuccess: e.now().Add(-2 * time.Hour),
		Targets: map[string]state.TargetRecord{
			"app": {State: "activated", Generation: "previous", Current: true},
		},
		Receipts: []model.Receipt{{Source: "a", Resource: "a", Version: "v0"}},
	}
	if err := e.State.Write(prior); err != nil {
		t.Fatal(err)
	}
	e.Reader = unavailableReader()
	generation, err := e.Resolve(context.Background(), "demo")
	if err != nil || !generation.Offline {
		t.Fatalf("fallback = %+v, %v", generation, err)
	}
	if actual := readStateRecord(t, e); !reflect.DeepEqual(actual, prior) {
		t.Fatalf("unavailable refresh altered prior state:\n got: %#v\nwant: %#v", actual, prior)
	}
}

func TestCompleteGenerationClearsFenceAndPreservesHistory(t *testing.T) {
	e, _ := fixture(t)
	prior := state.Record{
		Workload:      "demo",
		Generation:    "previous",
		State:         "blocked",
		Mode:          "hybrid",
		FallbackFence: true,
		Targets: map[string]state.TargetRecord{
			"app": {State: "activated", Generation: "previous", Current: true},
		},
		Receipts: []model.Receipt{{Source: "a", Resource: "a", Version: "v0"}},
	}
	if err := e.State.Write(prior); err != nil {
		t.Fatal(err)
	}
	generation, err := e.ResolveFresh(context.Background(), "demo")
	if err != nil || generation.Offline {
		t.Fatalf("fresh generation = %+v, %v", generation, err)
	}
	record := readStateRecord(t, e)
	if record.State != "ready" || record.FallbackFence || record.Error != "" {
		t.Fatalf("fresh generation did not normalize fence: %+v", record)
	}
	if record.Generation != prior.Generation || !reflect.DeepEqual(record.Targets, prior.Targets) || !reflect.DeepEqual(record.Receipts, prior.Receipts) {
		t.Fatalf("fresh generation lost retained metadata: %+v", record)
	}
}

func TestReadKeyDoesNotClearWholeWorkloadFence(t *testing.T) {
	e, snapshots := fixture(t)
	seedFallbackSnapshot(t, e)
	prior := state.Record{Workload: "demo", State: "blocked", Mode: "hybrid", FallbackFence: true}
	if err := e.State.Write(prior); err != nil {
		t.Fatal(err)
	}
	if _, err := e.ReadKey(context.Background(), "demo", "TOKEN"); err != nil {
		t.Fatalf("read key = %v", err)
	}
	if actual := readStateRecord(t, e); !reflect.DeepEqual(actual, prior) {
		t.Fatalf("single key read cleared workload fence: %#v", actual)
	}
	restarted := restartEngine(e, snapshots, unavailableReader())
	if _, err := restarted.Resolve(context.Background(), "demo"); !fault.Is(err, fault.Denied) {
		t.Fatalf("single key read enabled fallback: %v", err)
	}
}

func TestProviderFailuresPersistFallbackFence(t *testing.T) {
	for _, test := range []struct {
		name   string
		reader source.Reader
	}{
		{"denied", source.ReadFunc(func(context.Context, model.Source, model.Binding) (model.Value, error) {
			return model.Value{}, fault.New(fault.Denied, "fixture")
		})},
		{"trust", source.ReadFunc(func(context.Context, model.Source, model.Binding) (model.Value, error) {
			return model.Value{}, fault.New(fault.Trust, "fixture")
		})},
		{"expired", source.ReadFunc(func(context.Context, model.Source, model.Binding) (model.Value, error) {
			return model.Value{Data: "fixture", ExpiresAt: time.Date(2026, 11, 3, 11, 0, 0, 0, time.UTC)}, nil
		})},
		{"malformed", source.ReadFunc(func(context.Context, model.Source, model.Binding) (model.Value, error) {
			return model.Value{Data: "bad\x00value"}, nil
		})},
	} {
		t.Run(test.name, func(t *testing.T) {
			e, _ := fixture(t)
			e.Reader = test.reader
			if _, err := e.ResolveFresh(context.Background(), "demo"); err == nil {
				t.Fatal("unsafe provider result succeeded")
			}
			record := readStateRecord(t, e)
			if record.State != "blocked" || !record.FallbackFence {
				t.Fatalf("unsafe provider result did not persist fence: %+v", record)
			}
		})
	}
}

func seedFallbackSnapshot(t *testing.T, engine *Engine) {
	t.Helper()
	generation, err := engine.Resolve(context.Background(), "demo")
	if err != nil {
		t.Fatal(err)
	}
	if err := engine.Snapshot(context.Background(), generation); err != nil {
		t.Fatal(err)
	}
}

func restartEngine(engine *Engine, snapshots SnapshotIO, reader source.Reader) *Engine {
	return &Engine{Policy: engine.Policy, Reader: reader, Snapshots: snapshots, State: engine.State, Now: engine.Now}
}

func readStateRecord(t *testing.T, engine *Engine) state.Record {
	t.Helper()
	record, err := engine.State.Read("demo")
	if err != nil {
		t.Fatal(err)
	}
	return record
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
