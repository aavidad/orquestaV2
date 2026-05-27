package orquestaappcodexstack

import (
	"strings"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
)

func drainRunHasScopedPartialClosureV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	waitAgentRefs []string,
) bool {
	waitAgentRefs = compactStringsV0(waitAgentRefs)
	if len(waitAgentRefs) == 0 {
		return false
	}
	pending := false
	closed := false
	for _, agentRef := range waitAgentRefs {
		if drainRunHasPendingExternalAgentRefsV0(run, []string{agentRef}) {
			pending = true
		} else if drainRunAgentClosedV0(run, agentRef) {
			closed = true
		}
	}
	return pending && closed
}

func drainRunAgentClosedV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	agentRef string,
) bool {
	agentRef = strings.TrimSpace(agentRef)
	if agentRef == "" {
		return false
	}
	if codexStackStringInSetV0(run.DeliveredAgents, agentRef) ||
		codexStackStringInSetV0(run.FailedAgents, agentRef) ||
		codexStackStringInSetV0(run.LostAgents, agentRef) ||
		codexStackStringInSetV0(run.StoppedAgents, agentRef) ||
		codexStackStringInSetV0(run.ConfirmedStoppedAgents, agentRef) {
		return true
	}
	for _, artifact := range compactStringsV0(run.PhaseArtifacts) {
		if strings.Contains(artifact, "#agent:"+agentRef) {
			return true
		}
	}
	return false
}
