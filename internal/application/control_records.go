package application

import (
	"time"

	"orquesta/internal/goal"
	"orquesta/internal/identity"
	"orquesta/internal/ports"
)

func newControlRecord(
	ref, fingerprint string,
	principal identity.PrincipalRef,
	projectRef goal.ProjectRef,
	request ControlRequest,
	authorization identity.AuthorizationReceipt,
	at time.Time,
) ControlRecord {
	return ControlRecord{
		Ref: ref, RequestRef: request.RequestRef, RequestFingerprint: fingerprint,
		PrincipalRef: principal, ProjectRef: projectRef, GoalRef: request.GoalRef,
		WorkItemRef: request.WorkItemRef, WorkItemRevision: request.ExpectedWorkItemRevision,
		ExecutionRef: request.ExecutionRef, ExecutionAttempt: request.ExpectedExecutionAttempt,
		Operation: request.Operation, Target: request.Target, Mode: request.Mode, Reason: request.Reason,
		GoalRevision:   request.ExpectedGoalRevision,
		PlanGeneration: request.ExpectedPlanGeneration, AppSpecGeneration: request.ExpectedAppSpecGeneration,
		SpecHash: request.ExpectedSpecHash, Status: ControlRequested, RequestedAt: at,
		AuthorizationReceipt: authorization,
	}
}

func confirmLocalControl(record *ControlRecord, at time.Time) {
	record.Status = ControlConfirmed
	record.ConfirmedAt = at.UTC()
	record.ReceiptRef = "receipt:" + record.Ref
}

func cancelExecutions(records []ExecutionRecord, target ControlTarget, itemRef goal.WorkItemRef) []ExecutionRecord {
	result := make([]ExecutionRecord, 0)
	for _, execution := range records {
		if target == ControlTargetWorkItem && execution.WorkItemRef != itemRef {
			continue
		}
		if execution.State == ExecutionDispatching || execution.State == ExecutionRunning {
			result = append(result, execution)
		}
	}
	return result
}

func cancelStopMode(capabilities ports.AgentControlCapabilities) (ports.AgentStopMode, bool) {
	if capabilities.CooperativeStop {
		return ports.AgentStopCooperative, true
	}
	if capabilities.ForcedStop {
		return ports.AgentStopForced, true
	}
	return "", false
}

func controlEvents(
	before goal.Goal,
	after goal.Goal,
	control ControlRecord,
	executions []ExecutionRecord,
	at time.Time,
) []EventRecord {
	events := []EventRecord{{
		Ref: "event:control:" + control.Ref, Kind: "control." + string(control.Operation),
		GoalRef: control.GoalRef, WorkItemRef: control.WorkItemRef, ExecutionRef: control.ExecutionRef,
		OccurredAt: at,
	}}
	if control.SupersedesControlRef != "" {
		events = append(events, EventRecord{
			Ref:  "event:control-superseded:" + control.SupersedesControlRef,
			Kind: "control.stop_superseded", GoalRef: control.GoalRef,
			WorkItemRef: control.WorkItemRef, ExecutionRef: control.ExecutionRef,
			OccurredAt: at,
		})
	}
	for _, execution := range executions {
		if execution.State == ExecutionCanceled {
			events = append(events, EventRecord{
				Ref: "event:execution-canceled:" + execution.Ref.String(), Kind: "execution.canceled",
				GoalRef: execution.GoalRef, WorkItemRef: execution.WorkItemRef,
				ExecutionRef: execution.Ref, OccurredAt: at,
			})
		} else if execution.State == ExecutionQueued && execution.ReplacesExecutionRef.String() != "" {
			events = append(events, EventRecord{
				Ref: "event:execution-queued:" + execution.Ref.String(), Kind: "execution.queued",
				GoalRef: execution.GoalRef, WorkItemRef: execution.WorkItemRef,
				ExecutionRef: execution.Ref, OccurredAt: at,
			})
		}
	}
	if !before.IsTerminal() && after.IsTerminal() {
		events = append(events, EventRecord{
			Ref:  "event:goal-" + string(after.State()) + ":" + after.Ref().String(),
			Kind: "goal." + string(after.State()), GoalRef: after.Ref(), OccurredAt: at,
		})
	}
	return events
}
