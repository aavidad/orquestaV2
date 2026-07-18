package goal_test

import (
	"reflect"
	"testing"
	"time"

	domain "orquesta/internal/goal"
)

func TestPlanRejectsInvalidDAGKeysDuplicatesAndWriteSets(t *testing.T) {
	fixture := newPlanFixture(t)
	build := mustPhase(t, "phase:build")
	review := mustPhase(t, "phase:review")
	aRef := mustRef(t, "work-item:plan-a", domain.NewWorkItemRef)
	bRef := mustRef(t, "work-item:plan-b", domain.NewWorkItemRef)
	missing := mustRef(t, "work-item:missing", domain.NewWorkItemRef)

	cases := []struct {
		name   string
		phases []domain.PhaseInstance
		items  func() []domain.WorkItem
		code   domain.ErrorCode
	}{
		{
			name: "missing dependency", phases: []domain.PhaseInstance{build},
			items: func() []domain.WorkItem {
				return []domain.WorkItem{fixture.item(t, aRef, build.Key(), []domain.WorkItemRef{missing}, nil)}
			}, code: domain.ErrorInvalidPlan,
		},
		{
			name: "cycle", phases: []domain.PhaseInstance{build},
			items: func() []domain.WorkItem {
				return []domain.WorkItem{
					fixture.item(t, aRef, build.Key(), []domain.WorkItemRef{bRef}, nil),
					fixture.item(t, bRef, build.Key(), []domain.WorkItemRef{aRef}, nil),
				}
			}, code: domain.ErrorInvalidPlan,
		},
		{
			name: "unknown phase", phases: []domain.PhaseInstance{build},
			items: func() []domain.WorkItem {
				return []domain.WorkItem{fixture.item(t, aRef, review.Key(), nil, nil)}
			}, code: domain.ErrorInvalidPlan,
		},
		{
			name: "duplicate phase", phases: []domain.PhaseInstance{build, build},
			items: func() []domain.WorkItem {
				return []domain.WorkItem{fixture.item(t, aRef, build.Key(), nil, nil)}
			}, code: domain.ErrorInvalidPlan,
		},
		{
			name: "duplicate item", phases: []domain.PhaseInstance{build},
			items: func() []domain.WorkItem {
				item := fixture.item(t, aRef, build.Key(), nil, nil)
				return []domain.WorkItem{item, item}
			}, code: domain.ErrorDuplicateWorkItem,
		},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			_, err := domain.NewPlan(domain.PlanInput{Generation: 1, Phases: test.phases, WorkItems: test.items()})
			requireCode(t, err, test.code)
		})
	}

	_, err := fixture.newItem(domain.NewWorkItemInput{
		Ref: bRef, Phase: build.Key(), Dependencies: []domain.WorkItemRef{aRef, aRef},
	})
	requireCode(t, err, domain.ErrorInvalidPlan)
	scope := mustScope(t, "internal/goal")
	_, err = fixture.newItem(domain.NewWorkItemInput{
		Ref: aRef, Phase: build.Key(), WriteSet: []domain.WriteScope{scope, scope},
	})
	requireCode(t, err, domain.ErrorInvalidPlan)

	for _, invalid := range []string{"", "/absolute", "internal//goal", "internal/../goal", "../escape", "internal/**", " trailing "} {
		t.Run("invalid scope "+invalid, func(t *testing.T) {
			_, scopeErr := domain.NewWriteScope(invalid)
			requireCode(t, scopeErr, domain.ErrorInvalidPlan)
		})
	}
	for _, valid := range []string{"internal/goal/plan.go", ".github/workflows/build-test.yml", "docs/diseño limpio-1.md"} {
		if _, scopeErr := domain.NewWriteScope(valid); scopeErr != nil {
			t.Fatalf("NewWriteScope(%q) error = %v", valid, scopeErr)
		}
	}
}

