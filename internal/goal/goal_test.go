package goal_test

import (
	"testing"
	"time"

	domain "orquesta/internal/goal"
)

func TestGoalTransitionsAndClosesOnlyAfterAllWorkItemsAreTerminal(t *testing.T) {
	aggregate, refs := newGoalWithItems(t, "first unit", "second unit")
	if aggregate.State() != domain.GoalStatePending || aggregate.Revision() != 3 {
		t.Fatalf("prepared goal state/revision = %q/%d, want pending/3", aggregate.State(), aggregate.Revision())
	}
	if aggregate.Actor().String() == "" || aggregate.Project().String() == "" {
		t.Fatal("goal lost explicit actor/project scope")
	}

	_, err := aggregate.Start(1, baseTime().Add(3*time.Minute))
	requireCode(t, err, domain.ErrorRevisionConflict)
	running, err := aggregate.Start(aggregate.Revision(), baseTime().Add(3*time.Minute))
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	if running.State() != domain.GoalStateRunning || running.Revision() != 4 {
		t.Fatalf("running state/revision = %q/%d, want running/4", running.State(), running.Revision())
	}
	if aggregate.State() != domain.GoalStatePending {
		t.Fatal("Start mutated the source goal snapshot")
	}

	_, err = running.Close(running.Revision(), domain.GoalOutcomeSucceeded, baseTime().Add(4*time.Minute))
	requireCode(t, err, domain.ErrorWorkItemsNotTerminal)

	running = startGoalItem(t, running, refs[0], "execution:first", baseTime().Add(4*time.Minute))
	running = succeedGoalItem(t, running, refs[0], "first", baseTime().Add(5*time.Minute))
	_, err = running.Close(running.Revision(), domain.GoalOutcomeSucceeded, baseTime().Add(6*time.Minute))
	requireCode(t, err, domain.ErrorWorkItemsNotTerminal)

	running = startGoalItem(t, running, refs[1], "execution:second", baseTime().Add(6*time.Minute))
	running = succeedGoalItem(t, running, refs[1], "second", baseTime().Add(7*time.Minute))
	if running.Revision() != 8 {
		t.Fatalf("goal revision after child transitions = %d, want 8", running.Revision())
	}
	_, err = running.Close(running.Revision(), domain.GoalOutcomeFailed, baseTime().Add(8*time.Minute))
	requireCode(t, err, domain.ErrorOutcomeConflict)

	closed, err := running.Close(running.Revision(), domain.GoalOutcomeSucceeded, baseTime().Add(8*time.Minute))
	if err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	if closed.State() != domain.GoalStateSucceeded || closed.Revision() != 9 || !closed.IsTerminal() {
		t.Fatalf("closed state/revision/terminal = %q/%d/%v", closed.State(), closed.Revision(), closed.IsTerminal())
	}
	if _, ok := closed.ClosedAt(); !ok {
		t.Fatal("closed goal has no close timestamp")
	}
	_, err = closed.Close(closed.Revision(), domain.GoalOutcomeSucceeded, baseTime().Add(9*time.Minute))
	requireCode(t, err, domain.ErrorInvalidTransition)
}

func TestGoalFailedOutcomeRequiresAnExplicitFailedWorkItem(t *testing.T) {
	aggregate, refs := newGoalWithItems(t, "contains failed but remains pending", "deliver evidence")
	if aggregate.State() != domain.GoalStatePending {
		t.Fatalf("work-item words changed goal state to %q", aggregate.State())
	}
	running, err := aggregate.Start(aggregate.Revision(), baseTime().Add(3*time.Minute))
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	running = startGoalItem(t, running, refs[0], "execution:failed", baseTime().Add(4*time.Minute))
	item, ok := running.WorkItem(refs[0])
	if !ok {
		t.Fatal("first work item missing")
	}
	running, err = running.FailWorkItem(running.Revision(), item.Revision(), refs[0], baseTime().Add(5*time.Minute))
	if err != nil {
		t.Fatalf("FailWorkItem() error = %v", err)
	}

	running = startGoalItem(t, running, refs[1], "execution:success", baseTime().Add(6*time.Minute))
	running = succeedGoalItem(t, running, refs[1], "success", baseTime().Add(7*time.Minute))
	_, err = running.Close(running.Revision(), domain.GoalOutcomeSucceeded, baseTime().Add(8*time.Minute))
	requireCode(t, err, domain.ErrorOutcomeConflict)

	closed, err := running.Close(running.Revision(), domain.GoalOutcomeFailed, baseTime().Add(8*time.Minute))
	if err != nil {
		t.Fatalf("Close(failed) error = %v", err)
	}
	if closed.State() != domain.GoalStateFailed {
		t.Fatalf("closed state = %q, want failed", closed.State())
	}
}

