package application

import (
	"context"
	"reflect"
	"testing"
	"time"

	"orquesta/internal/goal"
)

func TestBuildMailboxResolutionGoalMapsAcknowledgedAndBlockedTransitions(t *testing.T) {
	system := newMailboxTestSystem(t, 2)
	acknowledgedFlow := system.consume(t, 0, "goal-helper-acknowledged")
	blockedFlow := system.consume(t, 1, "goal-helper-blocked")
	current, err := system.repository.GetGoal(context.Background(), system.goalRef)
	if err != nil {
		t.Fatal(err)
	}
	original := current.Goal.Snapshot()
	acknowledgedAt := system.clock.Now().Add(time.Second)

	acknowledged, err := BuildMailboxResolutionGoal(MailboxResolutionGoalInput{
		Current: current.Goal, ExpectedRevision: current.Goal.Revision(),
		ParentWorkItemRef: system.parentRef, ChildWorkItemRef: system.children[0],
		MessageRef: acknowledgedFlow.admission.Record.Envelope.Ref,
		Outcome:    MailboxOutcomeAcknowledged, ReceiptRef: "mailbox-ack:goal-helper-acknowledged",
		ResolvedAt: acknowledgedAt,
	})
	if err != nil {
		t.Fatalf("acknowledged transition error = %v", err)
	}
	if acknowledged.Revision() != current.Goal.Revision()+1 ||
		acknowledged.State() != goal.GoalStateRunning || acknowledged.IsTerminal() {
		t.Fatalf("acknowledged transition = state %q revision %d", acknowledged.State(), acknowledged.Revision())
	}
	resolutions := acknowledged.ChildHandoffResolutions()
	if len(resolutions) != 1 || resolutions[0].ParentRef() != system.parentRef ||
		resolutions[0].ChildRef() != system.children[0] ||
		resolutions[0].MessageRef() != acknowledgedFlow.admission.Record.Envelope.Ref.String() ||
		resolutions[0].Outcome() != goal.ChildHandoffAcknowledged ||
		resolutions[0].ReceiptRef() != "mailbox-ack:goal-helper-acknowledged" ||
		!resolutions[0].ResolvedAt().Equal(acknowledgedAt) {
		t.Fatalf("acknowledged resolution = %+v", resolutions)
	}
	if !reflect.DeepEqual(current.Goal.Snapshot(), original) {
		t.Fatal("pure transition mutated its input Goal")
	}

	blockedAt := acknowledgedAt.Add(time.Second)
	blocked, err := BuildMailboxResolutionGoal(MailboxResolutionGoalInput{
		Current: acknowledged, ExpectedRevision: acknowledged.Revision(),
		ParentWorkItemRef: system.parentRef, ChildWorkItemRef: system.children[1],
		MessageRef: blockedFlow.admission.Record.Envelope.Ref,
		Outcome:    MailboxOutcomeBlocked, ReceiptRef: "mailbox-ack:goal-helper-blocked",
		ResolvedAt: blockedAt,
	})
	if err != nil {
		t.Fatalf("blocked transition error = %v", err)
	}
	resolutions = blocked.ChildHandoffResolutions()
	if blocked.Revision() != acknowledged.Revision()+1 || blocked.State() != goal.GoalStateRunning ||
		len(resolutions) != 2 || resolutions[1].ParentRef() != system.parentRef ||
		resolutions[1].ChildRef() != system.children[1] ||
		resolutions[1].MessageRef() != blockedFlow.admission.Record.Envelope.Ref.String() ||
		resolutions[1].Outcome() != goal.ChildHandoffBlocked ||
		resolutions[1].ReceiptRef() != "mailbox-ack:goal-helper-blocked" ||
		!resolutions[1].ResolvedAt().Equal(blockedAt) {
		t.Fatalf("blocked transition = goal %+v resolutions %+v", blocked.Snapshot(), resolutions)
	}
	if len(acknowledged.ChildHandoffResolutions()) != 1 {
		t.Fatal("second pure transition mutated its input Goal")
	}
}

