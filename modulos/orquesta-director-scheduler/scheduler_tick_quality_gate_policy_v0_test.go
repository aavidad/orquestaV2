package orquestadirectorscheduler

import (
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
)

func TestBuildDirectorSchedulerTickV0QualityGateBlockedRequiresReplanBeforePhaseAdvance(t *testing.T) {
	input := validSchedulerTickInputV0()
	input.Snapshot.BlockingQualityGateRefs = []string{"quality-gate-ref-blocked-001"}
	input.WorkCandidates[0].CapacityCandidate.Payload.PhaseID = string(orquestacoreworkflow.OrchestrationPhaseDocumentacionV0)
	input.WorkCandidates[0].AgentCandidate.Payload.PhaseID = string(orquestacoreworkflow.OrchestrationPhaseDocumentacionV0)

	plan := mustSchedulerTickPlanV0(t, input)

	assertSchedulerPlanV0(t, plan, SchedulerTickStatusBlockedV0, 0)
	assertSchedulerWaitingV0(t, plan, SchedulerWaitingQualityGateFollowupV0)
	if len(plan.BlockedRefs) != 1 || plan.BlockedRefs[0] != "quality-gate-ref-blocked-001" {
		t.Fatalf("blocked_refs=%v", plan.BlockedRefs)
	}

	input.ReplanFollowupCandidates = []SchedulableReplanFollowupCandidateV0{
		validSchedulableReplanFollowupCandidateV0(orquestacoreworkflow.ReplanDecisionActionRetryTaskV0),
	}
	input.ReplanFollowupCandidates[0].ReplanFollowupsInput.DecisionPayload.SourceRef = "quality-gate-ref-blocked-001"
	plan = mustSchedulerTickPlanV0(t, input)

	assertSchedulerPlanV0(t, plan, SchedulerTickStatusCommandsReadyV0, 2)
	assertSchedulerCommandTypesV0(t, plan,
		orquestacoreworkflow.OrchestrationCommandRecordReplanDecisionV0,
		orquestacoreworkflow.OrchestrationCommandRequestCapacityV0,
	)
	assertSchedulerCapacityRefV0(t, plan.Commands[1], "capacity-ref-replan-scheduler-001")
}
