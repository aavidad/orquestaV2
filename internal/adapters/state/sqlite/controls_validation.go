package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"reflect"

	"orquesta/internal/application"
	"orquesta/internal/goal"
	"orquesta/internal/ports"
)

func supersededStopActionStatus(
	ctx context.Context,
	transaction *sql.Tx,
	control application.ControlRecord,
) (string, error) {
	var status string
	err := transaction.QueryRowContext(ctx, `SELECT COALESCE((SELECT CASE WHEN completed_at IS NULL AND retired_at IS NULL AND quarantined_at IS NULL THEN 'active'
 WHEN quarantined_at IS NOT NULL AND EXISTS (SELECT 1 FROM action_consumption_receipts consumed WHERE consumed.action_ref=outbox.ref AND consumed.outcome='quarantined' AND consumed.error_code='application.effect_unknown_applied') THEN 'quarantined' ELSE '' END
 FROM outbox WHERE ref=? AND kind='stop_agent' AND control_ref=? AND goal_ref=? AND work_item_ref=? AND execution_ref=? AND plan_generation=? AND work_item_generation=?),'')`,
		"action:stop:"+control.Ref+":"+control.ExecutionRef.String(), control.Ref,
		control.GoalRef.String(), control.WorkItemRef.String(), control.ExecutionRef.String(),
		int64(control.PlanGeneration), int64(control.WorkItemRevision),
	).Scan(&status)
	if err != nil {
		return "", mapDatabaseError(err)
	}
	return status, nil
}

func validateApplyControlState(state application.ApplyControlState) error {
	control := state.Control
	if !validText(state.RequestRef) || !validText(state.RequestFingerprint) ||
		state.AuthorizationReceipt.Ref() == "" || state.PrincipalRef.String() == "" ||
		state.ProjectRef.String() == "" || state.GoalRef.String() == "" ||
		state.ExpectedGoalRevision == 0 || state.ExpectedPlanGeneration == 0 || state.OperationAt.IsZero() ||
		!validText(control.Ref) || control.RequestRef != state.RequestRef ||
		control.RequestFingerprint != state.RequestFingerprint || control.PrincipalRef != state.PrincipalRef ||
		control.ProjectRef != state.ProjectRef || control.GoalRef != state.GoalRef || !validText(control.Reason) ||
		control.AuthorizationReceipt.Ref() != state.AuthorizationReceipt.Ref() ||
		control.PlanGeneration != state.ExpectedPlanGeneration {
		return errors.New("sqlite.control_state_invalid")
	}
	if state.ExpectedControlStatus == "" && control.GoalRevision != state.ExpectedGoalRevision {
		return errors.New("sqlite.control_state_invalid")
	}
	if err := application.ValidatePersistedControlRecord(control); err != nil {
		return errors.New("sqlite.control_state_invalid")
	}
	if _, err := goal.RestoreGoal(state.Goal.Snapshot()); err != nil {
		return err
	}
	if state.Claim.Action.Ref != "" {
		if err := validateClaim(state.Claim); err != nil || state.Claim.Action.GoalRef != state.GoalRef {
			return errors.New("sqlite.control_claim_invalid")
		}
	}
	if state.ClaimErrorCode != "" &&
		(state.ClaimErrorCode != "agent.launch_definitely_not_applied" ||
			state.Claim.Action.Kind != application.ActionLaunchAgent || state.EffectReceipt != nil ||
			!exactReleasedClaimBudget(state.Claim, state.BudgetSettlement)) {
		return errors.New("sqlite.control_claim_settlement_invalid")
	}
	if state.ExpectedControlStatus != "" && state.ExpectedControlStatus != application.ControlRequested {
		return errors.New("sqlite.control_status_invalid")
	}
	if control.Status != application.ControlRequested && control.Status != application.ControlConfirmed {
		return errors.New("sqlite.control_status_invalid")
	}
	for _, cleanup := range state.NewControls {
		if !application.IsRoundCleanupControl(cleanup) || cleanup.Status != application.ControlRequested ||
			cleanup.GoalRef != state.GoalRef || cleanup.ProjectRef != state.ProjectRef {
			return errors.New("sqlite.control_cleanup_invalid")
		}
	}
	return validateControlSupersessionState(state)
}

