package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"orquesta/internal/application"
	"orquesta/internal/goal"
	"orquesta/internal/ports"
)

type recoveryV14StopActionRow struct {
	action              application.ActionRecord
	controlRef          string
	executionState      string
	executionSpecHash   string
	externalRef         string
	planGeneration      int64
	itemGeneration      int64
	executionAttempt    int64
	executionPlan       int64
	executionAppSpec    int64
	currentItemRevision int64
	deliveryAttempt     int64
	actionFence         int64
	token               sql.NullString
	worker              sql.NullString
	claimedUntil        sql.NullInt64
	completedAt         sql.NullInt64
	quarantinedAt       sql.NullInt64
	currentFence        sql.NullInt64
	executionFinishedAt sql.NullInt64
}

func validateRecoveryV14StopActions(
	ctx context.Context,
	transaction *sql.Tx,
	controls map[string]application.ControlRecord,
) error {
	rows, err := transaction.QueryContext(ctx, `
SELECT action.ref, action.goal_ref, action.work_item_ref, action.execution_ref,
       action.control_ref, action.plan_generation, action.work_item_generation,
       action.available_at, action.claim_token, action.claimed_by, action.claimed_until,
       action.delivery_attempt, action.fence, action.completed_at, action.quarantined_at,
       fence.fence, execution.state, execution.attempt_no, execution.plan_generation,
       execution.app_spec_generation, execution.spec_hash, execution.external_ref,
       execution.finished_at, item.revision
FROM outbox action
JOIN executions execution ON execution.goal_ref = action.goal_ref
 AND execution.work_item_ref = action.work_item_ref AND execution.ref = action.execution_ref
JOIN work_items item ON item.goal_ref = action.goal_ref AND item.ref = action.work_item_ref
LEFT JOIN work_item_fences fence ON fence.goal_ref = action.goal_ref
 AND fence.work_item_ref = action.work_item_ref
WHERE action.kind = 'stop_agent'
ORDER BY action.ref`)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		row, err := scanRecoveryV14StopAction(rows)
		if err != nil {
			return err
		}
		if err := validateRecoveryV14StopAction(row, controls); err != nil {
			return err
		}
	}
	return rows.Err()
}

func scanRecoveryV14StopAction(rows *sql.Rows) (recoveryV14StopActionRow, error) {
	var row recoveryV14StopActionRow
	var goalValue, itemValue, executionValue string
	var availableAt int64
	if err := rows.Scan(
		&row.action.Ref, &goalValue, &itemValue, &executionValue, &row.controlRef,
		&row.planGeneration, &row.itemGeneration, &availableAt, &row.token, &row.worker,
		&row.claimedUntil, &row.deliveryAttempt, &row.actionFence, &row.completedAt,
		&row.quarantinedAt, &row.currentFence, &row.executionState, &row.executionAttempt,
		&row.executionPlan, &row.executionAppSpec, &row.executionSpecHash, &row.externalRef,
		&row.executionFinishedAt, &row.currentItemRevision,
	); err != nil {
		return recoveryV14StopActionRow{}, err
	}
	var err error
	row.action.Kind, row.action.ControlRef = application.ActionStopAgent, row.controlRef
	if row.action.GoalRef, err = goal.NewGoalRef(goalValue); err != nil {
		return recoveryV14StopActionRow{}, err
	}
	if row.action.WorkItemRef, err = goal.NewWorkItemRef(itemValue); err != nil {
		return recoveryV14StopActionRow{}, err
	}
	if row.action.ExecutionRef, err = goal.NewExecutionRef(executionValue); err != nil {
		return recoveryV14StopActionRow{}, err
	}
	if row.planGeneration <= 0 || row.itemGeneration <= 0 ||
		row.deliveryAttempt < 0 || row.actionFence < 0 {
		return recoveryV14StopActionRow{}, errors.New("sqlite.recovery_stop_action_generation_invalid")
	}
	row.action.PlanGeneration = goal.PlanGeneration(row.planGeneration)
	row.action.WorkItemGeneration = goal.Revision(row.itemGeneration)
	row.action.AvailableAt = time.Unix(0, availableAt).UTC()
	if err := validateAction(row.action); err != nil {
		return recoveryV14StopActionRow{}, err
	}
	return row, nil
}

func validateRecoveryV14StopAction(
	row recoveryV14StopActionRow,
	controls map[string]application.ControlRecord,
) error {
	control, found := controls[row.controlRef]
	if !found {
		return fmt.Errorf("sqlite.recovery_stop_action_binding_invalid:%s", row.action.Ref)
	}
	if err := validateRecoveryV14StopActionBinding(row, control); err != nil {
		return err
	}
	if err := validateRecoveryV14StopActionExecution(row, control); err != nil {
		return err
	}
	if err := validateRecoveryV14StopActionTarget(row, control); err != nil {
		return err
	}
	return validateRecoveryV14StopActionClaim(row)
}

