package application

import (
	"testing"

	"orquesta/internal/goal"
)

func TestSubmissionFingerprintFramesPlanCollections(t *testing.T) {
	actor, project := testScope(t)
	base := SubmitRequest{
		RequestRef: "request:fingerprint", ActorRef: actor, ProjectRef: project, Statement: "same",
		Plan: &PlanSpec{Phases: []string{"phase:test"}, WorkItems: []WorkItemSpec{{
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
	if submissionFingerprint(left) == submissionFingerprint(right) {
		t.Fatal("dependency/write-set boundary collision")
	}
}

func clonePlanSpec(input *PlanSpec) *PlanSpec {
	result := &PlanSpec{Phases: append([]string(nil), input.Phases...)}
	result.WorkItems = append([]WorkItemSpec(nil), input.WorkItems...)
	return result
}
