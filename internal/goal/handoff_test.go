package goal_test

import (
	"fmt"
	"reflect"
	"testing"
	"time"

	domain "orquesta/internal/goal"
)

func TestChildHandoffsGateParentSuccessAndAcceptAcknowledgedOrBlockedChildren(t *testing.T) {
	running, parentRef, children := newRunningHandoffGoal(t, 2)
	running = succeedGoalItem(t, running, children[0], "handoff-child-a", baseTime().Add(5*time.Minute))
	running = succeedGoalItem(t, running, children[1], "handoff-child-b", baseTime().Add(6*time.Minute))
	parent, _ := running.WorkItem(parentRef)

	_, err := running.SucceedWorkItem(
		running.Revision(), parent.Revision(), parentRef,
		[]domain.ArtifactRef{mustRef(t, "artifact:handoff-parent", domain.NewArtifactRef)},
		[]domain.AttestationRef{mustRef(t, "attestation:handoff-parent", domain.NewAttestationRef)},
		baseTime().Add(7*time.Minute),
	)
	requireCode(t, err, domain.ErrorChildHandoffsPending)

	beforeFirst := running.Revision()
	running, err = running.ResolveChildHandoff(
		beforeFirst, parentRef, children[0], "message:child-a",
		domain.ChildHandoffAcknowledged, "receipt:child-a", baseTime().Add(7*time.Minute),
	)
	if err != nil {
		t.Fatalf("ResolveChildHandoff(acknowledged) error = %v", err)
	}
	if running.Revision() != beforeFirst+1 {
		t.Fatalf("revision after first resolution = %d, want %d", running.Revision(), beforeFirst+1)
	}
	replayed, err := running.ResolveChildHandoff(
		beforeFirst, parentRef, children[0], "message:child-a",
		domain.ChildHandoffAcknowledged, "receipt:child-a", baseTime().Add(7*time.Minute),
	)
	if err != nil || !reflect.DeepEqual(replayed.Snapshot(), running.Snapshot()) {
		t.Fatalf("exact resolution replay changed aggregate: err=%v", err)
	}
	_, err = running.ResolveChildHandoff(
		running.Revision(), parentRef, children[0], "message:child-a",
		domain.ChildHandoffBlocked, "receipt:contradiction", baseTime().Add(7*time.Minute),
	)
	requireCode(t, err, domain.ErrorChildHandoffConflict)

	_, err = running.SucceedWorkItem(
		running.Revision(), parent.Revision(), parentRef,
		[]domain.ArtifactRef{mustRef(t, "artifact:parent-before-all", domain.NewArtifactRef)},
		[]domain.AttestationRef{mustRef(t, "attestation:parent-before-all", domain.NewAttestationRef)},
		baseTime().Add(8*time.Minute),
	)
	requireCode(t, err, domain.ErrorChildHandoffsPending)
	if outcome, closable := running.ClosableOutcome(); closable || outcome != "" {
		t.Fatalf("goal closable with one unresolved child: %q/%v", outcome, closable)
	}

	running, err = running.ResolveChildHandoff(
		running.Revision(), parentRef, children[1], "message:child-b",
		domain.ChildHandoffBlocked, "receipt:child-b-blocked", baseTime().Add(8*time.Minute),
	)
	if err != nil {
		t.Fatalf("ResolveChildHandoff(blocked) error = %v", err)
	}
	resolutions := running.ChildHandoffResolutions()
	if len(resolutions) != 2 || resolutions[0].ParentRef() != parentRef ||
		resolutions[0].ChildRef() != children[0] || resolutions[0].MessageRef() != "message:child-a" ||
		resolutions[0].Outcome() != domain.ChildHandoffAcknowledged ||
		resolutions[0].ReceiptRef() != "receipt:child-a" ||
		!resolutions[0].ResolvedAt().Equal(baseTime().Add(7*time.Minute)) ||
		resolutions[1].Outcome() != domain.ChildHandoffBlocked {
		t.Fatalf("child handoff resolutions = %+v", resolutions)
	}
	resolutions[0] = resolutions[1]
	if running.ChildHandoffResolutions()[0].ChildRef() != children[0] {
		t.Fatal("ChildHandoffResolutions exposed mutable aggregate slice")
	}

	parent, _ = running.WorkItem(parentRef)
	running, err = running.SucceedWorkItem(
		running.Revision(), parent.Revision(), parentRef,
		[]domain.ArtifactRef{mustRef(t, "artifact:handoff-parent", domain.NewArtifactRef)},
		[]domain.AttestationRef{mustRef(t, "attestation:handoff-parent", domain.NewAttestationRef)},
		baseTime().Add(9*time.Minute),
	)
	if err != nil {
		t.Fatalf("SucceedWorkItem(parent) error = %v", err)
	}
	if outcome, closable := running.ClosableOutcome(); !closable || outcome != domain.GoalOutcomeSucceeded {
		t.Fatalf("ClosableOutcome after resolutions = %q/%v", outcome, closable)
	}
	closed, err := running.Close(running.Revision(), domain.GoalOutcomeSucceeded, baseTime().Add(10*time.Minute))
	if err != nil || closed.State() != domain.GoalStateSucceeded {
		t.Fatalf("Close after child resolutions: state=%q err=%v", closed.State(), err)
	}
	closedReplay, err := closed.ResolveChildHandoff(
		beforeFirst, parentRef, children[0], "message:child-a",
		domain.ChildHandoffAcknowledged, "receipt:child-a", baseTime().Add(7*time.Minute),
	)
	if err != nil || !reflect.DeepEqual(closedReplay.Snapshot(), closed.Snapshot()) {
		t.Fatalf("exact resolution replay after Goal closure changed aggregate: err=%v", err)
	}
}