func TestGoalRejectsInvalidScopeDuplicateItemsAndInvalidTransitions(t *testing.T) {
	intent := newIntent(t)
	_, err := domain.NewGoal(domain.GoalRef{}, intent, baseTime().Add(time.Minute))
	requireCode(t, err, domain.ErrorInvalidRef)
	_, err = domain.NewGoal(
		mustRef(t, "goal:early", domain.NewGoalRef),
		intent,
		baseTime().Add(-time.Minute),
	)
	requireCode(t, err, domain.ErrorInvalidArgument)

	goalRef := mustRef(t, "goal:validation", domain.NewGoalRef)
	aggregate, err := domain.NewGoal(goalRef, intent, baseTime().Add(time.Minute))
	if err != nil {
		t.Fatalf("NewGoal() error = %v", err)
	}
	_, err = aggregate.Start(aggregate.Revision(), baseTime().Add(2*time.Minute))
	requireCode(t, err, domain.ErrorWorkItemsRequired)

	wrongScopeTests := []struct {
		name  string
		input domain.NewWorkItemInput
	}{
		{
			name:  "goal",
			input: validGoalItemInput(t, mustRef(t, "goal:other", domain.NewGoalRef), intent.Actor(), intent.Project()),
		},
		{
			name:  "actor",
			input: validGoalItemInput(t, goalRef, mustRef(t, "actor:other", domain.NewActorRef), intent.Project()),
		},
		{
			name:  "project",
			input: validGoalItemInput(t, goalRef, intent.Actor(), mustRef(t, "project:other", domain.NewProjectRef)),
		},
	}
	for _, test := range wrongScopeTests {
		t.Run("scope "+test.name, func(t *testing.T) {
			item, itemErr := domain.NewWorkItem(test.input)
			if itemErr != nil {
				t.Fatalf("NewWorkItem() error = %v", itemErr)
			}
			_, addErr := aggregate.AddWorkItem(aggregate.Revision(), item)
			requireCode(t, addErr, domain.ErrorScopeConflict)
		})
	}

	earlyInput := validGoalItemInput(t, goalRef, intent.Actor(), intent.Project())
	earlyInput.Ref = mustRef(t, "work-item:early", domain.NewWorkItemRef)
	earlyInput.CreatedAt = baseTime()
	early, err := domain.NewWorkItem(earlyInput)
	if err != nil {
		t.Fatalf("NewWorkItem(early) error = %v", err)
	}
	_, err = aggregate.AddWorkItem(aggregate.Revision(), early)
	requireCode(t, err, domain.ErrorInvalidArgument)

	validInput := validGoalItemInput(t, goalRef, intent.Actor(), intent.Project())
	item, err := domain.NewWorkItem(validInput)
	if err != nil {
		t.Fatalf("NewWorkItem(valid) error = %v", err)
	}
	withItem, err := aggregate.AddWorkItem(aggregate.Revision(), item)
	if err != nil {
		t.Fatalf("AddWorkItem() error = %v", err)
	}
	_, err = withItem.AddWorkItem(aggregate.Revision(), item)
	requireCode(t, err, domain.ErrorRevisionConflict)
	_, err = withItem.AddWorkItem(withItem.Revision(), item)
	requireCode(t, err, domain.ErrorDuplicateWorkItem)
	_, err = withItem.Start(withItem.Revision(), baseTime())
	requireCode(t, err, domain.ErrorInvalidArgument)

	running, err := withItem.Start(withItem.Revision(), baseTime().Add(3*time.Minute))
	if err != nil {
		t.Fatalf("Start(valid) error = %v", err)
	}
	_, err = running.AddWorkItem(running.Revision(), item)
	requireCode(t, err, domain.ErrorInvalidTransition)
	_, err = running.StartWorkItem(
		running.Revision(),
		1,
		mustRef(t, "work-item:missing", domain.NewWorkItemRef),
		mustRef(t, "execution:missing", domain.NewExecutionRef),
		baseTime().Add(4*time.Minute),
	)
	requireCode(t, err, domain.ErrorWorkItemNotFound)
	_, err = running.StartWorkItem(
		running.Revision(),
		1,
		domain.WorkItemRef{},
		mustRef(t, "execution:invalid-ref", domain.NewExecutionRef),
		baseTime().Add(4*time.Minute),
	)
	requireCode(t, err, domain.ErrorInvalidRef)
}

