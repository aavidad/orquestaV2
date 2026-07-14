package goal_test

import (
	"reflect"
	"testing"
	"time"

	domain "orquesta/internal/goal"
)

func TestV05PhaseAndExecutionRequirementsAreImmutableOpaqueMetadata(t *testing.T) {
	key, err := domain.NewPhaseKey("phase:research")
	if err != nil {
		t.Fatal(err)
	}
	phaseRef := mustRef(t, "phase-instance:research:1", domain.NewPhaseRef)
	templateRef := mustRef(t, "stage-template:STG-04", domain.NewPhaseTemplateRef)
	inputs := []domain.InputRef{
		mustRef(t, "input:app-spec", domain.NewInputRef),
		mustRef(t, "input:source-corpus", domain.NewInputRef),
	}
	criteria := []domain.CriterionRef{
		mustRef(t, "criterion:cited", domain.NewCriterionRef),
		mustRef(t, "criterion:reviewed", domain.NewCriterionRef),
	}
	phase, err := domain.NewPhaseInstanceWithMetadata(domain.PhaseInstanceInput{
		Ref: phaseRef, Key: key, TemplateRef: templateRef,
		InputRefs: inputs, CriterionRefs: criteria,
	})
	if err != nil {
		t.Fatalf("NewPhaseInstanceWithMetadata() error = %v", err)
	}
	if phase.Ref() != phaseRef || phase.TemplateRef() != templateRef || phase.Key() != key ||
		!reflect.DeepEqual(phase.InputRefs(), inputs) || !reflect.DeepEqual(phase.CriterionRefs(), criteria) {
		t.Fatalf("phase metadata drift: %#v", phase)
	}
	changedInputs := phase.InputRefs()
	changedInputs[0] = mustRef(t, "input:changed", domain.NewInputRef)
	if phase.InputRefs()[0] != inputs[0] {
		t.Fatal("phase input refs are mutable through accessor")
	}
	phaseType := reflect.TypeOf(phase)
	for _, forbidden := range []string{"State", "Start", "Succeed", "Fail", "Close"} {
		if _, exists := phaseType.MethodByName(forbidden); exists {
			t.Fatalf("PhaseInstance exposes lifecycle method %s", forbidden)
		}
	}
	_, err = domain.NewPhaseInstanceWithMetadata(domain.PhaseInstanceInput{
		Ref: phaseRef, Key: key, TemplateRef: templateRef,
		InputRefs: []domain.InputRef{inputs[0], inputs[0]},
	})
	requireCode(t, err, domain.ErrorInvalidPlan)

	fixture := newPlanFixture(t)
	ref := mustRef(t, "work-item:requirements", domain.NewWorkItemRef)
	role, _ := domain.NewRoleKey("role:researcher")
	skills := []domain.SkillRef{
		mustRef(t, "skill:web-research", domain.NewSkillRef),
		mustRef(t, "skill:citations", domain.NewSkillRef),
	}
	tools := []domain.ToolRef{mustRef(t, "tool:browser", domain.NewToolRef)}
	capabilities := []domain.CapabilityRef{
		mustRef(t, "capability:web", domain.NewCapabilityRef),
		mustRef(t, "capability:write", domain.NewCapabilityRef),
	}
	item, err := fixture.newItem(domain.NewWorkItemInput{
		Ref: ref, Phase: key, Role: role,
		SkillRefs: skills, ToolRefs: tools, CapabilityRefs: capabilities,
	})
	if err != nil {
		t.Fatalf("NewWorkItem(requirements) error = %v", err)
	}
	if item.Role() != role || !reflect.DeepEqual(item.SkillRefs(), skills) ||
		!reflect.DeepEqual(item.ToolRefs(), tools) || !reflect.DeepEqual(item.CapabilityRefs(), capabilities) {
		t.Fatalf("work requirements drift: role=%q skills=%v tools=%v capabilities=%v", item.Role(), item.SkillRefs(), item.ToolRefs(), item.CapabilityRefs())
	}
	changedSkills := item.SkillRefs()
	changedSkills[0] = mustRef(t, "skill:changed", domain.NewSkillRef)
	if item.SkillRefs()[0] != skills[0] {
		t.Fatal("skill refs are mutable through accessor")
	}
	_, err = fixture.newItem(domain.NewWorkItemInput{
		Ref: mustRef(t, "work-item:duplicate-skill", domain.NewWorkItemRef), Phase: key,
		SkillRefs: []domain.SkillRef{skills[0], skills[0]},
	})
	requireCode(t, err, domain.ErrorInvalidPlan)
}