func TestChildHandoffResolutionRejectsWrongRelationNonterminalChildAndMalformedFact(t *testing.T) {
	running, parentRef, children := newRunningHandoffGoal(t, 2)
	_, err := running.ResolveChildHandoff(
		running.Revision(), parentRef, children[0], "message:too-early",
		domain.ChildHandoffAcknowledged, "receipt:too-early", baseTime().Add(5*time.Minute),
	)
	requireCode(t, err, domain.ErrorChildHandoffInvalid)

	running = succeedGoalItem(t, running, children[0], "validation-child-a", baseTime().Add(6*time.Minute))
	tests := []struct {
		name       string
		parent     domain.WorkItemRef
		child      domain.WorkItemRef
		messageRef string
		outcome    domain.ChildHandoffOutcome
		receiptRef string
		at         time.Time
	}{
		{name: "wrong direct relation", parent: children[1], child: children[0], messageRef: "message:wrong", outcome: domain.ChildHandoffAcknowledged, receiptRef: "receipt:wrong", at: baseTime().Add(7 * time.Minute)},
		{name: "missing parent", child: children[0], messageRef: "message:missing-parent", outcome: domain.ChildHandoffAcknowledged, receiptRef: "receipt:missing-parent", at: baseTime().Add(7 * time.Minute)},
		{name: "empty message", parent: parentRef, child: children[0], outcome: domain.ChildHandoffAcknowledged, receiptRef: "receipt:empty-message", at: baseTime().Add(7 * time.Minute)},
		{name: "unknown outcome", parent: parentRef, child: children[0], messageRef: "message:unknown", outcome: domain.ChildHandoffOutcome("delivered"), receiptRef: "receipt:unknown", at: baseTime().Add(7 * time.Minute)},
		{name: "empty receipt", parent: parentRef, child: children[0], messageRef: "message:empty-receipt", outcome: domain.ChildHandoffAcknowledged, at: baseTime().Add(7 * time.Minute)},
		{name: "before child finish", parent: parentRef, child: children[0], messageRef: "message:early-time", outcome: domain.ChildHandoffAcknowledged, receiptRef: "receipt:early-time", at: baseTime().Add(5 * time.Minute)},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, resolveErr := running.ResolveChildHandoff(
				running.Revision(), test.parent, test.child, test.messageRef,
				test.outcome, test.receiptRef, test.at,
			)
			requireCode(t, resolveErr, domain.ErrorChildHandoffInvalid)
		})
	}
}

func TestFailedParentDoesNotRequireTerminalChildDeliveryBeforeGoalClosure(t *testing.T) {
	running, parentRef, children := newRunningHandoffGoal(t, 1)
	parent, _ := running.WorkItem(parentRef)
	var err error
	running, err = running.FailWorkItem(
		running.Revision(), parent.Revision(), parentRef, baseTime().Add(5*time.Minute),
	)
	if err != nil {
		t.Fatalf("FailWorkItem(parent) error = %v", err)
	}
	running = succeedGoalItem(t, running, children[0], "failed-parent-child", baseTime().Add(6*time.Minute))
	if len(running.ChildHandoffResolutions()) != 0 || !running.ChildHandoffsResolved(parentRef) {
		t.Fatalf("failed parent still waits for child delivery: %+v", running.ChildHandoffResolutions())
	}
	if outcome, closable := running.ClosableOutcome(); !closable || outcome != domain.GoalOutcomeFailed {
		t.Fatalf("failed Goal not closable with terminal child: %q/%v", outcome, closable)
	}
	closed, err := running.Close(running.Revision(), domain.GoalOutcomeFailed, baseTime().Add(7*time.Minute))
	if err != nil {
		t.Fatalf("Close(failed parent) error = %v", err)
	}
	restored, err := domain.RestoreGoal(closed.Snapshot())
	if err != nil || !reflect.DeepEqual(restored.Snapshot(), closed.Snapshot()) {
		t.Fatalf("failed parent round trip error=%v", err)
	}
}

