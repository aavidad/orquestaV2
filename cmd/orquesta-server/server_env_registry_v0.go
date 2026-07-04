package main

import (
	"strings"

	orquestarails "orquesta/modulos/orquesta-rails"
	orquestaserver "orquesta/modulos/orquesta-server"
)

const (
	envOrquestaServerURLV0  = "ORQUESTA_SERVER_URL"
	envOrquestaBaseURLV0    = "ORQUESTA_BASE_URL"
	envOrquestaRuntimeDirV0 = "ORQUESTA_RUNTIME_DIR"

	envServerAddrV0                                         = "ORQUESTA_SERVER_ADDR"
	envServerStateDirV0                                     = "ORQUESTA_SERVER_STATE_DIR"
	envServerAuditFileV0                                    = "ORQUESTA_SERVER_AUDIT_FILE"
	envServerAuditDisabledV0                                = "ORQUESTA_SERVER_AUDIT_DISABLED"
	envServerRemoteControlPlaneConfirmV0                    = "ORQUESTA_SERVER_REMOTE_CONTROL_PLANE_CONFIRM"
	envServerControlTokenV0                                 = "ORQUESTA_SERVER_CONTROL_TOKEN"
	envServerControlPrincipalV0                             = "ORQUESTA_SERVER_CONTROL_PRINCIPAL"
	envServerControlPermissionRefV0                         = "ORQUESTA_SERVER_CONTROL_PERMISSION_REF"
	envServerControlPublicReasonV0                          = "ORQUESTA_SERVER_CONTROL_PUBLIC_REASON"
	envServerSelfProgrammingOnlyV0                          = "ORQUESTA_SERVER_SELF_PROGRAMMING_ONLY"
	envServerSelfProgrammingRootV0                          = "ORQUESTA_SERVER_SELF_PROGRAMMING_ROOT"
	envServerTickIntervalMSV0                               = "ORQUESTA_SERVER_TICK_INTERVAL_MS"
	envServerMaxRunsPerTickV0                               = "ORQUESTA_SERVER_MAX_RUNS_PER_TICK"
	envServerMaxExecutionsPerTickV0                         = "ORQUESTA_SERVER_MAX_EXECUTIONS_PER_TICK"
	envServerSupervisorMaxTicksV0                           = "ORQUESTA_SERVER_SUPERVISOR_MAX_TICKS"
	envServerAllowRepeatedRunsV0                            = "ORQUESTA_SERVER_ALLOW_REPEATED_RUNS"
	envServerDrainMaxBurstsV0                               = "ORQUESTA_SERVER_DRAIN_MAX_BURSTS"
	envServerDrainMaxStepsV0                                = "ORQUESTA_SERVER_DRAIN_MAX_STEPS"
	envServerDrainMaxDispatchesV0                           = "ORQUESTA_SERVER_DRAIN_MAX_DISPATCHES"
	envServerDrainMaxCommandsV0                             = "ORQUESTA_SERVER_DRAIN_MAX_COMMANDS"
	envServerDrainMaxOutboxV0                               = "ORQUESTA_SERVER_DRAIN_MAX_OUTBOX"
	envServerDrainMaxDecisionsV0                            = "ORQUESTA_SERVER_DRAIN_MAX_DECISIONS"
	envServerDrainMaxExternalWaitsV0                        = "ORQUESTA_SERVER_DRAIN_MAX_EXTERNAL_WAITS"
	envServerQueueLimitV0                                   = "ORQUESTA_SERVER_QUEUE_LIMIT"
	envServerDefaultPriorityV0                              = "ORQUESTA_SERVER_DEFAULT_PRIORITY"
	envSecurityModeV0                                       = orquestarails.SecurityModeEnvV0
	envRailsModeV0                                          = orquestarails.RailsModeEnvV0
	envDetailProhibitedRailsV0                              = "ORQUESTA_DETAIL_PROHIBITED_RAILS"
	envDetailProhibitedRailsScopeV0                         = "ORQUESTA_DETAIL_PROHIBITED_RAILS_SCOPE"
	envServerIdleSelfImprovementAfterV0                     = "ORQUESTA_SERVER_IDLE_SELF_IMPROVEMENT_AFTER_SECONDS"
	envServerIdleSelfImprovementAfterLegacyV0               = "ORQUESTA_SERVER_IDLE_SELF_IMPROVEMENT_AFTER"
	envServerIdleSelfImprovementDisabledV0                  = "ORQUESTA_SERVER_IDLE_SELF_IMPROVEMENT_DISABLED"
	envServerIdleSelfImprovementProjectWorkDirV0            = "ORQUESTA_SERVER_IDLE_SELF_IMPROVEMENT_PROJECT_WORKDIR"
	envServerIdleSelfImprovementProjectRefV0                = "ORQUESTA_SERVER_IDLE_SELF_IMPROVEMENT_PROJECT_REF"
	envServerIdleSelfImprovementWorktreeRefV0               = "ORQUESTA_SERVER_IDLE_SELF_IMPROVEMENT_WORKTREE_REF"
	envServerIdleSelfImprovementBranchRefV0                 = "ORQUESTA_SERVER_IDLE_SELF_IMPROVEMENT_BRANCH_REF"
	envServerIdleSelfImprovementAreaV0                      = "ORQUESTA_SERVER_IDLE_SELF_IMPROVEMENT_AREA"
	envServerIdleSelfImprovementWriteSetV0                  = "ORQUESTA_SERVER_IDLE_SELF_IMPROVEMENT_WRITE_SET"
	envServerIdleSelfImprovementRequiredTestsV0             = "ORQUESTA_SERVER_IDLE_SELF_IMPROVEMENT_REQUIRED_TESTS"
	envServerIdleSelfImprovementContextRefsV0               = "ORQUESTA_SERVER_IDLE_SELF_IMPROVEMENT_CONTEXT_REFS"
	envServerIdleSelfImprovementEvidenceRefsV0              = "ORQUESTA_SERVER_IDLE_SELF_IMPROVEMENT_EVIDENCE_REFS"
	envServerIdleSelfImprovementAcceptanceV0                = "ORQUESTA_SERVER_IDLE_SELF_IMPROVEMENT_ACCEPTANCE"
	envServerIdleSelfImprovementGoalFirstV0                 = "ORQUESTA_SERVER_IDLE_SELF_IMPROVEMENT_GOAL_FIRST_ENABLED"
	envServerIdleSelfImprovementFrozenTestsV0               = "ORQUESTA_SERVER_IDLE_SELF_IMPROVEMENT_FROZEN_TESTS_ENABLED"
	envServerIdleSelfImprovementCompactRulesV0              = "ORQUESTA_SERVER_IDLE_SELF_IMPROVEMENT_COMPACT_RULES"
	envServerIdleSelfImprovementPriorityScoreV0             = "ORQUESTA_SERVER_IDLE_SELF_IMPROVEMENT_PRIORITY_SCORE"
	envServerIdleSelfImprovementMaxRequestsV0               = "ORQUESTA_SERVER_IDLE_SELF_IMPROVEMENT_MAX_REQUESTS"
	envServerIdleSelfImprovementTargetQueueV0               = "ORQUESTA_SERVER_IDLE_SELF_IMPROVEMENT_TARGET_QUEUE"
	envServerIdleSelfImprovementDailyGoalBudgetV0           = "ORQUESTA_SERVER_IDLE_SELF_IMPROVEMENT_DAILY_GOAL_BUDGET"
	envServerIdleSelfImprovementDailyContextBudgetBytesV0   = "ORQUESTA_SERVER_IDLE_SELF_IMPROVEMENT_DAILY_CONTEXT_BUDGET_BYTES"
	envSelfAuditBacklogEnabledV0                            = "ORQUESTA_SELF_AUDIT_BACKLOG_ENABLED"
	envServerAutoprogrammingPromotionEnabledV0              = "ORQUESTA_SERVER_AUTOPROGRAMMING_PROMOTION_ENABLED"
	envServerAutoprogrammingPromotionArchiveDirV0           = "ORQUESTA_SERVER_AUTOPROGRAMMING_PROMOTION_ARCHIVE_DIR"
	envServerAutoprogrammingPromotionRepoRefV0              = "ORQUESTA_SERVER_AUTOPROGRAMMING_PROMOTION_REPO_REF"
	envServerAutoprogrammingPromotionAppRefV0               = "ORQUESTA_SERVER_AUTOPROGRAMMING_PROMOTION_APP_REF"
	envServerAutoprogrammingPromotionCommitMessageV0        = "ORQUESTA_SERVER_AUTOPROGRAMMING_PROMOTION_COMMIT_MESSAGE"
	envStartupCleanupModeV0                                 = "ORQUESTA_STARTUP_CLEANUP_MODE"
	envStartupCleanupScopeRefsV0                            = "ORQUESTA_STARTUP_CLEANUP_SCOPE_REFS"
	envStartupQueueLimitV0                                  = "ORQUESTA_STARTUP_QUEUE_LIMIT"
	envAutoprogrammingLegacyDirectorLoopV0                  = "ORQUESTA_AUTOPROGRAMMING_LEGACY_DIRECTOR_LOOP"
	envAutoprogrammingCheckpointOnlyHighConsumptionTokensV0 = "ORQUESTA_AUTOPROGRAMMING_CHECKPOINT_ONLY_HIGH_CONSUMPTION_TOKENS"
	envAutoprogrammingCheckpointOnlyMaxWaitSecondsV0        = "ORQUESTA_AUTOPROGRAMMING_CHECKPOINT_ONLY_MAX_WAIT_SECONDS"
	envAutoprogrammingNoCheckpointWarningMaxWaitSecondsV0   = "ORQUESTA_AUTOPROGRAMMING_NO_CHECKPOINT_WARNING_MAX_WAIT_SECONDS"
	envExternalWorkLegacyDirectorLoopV0                     = "ORQUESTA_EXTERNAL_WORK_LEGACY_DIRECTOR_LOOP"
	envReviewGateStrictGoLineBudgetV0                       = "ORQUESTA_REVIEW_GATE_STRICT_GO_LINE_BUDGET"
	envDomainDeliveryLedgerPathV0                           = "ORQUESTA_DOMAIN_DELIVERY_LEDGER_PATH"
	envDirectorMaxBurstsV0                                  = "ORQUESTA_DIRECTOR_MAX_BURSTS"
	envDirectorMaxStepsV0                                   = "ORQUESTA_DIRECTOR_MAX_STEPS"
	envDirectorMaxDispatchesV0                              = "ORQUESTA_DIRECTOR_MAX_DISPATCHES"
	envDirectorMaxCommandsV0                                = "ORQUESTA_DIRECTOR_MAX_COMMANDS"
	envDirectorMaxOutboxV0                                  = "ORQUESTA_DIRECTOR_MAX_OUTBOX"
	envDirectorMaxExternalWaitsV0                           = "ORQUESTA_DIRECTOR_MAX_EXTERNAL_WAITS"
	envCodebaseBrokerProviderKindV0                         = "ORQUESTA_CODEBASE_BROKER_PROVIDER_KIND"
	envCodebaseBrokerExternalIndexerEnabledV0               = "ORQUESTA_CODEBASE_BROKER_EXTERNAL_INDEXER_ENABLED"
	envCodebaseBrokerMaxConcurrentV0                        = "ORQUESTA_CODEBASE_BROKER_MAX_CONCURRENT"
	envCodebaseBrokerTimeoutMSV0                            = "ORQUESTA_CODEBASE_BROKER_TIMEOUT_MS"
	envCodebaseBrokerStateDirV0                             = "ORQUESTA_CODEBASE_BROKER_STATE_DIR"
	envCodebaseBrokerWatchdogEnabledV0                      = "ORQUESTA_CODEBASE_BROKER_WATCHDOG_ENABLED"
	envCodebaseBrokerWatchdogStopOrphansV0                  = "ORQUESTA_CODEBASE_BROKER_WATCHDOG_STOP_ORPHANS"
	envCodebaseBrokerWatchdogOrphanMinAgeSecondsV0          = "ORQUESTA_CODEBASE_BROKER_WATCHDOG_ORPHAN_MIN_AGE_SECONDS"
	envCodebaseBrokerCommandV0                              = "ORQUESTA_CODEBASE_BROKER_COMMAND"
	envCodebaseBrokerProjectNameV0                          = "ORQUESTA_CODEBASE_BROKER_PROJECT_NAME"

	envCodexProjectWorkDirV0                        = "ORQUESTA_CODEX_PROJECT_WORKDIR"
	envCodexRuntimeWorkDirV0                        = "ORQUESTA_CODEX_RUNTIME_WORKDIR"
	envCodexCommandV0                               = "ORQUESTA_CODEX_COMMAND"
	envCodexCodeHomeV0                              = "ORQUESTA_CODEX_CODE_HOME"
	envCodexHomeV0                                  = "ORQUESTA_CODEX_HOME"
	envCodexCodeHomeLegacyV0                        = "CODEX_HOME"
	envCodexPathV0                                  = "ORQUESTA_CODEX_PATH"
	envCodexModelV0                                 = "ORQUESTA_CODEX_MODEL"
	envCodexReasoningEffortV0                       = "ORQUESTA_CODEX_REASONING_EFFORT"
	envCodexProfileV0                               = "ORQUESTA_CODEX_PROFILE"
	envCodexSandboxV0                               = "ORQUESTA_CODEX_SANDBOX"
	envCodexApprovalPolicyV0                        = "ORQUESTA_CODEX_APPROVAL_POLICY"
	envCodexDirectorSandboxV0                       = "ORQUESTA_CODEX_DIRECTOR_SANDBOX"
	envCodexDirectorApprovalPolicyV0                = "ORQUESTA_CODEX_DIRECTOR_APPROVAL_POLICY"
	envCodexAllowInteractiveApprovalV0              = "ORQUESTA_CODEX_ALLOW_INTERACTIVE_APPROVAL"
	envCodexExtraArgsV0                             = "ORQUESTA_CODEX_EXTRA_ARGS"
	envCodexSkillInstructionsJSONV0                 = "ORQUESTA_CODEX_SKILL_INSTRUCTIONS_JSON"
	envCodexWaitIntervalMSV0                        = "ORQUESTA_CODEX_WAIT_INTERVAL_MS"
	envCodexStalledTicksV0                          = "ORQUESTA_CODEX_STALLED_TICKS"
	envCodexLoopTicksV0                             = "ORQUESTA_CODEX_LOOP_TICKS"
	envCodexMaxExpectedSecondsV0                    = "ORQUESTA_CODEX_MAX_EXPECTED_SECONDS"
	envCodexNoActivitySecondsV0                     = "ORQUESTA_CODEX_NO_ACTIVITY_SECONDS"
	envCodexPromoteMaterializedArtifactWithoutAckV0 = "ORQUESTA_CODEX_PROMOTE_MATERIALIZED_ARTIFACT_WITHOUT_ACK"
	envCodexExecutionModeV0                         = "ORQUESTA_CODEX_EXECUTION_MODE"
	envCodexMaxBatchReadyV0                         = "ORQUESTA_CODEX_MAX_BATCH_READY"
	envCodexMaxConcurrencyV0                        = "ORQUESTA_CODEX_MAX_CONCURRENCY"
	envCodexDirectorWaveAgentsV0                    = "ORQUESTA_CODEX_DIRECTOR_WAVE_AGENTS"
	envCodexDirectorMaxSubagentsPerAgentV0          = "ORQUESTA_CODEX_DIRECTOR_MAX_SUBAGENTS_PER_AGENT"
	envCodexDirectorRecursiveAgentBudgetV0          = "ORQUESTA_CODEX_DIRECTOR_RECURSIVE_AGENT_BUDGET"
	envCodexDirectorProjectRefV0                    = "ORQUESTA_CODEX_DIRECTOR_PROJECT_REF"
	envCodexDirectorDomainRefsV0                    = "ORQUESTA_CODEX_DIRECTOR_DOMAIN_REFS"
	envCodexDirectorDomainContextFilesV0            = "ORQUESTA_CODEX_DIRECTOR_DOMAIN_CONTEXT_FILES"
	envCodexGoalBackendV0                           = "ORQUESTA_CODEX_GOAL_BACKEND"
	envAllowAppServerProxyDiagnosticV0              = "ORQUESTA_ALLOW_APP_SERVER_PROXY_DIAGNOSTIC"
	envCodexGoalTimeoutMSV0                         = "ORQUESTA_CODEX_GOAL_TIMEOUT_MS"
	envCodexGoalPreflightTimeoutMSV0                = "ORQUESTA_CODEX_GOAL_PREFLIGHT_TIMEOUT_MS"

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
	envCodexWaveProjectionMaxFilesV0         = "ORQUESTA_CODEX_WAVE_PROJECTION_MAX_FILES"
	envCodexWaveProjectionMaxFileBytesV0     = "ORQUESTA_CODEX_WAVE_PROJECTION_MAX_FILE_BYTES"
	envCodexWaveProjectionMaxTotalBytesV0    = "ORQUESTA_CODEX_WAVE_PROJECTION_MAX_TOTAL_BYTES"
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
	envCapacityPolicyRefV0       = "ORQUESTA_CAPACITY_POLICY_REF"
	envCapacityPoolRefV0         = "ORQUESTA_CAPACITY_POOL_REF"

	envWorktreeSnapshotMaxFilesV0      = "ORQUESTA_WORKTREE_SNAPSHOT_MAX_FILES"
	envWorktreeSnapshotMaxFileBytesV0  = "ORQUESTA_WORKTREE_SNAPSHOT_MAX_FILE_BYTES"
	envWorktreeSnapshotMaxTotalBytesV0 = "ORQUESTA_WORKTREE_SNAPSHOT_MAX_TOTAL_BYTES"

	envOPESBaseURLV0                            = "ORQUESTA_OPES_BASE_URL"
	envOPESBaseURLLegacyV0                      = "OPES_BASE_URL"
	envOPESTimeoutSecondsV0                     = "ORQUESTA_OPES_TIMEOUT_SECONDS"
	envOPESDefaultMaxAttemptsV0                 = "ORQUESTA_OPES_DEFAULT_MAX_ATTEMPTS"
	envOPESBridgeEnabledV0                      = "ORQUESTA_OPES_BRIDGE_ENABLED"
	envOPESBridgeConfirmV0                      = "ORQUESTA_OPES_BRIDGE_CONFIRM"
	envOPESBridgeDryRunV0                       = "ORQUESTA_OPES_BRIDGE_DRY_RUN"
	envOPESBridgeJobTypeV0                      = "ORQUESTA_OPES_BRIDGE_JOB_TYPE"
	envOPESBridgeJobRefV0                       = "ORQUESTA_OPES_BRIDGE_JOB_REF"
	envOPESBridgeProgramIDV0                    = "ORQUESTA_OPES_BRIDGE_PROGRAM_ID"
	envOPESBridgeTopicIDV0                      = "ORQUESTA_OPES_BRIDGE_TOPIC_ID"
	envOPESBridgeCorrelationIDV0                = "ORQUESTA_OPES_BRIDGE_CORRELATION_ID"
	envOPESBridgeJobTypeSequenceV0              = "ORQUESTA_OPES_BRIDGE_JOB_TYPE_SEQUENCE"
	envOPESBridgeLimitV0                        = "ORQUESTA_OPES_BRIDGE_LIMIT"
	envOPESBridgeTimeoutSecondsV0               = "ORQUESTA_OPES_BRIDGE_TIMEOUT_SECONDS"
	envOPESBridgePriorityV0                     = "ORQUESTA_OPES_BRIDGE_PRIORITY"
	envOPESBridgeIntervalSecondsV0              = "ORQUESTA_OPES_BRIDGE_INTERVAL_SECONDS"
	envOPESBridgeInitialDelaySecondsV0          = "ORQUESTA_OPES_BRIDGE_INITIAL_DELAY_SECONDS"
	envOPESBridgeMaxTicksV0                     = "ORQUESTA_OPES_BRIDGE_MAX_TICKS"
	envOPESBridgeSuperviseSubmittedV0           = "ORQUESTA_OPES_BRIDGE_SUPERVISE_SUBMITTED"
	envOPESBridgeWaitResidentSecondsV0          = "ORQUESTA_OPES_BRIDGE_WAIT_RESIDENT_SECONDS"
	envOPESBridgeWaitResidentIntervalMSV0       = "ORQUESTA_OPES_BRIDGE_WAIT_RESIDENT_INTERVAL_MS"
	envOPESBridgeRequireRuntimeCompatibilityV0  = "ORQUESTA_OPES_BRIDGE_REQUIRE_RUNTIME_COMPATIBILITY"
	envOPESBridgeRequiredRuntimeBinarySHA256V0  = "ORQUESTA_OPES_BRIDGE_REQUIRED_RUNTIME_BINARY_SHA256"
	envOPESBridgeRequiredRuntimeBuildRefV0      = "ORQUESTA_OPES_BRIDGE_REQUIRED_RUNTIME_BUILD_REF"
	envOPESBridgeRequiredRuntimeCommitRefV0     = "ORQUESTA_OPES_BRIDGE_REQUIRED_RUNTIME_COMMIT_REF"
	envOPESBridgeSpeechSynthesisCapabilityV0    = "ORQUESTA_OPES_BRIDGE_SPEECH_SYNTHESIS_CAPABILITY"
	envOPESBridgeSpeechSynthesisCapabilityRefV0 = "ORQUESTA_OPES_BRIDGE_SPEECH_SYNTHESIS_CAPABILITY_REF"
	envOPESBridgeSpeechSynthesisEvidenceRefsV0  = "ORQUESTA_OPES_BRIDGE_SPEECH_SYNTHESIS_EVIDENCE_REFS"
	envOPESBridgeSpeechSynthesisReasonV0        = "ORQUESTA_OPES_BRIDGE_SPEECH_SYNTHESIS_REASON"
	envOPESBridgeSpeechSynthesisNetworkReadyV0  = "ORQUESTA_OPES_BRIDGE_SPEECH_SYNTHESIS_NETWORK_READY"
	envOPESBridgeSpeechSynthesisToolPathReadyV0 = "ORQUESTA_OPES_BRIDGE_SPEECH_SYNTHESIS_TOOL_PATH_READY"
	envOPESBridgeSpeechSynthesisQuotaReadyV0    = "ORQUESTA_OPES_BRIDGE_SPEECH_SYNTHESIS_PROVIDER_QUOTA_READY"
	envOPESBridgeRemoteQACapabilityV0           = "ORQUESTA_OPES_BRIDGE_REMOTE_QA_CAPABILITY"
	envOPESBridgeRemoteQACapabilityRefV0        = "ORQUESTA_OPES_BRIDGE_REMOTE_QA_CAPABILITY_REF"
	envOPESBridgeRemoteQAEvidenceRefsV0         = "ORQUESTA_OPES_BRIDGE_REMOTE_QA_EVIDENCE_REFS"
	envOPESBridgeRemoteQAReasonV0               = "ORQUESTA_OPES_BRIDGE_REMOTE_QA_REASON"
	envOPESBridgeRemoteQANetworkReadyV0         = "ORQUESTA_OPES_BRIDGE_REMOTE_QA_NETWORK_READY"
	envOPESBridgeRemoteQAAuthStateReadyV0       = "ORQUESTA_OPES_BRIDGE_REMOTE_QA_AUTH_STATE_READY"
	envOPESBridgeRemoteQAQuotaReadyV0           = "ORQUESTA_OPES_BRIDGE_REMOTE_QA_PROVIDER_QUOTA_READY"
	envOPESBridgeAllowUnfilteredV0              = "ORQUESTA_OPES_BRIDGE_ALLOW_UNFILTERED"
	envOPESBridgeInputLedgerDisabledV0          = "ORQUESTA_OPES_BRIDGE_INPUT_LEDGER_DISABLED"
	envOPESBridgeInputLedgerPathV0              = "ORQUESTA_OPES_BRIDGE_INPUT_LEDGER_PATH"
	envOPESTemporalConfirmV0                    = "ORQUESTA_OPES_TEMPORAL_CONFIRM"
	envOPESBridgeProductiveConfirmV0            = "ORQUESTA_OPES_BRIDGE_PRODUCTIVE_CONFIRM"
	envOPESBridgeDestinationEvidenceV0          = "ORQUESTA_OPES_BRIDGE_DESTINATION_EVIDENCE_REF"
	envOPESRegistryFinalPkgEnabledV0            = "ORQUESTA_OPES_REGISTRY_FINALPKG_ENABLED"
	envOPESRegistryFinalPkgConfirmV0            = "ORQUESTA_OPES_REGISTRY_FINALPKG_CONFIRM"
	envOPESRegistryFinalPkgDryRunV0             = "ORQUESTA_OPES_REGISTRY_FINALPKG_DRY_RUN"
	envOPESRegistryFinalPkgRegistryV0           = "ORQUESTA_OPES_REGISTRY_FINALPKG_REGISTRY_PATH"
	envOPESRegistryFinalPkgCourseIDV0           = "ORQUESTA_OPES_REGISTRY_FINALPKG_COURSE_ID"
	envOPESRegistryFinalPkgCourseRootV0         = "ORQUESTA_OPES_REGISTRY_FINALPKG_COURSE_ROOT"
	envOPESRegistryFinalPkgTemplateRunV0        = "ORQUESTA_OPES_REGISTRY_FINALPKG_TEMPLATE_RUN_REF"
	envOPESRegistryFinalPkgTemplateTopicV0      = "ORQUESTA_OPES_REGISTRY_FINALPKG_TEMPLATE_TOPIC_ID"
	envOPESRegistryFinalPkgBatchSizeV0          = "ORQUESTA_OPES_REGISTRY_FINALPKG_BATCH_SIZE"
	envOPESRegistryFinalPkgMaxInFlightV0        = "ORQUESTA_OPES_REGISTRY_FINALPKG_MAX_IN_FLIGHT"
	envOPESRegistryFinalPkgIntervalV0           = "ORQUESTA_OPES_REGISTRY_FINALPKG_INTERVAL_SECONDS"
	envOPESRegistryFinalPkgMaxTicksV0           = "ORQUESTA_OPES_REGISTRY_FINALPKG_MAX_TICKS"
	envOPESRegistryFinalPkgQueueRefV0           = "ORQUESTA_OPES_REGISTRY_FINALPKG_QUEUE_REF"
	envOPESRegistryFinalPkgReconcileV0          = "ORQUESTA_OPES_REGISTRY_FINALPKG_RECONCILE_ENABLED"
	envOPESRegistryFinalPkgReconcileLimitV0     = "ORQUESTA_OPES_REGISTRY_FINALPKG_RECONCILE_LIMIT"
	envOPESTopicRegistryEnabledV0               = "ORQUESTA_OPES_TOPIC_REGISTRY_ENABLED"
	envOPESTopicRegistryToolPathV0              = "ORQUESTA_OPES_TOPIC_REGISTRY_TOOL_PATH"
	envOPESTopicRegistryAgentIDV0               = "ORQUESTA_OPES_TOPIC_REGISTRY_AGENT_ID"
	envOPESTopicRegistryForceV0                 = "ORQUESTA_OPES_TOPIC_REGISTRY_FORCE"
	envDomainWorkHTTPBaseURLV0                  = "ORQUESTA_DOMAIN_WORK_HTTP_BASE_URL"
	envDomainWorkHTTPDomainRefV0                = "ORQUESTA_DOMAIN_WORK_HTTP_DOMAIN_REF"
	envDomainWorkFileDirV0                      = "ORQUESTA_DOMAIN_WORK_FILE_DIR"
	envDomainWorkFileEnabledV0                  = "ORQUESTA_DOMAIN_WORK_FILE_ENABLED"
	envDomainWorkHTTPCreatePathV0               = "ORQUESTA_DOMAIN_WORK_HTTP_CREATE_PATH"
	envDomainWorkHTTPSubmitPathV0               = "ORQUESTA_DOMAIN_WORK_HTTP_SUBMIT_PATH"
	envDomainWorkHTTPTimeoutSecondsV0           = "ORQUESTA_DOMAIN_WORK_HTTP_TIMEOUT_SECONDS"
	envDomainWorkHTTPEgressModeV0               = "ORQUESTA_DOMAIN_WORK_HTTP_EGRESS_MODE"
	envDomainWorkHTTPAllowedHostsV0             = "ORQUESTA_DOMAIN_WORK_HTTP_ALLOWED_HOSTS"
	envRequiredTestRunnerEnabledV0              = "ORQUESTA_REQUIRED_TEST_RUNNER_ENABLED"
	envRequiredTestMaxOutputBytesV0             = "ORQUESTA_REQUIRED_TEST_MAX_OUTPUT_BYTES"
	envRequiredTestOutputMaxArtifactsV0         = "ORQUESTA_REQUIRED_TEST_OUTPUT_MAX_ARTIFACTS"
	envRequiredTestGoCommandV0                  = "ORQUESTA_REQUIRED_TEST_GO_COMMAND"
	envRequiredTestAllowedCommandsV0            = "ORQUESTA_REQUIRED_TEST_ALLOWED_COMMANDS"
	envRequiredTestOutputDirV0                  = "ORQUESTA_REQUIRED_TEST_OUTPUT_DIR"
	envRequiredTestEnvV0                        = "ORQUESTA_REQUIRED_TEST_ENV"

	defaultCodexWaitIntervalMSV0               = 2000
	defaultCodexStalledTicksV0                 = 300
	defaultCodexLoopTicksV0                    = 300
	defaultCodexMaxExpectedSecondsV0           = 1200
	defaultCodexNoActivitySecondsV0            = 600
	defaultCodexExecutionModeV0                = "parallel"
	defaultCodexMaxBatchReadyV0                = 70
	defaultCodexMaxConcurrencyV0               = 70
	defaultCodexGoalTimeoutMSV0                = 90000
	defaultCodexGoalPreflightTimeoutMSV0       = 3000
	defaultCodexServerMaxRunsPerTickV0         = 70
	defaultCodexServerQueueLimitV0             = 70
	defaultCodexServerDefaultPriorityV0        = 50
	defaultCodexServerMaxExecutionsV0          = 70
	defaultCodexDirectorWaveAgentsV0           = 70
	defaultCodexDirectorMaxSubagentsPerAgentV0 = 6
	defaultCodexDirectorRecursiveAgentBudgetV0 = 70
)