func TestV05ContractualChildrenAreSeparateFromDependenciesAndKeepGoalOpen(t *testing.T) {
	fixture := newPlanFixture(t)
	phase := mustPhase(t, "phase:delegation")
	parentRef := mustRef(t, "work-item:parent", domain.NewWorkItemRef)
	childRef := mustRef(t, "work-item:child", domain.NewWorkItemRef)
	parent := fixture.item(t, parentRef, phase.Key(), nil, []domain.WriteScope{mustScope(t, "internal/parent")})
	child, err := fixture.newItem(domain.NewWorkItemInput{
		Ref: childRef, Phase: phase.Key(), Parent: parentRef,
		WriteSet: []domain.WriteScope{mustScope(t, "internal/child")},
	})
	if err != nil {
		t.Fatalf("NewWorkItem(child) error = %v", err)
	}
	if len(child.Dependencies()) != 0 {
		t.Fatal("contractual parent was silently converted into dependency")
	}
	running := applyAndStartPlan(t, fixture.goal, domain.PlanInput{
		Generation: 1, Phases: []domain.PhaseInstance{phase}, WorkItems: []domain.WorkItem{parent, child},
	})
	requireReadyRefs(t, running, parentRef, childRef)
	children := running.ChildWorkItems(parentRef)
	if len(children) != 1 || children[0].Ref() != childRef {
		t.Fatalf("ChildWorkItems(parent) = %v", children)
	}
	running = startGoalItem(t, running, parentRef, "execution:parent", baseTime().Add(4*time.Minute))
	running = startGoalItem(t, running, childRef, "execution:child", baseTime().Add(4*time.Minute))
	parentRunning, _ := running.WorkItem(parentRef)
	_, err = running.SucceedWorkItem(
		running.Revision(), parentRunning.Revision()+1, parentRef,
		[]domain.ArtifactRef{mustRef(t, "artifact:parent", domain.NewArtifactRef)},
		[]domain.AttestationRef{mustRef(t, "attestation:parent", domain.NewAttestationRef)},
		baseTime().Add(5*time.Minute),
	)
	requireCode(t, err, domain.ErrorRevisionConflict)
	running, err = running.SucceedWorkItem(
		running.Revision(), parentRunning.Revision(), parentRef,
		[]domain.ArtifactRef{mustRef(t, "artifact:parent", domain.NewArtifactRef)},
		[]domain.AttestationRef{mustRef(t, "attestation:parent", domain.NewAttestationRef)},
		baseTime().Add(5*time.Minute),
	)
	if err != nil {
		t.Fatalf("SucceedWorkItem(parent) error = %v", err)
	}
	_, err = running.Close(running.Revision(), domain.GoalOutcomeSucceeded, baseTime().Add(6*time.Minute))
	requireCode(t, err, domain.ErrorWorkItemsNotTerminal)
	running = succeedGoalItem(t, running, childRef, "child", baseTime().Add(5*time.Minute))
	if _, err := domain.RestoreGoal(running.Snapshot()); err != nil {
		t.Fatalf("RestoreGoal(parent/child) error = %v", err)
	}

	missingParent := mustRef(t, "work-item:missing-parent", domain.NewWorkItemRef)
	orphan, err := fixture.newItem(domain.NewWorkItemInput{
		Ref: mustRef(t, "work-item:orphan", domain.NewWorkItemRef), Phase: phase.Key(), Parent: missingParent,
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = domain.NewPlan(domain.PlanInput{
		Generation: 1, Phases: []domain.PhaseInstance{phase}, WorkItems: []domain.WorkItem{orphan},
	})
	requireCode(t, err, domain.ErrorInvalidPlan)

	aRef := mustRef(t, "work-item:parent-cycle-a", domain.NewWorkItemRef)
	bRef := mustRef(t, "work-item:parent-cycle-b", domain.NewWorkItemRef)
	a, _ := fixture.newItem(domain.NewWorkItemInput{Ref: aRef, Phase: phase.Key(), Parent: bRef})
	b, _ := fixture.newItem(domain.NewWorkItemInput{Ref: bRef, Phase: phase.Key(), Parent: aRef})
	_, err = domain.NewPlan(domain.PlanInput{
		Generation: 1, Phases: []domain.PhaseInstance{phase}, WorkItems: []domain.WorkItem{a, b},
	})
	requireCode(t, err, domain.ErrorInvalidPlan)
}

func TestV05RunningPlanEvolutionAppendsWithoutChangingHistory(t *testing.T) {
	fixture := newPlanFixture(t)
	phase := mustPhase(t, "phase:evolution")
	parentRef := mustRef(t, "work-item:evolution-parent", domain.NewWorkItemRef)
	parent := fixture.item(t, parentRef, phase.Key(), nil, nil)
	running := applyAndStartPlan(t, fixture.goal, domain.PlanInput{
		Generation: 1, Phases: []domain.PhaseInstance{phase}, WorkItems: []domain.WorkItem{parent},
	})
	running = startGoalItem(t, running, parentRef, "execution:evolution-parent", baseTime().Add(4*time.Minute))
	child, err := fixture.newItem(domain.NewWorkItemInput{
		Ref:   mustRef(t, "work-item:evolution-child", domain.NewWorkItemRef),
		Phase: phase.Key(), Parent: parentRef,
		CreatedAt: baseTime().Add(5 * time.Minute),
	})
	if err != nil {
		t.Fatal(err)
	}
	// fixture.newItem fixes CreatedAt for compatibility, so construct the real
	// post-start child explicitly.
	child, err = domain.NewWorkItem(domain.NewWorkItemInput{
		Ref: child.Ref(), Goal: running.Ref(), Actor: running.Actor(), Project: running.Project(),
		Objective: "delegated after start", CreatedAt: baseTime().Add(5 * time.Minute),
		Phase: phase.Key(), Parent: parentRef,
	})
	if err != nil {
		t.Fatal(err)
	}
	parentBefore, _ := running.WorkItem(parentRef)
	evolved, err := running.ApplyPlan(running.Revision(), mustPlan(t, domain.PlanInput{
		Generation: 2, Phases: running.Phases(), WorkItems: append(running.WorkItems(), child),
	}))
	if err != nil {
		t.Fatalf("ApplyPlan(running append) error = %v", err)
	}
	if evolved.PlanGeneration() != 2 || evolved.WorkItemCount() != 2 {
		t.Fatalf("evolved generation/items = %d/%d", evolved.PlanGeneration(), evolved.WorkItemCount())
	}
	if parentAfter, _ := evolved.WorkItem(parentRef); !reflect.DeepEqual(parentAfter, parentBefore) {
		t.Fatal("running plan evolution changed the existing parent")
	}
	if _, err := domain.RestoreGoal(evolved.Snapshot()); err != nil {
		t.Fatalf("RestoreGoal(evolved running) error = %v", err)
	}
	_, err = evolved.ApplyPlan(evolved.Revision(), mustPlan(t, domain.PlanInput{
		Generation: 3, Phases: evolved.Phases(), WorkItems: []domain.WorkItem{parentBefore},
	}))
	requireCode(t, err, domain.ErrorInvalidPlan)
	changedPhase, err := domain.NewPhaseInstanceWithMetadata(domain.PhaseInstanceInput{
		Ref:         mustRef(t, "phase-instance:evolution-rewritten", domain.NewPhaseRef),
		Key:         phase.Key(),
		TemplateRef: phase.TemplateRef(),
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = evolved.ApplyPlan(evolved.Revision(), mustPlan(t, domain.PlanInput{
		Generation: 3, Phases: []domain.PhaseInstance{changedPhase}, WorkItems: evolved.WorkItems(),
	}))
	requireCode(t, err, domain.ErrorInvalidPlan)

	changedParent, err := domain.NewWorkItem(domain.NewWorkItemInput{
		Ref: parentRef, Goal: running.Ref(), Actor: running.Actor(), Project: running.Project(),
		Objective: "rewritten parent", CreatedAt: parent.CreatedAt(), Phase: phase.Key(),
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = running.ApplyPlan(running.Revision(), mustPlan(t, domain.PlanInput{
		Generation: 2, Phases: running.Phases(), WorkItems: []domain.WorkItem{changedParent, child},
	}))
	requireCode(t, err, domain.ErrorInvalidPlan)

	terminal := succeededGoalSnapshotFixture(t)
	newItem, err := domain.NewWorkItem(domain.NewWorkItemInput{
		Ref:  mustRef(t, "work-item:after-terminal", domain.NewWorkItemRef),
		Goal: terminal.Ref(), Actor: terminal.Actor(), Project: terminal.Project(),
		Objective: "must not append", CreatedAt: baseTime().Add(9 * time.Minute),
		Phase: terminal.Phases()[0].Key(),
	})
	if err != nil {
		t.Fatal(err)
	}
	terminalPlan := mustPlan(t, domain.PlanInput{
		Generation: terminal.PlanGeneration() + 1, Phases: terminal.Phases(),
		WorkItems: append(terminal.WorkItems(), newItem),
	})
	_, err = terminal.ApplyPlan(terminal.Revision(), terminalPlan)
	requireCode(t, err, domain.ErrorInvalidTransition)
}

func TestV05RestoreRejectsActiveDependencyAndRunningWriteSetIncoherence(t *testing.T) {
	fixture := newPlanFixture(t)
	phase := mustPhase(t, "phase:restore")
	aRef := mustRef(t, "work-item:restore-a", domain.NewWorkItemRef)
	bRef := mustRef(t, "work-item:restore-b", domain.NewWorkItemRef)
	a := fixture.item(t, aRef, phase.Key(), nil, []domain.WriteScope{mustScope(t, "internal/a")})
	b := fixture.item(t, bRef, phase.Key(), []domain.WorkItemRef{aRef}, []domain.WriteScope{mustScope(t, "internal/b")})
	running := applyAndStartPlan(t, fixture.goal, domain.PlanInput{
		Generation: 1, Phases: []domain.PhaseInstance{phase}, WorkItems: []domain.WorkItem{a, b},
	})
	running = startGoalItem(t, running, aRef, "execution:restore-a", baseTime().Add(4*time.Minute))
	running = succeedGoalItem(t, running, aRef, "restore-a", baseTime().Add(5*time.Minute))
	running = startGoalItem(t, running, bRef, "execution:restore-b", baseTime().Add(6*time.Minute))
	snapshot := running.Snapshot()
	snapshot.WorkItems[0].State = domain.WorkItemStatePending
	snapshot.WorkItems[0].Revision = 1
	snapshot.WorkItems[0].StartedAt = time.Time{}
	snapshot.WorkItems[0].FinishedAt = time.Time{}
	snapshot.WorkItems[0].ExecutionRef = ""
	snapshot.WorkItems[0].ArtifactRefs = nil
	snapshot.WorkItems[0].AttestationRefs = nil
	snapshot.Revision -= 2
	_, err := domain.RestoreGoal(snapshot)
	requireCode(t, err, domain.ErrorInvalidPlan)

	cRef := mustRef(t, "work-item:restore-c", domain.NewWorkItemRef)
	dRef := mustRef(t, "work-item:restore-d", domain.NewWorkItemRef)
	c := fixture.item(t, cRef, phase.Key(), nil, []domain.WriteScope{mustScope(t, "internal/c")})
	d := fixture.item(t, dRef, phase.Key(), nil, []domain.WriteScope{mustScope(t, "internal/d")})
	twoRunning := applyAndStartPlan(t, fixture.goal, domain.PlanInput{
		Generation: 1, Phases: []domain.PhaseInstance{phase}, WorkItems: []domain.WorkItem{c, d},
	})
	twoRunning = startGoalItem(t, twoRunning, cRef, "execution:restore-c", baseTime().Add(4*time.Minute))
	twoRunning = startGoalItem(t, twoRunning, dRef, "execution:restore-d", baseTime().Add(4*time.Minute))
	conflict := twoRunning.Snapshot()
	conflict.WorkItems[1].WriteSet = []string{"internal/c/child"}
	_, err = domain.RestoreGoal(conflict)
	requireCode(t, err, domain.ErrorInvalidPlan)
}

func TestV05MetadataSnapshotRoundTripAndDuplicateTamperRejection(t *testing.T) {
	fixture := newPlanFixture(t)
	key, _ := domain.NewPhaseKey("phase:metadata")
	phase, err := domain.NewPhaseInstanceWithMetadata(domain.PhaseInstanceInput{
		Ref: mustRef(t, "phase-instance:metadata", domain.NewPhaseRef), Key: key,
		TemplateRef:   mustRef(t, "stage-template:STG-00", domain.NewPhaseTemplateRef),
		InputRefs:     []domain.InputRef{mustRef(t, "input:spec", domain.NewInputRef)},
		CriterionRefs: []domain.CriterionRef{mustRef(t, "criterion:complete", domain.NewCriterionRef)},
	})
	if err != nil {
		t.Fatal(err)
	}
	item, err := fixture.newItem(domain.NewWorkItemInput{
		Ref: mustRef(t, "work-item:metadata", domain.NewWorkItemRef), Phase: key,
		SkillRefs:      []domain.SkillRef{mustRef(t, "skill:one", domain.NewSkillRef)},
		ToolRefs:       []domain.ToolRef{mustRef(t, "tool:one", domain.NewToolRef)},
		CapabilityRefs: []domain.CapabilityRef{mustRef(t, "capability:one", domain.NewCapabilityRef)},
	})
	if err != nil {
		t.Fatal(err)
	}
	applied, err := fixture.goal.ApplyPlan(fixture.goal.Revision(), mustPlan(t, domain.PlanInput{
		Generation: 1, Phases: []domain.PhaseInstance{phase}, WorkItems: []domain.WorkItem{item},
	}))
	if err != nil {
		t.Fatal(err)
	}
	snapshot := applied.Snapshot()
	restored, err := domain.RestoreGoal(snapshot)
	if err != nil {
		t.Fatalf("RestoreGoal(metadata) error = %v", err)
	}
	if !reflect.DeepEqual(restored.Snapshot(), snapshot) {
		t.Fatal("metadata snapshot did not round-trip exactly")
	}

	duplicateInput := cloneGoalSnapshot(snapshot)
	duplicateInput.Phases[0].InputRefs = append(duplicateInput.Phases[0].InputRefs, duplicateInput.Phases[0].InputRefs[0])
	_, err = domain.RestoreGoal(duplicateInput)
	requireCode(t, err, domain.ErrorInvalidPlan)
	duplicateSkill := cloneGoalSnapshot(snapshot)
	duplicateSkill.WorkItems[0].SkillRefs = append(duplicateSkill.WorkItems[0].SkillRefs, duplicateSkill.WorkItems[0].SkillRefs[0])
	_, err = domain.RestoreGoal(duplicateSkill)
	requireCode(t, err, domain.ErrorInvalidPlan)
}
