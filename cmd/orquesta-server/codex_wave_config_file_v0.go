package main

import (
	"os"
	"strconv"
	"strings"

	orquestaserver "orquesta/modulos/orquesta-server"
)

type serverProjectConfigCodexWaveV0 struct {
	Agents                     *int    `json:"agents,omitempty"`
	WaveRef                    *string `json:"wave_ref,omitempty"`
	RuntimeWorkDir             *string `json:"runtime_workdir,omitempty"`
	SourceCodeHome             *string `json:"source_code_home,omitempty"`
	Model                      *string `json:"model,omitempty"`
	ReasoningEffort            *string `json:"reasoning_effort,omitempty"`
	Profile                    *string `json:"profile,omitempty"`
	Sandbox                    *string `json:"sandbox,omitempty"`
	ApprovalPolicy             *string `json:"approval_policy,omitempty"`
	ExtraArgs                  *string `json:"extra_args,omitempty"`
	IsolateHome                *bool   `json:"isolate_home,omitempty"`
	StrictCredentialProjection *bool   `json:"strict_credential_projection,omitempty"`
	ProjectMemories            *bool   `json:"project_memories,omitempty"`
	ProjectionMaxFiles         *int    `json:"projection_max_files,omitempty"`
	ProjectionMaxFileBytes     *int    `json:"projection_max_file_bytes,omitempty"`
	ProjectionMaxTotalBytes    *int    `json:"projection_max_total_bytes,omitempty"`
	PurgeRuntime               *bool   `json:"purge_runtime,omitempty"`
	PurgeRuntimeConfirm        *string `json:"purge_runtime_confirm,omitempty"`
	PurgeRuntimeReport         *bool   `json:"purge_runtime_report,omitempty"`
	AllowUnmanagedLaunch       *bool   `json:"allow_unmanaged_launch,omitempty"`
	UnmanagedLaunchReason      *string `json:"unmanaged_launch_reason,omitempty"`
	UnmanagedLaunchConfirm     *string `json:"unmanaged_launch_confirm,omitempty"`
	Path                       *string `json:"path,omitempty"`
	TailReason                 *string `json:"tail_reason,omitempty"`
	StopConfirm                *string `json:"stop_confirm,omitempty"`
	StopForce                  *bool   `json:"stop_force,omitempty"`
}

func codexWaveProjectConfigFromArgsEnvV0(args []string) serverProjectConfigFileV0 {
	projectDir := strings.TrimSpace(os.Getenv(envCodexProjectWorkDirV0))
	for i, arg := range args {
		if arg == "--project-dir" && i+1 < len(args) {
			projectDir = strings.TrimSpace(args[i+1])
			break
		}
		if strings.HasPrefix(arg, "--project-dir=") {
			projectDir = strings.TrimSpace(strings.TrimPrefix(arg, "--project-dir="))
			break
		}
	}
	return projectConfigFromProjectDirBestEffortV0(projectDir)
}

func codexWaveProjectConfigFromEnvBestEffortV0() serverProjectConfigFileV0 {
	return projectConfigFromProjectDirBestEffortV0(os.Getenv(envCodexProjectWorkDirV0))
}

func codexWaveIntValueFromProjectConfigFileV0(config serverProjectConfigFileV0, key string, fallback int) int {
	switch key {
	case envCodexWaveAgentsV0:
		return intProjectConfigOrEnvOrDefaultV0(key, config.CodexWave.Agents, fallback)
	case envCodexWaveProjectionMaxFilesV0:
		return intProjectConfigOrEnvOrDefaultV0(key, config.CodexWave.ProjectionMaxFiles, fallback)
	case envCodexWaveProjectionMaxFileBytesV0:
		return intProjectConfigOrEnvOrDefaultV0(key, config.CodexWave.ProjectionMaxFileBytes, fallback)
	case envCodexWaveProjectionMaxTotalBytesV0:
		return intProjectConfigOrEnvOrDefaultV0(key, config.CodexWave.ProjectionMaxTotalBytes, fallback)
	default:
		return intEnvOrDefaultV0(key, fallback)
	}
}

