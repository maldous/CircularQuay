// Package resolve stages complete, lifecycle-aware workload generations.
package resolve

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"time"

	"github.com/maldous/CircularQuay/internal/fault"
	"github.com/maldous/CircularQuay/internal/model"
	"github.com/maldous/CircularQuay/internal/source"
	"github.com/maldous/CircularQuay/internal/state"
)

// SnapshotIO delegates cryptography to upstream SOPS.
type SnapshotIO interface {
	Encrypt(context.Context, model.Snapshot, map[string]string) error
	Decrypt(context.Context, model.Snapshot) (map[string]string, error)
}

// Engine has no implicit provider, environment, or fallback authority.
type Engine struct {
	Policy    model.Policy
	Reader    source.Reader
	Snapshots SnapshotIO
	State     state.Store
	Now       func() time.Time
}

func (e *Engine) now() time.Time {
	if e.Now != nil {
		return e.Now()
	}
	return time.Now().UTC()
}

// Resolve fetches all required bindings or a complete eligible fallback.
func (e *Engine) Resolve(ctx context.Context, id string) (model.Generation, error) {
	return e.resolve(ctx, id, true)
}

// ResolveFresh deliberately ignores an existing snapshot when seeding or refreshing it.
func (e *Engine) ResolveFresh(ctx context.Context, id string) (model.Generation, error) {
	return e.resolve(ctx, id, false)
}

func (e *Engine) resolve(ctx context.Context, id string, allowFallback bool) (model.Generation, error) {
	w, ok := e.Policy.Workloads[id]
	if !ok {
		return model.Generation{}, fault.New(fault.Invalid, "workload")
	}
	if e.Reader == nil {
		return model.Generation{}, fault.New(fault.Unsupported, "source reader")
	}
	if allowFallback && w.Mode == "offline" && w.Snapshot.Path != "" {
		return e.restore(ctx, id, w)
	}
	g := model.Generation{Workload: id, Values: make(map[string]model.Value), RetrievedAt: e.now()}
	for _, b := range w.Bindings {
		s, ok := e.Policy.Sources[b.Source]
		if !ok {
			return model.Generation{}, fault.New(fault.Invalid, "source")
		}
		v, err := e.Reader.Read(ctx, s, b)
		if err != nil {
			if !allowFallback {
				return model.Generation{}, err
			}
			return e.failed(ctx, id, w, err)
		}
		if err = validateValue(v, g.RetrievedAt, w.AuditRequired); err != nil {
			return e.failed(ctx, id, w, err)
		}
		g.Values[b.Name] = v
		g.Receipts = append(g.Receipts, model.Receipt{Source: b.Source, Resource: b.Resource, Version: v.Version})
	}
	return g, nil
}

func validateValue(v model.Value, now time.Time, audit bool) error {
	if !v.ExpiresAt.IsZero() && !now.Before(v.ExpiresAt) {
		return fault.New(fault.Expired, "source value")
	}
	if audit && !v.AuditVerified {
		return fault.New(fault.Unsupported, "required provider audit")
	}
	for _, c := range v.Data {
		if c == 0 {
			return fault.New(fault.Invalid, "NUL in value")
		}
	}
	return nil
}

func (e *Engine) failed(ctx context.Context, id string, w model.Workload, cause error) (model.Generation, error) {
	if w.Mode == "hybrid" && !w.AuditRequired && fault.Is(cause, fault.Unavailable) && w.Snapshot.Path != "" {
		return e.restore(ctx, id, w)
	}
	if e.State.Directory != "" {
		_ = e.State.Write(state.Record{Workload: id, State: "blocked", Mode: w.Mode, LastChecked: e.now(), Error: fault.Category(cause)})
	}
	return model.Generation{}, cause
}

