package main

import orquestaserver "orquesta/modulos/orquesta-server"

const (
	envOrquestaBaseURLV0 = "ORQUESTA_BASE_URL"

	envServerAddrV0                                  = "ORQUESTA_SERVER_ADDR"
	envServerStateDirV0                              = "ORQUESTA_SERVER_STATE_DIR"
	envServerAuditFileV0                             = "ORQUESTA_SERVER_AUDIT_FILE"
	envServerAuditDisabledV0                         = "ORQUESTA_SERVER_AUDIT_DISABLED"
	envServerRemoteControlPlaneConfirmV0             = "ORQUESTA_SERVER_REMOTE_CONTROL_PLANE_CONFIRM"
	envServerControlTokenV0                          = "ORQUESTA_SERVER_CONTROL_TOKEN"
	envServerControlPrincipalV0                      = "ORQUESTA_SERVER_CONTROL_PRINCIPAL"
	envServerControlPermissionRefV0                  = "ORQUESTA_SERVER_CONTROL_PERMISSION_REF"
	envServerControlPublicReasonV0                   = "ORQUESTA_SERVER_CONTROL_PUBLIC_REASON"
	envServerTickIntervalMSV0                        = "ORQUESTA_SERVER_TICK_INTERVAL_MS"
	envServerMaxRunsPerTickV0                        = "ORQUESTA_SERVER_MAX_RUNS_PER_TICK"
	envServerMaxExecutionsPerTickV0                  = "ORQUESTA_SERVER_MAX_EXECUTIONS_PER_TICK"
	envServerSupervisorMaxTicksV0                    = "ORQUESTA_SERVER_SUPERVISOR_MAX_TICKS"
	envServerAllowRepeatedRunsV0                     = "ORQUESTA_SERVER_ALLOW_REPEATED_RUNS"
	envServerDrainMaxBurstsV0                        = "ORQUESTA_SERVER_DRAIN_MAX_BURSTS"
	envServerDrainMaxStepsV0                         = "ORQUESTA_SERVER_DRAIN_MAX_STEPS"
	envServerDrainMaxDispatchesV0                    = "ORQUESTA_SERVER_DRAIN_MAX_DISPATCHES"
	envServerDrainMaxCommandsV0                      = "ORQUESTA_SERVER_DRAIN_MAX_COMMANDS"
	envServerDrainMaxOutboxV0                        = "ORQUESTA_SERVER_DRAIN_MAX_OUTBOX"
	envServerDrainMaxDecisionsV0                     = "ORQUESTA_SERVER_DRAIN_MAX_DECISIONS"
	envServerDrainMaxExternalWaitsV0                 = "ORQUESTA_SERVER_DRAIN_MAX_EXTERNAL_WAITS"
	envServerQueueLimitV0                            = "ORQUESTA_SERVER_QUEUE_LIMIT"
	envServerDefaultPriorityV0                       = "ORQUESTA_SERVER_DEFAULT_PRIORITY"
	envServerIdleSelfImprovementAfterV0              = "ORQUESTA_SERVER_IDLE_SELF_IMPROVEMENT_AFTER_SECONDS"
	envServerIdleSelfImprovementProjectRefV0         = "ORQUESTA_SERVER_IDLE_SELF_IMPROVEMENT_PROJECT_REF"
	envServerIdleSelfImprovementWorktreeRefV0        = "ORQUESTA_SERVER_IDLE_SELF_IMPROVEMENT_WORKTREE_REF"
	envServerIdleSelfImprovementBranchRefV0          = "ORQUESTA_SERVER_IDLE_SELF_IMPROVEMENT_BRANCH_REF"
	envServerIdleSelfImprovementAreaV0               = "ORQUESTA_SERVER_IDLE_SELF_IMPROVEMENT_AREA"
	envServerIdleSelfImprovementWriteSetV0           = "ORQUESTA_SERVER_IDLE_SELF_IMPROVEMENT_WRITE_SET"
	envServerIdleSelfImprovementRequiredTestsV0      = "ORQUESTA_SERVER_IDLE_SELF_IMPROVEMENT_REQUIRED_TESTS"
	envServerIdleSelfImprovementContextRefsV0        = "ORQUESTA_SERVER_IDLE_SELF_IMPROVEMENT_CONTEXT_REFS"
	envServerIdleSelfImprovementEvidenceRefsV0       = "ORQUESTA_SERVER_IDLE_SELF_IMPROVEMENT_EVIDENCE_REFS"
	envServerIdleSelfImprovementAcceptanceV0         = "ORQUESTA_SERVER_IDLE_SELF_IMPROVEMENT_ACCEPTANCE"
	envServerIdleSelfImprovementCompactRulesV0       = "ORQUESTA_SERVER_IDLE_SELF_IMPROVEMENT_COMPACT_RULES"
	envServerIdleSelfImprovementPriorityScoreV0      = "ORQUESTA_SERVER_IDLE_SELF_IMPROVEMENT_PRIORITY_SCORE"
	envServerIdleSelfImprovementMaxRequestsV0        = "ORQUESTA_SERVER_IDLE_SELF_IMPROVEMENT_MAX_REQUESTS"
	envServerIdleSelfImprovementTargetQueueV0        = "ORQUESTA_SERVER_IDLE_SELF_IMPROVEMENT_TARGET_QUEUE"
	envServerAutoprogrammingPromotionEnabledV0       = "ORQUESTA_SERVER_AUTOPROGRAMMING_PROMOTION_ENABLED"
	envServerAutoprogrammingPromotionArchiveDirV0    = "ORQUESTA_SERVER_AUTOPROGRAMMING_PROMOTION_ARCHIVE_DIR"
	envServerAutoprogrammingPromotionRepoRefV0       = "ORQUESTA_SERVER_AUTOPROGRAMMING_PROMOTION_REPO_REF"
	envServerAutoprogrammingPromotionAppRefV0        = "ORQUESTA_SERVER_AUTOPROGRAMMING_PROMOTION_APP_REF"
	envServerAutoprogrammingPromotionCommitMessageV0 = "ORQUESTA_SERVER_AUTOPROGRAMMING_PROMOTION_COMMIT_MESSAGE"
	envStartupCleanupModeV0                          = "ORQUESTA_STARTUP_CLEANUP_MODE"
	envStartupQueueLimitV0                           = "ORQUESTA_STARTUP_QUEUE_LIMIT"
	envReviewGateStrictGoLineBudgetV0                = "ORQUESTA_REVIEW_GATE_STRICT_GO_LINE_BUDGET"
	envDomainDeliveryLedgerPathV0                    = "ORQUESTA_DOMAIN_DELIVERY_LEDGER_PATH"
	envDirectorMaxBurstsV0                           = "ORQUESTA_DIRECTOR_MAX_BURSTS"
	envDirectorMaxStepsV0                            = "ORQUESTA_DIRECTOR_MAX_STEPS"
	envDirectorMaxDispatchesV0                       = "ORQUESTA_DIRECTOR_MAX_DISPATCHES"
	envDirectorMaxCommandsV0                         = "ORQUESTA_DIRECTOR_MAX_COMMANDS"
	envDirectorMaxOutboxV0                           = "ORQUESTA_DIRECTOR_MAX_OUTBOX"
	envDirectorMaxExternalWaitsV0                    = "ORQUESTA_DIRECTOR_MAX_EXTERNAL_WAITS"

	envCodexProjectWorkDirV0               = "ORQUESTA_CODEX_PROJECT_WORKDIR"
	envCodexRuntimeWorkDirV0               = "ORQUESTA_CODEX_RUNTIME_WORKDIR"
	envCodexCommandV0                      = "ORQUESTA_CODEX_COMMAND"
	envCodexCodeHomeV0                     = "ORQUESTA_CODEX_CODE_HOME"
	envCodexHomeV0                         = "ORQUESTA_CODEX_HOME"
	envCodexPathV0                         = "ORQUESTA_CODEX_PATH"
	envCodexModelV0                        = "ORQUESTA_CODEX_MODEL"
	envCodexReasoningEffortV0              = "ORQUESTA_CODEX_REASONING_EFFORT"
	envCodexProfileV0                      = "ORQUESTA_CODEX_PROFILE"
	envCodexSandboxV0                      = "ORQUESTA_CODEX_SANDBOX"
	envCodexApprovalPolicyV0               = "ORQUESTA_CODEX_APPROVAL_POLICY"
	envCodexDirectorSandboxV0              = "ORQUESTA_CODEX_DIRECTOR_SANDBOX"
	envCodexDirectorApprovalPolicyV0       = "ORQUESTA_CODEX_DIRECTOR_APPROVAL_POLICY"
	envCodexAllowInteractiveApprovalV0     = "ORQUESTA_CODEX_ALLOW_INTERACTIVE_APPROVAL"
	envCodexExtraArgsV0                    = "ORQUESTA_CODEX_EXTRA_ARGS"
	envCodexWaitIntervalMSV0               = "ORQUESTA_CODEX_WAIT_INTERVAL_MS"
	envCodexStalledTicksV0                 = "ORQUESTA_CODEX_STALLED_TICKS"
	envCodexLoopTicksV0                    = "ORQUESTA_CODEX_LOOP_TICKS"
	envCodexMaxExpectedSecondsV0           = "ORQUESTA_CODEX_MAX_EXPECTED_SECONDS"
	envCodexNoActivitySecondsV0            = "ORQUESTA_CODEX_NO_ACTIVITY_SECONDS"
	envCodexMaxBatchReadyV0                = "ORQUESTA_CODEX_MAX_BATCH_READY"
	envCodexMaxConcurrencyV0               = "ORQUESTA_CODEX_MAX_CONCURRENCY"
	envCodexDirectorWaveAgentsV0           = "ORQUESTA_CODEX_DIRECTOR_WAVE_AGENTS"
	envCodexDirectorMaxSubagentsPerAgentV0 = "ORQUESTA_CODEX_DIRECTOR_MAX_SUBAGENTS_PER_AGENT"
	envCodexDirectorRecursiveAgentBudgetV0 = "ORQUESTA_CODEX_DIRECTOR_RECURSIVE_AGENT_BUDGET"
	envCodexDirectorProjectRefV0           = "ORQUESTA_CODEX_DIRECTOR_PROJECT_REF"
	envCodexDirectorDomainRefsV0           = "ORQUESTA_CODEX_DIRECTOR_DOMAIN_REFS"
	envCodexDirectorDomainContextFilesV0   = "ORQUESTA_CODEX_DIRECTOR_DOMAIN_CONTEXT_FILES"

	envCodexWaveAgentsV0                     = "ORQUESTA_CODEX_WAVE_AGENTS"
	envCodexWaveRefV0                        = "ORQUESTA_CODEX_WAVE_REF"
	envCodexWaveRuntimeWorkDirV0             = "ORQUESTA_CODEX_WAVE_RUNTIME_WORKDIR"
	envCodexWaveSourceCodeHomeV0             = "ORQUESTA_CODEX_WAVE_SOURCE_CODEX_HOME"
	envCodexWaveModelV0                      = "ORQUESTA_CODEX_WAVE_MODEL"
	envCodexWaveReasoningEffortV0            = "ORQUESTA_CODEX_WAVE_REASONING_EFFORT"
	envCodexWaveProfileV0                    = "ORQUESTA_CODEX_WAVE_PROFILE"
	envCodexWaveSandboxV0                    = "ORQUESTA_CODEX_WAVE_SANDBOX"
	envCodexWaveApprovalPolicyV0             = "ORQUESTA_CODEX_WAVE_APPROVAL_POLICY"
	envCodexWaveExtraArgsV0                  = "ORQUESTA_CODEX_WAVE_EXTRA_ARGS"
	envCodexWaveIsolateHomeV0                = "ORQUESTA_CODEX_WAVE_ISOLATE_HOME"
	envCodexWaveStrictCredentialProjectionV0 = "ORQUESTA_CODEX_WAVE_STRICT_CREDENTIAL_PROJECTION"
	envCodexWaveProjectMemoriesV0            = "ORQUESTA_CODEX_WAVE_PROJECT_MEMORIES"
	envCodexWavePurgeRuntimeV0               = "ORQUESTA_CODEX_WAVE_PURGE_RUNTIME"
	envCodexWavePurgeRuntimeConfirmV0        = "ORQUESTA_CODEX_WAVE_PURGE_RUNTIME_CONFIRM"
	envCodexWavePurgeRuntimeReportV0         = "ORQUESTA_CODEX_WAVE_PURGE_RUNTIME_REPORT"
	envCodexWaveAllowUnmanagedLaunchV0       = "ORQUESTA_CODEX_WAVE_ALLOW_UNMANAGED_LAUNCH"
	envCodexWaveUnmanagedLaunchReasonV0      = "ORQUESTA_CODEX_WAVE_UNMANAGED_LAUNCH_REASON"
	envCodexWaveUnmanagedLaunchConfirmV0     = "ORQUESTA_CODEX_WAVE_UNMANAGED_LAUNCH_CONFIRM"
	envCodexWavePathV0                       = "ORQUESTA_CODEX_WAVE_PATH"
	envCodexWaveTailReasonV0                 = "ORQUESTA_CODEX_WAVE_TAIL_REASON"
	envCodexWaveStopConfirmV0                = "ORQUESTA_CODEX_WAVE_STOP_CONFIRM"
	envCodexWaveStopForceV0                  = "ORQUESTA_CODEX_WAVE_STOP_FORCE"
	envCodexUsageAccountingV0                = "ORQUESTA_CODEX_USAGE_ACCOUNTING"
	envCodexUsageLogMaxBytesV0               = "ORQUESTA_CODEX_USAGE_LOG_MAX_BYTES"

	envCapacityTierV0            = "ORQUESTA_CAPACITY_TIER"
	envCapacityReasoningEffortV0 = "ORQUESTA_CAPACITY_REASONING_EFFORT"

	envOPESBaseURLV0                    = "ORQUESTA_OPES_BASE_URL"
	envOPESBaseURLLegacyV0              = "OPES_BASE_URL"
	envOPESTimeoutSecondsV0             = "ORQUESTA_OPES_TIMEOUT_SECONDS"
	envOPESDefaultMaxAttemptsV0         = "ORQUESTA_OPES_DEFAULT_MAX_ATTEMPTS"
	envOPESBridgeEnabledV0              = "ORQUESTA_OPES_BRIDGE_ENABLED"
	envOPESBridgeConfirmV0              = "ORQUESTA_OPES_BRIDGE_CONFIRM"
	envOPESBridgeDryRunV0               = "ORQUESTA_OPES_BRIDGE_DRY_RUN"
	envOPESBridgeJobTypeV0              = "ORQUESTA_OPES_BRIDGE_JOB_TYPE"
	envOPESBridgeJobRefV0               = "ORQUESTA_OPES_BRIDGE_JOB_REF"
	envOPESBridgeJobTypeSequenceV0      = "ORQUESTA_OPES_BRIDGE_JOB_TYPE_SEQUENCE"
	envOPESBridgeLimitV0                = "ORQUESTA_OPES_BRIDGE_LIMIT"
	envOPESBridgeTimeoutSecondsV0       = "ORQUESTA_OPES_BRIDGE_TIMEOUT_SECONDS"
	envOPESBridgePriorityV0             = "ORQUESTA_OPES_BRIDGE_PRIORITY"
	envOPESBridgeIntervalSecondsV0      = "ORQUESTA_OPES_BRIDGE_INTERVAL_SECONDS"
	envOPESBridgeInitialDelaySecondsV0  = "ORQUESTA_OPES_BRIDGE_INITIAL_DELAY_SECONDS"
	envOPESBridgeMaxTicksV0             = "ORQUESTA_OPES_BRIDGE_MAX_TICKS"
	envOPESBridgeAllowUnfilteredV0      = "ORQUESTA_OPES_BRIDGE_ALLOW_UNFILTERED"
	envOPESBridgeInputLedgerDisabledV0  = "ORQUESTA_OPES_BRIDGE_INPUT_LEDGER_DISABLED"
	envOPESBridgeInputLedgerPathV0      = "ORQUESTA_OPES_BRIDGE_INPUT_LEDGER_PATH"
	envDomainWorkHTTPBaseURLV0          = "ORQUESTA_DOMAIN_WORK_HTTP_BASE_URL"
	envDomainWorkFileDirV0              = "ORQUESTA_DOMAIN_WORK_FILE_DIR"
	envDomainWorkFileEnabledV0          = "ORQUESTA_DOMAIN_WORK_FILE_ENABLED"
	envDomainWorkHTTPCreatePathV0       = "ORQUESTA_DOMAIN_WORK_HTTP_CREATE_PATH"
	envDomainWorkHTTPSubmitPathV0       = "ORQUESTA_DOMAIN_WORK_HTTP_SUBMIT_PATH"
	envDomainWorkHTTPTimeoutSecondsV0   = "ORQUESTA_DOMAIN_WORK_HTTP_TIMEOUT_SECONDS"
	envDomainWorkHTTPEgressModeV0       = "ORQUESTA_DOMAIN_WORK_HTTP_EGRESS_MODE"
	envDomainWorkHTTPAllowedHostsV0     = "ORQUESTA_DOMAIN_WORK_HTTP_ALLOWED_HOSTS"
	envRequiredTestRunnerEnabledV0      = "ORQUESTA_REQUIRED_TEST_RUNNER_ENABLED"
	envRequiredTestMaxOutputBytesV0     = "ORQUESTA_REQUIRED_TEST_MAX_OUTPUT_BYTES"
	envRequiredTestOutputMaxArtifactsV0 = "ORQUESTA_REQUIRED_TEST_OUTPUT_MAX_ARTIFACTS"
	envRequiredTestGoCommandV0          = "ORQUESTA_REQUIRED_TEST_GO_COMMAND"
	envRequiredTestAllowedCommandsV0    = "ORQUESTA_REQUIRED_TEST_ALLOWED_COMMANDS"
	envRequiredTestOutputDirV0          = "ORQUESTA_REQUIRED_TEST_OUTPUT_DIR"
	envRequiredTestEnvV0                = "ORQUESTA_REQUIRED_TEST_ENV"

	defaultCodexWaitIntervalMSV0               = 2000
	defaultCodexStalledTicksV0                 = 300
	defaultCodexLoopTicksV0                    = 300
	defaultCodexMaxExpectedSecondsV0           = 1200
	defaultCodexNoActivitySecondsV0            = 600
	defaultCodexMaxBatchReadyV0                = 10
	defaultCodexMaxConcurrencyV0               = 10
	defaultCodexServerMaxRunsPerTickV0         = 10
	defaultCodexServerQueueLimitV0             = 20
	defaultCodexServerDefaultPriorityV0        = 50
	defaultCodexServerMaxExecutionsV0          = 10
	defaultCodexDirectorWaveAgentsV0           = 10
	defaultCodexDirectorMaxSubagentsPerAgentV0 = 6
	defaultCodexDirectorRecursiveAgentBudgetV0 = 70
)

