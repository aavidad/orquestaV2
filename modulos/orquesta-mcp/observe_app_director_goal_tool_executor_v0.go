package orquestamcp

import (
	"context"
	"strings"

	orquestaappdirectorservice "orquesta/modulos/orquesta-app-director-service"
	orquestaestadovivo "orquesta/modulos/orquesta-estado-vivo"
)

type MCPObserveAppDirectorGoalToolExecutorV0 struct {
	Ports            orquestaappdirectorservice.StartAppDirectorPortsV0
	EstadoVivoSource orquestaestadovivo.FuenteEvidenciaEstadoPortV0
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
		if publicResult, ok := NewMCPObserveAppDirectorGoalErrorResultFromErrorV0(input, err); ok {
			return executor.withPartialSnapshotAfterObserveErrorV0(ctx, input, publicResult), nil
		}
		return MCPObserveAppDirectorGoalToolResultV0{}, err
	}
	return NewMCPObserveAppDirectorGoalResultV0(input, result), nil
}

func (executor MCPObserveAppDirectorGoalToolExecutorV0) withPartialSnapshotAfterObserveErrorV0(
	ctx context.Context,
	input MCPObserveAppDirectorGoalToolInputV0,
	publicResult MCPObserveAppDirectorGoalToolResultV0,
) MCPObserveAppDirectorGoalToolResultV0 {
	snapshot, err := executor.ObserveAppDirectorGoalTimeoutSnapshotV0(ctx, input)
	if err != nil {
		return publicResult
	}
	return NewMCPObserveAppDirectorGoalTimeoutResultWithPartialV0(publicResult, snapshot)
}

func (executor MCPObserveAppDirectorGoalToolExecutorV0) ObserveAppDirectorGoalTimeoutSnapshotV0(
	ctx context.Context,
	input MCPObserveAppDirectorGoalToolInputV0,
) (MCPObserveAppDirectorGoalToolResultV0, error) {
	if strings.TrimSpace(input.RunRef) == "" {
		return MCPObserveAppDirectorGoalToolResultV0{}, orquestaappdirectorservice.AppDirectorServiceIssueV0{Field: "run_ref"}
	}
	if executor.Ports.GoalStateStore == nil {
		return MCPObserveAppDirectorGoalToolResultV0{}, orquestaappdirectorservice.AppDirectorServiceIssueV0{Field: "ports.goal_state_store"}
	}
	state, err := executor.Ports.GoalStateStore.LoadGoalWorkStateV0(ctx, strings.TrimSpace(input.RunRef))
	if err != nil {
		return MCPObserveAppDirectorGoalToolResultV0{}, err
	}
	result, err := NewMCPObserveAppDirectorGoalPartialResultFromStateV0(input, state)
	if err != nil {
		return MCPObserveAppDirectorGoalToolResultV0{}, err
	}
	if executor.Ports.RunStore != nil {
		if run, loadErr := executor.Ports.RunStore.LoadRunV0(ctx, strings.TrimSpace(input.RunRef)); loadErr == nil {
			result.RunStatus = strings.TrimSpace(string(run.Status))
			result.CurrentPhase = strings.TrimSpace(string(run.CurrentPhase))
		}
	}
	if executor.Ports.EventReader != nil {
		if events, loadErr := executor.Ports.EventReader.LoadRunEventsV0(ctx, strings.TrimSpace(input.RunRef)); loadErr == nil {
			for _, event := range events {
				if occurredAt := strings.TrimSpace(event.OccurredAt); occurredAt != "" {
					result.LastEventAt = occurredAt
				}
			}
		}
	}
	result = executor.withEstadoVivoSnapshotV0(ctx, input, result)
	return result, nil
}