func validateControlSupersessionState(state application.ApplyControlState) error {
	if state.SupersededControl == nil {
		if state.ExpectedControlStatus == "" && state.Control.SupersedesControlRef != "" {
			return errors.New("sqlite.control_supersession_state_invalid")
		}
		return nil
	}
	old, next := *state.SupersededControl, state.Control
	wantOldAction := "action:stop:" + old.Ref + ":" + old.ExecutionRef.String()
	wantNewAction := "action:stop:" + next.Ref + ":" + next.ExecutionRef.String()
	newAction := application.ActionRecord{}
	if len(state.NewActions) == 1 {
		newAction = state.NewActions[0]
	}
	if state.ExpectedControlStatus != "" || next.Status != application.ControlRequested ||
		next.Operation != application.ControlStop || next.Target != application.ControlTargetExecution ||
		next.Mode != ports.AgentStopForced || next.SupersedesControlRef != old.Ref ||
		old.Status != application.ControlSuperseded || old.Operation != application.ControlStop ||
		old.Target != application.ControlTargetExecution || old.Mode != ports.AgentStopCooperative ||
		old.SupersededByControlRef != next.Ref || !old.SupersededAt.Equal(state.OperationAt) ||
		old.GoalRef != next.GoalRef || old.WorkItemRef != next.WorkItemRef ||
		old.WorkItemRevision > next.WorkItemRevision || old.ExecutionRef != next.ExecutionRef ||
		old.ExecutionAttempt != next.ExecutionAttempt || old.PlanGeneration != next.PlanGeneration ||
		old.AppSpecGeneration != next.AppSpecGeneration || old.SpecHash != next.SpecHash ||
		old.GoalRevision >= next.GoalRevision || old.RequestedAt.After(next.RequestedAt) ||
		!next.RequestedAt.Equal(state.OperationAt) ||
		len(state.RetireActionRefs) != 1 || state.RetireActionRefs[0] != wantOldAction ||
		len(state.NewActions) != 1 || newAction.Ref != wantNewAction ||
		newAction.Kind != application.ActionStopAgent || newAction.ControlRef != next.Ref ||
		newAction.GoalRef != next.GoalRef || newAction.WorkItemRef != next.WorkItemRef ||
		newAction.ExecutionRef != next.ExecutionRef || newAction.PlanGeneration != next.PlanGeneration ||
		newAction.WorkItemGeneration != next.WorkItemRevision || !newAction.AvailableAt.Equal(state.OperationAt) ||
		state.Claim.Action.Ref != "" || state.EffectReceipt != nil || state.BudgetSettlement != nil {
		return errors.New("sqlite.control_supersession_state_invalid")
	}
	if err := validateControlSupersessionEvent(state, old); err != nil {
		return err
	}
	if err := application.ValidatePersistedControlRecord(old); err != nil {
		return errors.New("sqlite.control_supersession_state_invalid")
	}
	return nil
}

func validateControlSupersessionEvent(
	state application.ApplyControlState,
	old application.ControlRecord,
) error {
	lineageEvents := 0
	for _, event := range state.Events {
		if event.Ref != "event:control-superseded:"+old.Ref {
			continue
		}
		lineageEvents++
		if event.Kind != "control.stop_superseded" || event.GoalRef != old.GoalRef ||
			event.WorkItemRef != old.WorkItemRef || event.ExecutionRef != old.ExecutionRef ||
			!event.OccurredAt.Equal(state.OperationAt) {
			return errors.New("sqlite.control_supersession_state_invalid")
		}
	}
	if lineageEvents != 1 {
		return errors.New("sqlite.control_supersession_state_invalid")
	}
	return nil
}

func validateControlTransition(state application.ApplyControlState, current application.GoalRecord) error {
	before, after := current.Goal.Snapshot(), state.Goal.Snapshot()
	if current.Goal.Project() != state.ProjectRef || current.Goal.Revision() != state.ExpectedGoalRevision ||
		current.Goal.PlanGeneration() != state.ExpectedPlanGeneration || before.Ref != after.Ref ||
		!reflect.DeepEqual(before.AppSpec, after.AppSpec) || before.ActorRef != after.ActorRef ||
		before.ProjectRef != after.ProjectRef || after.Revision < before.Revision ||
		after.PlanGeneration != before.PlanGeneration || !after.CreatedAt.Equal(before.CreatedAt) {
		return errors.New("sqlite.control_goal_fence_conflict")
	}
	itemRef := state.Control.WorkItemRef
	if state.Claim.Action.Ref != "" {
		itemRef = state.Claim.Action.WorkItemRef
	}
	if state.ExpectedWorkItemRevision != 0 {
		item, found := current.Goal.WorkItem(itemRef)
		if !found || item.Revision() != state.ExpectedWorkItemRevision {
			return errors.New("sqlite.control_item_fence_conflict")
		}
	}
	executionRef := state.Control.ExecutionRef
	if state.Claim.Action.Ref != "" {
		executionRef = state.Claim.Action.ExecutionRef
	}
	if state.ExpectedExecutionState != "" {
		execution, found := sqliteExecutionByRef(current.Executions, executionRef)
		if !found || execution.State != state.ExpectedExecutionState {
			return errors.New("sqlite.control_execution_fence_conflict")
		}
	}
	return nil
}

func validateControlStopReceipt(state application.ApplyControlState, current application.GoalRecord) error {
	if state.EffectReceipt == nil {
		return nil
	}
	receipt := *state.EffectReceipt
	execution, found := sqliteExecutionByRef(current.Executions, receipt.Subject.ExecutionRef)
	if !found || state.Claim.Action.Kind != application.ActionStopAgent ||
		receipt.ActionRef != state.Claim.Action.Ref || receipt.ActionFence != state.Claim.Fence ||
		receipt.Subject.GoalRef != execution.GoalRef || receipt.Subject.WorkItemRef != execution.WorkItemRef ||
		!receipt.ConfirmedAt.Equal(state.OperationAt) {
		return errors.New("sqlite.control_stop_receipt_invalid")
	}
	if receipt.ConfirmedAt.After(state.Claim.LeaseUntil) {
		return errors.New("sqlite.control_stop_receipt_after_lease")
	}
	if receipt.Status != application.EffectStatusStopped &&
		receipt.Status != application.EffectStatusAlreadyStopped &&
		receipt.Status != application.EffectStatusAlreadyCompleted &&
		receipt.Status != application.EffectStatusAlreadyFailed {
		return errors.New("sqlite.control_stop_receipt_nonterminal")
	}
	return nil
}