func TestFailedRequiredChildBlocksCausallyWithoutMailboxResolution(t *testing.T) {
	running, parentRef, children := newRunningHandoffGoal(t, 1)
	childRef := children[0]
	child, _ := running.WorkItem(childRef)
	failed, err := running.FailWorkItem(
		running.Revision(), child.Revision(), childRef, baseTime().Add(5*time.Minute),
	)
	if err != nil {
		t.Fatalf("FailWorkItem(child) error = %v", err)
	}
	_, err = failed.ResolveChildHandoff(
		failed.Revision(), parentRef, childRef, "message:failed-child",
		domain.ChildHandoffBlocked, "receipt:failed-child", baseTime().Add(6*time.Minute),
	)
	requireCode(t, err, domain.ErrorChildHandoffInvalid)
	if len(failed.ChildHandoffResolutions()) != 0 || !failed.ChildHandoffsResolved(parentRef) {
		t.Fatalf("failed child did not satisfy handoff causally: %+v", failed.ChildHandoffResolutions())
	}
	parent, _ := failed.WorkItem(parentRef)
	failed, err = failed.SucceedWorkItem(
		failed.Revision(), parent.Revision(), parentRef,
		[]domain.ArtifactRef{mustRef(t, "artifact:failed-child-parent", domain.NewArtifactRef)},
		[]domain.AttestationRef{mustRef(t, "attestation:failed-child-parent", domain.NewAttestationRef)},
		baseTime().Add(6*time.Minute),
	)
	if err != nil {
		t.Fatalf("SucceedWorkItem(parent after failed child) error = %v", err)
	}
	closed, err := failed.Close(failed.Revision(), domain.GoalOutcomeFailed, baseTime().Add(7*time.Minute))
	if err != nil {
		t.Fatalf("Close(failed child) error = %v", err)
	}
	restored, err := domain.RestoreGoal(closed.Snapshot())
	if err != nil || !reflect.DeepEqual(restored.Snapshot(), closed.Snapshot()) ||
		len(restored.ChildHandoffResolutions()) != 0 {
		t.Fatalf("failed child round trip error=%v resolutions=%+v", err, restored.ChildHandoffResolutions())
	}
}

func TestChildHandoffSnapshotRoundTripAndTamper(t *testing.T) {
	closed, _, _ := closedResolvedHandoffGoal(t)
	snapshot := closed.Snapshot()
	restored, err := domain.RestoreGoal(snapshot)
	if err != nil || !reflect.DeepEqual(restored.Snapshot(), snapshot) {
		t.Fatalf("resolved handoff snapshot round trip: err=%v", err)
	}

	tests := []struct {
		name   string
		mutate func(*domain.GoalSnapshot)
	}{
		{name: "empty message", mutate: func(value *domain.GoalSnapshot) { value.ChildHandoffResolutions[0].MessageRef = "" }},
		{name: "unknown outcome", mutate: func(value *domain.GoalSnapshot) {
			value.ChildHandoffResolutions[0].Outcome = domain.ChildHandoffOutcome("lost")
		}},
		{name: "wrong relation", mutate: func(value *domain.GoalSnapshot) {
			value.ChildHandoffResolutions[0].ParentRef = value.ChildHandoffResolutions[0].ChildRef
		}},
		{name: "duplicate relation", mutate: func(value *domain.GoalSnapshot) {
			value.ChildHandoffResolutions = append(value.ChildHandoffResolutions, value.ChildHandoffResolutions[0])
		}},
		{name: "before child finish", mutate: func(value *domain.GoalSnapshot) {
			value.ChildHandoffResolutions[0].ResolvedAt = value.WorkItems[1].FinishedAt.Add(-time.Second)
		}},
		{name: "after goal close", mutate: func(value *domain.GoalSnapshot) {
			value.ChildHandoffResolutions[0].ResolvedAt = value.ClosedAt.Add(time.Second)
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			candidate := cloneGoalSnapshot(snapshot)
			test.mutate(&candidate)
			_, restoreErr := domain.RestoreGoal(candidate)
			requireCode(t, restoreErr, domain.ErrorSnapshotInvalid)
		})
	}
}

