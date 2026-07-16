package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"orquesta/internal/application"
	"orquesta/internal/goal"
	"orquesta/internal/identity"
)

func validateRecoveryV14CancelCoverage(
	ctx context.Context,
	transaction *sql.Tx,
	control application.ControlRecord,
) error {
	var uncovered int
	if err := transaction.QueryRowContext(ctx, `
SELECT COUNT(*)
FROM executions execution
WHERE execution.goal_ref = ?
  AND execution.state IN ('dispatching', 'running')
  AND (? = 'goal' OR execution.work_item_ref = ?)
  AND NOT EXISTS (
      SELECT 1 FROM outbox action
      WHERE action.kind = 'stop_agent' AND action.control_ref = ?
        AND action.ref = 'action:stop:' || ? || ':' || execution.ref
        AND action.goal_ref = execution.goal_ref
        AND action.work_item_ref = execution.work_item_ref
        AND action.execution_ref = execution.ref
        AND action.plan_generation = execution.plan_generation
  )`,
		control.GoalRef.String(), string(control.Target), control.WorkItemRef.String(),
		control.Ref, control.Ref,
	).Scan(&uncovered); err != nil {
		return err
	}
	if uncovered != 0 {
		return fmt.Errorf("sqlite.recovery_control_cancel_coverage_invalid:%s", control.Ref)
	}
	if control.Status == application.ControlRequested {
		return nil
	}
	var unresolved int
	if err := transaction.QueryRowContext(ctx, `
SELECT COUNT(*) FROM outbox
WHERE kind = 'stop_agent' AND control_ref = ? AND completed_at IS NULL`, control.Ref,
	).Scan(&unresolved); err != nil {
		return err
	}
	if unresolved != 0 {
		return fmt.Errorf("sqlite.recovery_control_cancel_confirmation_invalid:%s", control.Ref)
	}
	var state string
	var finishedAt sql.NullInt64
	switch control.Target {
	case application.ControlTargetGoal:
		if err := transaction.QueryRowContext(ctx, `
SELECT state, closed_at FROM goals WHERE ref = ?`, control.GoalRef.String(),
		).Scan(&state, &finishedAt); err != nil {
			return err
		}
		if state != string(goal.GoalStateCanceled) {
			return fmt.Errorf("sqlite.recovery_control_cancel_confirmation_invalid:%s", control.Ref)
		}
	case application.ControlTargetWorkItem:
		if err := transaction.QueryRowContext(ctx, `
SELECT state, finished_at FROM work_items WHERE goal_ref = ? AND ref = ?`,
			control.GoalRef.String(), control.WorkItemRef.String(),
		).Scan(&state, &finishedAt); err != nil {
			return err
		}
		if state != string(goal.WorkItemStateCanceled) {
			return fmt.Errorf("sqlite.recovery_control_cancel_confirmation_invalid:%s", control.Ref)
		}
	}
	if !finishedAt.Valid || finishedAt.Int64 != requiredTime(control.ConfirmedAt) {
		return fmt.Errorf("sqlite.recovery_control_cancel_confirmation_invalid:%s", control.Ref)
	}
	return nil
}

func readRecoveryV14Controls(
	ctx context.Context,
	transaction *sql.Tx,
) (map[string]application.ControlRecord, error) {
	rows, err := transaction.QueryContext(ctx, `
SELECT principal_ref, project_ref, request_ref FROM controls ORDER BY ref`)
	if err != nil {
		return nil, err
	}
	type controlKey struct{ principal, project, request string }
	var keys []controlKey
	for rows.Next() {
		var key controlKey
		if err := rows.Scan(&key.principal, &key.project, &key.request); err != nil {
			_ = rows.Close()
			return nil, err
		}
		keys = append(keys, key)
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return nil, err
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}

	controls := make(map[string]application.ControlRecord, len(keys))
	for _, key := range keys {
		principal, err := identity.NewPrincipalRef(key.principal)
		if err != nil {
			return nil, err
		}
		project, err := goal.NewProjectRef(key.project)
		if err != nil {
			return nil, err
		}
		record, found, err := readControlByRequest(ctx, transaction, principal, project, key.request)
		if err != nil || !found {
			if err == nil {
				err = errors.New("sqlite.recovery_control_missing")
			}
			return nil, err
		}
		if err := application.ValidatePersistedControlRecord(record); err != nil {
			return nil, fmt.Errorf("sqlite.recovery_control_record_invalid:%s:%w", record.Ref, err)
		}
		if _, duplicate := controls[record.Ref]; duplicate {
			return nil, errors.New("sqlite.recovery_control_duplicate")
		}
		if err := validateRecoveryV14ControlBinding(ctx, transaction, record); err != nil {
			return nil, err
		}
		controls[record.Ref] = record
	}
	return controls, nil
}

