package main

import (
	"strconv"
	"strings"
	"time"

	orquestaserver "orquesta/modulos/orquesta-server"
)

const (
	codebaseBrokerConfiguredStateDirRefV0 = "codebase-broker-state-dir-configured"
	codebaseBrokerConfiguredCommandRefV0  = "codebase-broker-command-configured"
)

func codebaseBrokerProjectConfigFromProjectDirV0(projectDir string) serverProjectConfigCodebaseBrokerV0 {
	return projectConfigFromProjectDirBestEffortV0(projectDir).CodebaseBroker
}

func codebaseBrokerProviderKindFromProjectConfigFileV0(config serverProjectConfigFileV0) string {
	return stringProjectConfigOrEnvOrDefaultV0(
		envCodebaseBrokerProviderKindV0,
		config.CodebaseBroker.ProviderKind,
		"fallback_rg",
	)
}

func codebaseBrokerExternalIndexerEnabledFromProjectConfigFileV0(config serverProjectConfigFileV0) bool {
	return boolProjectConfigOrEnvOrDefaultV0(
		envCodebaseBrokerExternalIndexerEnabledV0,
		config.CodebaseBroker.ExternalIndexerEnabled,
		false,
	)
}

func codebaseBrokerMaxConcurrentFromProjectConfigFileV0(config serverProjectConfigFileV0) int {
	return intProjectConfigOrEnvOrDefaultV0(
		envCodebaseBrokerMaxConcurrentV0,
		config.CodebaseBroker.MaxConcurrent,
		4,
	)
}

func codebaseBrokerTimeoutFromProjectConfigFileV0(config serverProjectConfigFileV0) time.Duration {
	return time.Duration(intProjectConfigOrEnvOrDefaultV0(
		envCodebaseBrokerTimeoutMSV0,
		config.CodebaseBroker.TimeoutMS,
		3000,
	)) * time.Millisecond
}

func codebaseBrokerStateDirFromProjectConfigFileV0(config serverProjectConfigFileV0) string {
	return strings.TrimSpace(stringProjectConfigOrEnvOrDefaultV0(
		envCodebaseBrokerStateDirV0,
		config.CodebaseBroker.StateDir,
		"",
	))
}

func codebaseBrokerCommandFromProjectConfigFileV0(config serverProjectConfigFileV0) string {
	return stringProjectConfigOrEnvOrDefaultV0(
		envCodebaseBrokerCommandV0,
		config.CodebaseBroker.Command,
		"codebase-memory-mcp",
	)
}

func codebaseBrokerProjectNameFromProjectConfigFileV0(config serverProjectConfigFileV0) string {
	return stringProjectConfigOrEnvOrDefaultV0(
		envCodebaseBrokerProjectNameV0,
		config.CodebaseBroker.ProjectName,
		"",
	)
}

func codebaseBrokerWatchdogEnabledFromProjectConfigFileV0(config serverProjectConfigFileV0) bool {
	return boolProjectConfigOrEnvOrDefaultV0(
		envCodebaseBrokerWatchdogEnabledV0,
		config.CodebaseBroker.WatchdogEnabled,
		false,
	)
}

func codebaseBrokerWatchdogStopOrphansFromProjectConfigFileV0(config serverProjectConfigFileV0) bool {
	return boolProjectConfigOrEnvOrDefaultV0(
		envCodebaseBrokerWatchdogStopOrphansV0,
		config.CodebaseBroker.WatchdogStopOrphans,
		false,
	)
}

func codebaseBrokerWatchdogOrphanMinAgeSecondsFromProjectConfigFileV0(config serverProjectConfigFileV0) int {
	value := intProjectConfigOrEnvOrDefaultV0(
		envCodebaseBrokerWatchdogOrphanMinAgeSecondsV0,
		config.CodebaseBroker.WatchdogOrphanMinAgeSeconds,
		int(defaultCodeContextToolOrphanMinAgeV0/time.Second),
	)
	if value < 0 {
		return int(defaultCodeContextToolOrphanMinAgeV0 / time.Second)
	}
	return value
}