type serverEnvSettingMetadataV0 struct {
	Scope       string
	Label       string
	Description string
}

var serverEffectiveEnvRegistryV0 = map[string]serverEnvSettingMetadataV0{
	envServerMaxRunsPerTickV0: {
		Scope:       "server_supervisor",
		Label:       "Runs por tick",
		Description: "Runs candidatos por pulso residente.",
	},
	envServerMaxExecutionsPerTickV0: {
		Scope:       "server_supervisor",
		Label:       "Ejecuciones por tick",
		Description: "Ejecuciones lanzadas por pulso residente.",
	},
	envServerDrainMaxExternalWaitsV0: {
		Scope:       "server_supervisor",
		Label:       "Esperas externas",
		Description: "Esperas externas maximas por drain del supervisor residente.",
	},
	envServerTickIntervalMSV0: {
		Scope:       "server_supervisor",
		Label:       "Intervalo tick ms",
		Description: "Intervalo entre pulsos automaticos del servidor.",
	},
	envServerIdleSelfImprovementTargetQueueV0: {
		Scope:       "autoprogramming",
		Label:       "Cola objetivo",
		Description: "Tamano objetivo de cola de automejora.",
	},
	envServerIdleSelfImprovementMaxRequestsV0: {
		Scope:       "autoprogramming",
		Label:       "Nuevas tareas por tanda",
		Description: "Maximo de tareas nuevas por tanda de automejora.",
	},
	envCodexMaxBatchReadyV0: {
		Scope:       "codex_runtime",
		Label:       "Batch Codex ready",
		Description: "Agentes ready a despachar por tanda Codex.",
	},
	envCodexMaxConcurrencyV0: {
		Scope:       "codex_runtime",
		Label:       "Procesos Codex vivos",
		Description: "Limite global de procesos Codex vivos; si esta lleno, la tarea queda pendiente sin consumirse.",
	},
	envCodexReasoningEffortV0: {
		Scope:       "codex_runtime",
		Label:       "Reasoning Codex",
		Description: "Esfuerzo de razonamiento para agentes Codex.",
	},
	envCapacityReasoningEffortV0: {
		Scope:       "capacity",
		Label:       "Reasoning capacidad",
		Description: "Recomendacion de capacidad que recibe el stack.",
	},
	envCodexDirectorWaveAgentsV0: {
		Scope:       "director_wave",
		Label:       "Agentes por ola",
		Description: "Agentes padre que puede lanzar el Director en una ola.",
	},
	envCodexDirectorMaxSubagentsPerAgentV0: {
		Scope:       "director_wave",
		Label:       "Subagentes por agente",
		Description: "Fanout maximo de subagentes por agente padre en delegacion recursiva.",
	},
	envCodexDirectorRecursiveAgentBudgetV0: {
		Scope:       "director_wave",
		Label:       "Presupuesto recursivo",
		Description: "Presupuesto global de agentes en arbol recursivo.",
	},
}

func serverConfigSettingFromRegistryV0(key string, value string) orquestaserver.ServerConfigSettingV0 {
	metadata := serverEffectiveEnvRegistryV0[key]
	return serverConfigSettingV0(key, value, metadata.Scope, metadata.Label, metadata.Description)
}
