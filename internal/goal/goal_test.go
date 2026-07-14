package goal_test

import (
	"testing"
	"time"

	domain "orquesta/internal/goal"
)

func TestGoalTransitionsAndClosesAfterPlannedWorkSucceeds(t *testing.T) {
	aggregate, refs := newGoalWithItems(t, "first unit", "second unit")
	if aggregate.State() != domain.GoalStatePending || aggregate.Revision() != 2 {
		t.Fatalf("planned goal state/revision = %q/%d", aggregate.State(), aggregate.Revision())
	}

	_, err := aggregate.Start(1, baseTime().Add(3*time.Minute))
	requireCode(t, err, domain.ErrorRevisionConflict)
	running, err := aggregate.Start(aggregate.Revision(), baseTime().Add(3*time.Minute))
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	if running.Revision() != 3 || aggregate.State() != domain.GoalStatePending {
		t.Fatal("Start did not preserve immutable revisions")
	}

	running = startGoalItem(t, running, refs[0], "execution:first", baseTime().Add(4*time.Minute))
	running = succeedGoalItem(t, running, refs[0], "first", baseTime().Add(5*time.Minute))
	_, err = running.Close(running.Revision(), domain.GoalOutcomeSucceeded, baseTime().Add(6*time.Minute))
	requireCode(t, err, domain.ErrorWorkItemsNotTerminal)
	running = startGoalItem(t, running, refs[1], "execution:second", baseTime().Add(6*time.Minute))
	running = succeedGoalItem(t, running, refs[1], "second", baseTime().Add(7*time.Minute))
	if running.Revision() != 7 {
		t.Fatalf("revision after child transitions = %d, want 7", running.Revision())
	}
	if outcome, ok := running.ClosableOutcome(); !ok || outcome != domain.GoalOutcomeSucceeded {
		t.Fatalf("ClosableOutcome() = %q/%v", outcome, ok)
	}
	closed, err := running.Close(running.Revision(), domain.GoalOutcomeSucceeded, baseTime().Add(8*time.Minute))
	if err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	if closed.State() != domain.GoalStateSucceeded || closed.Revision() != 8 || !closed.IsTerminal() {
		t.Fatalf("closed state/revision = %q/%d", closed.State(), closed.Revision())
	}
}

func TestFailureCascadesDependencySkippedAndFailedGoalCanClose(t *testing.T) {
	fixture := newPlanFixture(t)
	phase := mustPhase(t, "phase:work")
	aRef := mustRef(t, "work-item:fail-root", domain.NewWorkItemRef)
	bRef := mustRef(t, "work-item:fail-child", domain.NewWorkItemRef)
	cRef := mustRef(t, "work-item:fail-grandchild", domain.NewWorkItemRef)
	dRef := mustRef(t, "work-item:independent", domain.NewWorkItemRef)
	a := fixture.item(t, aRef, phase.Key(), nil, nil)
	b := fixture.item(t, bRef, phase.Key(), []domain.WorkItemRef{aRef}, nil)
	c := fixture.item(t, cRef, phase.Key(), []domain.WorkItemRef{bRef}, nil)
	d := fixture.item(t, dRef, phase.Key(), nil, nil)
	running := applyAndStartPlan(t, fixture.goal, domain.PlanInput{
		Generation: 1, Phases: []domain.PhaseInstance{phase},
		WorkItems: []domain.WorkItem{a, b, c, d},
	})
	running = startGoalItem(t, running, aRef, "execution:failure-root", baseTime().Add(4*time.Minute))
	aRunning, _ := running.WorkItem(aRef)
	failed, err := running.FailWorkItem(running.Revision(), aRunning.Revision(), aRef, baseTime().Add(5*time.Minute))
	if err != nil {
		t.Fatalf("FailWorkItem() error = %v", err)
	}
	for _, ref := range []domain.WorkItemRef{bRef, cRef} {
		item, _ := failed.WorkItem(ref)
		reason, ok := item.SkipReason()
		if item.State() != domain.WorkItemStateSkipped || !ok || reason != domain.WorkItemSkipReasonDependencyFailed {
			t.Fatalf("item %q state/reason = %q/%q", ref, item.State(), reason)
		}
	}
	_, err = failed.Close(failed.Revision(), domain.GoalOutcomeFailed, baseTime().Add(6*time.Minute))
	requireCode(t, err, domain.ErrorWorkItemsNotTerminal)
	failed = startGoalItem(t, failed, dRef, "execution:independent", baseTime().Add(6*time.Minute))
	failed = succeedGoalItem(t, failed, dRef, "independent", baseTime().Add(7*time.Minute))
	closed, err := failed.Close(failed.Revision(), domain.GoalOutcomeFailed, baseTime().Add(8*time.Minute))
	if err != nil {
		t.Fatalf("Close(failed) error = %v", err)
	}
	if closed.State() != domain.GoalStateFailed {
		t.Fatalf("closed state = %q", closed.State())
	}
	restored, err := domain.RestoreGoal(closed.Snapshot())
	if err != nil {
		t.Fatalf("RestoreGoal(failed cascade) error = %v", err)
	}
	restoredChild, _ := restored.WorkItem(cRef)
	if reason, ok := restoredChild.SkipReason(); !ok || reason != domain.WorkItemSkipReasonDependencyFailed {
		t.Fatalf("restored skip reason = %q/%v", reason, ok)
	}
}

