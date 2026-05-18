package orquestacionnucleoapp

import orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"

func progressiveStatusAfterMaxBurstsV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	waitAgentRefs []string,
	waitScopeApplied bool,
) ProgressiveLoopStatusV0 {
	if waitScopeApplied {
		if len(compactStringsV0(waitAgentRefs)) > 0 &&
			runHasPendingExternalAgentRefsV0(run, waitAgentRefs) {
			return ProgressiveLoopStatusWaitExternalV0
		}
		return ProgressiveLoopStatusMaxBurstsV0
	}
	if len(compactStringsV0(waitAgentRefs)) > 0 {
		if runHasPendingExternalAgentRefsV0(run, waitAgentRefs) {
			return ProgressiveLoopStatusWaitExternalV0
		}
		return ProgressiveLoopStatusMaxBurstsV0
	}
	if runHasPendingExternalAgentsV0(run) {
		return ProgressiveLoopStatusWaitExternalV0
	}
	return ProgressiveLoopStatusMaxBurstsV0
}

func runHasPendingExternalAgentsV0(run orquestacoreworkflow.OrchestrationRunV0) bool {
	started := len(compactStringsV0(run.StartedAgents))
	if started == 0 {
		return false
	}
	closed := len(compactStringsV0(run.Deliveries)) +
		len(compactStringsV0(run.PhaseArtifacts)) +
		len(compactStringsV0(run.FailedAgents)) +
		len(compactStringsV0(run.StoppedAgents))
	return started > closed
}

func runHasPendingExternalAgentRefsV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	agentRefs []string,
) bool {
	started := compactStringsV0(run.StartedAgents)
	delivered := compactStringsV0(run.DeliveredAgents)
	failed := compactStringsV0(run.FailedAgents)
	lost := compactStringsV0(run.LostAgents)
	confirmedStopped := compactStringsV0(run.ConfirmedStoppedAgents)
	for _, agentRef := range compactStringsV0(agentRefs) {
		if !stringInSetV0(agentRef, started) {
			continue
		}
		if stringInSetV0(agentRef, delivered) ||
			stringInSetV0(agentRef, failed) ||
			stringInSetV0(agentRef, lost) ||
			stringInSetV0(agentRef, confirmedStopped) {
			continue
		}
		return true
	}
	return false
}
