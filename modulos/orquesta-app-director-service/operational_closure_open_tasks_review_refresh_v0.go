package orquestaappdirectorservice

import (
	"context"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectoroperativo "orquesta/modulos/orquesta-director-operativo"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
)

func continueOperationalDirectorPlanStateRefreshStaleClosureScopeV0(
	ctx context.Context,
	request ContinueAppDirectorRequestV0,
	ports StartAppDirectorPortsV0,
	state orquestacionnucleoapp.OperationalDirectorPlanStateV0,
) (orquestacionnucleoapp.OperationalDirectorPlanStateV0, bool, error) {
	if ports.RunStore == nil || ports.DirectorTaskStore == nil {
		return state, false, nil
	}
	activeStep, ok := operationalDirectorPlanStateActiveStepV0(state)
	if !ok ||
		state.Status != orquestacionnucleoapp.OperationalDirectorPlanStateActiveV0 ||
		activeStep.Kind != orquestadirectoroperativo.OperationalDirectorStepReplanOrCloseV0 ||
		activeStep.Status != orquestadirectoroperativo.OperationalDirectorStepRunningV0 {
		return state, false, nil
	}
	run, err := ports.RunStore.LoadRunV0(ctx, request.RunRef)
	if err != nil {
		return state, false, err
	}
	if run.Status != orquestacoreworkflow.OrchestrationRunStatusActiveV0 {
		return state, false, nil
	}
	scope, err := operationalDirectorPlanStateOpenAcceptedLeafTasksForClosureV0(
		ctx,
		request,
		ports,
		run,
		operationalDirectorPlanStateClosureScopeFilterFromStepV0(state, activeStep),
	)
	if err != nil || len(scope.TaskRefs) == 0 {
		return state, false, err
	}
	if operationalDirectorPlanStateSameRefsV0(activeStep.TaskRefs, scope.TaskRefs) &&
		operationalDirectorPlanStateSameRefsV0(activeStep.AgentRefs, scope.AgentRefs) &&
		operationalDirectorPlanStateSameRefsV0(activeStep.DeliveryRefs, scope.DeliveryRefs) &&
		operationalDirectorPlanStateSameRefsV0(activeStep.ReviewResultRefs, scope.ReviewResultRefs) {
		return state, false, nil
	}
	nextSteps := make([]orquestacionnucleoapp.OperationalDirectorPlanStepStateV0, 0, len(state.Steps))
	for _, step := range state.Steps {
		nextStep := step
		if step.StepID == activeStep.StepID {
			nextStep.TaskRefs = append([]string(nil), scope.TaskRefs...)
			nextStep.AgentRefs = append([]string(nil), scope.AgentRefs...)
			nextStep.DeliveryRefs = append([]string(nil), scope.DeliveryRefs...)
			nextStep.ReviewResultRefs = append([]string(nil), scope.ReviewResultRefs...)
			nextStep.BlockerRefs = nil
			nextStep.Reason = "closure-scope-refreshed-from-open-reviewed-tasks"
			nextStep.EvidenceRefs = compactServiceRefsV0(append(
				nextStep.EvidenceRefs,
				"evidence-ref-app-director-closure-scope-refreshed-v0",
			))
		}
		nextSteps = append(nextSteps, nextStep)
	}
	state.PendingAgentRefs = nil
	state.BlockerRefs = nil
	state.ClosureReason = ""
	state.Steps = nextSteps
	state.EvidenceRefs = compactServiceRefsV0(append(
		state.EvidenceRefs,
		"evidence-ref-app-director-closure-scope-refreshed-v0",
	))
	state.UpdatedAt = request.OccurredAt
	next, err := orquestacionnucleoapp.NewOperationalDirectorPlanStateV0(state)
	if err != nil {
		return state, false, err
	}
	return next, true, nil
}
