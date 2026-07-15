package application

import (
	"context"
	"reflect"
	"testing"
	"time"

	"orquesta/internal/goal"
	"orquesta/internal/ports"
)

func TestSubmissionFingerprintFramesPlanCollections(t *testing.T) {
	actor, project := testScope(t)
	access := accessForScope(t, actor, project)
	base := SubmitRequest{
		RequestRef: "request:fingerprint", Statement: "same", Confirm: true,
		Plan: &PlanSpec{Phases: []PhaseSpec{{Ref: "phase-instance:test", Key: "phase:test", TemplateRef: "phase-template:test"}}, WorkItems: []WorkItemSpec{{
			Key: "work:a", Objective: "same", Phase: "phase:test", Role: "role:test",
			OutputContract: goal.OutputContractEvidenceBundle,
		}}},
	}
	left := base
	left.Plan = clonePlanSpec(base.Plan)
	left.Plan.WorkItems[0].Dependencies = []string{"d"}
	left.Plan.WorkItems[0].WriteSet = []string{"w"}
	right := base
	right.Plan = clonePlanSpec(base.Plan)
	right.Plan.WorkItems[0].WriteSet = []string{"d", "w"}
	if submissionFingerprint(access, left) == submissionFingerprint(access, right) {
		t.Fatal("dependency/write-set boundary collision")
	}
}

func TestExplicitPlanPreservesContractsAndLaunchesMaximalSafeCohort(t *testing.T) {
	clock := &mutableClock{now: time.Date(2026, 7, 14, 21, 0, 0, 0, time.UTC)}
	repository := newMemoryRepository()
	agent := &scriptedAgent{now: clock.Now}
	orchestrator, _ := newTestOrchestrator(t, repository, clock, agent)
	actor, project := testScope(t)
	access := accessForScope(t, actor, project)
	result, err := orchestrator.Submit(context.Background(), access, SubmitRequest{
		RequestRef: "request:v05-plan",
		Statement:  "run typed plan", Confirm: true,
		Plan: &PlanSpec{
			Phases: []PhaseSpec{{
				Ref: "phase-instance:build", Key: "phase:build", TemplateRef: "phase-template:program",
				InputRefs: []string{"input:app-spec"}, CriterionRefs: []string{"criterion:tests-green"},
			}},
			WorkItems: []WorkItemSpec{
				{Key: "a", Objective: "first writer", Phase: "phase:build", Role: "role:worker",
					WriteSet: []string{"internal/shared"}, SkillRefs: []string{"skill:go"},
					ToolRefs: []string{"tool:test"}, CapabilityRefs: []string{"capability:patch"},
					OutputContract: goal.OutputContractEvidenceBundle},
				{Key: "b", Objective: "overlapping writer", Phase: "phase:build", Role: "role:worker",
					WriteSet: []string{"internal/shared/file.go"}, OutputContract: goal.OutputContractEvidenceBundle},
				{Key: "c", Objective: "free writer", Phase: "phase:build", Role: "role:worker",
					WriteSet: []string{"docs/free.md"}, OutputContract: goal.OutputContractEvidenceBundle},
			},
		},
	})
	if err != nil {
		t.Fatalf("submit explicit plan: %v", err)
	}
	if got := result.Record.Goal.Snapshot().Phases; len(got) != 1 ||
		got[0].Ref != "phase-instance:build" || got[0].TemplateRef != "phase-template:program" ||
		!reflect.DeepEqual(got[0].InputRefs, []string{"input:app-spec"}) ||
		!reflect.DeepEqual(got[0].CriterionRefs, []string{"criterion:tests-green"}) {
		t.Fatalf("phase contract lost: %+v", got)
	}
	if len(result.Record.Executions) != 2 {
		t.Fatalf("scheduled executions = %d, want maximal conflict-free cohort of 2", len(result.Record.Executions))
	}
	scheduled := make(map[string]bool)
	for _, execution := range result.Record.Executions {
		item, _ := result.Record.Goal.WorkItem(execution.WorkItemRef)
		scheduled[item.Objective()] = true
	}
	if !scheduled["first writer"] || !scheduled["free writer"] || scheduled["overlapping writer"] {
		t.Fatalf("scheduled cohort = %+v", scheduled)
	}
	if _, err := orchestrator.ProcessNext(context.Background(), "worker:v05"); err != nil {
		t.Fatalf("launch required work: %v", err)
	}
	agent.mu.Lock()
	requests := append([]ports.AgentLaunchRequest(nil), agent.launchRequests...)
	agent.mu.Unlock()
	if len(requests) != 1 || !reflect.DeepEqual(requests[0].SkillRefs, []string{"skill:go"}) ||
		!reflect.DeepEqual(requests[0].ToolRefs, []string{"tool:test"}) ||
		!reflect.DeepEqual(requests[0].CapabilityRefs, []string{"capability:patch"}) ||
		requests[0].PhaseRef != "phase-instance:build" || requests[0].PhaseTemplateRef != "phase-template:program" ||
		!reflect.DeepEqual(requests[0].PhaseInputRefs, []string{"input:app-spec"}) ||
		!reflect.DeepEqual(requests[0].PhaseCriterionRefs, []string{"criterion:tests-green"}) {
		t.Fatalf("launch requirements = %+v", requests)
	}
}