func codexWaveBoolValueFromProjectConfigFileV0(config serverProjectConfigFileV0, key string, fallback bool) bool {
	switch key {
	case envCodexWaveIsolateHomeV0:
		return boolProjectConfigOrEnvOrDefaultV0(key, config.CodexWave.IsolateHome, fallback)
	case envCodexWaveStrictCredentialProjectionV0:
		return boolProjectConfigOrEnvOrDefaultV0(key, config.CodexWave.StrictCredentialProjection, fallback)
	case envCodexWaveProjectMemoriesV0:
		return boolProjectConfigOrEnvOrDefaultV0(key, config.CodexWave.ProjectMemories, fallback)
	case envCodexWavePurgeRuntimeV0:
		return boolProjectConfigOrEnvOrDefaultV0(key, config.CodexWave.PurgeRuntime, fallback)
	case envCodexWavePurgeRuntimeReportV0:
		return boolProjectConfigOrEnvOrDefaultV0(key, config.CodexWave.PurgeRuntimeReport, fallback)
	case envCodexWaveAllowUnmanagedLaunchV0:
		return boolProjectConfigOrEnvOrDefaultV0(key, config.CodexWave.AllowUnmanagedLaunch, fallback)
	case envCodexWaveStopForceV0:
		return boolProjectConfigOrEnvOrDefaultV0(key, config.CodexWave.StopForce, fallback)
	default:
		return boolEnvOrDefaultV0(key, fallback)
	}
}

func codexWaveStringValueFromProjectConfigFileV0(config serverProjectConfigFileV0, key string, fallback string) string {
	switch key {
	case envCodexWaveRefV0:
		return stringProjectConfigOrEnvOrDefaultV0(key, config.CodexWave.WaveRef, fallback)
	case envCodexWaveRuntimeWorkDirV0:
		return stringProjectConfigOrEnvOrDefaultV0(key, config.CodexWave.RuntimeWorkDir, fallback)
	case envCodexWaveSourceCodeHomeV0:
		return stringProjectConfigOrEnvOrDefaultV0(key, config.CodexWave.SourceCodeHome, fallback)
	case envCodexWaveModelV0:
		return stringProjectConfigOrEnvOrDefaultV0(key, config.CodexWave.Model, fallback)
	case envCodexWaveReasoningEffortV0:
		return stringProjectConfigOrEnvOrDefaultV0(key, config.CodexWave.ReasoningEffort, fallback)
	case envCodexWaveProfileV0:
		return stringProjectConfigOrEnvOrDefaultV0(key, config.CodexWave.Profile, fallback)
	case envCodexWaveSandboxV0:
		return stringProjectConfigOrEnvOrDefaultV0(key, config.CodexWave.Sandbox, fallback)
	case envCodexWaveApprovalPolicyV0:
		return stringProjectConfigOrEnvOrDefaultV0(key, config.CodexWave.ApprovalPolicy, fallback)
	case envCodexWaveExtraArgsV0:
		return stringProjectConfigOrEnvOrDefaultV0(key, config.CodexWave.ExtraArgs, fallback)
	case envCodexWavePurgeRuntimeConfirmV0:
		return stringProjectConfigOrEnvOrDefaultV0(key, config.CodexWave.PurgeRuntimeConfirm, fallback)
	case envCodexWaveUnmanagedLaunchReasonV0:
		return stringProjectConfigOrEnvOrDefaultV0(key, config.CodexWave.UnmanagedLaunchReason, fallback)
	case envCodexWaveUnmanagedLaunchConfirmV0:
		return stringProjectConfigOrEnvOrDefaultV0(key, config.CodexWave.UnmanagedLaunchConfirm, fallback)
	case envCodexWavePathV0:
		return stringProjectConfigOrEnvOrDefaultV0(key, config.CodexWave.Path, fallback)
	case envCodexWaveTailReasonV0:
		return stringProjectConfigOrEnvOrDefaultV0(key, config.CodexWave.TailReason, fallback)
	case envCodexWaveStopConfirmV0:
		return stringProjectConfigOrEnvOrDefaultV0(key, config.CodexWave.StopConfirm, fallback)
	default:
		return envOrDefaultV0(key, fallback)
	}
}

