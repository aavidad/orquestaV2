package main

import (
	"context"

	orquestaappcodexstack "orquesta/modulos/orquesta-app-codex-stack"
	orquestaserver "orquesta/modulos/orquesta-server"
)

type serverResidentDirectorV0 struct {
	stack        *orquestaappcodexstack.StackV0
	serverConfig orquestaserver.ConfigV0
}

func newServerResidentDirectorV0(
	stack *orquestaappcodexstack.StackV0,
	serverConfig orquestaserver.ConfigV0,
) orquestaserver.ResidentDirectorPortV0 {
	serverConfig = orquestaserver.NormalizeConfigV0(serverConfig)
	if stack == nil || !serverConfig.ResidentDirectorEnabled {
		return nil
	}
	return serverResidentDirectorV0{stack: stack, serverConfig: serverConfig}
}

func (director serverResidentDirectorV0) RunResidentDirectorV0(
	ctx context.Context,
	command orquestaserver.ResidentDirectorCommandV0,
) (orquestaserver.ResidentDirectorResultV0, error) {
	result, err := director.stack.RunCodexStackResidentDirectorV0(
		ctx,
		orquestaappcodexstack.CodexStackResidentDirectorCommandV0{
			OccurredAt:           command.OccurredAt,
			CorrelationID:        command.CorrelationID,
			EvidenceRefs:         command.EvidenceRefs,
			MaxRunsPerTick:       director.serverConfig.SupervisorCommand.MaxRunsPerTick,
			MaxExecutions:        director.serverConfig.SupervisorCommand.MaxExecutions,
			MaxActions:           command.MaxActions,
			MaxBursts:            director.serverConfig.SupervisorCommand.DrainLimits.MaxBursts,
			MaxStepsPerBurst:     director.serverConfig.SupervisorCommand.DrainLimits.MaxStepsPerBurst,
			MaxDispatchesPerWait: director.serverConfig.SupervisorCommand.DrainLimits.MaxDispatchesPerWait,
			MaxCommands:          director.serverConfig.SupervisorCommand.DrainLimits.MaxCommands,
			MaxOutboxPerCycle:    director.serverConfig.SupervisorCommand.DrainLimits.MaxOutboxPerCycle,
			MaxDecisionCycles:    director.serverConfig.SupervisorCommand.DrainLimits.MaxDecisionCycles,
			MaxExternalWaits:     director.serverConfig.SupervisorCommand.DrainLimits.MaxExternalWaits,
		},
	)
	mapped := orquestaserver.ResidentDirectorResultV0{
		Status:          result.Status,
		RunRef:          result.RunRef,
		ExecutedActions: result.ExecutedActions,
		EvidenceRefs:    result.EvidenceRefs,
	}
	if err != nil {
		return mapped, err
	}
	return mapped, nil
}
