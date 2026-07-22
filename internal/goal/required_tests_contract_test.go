package goal_test

import (
	"reflect"
	"testing"
	"time"

	domain "orquesta/internal/goal"
)

func TestRequiredTestSpecIsImmutableAndDigestBindsEveryField(t *testing.T) {
	base := requiredTestSpec(t, "required-test:base", "tool:go-test", []string{"./..."}, ".")
	want := base.Digest()
	arguments := base.Arguments()
	arguments[0] = "./changed/..."
	if base.Arguments()[0] != "./..." || base.Digest() != want {
		t.Fatal("Arguments exposed mutable domain state")
	}

	mutations := []domain.RequiredTestSpec{
		requiredTestSpec(t, "required-test:other", "tool:go-test", []string{"./..."}, "."),
		requiredTestSpec(t, "required-test:base", "tool:other", []string{"./..."}, "."),
		requiredTestSpec(t, "required-test:base", "tool:go-test", []string{"./internal/..."}, "."),
		requiredTestSpec(t, "required-test:base", "tool:go-test", []string{"./..."}, "internal"),
	}
	for index, changed := range mutations {
		if changed.Digest() == want {
			t.Errorf("field mutation %d did not change digest", index)
		}
	}
}

func TestRequiredTestSpecRejectsInvalidArgvAndWorkingDirectory(t *testing.T) {
	ref, _ := domain.NewRequiredTestRef("required-test:invalid")
	toolRef, _ := domain.NewToolRef("tool:test")
	cases := []domain.RequiredTestSpecInput{
		{Ref: ref, ToolRef: toolRef, Arguments: []string{"bad\x00argument"}, WorkingDirectory: "."},
		{Ref: ref, ToolRef: toolRef, WorkingDirectory: ""},
		{Ref: ref, ToolRef: toolRef, WorkingDirectory: "/tmp"},
		{Ref: ref, ToolRef: toolRef, WorkingDirectory: "../outside"},
		{Ref: ref, ToolRef: toolRef, WorkingDirectory: "internal/../outside"},
		{Ref: ref, ToolRef: toolRef, WorkingDirectory: "internal\\tests"},
	}
	for index, input := range cases {
		if _, err := domain.NewRequiredTestSpec(input); err == nil {
			t.Errorf("invalid case %d accepted", index)
		}
	}
}

func TestWriterRequiresUniqueRequiredTestsWhileReadOnlyRemainsCompatible(t *testing.T) {
	fixture := newPlanFixture(t)
	phase := mustPhase(t, "phase:required-tests")
	writerRef := mustRef(t, "work-item:writer-required-tests", domain.NewWorkItemRef)
	_, err := domain.NewWorkItem(domain.NewWorkItemInput{
		Ref: writerRef, Goal: fixture.goal.Ref(), Actor: fixture.actor, Project: fixture.project,
		Objective: "write code", CreatedAt: baseTime().Add(2 * time.Minute), Phase: phase.Key(),
		WriteSet: []domain.WriteScope{mustScope(t, "internal/code")},
	})
	if domain.ErrorCodeOf(err) != domain.ErrorInvalidPlan {
		t.Fatalf("writer without tests error=%v", err)
	}

	readOnlyRef := mustRef(t, "work-item:read-only-required-tests", domain.NewWorkItemRef)
	if _, err := fixture.newItem(domain.NewWorkItemInput{Ref: readOnlyRef, Phase: phase.Key()}); err != nil {
		t.Fatalf("read-only item rejected: %v", err)
	}

	duplicate := requiredTestSpec(t, "required-test:duplicate", "tool:test", []string{"./..."}, ".")
	_, err = fixture.newItem(domain.NewWorkItemInput{
		Ref: writerRef, Phase: phase.Key(), WriteSet: []domain.WriteScope{mustScope(t, "internal/code")},
		RequiredTests: []domain.RequiredTestSpec{duplicate, duplicate},
	})
	if domain.ErrorCodeOf(err) != domain.ErrorInvalidPlan {
		t.Fatalf("duplicate tests error=%v", err)
	}
}

