package orquestamcp

import (
	"context"

	orquestaexternalworkrun "orquesta/modulos/orquesta-external-work-run"
)

type MCPExternalWorkRunToolExecutorV0 struct {
	Ports  orquestaexternalworkrun.StartExternalWorkRunPortsV0
	Config orquestaexternalworkrun.StartExternalWorkRunConfigV0
}

func NewMCPExternalWorkRunToolExecutorV0(
	ports orquestaexternalworkrun.StartExternalWorkRunPortsV0,
	config orquestaexternalworkrun.StartExternalWorkRunConfigV0,
) MCPExternalWorkRunToolExecutorV0 {
	return MCPExternalWorkRunToolExecutorV0{Ports: ports, Config: config}
}

func (executor MCPExternalWorkRunToolExecutorV0) Execute(
	ctx context.Context,
	input MCPExternalWorkRunToolInputV0,
) (MCPExternalWorkRunToolResultV0, error) {
	if issues := validateExternalWorkRunMCPInputV0(input); len(issues) > 0 {
		return newMCPExternalWorkRunInputErrorV0(input, issues), nil
	}
	result, err := orquestaexternalworkrun.StartExternalWorkRunV0(
		ctx,
		externalWorkRunRequestFromMCPV0(input),
		executor.Ports,
		executor.Config,
	)
	if err != nil {
		return MCPExternalWorkRunToolResultV0{}, err
	}
	return newMCPExternalWorkRunResultV0(result), nil
}
