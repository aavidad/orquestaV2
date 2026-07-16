package goal_test

import (
	"reflect"
	"testing"
	"time"

	domain "orquesta/internal/goal"
)

func TestControlsEffectivePauseRequiresBothScopesResumed(t *testing.T) {
	fixture := newPlanFixture(t)
	phase := mustPhase(t, "phase:v14-pause")
	aRef := mustRef(t, "work-item:v14-pause-a", domain.NewWorkItemRef)
	bRef := mustRef(t, "work-item:v14-pause-b", domain.NewWorkItemRef)
	running := applyAndStartPlan(t, fixture.goal, domain.PlanInput{
		Generation: 1, Phases: []domain.PhaseInstance{phase},
		WorkItems: []domain.WorkItem{
			fixture.item(t, aRef, phase.Key(), nil, nil),
			fixture.item(t, bRef, phase.Key(), nil, nil),
		},
	})
	running = startGoalItem(t, running, aRef, "execution:v14-pause-a", baseTime().Add(4*time.Minute))

	paused, err := running.SetPaused(running.Revision(), true, baseTime().Add(5*time.Minute))
	if err != nil || len(paused.ReadyWorkItems()) != 0 {
		t.Fatalf("SetPaused(goal): ready=%d err=%v", len(paused.ReadyWorkItems()), err)
	}
	b, _ := paused.WorkItem(bRef)
	paused, err = paused.SetWorkItemPaused(
		paused.Revision(), b.Revision(), bRef, true, baseTime().Add(6*time.Minute),
	)
	if err != nil {
		t.Fatal(err)
	}
	paused = succeedGoalItem(t, paused, aRef, "v14-pause-a", baseTime().Add(7*time.Minute))
	paused, err = paused.SetPaused(paused.Revision(), false, baseTime().Add(8*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	requireReadyRefs(t, paused)
	if effective, found := paused.EffectivePause(bRef); !found || !effective {
		t.Fatalf("EffectivePause(B)=%v/%v", effective, found)
	}
	b, _ = paused.WorkItem(bRef)
	resumed, err := paused.SetWorkItemPaused(
		paused.Revision(), b.Revision(), bRef, false, baseTime().Add(9*time.Minute),
	)
	if err != nil {
		t.Fatal(err)
	}
	requireReadyRefs(t, resumed, bRef)
	if resumed.ControlSequence() != 4 {
		t.Fatalf("ControlSequence=%d, want 4", resumed.ControlSequence())
	}
	assertV14RoundTrip(t, resumed)

	tampered := cloneGoalSnapshot(resumed.Snapshot())
	tampered.ControlSequence++
	_, err = domain.RestoreGoal(tampered)
	requireCode(t, err, domain.ErrorSnapshotInvalid)
}

func TestControlsExecutionExhaustionInterruptsWithoutClosingGoal(t *testing.T) {
	aggregate, refs := newGoalWithItems(t, "retry after exhaustion")
	running, err := aggregate.Start(aggregate.Revision(), baseTime().Add(3*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	running = startGoalItem(t, running, refs[0], "execution:v14-failed", baseTime().Add(4*time.Minute))
	item, _ := running.WorkItem(refs[0])
	failedExecution, _ := item.Execution()
	interrupted, err := running.InterruptWorkItem(
		running.Revision(), item.Revision(), refs[0], failedExecution,
		domain.WorkItemInterruptExecutionFailed, baseTime().Add(5*time.Minute),
	)
	if err != nil {
		t.Fatal(err)
	}
	item, _ = interrupted.WorkItem(refs[0])
	cause, hasCause := item.InterruptCause()
	if item.State() != domain.WorkItemStateInterrupted || !hasCause ||
		cause != domain.WorkItemInterruptExecutionFailed || interrupted.IsTerminal() {
		t.Fatalf("interrupted item=%q cause=%q/%v goal=%q", item.State(), cause, hasCause, interrupted.State())
	}
	if outcome, closable := interrupted.ClosableOutcome(); closable || outcome != "" {
		t.Fatalf("interrupted Goal closable=%v outcome=%q", closable, outcome)
	}
	assertV14RoundTrip(t, interrupted)

	replacement := mustRef(t, "execution:v14-retry", domain.NewExecutionRef)
	retried, err := interrupted.RetryWorkItem(
		interrupted.Revision(), item.Revision(), refs[0], failedExecution, replacement,
		baseTime().Add(6*time.Minute),
	)
	if err != nil {
		t.Fatal(err)
	}
	item, _ = retried.WorkItem(refs[0])
	current, _ := item.Execution()
	if item.State() != domain.WorkItemStateRunning || current != replacement {
		t.Fatalf("retried item=%q execution=%q", item.State(), current)
	}
	_, err = interrupted.RetryWorkItem(
		interrupted.Revision(), item.Revision(), refs[0], failedExecution, failedExecution,
		baseTime().Add(7*time.Minute),
	)
	if domain.ErrorCodeOf(err) == "" {
		t.Fatal("retry accepted stale or reused execution")
	}
}

func TestControlsNestedReplanResolvesLogicalOutcome(t *testing.T) {
	fixture := newPlanFixture(t)
	phase := mustPhase(t, "phase:v14-replan")
	sourceRef := mustRef(t, "work-item:v14-source", domain.NewWorkItemRef)
	running := applyAndStartPlan(t, fixture.goal, domain.PlanInput{
		Generation: 1, Phases: []domain.PhaseInstance{phase},
		WorkItems: []domain.WorkItem{fixture.item(t, sourceRef, phase.Key(), nil, nil)},
	})
	aRef := mustRef(t, "work-item:v14-successor-a", domain.NewWorkItemRef)
	bRef := mustRef(t, "work-item:v14-successor-b", domain.NewWorkItemRef)
	source, _ := running.WorkItem(sourceRef)
	running = mustApplyV14Replan(t, running, domain.ReplanInput{
		ExpectedPlanGeneration: running.PlanGeneration(), Source: sourceRef,
		ExpectedSourceRevision: source.Revision(), Cause: domain.ReplanCauseSplitPending,
		Successors: []domain.WorkItem{
			fixture.item(t, aRef, phase.Key(), nil, nil), fixture.item(t, bRef, phase.Key(), nil, nil),
		}, At: baseTime().Add(4 * time.Minute),
	})
	if outcome, resolved := running.LogicalWorkItemOutcome(sourceRef); resolved || outcome != "" {
		t.Fatalf("source resolved before successors: %q/%v", outcome, resolved)
	}
	a, _ := running.WorkItem(aRef)
	a1Ref := mustRef(t, "work-item:v14-successor-a1", domain.NewWorkItemRef)
	a2Ref := mustRef(t, "work-item:v14-successor-a2", domain.NewWorkItemRef)
	running = mustApplyV14Replan(t, running, domain.ReplanInput{
		ExpectedPlanGeneration: running.PlanGeneration(), Source: aRef,
		ExpectedSourceRevision: a.Revision(), Cause: domain.ReplanCauseSplitPending,
		Successors: []domain.WorkItem{
			fixture.item(t, a1Ref, phase.Key(), nil, nil), fixture.item(t, a2Ref, phase.Key(), nil, nil),
		}, At: baseTime().Add(5 * time.Minute),
	})
	for index, ref := range []domain.WorkItemRef{bRef, a1Ref, a2Ref} {
		running = startGoalItem(t, running, ref, "execution:v14-nested-"+ref.String(), baseTime().Add(6*time.Minute))
		running = succeedGoalItem(t, running, ref, "v14-nested-"+string(rune('a'+index)), baseTime().Add(7*time.Minute))
	}
	if outcome, resolved := running.LogicalWorkItemOutcome(sourceRef); !resolved || outcome != domain.WorkItemLogicalSucceeded {
		t.Fatalf("nested logical outcome=%q/%v", outcome, resolved)
	}
	closed, err := running.Close(running.Revision(), domain.GoalOutcomeSucceeded, baseTime().Add(8*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	assertV14RoundTrip(t, closed)
	cycle := cloneGoalSnapshot(closed.Snapshot())
	for index := range cycle.WorkItems {
		if cycle.WorkItems[index].Ref == a1Ref.String() {
			cycle.WorkItems[index].ReworkOf = a1Ref.String()
		}
	}
	_, err = domain.RestoreGoal(cycle)
	requireCode(t, err, domain.ErrorInvalidPlan)
	for _, ref := range []domain.WorkItemRef{aRef, bRef, a1Ref, a2Ref} {
		item, _ := closed.WorkItem(ref)
		if ref == bRef {
			continue
		}
		if source, linked := item.ReworkOf(); !linked || (ref == aRef && source != sourceRef) ||
			((ref == a1Ref || ref == a2Ref) && source != aRef) {
			t.Fatalf("ReworkOf(%s)=%s/%v", ref, source, linked)
		}
	}
}

func TestControlsReplanSplitStoppedAndFailedSources(t *testing.T) {
	for _, test := range []struct {
		name      string
		interrupt domain.WorkItemInterruptCause
		replan    domain.ReplanCause
	}{
		{name: "stopped", interrupt: domain.WorkItemInterruptExecutionStopped, replan: domain.ReplanCauseExecutionStopped},
		{name: "failed", interrupt: domain.WorkItemInterruptExecutionFailed, replan: domain.ReplanCauseExecutionFailed},
	} {
		t.Run(test.name, func(t *testing.T) {
			aggregate, refs := newGoalWithItems(t, "replace interrupted")
			running, _ := aggregate.Start(aggregate.Revision(), baseTime().Add(3*time.Minute))
			running = startGoalItem(t, running, refs[0], "execution:v14-"+test.name, baseTime().Add(4*time.Minute))
			item, _ := running.WorkItem(refs[0])
			execution, _ := item.Execution()
			running, err := running.InterruptWorkItem(
				running.Revision(), item.Revision(), refs[0], execution, test.interrupt, baseTime().Add(5*time.Minute),
			)
			if err != nil {
				t.Fatal(err)
			}
			item, _ = running.WorkItem(refs[0])
			successor := v14WorkItem(t, running, item.Phase(), "work-item:v14-replan-"+test.name, nil)
			running = mustApplyV14Replan(t, running, domain.ReplanInput{
				ExpectedPlanGeneration: running.PlanGeneration(), Source: refs[0],
				ExpectedSourceRevision: item.Revision(), Cause: test.replan, CausalExecution: execution,
				Successors: []domain.WorkItem{successor}, At: baseTime().Add(6 * time.Minute),
			})
			if source, _ := running.WorkItem(refs[0]); source.State() != domain.WorkItemStateSuperseded {
				t.Fatalf("source state=%q", source.State())
			}
			if test.name == "failed" {
				pending, _ := running.WorkItem(successor.Ref())
				running, err = running.RequestWorkItemCancel(
					running.Revision(), pending.Revision(), successor.Ref(), baseTime().Add(7*time.Minute),
				)
				if outcome, resolved := running.LogicalWorkItemOutcome(refs[0]); err != nil || !resolved || outcome != domain.WorkItemLogicalFailed {
					t.Fatalf("failed successor outcome=%q/%v err=%v", outcome, resolved, err)
				}
			}
			assertV14RoundTrip(t, running)
		})
	}
}

func TestControlsReplanRejectsSkippedDescendantAndHandoffEndpoints(t *testing.T) {
	fixture := newPlanFixture(t)
	phase := mustPhase(t, "phase:v14-skipped")
	sourceRef := mustRef(t, "work-item:v14-skipped-source", domain.NewWorkItemRef)
	failureRef := mustRef(t, "work-item:v14-skipped-failure", domain.NewWorkItemRef)
	descendantRef := mustRef(t, "work-item:v14-skipped-descendant", domain.NewWorkItemRef)
	running := applyAndStartPlan(t, fixture.goal, domain.PlanInput{
		Generation: 1, Phases: []domain.PhaseInstance{phase}, WorkItems: []domain.WorkItem{
			fixture.item(t, sourceRef, phase.Key(), nil, nil),
			fixture.item(t, failureRef, phase.Key(), nil, nil),
			fixture.item(t, descendantRef, phase.Key(), []domain.WorkItemRef{sourceRef, failureRef}, nil),
		},
	})
	running = startGoalItem(t, running, sourceRef, "execution:v14-skipped-source", baseTime().Add(4*time.Minute))
	running = startGoalItem(t, running, failureRef, "execution:v14-skipped-failure", baseTime().Add(4*time.Minute))
	failure, _ := running.WorkItem(failureRef)
	running, _ = running.FailWorkItem(running.Revision(), failure.Revision(), failureRef, baseTime().Add(5*time.Minute))
	source, _ := running.WorkItem(sourceRef)
	execution, _ := source.Execution()
	running, _ = running.InterruptWorkItem(
		running.Revision(), source.Revision(), sourceRef, execution,
		domain.WorkItemInterruptExecutionStopped, baseTime().Add(6*time.Minute),
	)
	source, _ = running.WorkItem(sourceRef)
	_, err := running.ApplyReplan(running.Revision(), domain.ReplanInput{
		ExpectedPlanGeneration: running.PlanGeneration(), Source: sourceRef,
		ExpectedSourceRevision: source.Revision(), Cause: domain.ReplanCauseExecutionStopped,
		CausalExecution: execution,
		Successors: []domain.WorkItem{fixture.item(t,
			mustRef(t, "work-item:v14-skipped-rework", domain.NewWorkItemRef), phase.Key(), nil, nil,
		)}, At: baseTime().Add(7 * time.Minute),
	})
	requireCode(t, err, domain.ErrorInvalidPlan)

	for _, endpoint := range []string{"parent", "child"} {
		t.Run(endpoint, func(t *testing.T) {
			handoff, parent, children := newRunningHandoffGoal(t, 1)
			ref := parent
			if endpoint == "child" {
				ref = children[0]
			}
			item, _ := handoff.WorkItem(ref)
			execution, _ := item.Execution()
			handoff, err = handoff.InterruptWorkItem(
				handoff.Revision(), item.Revision(), ref, execution,
				domain.WorkItemInterruptExecutionStopped, baseTime().Add(5*time.Minute),
			)
			if err != nil {
				t.Fatal(err)
			}
			item, _ = handoff.WorkItem(ref)
			_, err = handoff.ApplyReplan(handoff.Revision(), domain.ReplanInput{
				ExpectedPlanGeneration: handoff.PlanGeneration(), Source: ref,
				ExpectedSourceRevision: item.Revision(), Cause: domain.ReplanCauseExecutionStopped,
				CausalExecution: execution,
				Successors:      []domain.WorkItem{v14WorkItem(t, handoff, item.Phase(), "work-item:v14-handoff-"+endpoint, nil)},
				At:              baseTime().Add(6 * time.Minute),
			})
			requireCode(t, err, domain.ErrorInvalidPlan)
		})
	}
}

func TestControlsCanceledHandoffChildFailsWithoutSyntheticResolution(t *testing.T) {
	running, parentRef, children := newRunningHandoffGoal(t, 1)
	childRef := children[0]
	child, _ := running.WorkItem(childRef)
	canceling, err := running.RequestWorkItemCancel(
		running.Revision(), child.Revision(), childRef, baseTime().Add(5*time.Minute),
	)
	if err != nil {
		t.Fatal(err)
	}
	child, _ = canceling.WorkItem(childRef)
	_, err = canceling.SucceedWorkItem(
		canceling.Revision(), child.Revision(), childRef,
		[]domain.ArtifactRef{mustRef(t, "artifact:v14-late", domain.NewArtifactRef)},
		[]domain.AttestationRef{mustRef(t, "attestation:v14-late", domain.NewAttestationRef)},
		baseTime().Add(6*time.Minute),
	)
	requireCode(t, err, domain.ErrorInvalidTransition)
	canceled, err := canceling.CompleteWorkItemCancel(
		canceling.Revision(), child.Revision(), childRef, baseTime().Add(6*time.Minute),
	)
	if err != nil {
		t.Fatal(err)
	}
	parent, _ := canceled.WorkItem(parentRef)
	canceled, err = canceled.SucceedWorkItem(
		canceled.Revision(), parent.Revision(), parentRef,
		[]domain.ArtifactRef{mustRef(t, "artifact:v14-parent", domain.NewArtifactRef)},
		[]domain.AttestationRef{mustRef(t, "attestation:v14-parent", domain.NewAttestationRef)},
		baseTime().Add(7*time.Minute),
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(canceled.ChildHandoffResolutions()) != 0 {
		t.Fatal("cancel fabricated child handoff resolution")
	}
	closed, err := canceled.Close(canceled.Revision(), domain.GoalOutcomeFailed, baseTime().Add(8*time.Minute))
	if err != nil || closed.State() != domain.GoalStateFailed {
		t.Fatalf("Close(failed): state=%q err=%v", closed.State(), err)
	}
	assertV14RoundTrip(t, closed)
}

func TestControlsWorkItemCancelCascadesDependencyCanceled(t *testing.T) {
	fixture := newPlanFixture(t)
	phase := mustPhase(t, "phase:v14-cancel-cascade")
	rootRef := mustRef(t, "work-item:v14-cancel-root", domain.NewWorkItemRef)
	childRef := mustRef(t, "work-item:v14-cancel-child", domain.NewWorkItemRef)
	leafRef := mustRef(t, "work-item:v14-cancel-leaf", domain.NewWorkItemRef)
	running := applyAndStartPlan(t, fixture.goal, domain.PlanInput{
		Generation: 1, Phases: []domain.PhaseInstance{phase}, WorkItems: []domain.WorkItem{
			fixture.item(t, rootRef, phase.Key(), nil, nil),
			fixture.item(t, childRef, phase.Key(), []domain.WorkItemRef{rootRef}, nil),
			fixture.item(t, leafRef, phase.Key(), []domain.WorkItemRef{childRef}, nil),
		},
	})
	root, _ := running.WorkItem(rootRef)
	canceled, err := running.RequestWorkItemCancel(
		running.Revision(), root.Revision(), rootRef, baseTime().Add(4*time.Minute),
	)
	if err != nil {
		t.Fatal(err)
	}
	for _, ref := range []domain.WorkItemRef{childRef, leafRef} {
		item, _ := canceled.WorkItem(ref)
		reason, ok := item.SkipReason()
		if item.State() != domain.WorkItemStateSkipped || !ok || reason != domain.WorkItemSkipReasonDependencyCanceled {
			t.Fatalf("%s state/reason=%q/%q/%v", ref, item.State(), reason, ok)
		}
	}
	closed, err := canceled.Close(canceled.Revision(), domain.GoalOutcomeFailed, baseTime().Add(5*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	assertV14RoundTrip(t, closed)
}

func TestControlsGoalCancelWinsItsCASBranch(t *testing.T) {
	aggregate, refs := newGoalWithItems(t, "cancel running")
	running, _ := aggregate.Start(aggregate.Revision(), baseTime().Add(3*time.Minute))
	running = startGoalItem(t, running, refs[0], "execution:v14-cancel", baseTime().Add(4*time.Minute))
	item, _ := running.WorkItem(refs[0])
	canceling, err := running.RequestCancel(running.Revision(), baseTime().Add(5*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	item, _ = canceling.WorkItem(refs[0])
	_, err = canceling.SucceedWorkItem(
		canceling.Revision(), item.Revision(), refs[0],
		[]domain.ArtifactRef{mustRef(t, "artifact:v14-cancel-race", domain.NewArtifactRef)},
		[]domain.AttestationRef{mustRef(t, "attestation:v14-cancel-race", domain.NewAttestationRef)},
		baseTime().Add(6*time.Minute),
	)
	requireCode(t, err, domain.ErrorInvalidTransition)
	canceling, err = canceling.CompleteWorkItemCancel(
		canceling.Revision(), item.Revision(), refs[0], baseTime().Add(6*time.Minute),
	)
	if err != nil {
		t.Fatal(err)
	}
	closed, err := canceling.CompleteCancel(canceling.Revision(), baseTime().Add(7*time.Minute))
	if err != nil || closed.State() != domain.GoalStateCanceled || !closed.IsTerminal() {
		t.Fatalf("CompleteCancel: state=%q err=%v", closed.State(), err)
	}
	assertV14RoundTrip(t, closed)
}

func TestControlsRestoreRejectsPausedClosedGoal(t *testing.T) {
	aggregate, refs := newGoalWithItems(t, "closed pause invariant")
	running, err := aggregate.Start(aggregate.Revision(), baseTime().Add(3*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	running, err = running.SetPaused(running.Revision(), true, baseTime().Add(4*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	running, err = running.SetPaused(running.Revision(), false, baseTime().Add(5*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	running = startGoalItem(t, running, refs[0], "execution:v14-closed-pause", baseTime().Add(6*time.Minute))
	running = succeedGoalItem(t, running, refs[0], "v14-closed-pause", baseTime().Add(7*time.Minute))
	closed, err := running.Close(running.Revision(), domain.GoalOutcomeSucceeded, baseTime().Add(8*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	snapshot := cloneGoalSnapshot(closed.Snapshot())
	snapshot.Paused = true
	_, err = domain.RestoreGoal(snapshot)
	requireCode(t, err, domain.ErrorSnapshotInvalid)
}

func mustApplyV14Replan(t *testing.T, goal domain.Goal, input domain.ReplanInput) domain.Goal {
	t.Helper()
	updated, err := goal.ApplyReplan(goal.Revision(), input)
	if err != nil {
		t.Fatalf("ApplyReplan(%s): %v", input.Source, err)
	}
	return updated
}

func v14WorkItem(
	t *testing.T,
	goal domain.Goal,
	phase domain.PhaseKey,
	ref string,
	dependencies []domain.WorkItemRef,
) domain.WorkItem {
	t.Helper()
	item, err := domain.NewWorkItem(domain.NewWorkItemInput{
		Ref: mustRef(t, ref, domain.NewWorkItemRef), Goal: goal.Ref(), Actor: goal.Actor(), Project: goal.Project(),
		Objective: "execute " + ref, CreatedAt: baseTime().Add(2 * time.Minute), Phase: phase,
		Dependencies: dependencies,
	})
	if err != nil {
		t.Fatal(err)
	}
	return item
}

func assertV14RoundTrip(t *testing.T, goal domain.Goal) {
	t.Helper()
	snapshot := goal.Snapshot()
	restored, err := domain.RestoreGoal(snapshot)
	if err != nil {
		t.Fatalf("RestoreGoal(V14): %v", err)
	}
	if got := restored.Snapshot(); !reflect.DeepEqual(got, snapshot) {
		t.Fatalf("V14 snapshot differs:\n got=%#v\nwant=%#v", got, snapshot)
	}
}