const (
	envOPESBridgeSpeechSynthesisToolWorkDirV0                 = "ORQUESTA_OPES_BRIDGE_SPEECH_SYNTHESIS_TOOL_WORKDIR"
	envOPESBridgeSpeechSynthesisToolCommandV0                 = "ORQUESTA_OPES_BRIDGE_SPEECH_SYNTHESIS_TOOL_COMMAND"
	envOPESBridgeSpeechSynthesisToolPreflightTimeoutSecondsV0 = "ORQUESTA_OPES_BRIDGE_SPEECH_SYNTHESIS_TOOL_PREFLIGHT_TIMEOUT_SECONDS"
	envOPESBridgeSpeechSynthesisProgressHeartbeatReadyV0      = "ORQUESTA_OPES_BRIDGE_SPEECH_SYNTHESIS_PROGRESS_HEARTBEAT_READY"
	envOPESBridgeSpeechSynthesisProviderTimeoutReadyV0        = "ORQUESTA_OPES_BRIDGE_SPEECH_SYNTHESIS_PROVIDER_TIMEOUT_READY"
	envOPESBridgeSpeechSynthesisNoProgressTimeoutSecondsV0    = "ORQUESTA_OPES_BRIDGE_SPEECH_SYNTHESIS_NO_PROGRESS_TIMEOUT_SECONDS"
)