func validateRecoveryV14StopActionBinding(
	row recoveryV14StopActionRow,
	control application.ControlRecord,
) error {
	action := row.action
	if action.Ref != "action:stop:"+control.Ref+":"+action.ExecutionRef.String() ||
		action.GoalRef != control.GoalRef ||
		(control.Operation != application.ControlStop && control.Operation != application.ControlCancel) ||
		(control.Mode != ports.AgentStopCooperative && control.Mode != ports.AgentStopForced) ||
		row.planGeneration != row.executionPlan || row.planGeneration != int64(control.PlanGeneration) ||
		row.executionAppSpec != int64(control.AppSpecGeneration) || row.executionSpecHash != control.SpecHash ||
		row.itemGeneration > row.currentItemRevision || action.AvailableAt.Before(control.RequestedAt) {
		return fmt.Errorf("sqlite.recovery_stop_action_binding_invalid:%s", action.Ref)
	}
	return nil
}

func validateRecoveryV14StopActionTarget(
	row recoveryV14StopActionRow,
	control application.ControlRecord,
) error {
	action := row.action
	if control.Operation == application.ControlStop {
		if control.Target != application.ControlTargetExecution || control.WorkItemRef != action.WorkItemRef ||
			control.ExecutionRef != action.ExecutionRef || control.ExecutionAttempt != uint64(row.executionAttempt) ||
			action.WorkItemGeneration != control.WorkItemRevision {
			return fmt.Errorf("sqlite.recovery_stop_action_target_invalid:%s", action.Ref)
		}
	} else if control.Target != application.ControlTargetGoal &&
		(control.Target != application.ControlTargetWorkItem || control.WorkItemRef != action.WorkItemRef) {
		return fmt.Errorf("sqlite.recovery_stop_action_target_invalid:%s", action.Ref)
	}
	return nil
}

func validateRecoveryV14StopActionExecution(
	row recoveryV14StopActionRow,
	control application.ControlRecord,
) error {
	terminal := row.executionState == string(application.ExecutionSucceeded) ||
		row.executionState == string(application.ExecutionFailed) ||
		row.executionState == string(application.ExecutionCanceled) ||
		row.executionState == string(application.ExecutionStopped)
	if (row.executionState != string(application.ExecutionDispatching) &&
		row.executionState != string(application.ExecutionRunning) && !terminal) ||
		(terminal && (!row.executionFinishedAt.Valid ||
			row.executionFinishedAt.Int64 < requiredTime(control.RequestedAt))) {
		return fmt.Errorf("sqlite.recovery_stop_action_execution_invalid:%s", row.action.Ref)
	}
	return nil
}

func validateRecoveryV14StopActionClaim(row recoveryV14StopActionRow) error {
	claimed := row.token.Valid || row.worker.Valid || row.claimedUntil.Valid
	if claimed != (row.token.Valid && row.worker.Valid && row.claimedUntil.Valid &&
		row.deliveryAttempt > 0 && row.actionFence > 0) ||
		row.quarantinedAt.Valid && !row.completedAt.Valid ||
		row.currentFence.Valid && row.currentFence.Int64 < row.actionFence {
		return fmt.Errorf("sqlite.recovery_stop_action_claim_invalid:%s", row.action.Ref)
	}
	if !claimed {
		if row.completedAt.Valid {
			return fmt.Errorf("sqlite.recovery_stop_action_completion_unclaimed:%s", row.action.Ref)
		}
		return nil
	}
	claim := application.ActionClaim{
		Action: row.action, Token: row.token.String, WorkerRef: row.worker.String,
		DeliveryAttempt: uint64(row.deliveryAttempt), Fence: uint64(row.actionFence),
		LeaseUntil: time.Unix(0, row.claimedUntil.Int64).UTC(),
	}
	if err := validateClaim(claim); err != nil {
		return err
	}
	if !row.completedAt.Valid && !row.quarantinedAt.Valid &&
		(!row.currentFence.Valid || row.currentFence.Int64 != row.actionFence ||
			(row.executionState == string(application.ExecutionRunning) && row.externalRef == "") ||
			(row.executionState != string(application.ExecutionRunning) &&
				row.executionState != string(application.ExecutionSucceeded) &&
				row.executionState != string(application.ExecutionFailed) &&
				row.executionState != string(application.ExecutionCanceled) &&
				row.executionState != string(application.ExecutionStopped))) {
		return fmt.Errorf("sqlite.recovery_stop_action_live_claim_invalid:%s", row.action.Ref)
	}
	return nil
}
