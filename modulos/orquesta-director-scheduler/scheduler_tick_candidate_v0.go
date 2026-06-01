package orquestadirectorscheduler

import (
	orquestacoreconcurrency "orquesta/modulos/orquesta-core-concurrency"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirector "orquesta/modulos/orquesta-director"
)

func (collector *schedulerTickCollectorV0) collectCandidateV0(
	candidate SchedulableWorkCandidateV0,
) error {
	if candidate.CapacityCandidate == nil {
		collector.addNeedsDirectorV0(SchedulerWaitingCandidateMissingV0)
		return nil
	}
	capacityRef := candidate.CapacityCandidate.Payload.CapacityRequestID
	if collector.capacityDecisions[capacityRef] {
		return collector.collectAgentCandidateV0(candidate)
	}
	if collector.capacityRequests[capacityRef] {
		if candidate.RecoverCapacityOutbox {
			return collector.collectRecoverCapacityOutboxV0(candidate, capacityRef)
		}
		collector.addWaitingV0(SchedulerWaitingCapacityPendingV0)
		return nil
	}
	if collector.plannedCapacityRequests[capacityRef] {
		collector.addWaitingV0(SchedulerWaitingCapacityPendingV0)
		return nil
	}
	command, err := orquestacoreworkflow.NewRequestCapacityCommandV0(
		candidate.CapacityCandidate.CommandMeta,
		candidate.CapacityCandidate.Payload,
	)
	if err != nil {
		return err
	}
	collector.addReadyCommandV0(command)
	collector.plannedCapacityRequests[capacityRef] = true
	return nil
}

func (collector *schedulerTickCollectorV0) collectRecoverCapacityOutboxV0(
	candidate SchedulableWorkCandidateV0,
	capacityRef string,
) error {
	command, err := orquestacoreworkflow.NewRequestCapacityCommandV0(
		candidate.CapacityCandidate.CommandMeta,
		candidate.CapacityCandidate.Payload,
	)
	if err != nil {
		return err
	}
	collector.addReadyCommandV0(command)
	collector.plannedCapacityRequests[capacityRef] = true
	return nil
}

func (collector *schedulerTickCollectorV0) collectAgentCandidateV0(
	candidate SchedulableWorkCandidateV0,
) error {
	if candidate.AgentCandidate == nil {
		collector.addNeedsDirectorV0(SchedulerWaitingCandidateMissingV0)
		return nil
	}
	agentRef := candidate.AgentCandidate.Payload.AgentRequestID
	if collector.startedAgents[agentRef] {
		return nil
	}
	if collector.failedAgents[agentRef] || collector.stoppedAgents[agentRef] {
		collector.addBlockedRefsV0([]string{agentRef})
		return nil
	}
	if collector.agents[agentRef] {
		if candidate.RecoverAgentOutbox {
			return collector.collectRecoverAgentOutboxV0(candidate, agentRef)
		}
		collector.addWaitingV0(SchedulerWaitingAgentLifecyclePendingV0)
		return nil
	}
	if collector.plannedAgents[agentRef] {
		collector.addWaitingV0(SchedulerWaitingAgentLifecyclePendingV0)
		return nil
	}
	return collector.collectGateAgentCandidateV0(candidate, agentRef)
}

func (collector *schedulerTickCollectorV0) collectRecoverAgentOutboxV0(
	candidate SchedulableWorkCandidateV0,
	agentRef string,
) error {
	command, err := orquestacoreworkflow.NewRequestAgentCommandV0(
		candidate.AgentCandidate.CommandMeta,
		candidate.AgentCandidate.Payload,
	)
	if err != nil {
		return err
	}
	collector.addReadyCommandV0(command)
	collector.plannedAgents[agentRef] = true
	return nil
}

func (collector *schedulerTickCollectorV0) collectGateAgentCandidateV0(
	candidate SchedulableWorkCandidateV0,
	agentRef string,
) error {
	result, err := collector.buildSchedulerGateAgentResultV0(candidate)
	if err != nil {
		return err
	}
	gateRecorded := collector.concurrencyGates[result.Evaluation.GateRef] ||
		collector.plannedConcurrencyGates[result.Evaluation.GateRef]
	switch result.Evaluation.Decision {
	case orquestacoreconcurrency.ConcurrencyGateDecisionAllowRequestAgentV0:
		collector.addGateIfMissingV0(result, gateRecorded)
		for _, command := range result.AgentCommands {
			collector.addReadyCommandV0(command)
		}
		collector.plannedAgents[agentRef] = true
	case orquestacoreconcurrency.ConcurrencyGateDecisionBlockRequestAgentV0:
		collector.addGateIfMissingV0(result, gateRecorded)
		collector.addBlockedRefsV0(result.BlockedClaimRefs)
	default:
		collector.addGateIfMissingV0(result, gateRecorded)
		collector.needsDirector = true
	}
	return nil
}

func (collector *schedulerTickCollectorV0) buildSchedulerGateAgentResultV0(
	candidate SchedulableWorkCandidateV0,
) (orquestadirector.ConcurrencyGateAgentRequestsResultV0, error) {
	return orquestadirector.BuildConcurrencyGateAgentRequestsV0(orquestadirector.ConcurrencyGateAgentRequestsInputV0{
		GateCommandMeta:  candidate.GateCommandMeta,
		Claims:           collector.claimsForGateV0(candidate),
		SubjectClaimRefs: candidate.SubjectClaimRefs,
		EvidenceRefs:     candidate.GateEvidenceRefs,
		CandidateRequests: []orquestadirector.CandidateAgentRequestV0{
			{
				ClaimRef:    candidate.AgentCandidate.ClaimRef,
				CommandMeta: candidate.AgentCandidate.CommandMeta,
				Payload:     candidate.AgentCandidate.Payload,
			},
		},
	})
}

func (collector *schedulerTickCollectorV0) claimsForGateV0(
	candidate SchedulableWorkCandidateV0,
) []orquestacoreconcurrency.WorksetClaimV0 {
	if len(collector.input.WorkClaims) > 0 {
		return collector.input.WorkClaims
	}
	return candidate.Claims
}

func (collector *schedulerTickCollectorV0) addGateIfMissingV0(
	result orquestadirector.ConcurrencyGateAgentRequestsResultV0,
	gateRecorded bool,
) {
	if gateRecorded {
		return
	}
	collector.addCommandV0(result.GateCommand)
	collector.plannedConcurrencyGates[result.Evaluation.GateRef] = true
}
