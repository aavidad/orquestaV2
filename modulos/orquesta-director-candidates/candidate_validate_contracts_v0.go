package orquestadirectorcandidates

import (
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirector "orquesta/modulos/orquesta-director"
	orquestadirectorscheduler "orquesta/modulos/orquesta-director-scheduler"
)

func validateCandidateContractsV0(
	candidate orquestadirectorscheduler.SchedulableWorkCandidateV0,
) error {
	if err := validateCapacityContractV0(candidate); err != nil {
		return err
	}
	if err := validateAgentContractV0(candidate); err != nil {
		return err
	}
	return validateGateContractV0(candidate)
}

func validateCapacityContractV0(
	candidate orquestadirectorscheduler.SchedulableWorkCandidateV0,
) error {
	_, err := orquestacoreworkflow.NewRequestCapacityCommandV0(
		candidate.CapacityCandidate.CommandMeta,
		candidate.CapacityCandidate.Payload,
	)
	if err != nil {
		return candidateErrorV0("capacity")
	}
	return nil
}

func validateAgentContractV0(
	candidate orquestadirectorscheduler.SchedulableWorkCandidateV0,
) error {
	_, err := orquestacoreworkflow.NewRequestAgentCommandV0(
		candidate.AgentCandidate.CommandMeta,
		candidate.AgentCandidate.Payload,
	)
	if err != nil {
		return candidateErrorV0("agent")
	}
	return nil
}

func validateGateContractV0(
	candidate orquestadirectorscheduler.SchedulableWorkCandidateV0,
) error {
	_, err := orquestadirector.BuildConcurrencyGateAgentRequestsV0(orquestadirector.ConcurrencyGateAgentRequestsInputV0{
		GateCommandMeta:  candidate.GateCommandMeta,
		Claims:           candidate.Claims,
		SubjectClaimRefs: candidate.SubjectClaimRefs,
		EvidenceRefs:     candidate.GateEvidenceRefs,
		CandidateRequests: []orquestadirector.CandidateAgentRequestV0{{
			ClaimRef:    candidate.AgentCandidate.ClaimRef,
			CommandMeta: candidate.AgentCandidate.CommandMeta,
			Payload:     candidate.AgentCandidate.Payload,
		}},
	})
	if err != nil {
		return candidateErrorV0("gate")
	}
	return nil
}
