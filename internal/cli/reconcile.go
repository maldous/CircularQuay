package cli

import (
	"context"
	"sort"
	"time"

	"github.com/maldous/CircularQuay/internal/fault"
	"github.com/maldous/CircularQuay/internal/model"
	"github.com/maldous/CircularQuay/internal/observability"
	"github.com/maldous/CircularQuay/internal/platform"
	"github.com/maldous/CircularQuay/internal/state"
)

type targetOperations struct {
	project  func(context.Context, model.Workload, string, model.Generation) error
	activate func(context.Context, model.Target) error
}

func selectedTargets(w model.Workload, target string) ([]string, error) {
	if target != "" {
		if _, ok := w.Targets[target]; !ok {
			return nil, fault.New(fault.Denied, "target mapping")
		}
		return []string{target}, nil
	}
	ids := make([]string, 0, len(w.Targets))
	for id := range w.Targets {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids, nil
}

// Reconciliation exposes partial projection honestly. Destinations are separate
// stores and cannot form a distributed transaction. No service is transitioned
// until every selected projection has succeeded and its receipt is durable.
func reconcileTargets(ctx context.Context, s state.Store, w model.Workload, g model.Generation, ids []string, operations targetOperations) (record state.Record, err error) {
	ctx, complete := observability.Start(ctx, observability.Operation{Phase: observability.PhaseProjection, ProviderKind: "none"})
	defer func() { complete(observability.OutcomeForError(err)) }()
	record, err = generationRecord(w, g, "resolved")
	if err != nil {
		return record, err
	}
	previous, err := s.Read(g.Workload)
	if err != nil {
		return record, err
	}
	selectionComplete := completeSelection(w, ids)
	record.State = scopedPhase(selectionComplete, "resolved")
	record.Targets = retainedTargets(previous.Targets, len(ids))
	for _, id := range ids {
		record.Targets[id] = state.TargetRecord{State: "resolved", Generation: record.Generation, Current: true}
	}
	if err = s.Write(record); err != nil {
		return record, err
	}
	reconciler := targetReconciler{store: s, workload: w, generation: g, record: &record, operations: operations, complete: selectionComplete}
	if err = reconciler.projectAll(ctx, ids); err != nil {
		return record, err
	}
	if err = reconciler.activateAll(ctx, ids); err != nil {
		return record, err
	}
	record.State = reconciler.phase(completedPhase(w, ids))
	return record, s.Write(record)
}

func retainedTargets(previous map[string]state.TargetRecord, capacity int) map[string]state.TargetRecord {
	targets := make(map[string]state.TargetRecord, len(previous)+capacity)
	for id, target := range previous {
		target.Current = false
		targets[id] = target
	}
	return targets
}

type targetReconciler struct {
	store      state.Store
	workload   model.Workload
	generation model.Generation
	record     *state.Record
	operations targetOperations
	complete   bool
}

func (r targetReconciler) projectAll(ctx context.Context, ids []string) error {
	for _, id := range ids {
		if err := projectTarget(ctx, r.workload, r.generation, id, r.operations); err != nil {
			r.record.State = r.phase("apply_failed")
			r.record.Targets[id] = state.TargetRecord{State: "apply_failed", Generation: r.record.Generation, Current: true, Error: fault.Category(err)}
			return r.failed(err)
		}
		r.record.Targets[id] = state.TargetRecord{State: "applied", Generation: r.record.Generation, Current: true, AppliedAt: time.Now().UTC()}
		if err := r.store.Write(*r.record); err != nil {
			return err
		}
	}
	return nil
}

func (r targetReconciler) activateAll(ctx context.Context, ids []string) error {
	for _, id := range ids {
		if !needsActivation(r.workload.Targets[id]) {
			continue
		}
		if err := r.activate(ctx, id); err != nil {
			return err
		}
	}
	return nil
}

func (r targetReconciler) activate(ctx context.Context, id string) error {
	entry := r.record.Targets[id]
	entry.State = "pending_activation"
	r.record.Targets[id] = entry
	r.record.State = r.phase("pending_activation")
	if err := r.store.Write(*r.record); err != nil {
		return err
	}
	var err error
	if ctx.Err() != nil {
		err = fault.New(fault.Apply, "activation canceled")
	} else {
		err = r.operations.activate(ctx, r.workload.Targets[id])
	}
	if err != nil {
		entry.Error = fault.Category(err)
		r.record.Targets[id] = entry
		return r.failed(err)
	}
	entry.State = "activated"
	entry.ActivatedAt = time.Now().UTC()
	r.record.Targets[id] = entry
	return r.store.Write(*r.record)
}

func (r targetReconciler) phase(phase string) string {
	return scopedPhase(r.complete, phase)
}

func scopedPhase(complete bool, phase string) string {
	if complete {
		return phase
	}
	return "partial_" + phase
}

func (r targetReconciler) failed(cause error) error {
	r.record.Error = fault.Category(cause)
	if err := r.store.Write(*r.record); err != nil {
		return err
	}
	return cause
}

func completedPhase(w model.Workload, ids []string) string {
	if len(ids) == 0 {
		return "resolved"
	}
	if hasActivation(w, ids) {
		return "activated"
	}
	return "applied"
}

func completeSelection(w model.Workload, ids []string) bool {
	if len(ids) != len(w.Targets) {
		return false
	}
	for _, id := range ids {
		if _, ok := w.Targets[id]; !ok {
			return false
		}
	}
	return true
}

func projectTarget(ctx context.Context, w model.Workload, g model.Generation, id string, operations targetOperations) (err error) {
	provider := "none"
	if target, found := w.Targets[id]; found {
		provider = target.Type
	}
	ctx, complete := observability.Start(ctx, observability.Operation{Phase: observability.PhaseProjection, ProviderKind: provider})
	defer func() { complete(observability.OutcomeForError(err)) }()
	if ctx.Err() != nil {
		return fault.New(fault.Apply, "projection canceled")
	}
	for _, v := range g.Values {
		if v.Leased || v.Renewable || !v.ExpiresAt.IsZero() {
			return fault.New(fault.Denied, "static projection required")
		}
	}
	return operations.project(ctx, w, id, g)
}

func needsActivation(target model.Target) bool {
	return target.Activation == "restart" || target.Activation == "reload"
}
func hasActivation(w model.Workload, ids []string) bool {
	for _, id := range ids {
		if needsActivation(w.Targets[id]) {
			return true
		}
	}
	return false
}

func nativeOperations() targetOperations {
	return targetOperations{project: apply, activate: platform.Activate}
}
