package application

import (
	"context"
	"testing"

	"orquesta/internal/goal"
	"orquesta/internal/ports"
)

func TestControlStopMailboxRemainsLiveUntilConfirmationThenRetiresExactUnresolvedInbox(t *testing.T) {
	ctx := context.Background()
	system := newMailboxTestSystem(t, 2)
	beforeStop, err := system.repository.GetGoal(ctx, system.goalRef)
	if err != nil {
		t.Fatal(err)
	}
	parent, _ := beforeStop.Goal.WorkItem(system.parentRef)
	execution, found := executionByRef(beforeStop.Executions, system.parentExecution)
	if !found {
		t.Fatal("parent execution missing")
	}
	stop := ControlRequest{
		RequestRef: "control:mailbox-late-stop", Operation: ControlStop, Target: ControlTargetExecution,
		GoalRef: system.goalRef, ExpectedGoalRevision: beforeStop.Goal.Revision(),
		ExpectedPlanGeneration:    beforeStop.Goal.PlanGeneration(),
		ExpectedAppSpecGeneration: beforeStop.Goal.AppSpec().Generation(), ExpectedSpecHash: beforeStop.Goal.SpecHash(),
		WorkItemRef: system.parentRef, ExpectedWorkItemRevision: parent.Revision(),
		ExecutionRef: system.parentExecution, ExpectedExecutionAttempt: execution.AttemptNo,
		Mode: ports.AgentStopCooperative, Reason: "stop only after exact late mailbox frontier",
	}
	requested, err := system.orchestrator.Control(ctx, system.recipientAccess, stop)
	if err != nil || requested.Control.Status != ControlRequested {
		t.Fatalf("request stop: result=%+v err=%v", requested, err)
	}

	resolvedFlow := system.consume(t, 0, "after-stop-request")
	ack, err := system.orchestrator.AcknowledgeMailbox(
		ctx, system.recipientAccess,
		AcknowledgeMailboxRequest(system.resolveRequest(
			resolvedFlow, "mailbox-ack:after-stop-request", "effect:late-mailbox-resolved",
		)),
	)
	if err != nil || !ack.Created {
		t.Fatalf("ACK while stop pending: result=%+v err=%v", ack, err)
	}
	unresolved := system.admit(t, 1, "mailbox-admit:unresolved-before-stop-confirmation")

	processed, err := system.orchestrator.ProcessNext(ctx, "worker:confirm-mailbox-stop")
	if err != nil || !processed.Processed || processed.Action != ActionStopAgent {
		t.Fatalf("confirm stop: result=%+v err=%v", processed, err)
	}
	afterStop, err := system.repository.GetGoal(ctx, system.goalRef)
	if err != nil {
		t.Fatal(err)
	}
	stopped, _ := executionByRef(afterStop.Executions, system.parentExecution)
	if stopped.State != ExecutionStopped || !stopped.RecipientMailboxRetired {
		t.Fatalf("stopped execution retirement marker=%+v", stopped)
	}
	resolved, err := system.orchestrator.GetMailbox(ctx, system.recipientAccess, GetMailboxRequest{
		GoalRef: system.goalRef, MessageRef: resolvedFlow.admission.Record.Envelope.Ref,
		RecipientWorkItemRef: system.parentRef, RecipientExecutionRef: system.parentExecution,
	})
	if err != nil || resolved.State != MailboxStateAcknowledged || resolved.Acknowledgement == nil || resolved.Retirement != nil {
		t.Fatalf("resolved mailbox was rewritten by stop: record=%+v err=%v", resolved, err)
	}
	retired, err := system.orchestrator.GetMailbox(ctx, system.recipientAccess, GetMailboxRequest{
		GoalRef: system.goalRef, MessageRef: unresolved.Record.Envelope.Ref,
		RecipientWorkItemRef: system.parentRef, RecipientExecutionRef: system.parentExecution,
	})
	if err != nil || retired.State != MailboxStateRetired || retired.Retirement == nil || retired.Acknowledgement != nil ||
		retired.Retirement.RecipientExecutionRef != system.parentExecution || ValidateMailboxRetirement(retired) != nil {
		t.Fatalf("unresolved mailbox retirement=%+v err=%v", retired, err)
	}
	resolutions := afterStop.Goal.ChildHandoffResolutions()
	if len(resolutions) != 1 || resolutions[0].MessageRef() != resolved.Envelope.Ref.String() {
		t.Fatalf("stop fabricated or lost handoff resolutions: %+v", resolutions)
	}

	afterParent, _ := afterStop.Goal.WorkItem(system.parentRef)
	retry := ControlRequest{
		RequestRef: "control:retry-after-real-mailbox-retirement", Operation: ControlRetry,
		Target: ControlTargetWorkItem, GoalRef: system.goalRef,
		ExpectedGoalRevision: afterStop.Goal.Revision(), ExpectedPlanGeneration: afterStop.Goal.PlanGeneration(),
		ExpectedAppSpecGeneration: afterStop.Goal.AppSpec().Generation(), ExpectedSpecHash: afterStop.Goal.SpecHash(),
		WorkItemRef: system.parentRef, ExpectedWorkItemRevision: afterParent.Revision(),
		ExecutionRef: system.parentExecution, ExpectedExecutionAttempt: stopped.AttemptNo,
		Reason: "retry must honor durable unresolved inbox retirement",
	}
	system.repository.mu.Lock()
	controlsBefore, executionsBefore, actionsBefore := len(system.repository.records[system.goalRef].Controls),
		len(system.repository.records[system.goalRef].Executions), len(system.repository.actions)
	system.repository.mu.Unlock()
	if _, err := system.orchestrator.Control(ctx, system.recipientAccess, retry); !IsStateError(err, StateConflict) {
		t.Fatalf("retry after real retirement error=%v", err)
	}
	system.repository.mu.Lock()
	controlsAfter, executionsAfter, actionsAfter := len(system.repository.records[system.goalRef].Controls),
		len(system.repository.records[system.goalRef].Executions), len(system.repository.actions)
	system.repository.mu.Unlock()
	if controlsAfter != controlsBefore || executionsAfter != executionsBefore || actionsAfter != actionsBefore {
		t.Fatalf("blocked retry wrote state controls=%d/%d executions=%d/%d actions=%d/%d",
			controlsAfter, controlsBefore, executionsAfter, executionsBefore, actionsAfter, actionsBefore)
	}

	if _, err := goal.RestoreGoal(afterStop.Goal.Snapshot()); err != nil {
		t.Fatalf("post-stop Goal does not round-trip: %v", err)
	}
}
