package orquestamcp

import (
	"context"
	"strings"

	orquestaappdirectorservice "orquesta/modulos/orquesta-app-director-service"
)

type MCPObserveAppDirectorGoalToolExecutorV0 struct {
	Ports orquestaappdirectorservice.StartAppDirectorPortsV0
}

func NewMCPObserveAppDirectorGoalToolExecutorV0(
	ports orquestaappdirectorservice.StartAppDirectorPortsV0,
) MCPObserveAppDirectorGoalToolExecutorV0 {
	return MCPObserveAppDirectorGoalToolExecutorV0{Ports: ports}
}

func (executor MCPObserveAppDirectorGoalToolExecutorV0) Execute(
	ctx context.Context,
	input MCPObserveAppDirectorGoalToolInputV0,
) (MCPObserveAppDirectorGoalToolResultV0, error) {
	if strings.TrimSpace(input.RunRef) == "" {
		return NewMCPObserveAppDirectorGoalErrorResultV0(
			input,
			"observe_app_director_goal_input_invalid",
			"run_ref",
			"run_ref_requerido",
		), nil
	}
	result, err := orquestaappdirectorservice.ObserveAppDirectorGoalV0(
		ctx,
		ToObserveAppDirectorGoalRequestV0(input),
		executor.Ports,
	)
	if err != nil {
		return MCPObserveAppDirectorGoalToolResultV0{}, err
	}
	return NewMCPObserveAppDirectorGoalResultV0(input, result), nil
}
