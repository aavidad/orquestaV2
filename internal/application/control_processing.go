package application

import (
	"context"
	"errors"
	"time"

	"orquesta/internal/goal"
	"orquesta/internal/governance"
	"orquesta/internal/ports"
)

func (orchestrator *Orchestrator) processStop(ctx context.Context, claim ActionClaim) error {
	record, err := orchestrator.state.GetGoal(ctx, claim.Action.GoalRef)
	if err != nil {
		if IsStateError(err, StateNotFound) {
			return orchestrator.quarantine(ctx, claim, "application.action_goal_not_found")
		}
		return err
	}
	item, execution, control, err := validateStopClaim(claim, record)
	if err != nil {
		return orchestrator.quarantine(ctx, claim, err.Error())
	}
	if priorEffectAttemptBlocksDispatch(record, claim.Action) {
		return orchestrator.quarantineUnknownApplied(ctx, claim)
	}
	if execution.State == ExecutionSucceeded || execution.State == ExecutionFailed ||
		execution.State == ExecutionCanceled || execution.State == ExecutionStopped {
		return orchestrator.settleStopAgainstTerminal(ctx, claim, record, item, execution, control)
	}
	if err := validateClaimedEffect(claim, orchestrator.clock.Now()); err != nil {
		return orchestrator.quarantine(ctx, claim, err.Error())
	}
	if execution.State == ExecutionDispatching || execution.ExternalRef == "" {
		return orchestrator.requeueStop(ctx, claim, execution, "application.stop_waiting_launch_receipt")
	}
	request := stopRequest(control, execution)
	if stopTargetDigest(control, request) != claim.Action.EffectIntent.TargetDigest {
		return orchestrator.quarantine(ctx, claim, "application.effect_target_mismatch")
	}
	capabilities, err := orchestrator.controller.ControlCapabilities(ctx)
	if err != nil {
		return orchestrator.requeueStop(ctx, claim, execution, "agent.control_capabilities_failed")
	}
	if !ports.SupportsAgentStopMode(capabilities, control.Mode) {
		return orchestrator.requeueStop(ctx, claim, execution, "agent.stop_unsupported")
	}
	if err := orchestrator.validateCurrentAutomaticAuthority(ctx, claim); err != nil {
		return orchestrator.requeueStop(ctx, claim, execution, err.Error())
	}
	attempt, err := orchestrator.beginNewEffectAttempt(ctx, claim, orchestrator.clock.Now())
	if err != nil {
		return err
	}
	effectCtx, cancel := orchestrator.actionCallContext(ctx, claim)
	receipt, stopErr := orchestrator.controller.Stop(effectCtx, request)
	cancel()
	if stopErr != nil {
		return orchestrator.quarantineUnknownApplied(ctx, claim)
	}
	if err := ports.ValidateAgentStopReceipt(request, receipt); err != nil {
		return orchestrator.quarantineUnknownApplied(ctx, claim)
	}
	switch receipt.Status {
	case ports.AgentStopPending, ports.AgentStopUnsupported:
		return orchestrator.quarantineUnknownApplied(ctx, claim)
	case ports.AgentStopAlreadyCompleted, ports.AgentStopAlreadyFailed:
		confirmedAt := orchestrator.clock.Now().UTC()
		externalReceipt, err := effectReceipt(claim, attempt, receipt.ReceiptRef, EffectStatus(receipt.Status), unknownUsage(), confirmedAt)
		if err != nil {
			return orchestrator.quarantineUnknownApplied(ctx, claim)
		}
		if err := orchestrator.settleStopObservedTerminal(ctx, claim, record, item, execution, control, receipt, externalReceipt); err != nil {
			return orchestrator.quarantineUnknownApplied(ctx, claim)
		}
		return nil
	case ports.AgentStopped, ports.AgentStopAlreadyStopped:
		confirmedAt := orchestrator.clock.Now().UTC()
		externalReceipt, err := effectReceipt(claim, attempt, receipt.ReceiptRef, EffectStatus(receipt.Status), unknownUsage(), confirmedAt)
		if err != nil {
			return orchestrator.quarantineUnknownApplied(ctx, claim)
		}
		if err := orchestrator.settleStopped(ctx, claim, record, item, execution, control, receipt, externalReceipt); err != nil {
			return orchestrator.quarantineUnknownApplied(ctx, claim)
		}
		return nil
	default:
		return orchestrator.quarantineUnknownApplied(ctx, claim)
	}
}

