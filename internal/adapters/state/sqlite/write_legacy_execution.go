package sqlite

import (
	"context"
	"database/sql"

	"orquesta/internal/application"
)

// updateLegacyExecutionCAS keeps recovery/migration tooling able to construct
// an exact V14 running frontier before migration 010 adds governance columns.
func updateLegacyExecutionCAS(
	ctx context.Context,
	tx *sql.Tx,
	execution application.ExecutionRecord,
	expected application.ExecutionState,
) error {
	if expected == application.ExecutionDispatching && execution.State == application.ExecutionRunning {
		return acceptLegacyExecutionCAS(ctx, tx, execution)
	}
	result, err := tx.ExecContext(ctx, `
UPDATE executions
SET state=?, deadline_at=?, started_at=?, provider_accepted_at=?, last_observed_at=?,
    provider_observed_at=?, finished_at=?, failure_code=?, recipient_mailbox_retired=?
WHERE ref=? AND goal_ref=? AND work_item_ref=? AND state=?
  AND provider_ref=? AND model_ref=? AND agent_ref=? AND external_ref=?
  AND attempt_no=? AND max_execution_attempts=? AND replaces_execution_ref IS ?
  AND plan_generation=? AND app_spec_generation=? AND spec_hash=?
  AND artifact_media_type=? AND idempotency_key=? AND max_output_bytes=? AND created_at=?`,
		string(execution.State), storedTime(execution.DeadlineAt), storedTime(execution.StartedAt),
		storedTime(execution.ProviderAcceptedAt), storedTime(execution.LastObservedAt),
		storedTime(execution.ProviderObservedAt), storedTime(execution.FinishedAt), execution.FailureCode,
		storedBool(execution.RecipientMailboxRetired), execution.Ref.String(), execution.GoalRef.String(),
		execution.WorkItemRef.String(), string(expected), execution.ProviderRef, execution.ModelRef,
		execution.AgentRef, execution.ExternalRef, int64(execution.AttemptNo), int64(execution.MaxExecutionAttempts),
		nullableString(execution.ReplacesExecutionRef.String()), int64(execution.PlanGeneration),
		int64(execution.AppSpecGeneration), execution.SpecHash, execution.ArtifactMediaType,
		execution.IdempotencyKey, execution.MaxOutputBytes, requiredTime(execution.CreatedAt),
	)
	if err != nil {
		return mapDatabaseError(err)
	}
	return requireOneRow(result)
}

func acceptLegacyExecutionCAS(ctx context.Context, tx *sql.Tx, execution application.ExecutionRecord) error {
	result, err := tx.ExecContext(ctx, `
UPDATE executions
SET state=?, provider_ref=?, model_ref=?, agent_ref=?, external_ref=?,
    deadline_at=?, started_at=?, provider_accepted_at=?, last_observed_at=?,
    provider_observed_at=?, finished_at=?, failure_code=?, recipient_mailbox_retired=?
WHERE ref=? AND goal_ref=? AND work_item_ref=? AND state='dispatching'
  AND provider_ref='' AND model_ref='' AND agent_ref='' AND external_ref=''
  AND attempt_no=? AND max_execution_attempts=? AND replaces_execution_ref IS ?
  AND plan_generation=? AND app_spec_generation=? AND spec_hash=?
  AND artifact_media_type=? AND idempotency_key=? AND max_output_bytes=? AND created_at=?`,
		string(execution.State), execution.ProviderRef, execution.ModelRef, execution.AgentRef,
		execution.ExternalRef, storedTime(execution.DeadlineAt), storedTime(execution.StartedAt),
		storedTime(execution.ProviderAcceptedAt), storedTime(execution.LastObservedAt),
		storedTime(execution.ProviderObservedAt), storedTime(execution.FinishedAt), execution.FailureCode,
		storedBool(execution.RecipientMailboxRetired), execution.Ref.String(), execution.GoalRef.String(),
		execution.WorkItemRef.String(), int64(execution.AttemptNo), int64(execution.MaxExecutionAttempts),
		nullableString(execution.ReplacesExecutionRef.String()), int64(execution.PlanGeneration),
		int64(execution.AppSpecGeneration), execution.SpecHash, execution.ArtifactMediaType,
		execution.IdempotencyKey, execution.MaxOutputBytes, requiredTime(execution.CreatedAt),
	)
	if err != nil {
		return mapDatabaseError(err)
	}
	return requireOneRow(result)
}