func TestApplyPlanUsesGoalCASNextGenerationAndMonotonicEvolution(t *testing.T) {
	fixture := newPlanFixture(t)
	phase := mustPhase(t, "phase:build")
	first := fixture.item(t, mustRef(t, "work-item:first-plan", domain.NewWorkItemRef), phase.Key(), nil, nil)
	plan := mustPlan(t, domain.PlanInput{Generation: 1, Phases: []domain.PhaseInstance{phase}, WorkItems: []domain.WorkItem{first}})

	_, err := fixture.goal.ApplyPlan(99, plan)
	requireCode(t, err, domain.ErrorRevisionConflict)
	wrongGeneration := mustPlan(t, domain.PlanInput{Generation: 2, Phases: []domain.PhaseInstance{phase}, WorkItems: []domain.WorkItem{first}})
	_, err = fixture.goal.ApplyPlan(fixture.goal.Revision(), wrongGeneration)
	requireCode(t, err, domain.ErrorInvalidPlan)

	applied, err := fixture.goal.ApplyPlan(fixture.goal.Revision(), plan)
	if err != nil {
		t.Fatalf("ApplyPlan() error = %v", err)
	}
	if applied.State() != domain.GoalStatePending || applied.PlanGeneration() != 1 || applied.Revision() != 2 {
		t.Fatalf("applied state/generation/revision = %q/%d/%d", applied.State(), applied.PlanGeneration(), applied.Revision())
	}
	second := fixture.item(t, mustRef(t, "work-item:second-plan", domain.NewWorkItemRef), phase.Key(), nil, nil)
	_, err = applied.ApplyPlan(applied.Revision(), mustPlan(t, domain.PlanInput{
		Generation: 2, Phases: []domain.PhaseInstance{phase}, WorkItems: []domain.WorkItem{second},
	}))
	requireCode(t, err, domain.ErrorInvalidPlan)
	replanned, err := applied.ApplyPlan(applied.Revision(), mustPlan(t, domain.PlanInput{
		Generation: 2, Phases: []domain.PhaseInstance{phase}, WorkItems: []domain.WorkItem{first, second},
	}))
	if err != nil {
		t.Fatalf("ApplyPlan(second) error = %v", err)
	}
	if replanned.PlanGeneration() != 2 || replanned.Revision() != 3 || replanned.WorkItemCount() != 2 {
		t.Fatalf("replanned generation/revision/items = %d/%d/%d", replanned.PlanGeneration(), replanned.Revision(), replanned.WorkItemCount())
	}
	running, err := replanned.Start(replanned.Revision(), baseTime().Add(3*time.Minute))
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	third := fixture.item(t, mustRef(t, "work-item:third-plan", domain.NewWorkItemRef), phase.Key(), nil, nil)
	running, err = running.ApplyPlan(running.Revision(), mustPlan(t, domain.PlanInput{
		Generation: 3, Phases: []domain.PhaseInstance{phase}, WorkItems: append(running.WorkItems(), third),
	}))
	if err != nil || running.PlanGeneration() != 3 || running.WorkItemCount() != 3 {
		t.Fatalf("ApplyPlan(running append) = %d/%d/%v", running.PlanGeneration(), running.WorkItemCount(), err)
	}
}

