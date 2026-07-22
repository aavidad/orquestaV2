package application

import (
	"context"
	"time"

	"orquesta/internal/goal"
	"orquesta/internal/ports"
)

func (orchestrator *Orchestrator) settleStopped(
	ctx context.Context,
	claim ActionClaim,
	record GoalRecord,
	item goal.WorkItem,
	execution ExecutionRecord,
	control ControlRecord,
	receipt ports.AgentStopReceipt,
	effectReceipt EffectReceipt,
) error {
	if IsReviewCleanupControl(control) {
		return orchestrator.settleReviewCleanupStopped(
			ctx, claim, record, item, execution, control, receipt, effectReceipt,
		)
	}
	at := lifecycleTime(effectReceipt.ConfirmedAt, record.Goal, item)
	aggregate := record.Goal
	previousExecutionState := execution.State
	execution.State = ExecutionStopped
	execution.FailureCode = "application.execution_stopped"
	execution.FinishedAt = at
	updates := []ExecutionRecord{execution}
	retireActionRefs := []string{"action:observe:" + execution.Ref.String()}
	var retirementEvents []EventRecord
	var cleanupControls []ControlRecord
	var cleanupActions []ActionRecord
	var err error
	if control.Operation == ControlCancel {
		candidate := replaceExecution(record.Executions, execution)
		if !activeExecutionInItem(candidate, item.Ref()) && !item.IsTerminal() {
			aggregate, err = aggregate.CompleteWorkItemCancel(
				aggregate.Revision(), item.Revision(), item.Ref(), at,
			)
			if err == nil {
				aggregate, err = orchestrator.closeCanceledScope(aggregate, control, at)
			}
		}
		if err == nil && !activeExecutionInControlScope(candidate, control) {
			control.Status = ControlConfirmed
			control.ConfirmedAt = at
			control.ReceiptRef = confirmedControlReceiptRef(control, receipt.ReceiptRef)
		}
	} else {
		if isReviewerExecution(execution) {
			var retired []ReviewParticipantRetirement
			var refs []string
			retired, refs, cleanupControls, cleanupActions, retirementEvents, err =
				orchestrator.reviewCleanupPlan(record, item, execution.Ref, execution.ReviewSubjectDigest,
					control.Ref, at)
			if err == nil {
				for _, participant := range retired {
					updates = append(updates, participant.Execution)
				}
				retireActionRefs = append(retireActionRefs, refs...)
				if !reviewCleanupStillActive(record, execution, retired) {
					var author ExecutionRecord
					aggregate, author, err = orchestrator.interruptAuthorForReviewFailure(
						record, item, at, "review.unavailable",
					)
					if err == nil {
						updates = append(updates, author)
					}
				}
			}
		} else {
			aggregate, err = aggregate.InterruptWorkItem(
				aggregate.Revision(), item.Revision(), item.Ref(), execution.Ref,
				goal.WorkItemInterruptExecutionStopped, at,
			)
		}
		if err == nil {
			control.Status = ControlConfirmed
			control.ConfirmedAt = at
			control.ReceiptRef = confirmedControlReceiptRef(control, receipt.ReceiptRef)
		}
	}
	if err != nil {
		return err
	}
	settlement, err := settlementFor(record, execution, unknownUsage(), 0, at)
	if err != nil {
		return err
	}
	updatedItem, _ := aggregate.WorkItem(item.Ref())
	events := stoppedExecutionEvents(aggregate, updatedItem, execution, at)
	events = append(events, retirementEvents...)
	existing := replaceExecutions(record.Executions, updates)
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
		Executions:                   append(updates, newExecutions...),
		NewControls:                  cleanupControls,
		NewActions:                   append(cleanupActions, newActions...),
		RetireActionRefs:             retireActionRefs,
		RetireMailboxForExecutionRef: execution.Ref, EffectReceipt: &effectReceipt,
		BudgetSettlement: settlement,
		Events:           events, Control: control, OperationAt: at,
	})
	return err
}

func activeExecutionInItem(executions []ExecutionRecord, itemRef goal.WorkItemRef) bool {
	for _, execution := range executions {
		if execution.WorkItemRef == itemRef &&
			(execution.State == ExecutionDispatching || execution.State == ExecutionRunning) {
			return true
		}
	}
	return false
}

func activeExecutionInControlScope(executions []ExecutionRecord, control ControlRecord) bool {
	for _, execution := range executions {
		if control.Target == ControlTargetWorkItem && execution.WorkItemRef != control.WorkItemRef {
			continue
		}
		if execution.State == ExecutionDispatching || execution.State == ExecutionRunning {
			return true
		}
	}
	return false
}

