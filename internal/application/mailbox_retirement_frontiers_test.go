package application

import (
	"context"
	"reflect"
	"testing"
	"time"

	"orquesta/internal/goal"
	"orquesta/internal/ports"
)

func TestMailboxRecipientFailureRetiresEveryUnresolvedFrontier(t *testing.T) {
	tests := []struct {
		name      string
		depth     int
		wantState MailboxState
	}{
		{name: "admitted", depth: 0, wantState: MailboxStateAdmitted},
		{name: "claimed", depth: 1, wantState: MailboxStateClaimed},
		{name: "delivered", depth: 2, wantState: MailboxStateDelivered},
		{name: "consumed", depth: 3, wantState: MailboxStateConsumed},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx := context.Background()
			system := newMailboxTestSystem(t, 1)
			admission := system.admit(t, 0, "mailbox-admit:retire-"+test.name)
			frontier := advanceMailboxRetirementFrontier(t, system, admission, test.name, test.depth)
			if frontier.record.State != test.wantState {
				t.Fatalf("frontier state=%q want=%q", frontier.record.State, test.wantState)
			}

			beforeFailure, err := system.repository.GetGoal(ctx, system.goalRef)
			if err != nil {
				t.Fatal(err)
			}
			consumedReceipt, hadConsumedReceipt := mailboxConsumptionReceipt(
				beforeFailure.ConsumptionReceipts, admission.Record.Envelope.Ref,
			)
			if hadConsumedReceipt != (test.depth == 3) {
				t.Fatalf("consumption receipt before failure=%v depth=%d", hadConsumedReceipt, test.depth)
			}
			action := seedParentObservationAction(t, system)
			system.clock.Advance(time.Second)
			system.agent.mu.Lock()
			system.agent.observations = append(system.agent.observations, ports.AgentObservation{
				Status: ports.AgentFailed, ErrorCode: "agent.parent_failed_at_" + test.name,
				SpecHash: beforeFailure.Goal.SpecHash(),
			})
			system.agent.mu.Unlock()

			processed, err := system.orchestrator.ProcessNext(ctx, "worker:retire-"+test.name)
			if err != nil || !processed.Processed || processed.Action != ActionObserveAgent {
				t.Fatalf("recipient failure: result=%+v err=%v", processed, err)
			}
			failedRecord, err := system.repository.GetGoal(ctx, system.goalRef)
			if err != nil {
				t.Fatal(err)
			}
			parent, found := failedRecord.Goal.WorkItem(system.parentRef)
			failed, executionFound := executionForAction(failedRecord, action)
			if !found || parent.State() != goal.WorkItemStateFailed ||
				failedRecord.Goal.State() != goal.GoalStateFailed || !executionFound ||
				failed.State != ExecutionFailed || failed.AttemptNo != 1 ||
				failed.MaxExecutionAttempts <= failed.AttemptNo ||
				failed.FailureCode != "agent.parent_failed_at_"+test.name ||
				len(failedRecord.Executions) != len(system.children)+1 ||
				len(failedRecord.Goal.ChildHandoffResolutions()) != 0 {
				t.Fatalf("failure replaced execution or forged handoff: goal=%+v executions=%+v",
					failedRecord.Goal.Snapshot(), failedRecord.Executions)
			}

			retired, err := system.orchestrator.GetMailbox(ctx, system.recipientAccess, GetMailboxRequest{
				GoalRef: system.goalRef, MessageRef: admission.Record.Envelope.Ref,
				RecipientWorkItemRef: system.parentRef, RecipientExecutionRef: system.parentExecution,
			})
			if err != nil || retired.State != MailboxStateRetired || retired.Retirement == nil ||
				retired.Acknowledgement != nil || retired.Retirement.MessageRef != retired.Envelope.Ref ||
				retired.Retirement.ActionRef != retired.Action.Ref ||
				retired.Retirement.RecipientExecutionRef != system.parentExecution ||
				retired.Retirement.FailureCode != failed.FailureCode ||
				!retired.Retirement.RetiredAt.Equal(failed.FinishedAt) ||
				!reflect.DeepEqual(retired.Attempts, frontier.record.Attempts) ||
				ValidateMailboxRetirement(retired) != nil {
				t.Fatalf("retired frontier=%+v err=%v", retired, err)
			}
			system.repository.mu.Lock()
			_, mailboxActionLive := system.repository.actions[retired.Action.Ref]
			_, observationActionLive := system.repository.actions[action.Ref]
			system.repository.mu.Unlock()
			if mailboxActionLive || observationActionLive {
				t.Fatalf("retired actions remain live: mailbox=%v observation=%v",
					mailboxActionLive, observationActionLive)
			}
			if test.depth >= 1 {
				beforeReplay := system.mailboxCounts()
				if _, replayErr := system.orchestrator.ClaimMailbox(
					ctx, system.recipientAccess,
					system.claimRequest(admission.Record, "mailbox-claim:"+test.name),
				); !IsStateError(replayErr, StateConflict) {
					t.Fatalf("retired %s mailbox exposed historical claim replay: %v", test.name, replayErr)
				}
				if got := system.mailboxCounts(); got != beforeReplay {
					t.Fatalf("retired claim replay wrote mailbox state: got=%v want=%v", got, beforeReplay)
				}
			}

			if test.depth == 3 {
				preserved, ok := mailboxConsumptionReceipt(
					failedRecord.ConsumptionReceipts, admission.Record.Envelope.Ref,
				)
				attempt := retired.Attempts[len(retired.Attempts)-1]
				if !ok || preserved != consumedReceipt || preserved.Kind != ActionDeliverMailbox ||
					preserved.Outcome != ActionConsumedCompleted ||
					preserved.ActionRef != retired.Action.Ref ||
					preserved.ClaimToken != attempt.ClaimToken ||
					!preserved.ConsumedAt.Equal(attempt.ConsumedAt) {
					t.Fatalf("consumption receipt not preserved: before=%+v after=%+v", consumedReceipt, preserved)
				}
			}

			beforeStale := system.mailboxCounts()
			if err := advanceRetiredMailboxWithStaleHolder(
				ctx, system, admission, frontier.claim, failedRecord, test.name, test.depth,
			); !IsStateError(err, StateConflict) {
				t.Fatalf("stale holder advanced retired %s mailbox: %v", test.name, err)
			}
			if got := system.mailboxCounts(); got != beforeStale {
				t.Fatalf("stale holder wrote mailbox state: got=%v want=%v", got, beforeStale)
			}
			afterStale, err := system.orchestrator.GetMailbox(ctx, system.recipientAccess, GetMailboxRequest{
				GoalRef: system.goalRef, MessageRef: admission.Record.Envelope.Ref,
				RecipientWorkItemRef: system.parentRef, RecipientExecutionRef: system.parentExecution,
			})
			if err != nil || !reflect.DeepEqual(afterStale, retired) {
				t.Fatalf("stale holder changed retirement: record=%+v err=%v", afterStale, err)
			}
		})
	}
}