func TestReadyReturnsDeterministicMaximalConflictFreeCohort(t *testing.T) {
	fixture := newPlanFixture(t)
	phase := mustPhase(t, "phase:work")
	aRef := mustRef(t, "work-item:writer-a", domain.NewWorkItemRef)
	bRef := mustRef(t, "work-item:writer-b", domain.NewWorkItemRef)
	cRef := mustRef(t, "work-item:writer-c", domain.NewWorkItemRef)
	dRef := mustRef(t, "work-item:dependent", domain.NewWorkItemRef)
	a := fixture.item(t, aRef, phase.Key(), nil, []domain.WriteScope{mustScope(t, "internal/goal")})
	b := fixture.item(t, bRef, phase.Key(), nil, []domain.WriteScope{mustScope(t, "internal/goal/plan.go")})
	c := fixture.item(t, cRef, phase.Key(), nil, []domain.WriteScope{mustScope(t, "internal/application")})
	d := fixture.item(t, dRef, phase.Key(), []domain.WorkItemRef{bRef}, []domain.WriteScope{mustScope(t, "docs/result.md")})
	running := applyAndStartPlan(t, fixture.goal, domain.PlanInput{
		Generation: 1, Phases: []domain.PhaseInstance{phase}, WorkItems: []domain.WorkItem{a, b, c, d},
	})
	requireRunnableRefs(t, running, aRef, bRef, cRef)
	requireReadyRefs(t, running, aRef, cRef)

	dItem, _ := running.WorkItem(dRef)
	_, err := running.StartWorkItem(
		running.Revision(), dItem.Revision(), dRef,
		mustRef(t, "execution:too-early", domain.NewExecutionRef), baseTime().Add(4*time.Minute),
	)
	requireCode(t, err, domain.ErrorWorkItemNotReady)

	bItem, _ := running.WorkItem(bRef)
	_, err = running.StartWorkItem(
		running.Revision(), bItem.Revision(), bRef,
		mustRef(t, "execution:conflict", domain.NewExecutionRef), baseTime().Add(4*time.Minute),
	)
	requireCode(t, err, domain.ErrorWorkItemNotReady)
	running = startGoalItem(t, running, aRef, "execution:writer-a", baseTime().Add(4*time.Minute))
	requireRunnableRefs(t, running, cRef)
	requireReadyRefs(t, running, cRef)
	running = succeedGoalItem(t, running, aRef, "writer-a", baseTime().Add(5*time.Minute))
	requireReadyRefs(t, running, bRef, cRef)
}

func TestPlannedSnapshotRoundTripAndMutationChecks(t *testing.T) {
	fixture := newPlanFixture(t)
	build := mustPhase(t, "phase:build")
	review := mustPhase(t, "phase:review")
	aRef := mustRef(t, "work-item:snapshot-build", domain.NewWorkItemRef)
	bRef := mustRef(t, "work-item:snapshot-review", domain.NewWorkItemRef)
	a := fixture.item(t, aRef, build.Key(), nil, []domain.WriteScope{mustScope(t, "internal/goal")})
	b := fixture.itemWithContract(t, bRef, review.Key(), []domain.WorkItemRef{aRef}, nil, mustContract(t, domain.OutputContractArtifact))
	running := applyAndStartPlan(t, fixture.goal, domain.PlanInput{
		Generation: 1, Phases: []domain.PhaseInstance{build, review}, WorkItems: []domain.WorkItem{a, b},
	})
	running = startGoalItem(t, running, aRef, "execution:snapshot-plan", baseTime().Add(4*time.Minute))
	running = succeedGoalItem(t, running, aRef, "snapshot-plan", baseTime().Add(5*time.Minute))
	snapshot := running.Snapshot()
	restored, err := domain.RestoreGoal(snapshot)
	if err != nil {
		t.Fatalf("RestoreGoal() error = %v", err)
	}
	if got := restored.Snapshot(); !reflect.DeepEqual(got, snapshot) {
		t.Fatalf("snapshot round trip differs:\n got: %#v\nwant: %#v", got, snapshot)
	}

	mutations := []struct {
		name   string
		mutate func(*domain.GoalSnapshot)
	}{
		{name: "old schema", mutate: func(value *domain.GoalSnapshot) { value.SchemaVersion = 0 }},
		{name: "generation", mutate: func(value *domain.GoalSnapshot) { value.PlanGeneration++ }},
		{name: "phase", mutate: func(value *domain.GoalSnapshot) { value.WorkItems[0].PhaseKey = "phase:missing" }},
		{name: "dependency", mutate: func(value *domain.GoalSnapshot) { value.WorkItems[1].DependencyRefs[0] = "work-item:missing" }},
		{name: "cycle", mutate: func(value *domain.GoalSnapshot) { value.WorkItems[0].DependencyRefs = []string{bRef.String()} }},
		{name: "duplicate phase", mutate: func(value *domain.GoalSnapshot) { value.Phases = append(value.Phases, value.Phases[0]) }},
		{name: "write scope", mutate: func(value *domain.GoalSnapshot) { value.WorkItems[0].WriteSet[0] = "../escape" }},
		{name: "revision", mutate: func(value *domain.GoalSnapshot) { value.Revision++ }},
	}
	for _, mutation := range mutations {
		t.Run(mutation.name, func(t *testing.T) {
			changed := cloneGoalSnapshot(snapshot)
			mutation.mutate(&changed)
			if _, restoreErr := domain.RestoreGoal(changed); restoreErr == nil {
				t.Fatal("RestoreGoal(mutated) error = nil")
			}
		})
	}
}

