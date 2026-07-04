package main

import (
	"fmt"
	"strconv"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestarunsupervisor "orquesta/modulos/orquesta-run-supervisor"
	orquestaserver "orquesta/modulos/orquesta-server"
)

func serverSupervisorCommandFromProjectConfigV0(
	executionMode string,
	projectConfig serverProjectConfigFileV0,
	maxExternalWaits int,
) orquestarunsupervisor.RunSupervisorCommandV0 {
	command := serverSupervisorCommandFromEnvV0(executionMode, maxExternalWaits)
	command.MaxRunsPerTick = codexExecutionModeCapPositiveV0(executionMode, intProjectConfigOrEnvOrDefaultV0(
		envServerMaxRunsPerTickV0,
		projectConfig.ServerSupervisor.MaxRunsPerTick,
		defaultCodexServerMaxRunsPerTickV0,
	))
	command.MaxExecutions = codexExecutionModeCapPositiveV0(executionMode, intProjectConfigOrEnvOrDefaultV0(
		envServerMaxExecutionsPerTickV0,
		projectConfig.ServerSupervisor.MaxExecutionsPerTick,
		defaultCodexServerMaxExecutionsV0,
	))
	command.DrainLimits.MaxDispatchesPerWait = codexExecutionModeCapPositiveV0(executionMode, intProjectConfigOrEnvOrDefaultV0(
		envServerDrainMaxDispatchesV0,
		projectConfig.ServerSupervisor.DrainMaxDispatches,
		defaultServerSupervisorMaxDispatchesV0,
	))
	command.DrainLimits.MaxCommands = codexExecutionModeCapPositiveV0(executionMode, intProjectConfigOrEnvOrDefaultV0(
		envServerDrainMaxCommandsV0,
		projectConfig.ServerSupervisor.DrainMaxCommands,
		20,
	))
	command.DrainLimits.MaxOutboxPerCycle = codexExecutionModeCapPositiveV0(executionMode, intProjectConfigOrEnvOrDefaultV0(
		envServerDrainMaxOutboxV0,
		projectConfig.ServerSupervisor.DrainMaxOutbox,
		defaultServerSupervisorMaxOutboxV0,
	))
	command.DrainLimits.MaxExternalWaits = maxExternalWaits
	return command
}

func serverRunQueueLimitFromProjectConfigV0(projectDir string) int {
	projectConfig := projectConfigFromProjectDirBestEffortV0(projectDir)
	return serverRunQueueLimitFromProjectConfigFileV0(projectConfig)
}

func serverRunQueueLimitFromProjectConfigFileV0(projectConfig serverProjectConfigFileV0) int {
	return intProjectConfigOrEnvOrDefaultV0(
		envServerQueueLimitV0,
		projectConfig.ServerSupervisor.QueueLimit,
		defaultCodexServerQueueLimitV0,
	)
}

func serverSupervisorMaxExternalWaitsFromProjectConfigV0(projectConfig serverProjectConfigFileV0) (int, error) {
	value := intProjectConfigOrEnvOrDefaultV0(
		envServerDrainMaxExternalWaitsV0,
		projectConfig.ServerSupervisor.DrainMaxExternalWaits,
		defaultServerSupervisorMaxExternalWaitsV0,
	)
	if value <= 0 {
		return 0, fmt.Errorf("%s incompatible: usar entero positivo hasta %d", envServerDrainMaxExternalWaitsV0, maxServerSupervisorMaxExternalWaitsV0)
	}
	if value > maxServerSupervisorMaxExternalWaitsV0 {
		return 0, fmt.Errorf("%s incompatible con supervisor residente: max=%d", envServerDrainMaxExternalWaitsV0, maxServerSupervisorMaxExternalWaitsV0)
	}
	return value, nil
}

func serverIdleSelfImprovementMaxRequestsFromProjectConfigFileV0(
	projectConfig serverProjectConfigFileV0,
) int {
	return intProjectConfigOrEnvOrDefaultV0(
		envServerIdleSelfImprovementMaxRequestsV0,
		firstIntPointerV0(projectConfig.ServerIdle.MaxRequests, projectConfig.ServerIdleLegacy.MaxRequests),
		orquestaserver.DefaultIdleSelfImprovementMaxRequestsV0,
	)
}

func serverResidentDirectorMaxActionsFromProjectConfigFileV0(
	projectConfig serverProjectConfigFileV0,
) int {
	return intProjectConfigOrEnvOrDefaultV0(
		envServerResidentDirectorMaxActionsV0,
		projectConfig.ServerResident.MaxActions,
		orquestaserver.DefaultResidentDirectorMaxActionsV0,
	)
}

func codexRuntimeLimitsFromProjectConfigFileV0(projectConfig serverProjectConfigFileV0) codexRuntimeLimitsV0 {
	executionMode := codexExecutionModeFromProjectConfigFileV0(projectConfig)
	return codexRuntimeLimitsV0{
		MaxBatchReady: codexExecutionModeCapPositiveV0(executionMode, intProjectConfigOrEnvOrDefaultV0(
			envCodexMaxBatchReadyV0,
			projectConfig.CodexRuntime.MaxBatchReady,
			defaultCodexMaxBatchReadyV0,
		)),
		MaxLiveProcesses: codexExecutionModeCapPositiveV0(executionMode, intProjectConfigOrEnvOrDefaultV0(
			envCodexMaxConcurrencyV0,
			projectConfig.CodexRuntime.MaxConcurrency,
			defaultCodexMaxConcurrencyV0,
		)),
	}
}

