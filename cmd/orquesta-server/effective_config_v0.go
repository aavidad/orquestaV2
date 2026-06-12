package main

import (
	"os"
	"strconv"
	"strings"
	"time"

	orquestaserver "orquesta/modulos/orquesta-server"
)

func serverEffectiveConfigFromEnvV0(config orquestaserver.ConfigV0) orquestaserver.ServerEffectiveConfigV0 {
	config = orquestaserver.NormalizeConfigV0(config)
	codexRuntime := codexRuntimeEnvConfigFromEnvV0()
	stackCapacity := codexStackCapacityEnvConfigFromEnvV0()
	directorWaveLimits := codexDirectorWaveLimitsEnvConfigFromEnvV0()
	egressSanitizer := egressSanitizerConfigFromEnvV0()
	worktreeSnapshotBudget := codexServerWorktreeSnapshotReadBudgetFromEnvV0()
	daemonEnvPolicy := serverDaemonStartEnvPolicyV0(os.Environ(), config)
	settings := []orquestaserver.ServerConfigSettingV0{
		serverConfigSettingFromRegistryV0(envServerAuditFileV0, config.AuditFile),
		serverConfigSettingFromRegistryV0(envServerAuditDisabledV0, strconv.FormatBool(config.AuditDisabled)),
		serverConfigSettingFromRegistryV0(envServerRemoteControlPlaneConfirmV0, strconv.FormatBool(config.ControlPlane.RemoteAccessOptIn)),
		serverSensitiveConfigSettingFromRegistryV0(envServerControlTokenV0, tokenPresenceV0(config.ControlPlane.Token)),
		serverConfigSettingFromRegistryV0(envServerControlPrincipalV0, config.ControlPlane.Principal),
		serverConfigSettingFromRegistryV0(envServerControlPermissionRefV0, config.ControlPlane.PermissionRef),
		serverConfigSettingFromRegistryV0(envServerControlPublicReasonV0, config.ControlPlane.PublicReason),
		serverConfigSettingFromRegistryV0(envServerMaxRunsPerTickV0, strconv.Itoa(config.SupervisorCommand.MaxRunsPerTick)),
		serverConfigSettingFromRegistryV0(envServerMaxExecutionsPerTickV0, strconv.Itoa(config.SupervisorCommand.MaxExecutions)),
		serverConfigSettingFromRegistryV0(envServerQueueLimitV0, strconv.Itoa(serverRunQueueLimitFromEnvV0())),
		serverConfigSettingFromRegistryV0(envServerDrainMaxDispatchesV0, strconv.Itoa(config.SupervisorCommand.DrainLimits.MaxDispatchesPerWait)),
		serverConfigSettingFromRegistryV0(envServerDrainMaxOutboxV0, strconv.Itoa(config.SupervisorCommand.DrainLimits.MaxOutboxPerCycle)),
		serverConfigSettingFromRegistryV0(envServerDrainMaxExternalWaitsV0, strconv.Itoa(config.SupervisorCommand.DrainLimits.MaxExternalWaits)),
		serverConfigSettingFromRegistryV0(envServerResidentDirectorEnabledV0, strconv.FormatBool(config.ResidentDirectorEnabled)),
		serverConfigSettingFromRegistryV0(envServerResidentDirectorMaxActionsV0, strconv.Itoa(config.ResidentDirectorMaxActions)),
		serverConfigSettingFromRegistryV0(envServerTickIntervalMSV0, strconv.Itoa(int(config.TickInterval/time.Millisecond))),
		serverConfigSettingFromRegistryV0(envServerShutdownGraceMSV0, strconv.Itoa(int(config.ShutdownGracePeriod/time.Millisecond))),
		serverConfigSettingFromRegistryV0(envServerReadHeaderTimeoutMSV0, strconv.Itoa(int(config.HTTPResourceLimits.ReadHeaderTimeout/time.Millisecond))),
		serverConfigSettingFromRegistryV0(envServerReadTimeoutMSV0, strconv.Itoa(int(config.HTTPResourceLimits.ReadTimeout/time.Millisecond))),
		serverConfigSettingFromRegistryV0(envServerWriteTimeoutMSV0, strconv.Itoa(int(config.HTTPResourceLimits.WriteTimeout/time.Millisecond))),
		serverConfigSettingFromRegistryV0(envServerIdleTimeoutMSV0, strconv.Itoa(int(config.HTTPResourceLimits.IdleTimeout/time.Millisecond))),
		serverConfigSettingFromRegistryV0(envServerMaxHeaderBytesV0, strconv.Itoa(config.HTTPResourceLimits.MaxHeaderBytes)),
		serverConfigSettingFromRegistryV0(envServerControlBodyMaxBytesV0, strconv.FormatInt(config.HTTPResourceLimits.ControlBodyBytes, 10)),
		serverConfigSettingFromRegistryV0(envServerSelfWatchdogDisabledV0, strconv.FormatBool(config.SelfWatchdog.Disabled)),
		serverConfigSettingFromRegistryV0(envServerSelfWatchdogCPUHighPercentV0, strconv.Itoa(config.SelfWatchdog.CPUHighPercent)),
		serverConfigSettingFromRegistryV0(envServerSelfWatchdogSustainedSecondsV0, strconv.Itoa(int(config.SelfWatchdog.SustainedFor/time.Second))),
		serverConfigSettingFromRegistryV0(envServerSelfWatchdogNoProgressSecondsV0, strconv.Itoa(int(config.SelfWatchdog.NoProgressFor/time.Second))),
		serverConfigSettingFromRegistryV0(envServerIdleSelfImprovementTargetQueueV0, strconv.Itoa(config.IdleSelfImprovementTargetQueue)),
		serverSensitiveConfigSettingFromRegistryV0(
			envServerIdleSelfImprovementProjectWorkDirV0,
			configuredRefValueV0(config.IdleSelfImprovementProjectWorkDir, "idle-self-improvement-project-workdir-configured"),
		),
		serverConfigSettingFromRegistryV0(envServerIdleSelfImprovementMaxRequestsV0, strconv.Itoa(config.IdleSelfImprovementMaxRequests)),
		serverConfigSettingFromRegistryV0(envServerDaemonLogMaxBytesV0, strconv.FormatInt(config.DaemonLogPolicy.MaxBytes, 10)),
		serverConfigSettingFromRegistryV0(envServerDaemonLogMaxRotatedV0, strconv.Itoa(config.DaemonLogPolicy.MaxRotatedFiles)),
		serverConfigSettingFromRegistryV0(envServerDaemonLogRetentionDaysV0, strconv.Itoa(config.DaemonLogPolicy.RetentionDays)),
		serverConfigSettingFromRegistryV0(envServerDaemonLogRawEnabledV0, strconv.FormatBool(config.DaemonLogPolicy.LocalRawEnabled)),
		serverConfigSettingFromRegistryV0(envSecurityModeV0, serverSecurityModeEffectiveValueV0()),
		serverConfigSettingFromRegistryV0(envRailsModeV0, serverRailsModeEffectiveValueV0()),
		serverConfigSettingFromRegistryV0(envDetailProhibitedRailsV0, detailRailsEffectiveValueV0()),
		serverConfigSettingFromRegistryV0(envDetailProhibitedRailsScopeV0, detailRailsScopeEffectiveValueV0()),
		serverConfigSettingFromRegistryV0(envCodexExecutionModeV0, codexExecutionModeFromEnvV0()),
		serverConfigSettingFromRegistryV0(envCodexMaxBatchReadyV0, strconv.Itoa(codexRuntime.Limits.MaxBatchReady)),
		serverConfigSettingFromRegistryV0(envCodexMaxConcurrencyV0, strconv.Itoa(codexRuntime.Limits.MaxLiveProcesses)),
		serverConfigSettingFromRegistryV0(envCodexReasoningEffortV0, codexRuntime.ReasoningEffort),
		serverConfigSettingFromRegistryV0(envCapacityReasoningEffortV0, string(stackCapacity.ReasoningEffort)),
		serverConfigSettingFromRegistryV0(envCapacityPolicyRefV0, stackCapacity.PolicyRef),
		serverConfigSettingFromRegistryV0(envCapacityPoolRefV0, stackCapacity.PoolRef),
		serverConfigSettingFromRegistryV0(envCapacityModelRefV0, stackCapacity.ModelRef),
		serverConfigSettingFromRegistryV0(envCapacityQuotaRefV0, stackCapacity.QuotaRef),
		serverConfigSettingFromRegistryV0(envEgressSanitizerEnabledV0, strconv.FormatBool(egressSanitizer.Enabled)),
		serverConfigSettingFromRegistryV0(envEgressSanitizerRefV0, egressSanitizer.SanitizerRef),
		serverConfigSettingFromRegistryV0(envEgressSanitizerLocalModelEnabledV0, strconv.FormatBool(egressSanitizer.LocalModel.Enabled)),
		serverSensitiveConfigSettingFromRegistryV0(envEgressSanitizerLocalModelRefV0, configuredRefValueV0(egressSanitizer.LocalModel.ModelRef, "egress-sanitizer-local-model-ref-configured")),
		serverSensitiveConfigSettingFromRegistryV0(envEgressSanitizerLocalModelPathV0, configuredEnvValueV0(envEgressSanitizerLocalModelPathV0, "egress-sanitizer-local-model-path-configured")),
		serverSensitiveConfigSettingFromRegistryV0(envEgressSanitizerLocalRuntimeRefV0, configuredRefValueV0(egressSanitizer.LocalModel.RuntimeRef, "egress-sanitizer-local-runtime-ref-configured")),
		serverSensitiveConfigSettingFromRegistryV0(envEgressSanitizerLocalEvidenceRefV0, configuredRefValueV0(egressSanitizer.LocalModel.EvidenceRef, "egress-sanitizer-local-evidence-ref-configured")),
		serverConfigSettingFromRegistryV0(envEgressSanitizerSidecarEnabledV0, strconv.FormatBool(egressSanitizer.Sidecar.Enabled)),
		serverConfigSettingFromRegistryV0(envEgressSanitizerSidecarRefV0, egressSanitizer.Sidecar.SidecarRef),
		serverConfigSettingFromRegistryV0(envEgressSanitizerSidecarAdapterRefV0, egressSanitizer.Sidecar.AdapterRef),
		serverConfigSettingFromRegistryV0(envEgressSanitizerSidecarTransportRefV0, egressSanitizer.Sidecar.TransportRef),
		serverConfigSettingFromRegistryV0(envEgressSanitizerSidecarEvidenceRefV0, egressSanitizer.Sidecar.EvidenceRef),
		serverSensitiveConfigSettingFromRegistryV0(envEgressSanitizerSidecarCommandV0, configuredEnvValueV0(envEgressSanitizerSidecarCommandV0, "egress-sanitizer-sidecar-command-configured")),
		serverSensitiveConfigSettingFromRegistryV0(envEgressSanitizerSidecarLocalEndpointV0, configuredEnvValueV0(envEgressSanitizerSidecarLocalEndpointV0, "egress-sanitizer-sidecar-local-endpoint-configured")),
		serverConfigSettingFromRegistryV0(envCodexDirectorWaveAgentsV0, strconv.Itoa(directorWaveLimits.Agents)),
		serverConfigSettingFromRegistryV0(envCodexDirectorMaxSubagentsPerAgentV0, strconv.Itoa(directorWaveLimits.MaxSubagentsPerAgent)),
		serverConfigSettingFromRegistryV0(envCodexDirectorRecursiveAgentBudgetV0, strconv.Itoa(directorWaveLimits.RecursiveAgentBudget)),
	}
	settings = append(settings, codexServerWorktreeSnapshotBudgetSettingsV0(worktreeSnapshotBudget)...)
	settings = append(settings, hermesOperatorEffectiveConfigSettingsV0()...)
	settings = append(settings, ollamaModelManagerEffectiveConfigSettingsV0()...)
	settings = append(settings, opesRegistryFinalPkgEffectiveConfigSettingsV0()...)
	settings = append(settings, daemonStartEnvSettingsV0(daemonEnvPolicy)...)
	return orquestaserver.NormalizeServerEffectiveConfigV0(orquestaserver.ServerEffectiveConfigV0{
		SchemaVersion: orquestaserver.ServerEffectiveConfigSchemaVersionV0,
		Settings:      settings,
	})
}

func detailRailsEffectiveValueV0() string {
	return serverDetailRailsEffectiveValueV0()
}

func detailRailsScopeEffectiveValueV0() string {
	return serverDetailRailsScopeEffectiveValueV0()
}

func tokenPresenceV0(value string) string {
	if strings.TrimSpace(value) == "" {
		return "absent"
	}
	return "present"
}

func configuredRefValueV0(value string, configuredRef string) string {
	if strings.TrimSpace(value) == "" {
		return ""
	}
	return configuredRef
}

func configuredEnvValueV0(key string, configuredRef string) string {
	return configuredRefValueV0(os.Getenv(key), configuredRef)
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
		Source:          configSettingSourceFromEnvV0(key),
		Scope:           scope,
		Label:           label,
		Description:     description,
		RestartBehavior: "restart_required",
		Editable:        true,
		Canonical:       true,
	}
}

func configSettingSourceFromEnvV0(key string) string {
	if strings.TrimSpace(os.Getenv(key)) != "" {
		return "explicit"
	}
	return "defaulted"
}