func newIntent(t *testing.T) domain.IntentManifest {
	t.Helper()
	manifest, err := domain.NewIntentManifest(domain.IntentManifestInput{
		Ref:         mustRef(t, "intent:goal-tests", domain.NewIntentRef),
		Actor:       mustRef(t, "actor:local", domain.NewActorRef),
		Project:     mustRef(t, "project:orquesta", domain.NewProjectRef),
		Statement:   "execute the durable goal",
		SubmittedAt: baseTime(),
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
	refs := make([]domain.WorkItemRef, 0, len(objectives))
	for index, objective := range objectives {
		ref := mustRef(t, "work-item:"+string(rune('a'+index)), domain.NewWorkItemRef)
		item, itemErr := domain.NewWorkItem(domain.NewWorkItemInput{
			Ref:       ref,
			Goal:      goalRef,
			Actor:     intent.Actor(),
			Project:   intent.Project(),
			Objective: objective,
			CreatedAt: baseTime().Add(2 * time.Minute),
		})
		if itemErr != nil {
			t.Fatalf("NewWorkItem(%d) error = %v", index, itemErr)
		}
		aggregate, err = aggregate.AddWorkItem(aggregate.Revision(), item)
		if err != nil {
			t.Fatalf("AddWorkItem(%d) error = %v", index, err)
		}
		refs = append(refs, ref)
	}
	return aggregate, refs
}

func validGoalItemInput(
	t *testing.T,
	goalRef domain.GoalRef,
	actor domain.ActorRef,
	project domain.ProjectRef,
) domain.NewWorkItemInput {
	t.Helper()
	return domain.NewWorkItemInput{
		Ref:       mustRef(t, "work-item:validation", domain.NewWorkItemRef),
		Goal:      goalRef,
		Actor:     actor,
		Project:   project,
		Objective: "validate aggregate rules",
		CreatedAt: baseTime().Add(2 * time.Minute),
	}
}

func startGoalItem(
	t *testing.T,
	aggregate domain.Goal,
	ref domain.WorkItemRef,
	executionValue string,
	at time.Time,
) domain.Goal {
	t.Helper()
	item, ok := aggregate.WorkItem(ref)
	if !ok {
		t.Fatalf("WorkItem(%q) not found", ref)
	}
	updated, err := aggregate.StartWorkItem(
		aggregate.Revision(),
		item.Revision(),
		ref,
		mustRef(t, executionValue, domain.NewExecutionRef),
		at,
	)
	if err != nil {
		t.Fatalf("StartWorkItem(%q) error = %v", ref, err)
	}
	return updated
}

func succeedGoalItem(
	t *testing.T,
	aggregate domain.Goal,
	ref domain.WorkItemRef,
	evidenceSuffix string,
	at time.Time,
) domain.Goal {
	t.Helper()
	item, ok := aggregate.WorkItem(ref)
	if !ok {
		t.Fatalf("WorkItem(%q) not found", ref)
	}
	updated, err := aggregate.SucceedWorkItem(
		aggregate.Revision(),
		item.Revision(),
		ref,
		[]domain.ArtifactRef{mustRef(t, "artifact:"+evidenceSuffix, domain.NewArtifactRef)},
		[]domain.AttestationRef{mustRef(t, "attestation:"+evidenceSuffix, domain.NewAttestationRef)},
		at,
	)
	if err != nil {
		t.Fatalf("SucceedWorkItem(%q) error = %v", ref, err)
	}
	return updated
}