func codexWaveProjectConfigHasValueForEnvKeyV0(config serverProjectConfigCodexWaveV0, key string) bool {
	switch key {
	case envCodexWaveAgentsV0:
		return configIntPointerPositiveV0(config.Agents)
	case envCodexWaveRefV0:
		return configStringPointerHasValueV0(config.WaveRef)
	case envCodexWaveRuntimeWorkDirV0:
		return configStringPointerHasValueV0(config.RuntimeWorkDir)
	case envCodexWaveSourceCodeHomeV0:
		return configStringPointerHasValueV0(config.SourceCodeHome)
	case envCodexWaveModelV0:
		return configStringPointerHasValueV0(config.Model)
	case envCodexWaveReasoningEffortV0:
		return configStringPointerHasValueV0(config.ReasoningEffort)
	case envCodexWaveProfileV0:
		return configStringPointerHasValueV0(config.Profile)
	case envCodexWaveSandboxV0:
		return configStringPointerHasValueV0(config.Sandbox)
	case envCodexWaveApprovalPolicyV0:
		return configStringPointerHasValueV0(config.ApprovalPolicy)
	case envCodexWaveExtraArgsV0:
		return configStringPointerHasValueV0(config.ExtraArgs)
	case envCodexWaveIsolateHomeV0:
		return config.IsolateHome != nil
	case envCodexWaveStrictCredentialProjectionV0:
		return config.StrictCredentialProjection != nil
	case envCodexWaveProjectMemoriesV0:
		return config.ProjectMemories != nil
	case envCodexWaveProjectionMaxFilesV0:
		return configIntPointerPositiveV0(config.ProjectionMaxFiles)
	case envCodexWaveProjectionMaxFileBytesV0:
		return configIntPointerPositiveV0(config.ProjectionMaxFileBytes)
	case envCodexWaveProjectionMaxTotalBytesV0:
		return configIntPointerPositiveV0(config.ProjectionMaxTotalBytes)
	case envCodexWavePurgeRuntimeV0:
		return config.PurgeRuntime != nil
	case envCodexWavePurgeRuntimeConfirmV0:
		return configStringPointerHasValueV0(config.PurgeRuntimeConfirm)
	case envCodexWavePurgeRuntimeReportV0:
		return config.PurgeRuntimeReport != nil
	case envCodexWaveAllowUnmanagedLaunchV0:
		return config.AllowUnmanagedLaunch != nil
	case envCodexWaveUnmanagedLaunchReasonV0:
		return configStringPointerHasValueV0(config.UnmanagedLaunchReason)
	case envCodexWaveUnmanagedLaunchConfirmV0:
		return configStringPointerHasValueV0(config.UnmanagedLaunchConfirm)
	case envCodexWavePathV0:
		return configStringPointerHasValueV0(config.Path)
	case envCodexWaveTailReasonV0:
		return configStringPointerHasValueV0(config.TailReason)
	case envCodexWaveStopConfirmV0:
		return configStringPointerHasValueV0(config.StopConfirm)
	case envCodexWaveStopForceV0:
		return config.StopForce != nil
	default:
		return false
	}
}