func validateRecoveryV14ControlBinding(
	ctx context.Context,
	transaction *sql.Tx,
	record application.ControlRecord,
) error {
	var projectValue, specHash string
	var goalRevision, planGeneration, appSpecGeneration, goalCreatedAt int64
	err := transaction.QueryRowContext(ctx, `
SELECT goal.project_ref, goal.revision, goal.plan_generation, goal.created_at,
       spec.generation, spec.hash
FROM goals goal
JOIN app_specs spec ON spec.ref = goal.app_spec_ref
WHERE goal.ref = ?`, record.GoalRef.String()).Scan(
		&projectValue, &goalRevision, &planGeneration, &goalCreatedAt,
		&appSpecGeneration, &specHash,
	)
	if err != nil {
		return err
	}
	if projectValue != record.ProjectRef.String() || goalRevision < int64(record.GoalRevision) ||
		planGeneration < int64(record.PlanGeneration) || appSpecGeneration != int64(record.AppSpecGeneration) ||
		specHash != record.SpecHash || record.RequestedAt.Before(time.Unix(0, goalCreatedAt).UTC()) {
		return fmt.Errorf("sqlite.recovery_control_goal_binding_invalid:%s", record.Ref)
	}

	if record.WorkItemRef.String() != "" {
		var itemRevision, itemCreatedAt int64
		if err := transaction.QueryRowContext(ctx, `
SELECT revision, created_at FROM work_items WHERE goal_ref = ? AND ref = ?`,
			record.GoalRef.String(), record.WorkItemRef.String(),
		).Scan(&itemRevision, &itemCreatedAt); err != nil {
			return err
		}
		if itemRevision < int64(record.WorkItemRevision) ||
			record.RequestedAt.Before(time.Unix(0, itemCreatedAt).UTC()) {
			return fmt.Errorf("sqlite.recovery_control_work_item_binding_invalid:%s", record.Ref)
		}
	}
	if record.ExecutionRef.String() != "" {
		var workItemValue, executionSpecHash string
		var attempt, executionPlan, executionAppSpec, executionCreatedAt int64
		if err := transaction.QueryRowContext(ctx, `
SELECT work_item_ref, attempt_no, plan_generation, app_spec_generation, spec_hash, created_at
FROM executions WHERE goal_ref = ? AND ref = ?`,
			record.GoalRef.String(), record.ExecutionRef.String(),
		).Scan(
			&workItemValue, &attempt, &executionPlan, &executionAppSpec,
			&executionSpecHash, &executionCreatedAt,
		); err != nil {
			return err
		}
		if workItemValue != record.WorkItemRef.String() || attempt != int64(record.ExecutionAttempt) ||
			executionPlan != int64(record.PlanGeneration) || executionAppSpec != int64(record.AppSpecGeneration) ||
			executionSpecHash != record.SpecHash ||
			record.RequestedAt.Before(time.Unix(0, executionCreatedAt).UTC()) {
			return fmt.Errorf("sqlite.recovery_control_execution_binding_invalid:%s", record.Ref)
		}
	}

	var events int
	if err := transaction.QueryRowContext(ctx, `
SELECT COUNT(*) FROM events
WHERE ref = ? AND kind = ? AND goal_ref = ?
  AND work_item_ref IS ? AND execution_ref IS ? AND occurred_at = ?`,
		"event:control:"+record.Ref, "control."+string(record.Operation), record.GoalRef.String(),
		nullableString(record.WorkItemRef.String()), nullableString(record.ExecutionRef.String()),
		requiredTime(record.RequestedAt),
	).Scan(&events); err != nil {
		return err
	}
	if events != 1 {
		return fmt.Errorf("sqlite.recovery_control_event_invalid:%s", record.Ref)
	}
	return nil
}
