package orquestadirectorcandidates

import (
	orquestacoreconcurrency "orquesta/modulos/orquesta-core-concurrency"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectorscheduler "orquesta/modulos/orquesta-director-scheduler"
)

func BuildSchedulableWorkCandidateV0(
	input SchedulableWorkCandidateInputV0,
) (orquestadirectorscheduler.SchedulableWorkCandidateV0, error) {
	normalized := normalizeInputV0(input)
	if err := validateInputV0(normalized); err != nil {
		return orquestadirectorscheduler.SchedulableWorkCandidateV0{}, err
	}
	claims, err := buildClaimsV0(normalized)
	if err != nil {
		return orquestadirectorscheduler.SchedulableWorkCandidateV0{}, err
	}
	candidate := buildCandidateV0(normalized, claims)
	if err := validateCandidateContractsV0(candidate); err != nil {
		return orquestadirectorscheduler.SchedulableWorkCandidateV0{}, err
	}
	return candidate, nil
}

func buildCandidateV0(
	input SchedulableWorkCandidateInputV0,
	claims []orquestacoreconcurrency.WorksetClaimV0,
) orquestadirectorscheduler.SchedulableWorkCandidateV0 {
	return orquestadirectorscheduler.SchedulableWorkCandidateV0{
		CandidateRef:      input.CandidateRef,
		SubjectClaimRefs:  input.SubjectClaimRefs,
		Claims:            claims,
		CapacityCandidate: buildCapacityCandidateV0(input),
		AgentCandidate:    buildAgentCandidateV0(input),
		GateCommandMeta:   commandMetaV0(input, input.Commands.GateCommandID, input.Commands.GateIdempotencyKey),
		GateEvidenceRefs:  input.GateEvidenceRefs,
		EvidenceRefs:      input.EvidenceRefs,
	}
}

func buildCapacityCandidateV0(
	input SchedulableWorkCandidateInputV0,
) *orquestadirectorscheduler.SchedulerCapacityCommandCandidateV0 {
	return &orquestadirectorscheduler.SchedulerCapacityCommandCandidateV0{
		CommandMeta: commandMetaV0(input, input.Commands.CapacityCommandID, input.Commands.CapacityIdempotencyKey),
		Payload: orquestacoreworkflow.RequestCapacityCommandPayloadV0{
			CapacityRequestID:          input.Capacity.CapacityRequestID,
			PhaseID:                    input.PhaseID,
			TaskRef:                    input.TaskRef,
			ReasonCode:                 input.Capacity.ReasonCode,
			Summary:                    input.Capacity.Summary,
			MinimumRecommendedCapacity: input.Capacity.MinimumRecommendedCapacity,
			EvidenceRefs:               input.Capacity.EvidenceRefs,
		},
	}
}

func buildAgentCandidateV0(
	input SchedulableWorkCandidateInputV0,
) *orquestadirectorscheduler.SchedulerAgentCommandCandidateV0 {
	return &orquestadirectorscheduler.SchedulerAgentCommandCandidateV0{
		ClaimRef:    input.Agent.ClaimRef,
		CommandMeta: commandMetaV0(input, input.Commands.AgentCommandID, input.Commands.AgentIdempotencyKey),
		Payload: orquestacoreworkflow.RequestAgentCommandPayloadV0{
			AgentRequestID:     input.Agent.AgentRequestID,
			PhaseID:            input.PhaseID,
			TaskRef:            input.TaskRef,
			CapacityRequestRef: input.Capacity.CapacityRequestID,
			Role:               input.Agent.Role,
			Summary:            input.Agent.Summary,
			EvidenceRefs:       input.Agent.EvidenceRefs,
		},
	}
}
