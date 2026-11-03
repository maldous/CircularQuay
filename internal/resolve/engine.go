// Package resolve stages complete, lifecycle-aware workload generations.
package resolve

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"time"
	"unicode/utf8"

	"github.com/maldous/CircularQuay/internal/fault"
	"github.com/maldous/CircularQuay/internal/jsondocument"
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
	if err := checkCancellation(ctx); err != nil {
		return model.Generation{}, err
	}
	w, err := e.workload(id)
	if err != nil {
		return model.Generation{}, err
	}
	if allowFallback && w.Mode == "offline" && w.Snapshot.Path != "" {
		return e.restore(ctx, id, w)
	}
	return e.readGeneration(ctx, id, w, allowFallback)
}

func (e *Engine) workload(id string) (model.Workload, error) {
	w, ok := e.Policy.Workloads[id]
	if !ok {
		return model.Workload{}, fault.New(fault.Invalid, "workload")
	}
	if e.Reader == nil {
		return model.Workload{}, fault.New(fault.Unsupported, "source reader")
	}
	return w, nil
}

func (e *Engine) readGeneration(ctx context.Context, id string, w model.Workload, allowFallback bool) (model.Generation, error) {
	g := model.Generation{Workload: id, Values: make(map[string]model.Value), RetrievedAt: e.now()}
	for _, b := range w.Bindings {
		if err := e.readBinding(ctx, &g, b, w.AuditRequired); err != nil {
			return e.failed(ctx, id, w, err, allowFallback)
		}
	}
	return g, nil
}

func (e *Engine) readBinding(ctx context.Context, g *model.Generation, binding model.Binding, auditRequired bool) error {
	if err := checkCancellation(ctx); err != nil {
		return err
	}
	sourceConfig, ok := e.Policy.Sources[binding.Source]
	if !ok {
		return fault.New(fault.Invalid, "source")
	}
	value, err := e.Reader.Read(ctx, sourceConfig, binding)
	if canceled := checkCancellation(ctx); canceled != nil {
		return canceled
	}
	if err != nil {
		return err
	}
	if err := validateValue(value, g.RetrievedAt, auditRequired); err != nil {
		return err
	}
	g.Values[binding.Name] = value
	g.Receipts = append(g.Receipts, model.Receipt{Source: binding.Source, Resource: binding.Resource, Version: value.Version})
	return nil
}