// ReadKey performs a fresh exact-key read; callers must authorize disclosure first.
func (e *Engine) ReadKey(ctx context.Context, id, key string) (model.Value, error) {
	w, ok := e.Policy.Workloads[id]
	if !ok {
		return model.Value{}, fault.New(fault.Invalid, "workload")
	}
	if w.Mode == "offline" {
		g, err := e.Resolve(ctx, id)
		if err != nil {
			return model.Value{}, err
		}
		v, ok := g.Values[key]
		if !ok {
			return model.Value{}, fault.New(fault.Missing, "key")
		}
		return v, nil
	}
	for _, b := range w.Bindings {
		if b.Name == key {
			v, err := e.Reader.Read(ctx, e.Policy.Sources[b.Source], b)
			if err != nil {
				return model.Value{}, err
			}
			if err = validateValue(v, e.now(), w.AuditRequired); err != nil {
				return model.Value{}, err
			}
			if v.Leased || v.Renewable {
				return model.Value{}, fault.New(fault.Unsupported, "leased MCP disclosure")
			}
			return v, nil
		}
	}
	return model.Value{}, fault.New(fault.Missing, "key")
}

type snapshotPayload struct {
	Version    int              `json:"version"`
	Workload   string           `json:"workload"`
	Policy     string           `json:"policy"`
	Generation model.Generation `json:"generation"`
}

func (e *Engine) fingerprint(id string) (string, error) {
	w := e.Policy.Workloads[id]
	ss := make(map[string]model.Source)
	for _, b := range w.Bindings {
		ss[b.Source] = e.Policy.Sources[b.Source]
	}
	b, err := json.Marshal(struct {
		Workload model.Workload
		Sources  map[string]model.Source
	}{w, ss})
	if err != nil {
		return "", fault.New(fault.Invalid, "snapshot policy")
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:]), nil
}

// Snapshot stores only explicitly exportable static values with their provenance.
func (e *Engine) Snapshot(ctx context.Context, g model.Generation) error {
	w, ok := e.Policy.Workloads[g.Workload]
	if !ok || w.Mode == "online" || w.Snapshot.Path == "" || e.Snapshots == nil {
		return fault.New(fault.Denied, "snapshot policy")
	}
	for _, b := range w.Bindings {
		v, ok := g.Values[b.Name]
		if !ok || !b.AllowOffline || v.Leased || v.Renewable {
			return fault.New(fault.Denied, "snapshot lifecycle")
		}
		if err := validateValue(v, e.now(), false); err != nil {
			return err
		}
	}
	fp, err := e.fingerprint(g.Workload)
	if err != nil {
		return err
	}
	b, err := json.Marshal(snapshotPayload{Version: 1, Workload: g.Workload, Policy: fp, Generation: g})
	if err != nil {
		return fault.New(fault.Apply, "snapshot encode")
	}
	return e.Snapshots.Encrypt(ctx, w.Snapshot, map[string]string{"circularquay_snapshot": string(b)})
}

func (e *Engine) restore(ctx context.Context, id string, w model.Workload) (model.Generation, error) {
	if e.Snapshots == nil {
		return model.Generation{}, fault.New(fault.Unsupported, "snapshot backend")
	}
	if e.State.Directory != "" {
		r, err := e.State.Read(id)
		if err != nil {
			return model.Generation{}, err
		}
		if r.State == "blocked" {
			return model.Generation{}, fault.New(fault.Denied, "observed source failure")
		}
	}
	m, err := e.Snapshots.Decrypt(ctx, w.Snapshot)
	if err != nil {
		return model.Generation{}, err
	}
	var p snapshotPayload
	if json.Unmarshal([]byte(m["circularquay_snapshot"]), &p) != nil {
		return model.Generation{}, fault.New(fault.Trust, "snapshot format")
	}
	fp, err := e.fingerprint(id)
	if err != nil {
		return model.Generation{}, err
	}
	if p.Version != 1 || p.Workload != id || p.Policy != fp || p.Generation.Workload != id || len(p.Generation.Values) != len(w.Bindings) {
		return model.Generation{}, fault.New(fault.Trust, "snapshot scope")
	}
	for _, b := range w.Bindings {
		v, ok := p.Generation.Values[b.Name]
		if !ok || !b.AllowOffline || v.Leased || v.Renewable {
			return model.Generation{}, fault.New(fault.Denied, "snapshot scope")
		}
		if err := validateValue(v, e.now(), false); err != nil {
			return model.Generation{}, err
		}
	}
	p.Generation.Offline = true
	return p.Generation, nil
}
