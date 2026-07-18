package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"orquesta/internal/application"
	"orquesta/internal/ports"
)

func validateRecoveryV14StopSupersessions(
	ctx context.Context,
	transaction *sql.Tx,
	controls map[string]application.ControlRecord,
) error {
	_, statusColumn, timeColumn, effectJoin, err := recoveryV14EffectProjection(ctx, transaction)
	if err != nil {
		return err
	}
	pairs := 0
	for oldRef, old := range controls {
		if old.Status != application.ControlSuperseded {
			if old.SupersededByControlRef != "" {
				return fmt.Errorf("sqlite.recovery_control_supersession_orphan:%s", oldRef)
			}
			continue
		}
		next, found := controls[old.SupersededByControlRef]
		if !found || next.SupersedesControlRef != old.Ref ||
			next.Status == application.ControlSuperseded ||
			old.Operation != application.ControlStop || next.Operation != application.ControlStop ||
			old.Target != application.ControlTargetExecution || next.Target != application.ControlTargetExecution ||
			old.Mode != ports.AgentStopCooperative || next.Mode != ports.AgentStopForced ||
			old.ProjectRef != next.ProjectRef || old.GoalRef != next.GoalRef ||
			old.WorkItemRef != next.WorkItemRef || old.WorkItemRevision > next.WorkItemRevision ||
			old.ExecutionRef != next.ExecutionRef || old.ExecutionAttempt != next.ExecutionAttempt ||
			old.GoalRevision >= next.GoalRevision || old.PlanGeneration != next.PlanGeneration ||
			old.AppSpecGeneration != next.AppSpecGeneration || old.SpecHash != next.SpecHash ||
			!old.SupersededAt.Equal(next.RequestedAt) || old.RequestedAt.After(next.RequestedAt) {
			return fmt.Errorf("sqlite.recovery_control_supersession_lineage_invalid:%s", oldRef)
		}
		pairs++

		var eventCount int
		if err := transaction.QueryRowContext(ctx, `
SELECT COUNT(*) FROM events
WHERE ref = ? AND kind = 'control.stop_superseded' AND goal_ref = ?
  AND work_item_ref = ? AND execution_ref = ? AND occurred_at = ?`,
			"event:control-superseded:"+old.Ref, old.GoalRef.String(), old.WorkItemRef.String(),
			old.ExecutionRef.String(), requiredTime(next.RequestedAt),
		).Scan(&eventCount); err != nil {
			return err
		}
		if eventCount != 1 {
			return fmt.Errorf("sqlite.recovery_control_supersession_event_invalid:%s", oldRef)
		}

		var retirementCount int
		if err := transaction.QueryRowContext(ctx, `
SELECT COUNT(*)
FROM outbox action
JOIN action_consumption_receipts receipt ON receipt.action_ref = action.ref
`+effectJoin+`
WHERE action.ref = ? AND action.kind = 'stop_agent' AND action.control_ref = ?
  AND action.goal_ref = ? AND action.work_item_ref = ? AND action.execution_ref = ?
  AND action.plan_generation = ? AND action.work_item_generation = ?
  AND action.completed_at = ? AND action.retired_at IS NULL AND action.quarantined_at IS NULL
  AND action.last_error_code = 'application.action_retired'
  AND receipt.kind = 'stop_agent' AND receipt.goal_ref = action.goal_ref
  AND receipt.work_item_ref = action.work_item_ref AND receipt.execution_ref = action.execution_ref
  AND receipt.plan_generation = action.plan_generation
  AND receipt.work_item_generation = action.work_item_generation
  AND receipt.outcome = 'completed' AND receipt.error_code = 'application.action_retired'
  AND receipt.consumed_at = ? AND receipt.effect_receipt_ref IS NULL
  AND `+statusColumn+` IS NULL AND `+timeColumn+` IS NULL`,
			"action:stop:"+old.Ref+":"+old.ExecutionRef.String(), old.Ref,
			old.GoalRef.String(), old.WorkItemRef.String(), old.ExecutionRef.String(),
			int64(old.PlanGeneration), int64(old.WorkItemRevision), requiredTime(next.RequestedAt),
			requiredTime(next.RequestedAt),
		).Scan(&retirementCount); err != nil {
			return err
		}
		if retirementCount != 1 {
			return fmt.Errorf("sqlite.recovery_control_supersession_retirement_invalid:%s", oldRef)
		}
	}
	return validateRecoveryV14StopSupersessionCardinality(ctx, transaction, pairs)
}

func validateRecoveryV14StopSupersessionCardinality(
	ctx context.Context,
	transaction *sql.Tx,
	pairs int,
) error {
	var forwardLinks, lineageEvents int
	if err := transaction.QueryRowContext(ctx, `
SELECT COUNT(*) FROM controls WHERE supersedes_control_ref IS NOT NULL`).Scan(&forwardLinks); err != nil {
		return err
	}
	if err := transaction.QueryRowContext(ctx, `
SELECT COUNT(*) FROM events WHERE kind = 'control.stop_superseded'`).Scan(&lineageEvents); err != nil {
		return err
	}
	if pairs != forwardLinks || pairs != lineageEvents {
		return errors.New("sqlite.recovery_control_supersession_cardinality_invalid")
	}
	return nil
}
