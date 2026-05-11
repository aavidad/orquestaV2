package orquestadirectorscheduler

import (
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
)

func TestBuildDirectorSchedulerTickV0QualityGateBlocksUnrelatedReplanCandidate(t *testing.T) {
	input := validSchedulerTickInputV0()
	input.Snapshot.BlockingQualityGateRefs = []string{"quality-gate-ref-blocked-001"}
	input.ReplanFollowupCandidates = []SchedulableReplanFollowupCandidateV0{
		validSchedulableReplanFollowupCandidateV0(orquestacoreworkflow.ReplanDecisionActionRetryTaskV0),
	}
	input.ReplanFollowupCandidates[0].ReplanFollowupsInput.DecisionPayload.SourceRef = "quality-gate-ref-other-001"

	plan := mustSchedulerTickPlanV0(t, input)

	assertSchedulerPlanV0(t, plan, SchedulerTickStatusBlockedV0, 0)
	assertSchedulerWaitingV0(t, plan, SchedulerWaitingQualityGateFollowupV0)
	if len(plan.BlockedRefs) != 1 || plan.BlockedRefs[0] != "quality-gate-ref-blocked-001" {
		t.Fatalf("blocked_refs=%v", plan.BlockedRefs)
	}
}

func TestBuildDirectorSchedulerTickV0QualityGateRecordedReplanRequestsCapacityIfMissing(t *testing.T) {
	input := validSchedulerTickInputV0()
	input.Snapshot.BlockingQualityGateRefs = []string{"quality-gate-ref-blocked-001"}
	input.Snapshot.ReplanRefs = []string{"replan-ref-scheduler-001"}
	input.ReplanFollowupCandidates = []SchedulableReplanFollowupCandidateV0{
		validSchedulableReplanFollowupCandidateV0(orquestacoreworkflow.ReplanDecisionActionRetryTaskV0),
	}
	input.ReplanFollowupCandidates[0].ReplanFollowupsInput.DecisionPayload.SourceRef = "quality-gate-ref-blocked-001"

	plan := mustSchedulerTickPlanV0(t, input)

	assertSchedulerPlanV0(t, plan, SchedulerTickStatusCommandsReadyV0, 1)
	assertSchedulerCommandTypesV0(t, plan,
		orquestacoreworkflow.OrchestrationCommandRequestCapacityV0,
	)
}

func TestBuildDirectorSchedulerTickV0QualityGateRecordedReplanWaitsForRequestedCapacity(t *testing.T) {
	input := validSchedulerTickInputV0()
	input.Snapshot.BlockingQualityGateRefs = []string{"quality-gate-ref-blocked-001"}
	input.Snapshot.ReplanRefs = []string{"replan-ref-scheduler-001"}
	input.Snapshot.CapacityRequests = []string{"capacity-ref-replan-scheduler-001"}
	input.ReplanFollowupCandidates = []SchedulableReplanFollowupCandidateV0{
		validSchedulableReplanFollowupCandidateV0(orquestacoreworkflow.ReplanDecisionActionRetryTaskV0),
	}
	input.ReplanFollowupCandidates[0].ReplanFollowupsInput.DecisionPayload.SourceRef = "quality-gate-ref-blocked-001"

	plan := mustSchedulerTickPlanV0(t, input)

	assertSchedulerPlanV0(t, plan, SchedulerTickStatusWaitingV0, 0)
	assertSchedulerWaitingV0(t, plan, SchedulerWaitingCapacityPendingV0)
}

func TestBuildDirectorSchedulerTickV0QualityGateRecordedReplanRequestsAgentAfterCapacityDecision(t *testing.T) {
	input := validSchedulerTickInputV0()
	input.Snapshot.BlockingQualityGateRefs = []string{"quality-gate-ref-blocked-001"}
	input.Snapshot.ReplanRefs = []string{"replan-ref-scheduler-001"}
	input.Snapshot.CapacityDecisions = []string{"capacity-ref-replan-scheduler-001"}
	input.ReplanFollowupCandidates = []SchedulableReplanFollowupCandidateV0{
		validSchedulableReplanFollowupCandidateV0(orquestacoreworkflow.ReplanDecisionActionReplaceAgentV0),
	}
	input.ReplanFollowupCandidates[0].ReplanFollowupsInput.DecisionPayload.SourceRef = "quality-gate-ref-blocked-001"

	plan := mustSchedulerTickPlanV0(t, input)

	assertSchedulerPlanV0(t, plan, SchedulerTickStatusCommandsReadyV0, 1)
	assertSchedulerCommandTypesV0(t, plan,
		orquestacoreworkflow.OrchestrationCommandRequestAgentV0,
	)
	assertSchedulerAgentRefV0(t, plan.Commands[0], "agent-ref-replan-scheduler-001")
}
