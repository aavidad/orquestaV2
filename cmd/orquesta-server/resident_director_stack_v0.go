package main

import (
	"context"

	orquestaappcodexstack "orquesta/modulos/orquesta-app-codex-stack"
	orquestaopesdirector "orquesta/modulos/orquesta-opes-director"
	orquestaserver "orquesta/modulos/orquesta-server"
)

type serverResidentDirectorV0 struct {
	stack              *orquestaappcodexstack.StackV0
	serverConfig       orquestaserver.ConfigV0
	opesCausalProducer *serverOPESCausalProducerV0
}

func newServerResidentDirectorV0(
	stack *orquestaappcodexstack.StackV0,
	serverConfig orquestaserver.ConfigV0,
) orquestaserver.ResidentDirectorPortV0 {
	serverConfig = orquestaserver.NormalizeConfigV0(serverConfig)
	if stack == nil || !serverConfig.ResidentDirectorEnabled {
		return nil
	}
	return serverResidentDirectorV0{
		stack:              stack,
		serverConfig:       serverConfig,
		opesCausalProducer: newServerOPESCausalProducerV0(stack),
	}
}

func (director serverResidentDirectorV0) RunResidentDirectorV0(
	ctx context.Context,
	command orquestaserver.ResidentDirectorCommandV0,
) (orquestaserver.ResidentDirectorResultV0, error) {
	causalResult, err := director.runOPESCausalProducerV0(ctx, command)
	if err != nil {
		return orquestaserver.ResidentDirectorResultV0{
			Status:          causalResult.Status,
			ExecutedActions: len(causalResult.CreatedJobs),
			EvidenceRefs:    causalResult.EvidenceRefs,
		}, err
	}
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
		Status:          residentDirectorStatusWithOPESCausalProducerV0(result.Status, causalResult),
		RunRef:          result.RunRef,
		ExecutedActions: result.ExecutedActions + len(causalResult.CreatedJobs),
		EvidenceRefs:    compactStringsV0(append(result.EvidenceRefs, causalResult.EvidenceRefs...)),
	}
	if err != nil {
		return mapped, err
	}
	return mapped, nil
}

func (director serverResidentDirectorV0) runOPESCausalProducerV0(
	ctx context.Context,
	command orquestaserver.ResidentDirectorCommandV0,
) (orquestaopesdirector.OPESCausalProducerResultV0, error) {
	if director.opesCausalProducer == nil {
		return orquestaopesdirector.OPESCausalProducerResultV0{
			SchemaVersion: orquestaopesdirector.OPESCausalProducerResultSchemaV0,
			Status:        orquestaopesdirector.OPESCausalProducerStatusCompletedV0,
			EvidenceRefs:  compactStringsV0(command.EvidenceRefs),
		}, nil
	}
	maxActions := command.MaxActions
	if maxActions <= 0 {
		maxActions = director.serverConfig.ResidentDirectorMaxActions
	}
	return director.opesCausalProducer.ProduceV0(ctx, orquestaopesdirector.OPESCausalProducerRequestV0{
		DomainRef:     orquestaopesdirector.OPESCausalProducerDefaultDomainRefV0,
		CorrelationID: command.CorrelationID,
		MaxActions:    maxActions,
		EvidenceRefs:  command.EvidenceRefs,
	})
}

func residentDirectorStatusWithOPESCausalProducerV0(
	stackStatus string,
	causalResult orquestaopesdirector.OPESCausalProducerResultV0,
) string {
	if len(causalResult.CreatedJobs) == 0 {
		return stackStatus
	}
	if stackStatus == "" || stackStatus == orquestaappcodexstack.CodexStackResidentDirectorStatusIdleV0 {
		return "opes_causal_jobs_created"
	}
	return stackStatus
}
