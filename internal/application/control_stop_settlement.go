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
) error {
	at := lifecycleTime(receipt.ConfirmedAt, record.Goal, item)
	aggregate := record.Goal
	previousExecutionState := execution.State
	var err error
	if control.Operation == ControlCancel {
		aggregate, err = aggregate.CompleteWorkItemCancel(
			aggregate.Revision(), item.Revision(), item.Ref(), at,
		)
		if err == nil {
			aggregate, err = orchestrator.closeCanceledScope(aggregate, control, at)
		}
	} else {
		aggregate, err = aggregate.InterruptWorkItem(
			aggregate.Revision(), item.Revision(), item.Ref(), execution.Ref,
			goal.WorkItemInterruptExecutionStopped, at,
		)
	}
	if err != nil {
		return err
	}
	execution.State = ExecutionStopped
	execution.FailureCode = "application.execution_stopped"
	execution.FinishedAt = at
	updatedItem, _ := aggregate.WorkItem(item.Ref())
	if control.Operation != ControlCancel || control.Target == ControlTargetWorkItem || aggregate.IsTerminal() {
		control.Status = ControlConfirmed
		control.ConfirmedAt = at
		control.ReceiptRef = confirmedControlReceiptRef(control, receipt.ReceiptRef)
	}
	events := stoppedExecutionEvents(aggregate, updatedItem, execution, at)
	existing := replaceExecution(record.Executions, execution)
	newExecutions, newActions, scheduledEvents, err := orchestrator.scheduleReady(ctx, aggregate, existing, at)
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
		Executions:                   append([]ExecutionRecord{execution}, newExecutions...),
		NewActions:                   newActions,
		RetireActionRefs:             []string{"action:observe:" + execution.Ref.String()},
		RetireMailboxForExecutionRef: execution.Ref, StopReceipt: &receipt,
		Events: events, Control: control, OperationAt: at,
	})
	return err
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
) error {
	if control.Operation != ControlCancel {
		control.Status = ControlConfirmed
		control.ConfirmedAt = receipt.ConfirmedAt.UTC()
		control.ReceiptRef = receipt.ReceiptRef
	}
	_, _, err := orchestrator.state.ApplyControl(ctx, ApplyControlState{
		RequestRef: control.RequestRef, RequestFingerprint: control.RequestFingerprint,
		AuthorizationReceipt: control.AuthorizationReceipt, PrincipalRef: control.PrincipalRef,
		ProjectRef: control.ProjectRef, GoalRef: control.GoalRef,
		ExpectedGoalRevision: record.Goal.Revision(), ExpectedPlanGeneration: record.Goal.PlanGeneration(),
		ExpectedWorkItemRevision: item.Revision(), ExpectedExecutionState: execution.State,
		ExpectedControlStatus: ControlRequested, Claim: claim, Goal: record.Goal,
		StopReceipt: &receipt, Control: control, OperationAt: receipt.ConfirmedAt.UTC(),
		Events: []EventRecord{{
			Ref: "event:stop-observed-terminal:" + claim.Action.Ref, Kind: "control.stop_observed_terminal",
			GoalRef: record.Goal.Ref(), WorkItemRef: item.Ref(), ExecutionRef: execution.Ref,
			OccurredAt: receipt.ConfirmedAt.UTC(),
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
	status := ports.AgentStopAlreadyFailed
	if execution.State == ExecutionStopped {
		status = ports.AgentStopAlreadyStopped
	} else if execution.State == ExecutionSucceeded {
		status = ports.AgentStopAlreadyCompleted
	}
	at := orchestrator.clock.Now().UTC()
	receipt := ports.AgentStopReceipt{
		ExecutionRef: execution.Ref, GoalRef: execution.GoalRef, WorkItemRef: execution.WorkItemRef,
		PlanGeneration: execution.PlanGeneration, AppSpecGeneration: execution.AppSpecGeneration,
		ExecutionAttempt: execution.AttemptNo, SpecHash: execution.SpecHash,
		ProviderRef: execution.ProviderRef, ModelRef: execution.ModelRef, AgentRef: execution.AgentRef,
		ExternalRef: execution.ExternalRef, Mode: control.Mode,
		IdempotencyKey: "stop:" + control.Ref + ":" + execution.Ref.String(), Status: status,
		ReceiptRef: "receipt:terminal:" + execution.Ref.String(), ConfirmedAt: at,
	}
	return orchestrator.settleStopObservedTerminal(ctx, claim, record, item, execution, control, receipt)
}