func TestMailboxRetirementRejectsTimeBeforeLatestActivity(t *testing.T) {
	system := newMailboxTestSystem(t, 1)
	admission := system.admit(t, 0, "mailbox-admit:retirement-time")
	frontier := advanceMailboxRetirementFrontier(t, system, admission, "retirement-time", 3)
	attempt := frontier.record.Attempts[len(frontier.record.Attempts)-1]
	record, err := system.repository.GetGoal(context.Background(), system.goalRef)
	if err != nil {
		t.Fatal(err)
	}
	failed, found := executionForAction(record, ActionRecord{
		WorkItemRef: system.parentRef, ExecutionRef: system.parentExecution,
	})
	if !found {
		t.Fatal("recipient execution missing")
	}
	failed.State = ExecutionFailed
	failed.FailureCode = "agent.retirement_time"
	failed.FinishedAt = attempt.ClaimedAt
	if _, err := BuildMailboxRetirement(frontier.record, failed); err == nil ||
		err.Error() != "application.mailbox_retirement_before_activity" {
		t.Fatalf("BuildMailboxRetirement accepted time before activity: %v", err)
	}

	failed.FinishedAt = attempt.ConsumedAt.Add(time.Second)
	retirement, err := BuildMailboxRetirement(frontier.record, failed)
	if err != nil {
		t.Fatalf("BuildMailboxRetirement(valid) error=%v", err)
	}
	candidate := cloneMailboxRecord(frontier.record)
	candidate.State = MailboxStateRetired
	candidate.Retirement = &retirement
	if err := ValidateMailboxRetirement(candidate); err != nil {
		t.Fatalf("ValidateMailboxRetirement(valid) error=%v", err)
	}
	tampered := retirement
	tampered.RetiredAt = attempt.DeliveredAt
	candidate.Retirement = &tampered
	if err := ValidateMailboxRetirement(candidate); err == nil ||
		err.Error() != "application.mailbox_retirement_before_activity" {
		t.Fatalf("ValidateMailboxRetirement accepted time before consumption: %v", err)
	}
}

