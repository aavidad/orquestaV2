package orquestadirectorscheduler

import (
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirector "orquesta/modulos/orquesta-director"
)

func TestBuildDirectorSchedulerTickV0LeaseStopAgentPriorityOverWorkCandidate(t *testing.T) {
	input := validSchedulerTickInputWithLeaseV0(orquestacoreworkflow.AgentLeaseActionStopAgentV0)

	plan := mustSchedulerTickPlanV0(t, input)

	assertSchedulerPlanV0(t, plan, SchedulerTickStatusCommandsReadyV0, 2)
	assertSchedulerCommandTypesV0(t, plan,
		orquestacoreworkflow.OrchestrationCommandRegisterAgentLeaseExpiredV0,
		orquestacoreworkflow.OrchestrationCommandStopAgentV0,
	)
}

func TestBuildDirectorSchedulerTickV0LeaseAskDirector(t *testing.T) {
	input := validSchedulerTickInputWithLeaseV0(orquestacoreworkflow.AgentLeaseActionAskDirectorV0)
	input.WorkCandidates = nil

	plan := mustSchedulerTickPlanV0(t, input)

	assertSchedulerPlanV0(t, plan, SchedulerTickStatusCommandsReadyV0, 2)
	assertSchedulerCommandTypesV0(t, plan,
		orquestacoreworkflow.OrchestrationCommandRegisterAgentLeaseExpiredV0,
		orquestacoreworkflow.OrchestrationCommandAskDirectorV0,
	)
}

func TestBuildDirectorSchedulerTickV0LeaseUnsupportedNeedsDirector(t *testing.T) {
	input := validSchedulerTickInputWithLeaseV0(orquestacoreworkflow.AgentLeaseActionRetryV0)
	input.WorkCandidates = nil

	plan := mustSchedulerTickPlanV0(t, input)

	assertSchedulerPlanV0(t, plan, SchedulerTickStatusNeedsDirectorV0, 1)
	assertSchedulerCommandTypesV0(t, plan,
		orquestacoreworkflow.OrchestrationCommandRegisterAgentLeaseExpiredV0,
	)
}

func TestBuildDirectorSchedulerTickV0DoesNotRepeatExpiredLease(t *testing.T) {
	input := validSchedulerTickInputWithLeaseV0(orquestacoreworkflow.AgentLeaseActionStopAgentV0)
	input.WorkCandidates = nil
	input.Snapshot.ExpiredLeaseRefs = []string{"lease-ref-scheduler-001"}

	plan := mustSchedulerTickPlanV0(t, input)

	assertSchedulerPlanV0(t, plan, SchedulerTickStatusQuiescentV0, 0)
}

func TestBuildDirectorSchedulerTickV0PendingOutboxBlocksLeaseCandidates(t *testing.T) {
	input := validSchedulerTickInputWithLeaseV0(orquestacoreworkflow.AgentLeaseActionStopAgentV0)
	input.Snapshot.PendingOutboxRefs = []string{"outbox-ref-lease-001"}

	plan := mustSchedulerTickPlanV0(t, input)

	assertSchedulerPlanV0(t, plan, SchedulerTickStatusWaitingV0, 0)
	assertSchedulerWaitingV0(t, plan, SchedulerWaitingOutboxPendingV0)
}

func TestBuildDirectorSchedulerTickV0LeaseMissingAgentNeedsDirector(t *testing.T) {
	input := validSchedulerTickInputWithLeaseV0(orquestacoreworkflow.AgentLeaseActionStopAgentV0)
	input.WorkCandidates = nil
	input.Snapshot.Agents = nil

	plan := mustSchedulerTickPlanV0(t, input)

	assertSchedulerPlanV0(t, plan, SchedulerTickStatusNeedsDirectorV0, 0)
	assertSchedulerWaitingV0(t, plan, SchedulerWaitingCandidateMissingV0)
}

func TestBuildDirectorSchedulerTickV0RejectsForeignRunLeaseCandidate(t *testing.T) {
	input := validSchedulerTickInputWithLeaseV0(orquestacoreworkflow.AgentLeaseActionStopAgentV0)
	input.LeaseActionCandidates[0].PostLeaseActionInput.CommandMeta.RunID = "run-externo-001"

	_, err := BuildDirectorSchedulerTickV0(input)
	assertSchedulerTickErrorV0(t, err, "lease_action_candidates.post_lease_action_input.command_meta.run_id")
}

func validSchedulerTickInputWithLeaseV0(
	action orquestacoreworkflow.AgentLeaseRecommendedActionV0,
) DirectorSchedulerTickInputV0 {
	input := validSchedulerTickInputV0()
	input.Snapshot.Agents = []string{"agent-ref-scheduler-001"}
	input.LeaseActionCandidates = []SchedulableLeaseActionCandidateV0{
		validSchedulableLeaseActionCandidateV0(action),
	}
	return input
}

func validSchedulableLeaseActionCandidateV0(
	action orquestacoreworkflow.AgentLeaseRecommendedActionV0,
) SchedulableLeaseActionCandidateV0 {
	return SchedulableLeaseActionCandidateV0{
		CandidateRef:         "lease-action-candidate-ref-001",
		PostLeaseActionInput: validPostLeaseActionInputForSchedulerV0(action),
		EvidenceRefs:         []string{"evidence-ref-lease-candidate-001"},
	}
}

func validPostLeaseActionInputForSchedulerV0(
	action orquestacoreworkflow.AgentLeaseRecommendedActionV0,
) orquestadirector.PostLeaseActionInputV0 {
	input := orquestadirector.PostLeaseActionInputV0{
		CommandMeta:       schedulerMetaV0("cmd-lease-scheduler-001", "lease"),
		RunRef:            "run-scheduler-001",
		AgentRequestID:    "agent-ref-scheduler-001",
		LeaseRef:          "lease-ref-scheduler-001",
		ReasonCode:        "agent_lease_expired",
		ObservedAt:        "2026-05-06T11:00:00Z",
		RecommendedAction: action,
		EvidenceRefs:      []string{"evidence-ref-lease-001"},
	}
	if action == orquestacoreworkflow.AgentLeaseActionAskDirectorV0 {
		input.QuestionID = "question-ref-lease-001"
	}
	return input
}
