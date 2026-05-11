package orquestaapprunner

import orquestaappplanner "orquesta/modulos/orquesta-app-planner"

func appRunVoteRefV0(request PrepareAppOrchestrationRequestV0) string {
	return "vote-ref-" + safeAppRunnerRefPartV0(request.RunRef)
}

func appRunDecisionRefV0(request PrepareAppOrchestrationRequestV0) string {
	return "decision-ref-" + safeAppRunnerRefPartV0(request.RunRef)
}

func appRunContractRefV0(plan orquestaappplanner.AppMicrotaskPlanV0) string {
	return "contract:function:" + safeAppRunnerRefPartV0(plan.AppRef) + ":v0"
}
