package orquestaappcodexstack

import (
	"strings"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
)

func drainProgressiveResultV0(
	status orquestacionnucleoapp.ProgressiveLoopStatusV0,
	run orquestacoreworkflow.OrchestrationRunV0,
) orquestacionnucleoapp.ProgressiveLoopResultV0 {
	return orquestacionnucleoapp.ProgressiveLoopResultV0{
		Status: status,
		Run:    run,
	}
}

func drainRunHasPendingExternalAgentsV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	waitAgentRefs []string,
) bool {
	if len(compactStringsV0(waitAgentRefs)) > 0 {
		return drainRunHasPendingExternalAgentRefsV0(run, waitAgentRefs)
	}
	started := len(compactStringsV0(run.StartedAgents))
	closed := len(compactStringsV0(run.Deliveries)) +
		len(compactStringsV0(run.PhaseArtifacts)) +
		len(compactStringsV0(run.FailedAgents)) +
		len(compactStringsV0(run.LostAgents)) +
		len(compactStringsV0(run.ConfirmedStoppedAgents))
	return started > closed
}

func drainRunHasPendingExternalAgentRefsV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	agentRefs []string,
) bool {
	started := compactStringsV0(run.StartedAgents)
	delivered := compactStringsV0(run.DeliveredAgents)
	failed := compactStringsV0(run.FailedAgents)
	lost := compactStringsV0(run.LostAgents)
	stopped := compactStringsV0(run.StoppedAgents)
	confirmedStopped := compactStringsV0(run.ConfirmedStoppedAgents)
	for _, agentRef := range compactStringsV0(agentRefs) {
		if !codexStackStringInSetV0(started, agentRef) {
			continue
		}
		if codexStackStringInSetV0(delivered, agentRef) ||
			codexStackStringInSetV0(failed, agentRef) ||
			codexStackStringInSetV0(lost, agentRef) ||
			codexStackStringInSetV0(stopped, agentRef) ||
			codexStackStringInSetV0(confirmedStopped, agentRef) {
			continue
		}
		return true
	}
	return false
}

func drainObservationsForWaitAgentRefsV0(
	observations []orquestacionnucleoapp.AgentDeliveryObservationV0,
	waitAgentRefs []string,
) []orquestacionnucleoapp.AgentDeliveryObservationV0 {
	waitAgentRefs = compactStringsV0(waitAgentRefs)
	if len(waitAgentRefs) == 0 {
		return observations
	}
	filtered := make([]orquestacionnucleoapp.AgentDeliveryObservationV0, 0, len(observations))
	for _, observation := range observations {
		if drainObservationMatchesWaitAgentRefsV0(observation, waitAgentRefs) {
			filtered = append(filtered, observation)
		}
	}
	return filtered
}

func drainProgressObservationsForWaitAgentRefsV0(
	observations []orquestacionnucleoapp.AgentProgressObservationV0,
	waitAgentRefs []string,
) []orquestacionnucleoapp.AgentProgressObservationV0 {
	waitAgentRefs = compactStringsV0(waitAgentRefs)
	if len(waitAgentRefs) == 0 {
		return observations
	}
	filtered := make([]orquestacionnucleoapp.AgentProgressObservationV0, 0, len(observations))
	for _, observation := range observations {
		if drainAgentRefMatchesWaitAgentRefsV0(observation.Report.AgentRequestID, waitAgentRefs) {
			filtered = append(filtered, observation)
		}
	}
	return filtered
}

func drainObservationMatchesWaitAgentRefsV0(
	observation orquestacionnucleoapp.AgentDeliveryObservationV0,
	waitAgentRefs []string,
) bool {
	waitAgentRefs = compactStringsV0(waitAgentRefs)
	if len(waitAgentRefs) == 0 {
		return true
	}
	return drainAgentRefMatchesWaitAgentRefsV0(observation.AgentRef, waitAgentRefs)
}

func drainAgentRefMatchesWaitAgentRefsV0(
	agentRef string,
	waitAgentRefs []string,
) bool {
	return codexStackStringInSetV0(compactStringsV0(waitAgentRefs), strings.TrimSpace(agentRef))
}
