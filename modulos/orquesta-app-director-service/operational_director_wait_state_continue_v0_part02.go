package orquestaappdirectorservice

import (
	orquestadirectoroperativo "orquesta/modulos/orquesta-director-operativo"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	"strings"
)

func continueRequestWithOperationalDirectorPlanStepScopeV0(
	request ContinueAppDirectorRequestV0,
	state orquestacionnucleoapp.OperationalDirectorPlanStateV0,
	step orquestacionnucleoapp.OperationalDirectorPlanStepStateV0,
) ContinueAppDirectorRequestV0 {
	if strings.TrimSpace(request.WaitWaveRef) == "" {
		request.WaitWaveRef = state.ActiveWaveRef
		if request.WaitWaveRef == "" {
			request.WaitWaveRef = step.WaveRef
		}
	}
	if strings.TrimSpace(request.WaitCohortRef) == "" {
		request.WaitCohortRef = state.ActiveCohortRef
		if request.WaitCohortRef == "" {
			request.WaitCohortRef = step.CohortRef
		}
	}
	if strings.TrimSpace(request.WaitParentTaskRef) == "" {
		request.WaitParentTaskRef = state.ActiveParentTaskRef
		if request.WaitParentTaskRef == "" {
			request.WaitParentTaskRef = step.ParentTaskRef
		}
	}
	return request
}

func operationalDirectorPlanStateActiveStepV0(
	state orquestacionnucleoapp.OperationalDirectorPlanStateV0,
) (orquestacionnucleoapp.OperationalDirectorPlanStepStateV0, bool) {
	activeStepID := strings.TrimSpace(state.ActiveStepID)
	for _, step := range state.Steps {
		if activeStepID != "" && strings.TrimSpace(step.StepID) != activeStepID {
			continue
		}
		if activeStepID == "" && step.Status != orquestadirectoroperativo.OperationalDirectorStepRunningV0 {
			continue
		}
		return step, true
	}
	return orquestacionnucleoapp.OperationalDirectorPlanStepStateV0{}, false
}

func continueRequestHasWaitScopeV0(request ContinueAppDirectorRequestV0) bool {
	return len(request.WaitAgentRefs) > 0 ||
		strings.TrimSpace(request.WaitCohortRef) != "" ||
		strings.TrimSpace(request.WaitWaveRef) != "" ||
		strings.TrimSpace(request.WaitParentTaskRef) != ""
}

func continueRequestWithoutWaitScopeV0(request ContinueAppDirectorRequestV0) ContinueAppDirectorRequestV0 {
	request.WaitAgentRefs = nil
	request.WaitCohortRef = ""
	request.WaitWaveRef = ""
	request.WaitParentTaskRef = ""
	return request
}

func continueOperationalDirectorPlanRefV0(request ContinueAppDirectorRequestV0) string {
	if value := strings.TrimSpace(request.OperationalDirectorPlanRef); value != "" {
		return value
	}
	return strings.TrimSpace(request.OperationalDirectorPlan.PlanRef)
}
