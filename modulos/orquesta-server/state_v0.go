package orquestaserver

import (
	orquestaautoprogramming "orquesta/modulos/orquesta-autoprogramming"
	orquestagoal "orquesta/modulos/orquesta-goal"
)

const StateSchemaVersionV0 = "orquesta_server_state.v0"

type StateV0 struct {
	SchemaVersion                                     string                                                                     `json:"schema_version"`
	Status                                            string                                                                     `json:"status"`
	PID                                               int                                                                        `json:"pid"`
	Addr                                              string                                                                     `json:"addr"`
	ProcessRef                                        string                                                                     `json:"process_ref,omitempty"`
	DaemonEpochRef                                    string                                                                     `json:"daemon_epoch_ref,omitempty"`
	ProjectWorkDir                                    string                                                                     `json:"project_work_dir,omitempty"`
	RuntimeWorkDir                                    string                                                                     `json:"runtime_work_dir,omitempty"`
	DaemonLogPolicy                                   DaemonLogPolicyV0                                                          `json:"daemon_log_policy,omitempty"`
	ShutdownSignalPolicy                              ShutdownSignalPolicyV0                                                     `json:"shutdown_signal_policy,omitempty"`
	StartedAt                                         string                                                                     `json:"started_at,omitempty"`
	LastHeartbeatAt                                   string                                                                     `json:"last_heartbeat_at,omitempty"`
	StartupStatus                                     string                                                                     `json:"startup_status,omitempty"`
	StartupReady                                      bool                                                                       `json:"startup_ready,omitempty"`
	StartupMessage                                    string                                                                     `json:"startup_message,omitempty"`
	StartupOperationalMessage                         *ServerOperationalMessageV0                                                `json:"startup_operational_message,omitempty"`
	StartupRevision                                   StartupRevisionSummaryV0                                                   `json:"startup_revision,omitempty"`
	StartupBlockers                                   []StartupBlockerV0                                                         `json:"startup_blockers,omitempty"`
	EffectiveConfig                                   ServerEffectiveConfigV0                                                    `json:"effective_config,omitempty"`
	RuntimeIdentity                                   ServerRuntimeIdentityV0                                                    `json:"runtime_identity,omitempty"`
	LastStartupCheckAt                                string                                                                     `json:"last_startup_check_at,omitempty"`
	LastSupervisorAt                                  string                                                                     `json:"last_supervisor_at,omitempty"`
	LastSupervisorStatus                              string                                                                     `json:"last_supervisor_status,omitempty"`
	LastSupervisorStop                                string                                                                     `json:"last_supervisor_stop,omitempty"`
	LastSupervisorStopPublic                          string                                                                     `json:"last_supervisor_stop_public,omitempty"`
	LastSupervisorStopCategory                        string                                                                     `json:"last_supervisor_stop_category,omitempty"`
	LastSupervisorError                               string                                                                     `json:"last_supervisor_error,omitempty"`
	LastSupervisorOperationalMessage                  *ServerOperationalMessageV0                                                `json:"last_supervisor_operational_message,omitempty"`
	SupervisorLastErrorAt                             string                                                                     `json:"supervisor_last_error_at,omitempty"`
	SupervisorLastError                               string                                                                     `json:"supervisor_last_error,omitempty"`
	LastSupervisorQueueRef                            string                                                                     `json:"last_supervisor_queue_ref,omitempty"`
	LastSupervisorQueueSize                           int                                                                        `json:"last_supervisor_queue_size,omitempty"`
	LastSupervisorTickNumber                          int                                                                        `json:"last_supervisor_tick_number,omitempty"`
	LastSupervisorResultTicks                         int                                                                        `json:"last_supervisor_result_ticks,omitempty"`
	LastSupervisorExecutions                          int                                                                        `json:"last_supervisor_executions,omitempty"`
	LastSupervisorSkips                               int                                                                        `json:"last_supervisor_skips,omitempty"`
	SupervisorTickActive                              bool                                                                       `json:"supervisor_tick_active,omitempty"`
	SupervisorFrozen                                  bool                                                                       `json:"supervisor_frozen,omitempty"`
	SupervisorTicks                                   int                                                                        `json:"supervisor_ticks"`
	SupervisorErrorTicks                              int                                                                        `json:"supervisor_error_ticks,omitempty"`
	SupervisorExecutions                              int                                                                        `json:"supervisor_executions,omitempty"`
	SupervisorSkips                                   int                                                                        `json:"supervisor_skips,omitempty"`
	ShutdownInProgress                                bool                                                                       `json:"shutdown_in_progress,omitempty"`
	ShutdownStatus                                    string                                                                     `json:"shutdown_status,omitempty"`
	ShutdownReady                                     bool                                                                       `json:"shutdown_ready"`
	ShutdownHTTPStatus                                int                                                                        `json:"shutdown_http_status,omitempty"`
	ShutdownRunsRequested                             int                                                                        `json:"shutdown_runs_requested,omitempty"`
	ShutdownRunsStopped                               int                                                                        `json:"shutdown_runs_stopped,omitempty"`
	ShutdownAgentsInFlight                            int                                                                        `json:"shutdown_agents_in_flight,omitempty"`
	ShutdownCheckpointsPending                        int                                                                        `json:"shutdown_checkpoints_pending,omitempty"`
	ShutdownCheckpointAgentsPending                   int                                                                        `json:"shutdown_checkpoint_agents_pending,omitempty"`
	ShutdownActiveWorkCount                           int                                                                        `json:"shutdown_active_work_count,omitempty"`
	ShutdownActiveWorkRefs                            []string                                                                   `json:"shutdown_active_work_refs,omitempty"`
	ShutdownGoalActions                               []ShutdownGoalActionV0                                                     `json:"shutdown_goal_actions,omitempty"`
	ShutdownAsyncWorkActive                           int                                                                        `json:"shutdown_async_work_active,omitempty"`
	ShutdownStopTimeoutAt                             string                                                                     `json:"shutdown_stop_timeout_at,omitempty"`
	ShutdownSignalName                                string                                                                     `json:"shutdown_signal_name,omitempty"`
	ShutdownSignalCount                               int                                                                        `json:"shutdown_signal_count,omitempty"`
	ShutdownSignalEscalated                           bool                                                                       `json:"shutdown_signal_escalated,omitempty"`
	LastShutdownAt                                    string                                                                     `json:"last_shutdown_at,omitempty"`
	IdleSelfImprovementAfter                          string                                                                     `json:"idle_self_improvement_after,omitempty"`
	IdleSelfImprovementTarget                         int                                                                        `json:"idle_self_improvement_target_queue,omitempty"`
	IdleSelfImprovementCheck                          string                                                                     `json:"idle_self_improvement_check,omitempty"`
	IdleSelfImprovementReason                         string                                                                     `json:"idle_self_improvement_reason,omitempty"`
	IdleSelfImprovementOperationalMessage             *ServerOperationalMessageV0                                                `json:"idle_self_improvement_operational_message,omitempty"`
	IdleSelfImprovementGoalSpec                       *orquestagoal.GoalWorkSpecV0                                               `json:"idle_self_improvement_goal_spec,omitempty"`
	IdleSelfImprovementGoalReceipt                    *orquestagoal.GoalLaunchReceiptV0                                          `json:"idle_self_improvement_goal_receipt,omitempty"`
	IdleSelfImprovementGoalResult                     *orquestagoal.GoalWorkResultV0                                             `json:"idle_self_improvement_goal_result,omitempty"`
	IdleSelfImprovementGoalClosure                    *orquestagoal.GoalClosureValidationV0                                      `json:"idle_self_improvement_goal_closure,omitempty"`
	IdleSelfImprovementGoalUsefulProgressAt           string                                                                     `json:"idle_self_improvement_goal_useful_progress_at,omitempty"`
	IdleSelfImprovementGoalUsefulProgressSignature    string                                                                     `json:"idle_self_improvement_goal_useful_progress_signature,omitempty"`
	IdleSelfImprovementGoalObservedConsumption        int64                                                                      `json:"idle_self_improvement_goal_observed_consumption,omitempty"`
	IdleSelfImprovementGoalInvalidCheckpointSignature string                                                                     `json:"idle_self_improvement_goal_invalid_checkpoint_signature,omitempty"`
	IdleSelfImprovementGoalInvalidCheckpointRepeats   int                                                                        `json:"idle_self_improvement_goal_invalid_checkpoint_repeats,omitempty"`
	IdleSelfImprovementBudget                         orquestaautoprogramming.AutoprogrammingIdleSelfImprovementBudgetDecisionV0 `json:"idle_self_improvement_budget,omitempty"`
	IdleSelfImprovementFlight                         bool                                                                       `json:"idle_self_improvement_in_flight,omitempty"`
	IdleSelfImprovementRuns                           int                                                                        `json:"idle_self_improvement_runs,omitempty"`
	IdleSelfImprovementOK                             int                                                                        `json:"idle_self_improvement_ok,omitempty"`
	EscalationDirectorDay                             string                                                                     `json:"escalation_director_day,omitempty"`
	EscalationDirectorInvocationsToday                int                                                                        `json:"escalation_director_invocations_today,omitempty"`
	EscalationDirectorLastSignature                   string                                                                     `json:"escalation_director_last_signature,omitempty"`
	EscalationDirectorLastDecision                    string                                                                     `json:"escalation_director_last_decision,omitempty"`
	EscalationDirectorLastReason                      string                                                                     `json:"escalation_director_last_reason,omitempty"`
	EscalationDirectorLastAt                          string                                                                     `json:"escalation_director_last_at,omitempty"`
	ResidentDirectorStatus                            string                                                                     `json:"resident_director_status,omitempty"`
	ResidentDirectorTickActive                        bool                                                                       `json:"resident_director_tick_active,omitempty"`
	ResidentDirectorLastTickAt                        string                                                                     `json:"resident_director_last_tick_at,omitempty"`
	ResidentDirectorLastSuccessAt                     string                                                                     `json:"resident_director_last_success_at,omitempty"`
	ResidentDirectorLastErrorAt                       string                                                                     `json:"resident_director_last_error_at,omitempty"`
	ResidentDirectorLastError                         string                                                                     `json:"resident_director_last_error,omitempty"`
	ResidentDirectorLastRunRef                        string                                                                     `json:"resident_director_last_run_ref,omitempty"`
	ResidentDirectorLastResult                        string                                                                     `json:"resident_director_last_result,omitempty"`
	ResidentDirectorTicks                             int                                                                        `json:"resident_director_ticks,omitempty"`
	ResidentDirectorErrorTicks                        int                                                                        `json:"resident_director_error_ticks,omitempty"`
	ResidentDirectorExecutedActions                   int                                                                        `json:"resident_director_executed_actions,omitempty"`
	ResidentDirectorOperationalMessage                *ServerOperationalMessageV0                                                `json:"resident_director_operational_message,omitempty"`
	GoalObserverStatus                                string                                                                     `json:"goal_observer_status,omitempty"`
	GoalObserverTickActive                            bool                                                                       `json:"goal_observer_tick_active,omitempty"`
	GoalObserverLastTickAt                            string                                                                     `json:"goal_observer_last_tick_at,omitempty"`
	GoalObserverLastSuccessAt                         string                                                                     `json:"goal_observer_last_success_at,omitempty"`
	GoalObserverLastErrorAt                           string                                                                     `json:"goal_observer_last_error_at,omitempty"`
	GoalObserverLastError                             string                                                                     `json:"goal_observer_last_error,omitempty"`
	GoalObserverTicks                                 int                                                                        `json:"goal_observer_ticks,omitempty"`
	GoalObserverErrorTicks                            int                                                                        `json:"goal_observer_error_ticks,omitempty"`
	GoalObserverObserved                              int                                                                        `json:"goal_observer_observed,omitempty"`
	GoalObserverTerminal                              int                                                                        `json:"goal_observer_terminal,omitempty"`
	GoalObserverIssues                                int                                                                        `json:"goal_observer_issues,omitempty"`
	GoalObserverOperationalMessage                    *ServerOperationalMessageV0                                                `json:"goal_observer_operational_message,omitempty"`
	ExternalBridgeComponent                           string                                                                     `json:"external_bridge_component,omitempty"`
	ExternalBridgeStatus                              string                                                                     `json:"external_bridge_status,omitempty"`
	ExternalBridgeTickActive                          bool                                                                       `json:"external_bridge_tick_active,omitempty"`
	ExternalBridgeLastTickRef                         string                                                                     `json:"external_bridge_last_tick_ref,omitempty"`
	ExternalBridgeLastTickAt                          string                                                                     `json:"external_bridge_last_tick_at,omitempty"`
	ExternalBridgeLastSuccess                         string                                                                     `json:"external_bridge_last_success_at,omitempty"`
	ExternalBridgeLastErrorAt                         string                                                                     `json:"external_bridge_last_error_at,omitempty"`
	ExternalBridgeLastError                           string                                                                     `json:"external_bridge_last_error_code,omitempty"`
	ExternalBridgeOperationalMessage                  *ServerOperationalMessageV0                                                `json:"external_bridge_operational_message,omitempty"`
	ExternalBridgeStopReason                          string                                                                     `json:"external_bridge_stop_reason,omitempty"`
	ExternalBridgeTicks                               int                                                                        `json:"external_bridge_ticks,omitempty"`
	ExternalBridgeErrorTicks                          int                                                                        `json:"external_bridge_error_ticks,omitempty"`
	ExternalBridgeFilters                             []string                                                                   `json:"external_bridge_filters,omitempty"`
	ExternalBridgeCounters                            map[string]int                                                             `json:"external_bridge_counters,omitempty"`
	ExternalBridgeEvidenceRefs                        []string                                                                   `json:"external_bridge_evidence_refs,omitempty"`
	SelfWatchdogStatus                                string                                                                     `json:"self_watchdog_status,omitempty"`
	SelfWatchdogReason                                string                                                                     `json:"self_watchdog_reason,omitempty"`
	SelfWatchdogObservedAt                            string                                                                     `json:"self_watchdog_observed_at,omitempty"`
	SelfWatchdogHighCPUSince                          string                                                                     `json:"self_watchdog_high_cpu_since,omitempty"`
	SelfWatchdogCPUPercent                            int                                                                        `json:"self_watchdog_cpu_percent,omitempty"`
	SelfWatchdogShutdownRequested                     bool                                                                       `json:"self_watchdog_shutdown_requested,omitempty"`
	SelfWatchdogOperationalMessage                    *ServerOperationalMessageV0                                                `json:"self_watchdog_operational_message,omitempty"`
	SelfWatchdogEvidenceRefs                          []string                                                                   `json:"self_watchdog_evidence_refs,omitempty"`
	LastError                                         string                                                                     `json:"last_error,omitempty"`
	LastErrorOperationalMessage                       *ServerOperationalMessageV0                                                `json:"last_error_operational_message,omitempty"`
	StartupEvidenceRefs                               []string                                                                   `json:"startup_evidence_refs,omitempty"`
	StatePersistStatus                                string                                                                     `json:"state_persist_status,omitempty"`
	StatePersistFailures                              int                                                                        `json:"state_persist_failures,omitempty"`
	StatePersistLastFailedAt                          string                                                                     `json:"state_persist_last_failed_at,omitempty"`
	StatePersistLastCode                              string                                                                     `json:"state_persist_last_code,omitempty"`
	StatePersistLastTransition                        string                                                                     `json:"state_persist_last_transition,omitempty"`
	StatePersistLastConfirmed                         string                                                                     `json:"state_persist_last_confirmed_at,omitempty"`
	AuditStatus                                       string                                                                     `json:"audit_status,omitempty"`
	AuditFailures                                     int                                                                        `json:"audit_failures,omitempty"`
	AuditLastFailedAt                                 string                                                                     `json:"audit_last_failed_at,omitempty"`
	AuditLastCode                                     string                                                                     `json:"audit_last_code,omitempty"`
	AuditLastEvent                                    string                                                                     `json:"audit_last_event,omitempty"`
	AuditLastSeverity                                 string                                                                     `json:"audit_last_severity,omitempty"`
	AuditLastConfirmed                                string                                                                     `json:"audit_last_confirmed_at,omitempty"`
	ResponseWriteFailures                             int                                                                        `json:"response_write_failures,omitempty"`
	ResponseWriteLastFailedAt                         string                                                                     `json:"response_write_last_failed_at,omitempty"`
	ResponseWriteLastCode                             string                                                                     `json:"response_write_last_code,omitempty"`
	ResponseWriteLastStage                            string                                                                     `json:"response_write_last_stage,omitempty"`
	RecentErrors                                      []ServerDiagnosticV0                                                       `json:"recent_errors,omitempty"`
}

type ServerEffectiveConfigV0 struct {
	SchemaVersion string                  `json:"schema_version,omitempty"`
	RestartNote   string                  `json:"restart_note,omitempty"`
	Settings      []ServerConfigSettingV0 `json:"settings,omitempty"`
	Diagnostics   []ServerDiagnosticV0    `json:"diagnostics,omitempty"`
}

type ServerConfigSettingV0 struct {
	Key             string `json:"key"`
	Value           string `json:"value,omitempty"`
	Source          string `json:"source,omitempty"`
	Scope           string `json:"scope,omitempty"`
	Label           string `json:"label,omitempty"`
	Description     string `json:"description,omitempty"`
	RestartBehavior string `json:"restart_behavior,omitempty"`
	Editable        bool   `json:"editable,omitempty"`
	Sensitive       bool   `json:"sensitive,omitempty"`
	Canonical       bool   `json:"canonical,omitempty"`
}

type ServerDiagnosticV0 struct {
	OccurredAt   string   `json:"occurred_at,omitempty"`
	Code         string   `json:"code"`
	Scope        string   `json:"scope,omitempty"`
	Message      string   `json:"message,omitempty"`
	EvidenceRefs []string `json:"evidence_refs,omitempty"`
}
