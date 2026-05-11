package orquestacionnucleoapp

import orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"

func progressiveStatusAfterMaxBurstsV0(
	run orquestacoreworkflow.OrchestrationRunV0,
) ProgressiveLoopStatusV0 {
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
