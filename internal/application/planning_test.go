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

func clonePlanSpec(input *PlanSpec) *PlanSpec {
	result := &PlanSpec{Phases: append([]PhaseSpec(nil), input.Phases...)}
	result.WorkItems = append([]WorkItemSpec(nil), input.WorkItems...)
	return result
}
