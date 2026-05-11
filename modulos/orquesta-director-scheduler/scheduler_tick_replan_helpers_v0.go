package orquestadirectorscheduler

import (
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirector "orquesta/modulos/orquesta-director"
)

func replanActionRequiresCapacityV0(action orquestacoreworkflow.ReplanDecisionActionV0) bool {
	switch action {
	case orquestacoreworkflow.ReplanDecisionActionRetryTaskV0,
		orquestacoreworkflow.ReplanDecisionActionReplaceAgentV0,
		orquestacoreworkflow.ReplanDecisionActionEscalateCapacityV0:
		return true
	default:
		return false
	}
}

func replanActionRequiresAgentV0(action orquestacoreworkflow.ReplanDecisionActionV0) bool {
	switch action {
	case orquestacoreworkflow.ReplanDecisionActionRetryTaskV0,
		orquestacoreworkflow.ReplanDecisionActionReplaceAgentV0:
		return true
	default:
		return false
	}
}

func replanCapacityRefV0(input orquestadirector.ReplanFollowupsInputV0) string {
	if input.CapacityCandidate != nil {
		return input.CapacityCandidate.Payload.CapacityRequestID
	}
	if input.AgentCandidate != nil {
		return input.AgentCandidate.Payload.CapacityRequestRef
	}
	return ""
}

func replanAgentRefV0(input orquestadirector.ReplanFollowupsInputV0) string {
	if input.AgentCandidate == nil {
		return ""
	}
	return input.AgentCandidate.Payload.AgentRequestID
}

func (collector *schedulerTickCollectorV0) replanBlockedAgentRefsV0(existing []string) []string {
	refs := append([]string{}, existing...)
	refs = append(refs, schedulerMapRefsV0(collector.failedAgents)...)
	refs = append(refs, schedulerMapRefsV0(collector.stoppedAgents)...)
	refs = append(refs, schedulerMapRefsV0(collector.agents)...)
	refs = append(refs, schedulerMapRefsV0(collector.startedAgents)...)
	refs = append(refs, schedulerMapRefsV0(collector.plannedAgents)...)
	return compactSchedulerStringsV0(refs)
}

func schedulerMapRefsV0(values map[string]bool) []string {
	refs := make([]string, 0, len(values))
	for ref, ok := range values {
		if ok {
			refs = append(refs, ref)
		}
	}
	return refs
}