func codexExecutionModeFromProjectConfigFileV0(projectConfig serverProjectConfigFileV0) string {
	switch stringProjectConfigOrEnvOrDefaultV0(
		envCodexExecutionModeV0,
		projectConfig.CodexRuntime.ExecutionMode,
		defaultCodexExecutionModeV0,
	) {
	case codexExecutionModeSerialV0:
		return codexExecutionModeSerialV0
	case codexExecutionModeParallelV0, "":
		return codexExecutionModeParallelV0
	default:
		return defaultCodexExecutionModeV0
	}
}

func codexReasoningEffortFromProjectConfigFileV0(projectConfig serverProjectConfigFileV0) string {
	return codexReasoningEffortPolicyV0(stringProjectConfigOrEnvOrDefaultV0(
		envCodexReasoningEffortV0,
		projectConfig.CodexRuntime.ReasoningEffort,
		string(orquestacoreworkflow.OrchestrationCapacityMediumV0),
	))
}

func codexMaxExpectedSecondsFromProjectConfigFileV0(projectConfig serverProjectConfigFileV0) int {
	return intProjectConfigOrEnvOrDefaultV0(
		envCodexMaxExpectedSecondsV0,
		projectConfig.CodexRuntime.MaxExpectedSeconds,
		defaultCodexMaxExpectedSecondsV0,
	)
}

func codexStackCapacityEnvConfigFromProjectConfigV0(projectDir string) codexStackCapacityEnvConfigV0 {
	return codexStackCapacityEnvConfigFromProjectConfigFileV0(projectConfigFromProjectDirBestEffortV0(projectDir))
}

func codexStackCapacityEnvConfigFromProjectConfigFileV0(projectConfig serverProjectConfigFileV0) codexStackCapacityEnvConfigV0 {
	return codexStackCapacityEnvConfigV0{
		Tier: capacityRecommendationEnvOrDefaultV0(
			envCapacityTierV0,
			orquestacoreworkflow.OrchestrationCapacityMediumV0,
		),
		ReasoningEffort: capacityRecommendationEnvOrStringOrDefaultV0(
			envCapacityReasoningEffortV0,
			codexReasoningEffortFromProjectConfigFileV0(projectConfig),
			orquestacoreworkflow.OrchestrationCapacityMediumV0,
		),
		PolicyRef: envOrDefaultV0(envCapacityPolicyRefV0, "capacity-policy-ref-server-default-v0"),
		PoolRef:   envOrDefaultV0(envCapacityPoolRefV0, "capacity-pool-ref-server-default-v0"),
		ModelRef:  "capacity-model-ref-server-default-v0",
		QuotaRef:  "capacity-quota-ref-server-default-v0",
	}
}

func codexDirectorWaveLimitsFromProjectConfigV0(projectDir string) codexDirectorWaveLimitsEnvConfigV0 {
	return codexDirectorWaveLimitsFromProjectConfigFileV0(projectConfigFromProjectDirBestEffortV0(projectDir))
}

func codexDirectorWaveLimitsFromProjectConfigFileV0(projectConfig serverProjectConfigFileV0) codexDirectorWaveLimitsEnvConfigV0 {
	executionMode := codexExecutionModeFromProjectConfigFileV0(projectConfig)
	return codexDirectorWaveLimitsEnvConfigV0{
		Agents: codexExecutionModeCapPositiveV0(executionMode, intProjectConfigOrEnvOrDefaultV0(
			envCodexDirectorWaveAgentsV0,
			projectConfig.CodexDirector.WaveAgents,
			defaultCodexDirectorWaveAgentsV0,
		)),
		MaxSubagentsPerAgent: intProjectConfigOrEnvOrDefaultV0(
			envCodexDirectorMaxSubagentsPerAgentV0,
			projectConfig.CodexDirector.MaxSubagentsPerAgent,
			defaultCodexDirectorMaxSubagentsPerAgentV0,
		),
		RecursiveAgentBudget: intProjectConfigOrEnvOrDefaultV0(
			envCodexDirectorRecursiveAgentBudgetV0,
			projectConfig.CodexDirector.RecursiveAgentBudget,
			defaultCodexDirectorRecursiveAgentBudgetV0,
		),
	}
}

func capacityRecommendationEnvOrStringOrDefaultV0(
	key string,
	stringValue string,
	fallback orquestacoreworkflow.OrchestrationCapacityRecommendationV0,
) orquestacoreworkflow.OrchestrationCapacityRecommendationV0 {
	if value := capacityRecommendationEnvOrDefaultV0(key, ""); value != "" {
		return value
	}
	switch stringValue {
	case string(orquestacoreworkflow.OrchestrationCapacityLowV0):
		return orquestacoreworkflow.OrchestrationCapacityLowV0
	case string(orquestacoreworkflow.OrchestrationCapacityMediumV0):
		return orquestacoreworkflow.OrchestrationCapacityMediumV0
	case string(orquestacoreworkflow.OrchestrationCapacityHighV0):
		return orquestacoreworkflow.OrchestrationCapacityHighV0
	case string(orquestacoreworkflow.OrchestrationCapacityXHighV0):
		return orquestacoreworkflow.OrchestrationCapacityXHighV0
	default:
		return fallback
	}
}

func serverPositiveConfigSettingV0(projectDir string, key string, value int) orquestaserver.ServerConfigSettingV0 {
	return serverConfigSettingFromRegistryWithSourceV0(
		key,
		strconv.Itoa(value),
		configSettingSourceFromEnvOrProjectConfigV0(projectDir, key),
	)
}

func serverPositiveConfigSettingFromConfigV0(config orquestaserver.ConfigV0, key string, value int) orquestaserver.ServerConfigSettingV0 {
	return serverConfigSettingFromRegistryWithSourceV0(
		key,
		strconv.Itoa(value),
		configSettingSourceFromConfigOrProjectConfigV0(config, key),
	)
}
