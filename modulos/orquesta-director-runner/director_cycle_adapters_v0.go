package orquestadirectorrunner

import (
	"context"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectorscheduler "orquesta/modulos/orquesta-director-scheduler"
)

type DirectDirectorSchedulerPortV0 struct{}

func (DirectDirectorSchedulerPortV0) BuildDirectorSchedulerTickV0(
	ctx context.Context,
	input orquestadirectorscheduler.DirectorSchedulerTickInputV0,
) (orquestadirectorscheduler.DirectorSchedulerTickPlanV0, error) {
	if err := ctx.Err(); err != nil {
		return orquestadirectorscheduler.DirectorSchedulerTickPlanV0{}, err
	}
	return orquestadirectorscheduler.BuildDirectorSchedulerTickV0(input)
}

type DirectorSchedulerFuncV0 func(
	context.Context,
	orquestadirectorscheduler.DirectorSchedulerTickInputV0,
) (orquestadirectorscheduler.DirectorSchedulerTickPlanV0, error)

func (fn DirectorSchedulerFuncV0) BuildDirectorSchedulerTickV0(
	ctx context.Context,
	input orquestadirectorscheduler.DirectorSchedulerTickInputV0,
) (orquestadirectorscheduler.DirectorSchedulerTickPlanV0, error) {
	return fn(ctx, input)
}

type WorkflowCommandFuncV0 func(
	context.Context,
	orquestacoreworkflow.OrchestrationCommandV0,
) (orquestacoreworkflow.OrchestrationCommandResultV0, error)

func (fn WorkflowCommandFuncV0) HandleWorkflowCommandV0(
	ctx context.Context,
	command orquestacoreworkflow.OrchestrationCommandV0,
) (orquestacoreworkflow.OrchestrationCommandResultV0, error) {
	return fn(ctx, command)
}