func stoppedExecutionEvents(
	aggregate goal.Goal,
	item goal.WorkItem,
	execution ExecutionRecord,
	at time.Time,
) []EventRecord {
	events := []EventRecord{{
		Ref: "event:execution-stopped:" + execution.Ref.String(), Kind: "execution.stopped",
		GoalRef: execution.GoalRef, WorkItemRef: execution.WorkItemRef,
		ExecutionRef: execution.Ref, OccurredAt: at,
	}}
	if item.State() == goal.WorkItemStateInterrupted {
		events = append(events, EventRecord{
			Ref: "event:work-interrupted:" + execution.Ref.String(), Kind: "work_item.interrupted",
			GoalRef: execution.GoalRef, WorkItemRef: execution.WorkItemRef,
			ExecutionRef: execution.Ref, OccurredAt: at,
		})
	} else if item.State() == goal.WorkItemStateCanceled {
		events = append(events, EventRecord{
			Ref: "event:work-canceled:" + execution.Ref.String(), Kind: "work_item.canceled",
			GoalRef: execution.GoalRef, WorkItemRef: execution.WorkItemRef,
			ExecutionRef: execution.Ref, OccurredAt: at,
		})
	}
	if aggregate.IsTerminal() {
		events = append(events, EventRecord{
			Ref:  "event:goal-" + string(aggregate.State()) + ":" + aggregate.Ref().String(),
			Kind: "goal." + string(aggregate.State()), GoalRef: aggregate.Ref(), OccurredAt: at,
		})
	}
	return events
}

func (orchestrator *Orchestrator) settleStopObservedTerminal(
	ctx context.Context,
	claim ActionClaim,
	record GoalRecord,
	item goal.WorkItem,
	execution ExecutionRecord,
	control ControlRecord,
	receipt ports.AgentStopReceipt,
	effectReceipt EffectReceipt,
) error {
	if control.Operation != ControlCancel {
		control.Status = ControlConfirmed
		control.ConfirmedAt = effectReceipt.ConfirmedAt
		control.ReceiptRef = receipt.ReceiptRef
	}
	_, _, err := orchestrator.state.ApplyControl(ctx, ApplyControlState{
		RequestRef: control.RequestRef, RequestFingerprint: control.RequestFingerprint,
		AuthorizationReceipt: control.AuthorizationReceipt, PrincipalRef: control.PrincipalRef,
		ProjectRef: control.ProjectRef, GoalRef: control.GoalRef,
		ExpectedGoalRevision: record.Goal.Revision(), ExpectedPlanGeneration: record.Goal.PlanGeneration(),
		ExpectedWorkItemRevision: item.Revision(), ExpectedExecutionState: execution.State,
		ExpectedControlStatus: ControlRequested, Claim: claim, Goal: record.Goal,
		EffectReceipt: &effectReceipt, Control: control, OperationAt: effectReceipt.ConfirmedAt,
		Events: []EventRecord{{
			Ref: "event:stop-observed-terminal:" + claim.Action.Ref, Kind: "control.stop_observed_terminal",
			GoalRef: record.Goal.Ref(), WorkItemRef: item.Ref(), ExecutionRef: execution.Ref,
			OccurredAt: effectReceipt.ConfirmedAt,
		}},
	})
	return err
}

func (orchestrator *Orchestrator) settleStopAgainstTerminal(
	ctx context.Context,
	claim ActionClaim,
	record GoalRecord,
	item goal.WorkItem,
	execution ExecutionRecord,
	control ControlRecord,
) error {
	at := orchestrator.clock.Now().UTC()
	if control.Operation != ControlCancel {
		control.Status, control.ConfirmedAt = ControlConfirmed, at
		control.ReceiptRef = "receipt:local-terminal:" + execution.Ref.String()
	}
	_, _, err := orchestrator.state.ApplyControl(ctx, ApplyControlState{
		RequestRef: control.RequestRef, RequestFingerprint: control.RequestFingerprint,
		AuthorizationReceipt: control.AuthorizationReceipt, PrincipalRef: control.PrincipalRef,
		ProjectRef: control.ProjectRef, GoalRef: control.GoalRef,
		ExpectedGoalRevision: record.Goal.Revision(), ExpectedPlanGeneration: record.Goal.PlanGeneration(),
		ExpectedWorkItemRevision: item.Revision(), ExpectedExecutionState: execution.State,
		ExpectedControlStatus: ControlRequested, Claim: claim, Goal: record.Goal,
		Control: control, OperationAt: at,
		Events: []EventRecord{{
			Ref: "event:stop-local-terminal:" + claim.Action.Ref, Kind: "control.stop_local_terminal",
			GoalRef: record.Goal.Ref(), WorkItemRef: item.Ref(), ExecutionRef: execution.Ref, OccurredAt: at,
		}},
	})
	return err
}