func codexWaveEffectiveConfigSettingsV0(
	config orquestaserver.ConfigV0,
	projectConfig serverProjectConfigFileV0,
) []orquestaserver.ServerConfigSettingV0 {
	return []orquestaserver.ServerConfigSettingV0{
		serverPositiveConfigSettingFromConfigV0(config, envCodexWaveAgentsV0, codexWaveIntValueFromProjectConfigFileV0(projectConfig, envCodexWaveAgentsV0, 1)),
		serverSensitiveConfigSettingFromRegistryWithSourceV0(envCodexWaveRefV0, configuredRefValueV0(codexWaveStringValueFromProjectConfigFileV0(projectConfig, envCodexWaveRefV0, ""), "codex-wave-ref-configured"), configSettingSourceFromConfigOrProjectConfigV0(config, envCodexWaveRefV0)),
		serverSensitiveConfigSettingFromRegistryWithSourceV0(envCodexWaveRuntimeWorkDirV0, configuredRefValueV0(codexWaveStringValueFromProjectConfigFileV0(projectConfig, envCodexWaveRuntimeWorkDirV0, ""), "codex-wave-runtime-workdir-configured"), configSettingSourceFromConfigOrProjectConfigV0(config, envCodexWaveRuntimeWorkDirV0)),
		serverSensitiveConfigSettingFromRegistryWithSourceV0(envCodexWaveSourceCodeHomeV0, configuredRefValueV0(codexWaveStringValueFromProjectConfigFileV0(projectConfig, envCodexWaveSourceCodeHomeV0, ""), "codex-wave-source-code-home-configured"), configSettingSourceFromConfigOrProjectConfigV0(config, envCodexWaveSourceCodeHomeV0)),
		serverConfigSettingFromRegistryWithSourceV0(envCodexWaveModelV0, codexWaveStringValueFromProjectConfigFileV0(projectConfig, envCodexWaveModelV0, ""), configSettingSourceFromConfigOrProjectConfigV0(config, envCodexWaveModelV0)),
		serverConfigSettingFromRegistryWithSourceV0(envCodexWaveReasoningEffortV0, codexReasoningEffortPolicyV0(codexWaveStringValueFromProjectConfigFileV0(projectConfig, envCodexWaveReasoningEffortV0, envOrDefaultV0(envCodexReasoningEffortV0, "medium"))), configSettingSourceFromConfigOrProjectConfigV0(config, envCodexWaveReasoningEffortV0)),
		serverConfigSettingFromRegistryWithSourceV0(envCodexWaveProfileV0, codexWaveStringValueFromProjectConfigFileV0(projectConfig, envCodexWaveProfileV0, ""), configSettingSourceFromConfigOrProjectConfigV0(config, envCodexWaveProfileV0)),
		serverConfigSettingFromRegistryWithSourceV0(envCodexWaveSandboxV0, codexWaveStringValueFromProjectConfigFileV0(projectConfig, envCodexWaveSandboxV0, "workspace-write"), configSettingSourceFromConfigOrProjectConfigV0(config, envCodexWaveSandboxV0)),
		serverConfigSettingFromRegistryWithSourceV0(envCodexWaveApprovalPolicyV0, codexWaveStringValueFromProjectConfigFileV0(projectConfig, envCodexWaveApprovalPolicyV0, "never"), configSettingSourceFromConfigOrProjectConfigV0(config, envCodexWaveApprovalPolicyV0)),
		serverConfigSettingFromRegistryWithSourceV0(envCodexWaveExtraArgsV0, codexWaveStringValueFromProjectConfigFileV0(projectConfig, envCodexWaveExtraArgsV0, ""), configSettingSourceFromConfigOrProjectConfigV0(config, envCodexWaveExtraArgsV0)),
		serverConfigSettingFromRegistryWithSourceV0(envCodexWaveIsolateHomeV0, strconv.FormatBool(codexWaveBoolValueFromProjectConfigFileV0(projectConfig, envCodexWaveIsolateHomeV0, false)), configSettingSourceFromConfigOrProjectConfigV0(config, envCodexWaveIsolateHomeV0)),
		serverConfigSettingFromRegistryWithSourceV0(envCodexWaveStrictCredentialProjectionV0, strconv.FormatBool(codexWaveBoolValueFromProjectConfigFileV0(projectConfig, envCodexWaveStrictCredentialProjectionV0, false)), configSettingSourceFromConfigOrProjectConfigV0(config, envCodexWaveStrictCredentialProjectionV0)),
		serverConfigSettingFromRegistryWithSourceV0(envCodexWaveProjectMemoriesV0, strconv.FormatBool(codexWaveBoolValueFromProjectConfigFileV0(projectConfig, envCodexWaveProjectMemoriesV0, false)), configSettingSourceFromConfigOrProjectConfigV0(config, envCodexWaveProjectMemoriesV0)),
		serverPositiveConfigSettingFromConfigV0(config, envCodexWaveProjectionMaxFilesV0, codexWaveIntValueFromProjectConfigFileV0(projectConfig, envCodexWaveProjectionMaxFilesV0, codexWaveProjectionDefaultMaxFilesV0)),
		serverPositiveConfigSettingFromConfigV0(config, envCodexWaveProjectionMaxFileBytesV0, codexWaveIntValueFromProjectConfigFileV0(projectConfig, envCodexWaveProjectionMaxFileBytesV0, int(codexWaveProjectionDefaultMaxFileBytesV0))),
		serverPositiveConfigSettingFromConfigV0(config, envCodexWaveProjectionMaxTotalBytesV0, codexWaveIntValueFromProjectConfigFileV0(projectConfig, envCodexWaveProjectionMaxTotalBytesV0, int(codexWaveProjectionDefaultMaxTotalBytesV0))),
		serverConfigSettingFromRegistryWithSourceV0(envCodexWavePurgeRuntimeV0, strconv.FormatBool(codexWaveBoolValueFromProjectConfigFileV0(projectConfig, envCodexWavePurgeRuntimeV0, false)), configSettingSourceFromConfigOrProjectConfigV0(config, envCodexWavePurgeRuntimeV0)),
		serverSensitiveConfigSettingFromRegistryWithSourceV0(envCodexWavePurgeRuntimeConfirmV0, configuredRefValueV0(codexWaveStringValueFromProjectConfigFileV0(projectConfig, envCodexWavePurgeRuntimeConfirmV0, ""), "codex-wave-purge-confirm-configured"), configSettingSourceFromConfigOrProjectConfigV0(config, envCodexWavePurgeRuntimeConfirmV0)),
		serverConfigSettingFromRegistryWithSourceV0(envCodexWavePurgeRuntimeReportV0, strconv.FormatBool(codexWaveBoolValueFromProjectConfigFileV0(projectConfig, envCodexWavePurgeRuntimeReportV0, false)), configSettingSourceFromConfigOrProjectConfigV0(config, envCodexWavePurgeRuntimeReportV0)),
		serverConfigSettingFromRegistryWithSourceV0(envCodexWaveAllowUnmanagedLaunchV0, strconv.FormatBool(codexWaveBoolValueFromProjectConfigFileV0(projectConfig, envCodexWaveAllowUnmanagedLaunchV0, false)), configSettingSourceFromConfigOrProjectConfigV0(config, envCodexWaveAllowUnmanagedLaunchV0)),
		serverConfigSettingFromRegistryWithSourceV0(envCodexWaveUnmanagedLaunchReasonV0, codexWaveStringValueFromProjectConfigFileV0(projectConfig, envCodexWaveUnmanagedLaunchReasonV0, ""), configSettingSourceFromConfigOrProjectConfigV0(config, envCodexWaveUnmanagedLaunchReasonV0)),
		serverSensitiveConfigSettingFromRegistryWithSourceV0(envCodexWaveUnmanagedLaunchConfirmV0, configuredRefValueV0(codexWaveStringValueFromProjectConfigFileV0(projectConfig, envCodexWaveUnmanagedLaunchConfirmV0, ""), "codex-wave-unmanaged-confirm-configured"), configSettingSourceFromConfigOrProjectConfigV0(config, envCodexWaveUnmanagedLaunchConfirmV0)),
		serverSensitiveConfigSettingFromRegistryWithSourceV0(envCodexWavePathV0, configuredRefValueV0(codexWaveStringValueFromProjectConfigFileV0(projectConfig, envCodexWavePathV0, ""), "codex-wave-path-configured"), configSettingSourceFromConfigOrProjectConfigV0(config, envCodexWavePathV0)),
		serverConfigSettingFromRegistryWithSourceV0(envCodexWaveTailReasonV0, codexWaveStringValueFromProjectConfigFileV0(projectConfig, envCodexWaveTailReasonV0, ""), configSettingSourceFromConfigOrProjectConfigV0(config, envCodexWaveTailReasonV0)),
		serverSensitiveConfigSettingFromRegistryWithSourceV0(envCodexWaveStopConfirmV0, configuredRefValueV0(codexWaveStringValueFromProjectConfigFileV0(projectConfig, envCodexWaveStopConfirmV0, ""), "codex-wave-stop-confirm-configured"), configSettingSourceFromConfigOrProjectConfigV0(config, envCodexWaveStopConfirmV0)),
		serverConfigSettingFromRegistryWithSourceV0(envCodexWaveStopForceV0, strconv.FormatBool(codexWaveBoolValueFromProjectConfigFileV0(projectConfig, envCodexWaveStopForceV0, false)), configSettingSourceFromConfigOrProjectConfigV0(config, envCodexWaveStopForceV0)),
	}
}
