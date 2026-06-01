package orquestaappdirectorservice

import (
	"context"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectoroperativo "orquesta/modulos/orquesta-director-operativo"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
)

const operationalClosureOpenTasksNoProgressReasonV0 = "operational-closure-open-tasks-no-progress"

func operationalDirectorPlanStateReopenReviewForClosureOpenTasksNoProgressV0(
	ctx context.Context,
	request ContinueAppDirectorRequestV0,
	ports StartAppDirectorPortsV0,
	run orquestacoreworkflow.OrchestrationRunV0,
) (bool, error) {
	if ports.OperationalPlanStateStore == nil || ports.OperationalPlanStateWriter == nil {
		return false, nil
	}
	planRef := continueOperationalDirectorPlanRefV0(request)
	if planRef == "" {
		return false, nil
	}
	state, err := ports.OperationalPlanStateStore.LoadOperationalDirectorPlanStateV0(ctx, request.RunRef, planRef)
	if err != nil {
		return false, err
	}
	next, reopened, err := operationalDirectorPlanStateReopenReviewForOpenDeliveredTasksV0(
		ctx,
		request,
		ports,
		state,
		run,
		operationalClosureOpenTasksNoProgressReasonV0,
	)
	if err != nil || !reopened {
		return reopened, err
	}
	return true, ports.OperationalPlanStateWriter.SaveOperationalDirectorPlanStateV0(ctx, next)
}

func continueOperationalDirectorPlanStateAfterClosureOpenTasksNoProgressV0(
	ctx context.Context,
	request ContinueAppDirectorRequestV0,
	ports StartAppDirectorPortsV0,
	state orquestacionnucleoapp.OperationalDirectorPlanStateV0,
) (orquestacionnucleoapp.OperationalDirectorPlanStateV0, bool, error) {
	if ports.RunStore == nil {
		return state, false, nil
	}
	activeStep, ok := operationalDirectorPlanStateActiveStepV0(state)
	if !ok ||
		state.Status != orquestacionnucleoapp.OperationalDirectorPlanStateBlockedV0 ||
		activeStep.Kind != orquestadirectoroperativo.OperationalDirectorStepReplanOrCloseV0 ||
		activeStep.Status != orquestadirectoroperativo.OperationalDirectorStepBlockedV0 ||
		!operationalDirectorPlanStateHasReasonOrBlockerV0(state, activeStep, operationalClosureOpenTasksNoProgressReasonV0) {
		return state, false, nil
	}
	run, err := ports.RunStore.LoadRunV0(ctx, request.RunRef)
	if err != nil {
		return state, false, err
	}
	return operationalDirectorPlanStateReopenReviewForOpenDeliveredTasksV0(
		ctx,
		request,
		ports,
		state,
		run,
		operationalClosureOpenTasksNoProgressReasonV0,
	)
}