func TestGoalRejectsInvalidScopeTimesAndTransitions(t *testing.T) {
	intent := newIntent(t)
	_, err := domain.NewGoal(domain.GoalRef{}, intent, baseTime().Add(time.Minute))
	requireCode(t, err, domain.ErrorInvalidRef)
	goalRef := mustRef(t, "goal:validation", domain.NewGoalRef)
	aggregate, err := domain.NewGoal(goalRef, intent, baseTime().Add(time.Minute))
	if err != nil {
		t.Fatalf("NewGoal() error = %v", err)
	}
	_, err = aggregate.Start(aggregate.Revision(), baseTime().Add(2*time.Minute))
	requireCode(t, err, domain.ErrorWorkItemsRequired)

	wrong := validGoalItemInput(t, goalRef, mustRef(t, "actor:other", domain.NewActorRef), intent.Project())
	wrongItem, err := domain.NewWorkItem(wrong)
	if err != nil {
		t.Fatalf("NewWorkItem() error = %v", err)
	}
	_, err = applyGoalItems(aggregate, []domain.WorkItem{wrongItem})
	requireCode(t, err, domain.ErrorScopeConflict)

	validItem, err := domain.NewWorkItem(validGoalItemInput(t, goalRef, intent.Actor(), intent.Project()))
	if err != nil {
		t.Fatalf("NewWorkItem(valid) error = %v", err)
	}
	planned, err := applyGoalItems(aggregate, []domain.WorkItem{validItem})
	if err != nil {
		t.Fatalf("ApplyPlan(valid) error = %v", err)
	}
	_, err = planned.Start(planned.Revision(), baseTime())
	requireCode(t, err, domain.ErrorInvalidArgument)
	running, err := planned.Start(planned.Revision(), baseTime().Add(3*time.Minute))
	if err != nil {
		t.Fatalf("Start(valid) error = %v", err)
	}
	_, err = running.StartWorkItem(
		running.Revision(), 1,
		mustRef(t, "work-item:missing", domain.NewWorkItemRef),
		mustRef(t, "execution:missing", domain.NewExecutionRef),
		baseTime().Add(4*time.Minute),
	)
	requireCode(t, err, domain.ErrorWorkItemNotFound)
}

func newIntent(t *testing.T) domain.IntentManifest {
	t.Helper()
	manifest, err := domain.NewIntentManifest(domain.IntentManifestInput{
		Ref:       mustRef(t, "intent:goal-tests", domain.NewIntentRef),
		Actor:     mustRef(t, "actor:local", domain.NewActorRef),
		Project:   mustRef(t, "project:orquesta", domain.NewProjectRef),
		Statement: "execute the durable goal", SubmittedAt: baseTime(),
	})
	if err != nil {
		t.Fatalf("NewIntentManifest() error = %v", err)
	}
	return manifest
}

