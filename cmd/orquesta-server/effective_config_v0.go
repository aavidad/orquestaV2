package main

import (
	"strconv"
	"time"

	orquestaserver "orquesta/modulos/orquesta-server"
)

func serverEffectiveConfigFromEnvV0(config orquestaserver.ConfigV0) orquestaserver.ServerEffectiveConfigV0 {
	config = orquestaserver.NormalizeConfigV0(config)
	codexRuntime := codexRuntimeEnvConfigFromEnvV0()
	stackCapacity := codexStackCapacityEnvConfigFromEnvV0()
	directorWaveLimits := codexDirectorWaveLimitsEnvConfigFromEnvV0()
	return orquestaserver.NormalizeServerEffectiveConfigV0(orquestaserver.ServerEffectiveConfigV0{
		SchemaVersion: orquestaserver.ServerEffectiveConfigSchemaVersionV0,
		Settings: []orquestaserver.ServerConfigSettingV0{
			serverConfigSettingFromRegistryV0(envServerMaxRunsPerTickV0, strconv.Itoa(config.SupervisorCommand.MaxRunsPerTick)),
			serverConfigSettingFromRegistryV0(envServerMaxExecutionsPerTickV0, strconv.Itoa(config.SupervisorCommand.MaxExecutions)),
			serverConfigSettingFromRegistryV0(envServerDrainMaxExternalWaitsV0, strconv.Itoa(config.SupervisorCommand.DrainLimits.MaxExternalWaits)),
			serverConfigSettingFromRegistryV0(envServerTickIntervalMSV0, strconv.Itoa(int(config.TickInterval/time.Millisecond))),
			serverConfigSettingFromRegistryV0(envServerIdleSelfImprovementTargetQueueV0, strconv.Itoa(config.IdleSelfImprovementTargetQueue)),
			serverConfigSettingFromRegistryV0(envServerIdleSelfImprovementMaxRequestsV0, strconv.Itoa(config.IdleSelfImprovementMaxRequests)),
			serverConfigSettingFromRegistryV0(envCodexMaxBatchReadyV0, strconv.Itoa(codexRuntime.Limits.MaxBatchReady)),
			serverConfigSettingFromRegistryV0(envCodexMaxConcurrencyV0, strconv.Itoa(codexRuntime.Limits.MaxLiveProcesses)),
			serverConfigSettingFromRegistryV0(envCodexReasoningEffortV0, codexRuntime.ReasoningEffort),
			serverConfigSettingFromRegistryV0(envCapacityReasoningEffortV0, string(stackCapacity.ReasoningEffort)),
			serverConfigSettingFromRegistryV0(envCodexDirectorWaveAgentsV0, strconv.Itoa(directorWaveLimits.Agents)),
			serverConfigSettingFromRegistryV0(envCodexDirectorMaxSubagentsPerAgentV0, strconv.Itoa(directorWaveLimits.MaxSubagentsPerAgent)),
			serverConfigSettingFromRegistryV0(envCodexDirectorRecursiveAgentBudgetV0, strconv.Itoa(directorWaveLimits.RecursiveAgentBudget)),
		},
	})
}

func serverConfigSettingV0(
	key string,
	value string,
	scope string,
	label string,
	description string,
) orquestaserver.ServerConfigSettingV0 {
	return orquestaserver.ServerConfigSettingV0{
		Key:             key,
		Value:           value,
		Scope:           scope,
		Label:           label,
		Description:     description,
		RestartBehavior: "restart_required",
		Editable:        true,
		Canonical:       true,
	}
}
