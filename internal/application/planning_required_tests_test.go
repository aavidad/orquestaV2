package application

import (
	"crypto/sha256"
	"testing"
	"time"

	"orquesta/internal/goal"
	"orquesta/internal/governance"
)

func TestCompileWorkItemSpecPreservesRequiredTestsAndRejectsUntestedWriter(t *testing.T) {
	ref, _ := goal.NewWorkItemRef("work-item:planning-required-tests")
	goalRef, _ := goal.NewGoalRef("goal:planning-required-tests")
	actorRef, _ := goal.NewActorRef("actor:planning-required-tests")
	projectRef, _ := goal.NewProjectRef("project:planning-required-tests")
	scope := workItemCompileScope{
		goalRef: goalRef, actorRef: actorRef, projectRef: projectRef,
		createdAt:     time.Date(2026, 7, 22, 2, 0, 0, 0, time.UTC),
		defaultDemand: governance.ResourceVector{ProcessSlots: 1},
		goalLimit:     governance.ResourceVector{ProcessSlots: 2},
	}
	spec := WorkItemSpec{
		Key: "writer", Objective: "write exact change", Phase: goal.DefaultPhaseKey().String(),
		Role: goal.DefaultRoleKey().String(), WriteSet: []string{"internal/planning"},
		CouncilPolicy: "skip_by_operator", RequiredTests: requiredTestSpecs("required-test:planning"),
		OutputContract: goal.OutputContractEvidenceBundle,
	}
	item, err := compileWorkItemSpec(spec, ref, scope, workItemRefResolver{})
	appTestNoError(t, err)
	tests := item.RequiredTests()
	if len(tests) != 1 || tests[0].Ref().String() != "required-test:planning" ||
		tests[0].ToolRef().String() != "tool:test" || tests[0].WorkingDirectory() != "." {
		t.Fatalf("compiled tests=%+v", tests)
	}

	spec.RequiredTests = nil
	if _, err := compileWorkItemSpec(spec, ref, scope, workItemRefResolver{}); goal.ErrorCodeOf(err) != goal.ErrorInvalidPlan {
		t.Fatalf("untested writer error=%v", err)
	}
}

func TestPlanFingerprintBindsEveryRequiredTestField(t *testing.T) {
	base := &PlanSpec{WorkItems: []WorkItemSpec{{
		Key: "writer", WriteSet: []string{"internal/planning"},
		CouncilPolicy: "skip_by_operator", RequiredTests: requiredTestSpecs("required-test:fingerprint"),
	}}}
	fingerprint := func(spec *PlanSpec) string {
		digest := sha256.New()
		writePlanFingerprint(digest, spec)
		return fingerprintHex(digest)
	}
	want := fingerprint(base)
	mutations := []func(*RequiredTestSpec){
		func(spec *RequiredTestSpec) { spec.Ref += ":changed" },
		func(spec *RequiredTestSpec) { spec.ToolRef += ":changed" },
		func(spec *RequiredTestSpec) { spec.Arguments[0] = "./internal/..." },
		func(spec *RequiredTestSpec) { spec.WorkingDirectory = "internal" },
	}
	seen := map[string]struct{}{want: {}}
	for index, mutate := range mutations {
		changed := clonePlanSpec(base)
		mutate(&changed.WorkItems[0].RequiredTests[0])
		got := fingerprint(changed)
		if got == want {
			t.Errorf("mutation %d did not change fingerprint", index)
		}
		if _, duplicate := seen[got]; duplicate {
			t.Errorf("mutation %d collided", index)
		}
		seen[got] = struct{}{}
	}

	legacy := clonePlanSpec(base)
	legacy.WorkItems[0].RequiredTests = nil
	if fingerprint(legacy) == want {
		t.Fatal("required-tests V3 fingerprint equals legacy fingerprint")
	}
}