func newGoalWithItems(t *testing.T, objectives ...string) (domain.Goal, []domain.WorkItemRef) {
	t.Helper()
	intent := newIntent(t)
	goalRef := mustRef(t, "goal:001", domain.NewGoalRef)
	aggregate, err := domain.NewGoal(goalRef, intent, baseTime().Add(time.Minute))
	if err != nil {
		t.Fatalf("NewGoal() error = %v", err)
	}
	items := make([]domain.WorkItem, 0, len(objectives))
	refs := make([]domain.WorkItemRef, 0, len(objectives))
	for index, objective := range objectives {
		ref := mustRef(t, "work-item:"+string(rune('a'+index)), domain.NewWorkItemRef)
		item, itemErr := domain.NewWorkItem(domain.NewWorkItemInput{
			Ref: ref, Goal: goalRef, Actor: intent.Actor(), Project: intent.Project(),
			Objective: objective, CreatedAt: baseTime().Add(2 * time.Minute),
		})
		if itemErr != nil {
			t.Fatalf("NewWorkItem(%d) error = %v", index, itemErr)
		}
		items = append(items, item)
		refs = append(refs, ref)
	}
	aggregate, err = applyGoalItems(aggregate, items)
	if err != nil {
		t.Fatalf("ApplyPlan() error = %v", err)
	}
	return aggregate, refs
}

func applyGoalItems(aggregate domain.Goal, items []domain.WorkItem) (domain.Goal, error) {
	phase, err := domain.NewPhaseInstance(domain.DefaultPhaseKey())
	if err != nil {
		return domain.Goal{}, err
	}
	plan, err := domain.NewPlan(domain.PlanInput{
		Generation: aggregate.PlanGeneration() + 1,
		Phases:     []domain.PhaseInstance{phase}, WorkItems: items,
	})
	if err != nil {
		return domain.Goal{}, err
	}
	return aggregate.ApplyPlan(aggregate.Revision(), plan)
}

func validGoalItemInput(t *testing.T, goalRef domain.GoalRef, actor domain.ActorRef, project domain.ProjectRef) domain.NewWorkItemInput {
	t.Helper()
	return domain.NewWorkItemInput{
		Ref:  mustRef(t, "work-item:validation", domain.NewWorkItemRef),
		Goal: goalRef, Actor: actor, Project: project,
		Objective: "validate aggregate rules", CreatedAt: baseTime().Add(2 * time.Minute),
	}
}

func startGoalItem(t *testing.T, aggregate domain.Goal, ref domain.WorkItemRef, executionValue string, at time.Time) domain.Goal {
	t.Helper()
	item, ok := aggregate.WorkItem(ref)
	if !ok {
		t.Fatalf("WorkItem(%q) not found", ref)
	}
	updated, err := aggregate.StartWorkItem(
		aggregate.Revision(), item.Revision(), ref,
		mustRef(t, executionValue, domain.NewExecutionRef), at,
	)
	if err != nil {
		t.Fatalf("StartWorkItem(%q) error = %v", ref, err)
	}
	return updated
}

func succeedGoalItem(t *testing.T, aggregate domain.Goal, ref domain.WorkItemRef, suffix string, at time.Time) domain.Goal {
	t.Helper()
	item, ok := aggregate.WorkItem(ref)
	if !ok {
		t.Fatalf("WorkItem(%q) not found", ref)
	}
	updated, err := aggregate.SucceedWorkItem(
		aggregate.Revision(), item.Revision(), ref,
		[]domain.ArtifactRef{mustRef(t, "artifact:"+suffix, domain.NewArtifactRef)},
		[]domain.AttestationRef{mustRef(t, "attestation:"+suffix, domain.NewAttestationRef)}, at,
	)
	if err != nil {
		t.Fatalf("SucceedWorkItem(%q) error = %v", ref, err)
	}
	return updated
}
