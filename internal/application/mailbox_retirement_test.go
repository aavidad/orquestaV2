package application

import (
	"context"
	"testing"

	"orquesta/internal/goal"
	"orquesta/internal/ports"
)

func TestMailboxRecipientFailureRetiresWithoutReplacementOrForgedResolution(t *testing.T) {
	ctx := context.Background()
	system := newMailboxTestSystem(t, 1)
	admitted := system.admit(t, 0, "mailbox-admit:recipient-failure")
	action := seedParentObservationAction(t, system)

	system.agent.mu.Lock()
	system.agent.observations = append(system.agent.observations, ports.AgentObservation{
		Status: ports.AgentFailed, ErrorCode: "agent.parent_failed_after_admission",
		SpecHash: system.repository.records[system.goalRef].Goal.SpecHash(),
	})
	system.agent.mu.Unlock()

	processed, err := system.orchestrator.ProcessNext(ctx, "worker:recipient-failure")
	if err != nil || !processed.Processed || processed.Action != ActionObserveAgent {
		t.Fatalf("recipient failure: result=%+v err=%v", processed, err)
	}
	record, err := system.repository.GetGoal(ctx, system.goalRef)
	if err != nil {
		t.Fatal(err)
	}
	parent, found := record.Goal.WorkItem(system.parentRef)
	if !found || parent.State() != goal.WorkItemStateFailed ||
		record.Goal.State() != goal.GoalStateFailed || len(record.Executions) != len(system.children)+1 ||
		len(record.Goal.ChildHandoffResolutions()) != 0 {
		t.Fatalf("recipient failure invented replacement/resolution: goal=%+v executions=%+v",
			record.Goal.Snapshot(), record.Executions)
	}
	failed, found := executionForAction(record, action)
	if !found || failed.State != ExecutionFailed || failed.AttemptNo != 1 ||
		failed.FailureCode != "agent.parent_failed_after_admission" {
		t.Fatalf("failed execution changed identity: %+v", failed)
	}

	mailbox, err := system.orchestrator.GetMailbox(ctx, system.recipientAccess, GetMailboxRequest{
		GoalRef: system.goalRef, MessageRef: admitted.Record.Envelope.Ref,
		RecipientWorkItemRef: system.parentRef, RecipientExecutionRef: system.parentExecution,
	})
	if err != nil || mailbox.State != MailboxStateRetired || mailbox.Retirement == nil ||
		mailbox.Acknowledgement != nil || mailbox.Retirement.MessageRef != mailbox.Envelope.Ref ||
		mailbox.Retirement.ActionRef != mailbox.Action.Ref ||
		mailbox.Retirement.RecipientExecutionRef != system.parentExecution ||
		mailbox.Retirement.FailureCode != failed.FailureCode ||
		!mailbox.Retirement.RetiredAt.Equal(failed.FinishedAt) ||
		ValidateMailboxRetirement(mailbox) != nil {
		t.Fatalf("mailbox retirement=%+v err=%v", mailbox, err)
	}
	system.repository.mu.Lock()
	_, liveMailboxAction := system.repository.actions[mailbox.Action.Ref]
	system.repository.mu.Unlock()
	if liveMailboxAction {
		t.Fatal("retired mailbox action remained live")
	}
	if _, err = system.orchestrator.ClaimMailbox(
		ctx, system.recipientAccess,
		system.claimRequest(mailbox, "mailbox-claim:after-recipient-failure"),
	); !IsStateError(err, StateConflict) {
		t.Fatalf("retired mailbox was claimable: %v", err)
	}
}

func seedParentObservationAction(t *testing.T, system *mailboxTestSystem) ActionRecord {
	t.Helper()
	record, err := system.repository.GetGoal(context.Background(), system.goalRef)
	if err != nil {
		t.Fatal(err)
	}
	parent, found := record.Goal.WorkItem(system.parentRef)
	if !found {
		t.Fatal("parent missing")
	}
	action := ActionRecord{
		Ref: "action:observe:" + system.parentExecution.String(), Kind: ActionObserveAgent,
		GoalRef: system.goalRef, WorkItemRef: system.parentRef,
		ExecutionRef: system.parentExecution, PlanGeneration: record.Goal.PlanGeneration(),
		WorkItemGeneration: parent.Revision(), AvailableAt: system.clock.Now(),
	}
	system.repository.mu.Lock()
	system.repository.actions[action.Ref] = memoryAction{record: action}
	system.repository.mu.Unlock()
	return action
}