func codebaseBrokerEffectiveSettingsV0(
	projectDir string,
	config serverProjectConfigFileV0,
) []orquestaserver.ServerConfigSettingV0 {
	stateDirSource := configSettingSourceFromEnvOrProjectConfigV0(projectDir, envCodebaseBrokerStateDirV0)
	commandSource := configSettingSourceFromEnvOrProjectConfigV0(projectDir, envCodebaseBrokerCommandV0)
	return []orquestaserver.ServerConfigSettingV0{
		serverConfigSettingFromRegistryWithSourceV0(
			envCodebaseBrokerProviderKindV0,
			codebaseBrokerProviderKindFromProjectConfigFileV0(config),
			configSettingSourceFromEnvOrProjectConfigV0(projectDir, envCodebaseBrokerProviderKindV0),
		),
		serverConfigSettingFromRegistryWithSourceV0(
			envCodebaseBrokerExternalIndexerEnabledV0,
			strconv.FormatBool(codebaseBrokerExternalIndexerEnabledFromProjectConfigFileV0(config)),
			configSettingSourceFromEnvOrProjectConfigV0(projectDir, envCodebaseBrokerExternalIndexerEnabledV0),
		),
		serverConfigSettingFromRegistryWithSourceV0(
			envCodebaseBrokerMaxConcurrentV0,
			strconv.Itoa(codebaseBrokerMaxConcurrentFromProjectConfigFileV0(config)),
			configSettingSourceFromEnvOrProjectConfigV0(projectDir, envCodebaseBrokerMaxConcurrentV0),
		),
		serverConfigSettingFromRegistryWithSourceV0(
			envCodebaseBrokerTimeoutMSV0,
			strconv.Itoa(int(codebaseBrokerTimeoutFromProjectConfigFileV0(config)/time.Millisecond)),
			configSettingSourceFromEnvOrProjectConfigV0(projectDir, envCodebaseBrokerTimeoutMSV0),
		),
		serverSensitiveConfigSettingFromRegistryWithSourceV0(
			envCodebaseBrokerStateDirV0,
			sensitiveConfigValueFromSourceV0(
				codebaseBrokerStateDirFromProjectConfigFileV0(config),
				codebaseBrokerConfiguredStateDirRefV0,
				stateDirSource,
			),
			stateDirSource,
		),
		serverConfigSettingFromRegistryWithSourceV0(
			envCodebaseBrokerWatchdogEnabledV0,
			strconv.FormatBool(codebaseBrokerWatchdogEnabledFromProjectConfigFileV0(config)),
			configSettingSourceFromEnvOrProjectConfigV0(projectDir, envCodebaseBrokerWatchdogEnabledV0),
		),
		serverConfigSettingFromRegistryWithSourceV0(
			envCodebaseBrokerWatchdogStopOrphansV0,
			strconv.FormatBool(codebaseBrokerWatchdogStopOrphansFromProjectConfigFileV0(config)),
			configSettingSourceFromEnvOrProjectConfigV0(projectDir, envCodebaseBrokerWatchdogStopOrphansV0),
		),
		serverConfigSettingFromRegistryWithSourceV0(
			envCodebaseBrokerWatchdogOrphanMinAgeSecondsV0,
			strconv.Itoa(codebaseBrokerWatchdogOrphanMinAgeSecondsFromProjectConfigFileV0(config)),
			configSettingSourceFromEnvOrProjectConfigV0(projectDir, envCodebaseBrokerWatchdogOrphanMinAgeSecondsV0),
		),
		serverSensitiveConfigSettingFromRegistryWithSourceV0(
			envCodebaseBrokerCommandV0,
			sensitiveConfigValueFromSourceV0(
				codebaseBrokerCommandFromProjectConfigFileV0(config),
				codebaseBrokerConfiguredCommandRefV0,
				commandSource,
			),
			commandSource,
		),
		serverConfigSettingFromRegistryWithSourceV0(
			envCodebaseBrokerProjectNameV0,
			codebaseBrokerProjectNameFromProjectConfigFileV0(config),
			configSettingSourceFromEnvOrProjectConfigV0(projectDir, envCodebaseBrokerProjectNameV0),
		),
	}
}

func sensitiveConfigValueFromSourceV0(value string, configuredRef string, source string) string {
	if strings.TrimSpace(source) == "defaulted" {
		return strings.TrimSpace(value)
	}
	return configuredRefValueV0(value, configuredRef)
}