func TestRequiredTestsSurvivePlanSnapshotRestoreAndReplan(t *testing.T) {
	fixture := newPlanFixture(t)
	phase := mustPhase(t, "phase:required-tests-roundtrip")
	sourceRef := mustRef(t, "work-item:required-tests-source", domain.NewWorkItemRef)
	source := fixture.item(t, sourceRef, phase.Key(), nil, []domain.WriteScope{mustScope(t, "internal/source")})
	plan := mustPlan(t, domain.PlanInput{Generation: 1, Phases: []domain.PhaseInstance{phase}, WorkItems: []domain.WorkItem{source}})
	planned, err := fixture.goal.ApplyPlan(fixture.goal.Revision(), plan)
	if err != nil {
		t.Fatal(err)
	}
	restored, err := domain.RestoreGoal(planned.Snapshot())
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(restored.Snapshot().WorkItems[0].RequiredTests, planned.Snapshot().WorkItems[0].RequiredTests) {
		t.Fatal("snapshot/restore changed required tests")
	}
	running, err := restored.Start(restored.Revision(), baseTime().Add(3*time.Minute))
	if err != nil {
		t.Fatal(err)
	}

	successorRef := mustRef(t, "work-item:required-tests-successor", domain.NewWorkItemRef)
	successor, err := fixture.newItem(domain.NewWorkItemInput{
		Ref: successorRef, Phase: phase.Key(), WriteSet: []domain.WriteScope{mustScope(t, "internal/successor")},
		RequiredTests: []domain.RequiredTestSpec{requiredTestSpec(t, "required-test:successor", "tool:test", []string{"./..."}, ".")},
	})
	if err != nil {
		t.Fatal(err)
	}
	replanned, err := running.ApplyReplan(running.Revision(), domain.ReplanInput{
		ExpectedPlanGeneration: running.PlanGeneration(), Source: sourceRef,
		ExpectedSourceRevision: source.Revision(), Cause: domain.ReplanCauseSplitPending,
		Successors: []domain.WorkItem{successor}, At: baseTime().Add(4 * time.Minute),
	})
	if err != nil {
		t.Fatal(err)
	}
	items := replanned.WorkItems()
	if len(items[0].RequiredTests()) != 1 || len(items[1].RequiredTests()) != 1 ||
		items[1].RequiredTests()[0].Ref().String() != "required-test:successor" {
		t.Fatalf("replan lost required tests: %+v", replanned.Snapshot().WorkItems)
	}
}

func TestLegacyWriterWithoutTestsRestoresFailClosedAndSurvivesRestart(t *testing.T) {
	fixture := newPlanFixture(t)
	phase := mustPhase(t, "phase:required-tests-legacy")
	ref := mustRef(t, "work-item:required-tests-legacy", domain.NewWorkItemRef)
	writer := fixture.item(t, ref, phase.Key(), nil, []domain.WriteScope{mustScope(t, "internal/legacy")})
	plan := mustPlan(t, domain.PlanInput{Generation: 1, Phases: []domain.PhaseInstance{phase}, WorkItems: []domain.WorkItem{writer}})
	planned, err := fixture.goal.ApplyPlan(fixture.goal.Revision(), plan)
	if err != nil {
		t.Fatal(err)
	}
	legacy := cloneGoalSnapshot(planned.Snapshot())
	legacy.SchemaVersion = domain.GoalSnapshotSchemaVersion - 1
	legacy.WorkItems[0].RequiredTests = nil
	restored, err := domain.RestoreGoal(legacy)
	if err != nil {
		t.Fatalf("legacy writer became unreadable: %v", err)
	}
	if len(restored.WorkItems()[0].RequiredTests()) != 0 {
		t.Fatal("restore invented a legacy required test")
	}
	reemitted := restored.Snapshot()
	if reemitted.SchemaVersion != domain.GoalSnapshotSchemaVersion || len(reemitted.WorkItems[0].RequiredTests) != 0 {
		t.Fatalf("unsafe upgrade snapshot=%+v", reemitted.WorkItems[0].RequiredTests)
	}
	restarted, err := domain.RestoreGoal(reemitted)
	if err != nil || len(restarted.WorkItems()[0].RequiredTests()) != 0 {
		t.Fatalf("restart lost fail-closed legacy candidate: tests=%d err=%v", len(restarted.WorkItems()[0].RequiredTests()), err)
	}
}

func requiredTestSpec(t *testing.T, refValue, toolValue string, arguments []string, cwd string) domain.RequiredTestSpec {
	t.Helper()
	ref, err := domain.NewRequiredTestRef(refValue)
	if err != nil {
		t.Fatal(err)
	}
	toolRef, err := domain.NewToolRef(toolValue)
	if err != nil {
		t.Fatal(err)
	}
	spec, err := domain.NewRequiredTestSpec(domain.RequiredTestSpecInput{
		Ref: ref, ToolRef: toolRef, Arguments: arguments, WorkingDirectory: cwd,
	})
	if err != nil {
		t.Fatal(err)
	}
	return spec
}
