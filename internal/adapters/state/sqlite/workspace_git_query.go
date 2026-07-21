package sqlite

import (
	"context"
	"errors"

	"orquesta/internal/application"
	"orquesta/internal/goal"
	"orquesta/internal/ports"
)

func (r *Repository) ListPendingChanges(ctx context.Context, q application.PendingChangeQuery) ([]application.PendingChange, error) {
	if q.ProjectRef.String() == "" || q.RepositoryRef.String() == "" || q.Limit <= 0 {
		return nil, invalid(errors.New("sqlite.pending_query_invalid"))
	}
	rows, err := r.db.QueryContext(ctx, `
SELECT goal_ref, ref
FROM change_sets change_set
WHERE project_ref=? AND repository_ref=?
  AND (?='' OR actor_ref=?)
  AND NOT EXISTS (
      SELECT 1 FROM integration_receipts receipt
      WHERE receipt.change_set_ref=change_set.ref AND receipt.status='integrated'
  )
ORDER BY committed_at, ref
LIMIT ?`, q.ProjectRef.String(), q.RepositoryRef.String(), q.ActorRef.String(), q.ActorRef.String(), q.Limit)
	if err != nil {
		return nil, mapDatabaseError(err)
	}
	type pendingSelector struct{ goalRef, changeRef string }
	selectors := make([]pendingSelector, 0, q.Limit)
	for rows.Next() {
		var selector pendingSelector
		if err := rows.Scan(&selector.goalRef, &selector.changeRef); err != nil {
			_ = rows.Close()
			return nil, mapDatabaseError(err)
		}
		selectors = append(selectors, selector)
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return nil, mapDatabaseError(err)
	}
	if err := rows.Close(); err != nil {
		return nil, mapDatabaseError(err)
	}
	records := make(map[string]application.GoalRecord)
	out := make([]application.PendingChange, 0, len(selectors))
	for _, selector := range selectors {
		rec, cached := records[selector.goalRef]
		if !cached {
			gr, err := goal.NewGoalRef(selector.goalRef)
			if err != nil {
				return nil, invalid(err)
			}
			rec, err = r.GetGoal(ctx, gr)
			if err != nil {
				return nil, err
			}
			records[selector.goalRef] = rec
		}
		changeRef, err := ports.NewChangeSetRef(selector.changeRef)
		if err != nil {
			return nil, invalid(err)
		}
		pending, err := pendingChangeProjection(rec, q, changeRef)
		if err != nil {
			return nil, err
		}
		out = append(out, pending)
	}
	return out, nil
}

func pendingChangeProjection(
	record application.GoalRecord,
	query application.PendingChangeQuery,
	ref ports.ChangeSetRef,
) (application.PendingChange, error) {
	change, ok := pendingChangeSet(record, ref)
	if !ok || change.ProjectRef != query.ProjectRef || change.RepositoryRef != query.RepositoryRef ||
		(query.ActorRef.String() != "" && change.ActorRef != query.ActorRef) {
		return application.PendingChange{}, conflict(errors.New("sqlite.pending_change_scope_conflict"))
	}
	pending := application.PendingChange{ChangeSet: change}
	for _, binding := range record.WorkspaceBindings {
		if binding.Ref == change.WorkspaceRef {
			pending.Binding = binding
		}
	}
	if pending.Binding.Ref.String() == "" {
		return application.PendingChange{}, conflict(errors.New("sqlite.pending_change_binding_missing"))
	}
	for _, observation := range record.MergeObservations {
		if observation.ChangeRef == change.Ref {
			pending.Observation = observation
		}
	}
	for _, integration := range record.IntegrationReceipts {
		if integration.ChangeRef == change.Ref {
			pending.Integration = integration
		}
	}
	if pending.Integration.Status == ports.IntegrationStatusIntegrated {
		return application.PendingChange{}, conflict(errors.New("sqlite.pending_change_integrated"))
	}
	return pending, nil
}

func pendingChangeSet(record application.GoalRecord, ref ports.ChangeSetRef) (application.ChangeSet, bool) {
	for _, change := range record.ChangeSets {
		if change.Ref == ref {
			return change, true
		}
	}
	return application.ChangeSet{}, false
}
