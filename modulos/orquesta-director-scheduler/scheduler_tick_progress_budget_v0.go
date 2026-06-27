package orquestadirectorscheduler

import (
	orquestaagentprogress "orquesta/modulos/orquesta-agent-progress"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirector "orquesta/modulos/orquesta-director"
)

func schedulerProgressBudgetPreparedInputV0(
	input orquestadirector.AgentProgressSupervisionInputV0,
) orquestadirector.AgentProgressSupervisionInputV0 {
	switch input.Report.BudgetStatus {
	case orquestaagentprogress.AgentProgressBudgetOverBudgetButActiveV0:
		input.Report.Status = orquestaagentprogress.AgentStalledV0
		return schedulerProgressWithStopAllowedV0(input, false)
	case orquestaagentprogress.AgentProgressBudgetOverBudgetNoActivityV0:
		if schedulerProgressBudgetNoActivityCanStopV0(input) {
			input.Report.Status = orquestaagentprogress.AgentLoopDetectedV0
			return input
		}
		input.Report.Status = orquestaagentprogress.AgentStalledV0
		return schedulerProgressWithStopAllowedV0(input, false)
	}
	if schedulerProgressBudgetForcesDirectorReviewV0(input) {
		return schedulerProgressWithStopAllowedV0(input, false)
	}
	return input
}

func schedulerProgressBudgetForcesDirectorReviewV0(
	input orquestadirector.AgentProgressSupervisionInputV0,
) bool {
	return input.Report.Status == orquestaagentprogress.AgentLoopDetectedV0 &&
		!progressSupervisionStopAllowedV0(input)
}

func schedulerProgressBudgetStopAssessmentAllowedV0(
	input orquestadirector.AgentProgressSupervisionInputV0,
) bool {
	return progressSupervisionStopAllowedV0(input) &&
		input.PhaseID == string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0)
}

func schedulerProgressBudgetNoActivityCanStopV0(
	input orquestadirector.AgentProgressSupervisionInputV0,
) bool {
	return schedulerProgressBudgetStopAssessmentAllowedV0(input) &&
		(input.Report.NoProgressTicks > 0 || input.Report.RepeatedActionCount > 0)
}

func schedulerProgressWithStopAllowedV0(
	input orquestadirector.AgentProgressSupervisionInputV0,
	allowed bool,
) orquestadirector.AgentProgressSupervisionInputV0 {
	input.StopAllowed = &allowed
	return input
}