func TestOutputContractDefinesMinimumEvidence(t *testing.T) {
	fixture := newPlanFixture(t)
	phase := mustPhase(t, "phase:evidence")
	ref := mustRef(t, "work-item:artifact-only", domain.NewWorkItemRef)
	item := fixture.itemWithContract(t, ref, phase.Key(), nil, nil, mustContract(t, domain.OutputContractArtifact))
	running := applyAndStartPlan(t, fixture.goal, domain.PlanInput{
		Generation: 1, Phases: []domain.PhaseInstance{phase}, WorkItems: []domain.WorkItem{item},
	})
	running = startGoalItem(t, running, ref, "execution:artifact-only", baseTime().Add(4*time.Minute))
	started, _ := running.WorkItem(ref)
	updated, err := running.SucceedWorkItem(
		running.Revision(), started.Revision(), ref,
		[]domain.ArtifactRef{mustRef(t, "artifact:only", domain.NewArtifactRef)}, nil,
		baseTime().Add(5*time.Minute),
	)
	if err != nil {
		t.Fatalf("SucceedWorkItem() error = %v", err)
	}
	succeeded, _ := updated.WorkItem(ref)
	if succeeded.State() != domain.WorkItemStateSucceeded || len(succeeded.Attestations()) != 0 {
		t.Fatalf("artifact result state/attestations = %q/%d", succeeded.State(), len(succeeded.Attestations()))
	}
}

func TestNewWorkItemDefaultsStayExplicit(t *testing.T) {
	item := newWorkItem(t, "compatibility defaults")
	if item.Phase() != domain.DefaultPhaseKey() || item.Role() != domain.DefaultRoleKey() ||
		item.OutputContract().Kind() != domain.OutputContractEvidenceBundle {
		t.Fatalf("defaults phase/role/contract = %q/%q/%q", item.Phase(), item.Role(), item.OutputContract().Kind())
	}
}

type planFixture struct {
	goal    domain.Goal
	actor   domain.ActorRef
	project domain.ProjectRef
}

func newPlanFixture(t *testing.T) planFixture {
	t.Helper()
	intent := newIntent(t)
	aggregate, err := domain.NewGoal(
		mustRef(t, "goal:plan-tests", domain.NewGoalRef), newInitialAppSpec(t, intent), baseTime().Add(time.Minute),
	)
	if err != nil {
		t.Fatalf("NewGoal() error = %v", err)
	}
	return planFixture{goal: aggregate, actor: intent.Actor(), project: intent.Project()}
}

func (fixture planFixture) item(t *testing.T, ref domain.WorkItemRef, phase domain.PhaseKey, deps []domain.WorkItemRef, writes []domain.WriteScope) domain.WorkItem {
	t.Helper()
	return fixture.itemWithContract(t, ref, phase, deps, writes, domain.EvidenceBundleOutputContract())
}

func (fixture planFixture) itemWithContract(t *testing.T, ref domain.WorkItemRef, phase domain.PhaseKey, deps []domain.WorkItemRef, writes []domain.WriteScope, contract domain.OutputContract) domain.WorkItem {
	t.Helper()
	item, err := fixture.newItem(domain.NewWorkItemInput{
		Ref: ref, Phase: phase, Dependencies: deps, WriteSet: writes, OutputContract: contract,
	})
	if err != nil {
		t.Fatalf("NewWorkItem(%q) error = %v", ref, err)
	}
	return item
}

