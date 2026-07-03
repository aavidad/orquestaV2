package main

import (
	"os"
	"strconv"
	"strings"
	"time"

	orquestamcp "orquesta/modulos/orquesta-mcp"
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
	goalProgressPolicy := serverAutoprogrammingGoalProgressPolicyFromEnvV0()
	settings := []orquestaserver.ServerConfigSettingV0{
		serverConfigSettingFromRegistryV0(envServerAuditFileV0, config.AuditFile),
		serverConfigSettingFromRegistryV0(envServerAuditDisabledV0, strconv.FormatBool(config.AuditDisabled)),
		serverConfigSettingFromRegistryV0(envServerRemoteControlPlaneConfirmV0, strconv.FormatBool(config.ControlPlane.RemoteAccessOptIn)),
		serverSensitiveConfigSettingFromRegistryV0(envServerControlTokenV0, tokenPresenceV0(config.ControlPlane.Token)),
		serverConfigSettingFromRegistryV0(envServerControlPrincipalV0, config.ControlPlane.Principal),
		serverConfigSettingFromRegistryV0(envServerControlPermissionRefV0, config.ControlPlane.PermissionRef),
		serverConfigSettingFromRegistryV0(envServerControlPublicReasonV0, config.ControlPlane.PublicReason),
		serverConfigSettingFromRegistryV0(envServerSelfProgrammingOnlyV0, strconv.FormatBool(serverSelfProgrammingOnlyFromEnvV0())),
		serverSensitiveConfigSettingFromRegistryV0(
			envServerSelfProgrammingRootV0,
			configuredEnvValueV0(envServerSelfProgrammingRootV0, "self-programming-root-configured"),
		),
		serverConfigSettingFromRegistryV0(envServerMaxRunsPerTickV0, strconv.Itoa(config.SupervisorCommand.MaxRunsPerTick)),
		serverConfigSettingFromRegistryV0(envServerMaxExecutionsPerTickV0, strconv.Itoa(config.SupervisorCommand.MaxExecutions)),
		serverConfigSettingFromRegistryV0(envServerQueueLimitV0, strconv.Itoa(serverRunQueueLimitFromEnvV0())),
		serverConfigSettingFromRegistryV0(envServerDrainMaxDispatchesV0, strconv.Itoa(config.SupervisorCommand.DrainLimits.MaxDispatchesPerWait)),
		serverConfigSettingFromRegistryV0(envServerDrainMaxOutboxV0, strconv.Itoa(config.SupervisorCommand.DrainLimits.MaxOutboxPerCycle)),
		serverConfigSettingFromRegistryV0(envServerDrainMaxExternalWaitsV0, strconv.Itoa(config.SupervisorCommand.DrainLimits.MaxExternalWaits)),
		serverConfigSettingFromRegistryV0(envServerAutonomyEnabledV0, strconv.FormatBool(serverAutonomyEffectiveEnabledFromEnvV0())),
		serverConfigSettingFromRegistryV0(envServerGoalObserverEnabledV0, strconv.FormatBool(config.GoalObserverEnabled)),
		serverConfigSettingFromRegistryV0(envServerGoalObserverIntervalMSV0, strconv.Itoa(int(config.GoalObserverInterval/time.Millisecond))),
		serverConfigSettingFromRegistryV0(envServerGoalObserverMaxItemsV0, strconv.Itoa(config.GoalObserverMaxItems)),
		serverConfigSettingFromRegistryV0(
			envServerGoalObserverFingerprintEnabledV0,
			strconv.FormatBool(serverGoalObserverFingerprintEnabledFromEnvV0()),
		),
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
		serverConfigSettingFromRegistryV0(envServerIdleSelfImprovementDisabledV0, strconv.FormatBool(config.IdleSelfImprovementDisabled)),
		serverIdleSelfImprovementAfterSettingV0(config),
		serverConfigSettingFromRegistryV0(envServerIdleSelfImprovementTargetQueueV0, strconv.Itoa(config.IdleSelfImprovementTargetQueue)),
		serverIdleSelfImprovementGoalFirstSettingV0(config),
		serverConfigSettingFromRegistryV0(envServerIdleSelfImprovementFrozenTestsV0, strconv.FormatBool(config.IdleSelfImprovementFrozenTests)),
		serverSensitiveConfigSettingFromRegistryV0(
			envServerIdleSelfImprovementProjectWorkDirV0,
			configuredRefValueV0(config.IdleSelfImprovementProjectWorkDir, "idle-self-improvement-project-workdir-configured"),
		),
		serverConfigSettingFromRegistryV0(envServerIdleSelfImprovementMaxRequestsV0, strconv.Itoa(config.IdleSelfImprovementMaxRequests)),
		serverConfigSettingFromRegistryV0(envServerIdleSelfImprovementDailyGoalBudgetV0, strconv.Itoa(config.IdleSelfImprovementBudget.MaxGoalsPerDay)),
		serverConfigSettingFromRegistryV0(envServerIdleSelfImprovementDailyContextBudgetBytesV0, strconv.FormatInt(config.IdleSelfImprovementBudget.MaxContextBudgetBytesPerDay, 10)),
		serverConfigSettingFromRegistryV0(envSelfAuditBacklogEnabledV0, strconv.FormatBool(config.SelfAuditBacklogEnabled)),
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
		serverConfigSettingFromRegistryV0(
			envCodexPromoteMaterializedArtifactWithoutAckV0,
			strconv.FormatBool(codexPromoteMaterializedArtifactWithoutAckFromEnvV0()),
		),
		serverConfigSettingFromRegistryV0(envCodexGoalBackendV0, codexGoalBackendFromEnvV0()),
		serverConfigSettingFromRegistryV0(envCodexGoalTimeoutMSV0, strconv.Itoa(codexGoalTimeoutMSFromEnvV0())),
		serverConfigSettingFromRegistryV0(envCodexGoalPreflightTimeoutMSV0, strconv.Itoa(codexGoalPreflightTimeoutMSFromEnvV0())),
		serverConfigSettingFromRegistryV0(
			envAutoprogrammingLegacyDirectorLoopV0,
			strconv.FormatBool(boolEnvOrDefaultV0(envAutoprogrammingLegacyDirectorLoopV0, false)),
		),
		serverConfigSettingFromRegistryV0(
			envAutoprogrammingCheckpointOnlyHighConsumptionTokensV0,
			strconv.FormatInt(goalProgressPolicy.CheckpointOnlyHighConsumptionTokens, 10),
		),
		serverConfigSettingFromRegistryV0(
			envAutoprogrammingCheckpointOnlyMaxWaitSecondsV0,
			strconv.FormatInt(goalProgressPolicy.CheckpointOnlyMaxWaitSeconds, 10),
		),
		serverConfigSettingFromRegistryV0(
			envAutoprogrammingNoCheckpointWarningMaxWaitSecondsV0,
			strconv.FormatInt(goalProgressPolicy.NoCheckpointWarningMaxWaitSeconds, 10),
		),
		serverConfigSettingFromRegistryV0(
			envExternalWorkLegacyDirectorLoopV0,
			strconv.FormatBool(boolEnvOrDefaultV0(envExternalWorkLegacyDirectorLoopV0, false)),
		),
		serverConfigSettingFromRegistryV0(
			envStartupCleanupModeV0,
			startupCleanupModeEffectiveValueV0(),
		),
		serverConfigSettingFromRegistryV0(
			envStartupCleanupScopeRefsV0,
			strings.Join(csvEnvOrDefaultV0(envStartupCleanupScopeRefsV0, nil), ","),
		),
		serverConfigSettingFromRegistryV0(
			envCodebaseBrokerProviderKindV0,
			envOrDefaultV0(envCodebaseBrokerProviderKindV0, "fallback_rg"),
		),
		serverConfigSettingFromRegistryV0(
			envCodebaseBrokerExternalIndexerEnabledV0,
			strconv.FormatBool(boolEnvOrDefaultV0(envCodebaseBrokerExternalIndexerEnabledV0, false)),
		),
		serverConfigSettingFromRegistryV0(
			envCodebaseBrokerMaxConcurrentV0,
			strconv.Itoa(intEnvOrDefaultV0(envCodebaseBrokerMaxConcurrentV0, 4)),
		),
		serverConfigSettingFromRegistryV0(
			envCodebaseBrokerTimeoutMSV0,
			strconv.Itoa(intEnvOrDefaultV0(envCodebaseBrokerTimeoutMSV0, 3000)),
		),
		serverConfigSettingFromRegistryV0(
			envCodebaseBrokerStateDirV0,
			envOrDefaultV0(envCodebaseBrokerStateDirV0, ""),
		),
		serverConfigSettingFromRegistryV0(
			envCodebaseBrokerWatchdogEnabledV0,
			strconv.FormatBool(boolEnvOrDefaultV0(envCodebaseBrokerWatchdogEnabledV0, false)),
		),
		serverConfigSettingFromRegistryV0(
			envCodebaseBrokerWatchdogStopOrphansV0,
			strconv.FormatBool(boolEnvOrDefaultV0(envCodebaseBrokerWatchdogStopOrphansV0, false)),
		),
		serverConfigSettingFromRegistryV0(
			envCodebaseBrokerWatchdogOrphanMinAgeSecondsV0,
			strconv.Itoa(intEnvOrDefaultV0(envCodebaseBrokerWatchdogOrphanMinAgeSecondsV0, 120)),
		),
		serverConfigSettingFromRegistryV0(
			envCodebaseBrokerCommandV0,
			envOrDefaultV0(envCodebaseBrokerCommandV0, "codebase-memory-mcp"),
		),
		serverConfigSettingFromRegistryV0(
			envCodebaseBrokerProjectNameV0,
			envOrDefaultV0(envCodebaseBrokerProjectNameV0, ""),
		),
		serverConfigSettingFromRegistryV0(envOPESBridgeWaitResidentSecondsV0, strconv.Itoa(intEnvOrDefaultV0(envOPESBridgeWaitResidentSecondsV0, 0))),
		serverConfigSettingFromRegistryV0(
			envOPESBridgeWaitResidentIntervalMSV0,
			strconv.Itoa(intEnvOrDefaultV0(envOPESBridgeWaitResidentIntervalMSV0, int(defaultOPESBridgeResidentDispatchIntervalV0/time.Millisecond))),
		),
		serverConfigSettingFromRegistryV0(
			envOPESBridgeRequireRuntimeCompatibilityV0,
			strconv.FormatBool(boolEnvOrDefaultV0(envOPESBridgeRequireRuntimeCompatibilityV0, false)),
		),
		serverConfigSettingFromRegistryV0(
			envOPESBridgeRequiredRuntimeBinarySHA256V0,
			strings.TrimSpace(os.Getenv(envOPESBridgeRequiredRuntimeBinarySHA256V0)),
		),
		serverConfigSettingFromRegistryV0(
			envOPESBridgeRequiredRuntimeBuildRefV0,
			strings.TrimSpace(os.Getenv(envOPESBridgeRequiredRuntimeBuildRefV0)),
		),
		serverConfigSettingFromRegistryV0(
			envOPESBridgeRequiredRuntimeCommitRefV0,
			strings.TrimSpace(os.Getenv(envOPESBridgeRequiredRuntimeCommitRefV0)),
		),
		serverConfigSettingFromRegistryV0(
			envOPESBridgeSpeechSynthesisCapabilityV0,
			strings.TrimSpace(os.Getenv(envOPESBridgeSpeechSynthesisCapabilityV0)),
		),
		serverConfigSettingFromRegistryV0(
			envOPESBridgeSpeechSynthesisCapabilityRefV0,
			strings.TrimSpace(os.Getenv(envOPESBridgeSpeechSynthesisCapabilityRefV0)),
		),
		serverConfigSettingFromRegistryV0(
			envOPESBridgeSpeechSynthesisEvidenceRefsV0,
			strings.TrimSpace(os.Getenv(envOPESBridgeSpeechSynthesisEvidenceRefsV0)),
		),
		serverConfigSettingFromRegistryV0(
			envOPESBridgeSpeechSynthesisReasonV0,
			strings.TrimSpace(os.Getenv(envOPESBridgeSpeechSynthesisReasonV0)),
		),
		serverConfigSettingFromRegistryV0(
			envOPESBridgeSpeechSynthesisNetworkReadyV0,
			strings.TrimSpace(os.Getenv(envOPESBridgeSpeechSynthesisNetworkReadyV0)),
		),
		serverConfigSettingFromRegistryV0(
			envOPESBridgeSpeechSynthesisToolPathReadyV0,
			strings.TrimSpace(os.Getenv(envOPESBridgeSpeechSynthesisToolPathReadyV0)),
		),
		serverConfigSettingFromRegistryV0(
			envOPESBridgeSpeechSynthesisQuotaReadyV0,
			strings.TrimSpace(os.Getenv(envOPESBridgeSpeechSynthesisQuotaReadyV0)),
		),
		serverConfigSettingFromRegistryV0(
			envOPESBridgeSpeechSynthesisProgressHeartbeatReadyV0,
			strings.TrimSpace(os.Getenv(envOPESBridgeSpeechSynthesisProgressHeartbeatReadyV0)),
		),
		serverConfigSettingFromRegistryV0(
			envOPESBridgeSpeechSynthesisProviderTimeoutReadyV0,
			strings.TrimSpace(os.Getenv(envOPESBridgeSpeechSynthesisProviderTimeoutReadyV0)),
		),
		serverConfigSettingFromRegistryV0(
			envOPESBridgeSpeechSynthesisNoProgressTimeoutSecondsV0,
			strconv.Itoa(intEnvOrDefaultV0(envOPESBridgeSpeechSynthesisNoProgressTimeoutSecondsV0, 300)),
		),
		serverSensitiveConfigSettingFromRegistryV0(
			envOPESBridgeSpeechSynthesisToolWorkDirV0,
			configuredEnvValueV0(envOPESBridgeSpeechSynthesisToolWorkDirV0, "opes-speech-synthesis-tool-workdir-configured"),
		),
		serverSensitiveConfigSettingFromRegistryV0(
			envOPESBridgeSpeechSynthesisToolCommandV0,
			configuredEnvValueV0(envOPESBridgeSpeechSynthesisToolCommandV0, "opes-speech-synthesis-tool-command-configured"),
		),
		serverConfigSettingFromRegistryV0(
			envOPESBridgeSpeechSynthesisToolPreflightTimeoutSecondsV0,
			strconv.Itoa(intEnvOrDefaultV0(
				envOPESBridgeSpeechSynthesisToolPreflightTimeoutSecondsV0,
				int(defaultOPESBridgeSpeechSynthesisToolPreflightTimeoutV0/time.Second),
			)),
		),
		serverConfigSettingFromRegistryV0(
			envOPESBridgeRemoteQACapabilityV0,
			strings.TrimSpace(os.Getenv(envOPESBridgeRemoteQACapabilityV0)),
		),
		serverConfigSettingFromRegistryV0(
			envOPESBridgeRemoteQACapabilityRefV0,
			strings.TrimSpace(os.Getenv(envOPESBridgeRemoteQACapabilityRefV0)),
		),
		serverConfigSettingFromRegistryV0(
			envOPESBridgeRemoteQAEvidenceRefsV0,
			strings.TrimSpace(os.Getenv(envOPESBridgeRemoteQAEvidenceRefsV0)),
		),
		serverConfigSettingFromRegistryV0(
			envOPESBridgeRemoteQAReasonV0,
			strings.TrimSpace(os.Getenv(envOPESBridgeRemoteQAReasonV0)),
		),
		serverConfigSettingFromRegistryV0(
			envOPESBridgeRemoteQANetworkReadyV0,
			strings.TrimSpace(os.Getenv(envOPESBridgeRemoteQANetworkReadyV0)),
		),
		serverConfigSettingFromRegistryV0(
			envOPESBridgeRemoteQAAuthStateReadyV0,
			strings.TrimSpace(os.Getenv(envOPESBridgeRemoteQAAuthStateReadyV0)),
		),
		serverConfigSettingFromRegistryV0(
			envOPESBridgeRemoteQAQuotaReadyV0,
			strings.TrimSpace(os.Getenv(envOPESBridgeRemoteQAQuotaReadyV0)),
		),
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
	settings = append(settings, opesTopicRegistryEffectiveConfigSettingsV0()...)
	settings = append(settings, daemonStartEnvSettingsV0(daemonEnvPolicy)...)
	return orquestaserver.NormalizeServerEffectiveConfigV0(orquestaserver.ServerEffectiveConfigV0{
		SchemaVersion: orquestaserver.ServerEffectiveConfigSchemaVersionV0,
		Settings:      settings,
		Diagnostics:   serverEffectiveConfigDiagnosticsFromEnvV0(),
	})
}

func serverIdleSelfImprovementAfterSettingV0(config orquestaserver.ConfigV0) orquestaserver.ServerConfigSettingV0 {
	setting := serverConfigSettingFromRegistryV0(
		envServerIdleSelfImprovementAfterV0,
		strconv.Itoa(int(config.IdleSelfImprovementAfter/time.Second)),
	)
	if serverIdleSelfImprovementAfterLegacyActiveV0() {
		setting.Source = "legacy_alias"
	}
	return setting
}

func serverIdleSelfImprovementGoalFirstSettingV0(config orquestaserver.ConfigV0) orquestaserver.ServerConfigSettingV0 {
	setting := serverConfigSettingFromRegistryV0(
		envServerIdleSelfImprovementGoalFirstV0,
		strconv.FormatBool(config.IdleSelfImprovementGoalFirst),
	)
	if strings.TrimSpace(os.Getenv(envServerIdleSelfImprovementGoalFirstV0)) == "" &&
		codexGoalBackendOperationalFromEnvV0() {
		setting.Source = "derived_from_codex_goal_backend"
	}
	return setting
}

func serverEffectiveConfigDiagnosticsFromEnvV0() []orquestaserver.ServerDiagnosticV0 {
	diagnostics := []orquestaserver.ServerDiagnosticV0{}
	if !codexGoalBackendOperationalFromEnvV0() &&
		!boolEnvOrDefaultV0(envExternalWorkLegacyDirectorLoopV0, false) {
		diagnostics = append(diagnostics, orquestaserver.ServerDiagnosticV0{
			Code:    orquestamcp.MCPExternalWorkRunGoalBackendRequiredV0,
			Scope:   "external_work",
			Message: "external_work goal-first no ejecutable sin " + envCodexGoalBackendV0 + "; export " + envCodexGoalBackendV0 + "=" + codexGoalBackendAppServerTmuxV0 + " y reinicia el servidor",
			EvidenceRefs: []string{
				"evidence-ref-server-external-work-goal-backend-required",
			},
		})
	}
	if strings.TrimSpace(os.Getenv(envServerIdleSelfImprovementGoalFirstV0)) == "" &&
		codexGoalBackendOperationalFromEnvV0() {
		diagnostics = append(diagnostics, orquestaserver.ServerDiagnosticV0{
			Code:         "idle_self_improvement_goal_first_derived",
			Scope:        "autoprogramming",
			Message:      envServerIdleSelfImprovementGoalFirstV0 + " derivada de " + envCodexGoalBackendV0,
			EvidenceRefs: []string{"evidence-ref-server-idle-self-improvement-goal-first-derived"},
		})
	}
	legacy := strings.TrimSpace(os.Getenv(envServerIdleSelfImprovementAfterLegacyV0))
	if legacy == "" {
		return diagnostics
	}
	if strings.TrimSpace(os.Getenv(envServerIdleSelfImprovementAfterV0)) != "" {
		return append(diagnostics, orquestaserver.ServerDiagnosticV0{
			Code:         "legacy_env_ignored",
			Scope:        "autoprogramming",
			Message:      envServerIdleSelfImprovementAfterLegacyV0 + " ignorada porque " + envServerIdleSelfImprovementAfterV0 + " esta definida",
			EvidenceRefs: []string{"evidence-ref-server-idle-self-improvement-env-legacy-ignored"},
		})
	}
	return append(diagnostics, orquestaserver.ServerDiagnosticV0{
		Code:         "legacy_env_alias",
		Scope:        "autoprogramming",
		Message:      envServerIdleSelfImprovementAfterLegacyV0 + " es legacy; usar " + envServerIdleSelfImprovementAfterV0,
		EvidenceRefs: []string{"evidence-ref-server-idle-self-improvement-env-legacy-alias"},
	})
}

func serverIdleSelfImprovementAfterLegacyActiveV0() bool {
	return strings.TrimSpace(os.Getenv(envServerIdleSelfImprovementAfterV0)) == "" &&
		strings.TrimSpace(os.Getenv(envServerIdleSelfImprovementAfterLegacyV0)) != ""
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
