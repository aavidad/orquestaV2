package orquestadirectorscheduler

import (
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirector "orquesta/modulos/orquesta-director"
	orquestaruntime "orquesta/modulos/orquesta-runtime"
)

func schedulerProgressBudgetPreparedInputV0(
	input orquestadirector.AgentProgressSupervisionInputV0,
) orquestadirector.AgentProgressSupervisionInputV0 {
	switch input.Report.BudgetStatus {
	case orquestaruntime.AgentProgressBudgetOverBudgetButActiveV0:
		input.Report.Status = orquestaruntime.AgentStalledV0
		return schedulerProgressWithStopAllowedV0(input, false)
	case orquestaruntime.AgentProgressBudgetOverBudgetNoActivityV0:
		if schedulerProgressBudgetNoActivityCanStopV0(input) {
			input.Report.Status = orquestaruntime.AgentLoopDetectedV0
			return input
		}
		input.Report.Status = orquestaruntime.AgentStalledV0
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
	return input.Report.Status == orquestaruntime.AgentLoopDetectedV0 &&
		!schedulerProgressBudgetStopAssessmentAllowedV0(input)
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

func schedulerProgressBudgetNeedsDirectorQuestionV0(
	input orquestadirector.AgentProgressSupervisionInputV0,
) bool {
	return input.Report.Status == orquestaruntime.AgentStalledV0 ||
		schedulerProgressBudgetForcesDirectorReviewV0(input)
}

func schedulerProgressWithStopAllowedV0(
	input orquestadirector.AgentProgressSupervisionInputV0,
	allowed bool,
) orquestadirector.AgentProgressSupervisionInputV0 {
	input.StopAllowed = &allowed
	return input
}
