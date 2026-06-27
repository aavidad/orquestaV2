package main

import (
	orquestaruncoordinator "orquesta/modulos/orquesta-run-coordinator"
	orquestarunsupervisor "orquesta/modulos/orquesta-run-supervisor"
	orquestaserver "orquesta/modulos/orquesta-server"
)

func serverSupervisorCommandFromEnvV0(
	executionMode string,
	maxExternalWaits int,
) orquestarunsupervisor.RunSupervisorCommandV0 {
	return orquestarunsupervisor.RunSupervisorCommandV0{
		QueueRef:          "global",
		MaxRunsPerTick:    codexExecutionModeCapIntEnvOrDefaultV0(executionMode, envServerMaxRunsPerTickV0, defaultCodexServerMaxRunsPerTickV0),
		MaxTicks:          intEnvOrDefaultV0(envServerSupervisorMaxTicksV0, orquestaserver.DefaultSupervisorMaxTicksV0),
		MaxExecutions:     codexExecutionModeCapIntEnvOrDefaultV0(executionMode, envServerMaxExecutionsPerTickV0, defaultCodexServerMaxExecutionsV0),
		StopOnNoExecution: true,
		AllowRepeatedRuns: boolEnvOrDefaultV0(envServerAllowRepeatedRunsV0, false),
		AllowLegacyDrain: boolEnvOrDefaultV0(envAutoprogrammingLegacyDirectorLoopV0, false) ||
			boolEnvOrDefaultV0(envExternalWorkLegacyDirectorLoopV0, false),
		DrainLimits: orquestaruncoordinator.RunDrainLimitsV0{
			MaxBursts:            intEnvOrDefaultV0(envServerDrainMaxBurstsV0, 4),
			MaxStepsPerBurst:     intEnvOrDefaultV0(envServerDrainMaxStepsV0, 6),
			MaxDispatchesPerWait: codexExecutionModeCapIntEnvOrDefaultV0(executionMode, envServerDrainMaxDispatchesV0, defaultServerSupervisorMaxDispatchesV0),
			MaxCommands:          codexExecutionModeCapIntEnvOrDefaultV0(executionMode, envServerDrainMaxCommandsV0, 20),
			MaxOutboxPerCycle:    codexExecutionModeCapIntEnvOrDefaultV0(executionMode, envServerDrainMaxOutboxV0, defaultServerSupervisorMaxOutboxV0),
			MaxDecisionCycles:    intEnvOrDefaultV0(envServerDrainMaxDecisionsV0, 1),
			MaxExternalWaits:     maxExternalWaits,
		},
	}
}