func validateValue(v model.Value, now time.Time, audit bool) error {
	if len(v.Data) > 1<<20 || !utf8.ValidString(v.Data) || !validVersion(v.Version) {
		return fault.New(fault.Trust, "source value bounds")
	}
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

func (e *Engine) failed(ctx context.Context, id string, w model.Workload, cause error, allowFallback bool) (model.Generation, error) {
	if allowFallback && w.Mode == "hybrid" && !w.AuditRequired && fault.Is(cause, fault.Unavailable) && w.Snapshot.Path != "" {
		return e.restore(ctx, id, w)
	}
	if e.State.Directory != "" {
		_ = e.State.Write(state.Record{Workload: id, State: "blocked", Mode: w.Mode, LastChecked: e.now(), Error: fault.Category(cause)})
	}
	return model.Generation{}, cause
}

// ReadKey performs a fresh exact-key read; callers must authorize disclosure first.
func (e *Engine) ReadKey(ctx context.Context, id, key string) (model.Value, error) {
	if err := checkCancellation(ctx); err != nil {
		return model.Value{}, err
	}
	w, err := e.workload(id)
	if err != nil {
		return model.Value{}, err
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
	binding, found := binding(w, key)
	if !found {
		return model.Value{}, fault.New(fault.Missing, "key")
	}
	return e.readKeyBinding(ctx, w, binding)
}

func binding(workload model.Workload, key string) (model.Binding, bool) {
	for _, binding := range workload.Bindings {
		if binding.Name == key {
			return binding, true
		}
	}
	return model.Binding{}, false
}

func (e *Engine) readKeyBinding(ctx context.Context, workload model.Workload, binding model.Binding) (model.Value, error) {
	sourceConfig, exists := e.Policy.Sources[binding.Source]
	if !exists {
		return model.Value{}, fault.New(fault.Invalid, "source")
	}
	value, err := e.Reader.Read(ctx, sourceConfig, binding)
	if canceled := checkCancellation(ctx); canceled != nil {
		return model.Value{}, canceled
	}
	if err != nil {
		return model.Value{}, err
	}
	if err := validateValue(value, e.now(), workload.AuditRequired); err != nil {
		return model.Value{}, err
	}
	if value.Leased || value.Renewable {
		return model.Value{}, fault.New(fault.Unsupported, "leased MCP disclosure")
	}
	return value, nil
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
	if err := checkCancellation(ctx); err != nil {
		return err
	}
	w, err := e.snapshotWorkload(g.Workload)
	if err != nil {
		return err
	}
	if err := validateGeneration(w, g, e.now()); err != nil {
		return err
	}
	encoded, err := e.encodeSnapshot(g)
	if err != nil {
		return err
	}
	return e.Snapshots.Encrypt(ctx, w.Snapshot, map[string]string{"circularquay_snapshot": encoded})
}

func (e *Engine) snapshotWorkload(id string) (model.Workload, error) {
	workload, found := e.Policy.Workloads[id]
	if !found || workload.AuditRequired || workload.Mode == "online" || workload.Snapshot.Path == "" || e.Snapshots == nil {
		return model.Workload{}, fault.New(fault.Denied, "snapshot policy")
	}
	return workload, nil
}

func (e *Engine) encodeSnapshot(generation model.Generation) (string, error) {
	fingerprint, err := e.fingerprint(generation.Workload)
	if err != nil {
		return "", err
	}
	encoded, err := json.Marshal(snapshotPayload{Version: 1, Workload: generation.Workload, Policy: fingerprint, Generation: generation})
	if err != nil || len(encoded) > 2<<20 {
		return "", fault.New(fault.Apply, "snapshot encode")
	}
	return string(encoded), nil
}

func (e *Engine) restore(ctx context.Context, id string, w model.Workload) (model.Generation, error) {
	if err := checkCancellation(ctx); err != nil {
		return model.Generation{}, err
	}
	if w.AuditRequired {
		return model.Generation{}, fault.New(fault.Denied, "snapshot audit policy")
	}
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
	payload, err := e.decryptSnapshot(ctx, w)
	if err != nil {
		return model.Generation{}, err
	}
	if err := e.validateSnapshot(id, w, payload); err != nil {
		return model.Generation{}, err
	}
	payload.Generation.Offline = true
	return payload.Generation, nil
}

func (e *Engine) decryptSnapshot(ctx context.Context, workload model.Workload) (snapshotPayload, error) {
	contents, err := e.Snapshots.Decrypt(ctx, workload.Snapshot)
	if canceled := checkCancellation(ctx); canceled != nil {
		return snapshotPayload{}, canceled
	}
	if err != nil {
		return snapshotPayload{}, err
	}
	encoded, valid := contents["circularquay_snapshot"]
	if len(contents) != 1 || !valid || len(encoded) > 2<<20 {
		return snapshotPayload{}, fault.New(fault.Trust, "snapshot format")
	}
	var payload snapshotPayload
	if jsondocument.Decode([]byte(encoded), &payload) != nil {
		return snapshotPayload{}, fault.New(fault.Trust, "snapshot format")
	}
	return payload, nil
}

func (e *Engine) validateSnapshot(id string, workload model.Workload, payload snapshotPayload) error {
	fingerprint, err := e.fingerprint(id)
	if err != nil {
		return err
	}
	if payload.Version != 1 || payload.Workload != id || payload.Policy != fingerprint || payload.Generation.Workload != id || len(payload.Generation.Values) != len(workload.Bindings) {
		return fault.New(fault.Trust, "snapshot scope")
	}
	return validateGeneration(workload, payload.Generation, e.now())
}