func TestBuildMailboxResolutionGoalClosesWhenResolutionLeavesGoalClosable(t *testing.T) {
	system := newMailboxTestSystem(t, 1)
	flow := system.consume(t, 0, "goal-helper-close")
	current, err := system.repository.GetGoal(context.Background(), system.goalRef)
	if err != nil {
		t.Fatal(err)
	}
	resolvedAt := system.clock.Now().Add(time.Second)
	input := MailboxResolutionGoalInput{
		Current: current.Goal, ExpectedRevision: current.Goal.Revision(),
		ParentWorkItemRef: system.parentRef, ChildWorkItemRef: system.children[0],
		MessageRef: flow.admission.Record.Envelope.Ref,
		Outcome:    MailboxOutcomeAcknowledged, ReceiptRef: "mailbox-ack:goal-helper-close",
		ResolvedAt: resolvedAt,
	}
	resolved, err := BuildMailboxResolutionGoal(input)
	if err != nil {
		t.Fatalf("initial resolution error = %v", err)
	}
	parent, ok := resolved.WorkItem(system.parentRef)
	if !ok {
		t.Fatal("parent WorkItem missing")
	}
	parentSucceeded, err := resolved.SucceedWorkItem(
		resolved.Revision(), parent.Revision(), system.parentRef,
		[]goal.ArtifactRef{mailboxMustRef(t, "artifact:goal-helper-parent", goal.NewArtifactRef)},
		[]goal.AttestationRef{mailboxMustRef(t, "attestation:goal-helper-parent", goal.NewAttestationRef)},
		resolvedAt,
	)
	if err != nil {
		t.Fatalf("parent success error = %v", err)
	}
	if outcome, closable := parentSucceeded.ClosableOutcome(); !closable || outcome != goal.GoalOutcomeSucceeded {
		t.Fatalf("parent-complete Goal closable outcome = %q/%v", outcome, closable)
	}

	// An exact handoff replay is immutable even with its original CAS. It gives
	// the helper a closable running Goal and exercises the optional close path.
	input.Current = parentSucceeded
	closed, err := BuildMailboxResolutionGoal(input)
	if err != nil {
		t.Fatalf("resolution plus close error = %v", err)
	}
	if closed.State() != goal.GoalStateSucceeded || !closed.IsTerminal() ||
		closed.Revision() != parentSucceeded.Revision()+1 {
		t.Fatalf("closed Goal = state %q revision %d", closed.State(), closed.Revision())
	}
	closedAt, ok := closed.ClosedAt()
	if !ok || !closedAt.Equal(resolvedAt) {
		t.Fatalf("closed_at = %v/%v, want %v", closedAt, ok, resolvedAt)
	}
	if parentSucceeded.State() != goal.GoalStateRunning {
		t.Fatal("close transition mutated its input Goal")
	}
}

func TestBuildMailboxResolutionGoalRejectsInvalidOutcomeAndRevision(t *testing.T) {
	system := newMailboxTestSystem(t, 1)
	flow := system.consume(t, 0, "goal-helper-invalid")
	current, err := system.repository.GetGoal(context.Background(), system.goalRef)
	if err != nil {
		t.Fatal(err)
	}
	base := MailboxResolutionGoalInput{
		Current: current.Goal, ExpectedRevision: current.Goal.Revision(),
		ParentWorkItemRef: system.parentRef, ChildWorkItemRef: system.children[0],
		MessageRef: flow.admission.Record.Envelope.Ref,
		Outcome:    MailboxOutcomeAcknowledged, ReceiptRef: "mailbox-ack:goal-helper-invalid",
		ResolvedAt: system.clock.Now().Add(time.Second),
	}

	invalidOutcome := base
	invalidOutcome.Outcome = MailboxOutcome("lost")
	if _, err = BuildMailboxResolutionGoal(invalidOutcome); err == nil ||
		err.Error() != "application.mailbox_resolution_outcome_invalid" {
		t.Fatalf("invalid outcome error = %v", err)
	}

	stale := base
	stale.ExpectedRevision++
	if _, err = BuildMailboxResolutionGoal(stale); goal.ErrorCodeOf(err) != goal.ErrorRevisionConflict {
		t.Fatalf("stale revision error = %v", err)
	}
	if len(current.Goal.ChildHandoffResolutions()) != 0 {
		t.Fatal("rejected transition mutated its input Goal")
	}
}
