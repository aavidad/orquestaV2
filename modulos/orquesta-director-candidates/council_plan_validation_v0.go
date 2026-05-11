package orquestadirectorcandidates

import (
	"strings"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadecisioncouncil "orquesta/modulos/orquesta-decision-council"
)

func normalizeCouncilPlanCandidatesInputV0(input CouncilPlanCandidatesInputV0) CouncilPlanCandidatesInputV0 {
	input.RunRef = strings.TrimSpace(input.RunRef)
	input.PhaseID = strings.TrimSpace(input.PhaseID)
	input.Role = strings.TrimSpace(input.Role)
	input.OccurredAt = strings.TrimSpace(input.OccurredAt)
	input.CorrelationID = strings.TrimSpace(input.CorrelationID)
	input.RequestedBy = strings.TrimSpace(input.RequestedBy)
	input.EvidenceRefs = normalizeRefsV0(input.EvidenceRefs)
	input.Plan.RunRef = strings.TrimSpace(input.Plan.RunRef)
	input.Plan.DecisionTopicRef = strings.TrimSpace(input.Plan.DecisionTopicRef)
	return input
}

func validateCouncilPlanCandidatesInputV0(input CouncilPlanCandidatesInputV0) error {
	switch {
	case input.RunRef == "":
		return candidateErrorV0("run_ref")
	case input.PhaseID == "":
		return candidateErrorV0("phase_id")
	case input.Role == "":
		return candidateErrorV0("role")
	case input.OccurredAt == "":
		return candidateErrorV0("occurred_at")
	case input.Plan.RunRef != input.RunRef:
		return candidateErrorV0("plan.run_ref")
	case input.Plan.DecisionTopicRef == "":
		return candidateErrorV0("plan.decision_topic_ref")
	case len(input.Plan.Assignments) == 0:
		return candidateErrorV0("plan.assignments")
	case !validCouncilCandidateRoleV0(input.Role):
		return candidateErrorV0("role")
	}
	if input.MinimumRecommendedCapacity != "" {
		if !validCouncilCapacityRecommendationV0(input.MinimumRecommendedCapacity) {
			return candidateErrorV0("minimum_recommended_capacity")
		}
	}
	for _, assignment := range input.Plan.Assignments {
		if err := validateCouncilAssignmentForCandidatesV0(assignment); err != nil {
			return err
		}
	}
	return nil
}

func validCouncilCapacityRecommendationV0(
	capacity orquestacoreworkflow.OrchestrationCapacityRecommendationV0,
) bool {
	switch capacity {
	case orquestacoreworkflow.OrchestrationCapacityLowV0,
		orquestacoreworkflow.OrchestrationCapacityMediumV0,
		orquestacoreworkflow.OrchestrationCapacityHighV0,
		orquestacoreworkflow.OrchestrationCapacityXHighV0:
		return true
	default:
		return false
	}
}

func validateCouncilAssignmentForCandidatesV0(
	assignment orquestadecisioncouncil.CouncilAssignmentV0,
) error {
	switch {
	case strings.TrimSpace(assignment.AssignmentRef) == "":
		return candidateErrorV0("plan.assignments.assignment_ref")
	case strings.TrimSpace(assignment.Role) == "":
		return candidateErrorV0("plan.assignments.role")
	case strings.TrimSpace(assignment.AgentRef) == "":
		return candidateErrorV0("plan.assignments.agent_ref")
	case strings.TrimSpace(assignment.ExpectedArtifact) == "":
		return candidateErrorV0("plan.assignments.expected_artifact")
	case !validCouncilCandidateRoleV0(assignment.Role):
		return candidateErrorV0("plan.assignments.role")
	}
	return nil
}

func validCouncilCandidateRoleV0(role string) bool {
	switch role {
	case orquestadecisioncouncil.CouncilRoleProposalV0,
		orquestadecisioncouncil.CouncilRoleCritiqueV0,
		orquestadecisioncouncil.CouncilRoleVoteV0:
		return true
	default:
		return false
	}
}