type serverEnvSettingMetadataV0 struct {
	Scope       string
	Label       string
	Description string
}

var serverEffectiveEnvRegistryV0 = map[string]serverEnvSettingMetadataV0{
	envServerAddrV0: {
		Scope:       "server_bootstrap",
		Label:       "Direccion servidor",
		Description: "Direccion bind local del servidor Orquesta.",
	},
	envServerStateDirV0: {
		Scope:       "server_bootstrap",
		Label:       "Estado servidor",
		Description: "Directorio local donde el servidor conserva statefile y runtime operacional.",
	},
	envOrquestaServerURLV0: {
		Scope:       "server_endpoint",
		Label:       "URL servidor Orquesta",
		Description: "Endpoint canonico gestionado del servidor Orquesta para wrappers y conectores operadores.",
	},
	envOrquestaBaseURLV0: {
		Scope:       "server_endpoint",
		Label:       "Base URL Orquesta",
		Description: "Alias historico de ORQUESTA_SERVER_URL mantenido por compatibilidad con conectores existentes.",
	},
	envOrquestaRuntimeDirV0: {
		Scope:       "server_endpoint",
		Label:       "Runtime Orquesta",
		Description: "Directorio runtime gestionado desde el que los conectores pueden leer base_url.txt durable.",
	},
	envServerAuditFileV0: {
		Scope:       "audit",
		Label:       "Archivo audit",
		Description: "Archivo JSONL local donde el servidor registra eventos de auditoria.",
	},
	envServerAuditDisabledV0: {
		Scope:       "audit",
		Label:       "Audit desactivado",
		Description: "Opt-out explicito de escritura de auditoria local.",
	},
	envServerRemoteControlPlaneConfirmV0: {
		Scope:       "control_plane",
		Label:       "Control remoto opt-in",
		Description: "Confirmacion explicita requerida para exponer el control-plane fuera de loopback.",
	},
	envServerControlTokenV0: {
		Scope:       "control_plane",
		Label:       "Token control-plane",
		Description: "Presencia del token de control-plane; el valor real nunca se publica.",
	},
	envServerControlPrincipalV0: {
		Scope:       "control_plane",
		Label:       "Principal control-plane",
		Description: "Ref publica del principal autorizado para control-plane.",
	},
	envServerControlPermissionRefV0: {
		Scope:       "control_plane",
		Label:       "Permiso control-plane",
		Description: "Ref publica de permiso asociada al acceso de control-plane.",
	},
	envServerControlPublicReasonV0: {
		Scope:       "control_plane",
		Label:       "Motivo control-plane",
		Description: "Motivo publico registrado para el modo de acceso del control-plane.",
	},
	envServerSelfProgrammingOnlyV0: {
		Scope:       "self_programming",
		Label:       "Solo autoprogramacion",
		Description: "Perfil de composicion que permite solo autoprogramacion aislada de Orquesta y bloquea conectores productivos.",
	},
	envServerSelfProgrammingRootV0: {
		Scope:       "self_programming",
		Label:       "Raiz autoprogramacion",
		Description: "Raiz aislada bajo la que deben vivir proyecto, runtime y estado en modo self-programming.",
	},
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
	envServerQueueLimitV0: {
		Scope:       "server_supervisor",
		Label:       "Limite de cola",
		Description: "Runs activos que el supervisor residente considera por cola antes de posponer nuevos lanzamientos.",
	},
	envServerDrainMaxDispatchesV0: {
		Scope:       "server_supervisor",
		Label:       "Despachos por espera",
		Description: "Despachos de agentes que puede emitir cada drain del supervisor residente.",
	},
	envServerDrainMaxOutboxV0: {
		Scope:       "server_supervisor",
		Label:       "Outbox por ciclo",
		Description: "Comandos outbox que puede materializar cada ciclo del supervisor residente.",
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
	envServerIdleSelfImprovementAfterV0: {
		Scope:       "autoprogramming",
		Label:       "Espera idle segundos",
		Description: "Segundos sin ejecuciones antes de proponer automejora idle; 0 desactiva el disparador.",
	},
	envServerIdleSelfImprovementDisabledV0: {
		Scope:       "autoprogramming",
		Label:       "Automejora idle desactivada",
		Description: "Opt-out explicito de toda automejora residente por idle o capacidad libre.",
	},
	envServerIdleSelfImprovementTargetQueueV0: {
		Scope:       "autoprogramming",
		Label:       "Cola objetivo",
		Description: "Tamano objetivo de cola de automejora.",
	},
	envServerIdleSelfImprovementProjectWorkDirV0: {
		Scope:       "autoprogramming",
		Label:       "Repo automejora",
		Description: "Directorio de trabajo usado solo por automejora idle; separa el repo de Orquesta del proyecto externo.",
	},
	envServerIdleSelfImprovementMaxRequestsV0: {
		Scope:       "autoprogramming",
		Label:       "Nuevas tareas por tanda",
		Description: "Maximo de tareas nuevas por tanda de automejora.",
	},
	envServerIdleSelfImprovementGoalFirstV0: {
		Scope:       "autoprogramming",
		Label:       "Goal-first automejora",
		Description: "Activa modo goal-first estricto para automejora residente; si no se define y hay backend Codex Goal, se deriva automaticamente.",
	},
	envServerIdleSelfImprovementFrozenTestsV0: {
		Scope:       "autoprogramming",
		Label:       "Tests congelados",
		Description: "Opt-in para separar goal definidor de required tests y goal implementador con hashes congelados.",
	},
	envServerIdleSelfImprovementDailyGoalBudgetV0: {
		Scope:       "autoprogramming",
		Label:       "Presupuesto diario goals",
		Description: "Maximo diario opt-in de goals de automejora idle; 0 deja comportamiento compatible sin limite.",
	},
	envServerIdleSelfImprovementDailyContextBudgetBytesV0: {
		Scope:       "autoprogramming",
		Label:       "Presupuesto diario contexto",
		Description: "Maximo diario opt-in de bytes de contexto 801A para automejora idle; 0 deja comportamiento compatible sin limite.",
	},
	envAutoprogrammingLegacyDirectorLoopV0: {
		Scope:       "autoprogramming",
		Label:       "Autoprogramming legacy",
		Description: "Breakglass opt-in para permitir que autoprogramming prepare runs del Director historico; por defecto false.",
	},
	envAutoprogrammingCheckpointOnlyHighConsumptionTokensV0: {
		Scope:       "autoprogramming",
		Label:       "Umbral checkpoint",
		Description: "Tokens a partir de los que un Goal activo con solo checkpoint se trata como alto consumo y requiere replan acotado.",
	},
	envAutoprogrammingCheckpointOnlyMaxWaitSecondsV0: {
		Scope:       "autoprogramming",
		Label:       "Espera checkpoint",
		Description: "Segundos maximos que un Goal activo puede permanecer solo con checkpoint tras timeout local antes de requerir replan acotado.",
	},
	envAutoprogrammingNoCheckpointWarningMaxWaitSecondsV0: {
		Scope:       "autoprogramming",
		Label:       "Espera sin checkpoint",
		Description: "Segundos maximos que un Goal activo puede permanecer en warning de consumo sin checkpoint antes de requerir replan acotado.",
	},
	envExternalWorkLegacyDirectorLoopV0: {
		Scope:       "external_work",
		Label:       "External work legacy",
		Description: "Breakglass opt-in para permitir que external_work.run use el loop historico si no hay backend Goal; por defecto false.",
	},
	envCodebaseBrokerProviderKindV0: {
		Scope:       "codebase_broker",
		Label:       "Proveedor Codebase",
		Description: "Proveedor central de contexto de codigo; por defecto fallback_rg. codebase_memory_mcp queda reservado a opt-in central.",
	},
	envCodebaseBrokerExternalIndexerEnabledV0: {
		Scope:       "codebase_broker",
		Label:       "Indexador externo",
		Description: "Permite al broker central usar indexador externo opt-in; no habilita MCPs libres por agente.",
	},
	envCodebaseBrokerMaxConcurrentV0: {
		Scope:       "codebase_broker",
		Label:       "Consultas contexto",
		Description: "Maximo de consultas concurrentes del broker central de contexto de codigo.",
	},
	envCodebaseBrokerTimeoutMSV0: {
		Scope:       "codebase_broker",
		Label:       "Timeout contexto ms",
		Description: "Timeout por consulta del broker central de contexto de codigo.",
	},
	envCodebaseBrokerStateDirV0: {
		Scope:       "codebase_broker",
		Label:       "Estado contexto",
		Description: "Directorio opt-in para persistir cache y leases del broker central de contexto de codigo.",
	},
	envCodebaseBrokerWatchdogEnabledV0: {
		Scope:       "codebase_broker",
		Label:       "Watchdog contexto",
		Description: "Habilita watchdog opt-in de leases codebase-memory-mcp con parada cooperativa por owner marker.",
	},
	envCodebaseBrokerWatchdogStopOrphansV0: {
		Scope:       "codebase_broker",
		Label:       "Parar huerfanos Codebase",
		Description: "Permite al watchdog parar con SIGTERM procesos codebase-memory-mcp antiguos sin owner marker de Orquesta.",
	},
	envCodebaseBrokerWatchdogOrphanMinAgeSecondsV0: {
		Scope:       "codebase_broker",
		Label:       "Edad huerfano Codebase",
		Description: "Edad minima en segundos antes de considerar huerfano un proceso codebase-memory-mcp sin owner marker.",
	},
	envCodebaseBrokerCommandV0: {
		Scope:       "codebase_broker",
		Label:       "Comando Codebase",
		Description: "Comando opt-in usado por el broker central para invocar codebase-memory-mcp cli; por defecto codebase-memory-mcp.",
	},
	envCodebaseBrokerProjectNameV0: {
		Scope:       "codebase_broker",
		Label:       "Proyecto Codebase",
		Description: "Nombre de proyecto indexado en codebase-memory-mcp; si falta se deriva del project workdir.",
	},
	envSecurityModeV0: {Scope: "rails", Label: "Modo seguridad", Description: "Modo historico de seguridad; no reactiva rails offline hasta nueva orden."},
	envRailsModeV0:    {Scope: "rails", Label: "Modo rails", Description: "offline fijo hasta nueva orden; no reactiva politicas de bloqueo."},
	envDetailProhibitedRailsV0: {
		Scope:       "rails",
		Label:       "Rails detalle",
		Description: "off fijo hasta nueva orden; los rails de detalle no se reactivan por env.",
	},
	envDetailProhibitedRailsScopeV0: {
		Scope:       "rails",
		Label:       "Scope rails detalle",
		Description: "Inventario historico de scopes; no aplica bloqueo mientras los rails esten offline.",
	},
	envCodexMaxBatchReadyV0: {
		Scope:       "codex_runtime",
		Label:       "Batch Codex ready",
		Description: "Agentes ready a despachar por tanda Codex.",
	},
	envCodexExecutionModeV0: {
		Scope:       "codex_runtime",
		Label:       "Modo ejecucion Codex",
		Description: "parallel respeta limites configurados; serial fuerza ejecucion de agentes de uno en uno.",
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
	envCodexHomeV0: {
		Scope:       "codex_runtime",
		Label:       "CODEX_HOME alias Orquesta",
		Description: "Alias historico de ORQUESTA_CODEX_CODE_HOME mantenido por compatibilidad; la fuente canonica de auth/config es ORQUESTA_CODEX_CODE_HOME.",
	},
	envCodexRuntimeWorkDirV0: {
		Scope:       "codex_runtime",
		Label:       "Runtime Codex",
		Description: "Directorio de trabajo runtime para procesos Codex gestionados por la composicion.",
	},
	envCodexCodeHomeV0: {
		Scope:       "codex_runtime",
		Label:       "CODEX_HOME fuente",
		Description: "Directorio fuente de auth.json/config.toml para backends Codex; ORQUESTA_CODEX_HOME y CODEX_HOME se conservan solo como aliases legacy.",
	},
	envCodexPromoteMaterializedArtifactWithoutAckV0: {
		Scope:       "codex_runtime",
		Label:       "Promocion sin ACK",
		Description: "Promueve artefactos materializados sin agent_ack.json con gate-issue de revision; por defecto true en servidor.",
	},
	envCodexGoalBackendV0: {
		Scope:       "codex_goal",
		Label:       "Backend Codex Goal",
		Description: "Backend opt-in para lanzar y observar Codex Goal desde la composicion: app_server_tmux es la unica ruta operativa normal; app_server_proxy no crea backend y solo devuelve diagnostico no operacional.",
	},
	envAllowAppServerProxyDiagnosticV0: {
		Scope:       "codex_goal",
		Label:       "Proxy diagnostico",
		Description: "Permite comprobar app_server_proxy como diagnostico no operacional; no lanza goals ni sustituye app_server_tmux.",
	},
	envCodexGoalTimeoutMSV0: {
		Scope:       "codex_goal",
		Label:       "Timeout Codex Goal",
		Description: "Timeout por llamada al backend Codex Goal app-server.",
	},
	envCodexGoalPreflightTimeoutMSV0: {
		Scope:       "codex_goal",
		Label:       "Preflight Codex Goal",
		Description: "Timeout de comprobacion rapida del backend Codex Goal app-server al montar la composicion.",
	},
	envSelfAuditBacklogEnabledV0: {
		Scope:       "autoprogramming",
		Label:       "Backlog autoauditoria",
		Description: "Activa una fuente opt-in que convierte hallazgos de auditoria local en backlog goal-first de automejora.",
	},
	envOPESBridgeWaitResidentSecondsV0: {
		Scope:       "opes_bridge",
		Label:       "Espera residente OPES",
		Description: "Segundos que opes-drain-once observa director/stats para ver dispatch residente sin supervisar manualmente.",
	},
	envOPESBridgeWaitResidentIntervalMSV0: {
		Scope:       "opes_bridge",
		Label:       "Intervalo espera OPES",
		Description: "Intervalo en milisegundos entre lecturas pasivas de director/stats durante la espera residente OPES.",
	},
	envOPESBaseURLV0: {
		Scope:       "opes_bridge",
		Label:       "Base URL OPES",
		Description: "Endpoint de la instancia OPES temporal/productiva autorizada; OPES_BASE_URL se conserva solo como alias legacy.",
	},
	envOPESBridgeRequireRuntimeCompatibilityV0: {
		Scope:       "opes_bridge",
		Label:       "Runtime compatible OPES",
		Description: "Opt-in para exigir readiness e identidad runtime compatible antes de postear external-work desde OPES bridge.",
	},
	envOPESBridgeRequiredRuntimeBinarySHA256V0: {
		Scope:       "opes_bridge",
		Label:       "Runtime SHA OPES",
		Description: "SHA256 de binario Orquesta requerido por OPES bridge cuando la compatibilidad runtime esta habilitada.",
	},
	envOPESBridgeRequiredRuntimeBuildRefV0: {
		Scope:       "opes_bridge",
		Label:       "Runtime build OPES",
		Description: "Build ref Orquesta requerido por OPES bridge cuando la compatibilidad runtime esta habilitada.",
	},
	envOPESBridgeRequiredRuntimeCommitRefV0: {
		Scope:       "opes_bridge",
		Label:       "Runtime commit OPES",
		Description: "Commit ref Orquesta requerido por OPES bridge cuando la compatibilidad runtime esta habilitada.",
	},
	envOPESBridgeSpeechSynthesisCapabilityV0: {
		Scope:       "opes_bridge",
		Label:       "Speech synthesis OPES",
		Description: "Declaracion opt-in de capacidad speech_synthesis para trabajos OPES que generan audio_asset.",
	},
	envOPESBridgeSpeechSynthesisCapabilityRefV0: {
		Scope:       "opes_bridge",
		Label:       "Speech synthesis ref",
		Description: "Ref opaca de la capacidad speech_synthesis declarada por la composicion OPES bridge.",
	},
	envOPESBridgeSpeechSynthesisEvidenceRefsV0: {
		Scope:       "opes_bridge",
		Label:       "Speech synthesis evidencias",
		Description: "Refs de evidencia asociadas a la capacidad speech_synthesis declarada por la composicion OPES bridge.",
	},
	envOPESBridgeSpeechSynthesisReasonV0: {
		Scope:       "opes_bridge",
		Label:       "Speech synthesis razon",
		Description: "Razon operativa cuando speech_synthesis no esta disponible o se declara de forma invalida.",
	},
	envOPESBridgeSpeechSynthesisNetworkReadyV0: {
		Scope:       "opes_bridge",
		Label:       "Speech synthesis red",
		Description: "Subcheck opt-in para declarar si speech_synthesis tiene red disponible.",
	},
	envOPESBridgeSpeechSynthesisToolPathReadyV0: {
		Scope:       "opes_bridge",
		Label:       "Speech synthesis herramienta",
		Description: "Subcheck opt-in para declarar si la herramienta TTS/audio esta disponible en la composicion.",
	},
	envOPESBridgeSpeechSynthesisQuotaReadyV0: {
		Scope:       "opes_bridge",
		Label:       "Speech synthesis cuota",
		Description: "Subcheck opt-in para declarar si el proveedor TTS tiene cuota/capacidad disponible.",
	},
	envOPESBridgeSpeechSynthesisProgressHeartbeatReadyV0: {
		Scope:       "opes_bridge",
		Label:       "Speech synthesis progreso",
		Description: "Subcheck opt-in para declarar si el proveedor TTS publica heartbeat/progreso granular observable.",
	},
	envOPESBridgeSpeechSynthesisProviderTimeoutReadyV0: {
		Scope:       "opes_bridge",
		Label:       "Speech synthesis provider timeout",
		Description: "Subcheck opt-in para declarar si el proveedor TTS corta sin avance y publica provider_timeout recuperable.",
	},
	envOPESBridgeSpeechSynthesisNoProgressTimeoutSecondsV0: {
		Scope:       "opes_bridge",
		Label:       "Speech synthesis no-avance",
		Description: "Ventana maxima sin progreso en segundos para cortar proveedor TTS antes de bloquear una ola OPES.",
	},
	envOPESBridgeSpeechSynthesisToolWorkDirV0: {
		Scope:       "opes_bridge",
		Label:       "Speech synthesis workdir",
		Description: "Directorio local opt-in donde Orquesta ejecuta el preflight de scripts/opes_audio_app.py antes de lanzar audio OPES.",
	},
	envOPESBridgeSpeechSynthesisToolCommandV0: {
		Scope:       "opes_bridge",
		Label:       "Speech synthesis preflight",
		Description: "Comando local opt-in de preflight TTS/audio OPES; por defecto python3 scripts/opes_audio_app.py --help.",
	},
	envOPESBridgeSpeechSynthesisToolPreflightTimeoutSecondsV0: {
		Scope:       "opes_bridge",
		Label:       "Speech synthesis timeout",
		Description: "Timeout en segundos para el preflight local de la herramienta TTS/audio OPES.",
	},
	envOPESBridgeRemoteQACapabilityV0: {
		Scope:       "opes_bridge",
		Label:       "Remote QA OPES",
		Description: "Declaracion opt-in de capacidad remote_qa_provider para revisiones OPES que requieren agente externo.",
	},
	envOPESBridgeRemoteQACapabilityRefV0: {
		Scope:       "opes_bridge",
		Label:       "Remote QA ref",
		Description: "Ref opaca de la capacidad remote_qa_provider declarada por la composicion OPES bridge.",
	},
	envOPESBridgeRemoteQAEvidenceRefsV0: {
		Scope:       "opes_bridge",
		Label:       "Remote QA evidencias",
		Description: "Refs de evidencia asociadas a la capacidad remote_qa_provider declarada por la composicion OPES bridge.",
	},
	envOPESBridgeRemoteQAReasonV0: {
		Scope:       "opes_bridge",
		Label:       "Remote QA razon",
		Description: "Razon operativa cuando remote_qa_provider no esta disponible o se declara de forma invalida.",
	},
	envOPESBridgeRemoteQANetworkReadyV0: {
		Scope:       "opes_bridge",
		Label:       "Remote QA red",
		Description: "Subcheck opt-in para declarar si remote_qa_provider tiene red disponible.",
	},
	envOPESBridgeRemoteQAAuthStateReadyV0: {
		Scope:       "opes_bridge",
		Label:       "Remote QA auth",
		Description: "Subcheck opt-in para declarar si remote_qa_provider tiene sesion/auth vigente.",
	},
	envOPESBridgeRemoteQAQuotaReadyV0: {
		Scope:       "opes_bridge",
		Label:       "Remote QA cuota",
		Description: "Subcheck opt-in para declarar si remote_qa_provider tiene cuota/capacidad disponible.",
	},
	envCapacityReasoningEffortV0: {
		Scope:       "capacity",
		Label:       "Reasoning capacidad",
		Description: "Recomendacion de capacidad que recibe el stack.",
	},
	envCapacityPolicyRefV0: {
		Scope:       "capacity",
		Label:       "Politica capacidad",
		Description: "Ref opaca de la politica de capacidad inyectada por la composicion.",
	},
	envCapacityPoolRefV0: {
		Scope:       "capacity",
		Label:       "Pool capacidad",
		Description: "Ref opaca del pool de capacidad usado por la politica.",
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
	envOPESRegistryFinalPkgEnabledV0: {
		Scope:       "opes_registry_finalpkg",
		Label:       "Productor finalpkg",
		Description: "Activa el productor residente opt-in de paquetes finales desde registro local OPES.",
	},
	envOPESRegistryFinalPkgConfirmV0: {
		Scope:       "opes_registry_finalpkg",
		Label:       "Confirmacion finalpkg",
		Description: "Confirmacion explicita requerida para crear runs desde el registro local.",
	},
	envOPESRegistryFinalPkgDryRunV0: {
		Scope:       "opes_registry_finalpkg",
		Label:       "Dry-run finalpkg",
		Description: "Mantiene el productor en modo simulacion; por defecto no crea runs.",
	},
	envOPESRegistryFinalPkgRegistryV0: {
		Scope:       "opes_registry_finalpkg",
		Label:       "Registro OPES",
		Description: "Ruta local del registro compartido de temas OPES; se publica solo como presencia configurada.",
	},
	envOPESRegistryFinalPkgCourseIDV0: {
		Scope:       "opes_registry_finalpkg",
		Label:       "Curso OPES",
		Description: "Identificador del curso dentro del registro local de temas.",
	},
	envOPESRegistryFinalPkgCourseRootV0: {
		Scope:       "opes_registry_finalpkg",
		Label:       "Raiz curso",
		Description: "Raiz local del curso para detectar paquetes completos; se publica solo como presencia configurada.",
	},
	envOPESRegistryFinalPkgTemplateRunV0: {
		Scope:       "opes_registry_finalpkg",
		Label:       "Run plantilla",
		Description: "Run existente usado como plantilla para construir nuevos paquetes finales.",
	},
	envOPESRegistryFinalPkgTemplateTopicV0: {
		Scope:       "opes_registry_finalpkg",
		Label:       "Tema plantilla",
		Description: "Topic id que se sustituye al clonar la plantilla de paquete final.",
	},
	envOPESRegistryFinalPkgBatchSizeV0: {
		Scope:       "opes_registry_finalpkg",
		Label:       "Tanda finalpkg",
		Description: "Maximo de temas nuevos que el productor intenta lanzar por tick.",
	},
	envOPESRegistryFinalPkgMaxInFlightV0: {
		Scope:       "opes_registry_finalpkg",
		Label:       "Finalpkg en vuelo",
		Description: "Maximo de runs finalpkg no terminales antes de pausar nuevos lanzamientos.",
	},
	envOPESRegistryFinalPkgIntervalV0: {
		Scope:       "opes_registry_finalpkg",
		Label:       "Intervalo finalpkg",
		Description: "Segundos entre ticks del productor residente de paquetes finales.",
	},
	envOPESRegistryFinalPkgMaxTicksV0: {
		Scope:       "opes_registry_finalpkg",
		Label:       "Ticks finalpkg",
		Description: "Limite opcional de ticks del productor residente; cero significa continuo.",
	},
	envOPESRegistryFinalPkgQueueRefV0: {
		Scope:       "opes_registry_finalpkg",
		Label:       "Cola finalpkg",
		Description: "Cola Orquesta donde se encolan los runs generados desde el registro.",
	},
	envOPESRegistryFinalPkgReconcileV0: {
		Scope:       "opes_registry_finalpkg",
		Label:       "Reconciliacion finalpkg",
		Description: "Activa el cierre causal de runs finalpkg con paquete completo pero estado no terminal.",
	},
	envOPESRegistryFinalPkgReconcileLimitV0: {
		Scope:       "opes_registry_finalpkg",
		Label:       "Limite reconciliacion finalpkg",
		Description: "Maximo de runs finalpkg completos que el reconciliador intenta cerrar por tick.",
	},
	envOPESTopicRegistryEnabledV0: {
		Scope:       "opes_topic_registry",
		Label:       "Registro por tema",
		Description: "Activa el conector causal de registro OPES por course_id y topic_id.",
	},
	envOPESTopicRegistryToolPathV0: {
		Scope:       "opes_topic_registry",
		Label:       "Herramienta registro",
		Description: "Ruta de la herramienta oficial registro_trabajo_temas.py; se publica solo como presencia configurada.",
	},
	envOPESTopicRegistryAgentIDV0: {
		Scope:       "opes_topic_registry",
		Label:       "Agente registro",
		Description: "Identificador que usa Orquesta al ejecutar update/release del registro por tema.",
	},
	envOPESTopicRegistryForceV0: {
		Scope:       "opes_topic_registry",
		Label:       "Force registro",
		Description: "Permite --force en la herramienta oficial tras configuracion explicita del operador.",
	},
}

func serverConfigSettingFromRegistryV0(key string, value string) orquestaserver.ServerConfigSettingV0 {
	metadata := serverEffectiveEnvRegistryV0[key]
	return serverConfigSettingV0(key, value, metadata.Scope, metadata.Label, metadata.Description)
}

func serverConfigSettingFromRegistryWithSourceV0(key string, value string, source string) orquestaserver.ServerConfigSettingV0 {
	setting := serverConfigSettingFromRegistryV0(key, value)
	source = strings.TrimSpace(source)
	if source != "" {
		setting.Source = source
	}
	return setting
}

func serverSensitiveConfigSettingFromRegistryV0(key string, value string) orquestaserver.ServerConfigSettingV0 {
	setting := serverConfigSettingFromRegistryV0(key, value)
	setting.Sensitive = true
	return setting
}

func serverSensitiveConfigSettingFromRegistryWithSourceV0(key string, value string, source string) orquestaserver.ServerConfigSettingV0 {
	setting := serverConfigSettingFromRegistryWithSourceV0(key, value, source)
	setting.Sensitive = true
	return setting
}
