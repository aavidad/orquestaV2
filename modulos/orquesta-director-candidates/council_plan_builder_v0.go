package orquestadirectorcandidates

import (
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadecisioncouncil "orquesta/modulos/orquesta-decision-council"
	orquestadirectorscheduler "orquesta/modulos/orquesta-director-scheduler"
)

func BuildSchedulableWorkCandidatesFromCouncilPlanV0(
	input CouncilPlanCandidatesInputV0,
) ([]orquestadirectorscheduler.SchedulableWorkCandidateV0, error) {
	normalized := normalizeCouncilPlanCandidatesInputV0(input)
	if err := validateCouncilPlanCandidatesInputV0(normalized); err != nil {
		return nil, err
	}
	candidates := make([]orquestadirectorscheduler.SchedulableWorkCandidateV0, 0)
	ordinal := 0
	for _, assignment := range normalized.Plan.Assignments {
		if assignment.Role != normalized.Role {
			continue
		}
		ordinal++
		candidate, err := BuildSchedulableWorkCandidateV0(
			candidateInputFromCouncilAssignmentV0(normalized, assignment, ordinal),
		)
		if err != nil {
			return nil, err
		}
		candidates = append(candidates, candidate)
	}
	if len(candidates) == 0 {
		return nil, candidateErrorV0("role")
	}
	return candidates, nil
}

func candidateInputFromCouncilAssignmentV0(
	input CouncilPlanCandidatesInputV0,
	assignment orquestadecisioncouncil.CouncilAssignmentV0,
	ordinal int,
) SchedulableWorkCandidateInputV0 {
	refs := councilDerivedRefsV0(assignment, ordinal)
	return SchedulableWorkCandidateInputV0{
		CandidateRef:     refs.candidateRef,
		RunRef:           input.RunRef,
		PhaseID:          input.PhaseID,
		TaskRef:          refs.taskRef,
		SubjectClaimRefs: []string{refs.claimRef},
		ScopeClaims: []WorkCandidateScopeClaimV0{{
			ClaimRef:       refs.claimRef,
			AgentRequestID: refs.agentRequestRef,
			GroupRef:       "council",
			ReadScopes:     councilReadScopesV0(assignment.Role),
			WriteScopes:    []string{councilWriteScopeV0(assignment, ordinal)},
			EvidenceRefs:   councilAssignmentEvidenceRefsV0(input, assignment, ordinal),
		}},
		Commands: WorkCandidateCommandsV0{
			CapacityCommandID:      refs.capacityCommandRef,
			CapacityIdempotencyKey: refs.capacityIdempotencyRef,
			GateCommandID:          refs.gateCommandRef,
			GateIdempotencyKey:     refs.gateIdempotencyRef,
			AgentCommandID:         refs.agentCommandRef,
			AgentIdempotencyKey:    refs.agentIdempotencyRef,
			CorrelationID:          input.CorrelationID,
			RequestedBy:            input.RequestedBy,
			OccurredAt:             input.OccurredAt,
		},
		Capacity: WorkCandidateCapacityInputV0{
			CapacityRequestID:          refs.capacityRequestRef,
			ReasonCode:                 councilReasonCodeV0(assignment.Role),
			Summary:                    councilSummaryV0(assignment.Role),
			MinimumRecommendedCapacity: councilCapacityForAssignmentV0(input, assignment),
			EvidenceRefs:               councilAssignmentEvidenceRefsV0(input, assignment, ordinal),
		},
		Agent: WorkCandidateAgentInputV0{
			AgentRequestID: refs.agentRequestRef,
			ClaimRef:       refs.claimRef,
			Role:           assignment.Role,
			Summary:        councilSummaryV0(assignment.Role),
			EvidenceRefs:   councilAssignmentEvidenceRefsV0(input, assignment, ordinal),
		},
		GateEvidenceRefs: councilAssignmentEvidenceRefsV0(input, assignment, ordinal),
		EvidenceRefs:     councilAssignmentEvidenceRefsV0(input, assignment, ordinal),
	}
}

func councilCapacityForAssignmentV0(
	input CouncilPlanCandidatesInputV0,
	assignment orquestadecisioncouncil.CouncilAssignmentV0,
) orquestacoreworkflow.OrchestrationCapacityRecommendationV0 {
	if input.MinimumRecommendedCapacity != "" {
		return input.MinimumRecommendedCapacity
	}
	for _, agent := range input.Plan.SelectedAgents {
		if agent.AgentRef == assignment.AgentRef {
			return orquestacoreworkflow.OrchestrationCapacityRecommendationV0(agent.CapacityLevel)
		}
	}
	return orquestacoreworkflow.OrchestrationCapacityHighV0
}
