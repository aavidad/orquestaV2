package main

import (
	"os"
	"strconv"
	"strings"
	"time"

	orquestamcp "orquesta/modulos/orquesta-mcp"
	orquestaserver "orquesta/modulos/orquesta-server"
)

func init() {
	serverEffectiveEnvRegistryV0[envServerIdleSelfImprovementAfterLegacyV0] = serverEnvSettingMetadataV0{
		Scope:       "autoprogramming",
		Label:       "Espera idle legacy",
		Description: "Alias legacy de la espera idle; se conserva solo para diagnostico y compatibilidad.",
	}
}

func serverEffectiveConfigFromEnvV0(config orquestaserver.ConfigV0) orquestaserver.ServerEffectiveConfigV0 {
	config = orquestaserver.NormalizeConfigV0(config)
	projectConfig := projectConfigFromServerConfigBestEffortV0(config)
	codexRuntime := codexRuntimeEnvConfigFromProjectFileV0(projectConfig)
	stackCapacity := codexStackCapacityEnvConfigFromProjectConfigFileV0(projectConfig)
	directorWaveLimits := codexDirectorWaveLimitsFromProjectConfigFileV0(projectConfig)
	egressSanitizer := egressSanitizerConfigFromProjectConfigFileV0(projectConfig)
	worktreeSnapshotBudget := codexServerWorktreeSnapshotReadBudgetFromProjectConfigV0(config.ProjectWorkDir)
	daemonEnvPolicy := serverDaemonStartEnvPolicyV0(os.Environ(), config)
	goalProgressPolicy := serverAutoprogrammingGoalProgressPolicyFromConfigV0(config)
	settings := []orquestaserver.ServerConfigSettingV0{
		serverOrquestaBaseURLSettingV0(),
		serverSensitiveConfigSettingFromRegistryV0(
			envOrquestaRuntimeDirV0,
			configuredEnvValueV0(envOrquestaRuntimeDirV0, "orquesta-runtime-dir-configured"),
		),
		serverConfigSettingFromRegistryWithSourceV0(
			envServerAddrV0,
			config.Addr,
			configSettingSourceFromConfigOrProjectConfigV0(config, envServerAddrV0),
		),
		serverSensitiveConfigSettingFromRegistryWithSourceV0(
			envServerStateDirV0,
			configuredRefValueV0(config.StateDir, "server-state-dir-configured"),
			configSettingSourceFromConfigOrProjectConfigV0(config, envServerStateDirV0),
		),
		serverConfigSettingFromRegistryWithSourceV0(
			envServerAuditFileV0,
			config.AuditFile,
			configSettingSourceFromConfigOrProjectConfigV0(config, envServerAuditFileV0),
		),
		serverConfigSettingFromRegistryWithSourceV0(
			envServerAuditDisabledV0,
			strconv.FormatBool(config.AuditDisabled),
			configSettingSourceFromConfigOrProjectConfigV0(config, envServerAuditDisabledV0),
		),
		serverConfigSettingFromRegistryWithSourceV0(
			envServerRemoteControlPlaneConfirmV0,
			strconv.FormatBool(config.ControlPlane.RemoteAccessOptIn),
			configSettingSourceFromConfigOrProjectConfigV0(config, envServerRemoteControlPlaneConfirmV0),
		),
		serverSensitiveConfigSettingFromRegistryWithSourceV0(
			envServerControlTokenV0,
			tokenPresenceV0(config.ControlPlane.Token),
			configSettingSourceFromConfigOrProjectConfigV0(config, envServerControlTokenV0),
		),
		serverConfigSettingFromRegistryWithSourceV0(
			envServerControlPrincipalV0,
			config.ControlPlane.Principal,
			configSettingSourceFromConfigOrProjectConfigV0(config, envServerControlPrincipalV0),
		),
		serverConfigSettingFromRegistryWithSourceV0(
			envServerControlPermissionRefV0,
			config.ControlPlane.PermissionRef,
			configSettingSourceFromConfigOrProjectConfigV0(config, envServerControlPermissionRefV0),
		),
		serverConfigSettingFromRegistryWithSourceV0(
			envServerControlPublicReasonV0,
			config.ControlPlane.PublicReason,
			configSettingSourceFromConfigOrProjectConfigV0(config, envServerControlPublicReasonV0),
		),
		serverConfigSettingFromRegistryV0(envServerSelfProgrammingOnlyV0, strconv.FormatBool(serverSelfProgrammingOnlyFromEnvV0())),
		serverSensitiveConfigSettingFromRegistryV0(
			envServerSelfProgrammingRootV0,
			configuredEnvValueV0(envServerSelfProgrammingRootV0, "self-programming-root-configured"),
		),
		serverPositiveConfigSettingFromConfigV0(config, envServerMaxRunsPerTickV0, config.SupervisorCommand.MaxRunsPerTick),
		serverPositiveConfigSettingFromConfigV0(config, envServerMaxExecutionsPerTickV0, config.SupervisorCommand.MaxExecutions),
		serverPositiveConfigSettingFromConfigV0(config, envServerQueueLimitV0, serverRunQueueLimitFromProjectConfigFileV0(projectConfig)),
		serverPositiveConfigSettingFromConfigV0(config, envServerDrainMaxDispatchesV0, config.SupervisorCommand.DrainLimits.MaxDispatchesPerWait),
		serverPositiveConfigSettingFromConfigV0(config, envServerDrainMaxCommandsV0, config.SupervisorCommand.DrainLimits.MaxCommands),
		serverPositiveConfigSettingFromConfigV0(config, envServerDrainMaxOutboxV0, config.SupervisorCommand.DrainLimits.MaxOutboxPerCycle),
		serverPositiveConfigSettingFromConfigV0(config, envServerDrainMaxExternalWaitsV0, config.SupervisorCommand.DrainLimits.MaxExternalWaits),
		serverConfigSettingFromRegistryV0(envServerAutonomyEnabledV0, strconv.FormatBool(serverAutonomyEffectiveEnabledFromEnvV0())),
		serverConfigSettingFromRegistryV0(envServerGoalObserverEnabledV0, strconv.FormatBool(config.GoalObserverEnabled)),
		serverConfigSettingFromRegistryV0(envServerGoalObserverIntervalMSV0, strconv.Itoa(int(config.GoalObserverInterval/time.Millisecond))),
		serverConfigSettingFromRegistryV0(envServerGoalObserverMaxItemsV0, strconv.Itoa(config.GoalObserverMaxItems)),
		serverConfigSettingFromRegistryV0(
			envServerGoalObserverFingerprintEnabledV0,
			strconv.FormatBool(serverGoalObserverFingerprintEnabledFromEnvV0()),
		),
		serverConfigSettingFromRegistryV0(envServerResidentDirectorEnabledV0, strconv.FormatBool(config.ResidentDirectorEnabled)),
		serverPositiveConfigSettingFromConfigV0(config, envServerResidentDirectorMaxActionsV0, config.ResidentDirectorMaxActions),
		serverConfigSettingFromRegistryV0(envServerEscalationDirectorEnabledV0, strconv.FormatBool(config.EscalationDirectorEnabled)),
		serverSensitiveConfigSettingFromRegistryV0(
			envServerEscalationDirectorCommandV0,
			configuredRefValueV0(strings.Join(config.EscalationDirectorCommand, ","), "escalation-director-command-configured"),
		),
		serverConfigSettingFromRegistryV0(envServerEscalationDirectorTimeoutSecondsV0, strconv.Itoa(int(config.EscalationDirectorTimeout/time.Second))),
		serverConfigSettingFromRegistryV0(envServerEscalationDirectorMaxPerDayV0, strconv.Itoa(config.EscalationDirectorMaxPerDay)),
		serverConfigSettingFromRegistryV0(envServerTickIntervalMSV0, strconv.Itoa(int(config.TickInterval/time.Millisecond))),
		serverConfigSettingFromRegistryWithSourceV0(
			envServerShutdownGraceMSV0,
			strconv.Itoa(int(config.ShutdownGracePeriod/time.Millisecond)),
			configSettingSourceFromConfigOrProjectConfigV0(config, envServerShutdownGraceMSV0),
		),
		serverConfigSettingFromRegistryWithSourceV0(
			envServerReadHeaderTimeoutMSV0,
			strconv.Itoa(int(config.HTTPResourceLimits.ReadHeaderTimeout/time.Millisecond)),
			configSettingSourceFromConfigOrProjectConfigV0(config, envServerReadHeaderTimeoutMSV0),
		),
		serverConfigSettingFromRegistryWithSourceV0(
			envServerReadTimeoutMSV0,
			strconv.Itoa(int(config.HTTPResourceLimits.ReadTimeout/time.Millisecond)),
			configSettingSourceFromConfigOrProjectConfigV0(config, envServerReadTimeoutMSV0),
		),
		serverConfigSettingFromRegistryWithSourceV0(
			envServerWriteTimeoutMSV0,
			strconv.Itoa(int(config.HTTPResourceLimits.WriteTimeout/time.Millisecond)),
			configSettingSourceFromConfigOrProjectConfigV0(config, envServerWriteTimeoutMSV0),
		),
		serverConfigSettingFromRegistryWithSourceV0(
			envServerIdleTimeoutMSV0,
			strconv.Itoa(int(config.HTTPResourceLimits.IdleTimeout/time.Millisecond)),
			configSettingSourceFromConfigOrProjectConfigV0(config, envServerIdleTimeoutMSV0),
		),
		serverConfigSettingFromRegistryWithSourceV0(
			envServerMaxHeaderBytesV0,
			strconv.Itoa(config.HTTPResourceLimits.MaxHeaderBytes),
			configSettingSourceFromConfigOrProjectConfigV0(config, envServerMaxHeaderBytesV0),
		),
		serverConfigSettingFromRegistryWithSourceV0(
			envServerControlBodyMaxBytesV0,
			strconv.FormatInt(config.HTTPResourceLimits.ControlBodyBytes, 10),
			configSettingSourceFromConfigOrProjectConfigV0(config, envServerControlBodyMaxBytesV0),
		),
		serverConfigSettingFromRegistryV0(envServerSelfWatchdogDisabledV0, strconv.FormatBool(config.SelfWatchdog.Disabled)),
		serverConfigSettingFromRegistryV0(envServerSelfWatchdogCPUHighPercentV0, strconv.Itoa(config.SelfWatchdog.CPUHighPercent)),
		serverConfigSettingFromRegistryV0(envServerSelfWatchdogSustainedSecondsV0, strconv.Itoa(int(config.SelfWatchdog.SustainedFor/time.Second))),
		serverConfigSettingFromRegistryV0(envServerSelfWatchdogNoProgressSecondsV0, strconv.Itoa(int(config.SelfWatchdog.NoProgressFor/time.Second))),
		serverConfigSettingFromRegistryWithSourceV0(
			envServerIdleSelfImprovementDisabledV0,
			strconv.FormatBool(config.IdleSelfImprovementDisabled),
			configSettingSourceFromConfigOrProjectConfigV0(config, envServerIdleSelfImprovementDisabledV0),
		),
		serverIdleSelfImprovementAfterSettingV0(config),
		serverConfigSettingFromRegistryWithSourceV0(
			envServerIdleSelfImprovementTargetQueueV0,
			strconv.Itoa(config.IdleSelfImprovementTargetQueue),
			configSettingSourceFromConfigOrProjectConfigV0(config, envServerIdleSelfImprovementTargetQueueV0),
		),
		serverIdleSelfImprovementGoalFirstSettingV0(config, projectConfig),
		serverConfigSettingFromRegistryWithSourceV0(
			envServerIdleSelfImprovementFrozenTestsV0,
			strconv.FormatBool(config.IdleSelfImprovementFrozenTests),
			configSettingSourceFromConfigOrProjectConfigV0(config, envServerIdleSelfImprovementFrozenTestsV0),
		),
		serverSensitiveConfigSettingFromRegistryWithSourceV0(
			envServerIdleSelfImprovementProjectWorkDirV0,
			configuredRefValueV0(config.IdleSelfImprovementProjectWorkDir, "idle-self-improvement-project-workdir-configured"),
			configSettingSourceFromConfigOrProjectConfigV0(config, envServerIdleSelfImprovementProjectWorkDirV0),
		),
		serverConfigSettingFromRegistryWithSourceV0(envServerIdleSelfImprovementProjectRefV0, config.IdleSelfImprovementProjectRef, configSettingSourceFromConfigOrProjectConfigV0(config, envServerIdleSelfImprovementProjectRefV0)),
		serverConfigSettingFromRegistryWithSourceV0(envServerIdleSelfImprovementWorktreeRefV0, config.IdleSelfImprovementWorktreeRef, configSettingSourceFromConfigOrProjectConfigV0(config, envServerIdleSelfImprovementWorktreeRefV0)),
		serverConfigSettingFromRegistryWithSourceV0(envServerIdleSelfImprovementBranchRefV0, config.IdleSelfImprovementBranchRef, configSettingSourceFromConfigOrProjectConfigV0(config, envServerIdleSelfImprovementBranchRefV0)),
		serverConfigSettingFromRegistryWithSourceV0(envServerIdleSelfImprovementAreaV0, config.IdleSelfImprovementSuggestedArea, configSettingSourceFromConfigOrProjectConfigV0(config, envServerIdleSelfImprovementAreaV0)),
		serverConfigSettingFromRegistryWithSourceV0(envServerIdleSelfImprovementWriteSetV0, strings.Join(config.IdleSelfImprovementWriteSet, ","), configSettingSourceFromConfigOrProjectConfigV0(config, envServerIdleSelfImprovementWriteSetV0)),
		serverConfigSettingFromRegistryWithSourceV0(envServerIdleSelfImprovementRequiredTestsV0, strings.Join(config.IdleSelfImprovementRequiredTests, ","), configSettingSourceFromConfigOrProjectConfigV0(config, envServerIdleSelfImprovementRequiredTestsV0)),
		serverConfigSettingFromRegistryWithSourceV0(envServerIdleSelfImprovementContextRefsV0, strings.Join(config.IdleSelfImprovementContextRefs, ","), configSettingSourceFromConfigOrProjectConfigV0(config, envServerIdleSelfImprovementContextRefsV0)),
		serverConfigSettingFromRegistryWithSourceV0(envServerIdleSelfImprovementEvidenceRefsV0, strings.Join(config.IdleSelfImprovementEvidenceRefs, ","), configSettingSourceFromConfigOrProjectConfigV0(config, envServerIdleSelfImprovementEvidenceRefsV0)),
		serverConfigSettingFromRegistryWithSourceV0(envServerIdleSelfImprovementAcceptanceV0, strings.Join(config.IdleSelfImprovementAcceptance, ","), configSettingSourceFromConfigOrProjectConfigV0(config, envServerIdleSelfImprovementAcceptanceV0)),
		serverConfigSettingFromRegistryWithSourceV0(envServerIdleSelfImprovementCompactRulesV0, strings.Join(config.IdleSelfImprovementCompactRules, ","), configSettingSourceFromConfigOrProjectConfigV0(config, envServerIdleSelfImprovementCompactRulesV0)),
		serverPositiveConfigSettingFromConfigV0(config, envServerIdleSelfImprovementPriorityScoreV0, config.IdleSelfImprovementPriorityScore),
		serverPositiveConfigSettingFromConfigV0(config, envServerIdleSelfImprovementMaxRequestsV0, config.IdleSelfImprovementMaxRequests),
		serverConfigSettingFromRegistryWithSourceV0(envServerIdleSelfImprovementDailyGoalBudgetV0, strconv.Itoa(config.IdleSelfImprovementBudget.MaxGoalsPerDay), configSettingSourceFromConfigOrProjectConfigV0(config, envServerIdleSelfImprovementDailyGoalBudgetV0)),
		serverConfigSettingFromRegistryWithSourceV0(envServerIdleSelfImprovementDailyContextBudgetBytesV0, strconv.FormatInt(config.IdleSelfImprovementBudget.MaxContextBudgetBytesPerDay, 10), configSettingSourceFromConfigOrProjectConfigV0(config, envServerIdleSelfImprovementDailyContextBudgetBytesV0)),
		serverConfigSettingFromRegistryV0(envSelfAuditBacklogEnabledV0, strconv.FormatBool(config.SelfAuditBacklogEnabled)),
		serverConfigSettingFromRegistryWithSourceV0(
			envServerDaemonLogMaxBytesV0,
			strconv.FormatInt(config.DaemonLogPolicy.MaxBytes, 10),
			configSettingSourceFromConfigOrProjectConfigV0(config, envServerDaemonLogMaxBytesV0),
		),
		serverConfigSettingFromRegistryWithSourceV0(
			envServerDaemonLogMaxRotatedV0,
			strconv.Itoa(config.DaemonLogPolicy.MaxRotatedFiles),
			configSettingSourceFromConfigOrProjectConfigV0(config, envServerDaemonLogMaxRotatedV0),
		),
		serverConfigSettingFromRegistryWithSourceV0(
			envServerDaemonLogRetentionDaysV0,
			strconv.Itoa(config.DaemonLogPolicy.RetentionDays),
			configSettingSourceFromConfigOrProjectConfigV0(config, envServerDaemonLogRetentionDaysV0),
		),
		serverConfigSettingFromRegistryWithSourceV0(
			envServerDaemonLogRawEnabledV0,
			strconv.FormatBool(config.DaemonLogPolicy.LocalRawEnabled),
			configSettingSourceFromConfigOrProjectConfigV0(config, envServerDaemonLogRawEnabledV0),
		),
		serverConfigSettingFromRegistryWithSourceV0(
			envServerDaemonLogRawReasonV0,
			config.DaemonLogPolicy.LocalRawReason,
			configSettingSourceFromConfigOrProjectConfigV0(config, envServerDaemonLogRawReasonV0),
		),
		serverConfigSettingFromRegistryWithSourceV0(
			envSecurityModeV0,
			serverSecurityModeEffectiveValueFromProjectConfigFileV0(projectConfig),
			configSettingSourceFromConfigOrProjectConfigV0(config, envSecurityModeV0),
		),
		serverConfigSettingFromRegistryWithSourceV0(
			envRailsModeV0,
			serverRailsModeEffectiveValueFromProjectConfigFileV0(projectConfig),
			configSettingSourceFromConfigOrProjectConfigV0(config, envRailsModeV0),
		),
		serverConfigSettingFromRegistryWithSourceV0(
			envDetailProhibitedRailsV0,
			serverDetailRailsEffectiveValueFromProjectConfigFileV0(projectConfig),
			configSettingSourceFromConfigOrProjectConfigV0(config, envDetailProhibitedRailsV0),
		),
		serverConfigSettingFromRegistryWithSourceV0(
			envDetailProhibitedRailsScopeV0,
			serverDetailRailsScopeEffectiveValueFromProjectConfigFileV0(projectConfig),
			configSettingSourceFromConfigOrProjectConfigV0(config, envDetailProhibitedRailsScopeV0),
		),
		serverConfigSettingFromRegistryWithSourceV0(
			envCodexExecutionModeV0,
			codexExecutionModeFromProjectConfigFileV0(projectConfig),
			configSettingSourceFromConfigOrProjectConfigV0(config, envCodexExecutionModeV0),
		),
		serverPositiveConfigSettingFromConfigV0(config, envCodexMaxBatchReadyV0, codexRuntime.Limits.MaxBatchReady),
		serverPositiveConfigSettingFromConfigV0(config, envCodexMaxConcurrencyV0, codexRuntime.Limits.MaxLiveProcesses),
		serverConfigSettingFromRegistryWithSourceV0(
			envCodexReasoningEffortV0,
			codexRuntime.ReasoningEffort,
			configSettingSourceFromConfigOrProjectConfigV0(config, envCodexReasoningEffortV0),
		),
		serverConfigSettingWithSourceV0(
			"codex_model_routing.policy_ref",
			codexModelRoutingFromProjectConfigFileV0(projectConfig).Policy.PolicyRef,
			codexModelRoutingConfigSourceV0(projectConfig),
			"codex_model_routing", "Política de modelo Codex", "Ref opaca de la política tipada de selección de modelo.",
		),
		serverConfigSettingWithSourceV0(
			"codex_model_routing.alias_refs",
			strings.Join(codexModelRoutingAliasRefsV0(codexModelRoutingFromProjectConfigFileV0(projectConfig)), ","),
			codexModelRoutingConfigSourceV0(projectConfig),
			"codex_model_routing", "Aliases de modelo Codex", "Solo refs de alias; nunca nombres de modelo ni secretos.",
		),
		serverConfigSettingWithSourceV0(
			"codex_model_routing.efforts",
			codexModelRoutingEffortsSummaryV0(codexModelRoutingFromProjectConfigFileV0(projectConfig)),
			codexModelRoutingConfigSourceV0(projectConfig),
			"codex_model_routing", "Esfuerzos de modelo Codex", "Esfuerzo explícito por clase de tarea.",
		),
		serverConfigSettingWithSourceV0(
			"claude_model_routing.policy_ref",
			claudeModelRoutingFromProjectConfigFileV0(projectConfig).Policy.PolicyRef,
			claudeModelRoutingConfigSourceV0(projectConfig),
			"claude_model_routing", "Política de modelo Claude", "Ref opaca de la política tipada de selección de modelo.",
		),
		serverConfigSettingWithSourceV0(
			"claude_model_routing.alias_refs",
			strings.Join(claudeModelRoutingAliasRefsV0(claudeModelRoutingFromProjectConfigFileV0(projectConfig)), ","),
			claudeModelRoutingConfigSourceV0(projectConfig),
			"claude_model_routing", "Aliases de modelo Claude", "Solo refs de alias; nunca nombres de modelo ni secretos.",
		),
		serverConfigSettingWithSourceV0(
			"claude_model_routing.efforts",
			claudeModelRoutingEffortsSummaryV0(claudeModelRoutingFromProjectConfigFileV0(projectConfig)),
			claudeModelRoutingConfigSourceV0(projectConfig),
			"claude_model_routing", "Esfuerzos de modelo Claude", "Esfuerzo explícito por clase de tarea.",
		),
		serverPositiveConfigSettingFromConfigV0(config, envCodexMaxExpectedSecondsV0, int(codexRuntime.ProgressBudget.MaxExpected/time.Second)),
		serverConfigSettingFromRegistryWithSourceV0(
			envCodexUsageAccountingV0,
			codexUsageAccountingModeFromProjectConfigFileV0(projectConfig),
			configSettingSourceFromConfigOrProjectConfigV0(config, envCodexUsageAccountingV0),
		),
		serverConfigSettingFromRegistryWithSourceV0(
			envCodexUsageLogMaxBytesV0,
			strconv.FormatInt(codexUsageLogMaxBytesFromProjectConfigFileV0(projectConfig), 10),
			configSettingSourceFromConfigOrProjectConfigV0(config, envCodexUsageLogMaxBytesV0),
		),
		serverSensitiveConfigSettingFromRegistryWithSourceV0(
			envCodexRuntimeWorkDirV0,
			configuredRefValueV0(config.RuntimeWorkDir, "codex-runtime-workdir-configured"),
			configSettingSourceFromConfigOrProjectConfigV0(config, envCodexRuntimeWorkDirV0),
		),
		serverSensitiveConfigSettingFromRegistryV0(
			envCodexHomeV0,
			configuredEnvValueV0(envCodexHomeV0, "codex-process-home-configured"),
		),
		serverCodexCodeHomeSettingV0(),
		serverConfigSettingFromRegistryV0(
			envCodexPromoteMaterializedArtifactWithoutAckV0,
			strconv.FormatBool(codexPromoteMaterializedArtifactWithoutAckFromEnvV0()),
		),
		serverConfigSettingFromRegistryWithSourceV0(
			envCodexGoalBackendV0,
			codexGoalBackendFromProjectConfigFileV0(projectConfig),
			configSettingSourceFromConfigOrProjectConfigV0(config, envCodexGoalBackendV0),
		),
		serverConfigSettingFromRegistryWithSourceV0(
			envAllowAppServerProxyDiagnosticV0,
			strconv.FormatBool(codexGoalBackendProxyDiagnosticAllowedFromProjectConfigFileV0(projectConfig)),
			configSettingSourceFromConfigOrProjectConfigV0(config, envAllowAppServerProxyDiagnosticV0),
		),
		serverConfigSettingFromRegistryWithSourceV0(
			envCodexGoalTimeoutMSV0,
			strconv.Itoa(codexGoalTimeoutMSFromProjectConfigFileV0(projectConfig)),
			configSettingSourceFromConfigOrProjectConfigV0(config, envCodexGoalTimeoutMSV0),
		),
		serverConfigSettingFromRegistryWithSourceV0(
			envCodexGoalPreflightTimeoutMSV0,
			strconv.Itoa(codexGoalPreflightTimeoutMSFromProjectConfigFileV0(projectConfig)),
			configSettingSourceFromConfigOrProjectConfigV0(config, envCodexGoalPreflightTimeoutMSV0),
		),
		serverConfigSettingFromRegistryV0(
			envAutoprogrammingLegacyDirectorLoopV0,
			strconv.FormatBool(boolEnvOrDefaultV0(envAutoprogrammingLegacyDirectorLoopV0, false)),
		),
		serverConfigSettingFromRegistryWithSourceV0(
			envAutoprogrammingCheckpointOnlyHighConsumptionTokensV0,
			strconv.FormatInt(goalProgressPolicy.CheckpointOnlyHighConsumptionTokens, 10),
			configSettingSourceFromConfigOrProjectConfigV0(config, envAutoprogrammingCheckpointOnlyHighConsumptionTokensV0),
		),
		serverConfigSettingFromRegistryWithSourceV0(
			envAutoprogrammingCheckpointOnlyMaxWaitSecondsV0,
			strconv.FormatInt(goalProgressPolicy.CheckpointOnlyMaxWaitSeconds, 10),
			configSettingSourceFromConfigOrProjectConfigV0(config, envAutoprogrammingCheckpointOnlyMaxWaitSecondsV0),
		),
		serverConfigSettingFromRegistryWithSourceV0(
			envAutoprogrammingNoCheckpointWarningMaxWaitSecondsV0,
			strconv.FormatInt(goalProgressPolicy.NoCheckpointWarningMaxWaitSeconds, 10),
			configSettingSourceFromConfigOrProjectConfigV0(config, envAutoprogrammingNoCheckpointWarningMaxWaitSecondsV0),
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
		serverConfigSettingFromRegistryV0(envCapacityReasoningEffortV0, string(stackCapacity.ReasoningEffort)),
		serverConfigSettingFromRegistryV0(envCapacityPolicyRefV0, stackCapacity.PolicyRef),
		serverConfigSettingFromRegistryV0(envCapacityPoolRefV0, stackCapacity.PoolRef),
		serverConfigSettingFromRegistryWithSourceV0(
			envEgressSanitizerEnabledV0,
			strconv.FormatBool(egressSanitizer.Enabled),
			configSettingSourceFromConfigOrProjectConfigV0(config, envEgressSanitizerEnabledV0),
		),
		serverConfigSettingFromRegistryWithSourceV0(
			envEgressSanitizerRefV0,
			egressSanitizer.SanitizerRef,
			configSettingSourceFromConfigOrProjectConfigV0(config, envEgressSanitizerRefV0),
		),
		serverConfigSettingFromRegistryWithSourceV0(
			envEgressSanitizerLocalModelEnabledV0,
			strconv.FormatBool(egressSanitizer.LocalModel.Enabled),
			configSettingSourceFromConfigOrProjectConfigV0(config, envEgressSanitizerLocalModelEnabledV0),
		),
		serverSensitiveConfigSettingFromRegistryWithSourceV0(
			envEgressSanitizerLocalModelRefV0,
			sensitiveConfigValueFromSourceV0(
				egressSanitizer.LocalModel.ModelRef,
				"egress-sanitizer-local-model-ref-configured",
				configSettingSourceFromConfigOrProjectConfigV0(config, envEgressSanitizerLocalModelRefV0),
			),
			configSettingSourceFromConfigOrProjectConfigV0(config, envEgressSanitizerLocalModelRefV0),
		),
		serverSensitiveConfigSettingFromRegistryWithSourceV0(
			envEgressSanitizerLocalModelPathV0,
			sensitiveConfigValueFromSourceV0(
				stringProjectConfigOrEnvOrDefaultV0(envEgressSanitizerLocalModelPathV0, projectConfig.EgressSanitizer.LocalModel.ModelPath, ""),
				"egress-sanitizer-local-model-path-configured",
				configSettingSourceFromConfigOrProjectConfigV0(config, envEgressSanitizerLocalModelPathV0),
			),
			configSettingSourceFromConfigOrProjectConfigV0(config, envEgressSanitizerLocalModelPathV0),
		),
		serverSensitiveConfigSettingFromRegistryWithSourceV0(
			envEgressSanitizerLocalRuntimeRefV0,
			sensitiveConfigValueFromSourceV0(
				egressSanitizer.LocalModel.RuntimeRef,
				"egress-sanitizer-local-runtime-ref-configured",
				configSettingSourceFromConfigOrProjectConfigV0(config, envEgressSanitizerLocalRuntimeRefV0),
			),
			configSettingSourceFromConfigOrProjectConfigV0(config, envEgressSanitizerLocalRuntimeRefV0),
		),
		serverSensitiveConfigSettingFromRegistryWithSourceV0(
			envEgressSanitizerLocalEvidenceRefV0,
			sensitiveConfigValueFromSourceV0(
				egressSanitizer.LocalModel.EvidenceRef,
				"egress-sanitizer-local-evidence-ref-configured",
				configSettingSourceFromConfigOrProjectConfigV0(config, envEgressSanitizerLocalEvidenceRefV0),
			),
			configSettingSourceFromConfigOrProjectConfigV0(config, envEgressSanitizerLocalEvidenceRefV0),
		),
		serverConfigSettingFromRegistryWithSourceV0(
			envEgressSanitizerSidecarEnabledV0,
			strconv.FormatBool(egressSanitizer.Sidecar.Enabled),
			configSettingSourceFromConfigOrProjectConfigV0(config, envEgressSanitizerSidecarEnabledV0),
		),
		serverConfigSettingFromRegistryWithSourceV0(
			envEgressSanitizerSidecarRefV0,
			egressSanitizer.Sidecar.SidecarRef,
			configSettingSourceFromConfigOrProjectConfigV0(config, envEgressSanitizerSidecarRefV0),
		),
		serverConfigSettingFromRegistryWithSourceV0(
			envEgressSanitizerSidecarAdapterRefV0,
			egressSanitizer.Sidecar.AdapterRef,
			configSettingSourceFromConfigOrProjectConfigV0(config, envEgressSanitizerSidecarAdapterRefV0),
		),
		serverConfigSettingFromRegistryWithSourceV0(
			envEgressSanitizerSidecarTransportRefV0,
			egressSanitizer.Sidecar.TransportRef,
			configSettingSourceFromConfigOrProjectConfigV0(config, envEgressSanitizerSidecarTransportRefV0),
		),
		serverConfigSettingFromRegistryWithSourceV0(
			envEgressSanitizerSidecarEvidenceRefV0,
			egressSanitizer.Sidecar.EvidenceRef,
			configSettingSourceFromConfigOrProjectConfigV0(config, envEgressSanitizerSidecarEvidenceRefV0),
		),
		serverSensitiveConfigSettingFromRegistryWithSourceV0(
			envEgressSanitizerSidecarCommandV0,
			sensitiveConfigValueFromSourceV0(
				egressSanitizerSidecarCommandFromProjectConfigFileV0(projectConfig),
				"egress-sanitizer-sidecar-command-configured",
				configSettingSourceFromConfigOrProjectConfigV0(config, envEgressSanitizerSidecarCommandV0),
			),
			configSettingSourceFromConfigOrProjectConfigV0(config, envEgressSanitizerSidecarCommandV0),
		),
		serverSensitiveConfigSettingFromRegistryWithSourceV0(
			envEgressSanitizerSidecarLocalEndpointV0,
			sensitiveConfigValueFromSourceV0(
				egressSanitizerSidecarLocalEndpointFromProjectConfigFileV0(projectConfig),
				"egress-sanitizer-sidecar-local-endpoint-configured",
				configSettingSourceFromConfigOrProjectConfigV0(config, envEgressSanitizerSidecarLocalEndpointV0),
			),
			configSettingSourceFromConfigOrProjectConfigV0(config, envEgressSanitizerSidecarLocalEndpointV0),
		),
		serverPositiveConfigSettingFromConfigV0(config, envCodexDirectorWaveAgentsV0, directorWaveLimits.Agents),
		serverPositiveConfigSettingFromConfigV0(config, envCodexDirectorMaxSubagentsPerAgentV0, directorWaveLimits.MaxSubagentsPerAgent),
		serverPositiveConfigSettingFromConfigV0(config, envCodexDirectorRecursiveAgentBudgetV0, directorWaveLimits.RecursiveAgentBudget),
	}
	settings = append(settings, codebaseBrokerEffectiveSettingsV0(config.ProjectWorkDir, projectConfig)...)
	settings = append(settings, geminiRuntimeEffectiveConfigSettingsV0(config, projectConfig)...)
	settings = append(settings, requiredTestRunnerEffectiveConfigSettingsV0(projectConfig)...)
	settings = append(settings, autoprogrammingPromotionEffectiveConfigSettingsV0(config, projectConfig)...)
	settings = append(settings, telegramOperatorEffectiveConfigSettingsV0(config, projectConfig)...)
	settings = append(settings, domainWorkEffectiveSettingsV0(config.ProjectWorkDir, projectConfig)...)
	settings = append(settings, opesBridgeEffectiveConfigSettingsV0(config, projectConfig)...)
	settings = append(settings, codexWaveEffectiveConfigSettingsV0(config, projectConfig)...)
	settings = append(settings, codexServerWorktreeSnapshotBudgetSettingsV0(config.ProjectWorkDir, worktreeSnapshotBudget)...)
	settings = append(settings, hermesOperatorEffectiveConfigSettingsV0()...)
	settings = append(settings, ollamaModelManagerEffectiveConfigSettingsV0()...)
	settings = append(settings, opesRegistryFinalPkgEffectiveConfigSettingsV0(config, projectConfig)...)
	settings = append(settings, opesTopicRegistryEffectiveConfigSettingsV0(config, projectConfig)...)
	settings = append(settings, daemonStartEnvSettingsV0(daemonEnvPolicy)...)
	return orquestaserver.NormalizeServerEffectiveConfigV0(orquestaserver.ServerEffectiveConfigV0{
		SchemaVersion: orquestaserver.ServerEffectiveConfigSchemaVersionV0,
		Settings:      settings,
		Diagnostics:   serverEffectiveConfigDiagnosticsFromConfigV0(projectConfig),
	})
}

func serverIdleSelfImprovementAfterSettingV0(config orquestaserver.ConfigV0) orquestaserver.ServerConfigSettingV0 {
	setting := serverConfigSettingFromRegistryWithSourceV0(
		envServerIdleSelfImprovementAfterV0,
		strconv.Itoa(int(config.IdleSelfImprovementAfter/time.Second)),
		configSettingSourceFromConfigOrProjectConfigV0(config, envServerIdleSelfImprovementAfterV0),
	)
	if serverIdleSelfImprovementAfterLegacyActiveV0() {
		setting.Source = "legacy_alias"
	}
	return setting
}

func serverIdleSelfImprovementGoalFirstSettingV0(
	config orquestaserver.ConfigV0,
	projectConfig serverProjectConfigFileV0,
) orquestaserver.ServerConfigSettingV0 {
	setting := serverConfigSettingFromRegistryV0(
		envServerIdleSelfImprovementGoalFirstV0,
		strconv.FormatBool(config.IdleSelfImprovementGoalFirst),
	)
	if strings.TrimSpace(os.Getenv(envServerIdleSelfImprovementGoalFirstV0)) == "" &&
		!serverIdleProjectConfigHasValueForEnvKeyV0(projectConfig, envServerIdleSelfImprovementGoalFirstV0) &&
		serverGoalBackendOperationalFromProjectConfigFileV0(projectConfig) {
		setting.Source = serverGoalBackendDerivationSourceFromProjectConfigFileV0(projectConfig)
	} else {
		setting.Source = configSettingSourceFromConfigOrProjectConfigV0(config, envServerIdleSelfImprovementGoalFirstV0)
	}
	return setting
}

func serverOrquestaBaseURLSettingV0() orquestaserver.ServerConfigSettingV0 {
	setting := serverSensitiveConfigSettingFromRegistryV0(
		envOrquestaServerURLV0,
		configuredRefValueV0(firstNonEmptyEnvV0(envOrquestaServerURLV0, envOrquestaBaseURLV0), "orquesta-server-url-configured"),
	)
	if configSettingSourceFromEnvOrLegacyV0(envOrquestaServerURLV0, envOrquestaBaseURLV0) == "legacy_alias" {
		setting.Source = "legacy_alias"
	}
	return setting
}

func serverCodexCodeHomeSettingV0() orquestaserver.ServerConfigSettingV0 {
	setting := serverSensitiveConfigSettingFromRegistryV0(
		envCodexCodeHomeV0,
		configuredRefValueV0(firstNonEmptyEnvV0(envCodexCodeHomeV0, envCodexHomeV0, envCodexCodeHomeLegacyV0), "codex-code-home-configured"),
	)
	if configSettingSourceFromEnvOrLegacyV0(envCodexCodeHomeV0, envCodexHomeV0, envCodexCodeHomeLegacyV0) == "legacy_alias" {
		setting.Source = "legacy_alias"
	}
	return setting
}

func serverEffectiveConfigDiagnosticsFromConfigV0(projectConfig serverProjectConfigFileV0) []orquestaserver.ServerDiagnosticV0 {
	diagnostics := []orquestaserver.ServerDiagnosticV0{}
	diagnostics = append(diagnostics, serverEnvAliasDiagnosticsV0(
		envOrquestaServerURLV0,
		"server_endpoint",
		"evidence-ref-config-alias-orquesta-server-url",
		envOrquestaBaseURLV0,
	)...)
	diagnostics = append(diagnostics, serverEnvAliasDiagnosticsV0(
		envCodexCodeHomeV0,
		"codex_runtime",
		"evidence-ref-config-alias-codex-code-home",
		envCodexHomeV0,
		envCodexCodeHomeLegacyV0,
	)...)
	diagnostics = append(diagnostics, serverEnvAliasDiagnosticsV0(
		envOPESBaseURLV0,
		"opes_bridge",
		"evidence-ref-config-alias-opes-base-url",
		envOPESBaseURLLegacyV0,
	)...)
	diagnostics = append(diagnostics, serverDeprecatedEnvOverridesForProjectConfigV0(
		projectConfig,
		"control_plane",
		"control_plane.*",
		"evidence-ref-config-deprecated-env-override-control-plane",
		controlPlaneEnvKeysV0()...,
	)...)
	diagnostics = append(diagnostics, serverDeprecatedEnvOverridesForProjectConfigV0(
		projectConfig,
		"opes_bridge",
		"opes_bridge.*",
		"evidence-ref-config-deprecated-env-override-opes-bridge",
		opesBridgeEnvKeysV0()...,
	)...)
	diagnostics = append(diagnostics, serverDeprecatedEnvOverridesForProjectConfigV0(
		projectConfig,
		"codex_wave",
		"codex_wave.*",
		"evidence-ref-config-deprecated-env-override-codex-wave",
		codexWaveEnvKeysV0()...,
	)...)
	diagnostics = append(diagnostics, serverDeprecatedEnvOverridesForProjectConfigV0(
		projectConfig,
		"server_idle",
		"server_idle.*",
		"evidence-ref-config-deprecated-env-override-server-idle",
		serverIdleEnvKeysV0()...,
	)...)
	if !serverGoalBackendOperationalFromProjectConfigFileV0(projectConfig) &&
		!boolEnvOrDefaultV0(envExternalWorkLegacyDirectorLoopV0, false) {
		diagnostics = append(diagnostics, orquestaserver.ServerDiagnosticV0{
			Code:    orquestamcp.MCPExternalWorkRunGoalBackendRequiredV0,
			Scope:   "external_work",
			Message: "external_work goal-first no ejecutable sin backend goal; export " + envCodexGoalBackendV0 + "=" + codexGoalBackendAppServerTmuxV0 + ", " + envCodexGoalBackendV0 + "=" + claudeGoalBackendFileControlV0 + ", " + envCodexGoalBackendV0 + "=" + claudeGoalBackendProcessV0 + ", " + envCodexGoalBackendV0 + "=" + geminiGoalBackendFileControlV0 + " o " + envCodexGoalBackendV0 + "=" + geminiGoalBackendProcessV0 + " y reinicia el servidor",
			EvidenceRefs: []string{
				"evidence-ref-server-external-work-goal-backend-required",
			},
		})
	}
	if strings.TrimSpace(os.Getenv(envServerIdleSelfImprovementGoalFirstV0)) == "" &&
		serverGoalBackendOperationalFromProjectConfigFileV0(projectConfig) {
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

func serverDeprecatedEnvOverridesForProjectConfigV0(
	projectConfig serverProjectConfigFileV0,
	scope string,
	configPath string,
	evidenceRef string,
	keys ...string,
) []orquestaserver.ServerDiagnosticV0 {
	diagnostics := []orquestaserver.ServerDiagnosticV0{}
	for _, key := range keys {
		key = strings.TrimSpace(key)
		if key == "" || strings.TrimSpace(os.Getenv(key)) == "" {
			continue
		}
		if !serverProjectConfigHasEffectiveValueForEnvKeyV0(projectConfig, key) {
			continue
		}
		diagnostics = append(diagnostics, orquestaserver.ServerDiagnosticV0{
			Code:         "deprecated_env_used",
			Scope:        scope,
			Message:      key + " override deprecated; usar " + configPath + " en orquesta.config.json",
			EvidenceRefs: []string{evidenceRef},
		})
	}
	return diagnostics
}

func controlPlaneEnvKeysV0() []string {
	return []string{
		envServerRemoteControlPlaneConfirmV0,
		envServerControlTokenV0,
		envServerControlPrincipalV0,
		envServerControlPermissionRefV0,
		envServerControlPublicReasonV0,
	}
}

func opesBridgeEnvKeysV0() []string {
	return []string{
		envOPESBridgeEnabledV0, envOPESBridgeConfirmV0, envOPESBridgeDryRunV0,
		envOPESBridgeJobTypeV0, envOPESBridgeJobRefV0, envOPESBridgeProgramIDV0,
		envOPESBridgeTopicIDV0, envOPESBridgeCorrelationIDV0,
		envOPESBridgeJobTypeSequenceV0, envOPESBridgeLimitV0,
		envOPESBridgeTimeoutSecondsV0, envOPESBridgePriorityV0,
		envOPESBridgeIntervalSecondsV0, envOPESBridgeInitialDelaySecondsV0,
		envOPESBridgeMaxTicksV0, envOPESBridgeSuperviseSubmittedV0,
		envOPESBridgeWaitResidentSecondsV0, envOPESBridgeWaitResidentIntervalMSV0,
		envOPESBridgeRequireRuntimeCompatibilityV0,
		envOPESBridgeRequiredRuntimeBinarySHA256V0,
		envOPESBridgeRequiredRuntimeBuildRefV0,
		envOPESBridgeRequiredRuntimeCommitRefV0,
		envOPESBridgeSpeechSynthesisCapabilityV0,
		envOPESBridgeSpeechSynthesisCapabilityRefV0,
		envOPESBridgeSpeechSynthesisEvidenceRefsV0,
		envOPESBridgeSpeechSynthesisReasonV0,
		envOPESBridgeSpeechSynthesisNetworkReadyV0,
		envOPESBridgeSpeechSynthesisToolPathReadyV0,
		envOPESBridgeSpeechSynthesisQuotaReadyV0,
		envOPESBridgeSpeechSynthesisProgressHeartbeatReadyV0,
		envOPESBridgeSpeechSynthesisProviderTimeoutReadyV0,
		envOPESBridgeSpeechSynthesisNoProgressTimeoutSecondsV0,
		envOPESBridgeSpeechSynthesisToolWorkDirV0,
		envOPESBridgeSpeechSynthesisToolCommandV0,
		envOPESBridgeSpeechSynthesisToolPreflightTimeoutSecondsV0,
		envOPESBridgeRemoteQACapabilityV0,
		envOPESBridgeRemoteQACapabilityRefV0,
		envOPESBridgeRemoteQAEvidenceRefsV0,
		envOPESBridgeRemoteQAReasonV0,
		envOPESBridgeRemoteQANetworkReadyV0,
		envOPESBridgeRemoteQAAuthStateReadyV0,
		envOPESBridgeRemoteQAQuotaReadyV0,
		envOPESBridgeAllowUnfilteredV0,
		envOPESBridgeInputLedgerDisabledV0,
		envOPESBridgeInputLedgerPathV0,
		envOPESBridgeProductiveConfirmV0,
		envOPESBridgeDestinationEvidenceV0,
	}
}

func codexWaveEnvKeysV0() []string {
	return []string{
		envCodexWaveAgentsV0, envCodexWaveRefV0, envCodexWaveRuntimeWorkDirV0,
		envCodexWaveSourceCodeHomeV0, envCodexWaveModelV0,
		envCodexWaveReasoningEffortV0, envCodexWaveProfileV0,
		envCodexWaveSandboxV0, envCodexWaveApprovalPolicyV0,
		envCodexWaveExtraArgsV0, envCodexWaveIsolateHomeV0,
		envCodexWaveStrictCredentialProjectionV0,
		envCodexWaveProjectMemoriesV0, envCodexWaveProjectionMaxFilesV0,
		envCodexWaveProjectionMaxFileBytesV0,
		envCodexWaveProjectionMaxTotalBytesV0,
		envCodexWavePurgeRuntimeV0, envCodexWavePurgeRuntimeConfirmV0,
		envCodexWavePurgeRuntimeReportV0,
		envCodexWaveAllowUnmanagedLaunchV0,
		envCodexWaveUnmanagedLaunchReasonV0,
		envCodexWaveUnmanagedLaunchConfirmV0, envCodexWavePathV0,
		envCodexWaveTailReasonV0, envCodexWaveStopConfirmV0,
		envCodexWaveStopForceV0,
	}
}

func serverEnvAliasDiagnosticsV0(
	canonical string,
	scope string,
	evidenceRef string,
	legacies ...string,
) []orquestaserver.ServerDiagnosticV0 {
	canonical = strings.TrimSpace(canonical)
	canonicalValue := strings.TrimSpace(os.Getenv(canonical))
	diagnostics := make([]orquestaserver.ServerDiagnosticV0, 0, len(legacies))
	for _, legacy := range legacies {
		legacy = strings.TrimSpace(legacy)
		legacyValue := strings.TrimSpace(os.Getenv(legacy))
		if legacy == "" || legacyValue == "" {
			continue
		}
		code := "deprecated_env_used"
		message := legacy + " es legacy; usar " + canonical
		if canonicalValue != "" && canonicalValue != legacyValue {
			code = "env_alias_conflict"
			message = legacy + " ignorada porque " + canonical + " esta definida con otro valor"
		} else if canonicalValue != "" {
			code = "deprecated_env_duplicate"
			message = legacy + " duplica " + canonical + "; mantener solo " + canonical
		}
		diagnostics = append(diagnostics, orquestaserver.ServerDiagnosticV0{
			Code:         code,
			Scope:        scope,
			Message:      message,
			EvidenceRefs: []string{evidenceRef},
		})
	}
	return diagnostics
}

func configSettingSourceFromEnvOrLegacyV0(canonical string, legacies ...string) string {
	if strings.TrimSpace(os.Getenv(canonical)) != "" {
		return "explicit"
	}
	for _, legacy := range legacies {
		if strings.TrimSpace(os.Getenv(legacy)) != "" {
			return "legacy_alias"
		}
	}
	return "defaulted"
}

func serverIdleSelfImprovementAfterLegacyActiveV0() bool {
	return strings.TrimSpace(os.Getenv(envServerIdleSelfImprovementAfterV0)) == "" &&
		strings.TrimSpace(os.Getenv(envServerIdleSelfImprovementAfterLegacyV0)) != ""
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

func serverConfigSettingWithSourceV0(
	key string,
	value string,
	source string,
	scope string,
	label string,
	description string,
) orquestaserver.ServerConfigSettingV0 {
	setting := serverConfigSettingV0(key, value, scope, label, description)
	setting.Source = strings.TrimSpace(source)
	return setting
}

func configSettingSourceFromEnvV0(key string) string {
	if strings.TrimSpace(os.Getenv(key)) != "" {
		return "explicit"
	}
	return "defaulted"
}