func TestSharedPlanCompilerPreservesInitialAndExtensionGraph(t *testing.T) {
	clock := &mutableClock{now: time.Date(2026, 7, 15, 1, 0, 0, 0, time.UTC)}
	repository := newMemoryRepository()
	agent := &scriptedAgent{now: clock.Now}
	orchestrator, _ := newTestOrchestrator(t, repository, clock, agent)
	actor, project := testScope(t)

	result, err := orchestrator.Submit(context.Background(), accessForScope(t, actor, project), SubmitRequest{
		RequestRef: "request:shared-plan-compiler", Statement: "compile initial graph", Confirm: true,
		Plan: &PlanSpec{
			Phases: []PhaseSpec{{
				Ref: "phase-instance:initial", Key: "phase:initial", TemplateRef: "phase-template:initial",
			}},
			WorkItems: []WorkItemSpec{
				{Key: "root", Objective: "root", Phase: "phase:initial", Role: "role:worker", OutputContract: goal.OutputContractEvidenceBundle},
				{Key: "child", Objective: "child", Phase: "phase:initial", Role: "role:worker", Parent: "root", OutputContract: goal.OutputContractArtifact},
				{Key: "dependent", Objective: "dependent", Phase: "phase:initial", Role: "role:reviewer", Dependencies: []string{"root"}, OutputContract: goal.OutputContractAttestation},
			},
		},
	})
	if err != nil {
		t.Fatalf("submit initial graph: %v", err)
	}
	initial := result.Record.Goal.WorkItems()
	if len(initial) != 3 {
		t.Fatalf("initial items = %d, want 3", len(initial))
	}
	parent, ok := initial[1].Parent()
	if !ok || parent != initial[0].Ref() || !reflect.DeepEqual(initial[2].Dependencies(), []goal.WorkItemRef{initial[0].Ref()}) {
		t.Fatalf("initial refs unresolved: parent=%s/%v dependencies=%v", parent.String(), ok, initial[2].Dependencies())
	}

	plan, err := orchestrator.compilePlanExtension(context.Background(), result.Record.Goal, PlanSpec{
		Phases: []PhaseSpec{{
			Ref: "phase-instance:extension", Key: "phase:extension", TemplateRef: "phase-template:extension",
			InputRefs: []string{"input:extension"}, CriterionRefs: []string{"criterion:extension"},
		}},
		WorkItems: []WorkItemSpec{
			{Key: "extension-root", Objective: "extension root", Phase: "phase:extension", Role: "role:worker", Dependencies: []string{initial[2].Ref().String()}, OutputContract: goal.OutputContractEvidenceBundle},
			{Key: "extension-child", Objective: "extension child", Phase: "phase:extension", Role: "role:reviewer", Parent: initial[1].Ref().String(), Dependencies: []string{"extension-root"}, OutputContract: goal.OutputContractAttestation},
		},
	}, clock.Now().Add(time.Second))
	if err != nil {
		t.Fatalf("compile extension: %v", err)
	}
	items := plan.WorkItems()
	phases := plan.Phases()
	if plan.Generation() != 2 || len(items) != 5 || len(phases) != 2 || !reflect.DeepEqual(items[:3], initial) {
		t.Fatalf("non-monotonic extension: generation=%d phases=%d items=%d prefix=%v", plan.Generation(), len(phases), len(items), reflect.DeepEqual(items[:3], initial))
	}
	if !reflect.DeepEqual(items[3].Dependencies(), []goal.WorkItemRef{initial[2].Ref()}) {
		t.Fatalf("existing dependency unresolved: %v", items[3].Dependencies())
	}
	extensionParent, ok := items[4].Parent()
	if !ok || extensionParent != initial[1].Ref() || !reflect.DeepEqual(items[4].Dependencies(), []goal.WorkItemRef{items[3].Ref()}) {
		t.Fatalf("extension refs unresolved: parent=%s/%v dependencies=%v", extensionParent.String(), ok, items[4].Dependencies())
	}
	if _, err := result.Record.Goal.ApplyPlan(result.Record.Goal.Revision(), plan); err != nil {
		t.Fatalf("apply compiled extension: %v", err)
	}
}

func TestPlanExtensionRejectsRequestKeyCollidingWithExistingWorkItemRef(t *testing.T) {
	clock := &mutableClock{now: time.Date(2026, 7, 15, 1, 30, 0, 0, time.UTC)}
	repository := newMemoryRepository()
	agent := &scriptedAgent{now: clock.Now}
	orchestrator, _ := newTestOrchestrator(t, repository, clock, agent)
	actor, project := testScope(t)
	result, err := orchestrator.Submit(context.Background(), accessForScope(t, actor, project), SubmitRequest{
		RequestRef: "request:key-ref-collision", Statement: "compile base", Confirm: true,
	})
	if err != nil {
		t.Fatalf("submit base: %v", err)
	}
	existingRef := result.Record.Goal.WorkItems()[0].Ref().String()
	_, err = orchestrator.compilePlanExtension(context.Background(), result.Record.Goal, PlanSpec{
		WorkItems: []WorkItemSpec{{
			Key: existingRef, Objective: "ambiguous item", Phase: goal.DefaultPhaseKey().String(),
			Role: goal.DefaultRoleKey().String(), OutputContract: goal.OutputContractEvidenceBundle,
		}},
	}, clock.Now().Add(time.Second))
	if err == nil || err.Error() != "application.plan_item_key_ref_collision" {
		t.Fatalf("collision error = %v", err)
	}
}

func clonePlanSpec(input *PlanSpec) *PlanSpec {
	result := &PlanSpec{Phases: append([]PhaseSpec(nil), input.Phases...)}
	result.WorkItems = append([]WorkItemSpec(nil), input.WorkItems...)
	return result
}
