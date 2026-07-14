package sqlite

import (
	"context"
	"database/sql"
	"fmt"

	"orquesta/internal/application"
	"orquesta/internal/goal"
)

func insertCreateState(ctx context.Context, transaction *sql.Tx, state application.CreateGoalState) error {
	intent := state.Intent.Snapshot()
	snapshot := state.Goal.Snapshot()
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
	if _, err := transaction.ExecContext(ctx, `
INSERT INTO goals(
    ref, request_ref, request_fingerprint, intent_ref, actor_ref, project_ref, state, revision,
    created_at, started_at, closed_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		snapshot.Ref,
		state.RequestRef,
		state.RequestFingerprint,
		snapshot.Intent.Ref,
		snapshot.ActorRef,
		snapshot.ProjectRef,
		string(snapshot.State),
		int64(snapshot.Revision),
		requiredTime(snapshot.CreatedAt),
		storedTime(snapshot.StartedAt),
		storedTime(snapshot.ClosedAt),
	); err != nil {
		return mapDatabaseError(err)
	}
	for position, item := range snapshot.WorkItems {
		if _, err := transaction.ExecContext(ctx, `
INSERT INTO work_items(
    ref, goal_ref, actor_ref, project_ref, objective, state, revision, position,
    created_at, started_at, finished_at, execution_ref
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			item.Ref,
			item.GoalRef,
			item.ActorRef,
			item.ProjectRef,
			item.Objective,
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
	}
	if err := insertExecution(ctx, transaction, state.Execution); err != nil {
		return err
	}
	if err := insertAction(ctx, transaction, state.Action); err != nil {
		return err
	}
	return insertEvent(ctx, transaction, state.Event)
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
		requiredTime(execution.DeadlineAt),
		storedTime(execution.StartedAt),
		storedTime(execution.ProviderAcceptedAt),
		storedTime(execution.LastObservedAt),
		storedTime(execution.ProviderObservedAt),
		storedTime(execution.FinishedAt),
		execution.FailureCode,
	)
	return mapDatabaseError(err)
}

func updateExecution(ctx context.Context, transaction *sql.Tx, execution application.ExecutionRecord) error {
	result, err := transaction.ExecContext(ctx, `
UPDATE executions
SET state = ?, artifact_media_type = ?, idempotency_key = ?, max_output_bytes = ?,
    max_attempts = ?, provider_ref = ?, external_ref = ?, created_at = ?,
    deadline_at = ?, started_at = ?, provider_accepted_at = ?, last_observed_at = ?,
    provider_observed_at = ?, finished_at = ?, failure_code = ?
WHERE ref = ? AND goal_ref = ? AND work_item_ref = ?`,
		string(execution.State),
		execution.ArtifactMediaType,
		execution.IdempotencyKey,
		execution.MaxOutputBytes,
		int64(execution.MaxAttempts),
		execution.ProviderRef,
		execution.ExternalRef,
		requiredTime(execution.CreatedAt),
		requiredTime(execution.DeadlineAt),
		storedTime(execution.StartedAt),
		storedTime(execution.ProviderAcceptedAt),
		storedTime(execution.LastObservedAt),
		storedTime(execution.ProviderObservedAt),
		storedTime(execution.FinishedAt),
		execution.FailureCode,
		execution.Ref.String(),
		execution.GoalRef.String(),
		execution.WorkItemRef.String(),
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
		Ref:          item.Ref().String(),
		GoalRef:      item.Goal().String(),
		ActorRef:     item.Actor().String(),
		ProjectRef:   item.Project().String(),
		Objective:    item.Objective(),
		State:        item.State(),
		Revision:     item.Revision(),
		CreatedAt:    item.CreatedAt(),
		StartedAt:    startedAt,
		FinishedAt:   finishedAt,
		ExecutionRef: executionValue,
	}
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
		event.WorkItemRef.String(),
		event.ExecutionRef.String(),
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