func (orchestrator *Orchestrator) settleCanceledObservation(
	ctx context.Context,
	claim ActionClaim,
	record GoalRecord,
	item goal.WorkItem,
	execution ExecutionRecord,
	observation ports.AgentObservation,
) error {
	at := lifecycleTime(observation.ObservedAt, record.Goal, item)
	state, failureCode := ExecutionFailed, stableFailureCode(observation.ErrorCode)
	if observation.Status == ports.AgentCompleted {
		state, failureCode = ExecutionSucceeded, ""
	}
	settlement, err := settlementFor(record, execution, observation.Usage, int64(len(observation.Content)), at)
	if err != nil {
		return err
	}
	return orchestrator.settleCanceledExecution(
		ctx, claim, record, item, execution, state, failureCode,
		observation.ObservedAt, "receipt:observation:"+execution.Ref.String(), at, settlement, "",
	)
}

func (orchestrator *Orchestrator) settleCanceledLaunchRejection(
	ctx context.Context,
	claim ActionClaim,
	record GoalRecord,
	item goal.WorkItem,
	execution ExecutionRecord,
	causalAttemptRef string,
	failureCode string,
) error {
	at := lifecycleTime(orchestrator.clock.Now(), record.Goal, item)
	settlement, err := releaseSettlementForAttempt(claim, causalAttemptRef, at)
	if err != nil {
		return err
	}
	return orchestrator.settleCanceledExecution(
		ctx, claim, record, item, execution, ExecutionFailed, stableFailureCode(failureCode),
		time.Time{}, "receipt:launch-rejected:"+execution.Ref.String(), at, &settlement,
		"agent.launch_definitely_not_applied",
	)
}

func (orchestrator *Orchestrator) settleCanceledExecution(ctx context.Context, claim ActionClaim, record GoalRecord, item goal.WorkItem, execution ExecutionRecord, terminalState ExecutionState, failureCode string, providerObservedAt time.Time, receiptRef string, at time.Time, settlement *governance.BudgetSettlement, claimErrorCode string) error {
	control, found := pendingCancelControl(record.Controls, item.Ref())
	if !found || (terminalState != ExecutionSucceeded && terminalState != ExecutionFailed) {
		return &StateError{Code: StateConflict}
	}
	aggregate, err := record.Goal.CompleteWorkItemCancel(
		record.Goal.Revision(), item.Revision(), item.Ref(), at,
	)
	if err != nil {
		return err
	}
	aggregate, err = orchestrator.closeCanceledScope(aggregate, control, at)
	if err != nil {
		return err
	}
	previousExecutionState := execution.State
	execution.State, execution.FailureCode, execution.FinishedAt = terminalState, failureCode, at
	if !providerObservedAt.IsZero() {
		execution.LastObservedAt = at
		execution.ProviderObservedAt = providerObservedAt.UTC()
	}
	if control.Target == ControlTargetWorkItem || aggregate.IsTerminal() {
		control.Status = ControlConfirmed
		control.ConfirmedAt = at
		control.ReceiptRef = confirmedControlReceiptRef(control, receiptRef)
	}
	events := []EventRecord{
		{
			Ref:  "event:execution-" + string(execution.State) + ":" + execution.Ref.String(),
			Kind: "execution." + string(execution.State), GoalRef: execution.GoalRef,
			WorkItemRef: execution.WorkItemRef, ExecutionRef: execution.Ref, OccurredAt: at,
		},
		{
			Ref: "event:work-canceled:" + execution.Ref.String(), Kind: "work_item.canceled",
			GoalRef: execution.GoalRef, WorkItemRef: execution.WorkItemRef,
			ExecutionRef: execution.Ref, OccurredAt: at,
		},
	}
	if aggregate.IsTerminal() {
		events = append(events, EventRecord{
			Ref:  "event:goal-" + string(aggregate.State()) + ":" + aggregate.Ref().String(),
			Kind: "goal." + string(aggregate.State()), GoalRef: aggregate.Ref(), OccurredAt: at,
		})
	}
	existing := replaceExecution(record.Executions, execution)
	newExecutions, newActions, scheduledEvents, err := orchestrator.scheduleHistoricalReady(
		ctx, record, aggregate, existing, at,
	)
	if err != nil {
		return err
	}
	events = append(events, scheduledEvents...)
	_, _, err = orchestrator.state.ApplyControl(ctx, ApplyControlState{
		RequestRef: control.RequestRef, RequestFingerprint: control.RequestFingerprint,
		AuthorizationReceipt: control.AuthorizationReceipt, PrincipalRef: control.PrincipalRef,
		ProjectRef: control.ProjectRef, GoalRef: control.GoalRef,
		ExpectedGoalRevision: record.Goal.Revision(), ExpectedPlanGeneration: record.Goal.PlanGeneration(),
		ExpectedWorkItemRevision: item.Revision(), ExpectedExecutionState: previousExecutionState,
		ExpectedControlStatus: ControlRequested, Claim: claim, Goal: aggregate,
		ClaimErrorCode:               claimErrorCode,
		Executions:                   append([]ExecutionRecord{execution}, newExecutions...),
		NewActions:                   newActions,
		RetireActionRefs:             []string{"action:stop:" + control.Ref + ":" + execution.Ref.String()},
		RetireMailboxForExecutionRef: execution.Ref,
		BudgetSettlement:             settlement,
		Events:                       events, Control: control, OperationAt: at,
	})
	return err
}

