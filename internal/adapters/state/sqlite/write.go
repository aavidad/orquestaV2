package sqlite

import (
	"context"
	"database/sql"
	"fmt"

	"orquesta/internal/application"
	"orquesta/internal/goal"
)

func insertCreateState(ctx context.Context, transaction *sql.Tx, state application.CreateGoalState) error {
	snapshot := state.Goal.Snapshot()
	if err := insertGoalHeader(ctx, transaction, state.RequestRef, state.RequestFingerprint, snapshot); err != nil {
		return err
	}
	for position, phase := range snapshot.Phases {
		if _, err := transaction.ExecContext(ctx, `
INSERT INTO goal_phases(goal_ref, phase_key, position) VALUES (?, ?, ?)`,
			snapshot.Ref, phase.Key, position,
		); err != nil {
			return mapDatabaseError(err)
		}
	}
	for position, item := range snapshot.WorkItems {
		if _, err := transaction.ExecContext(ctx, `
INSERT INTO work_items(
    ref, goal_ref, actor_ref, project_ref, objective, phase_key, role_key,
    output_contract, skip_reason, state, revision, position,
    created_at, started_at, finished_at, execution_ref
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			item.Ref,
			item.GoalRef,
			item.ActorRef,
			item.ProjectRef,
			item.Objective,
			item.PhaseKey,
			item.RoleKey,
			string(item.OutputContract),
			string(item.SkipReason),
			string(item.State),
			int64(item.Revision),
			position,
			requiredTime(item.CreatedAt),
			storedTime(item.StartedAt),
			storedTime(item.FinishedAt),
			nullableString(item.ExecutionRef),
		); err != nil {
			return mapDatabaseError(err)
		}
		for dependencyPosition, dependency := range item.DependencyRefs {
			if _, err := transaction.ExecContext(ctx, `
INSERT INTO work_item_dependencies(goal_ref, work_item_ref, dependency_ref, position)
VALUES (?, ?, ?, ?)`, item.GoalRef, item.Ref, dependency, dependencyPosition); err != nil {
				return mapDatabaseError(err)
			}
		}
		for scopePosition, scope := range item.WriteSet {
			if _, err := transaction.ExecContext(ctx, `
INSERT INTO work_item_write_scopes(goal_ref, work_item_ref, scope, position)
VALUES (?, ?, ?, ?)`, item.GoalRef, item.Ref, scope, scopePosition); err != nil {
				return mapDatabaseError(err)
			}
		}
	}
	for _, execution := range state.Executions {
		if err := insertExecution(ctx, transaction, execution); err != nil {
			return err
		}
	}
	for _, action := range state.Actions {
		if err := insertAction(ctx, transaction, action); err != nil {
			return err
		}
	}
	return insertEvents(ctx, transaction, state.Events)
}

func insertGoalHeader(
	ctx context.Context,
	transaction *sql.Tx,
	requestRef string,
	requestFingerprint string,
	snapshot goal.GoalSnapshot,
) error {
	intent := snapshot.AppSpec.Intent
	if _, err := transaction.ExecContext(ctx, `
INSERT INTO intents(ref, actor_ref, project_ref, statement, submitted_at, hash)
VALUES (?, ?, ?, ?, ?, ?)`,
		intent.Ref,
		intent.ActorRef,
		intent.ProjectRef,
		intent.Statement,
		requiredTime(intent.SubmittedAt),
		intent.Hash,
	); err != nil {
		return mapDatabaseError(err)
	}
	if err := insertAppSpecSnapshot(ctx, transaction, snapshot.AppSpec); err != nil {
		return err
	}
	if _, err := transaction.ExecContext(ctx, `
INSERT INTO goals(
    ref, request_ref, request_fingerprint, app_spec_ref, actor_ref, project_ref, state, revision,
    created_at, started_at, closed_at, plan_generation
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		snapshot.Ref,
		requestRef,
		requestFingerprint,
		snapshot.AppSpec.Ref,
		snapshot.ActorRef,
		snapshot.ProjectRef,
		string(snapshot.State),
		int64(snapshot.Revision),
		requiredTime(snapshot.CreatedAt),
		storedTime(snapshot.StartedAt),
		storedTime(snapshot.ClosedAt),
		int64(snapshot.PlanGeneration),
	); err != nil {
		return mapDatabaseError(err)
	}
	return nil
}

func insertAppSpecSnapshot(ctx context.Context, transaction *sql.Tx, snapshot goal.AppSpecSnapshot) error {
	_, err := transaction.ExecContext(ctx, `
INSERT INTO app_specs(
    ref, intent_ref, generation, parent_ref, parent_hash, objective, reason,
    confirmed_by, confirmed_at, hash
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		snapshot.Ref,
		snapshot.Intent.Ref,
		int64(snapshot.Generation),
		nullableString(snapshot.ParentRef),
		nullableString(snapshot.ParentHash),
		snapshot.Objective,
		snapshot.Reason,
		snapshot.ConfirmedBy,
		requiredTime(snapshot.ConfirmedAt),
		snapshot.Hash,
	)
	return mapDatabaseError(err)
}

func insertExecution(ctx context.Context, transaction *sql.Tx, execution application.ExecutionRecord) error {
	_, err := transaction.ExecContext(ctx, `
INSERT INTO executions(
    ref, goal_ref, work_item_ref, state, artifact_media_type, idempotency_key,
    max_output_bytes, max_attempts, provider_ref, external_ref, created_at,
    deadline_at, started_at, provider_accepted_at, last_observed_at,
    provider_observed_at, finished_at, failure_code
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		execution.Ref.String(),
		execution.GoalRef.String(),
		execution.WorkItemRef.String(),
		string(execution.State),
		execution.ArtifactMediaType,
		execution.IdempotencyKey,
		execution.MaxOutputBytes,
		int64(execution.MaxAttempts),
		execution.ProviderRef,
		execution.ExternalRef,
		requiredTime(execution.CreatedAt),
		storedTime(execution.DeadlineAt),
		storedTime(execution.StartedAt),
		storedTime(execution.ProviderAcceptedAt),
		storedTime(execution.LastObservedAt),
		storedTime(execution.ProviderObservedAt),
		storedTime(execution.FinishedAt),
		execution.FailureCode,
	)
	return mapDatabaseError(err)
}

func updateExecutionCAS(
	ctx context.Context,
	transaction *sql.Tx,
	execution application.ExecutionRecord,
	expected application.ExecutionState,
) error {
	result, err := transaction.ExecContext(ctx, `
UPDATE executions
SET state = ?, artifact_media_type = ?, idempotency_key = ?, max_output_bytes = ?,
    max_attempts = ?, provider_ref = ?, external_ref = ?, created_at = ?,
    deadline_at = ?, started_at = ?, provider_accepted_at = ?, last_observed_at = ?,
    provider_observed_at = ?, finished_at = ?, failure_code = ?
WHERE ref = ? AND goal_ref = ? AND work_item_ref = ? AND state = ?`,
		string(execution.State),
		execution.ArtifactMediaType,
		execution.IdempotencyKey,
		execution.MaxOutputBytes,
		int64(execution.MaxAttempts),
		execution.ProviderRef,
		execution.ExternalRef,
		requiredTime(execution.CreatedAt),
		storedTime(execution.DeadlineAt),
		storedTime(execution.StartedAt),
		storedTime(execution.ProviderAcceptedAt),
		storedTime(execution.LastObservedAt),
		storedTime(execution.ProviderObservedAt),
		storedTime(execution.FinishedAt),
		execution.FailureCode,
		execution.Ref.String(),
		execution.GoalRef.String(),
		execution.WorkItemRef.String(),
		string(expected),
	)
	if err != nil {
		return mapDatabaseError(err)
	}
	return requireOneRow(result)
}

func updateGoalCAS(
	ctx context.Context,
	transaction *sql.Tx,
	aggregate goal.Goal,
	expected goal.Revision,
) error {
	snapshot := aggregate.Snapshot()
	result, err := transaction.ExecContext(ctx, `
UPDATE goals
SET state = ?, revision = ?, started_at = ?, closed_at = ?
WHERE ref = ? AND revision = ?`,
		string(snapshot.State),
		int64(snapshot.Revision),
		storedTime(snapshot.StartedAt),
		storedTime(snapshot.ClosedAt),
		snapshot.Ref,
		int64(expected),
	)
	if err != nil {
		return mapDatabaseError(err)
	}
	return requireOneRow(result)
}

func updateGoalWorkItems(ctx context.Context, transaction *sql.Tx, aggregate goal.Goal) error {
	for _, item := range aggregate.WorkItems() {
		if err := updateWorkItemSnapshot(ctx, transaction, itemSnapshot(item)); err != nil {
			return err
		}
	}
	return nil
}

func updateWorkItemSnapshot(ctx context.Context, transaction *sql.Tx, snapshot goal.WorkItemSnapshot) error {
	result, err := transaction.ExecContext(ctx, `
UPDATE work_items
SET state = ?, revision = ?, started_at = ?, finished_at = ?, execution_ref = ?, skip_reason = ?
WHERE ref = ? AND goal_ref = ?`,
		string(snapshot.State), int64(snapshot.Revision), storedTime(snapshot.StartedAt),
		storedTime(snapshot.FinishedAt), nullableString(snapshot.ExecutionRef), string(snapshot.SkipReason),
		snapshot.Ref, snapshot.GoalRef,
	)
	if err != nil {
		return mapDatabaseError(err)
	}
	return requireOneRow(result)
}

func updateWorkItemCAS(
	ctx context.Context,
	transaction *sql.Tx,
	item goal.WorkItem,
	expected goal.Revision,
) error {
	snapshot := itemSnapshot(item)
	result, err := transaction.ExecContext(ctx, `
UPDATE work_items
SET state = ?, revision = ?, started_at = ?, finished_at = ?, execution_ref = ?
WHERE ref = ? AND goal_ref = ? AND revision = ?`,
		string(snapshot.State),
		int64(snapshot.Revision),
		storedTime(snapshot.StartedAt),
		storedTime(snapshot.FinishedAt),
		nullableString(snapshot.ExecutionRef),
		snapshot.Ref,
		snapshot.GoalRef,
		int64(expected),
	)
	if err != nil {
		return mapDatabaseError(err)
	}
	return requireOneRow(result)
}

func itemSnapshot(item goal.WorkItem) goal.WorkItemSnapshot {
	startedAt, _ := item.StartedAt()
	finishedAt, _ := item.FinishedAt()
	executionRef, hasExecution := item.Execution()
	var executionValue string
	if hasExecution {
		executionValue = executionRef.String()
	}
	return goal.WorkItemSnapshot{
		Ref:            item.Ref().String(),
		GoalRef:        item.Goal().String(),
		ActorRef:       item.Actor().String(),
		ProjectRef:     item.Project().String(),
		Objective:      item.Objective(),
		PhaseKey:       item.Phase().String(),
		RoleKey:        item.Role().String(),
		OutputContract: item.OutputContract().Kind(),
		SkipReason: func() goal.WorkItemSkipReason {
			reason, _ := item.SkipReason()
			return reason
		}(),
		DependencyRefs: workItemRefStrings(item.Dependencies()),
		WriteSet:       writeScopeStrings(item.WriteSet()),
		State:          item.State(),
		Revision:       item.Revision(),
		CreatedAt:      item.CreatedAt(),
		StartedAt:      startedAt,
		FinishedAt:     finishedAt,
		ExecutionRef:   executionValue,
	}
}

func workItemRefStrings(refs []goal.WorkItemRef) []string {
	values := make([]string, len(refs))
	for index, ref := range refs {
		values[index] = ref.String()
	}
	return values
}

func writeScopeStrings(scopes []goal.WriteScope) []string {
	values := make([]string, len(scopes))
	for index, scope := range scopes {
		values[index] = scope.String()
	}
	return values
}

func insertAction(ctx context.Context, transaction *sql.Tx, action application.ActionRecord) error {
	_, err := transaction.ExecContext(ctx, `
INSERT INTO outbox(ref, kind, goal_ref, work_item_ref, execution_ref, available_at)
VALUES (?, ?, ?, ?, ?, ?)`,
		action.Ref,
		string(action.Kind),
		action.GoalRef.String(),
		action.WorkItemRef.String(),
		action.ExecutionRef.String(),
		requiredTime(action.AvailableAt),
	)
	return mapDatabaseError(err)
}

func insertEvent(ctx context.Context, transaction *sql.Tx, event application.EventRecord) error {
	_, err := transaction.ExecContext(ctx, `
INSERT INTO events(ref, kind, goal_ref, work_item_ref, execution_ref, occurred_at)
VALUES (?, ?, ?, ?, ?, ?)`,
		event.Ref,
		event.Kind,
		event.GoalRef.String(),
		nullableString(event.WorkItemRef.String()),
		nullableString(event.ExecutionRef.String()),
		requiredTime(event.OccurredAt),
	)
	return mapDatabaseError(err)
}

func insertEvents(ctx context.Context, transaction *sql.Tx, events []application.EventRecord) error {
	for _, event := range events {
		if err := insertEvent(ctx, transaction, event); err != nil {
			return err
		}
	}
	return nil
}

func insertArtifact(ctx context.Context, transaction *sql.Tx, artifact application.ArtifactRecord) error {
	_, err := transaction.ExecContext(ctx, `
INSERT INTO artifacts(ref, goal_ref, work_item_ref, digest, media_type, size, created_at)
VALUES (?, ?, ?, ?, ?, ?, ?)`,
		artifact.Stored.Ref.String(),
		artifact.GoalRef.String(),
		artifact.WorkItemRef.String(),
		artifact.Stored.Digest,
		artifact.Stored.MediaType,
		artifact.Stored.Size,
		requiredTime(artifact.CreatedAt),
	)
	return mapDatabaseError(err)
}

func insertAttestation(ctx context.Context, transaction *sql.Tx, attestation application.AttestationRecord) error {
	_, err := transaction.ExecContext(ctx, `
INSERT INTO attestations(
    ref, goal_ref, work_item_ref, execution_ref, artifact_ref, policy, accepted_at
) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		attestation.Ref.String(),
		attestation.GoalRef.String(),
		attestation.WorkItemRef.String(),
		attestation.ExecutionRef.String(),
		attestation.ArtifactRef.String(),
		attestation.Policy,
		requiredTime(attestation.AcceptedAt),
	)
	return mapDatabaseError(err)
}

func requireOneRow(result sql.Result) error {
	rows, err := result.RowsAffected()
	if err != nil {
		return mapDatabaseError(err)
	}
	if rows != 1 {
		return conflict(fmt.Errorf("sqlite.cas_conflict"))
	}
	return nil
}

func nullableString(value string) any {
	if value == "" {
		return nil
	}
	return value
}
