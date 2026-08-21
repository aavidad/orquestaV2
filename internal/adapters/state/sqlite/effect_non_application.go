package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"orquesta/internal/application"
)

func insertEffectAttemptOutcome(ctx context.Context, tx *sql.Tx, state application.ActionRequeuedState) error {
	outcome := state.EffectAttemptOutcome
	if outcome == nil {
		return nil
	}
	if state.Claim.Action.Kind != application.ActionStopAgent || state.BudgetSettlement != nil ||
		state.ClearEffectBinding || !outcome.ObservedAt.Equal(state.OperationAt.UTC()) {
		return invalid(errors.New("sqlite.effect_attempt_outcome_requeue_invalid"))
	}
	attempt, found, err := readEffectAttemptByFence(ctx, tx, state.Claim.Action.Ref, outcome.ActionFence)
	if err != nil {
		return err
	}
	if !found || attempt.Ref != outcome.AttemptRef || attempt.ActionFence != state.Claim.Fence ||
		application.ValidateEffectAttemptOutcome(attempt, *outcome) != nil {
		return conflict(errors.New("sqlite.effect_attempt_outcome_attempt_conflict"))
	}
	var receipts, outcomes int
	if err := tx.QueryRowContext(ctx, `SELECT (SELECT COUNT(*) FROM effect_receipts WHERE attempt_ref=?),
 (SELECT COUNT(*) FROM effect_non_application_evidence WHERE attempt_ref=?)`,
		attempt.Ref, attempt.Ref).Scan(&receipts, &outcomes); err != nil {
		return mapDatabaseError(err)
	}
	if receipts != 0 || outcomes != 0 {
		return conflict(errors.New("sqlite.effect_attempt_outcome_already_resolved"))
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO effect_non_application_evidence(
 ref,attempt_ref,intent_ref,intent_digest,approval_ref,project_ref,goal_ref,work_item_ref,execution_ref,
 plan_generation,app_spec_generation,spec_hash,actor_ref,action_ref,action_fence,idempotency_key,outcome,observed_at
) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		outcome.Ref, outcome.AttemptRef, outcome.IntentRef, outcome.IntentDigest, outcome.ApprovalRef,
		outcome.Subject.ProjectRef.String(), outcome.Subject.GoalRef.String(), outcome.Subject.WorkItemRef.String(), outcome.Subject.ExecutionRef.String(),
		int64(outcome.Subject.PlanGeneration), int64(outcome.Subject.AppSpecGeneration),
		outcome.Subject.SpecHash, outcome.Subject.ActorRef.String(), outcome.ActionRef, int64(outcome.ActionFence),
		outcome.IdempotencyKey, string(outcome.Outcome), requiredTime(outcome.ObservedAt),
	)
	return mapDatabaseError(err)
}

func readEffectAttemptOutcomesForGoal(ctx context.Context, source queryer, goalRef string) ([]application.EffectAttemptOutcome, error) {
	persisted, err := sqliteTableHasColumn(ctx, source, "effect_non_application_evidence", "attempt_ref")
	if err != nil {
		return nil, mapDatabaseError(err)
	}
	if !persisted {
		return nil, nil
	}
	rows, err := source.QueryContext(ctx, `SELECT ref,attempt_ref,intent_ref,intent_digest,approval_ref,project_ref,goal_ref,
 work_item_ref,execution_ref,plan_generation,app_spec_generation,spec_hash,actor_ref,action_ref,action_fence,idempotency_key,outcome,observed_at
FROM effect_non_application_evidence WHERE goal_ref=? ORDER BY observed_at,ref`, goalRef)
	if err != nil {
		return nil, mapDatabaseError(err)
	}
	defer rows.Close()
	var result []application.EffectAttemptOutcome
	for rows.Next() {
		var value application.EffectAttemptOutcome
		var projectRef, storedGoalRef, workRef, executionRef, actorRef, outcome string
		var plan, appSpec, fence, observed int64
		if err := rows.Scan(&value.Ref, &value.AttemptRef, &value.IntentRef, &value.IntentDigest, &value.ApprovalRef, &projectRef,
			&storedGoalRef, &workRef, &executionRef, &plan, &appSpec, &value.Subject.SpecHash, &actorRef, &value.ActionRef, &fence,
			&value.IdempotencyKey, &outcome, &observed); err != nil {
			return nil, mapDatabaseError(err)
		}
		if err := restoreEffectSubject(&value.Subject, projectRef, storedGoalRef, workRef,
			executionRef, plan, appSpec, actorRef); err != nil {
			return nil, err
		}
		if fence <= 0 {
			return nil, invalid(errors.New("sqlite.effect_attempt_outcome_fence_invalid"))
		}
		value.ActionFence = uint64(fence)
		value.Outcome = application.EffectAttemptOutcomeKind(outcome)
		value.ObservedAt = time.Unix(0, observed).UTC()
		result = append(result, value)
	}
	return result, mapDatabaseError(rows.Err())
}