func (fixture planFixture) newItem(overrides domain.NewWorkItemInput) (domain.WorkItem, error) {
	return domain.NewWorkItem(domain.NewWorkItemInput{
		Ref: overrides.Ref, Goal: fixture.goal.Ref(), Actor: fixture.actor, Project: fixture.project,
		Objective: "execute " + overrides.Ref.String(), CreatedAt: baseTime().Add(2 * time.Minute),
		Phase: overrides.Phase, Role: overrides.Role, Parent: overrides.Parent,
		HandoffRequired: overrides.HandoffRequired,
		Dependencies:    overrides.Dependencies, WriteSet: overrides.WriteSet,
		SkillRefs: overrides.SkillRefs, ToolRefs: overrides.ToolRefs, CapabilityRefs: overrides.CapabilityRefs,
		OutputContract: overrides.OutputContract, BudgetDemand: overrides.BudgetDemand,
		SecurityCriticality: overrides.SecurityCriticality, ReasoningEffort: overrides.ReasoningEffort,
	})
}

func mustPhase(t *testing.T, value string) domain.PhaseInstance {
	t.Helper()
	key, err := domain.NewPhaseKey(value)
	if err != nil {
		t.Fatalf("NewPhaseKey(%q) error = %v", value, err)
	}
	phase, err := domain.NewPhaseInstance(key)
	if err != nil {
		t.Fatalf("NewPhaseInstance(%q) error = %v", value, err)
	}
	return phase
}

func mustScope(t *testing.T, value string) domain.WriteScope {
	t.Helper()
	scope, err := domain.NewWriteScope(value)
	if err != nil {
		t.Fatalf("NewWriteScope(%q) error = %v", value, err)
	}
	return scope
}

func mustContract(t *testing.T, kind domain.OutputContractKind) domain.OutputContract {
	t.Helper()
	contract, err := domain.NewOutputContract(kind)
	if err != nil {
		t.Fatalf("NewOutputContract(%q) error = %v", kind, err)
	}
	return contract
}

func mustPlan(t *testing.T, input domain.PlanInput) domain.Plan {
	t.Helper()
	plan, err := domain.NewPlan(input)
	if err != nil {
		t.Fatalf("NewPlan() error = %v", err)
	}
	return plan
}

func applyAndStartPlan(t *testing.T, aggregate domain.Goal, input domain.PlanInput) domain.Goal {
	t.Helper()
	applied, err := aggregate.ApplyPlan(aggregate.Revision(), mustPlan(t, input))
	if err != nil {
		t.Fatalf("ApplyPlan() error = %v", err)
	}
	running, err := applied.Start(applied.Revision(), baseTime().Add(3*time.Minute))
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	return running
}

func requireReadyRefs(t *testing.T, aggregate domain.Goal, want ...domain.WorkItemRef) {
	t.Helper()
	ready := aggregate.ReadyWorkItems()
	if len(ready) != len(want) {
		t.Fatalf("ReadyWorkItems count = %d, want %d", len(ready), len(want))
	}
	for index, item := range ready {
		if item.Ref() != want[index] {
			t.Fatalf("ReadyWorkItems[%d] = %q, want %q", index, item.Ref(), want[index])
		}
	}
}

func requireRunnableRefs(t *testing.T, aggregate domain.Goal, want ...domain.WorkItemRef) {
	t.Helper()
	runnable := aggregate.RunnableWorkItems()
	if len(runnable) != len(want) {
		t.Fatalf("RunnableWorkItems count = %d, want %d", len(runnable), len(want))
	}
	for index, item := range runnable {
		if item.Ref() != want[index] {
			t.Fatalf("RunnableWorkItems[%d] = %q, want %q", index, item.Ref(), want[index])
		}
	}
}