func TestRequiredDependencySkippedChildClosesWithoutSyntheticMailboxFact(t *testing.T) {
	fixture := newPlanFixture(t)
	phase := mustPhase(t, "phase:skipped-handoff")
	failureRef := mustRef(t, "work-item:skipped-failure", domain.NewWorkItemRef)
	parentRef := mustRef(t, "work-item:skipped-parent", domain.NewWorkItemRef)
	childRef := mustRef(t, "work-item:skipped-child", domain.NewWorkItemRef)
	failure := fixture.item(t, failureRef, phase.Key(), nil, nil)
	parent := fixture.item(t, parentRef, phase.Key(), nil, nil)
	child, err := fixture.newItem(domain.NewWorkItemInput{
		Ref: childRef, Phase: phase.Key(), Parent: parentRef, HandoffRequired: true,
		Dependencies: []domain.WorkItemRef{failureRef},
	})
	if err != nil {
		t.Fatal(err)
	}
	running := applyAndStartPlan(t, fixture.goal, domain.PlanInput{
		Generation: 1, Phases: []domain.PhaseInstance{phase},
		WorkItems: []domain.WorkItem{failure, parent, child},
	})
	running = startGoalItem(t, running, failureRef, "execution:skipped-failure", baseTime().Add(4*time.Minute))
	running = startGoalItem(t, running, parentRef, "execution:skipped-parent", baseTime().Add(4*time.Minute))
	failureRunning, _ := running.WorkItem(failureRef)
	beforeFailure := running.Revision()
	failed, err := running.FailWorkItem(
		beforeFailure, failureRunning.Revision(), failureRef, baseTime().Add(5*time.Minute),
	)
	if err != nil {
		t.Fatal(err)
	}
	replayedFailure, err := running.FailWorkItem(
		beforeFailure, failureRunning.Revision(), failureRef, baseTime().Add(5*time.Minute),
	)
	if err != nil || !reflect.DeepEqual(replayedFailure.Snapshot(), failed.Snapshot()) {
		t.Fatalf("deterministic failure replay error=%v", err)
	}
	skipped, _ := failed.WorkItem(childRef)
	resolutions := failed.ChildHandoffResolutions()
	if !skipped.HandoffRequired() || skipped.State() != domain.WorkItemStateSkipped ||
		failed.Revision() != beforeFailure+1 || len(resolutions) != 0 {
		t.Fatalf("skipped handoff=%+v goal_revision=%d", resolutions, failed.Revision())
	}
	if reason, ok := skipped.SkipReason(); !ok || reason != domain.WorkItemSkipReasonDependencyFailed {
		t.Fatalf("skip reason=%q/%v", reason, ok)
	}
	parentRunning, _ := failed.WorkItem(parentRef)
	failed, err = failed.SucceedWorkItem(
		failed.Revision(), parentRunning.Revision(), parentRef,
		[]domain.ArtifactRef{mustRef(t, "artifact:skipped-parent", domain.NewArtifactRef)},
		[]domain.AttestationRef{mustRef(t, "attestation:skipped-parent", domain.NewAttestationRef)},
		baseTime().Add(6*time.Minute),
	)
	if err != nil {
		t.Fatalf("blocked child did not release parent: %v", err)
	}
	closed, err := failed.Close(failed.Revision(), domain.GoalOutcomeFailed, baseTime().Add(7*time.Minute))
	if err != nil || closed.State() != domain.GoalStateFailed {
		t.Fatalf("Close(failed skipped handoff): state=%q err=%v", closed.State(), err)
	}
	restored, err := domain.RestoreGoal(closed.Snapshot())
	if err != nil || !reflect.DeepEqual(restored.Snapshot(), closed.Snapshot()) {
		t.Fatalf("skipped handoff restart error=%v", err)
	}
	restoredChild, found := restored.WorkItem(childRef)
	restoredReason, hasReason := restoredChild.SkipReason()
	if !found || !restoredChild.HandoffRequired() || !hasReason ||
		restoredReason != domain.WorkItemSkipReasonDependencyFailed ||
		len(restored.ChildHandoffResolutions()) != 0 {
		t.Fatalf("restored skipped child=%+v reason=%q/%v", restoredChild, restoredReason, hasReason)
	}
}

