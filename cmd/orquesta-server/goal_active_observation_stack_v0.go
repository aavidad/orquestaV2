package main

import (
	"context"
	"fmt"

	orquestagoal "orquesta/modulos/orquesta-goal"
)

func (supervisor serverStackSupervisorV0) ObserveActiveGoalWorksV0(
	ctx context.Context,
	request orquestagoal.GoalWorkObserveActiveRequestV0,
) (orquestagoal.GoalWorkObserveActiveResultV0, error) {
	if supervisor.stack == nil {
		return orquestagoal.GoalWorkObserveActiveResultV0{}, fmt.Errorf("stack requerido")
	}
	return supervisor.stack.ObserveActiveGoalWorksV0(ctx, request)
}
