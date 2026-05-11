package orquestadirectorsupervisedburst

import (
	"context"

	orquestadirectorcycle "orquesta/modulos/orquesta-director-cycle"
	orquestadirectorsupervisor "orquesta/modulos/orquesta-director-supervisor"
)

type DirectDirectorCycleStepExecutorV0 struct{}

func (DirectDirectorCycleStepExecutorV0) ExecuteDirectorCycleStepV0(
	ctx context.Context,
	input orquestadirectorcycle.DirectorCycleStepInputV0,
) (orquestadirectorcycle.DirectorCycleStepResultV0, error) {
	return orquestadirectorcycle.ExecuteDirectorCycleStepV0(ctx, input)
}

type DirectDirectorSupervisorPolicyV0 struct{}

func (DirectDirectorSupervisorPolicyV0) DecideDirectorSupervisorNextActionV0(
	input orquestadirectorsupervisor.DirectorSupervisorDecisionInputV0,
) (orquestadirectorsupervisor.DirectorSupervisorDecisionV0, error) {
	return orquestadirectorsupervisor.DecideDirectorSupervisorNextActionV0(input)
}

type DirectorCycleStepInputBuilderFuncV0 func(
	context.Context,
	DirectorSupervisedBurstStepRequestV0,
) (orquestadirectorcycle.DirectorCycleStepInputV0, error)

func (fn DirectorCycleStepInputBuilderFuncV0) BuildDirectorCycleStepInputV0(
	ctx context.Context,
	request DirectorSupervisedBurstStepRequestV0,
) (orquestadirectorcycle.DirectorCycleStepInputV0, error) {
	return fn(ctx, request)
}

type DirectorCycleStepExecutorFuncV0 func(
	context.Context,
	orquestadirectorcycle.DirectorCycleStepInputV0,
) (orquestadirectorcycle.DirectorCycleStepResultV0, error)

func (fn DirectorCycleStepExecutorFuncV0) ExecuteDirectorCycleStepV0(
	ctx context.Context,
	input orquestadirectorcycle.DirectorCycleStepInputV0,
) (orquestadirectorcycle.DirectorCycleStepResultV0, error) {
	return fn(ctx, input)
}

type DirectorSupervisorPolicyFuncV0 func(
	orquestadirectorsupervisor.DirectorSupervisorDecisionInputV0,
) (orquestadirectorsupervisor.DirectorSupervisorDecisionV0, error)

func (fn DirectorSupervisorPolicyFuncV0) DecideDirectorSupervisorNextActionV0(
	input orquestadirectorsupervisor.DirectorSupervisorDecisionInputV0,
) (orquestadirectorsupervisor.DirectorSupervisorDecisionV0, error) {
	return fn(input)
}