func TestSchemaV4RequiresExplicitCoherentHandoffField(t *testing.T) {
	pending, _, _ := newPendingHandoffGoal(t, 1)
	missing := pending.Snapshot()
	missing.WorkItems[1].HandoffRequired = nil
	_, err := domain.RestoreGoal(missing)
	requireCode(t, err, domain.ErrorSnapshotInvalid)

	incoherent := pending.Snapshot()
	required := true
	incoherent.WorkItems[0].HandoffRequired = &required
	_, err = domain.RestoreGoal(incoherent)
	requireCode(t, err, domain.ErrorSnapshotInvalid)

	wrongLegacyShape := pending.Snapshot()
	wrongLegacyShape.SchemaVersion = domain.GoalSnapshotSchemaVersion - 2
	_, err = domain.RestoreGoal(wrongLegacyShape)
	requireCode(t, err, domain.ErrorSnapshotInvalid)
}

func newPendingHandoffGoal(t *testing.T, childCount int) (domain.Goal, domain.WorkItemRef, []domain.WorkItemRef) {
	t.Helper()
	fixture := newPlanFixture(t)
	phase := mustPhase(t, "phase:handoff")
	parentRef := mustRef(t, "work-item:handoff-parent", domain.NewWorkItemRef)
	parent := fixture.item(t, parentRef, phase.Key(), nil, nil)
	items := []domain.WorkItem{parent}
	children := make([]domain.WorkItemRef, 0, childCount)
	for index := 0; index < childCount; index++ {
		childRef := mustRef(t, fmt.Sprintf("work-item:handoff-child-%d", index+1), domain.NewWorkItemRef)
		child, err := fixture.newItem(domain.NewWorkItemInput{
			Ref: childRef, Phase: phase.Key(), Parent: parentRef, HandoffRequired: true,
		})
		if err != nil {
			t.Fatalf("NewWorkItem(child %d) error = %v", index, err)
		}
		items = append(items, child)
		children = append(children, childRef)
	}
	plan := mustPlan(t, domain.PlanInput{
		Generation: 1, Phases: []domain.PhaseInstance{phase}, WorkItems: items,
	})
	planned, err := fixture.goal.ApplyPlan(fixture.goal.Revision(), plan)
	if err != nil {
		t.Fatalf("ApplyPlan(handoff) error = %v", err)
	}
	return planned, parentRef, children
}

func newRunningHandoffGoal(t *testing.T, childCount int) (domain.Goal, domain.WorkItemRef, []domain.WorkItemRef) {
	t.Helper()
	planned, parentRef, children := newPendingHandoffGoal(t, childCount)
	running, err := planned.Start(planned.Revision(), baseTime().Add(3*time.Minute))
	if err != nil {
		t.Fatalf("Start(handoff Goal) error = %v", err)
	}
	running = startGoalItem(t, running, parentRef, "execution:handoff-parent", baseTime().Add(4*time.Minute))
	for index, childRef := range children {
		running = startGoalItem(
			t, running, childRef, fmt.Sprintf("execution:handoff-child-%d", index+1), baseTime().Add(4*time.Minute),
		)
	}
	return running, parentRef, children
}

func resolvedRunningHandoffGoal(t *testing.T) (domain.Goal, domain.WorkItemRef, domain.WorkItemRef) {
	t.Helper()
	running, parentRef, children := newRunningHandoffGoal(t, 1)
	childRef := children[0]
	running = succeedGoalItem(t, running, childRef, "snapshot-handoff-child", baseTime().Add(5*time.Minute))
	var err error
	running, err = running.ResolveChildHandoff(
		running.Revision(), parentRef, childRef, "message:snapshot-handoff",
		domain.ChildHandoffAcknowledged, "receipt:snapshot-handoff", baseTime().Add(6*time.Minute),
	)
	if err != nil {
		t.Fatalf("ResolveChildHandoff(snapshot) error = %v", err)
	}
	running = succeedGoalItem(t, running, parentRef, "snapshot-handoff-parent", baseTime().Add(7*time.Minute))
	return running, parentRef, childRef
}

func closedResolvedHandoffGoal(t *testing.T) (domain.Goal, domain.WorkItemRef, domain.WorkItemRef) {
	t.Helper()
	running, parentRef, childRef := resolvedRunningHandoffGoal(t)
	closed, err := running.Close(running.Revision(), domain.GoalOutcomeSucceeded, baseTime().Add(8*time.Minute))
	if err != nil {
		t.Fatalf("Close(snapshot handoff) error = %v", err)
	}
	return closed, parentRef, childRef
}