func validateStopClaim(
	claim ActionClaim,
	record GoalRecord,
) (goal.WorkItem, ExecutionRecord, ControlRecord, error) {
	if claim.Token == "" || claim.WorkerRef == "" || claim.DeliveryAttempt == 0 || claim.Fence == 0 ||
		claim.LeaseUntil.IsZero() || claim.Action.Kind != ActionStopAgent || claim.Action.ControlRef == "" ||
		claim.Action.GoalRef != record.Goal.Ref() || claim.Action.Ref !=
		"action:stop:"+claim.Action.ControlRef+":"+claim.Action.ExecutionRef.String() {
		return goal.WorkItem{}, ExecutionRecord{}, ControlRecord{}, errors.New("application.stop_claim_invalid")
	}
	item, found := record.Goal.WorkItem(claim.Action.WorkItemRef)
	if !found || claim.Action.WorkItemGeneration > item.Revision() {
		return goal.WorkItem{}, ExecutionRecord{}, ControlRecord{}, errors.New("application.stop_item_invalid")
	}
	execution, found := executionForAction(record, claim.Action)
	if !found || execution.WorkItemRef != item.Ref() || execution.PlanGeneration != claim.Action.PlanGeneration {
		return goal.WorkItem{}, ExecutionRecord{}, ControlRecord{}, errors.New("application.stop_execution_invalid")
	}
	control, found := controlByRef(record.Controls, claim.Action.ControlRef)
	if !found || control.Status != ControlRequested || control.GoalRef != record.Goal.Ref() ||
		(control.Operation != ControlStop && control.Operation != ControlCancel) ||
		!validStopMode(control.Mode) ||
		(control.Target == ControlTargetWorkItem && control.WorkItemRef != item.Ref()) ||
		(control.Operation == ControlStop && (control.ExecutionRef != execution.Ref ||
			control.ExecutionAttempt != execution.AttemptNo)) {
		return goal.WorkItem{}, ExecutionRecord{}, ControlRecord{}, errors.New("application.stop_control_invalid")
	}
	return item, execution, control, nil
}

func stopRequest(control ControlRecord, execution ExecutionRecord) ports.AgentStopRequest {
	return ports.AgentStopRequest{
		ExecutionRef: execution.Ref, GoalRef: execution.GoalRef, WorkItemRef: execution.WorkItemRef,
		PlanGeneration: execution.PlanGeneration, AppSpecGeneration: execution.AppSpecGeneration,
		ExecutionAttempt: execution.AttemptNo, SpecHash: execution.SpecHash,
		ProviderRef: execution.ProviderRef, ModelRef: execution.ModelRef, AgentRef: execution.AgentRef,
		ExternalRef: execution.ExternalRef, Mode: control.Mode,
		IdempotencyKey: "stop:" + control.Ref + ":" + execution.Ref.String(),
	}
}

func (orchestrator *Orchestrator) closeCanceledScope(
	aggregate goal.Goal,
	control ControlRecord,
	at time.Time,
) (goal.Goal, error) {
	if control.Target == ControlTargetGoal {
		for _, candidate := range aggregate.WorkItems() {
			if !candidate.IsTerminal() {
				return aggregate, nil
			}
		}
		return aggregate.CompleteCancel(aggregate.Revision(), at)
	}
	if outcome, closable := aggregate.ClosableOutcome(); closable {
		return aggregate.Close(aggregate.Revision(), outcome, at)
	}
	return aggregate, nil
}

func confirmedControlReceiptRef(control ControlRecord, providerReceiptRef string) string {
	if control.Operation == ControlCancel && control.Target == ControlTargetGoal {
		return "receipt:" + control.Ref
	}
	return providerReceiptRef
}

func controlByRef(records []ControlRecord, ref string) (ControlRecord, bool) {
	for _, record := range records {
		if record.Ref == ref {
			return record, true
		}
	}
	return ControlRecord{}, false
}

func pendingCancelControl(records []ControlRecord, itemRef goal.WorkItemRef) (ControlRecord, bool) {
	var selected ControlRecord
	for _, record := range records {
		if record.Operation != ControlCancel || record.Status != ControlRequested ||
			(record.Target == ControlTargetWorkItem && record.WorkItemRef != itemRef) {
			continue
		}
		if selected.Ref == "" || record.RequestedAt.After(selected.RequestedAt) {
			selected = record
		}
	}
	return selected, selected.Ref != ""
}