type mailboxRetirementFrontierFixture struct {
	record MailboxRecord
	claim  MailboxClaim
}

func advanceMailboxRetirementFrontier(
	t *testing.T,
	system *mailboxTestSystem,
	admission MailboxAdmissionResult,
	suffix string,
	depth int,
) mailboxRetirementFrontierFixture {
	t.Helper()
	ctx := context.Background()
	fixture := mailboxRetirementFrontierFixture{record: admission.Record}
	if depth < 1 {
		return fixture
	}
	system.clock.Advance(time.Second)
	claimed, err := system.orchestrator.ClaimMailbox(
		ctx, system.recipientAccess,
		system.claimRequest(admission.Record, "mailbox-claim:"+suffix),
	)
	if err != nil || !claimed.Claimed {
		t.Fatalf("claim %s: result=%+v err=%v", suffix, claimed, err)
	}
	fixture.record = claimed.Claim.Record
	fixture.claim = claimed.Claim
	if depth < 2 {
		return fixture
	}
	system.clock.Advance(time.Second)
	delivered, err := system.orchestrator.MarkMailboxDelivered(
		ctx, system.recipientAccess,
		system.deliverRequest(fixture.claim, "mailbox-deliver:"+suffix),
	)
	if err != nil || !delivered.Changed {
		t.Fatalf("deliver %s: result=%+v err=%v", suffix, delivered, err)
	}
	fixture.record = delivered.Record
	if depth < 3 {
		return fixture
	}
	system.clock.Advance(time.Second)
	consumed, err := system.orchestrator.ConsumeMailbox(ctx, system.recipientAccess, ConsumeMailboxRequest{
		RequestRef: "mailbox-consume:" + suffix, GoalRef: system.goalRef,
		MessageRef: admission.Record.Envelope.Ref, RecipientWorkItemRef: system.parentRef,
		RecipientExecutionRef: system.parentExecution, ClaimToken: fixture.claim.Attempt.ClaimToken,
		Fence: fixture.claim.Attempt.Fence,
	})
	if err != nil || !consumed.Changed {
		t.Fatalf("consume %s: result=%+v err=%v", suffix, consumed, err)
	}
	fixture.record = consumed.Record
	return fixture
}

func advanceRetiredMailboxWithStaleHolder(
	ctx context.Context,
	system *mailboxTestSystem,
	admission MailboxAdmissionResult,
	claim MailboxClaim,
	failedRecord GoalRecord,
	suffix string,
	depth int,
) error {
	switch depth {
	case 0:
		_, err := system.orchestrator.ClaimMailbox(
			ctx, system.recipientAccess,
			system.claimRequest(admission.Record, "mailbox-stale-claim:"+suffix),
		)
		return err
	case 1:
		_, err := system.orchestrator.MarkMailboxDelivered(
			ctx, system.recipientAccess,
			system.deliverRequest(claim, "mailbox-stale-deliver:"+suffix),
		)
		return err
	case 2:
		_, err := system.orchestrator.ConsumeMailbox(ctx, system.recipientAccess, ConsumeMailboxRequest{
			RequestRef: "mailbox-stale-consume:" + suffix, GoalRef: system.goalRef,
			MessageRef: admission.Record.Envelope.Ref, RecipientWorkItemRef: system.parentRef,
			RecipientExecutionRef: system.parentExecution, ClaimToken: claim.Attempt.ClaimToken,
			Fence: claim.Attempt.Fence,
		})
		return err
	default:
		_, err := system.orchestrator.AcknowledgeMailbox(
			ctx, system.recipientAccess, AcknowledgeMailboxRequest(ResolveMailboxRequest{
				RequestRef: "mailbox-stale-ack:" + suffix, GoalRef: system.goalRef,
				MessageRef: admission.Record.Envelope.Ref, RecipientWorkItemRef: system.parentRef,
				RecipientExecutionRef: system.parentExecution, ClaimToken: claim.Attempt.ClaimToken,
				Fence:                  claim.Attempt.Fence,
				ExpectedGoalRevision:   failedRecord.Goal.Revision(),
				ExpectedPlanGeneration: failedRecord.Goal.PlanGeneration(),
				EffectOrReworkRef:      "effect:must-not-advance-retired",
			}),
		)
		return err
	}
}

func mailboxConsumptionReceipt(
	receipts []ActionConsumptionReceipt,
	messageRef MailboxMessageRef,
) (ActionConsumptionReceipt, bool) {
	for _, receipt := range receipts {
		if receipt.MailboxMessageRef == messageRef {
			return receipt, true
		}
	}
	return ActionConsumptionReceipt{}, false
}
