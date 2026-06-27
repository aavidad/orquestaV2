package orquestamcp

import (
	"context"
	"strings"

	orquestaappdirectorservice "orquesta/modulos/orquesta-app-director-service"
	orquestaapprunner "orquesta/modulos/orquesta-app-runner"
)

type MCPEjecutarOrquestacionAppToolExecutorV0 struct {
	Ports orquestaapprunner.RunPreparedAppOrchestrationPortsV0
}

func NewMCPEjecutarOrquestacionAppToolExecutorV0(
	ports orquestaapprunner.RunPreparedAppOrchestrationPortsV0,
) MCPEjecutarOrquestacionAppToolExecutorV0 {
	return MCPEjecutarOrquestacionAppToolExecutorV0{Ports: ports}
}

func (executor MCPEjecutarOrquestacionAppToolExecutorV0) Execute(
	ctx context.Context,
	input MCPEjecutarOrquestacionAppToolInputV0,
) (MCPEjecutarOrquestacionAppToolResultV0, error) {
	if strings.TrimSpace(input.DirectorExecutionMode) != orquestaappdirectorservice.AppDirectorExecutionModeLegacyDirectorLoopV0 {
		return NewMCPEjecutarOrquestacionAppLegacyModeRequiredResultV0(input), nil
	}
	prepared, err := orquestaapprunner.PrepareAppOrchestrationV0(
		ToPrepareAppOrchestrationRequestMCPV0(prepareInputFromRunInputMCPV0(input)),
	)
	if err != nil {
		return NewMCPEjecutarOrquestacionAppErrorResultV0(input, err), nil
	}
	result, err := orquestaapprunner.RunPreparedAppOrchestrationV0(
		ctx,
		ToRunPreparedAppOrchestrationRequestMCPV0(input, prepared),
		executor.Ports,
	)
	if err != nil {
		return NewMCPEjecutarOrquestacionAppErrorResultV0(input, err), nil
	}
	return NewMCPEjecutarOrquestacionAppOKResultV0(input, prepared, result), nil
}

func prepareInputFromRunInputMCPV0(
	input MCPEjecutarOrquestacionAppToolInputV0,
) MCPPrepararOrquestacionAppToolInputV0 {
	return MCPPrepararOrquestacionAppToolInputV0{
		RequestID:     input.RequestID,
		CorrelationID: input.CorrelationID,
		RunRef:        input.RunRef,
		ProjectRef:    input.ProjectRef,
		OccurredAt:    input.OccurredAt,
		RequestedBy:   input.RequestedBy,
		AppSpec:       input.AppSpec,
	}
}
