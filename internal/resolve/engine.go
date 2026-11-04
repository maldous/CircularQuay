// Package resolve stages complete, lifecycle-aware workload generations.
package resolve

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/maldous/CircularQuay/internal/fault"
	"github.com/maldous/CircularQuay/internal/jsondocument"
	"github.com/maldous/CircularQuay/internal/model"
	"github.com/maldous/CircularQuay/internal/observability"
	"github.com/maldous/CircularQuay/internal/source"
	"github.com/maldous/CircularQuay/internal/state"
)

// SnapshotIO delegates cryptography to upstream SOPS.
type SnapshotIO interface {
	Encrypt(context.Context, model.Snapshot, map[string]string) error
	Decrypt(context.Context, model.Snapshot) (map[string]string, error)
}

// Engine has no implicit provider, environment, or fallback authority.
//
// Callers sharing a persisted State hold Store.Lock for the workload around an
// Engine operation. Engine deliberately does not take that lock so higher
// layers can compose resolution with activation without nested locks.
type Engine struct {
	Policy    model.Policy
	Reader    source.Reader
	Snapshots SnapshotIO
	State     state.Store
	Now       func() time.Time

	mu      sync.Mutex
	blocked map[string]struct{}
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
func (e *Engine) ResolveFresh(ctx context.Context, id string) (generation model.Generation, err error) {
	ctx, complete := observability.Start(ctx, observability.Operation{Phase: observability.PhaseRefresh, ProviderKind: "none"})
	defer func() { complete(observability.OutcomeForError(err)) }()
	return e.resolve(ctx, id, false)
}

func (e *Engine) resolve(ctx context.Context, id string, allowFallback bool) (generation model.Generation, err error) {
	ctx, complete := observability.Start(ctx, observability.Operation{Phase: observability.PhaseResolve, ProviderKind: "none"})
	defer func() { complete(observability.OutcomeForError(err)) }()
	if err := checkCancellation(ctx); err != nil {
		return model.Generation{}, err
	}
	w, err := e.workload(id)
	if err != nil {
		return model.Generation{}, err
	}
	if allowFallback && w.Mode == "offline" && w.Snapshot.Path != "" {
		generation, err = e.restore(ctx, id, w)
		return generation, err
	}
	generation, err = e.readGeneration(ctx, id, w, allowFallback)
	return generation, err
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
	fence, err := e.prepareProviderRead(id, w)
	if err != nil {
		return model.Generation{}, err
	}
	g := model.Generation{Workload: id, Values: make(map[string]model.Value), RetrievedAt: e.now()}
	for _, b := range w.Bindings {
		if err := e.readBinding(ctx, &g, b, w.AuditRequired); err != nil {
			return e.failed(ctx, id, w, err, allowFallback, fence)
		}
	}
	if err := validateFreshGeneration(w, g, e.now(), w.AuditRequired); err != nil {
		return e.failed(ctx, id, w, err, allowFallback, fence)
	}
	if err := e.completeProviderRead(id, w, g, fence); err != nil {
		return model.Generation{}, err
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
	if err := validateValue(value, e.now(), auditRequired); err != nil {
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

func (e *Engine) failed(ctx context.Context, id string, w model.Workload, cause error, allowFallback bool, fence providerReadFence) (model.Generation, error) {
	if fault.Is(cause, fault.Unavailable) {
		if err := e.restoreProviderFence(fence); err != nil {
			return model.Generation{}, err
		}
		if eligibleFallback(w, cause, allowFallback) {
			observability.Record(ctx, observability.Operation{Phase: observability.PhaseRecovery, ProviderKind: "none"}, observability.OutcomeFallback)
			return e.restore(ctx, id, w)
		}
		return model.Generation{}, cause
	}
	if blocksFallback(cause) {
		e.blockFallback(id)
	}
	if err := e.persistProviderFailure(id, w, cause, fence); err != nil {
		return model.Generation{}, err
	}
	return model.Generation{}, cause
}

func eligibleFallback(workload model.Workload, cause error, allowed bool) bool {
	return allowed && workload.Mode == "hybrid" && !workload.AuditRequired && fault.Is(cause, fault.Unavailable) && workload.Snapshot.Path != ""
}

func blocksFallback(cause error) bool {
	return !fault.Is(cause, fault.Unavailable) && !fault.Is(cause, fault.Apply)
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
	if w.Mode == "offline" && w.Snapshot.Path != "" {
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
	return e.readKeyOnline(ctx, id, w, binding)
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

func (e *Engine) readKeyOnline(ctx context.Context, id string, workload model.Workload, binding model.Binding) (model.Value, error) {
	fence, err := e.prepareProviderRead(id, workload)
	if err != nil {
		return model.Value{}, err
	}
	value, err := e.readKeyBinding(ctx, workload, binding)
	if err != nil {
		return model.Value{}, e.finishKeyRead(id, workload, err, fence)
	}
	if err := e.restoreProviderFence(fence); err != nil {
		return model.Value{}, err
	}
	return value, nil
}

func (e *Engine) finishKeyRead(id string, workload model.Workload, cause error, fence providerReadFence) error {
	if fault.Is(cause, fault.Unavailable) {
		if err := e.restoreProviderFence(fence); err != nil {
			return err
		}
		return cause
	}
	if blocksFallback(cause) {
		e.blockFallback(id)
	}
	if err := e.persistProviderFailure(id, workload, cause, fence); err != nil {
		return err
	}
	return cause
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
		Version  int
		Workload model.Workload
		Sources  map[string]model.Source
	}{Version: e.Policy.Version, Workload: w, Sources: ss})
	if err != nil {
		return "", fault.New(fault.Invalid, "snapshot policy")
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:]), nil
}

// Snapshot stores only explicitly exportable static values with their provenance.
func (e *Engine) Snapshot(ctx context.Context, g model.Generation) (err error) {
	ctx, complete := observability.Start(ctx, observability.Operation{Phase: observability.PhaseSnapshot, ProviderKind: "sops"})
	defer func() { complete(observability.OutcomeForError(err)) }()
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

func (e *Engine) restore(ctx context.Context, id string, w model.Workload) (generation model.Generation, err error) {
	ctx, complete := observability.Start(ctx, observability.Operation{Phase: observability.PhaseRecovery, ProviderKind: "sops"})
	defer func() { complete(observability.OutcomeForError(err)) }()
	if err := e.restoreAllowed(ctx, id, w); err != nil {
		return model.Generation{}, err
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

func (e *Engine) restoreAllowed(ctx context.Context, id string, workload model.Workload) error {
	if err := checkCancellation(ctx); err != nil {
		return err
	}
	if workload.AuditRequired {
		return fault.New(fault.Denied, "snapshot audit policy")
	}
	if e.Snapshots == nil {
		return fault.New(fault.Unsupported, "snapshot backend")
	}
	if e.fallbackBlocked(id) {
		return fault.New(fault.Denied, "observed source failure")
	}
	return e.restoreStateAllows(id)
}

func (e *Engine) restoreStateAllows(id string) error {
	if e.State.Directory == "" {
		return nil
	}
	record, err := e.State.Read(id)
	if err != nil {
		return err
	}
	if record.State == "blocked" || record.FallbackFence {
		return fault.New(fault.Denied, "observed source failure")
	}
	return nil
}

type providerReadFence struct {
	prior      state.Record
	persistent bool
}

func (e *Engine) prepareProviderRead(id string, workload model.Workload) (providerReadFence, error) {
	if e.State.Directory == "" {
		return providerReadFence{}, nil
	}
	prior, err := e.State.Read(id)
	if err != nil {
		return providerReadFence{}, err
	}
	fenced := prior
	fenced.Workload = id
	if fenced.Mode == "" {
		fenced.Mode = workload.Mode
	}
	fenced.FallbackFence = true
	fenced.LastChecked = e.now()
	if err := e.State.Write(fenced); err != nil {
		return providerReadFence{}, err
	}
	return providerReadFence{prior: prior, persistent: true}, nil
}

func (e *Engine) restoreProviderFence(fence providerReadFence) error {
	if !fence.persistent {
		return nil
	}
	return e.State.Write(fence.prior)
}

func (e *Engine) persistProviderFailure(id string, workload model.Workload, cause error, fence providerReadFence) error {
	if !fence.persistent {
		return nil
	}
	record := fence.prior
	record.Workload = id
	record.State = "blocked"
	record.Mode = workload.Mode
	record.FallbackFence = true
	record.LastChecked = e.now()
	record.Error = fault.Category(cause)
	return e.State.Write(record)
}

func (e *Engine) completeProviderRead(id string, workload model.Workload, generation model.Generation, fence providerReadFence) error {
	if fence.persistent {
		record := fence.prior
		record.Workload = id
		record.Mode = workload.Mode
		record.FallbackFence = false
		if record.State == "blocked" {
			record.State = "ready"
		}
		record.Offline = false
		record.LastChecked = e.now()
		record.LastSuccess = generation.RetrievedAt
		record.Error = ""
		if err := e.State.Write(record); err != nil {
			return err
		}
	}
	e.clearFallbackBlock(id)
	return nil
}

func (e *Engine) blockFallback(id string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.blocked == nil {
		e.blocked = make(map[string]struct{})
	}
	e.blocked[id] = struct{}{}
}

func (e *Engine) clearFallbackBlock(id string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	delete(e.blocked, id)
}

func (e *Engine) fallbackBlocked(id string) bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	_, blocked := e.blocked[id]
	return blocked
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
