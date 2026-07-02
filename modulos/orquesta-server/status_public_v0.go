package orquestaserver

import (
	"path/filepath"
	"strings"
)

const (
	ServerStatusConfigVisibilityPublicRedactedV0 = "public_redacted"
	ServerStatusConfigHiddenValueV0              = "redacted"
	serverPublicResidentDirectorEnabledKeyV0     = "ORQUESTA_SERVER_RESIDENT_DIRECTOR_ENABLED"
)

type ServerPublicStatusV0 struct {
	SchemaVersion                         string                                      `json:"schema_version"`
	Status                                string                                      `json:"status"`
	AvailabilityStatus                    string                                      `json:"availability_status,omitempty"`
	AvailabilityReason                    string                                      `json:"availability_reason,omitempty"`
	AvailabilityNextActions               []string                                    `json:"availability_next_actions,omitempty"`
	AvailabilityEvidenceRefs              []string                                    `json:"availability_evidence_refs,omitempty"`
	Addr                                  string                                      `json:"addr,omitempty"`
	ProcessRef                            string                                      `json:"process_ref,omitempty"`
	DaemonEpochRef                        string                                      `json:"daemon_epoch_ref,omitempty"`
	StartedAt                             string                                      `json:"started_at,omitempty"`
	LastHeartbeatAt                       string                                      `json:"last_heartbeat_at,omitempty"`
	StartupStatus                         string                                      `json:"startup_status,omitempty"`
	StartupReady                          bool                                        `json:"startup_ready,omitempty"`
	StartupMessage                        string                                      `json:"startup_message,omitempty"`
	StartupOperationalMessage             *ServerOperationalMessageV0                 `json:"startup_operational_message,omitempty"`
	StartupRevision                       StartupRevisionSummaryV0                    `json:"startup_revision,omitempty"`
	RuntimeIdentity                       ServerPublicRuntimeIdentityV0               `json:"runtime_identity,omitempty"`
	DaemonLogPolicy                       DaemonLogPolicyV0                           `json:"daemon_log_policy,omitempty"`
	ShutdownSignalPolicy                  ShutdownSignalPolicyV0                      `json:"shutdown_signal_policy,omitempty"`
	EffectiveConfig                       ServerEffectiveConfigV0                     `json:"effective_config,omitempty"`
	ConfigVisibility                      string                                      `json:"config_visibility,omitempty"`
	HiddenConfigFields                    []string                                    `json:"hidden_config_fields,omitempty"`
	ProjectWorkDirRef                     string                                      `json:"project_work_dir_ref,omitempty"`
	RuntimeWorkDirRef                     string                                      `json:"runtime_work_dir_ref,omitempty"`
	LastStartupCheckAt                    string                                      `json:"last_startup_check_at,omitempty"`
	LastSupervisorAt                      string                                      `json:"last_supervisor_at,omitempty"`
	LastSupervisorStatus                  string                                      `json:"last_supervisor_status,omitempty"`
	LastSupervisorStopPublic              string                                      `json:"last_supervisor_stop_public,omitempty"`
	LastSupervisorStopCategory            string                                      `json:"last_supervisor_stop_category,omitempty"`
	LastSupervisorError                   string                                      `json:"last_supervisor_error,omitempty"`
	LastSupervisorOperationalMessage      *ServerOperationalMessageV0                 `json:"last_supervisor_operational_message,omitempty"`
	SupervisorLastErrorAt                 string                                      `json:"supervisor_last_error_at,omitempty"`
	SupervisorLastError                   string                                      `json:"supervisor_last_error,omitempty"`
	LastSupervisorQueueRef                string                                      `json:"last_supervisor_queue_ref,omitempty"`
	LastSupervisorQueueSize               int                                         `json:"last_supervisor_queue_size,omitempty"`
	LastSupervisorTickNumber              int                                         `json:"last_supervisor_tick_number,omitempty"`
	LastSupervisorResultTicks             int                                         `json:"last_supervisor_result_ticks,omitempty"`
	LastSupervisorExecutions              int                                         `json:"last_supervisor_executions,omitempty"`
	LastSupervisorSkips                   int                                         `json:"last_supervisor_skips,omitempty"`
	SupervisorTickActive                  bool                                        `json:"supervisor_tick_active,omitempty"`
	SupervisorFrozen                      bool                                        `json:"supervisor_frozen,omitempty"`
	SupervisorTicks                       int                                         `json:"supervisor_ticks"`
	SupervisorErrorTicks                  int                                         `json:"supervisor_error_ticks,omitempty"`
	SupervisorExecutions                  int                                         `json:"supervisor_executions,omitempty"`
	SupervisorSkips                       int                                         `json:"supervisor_skips,omitempty"`
	ShutdownInProgress                    bool                                        `json:"shutdown_in_progress,omitempty"`
	ShutdownStatus                        string                                      `json:"shutdown_status,omitempty"`
	ShutdownReady                         bool                                        `json:"shutdown_ready"`
	ShutdownHTTPStatus                    int                                         `json:"shutdown_http_status,omitempty"`
	ShutdownRunsRequested                 int                                         `json:"shutdown_runs_requested,omitempty"`
	ShutdownRunsStopped                   int                                         `json:"shutdown_runs_stopped,omitempty"`
	ShutdownAgentsInFlight                int                                         `json:"shutdown_agents_in_flight,omitempty"`
	ShutdownCheckpointsPending            int                                         `json:"shutdown_checkpoints_pending,omitempty"`
	ShutdownCheckpointAgentsPending       int                                         `json:"shutdown_checkpoint_agents_pending,omitempty"`
	ShutdownActiveWorkCount               int                                         `json:"shutdown_active_work_count,omitempty"`
	ShutdownActiveWorkRefs                []string                                    `json:"shutdown_active_work_refs,omitempty"`
	ShutdownAsyncWorkActive               int                                         `json:"shutdown_async_work_active,omitempty"`
	ShutdownStopTimeoutAt                 string                                      `json:"shutdown_stop_timeout_at,omitempty"`
	ShutdownSignalName                    string                                      `json:"shutdown_signal_name,omitempty"`
	ShutdownSignalCount                   int                                         `json:"shutdown_signal_count,omitempty"`
	ShutdownSignalEscalated               bool                                        `json:"shutdown_signal_escalated,omitempty"`
	LastShutdownAt                        string                                      `json:"last_shutdown_at,omitempty"`
	IdleSelfImprovementAfter              string                                      `json:"idle_self_improvement_after,omitempty"`
	IdleSelfImprovementTarget             int                                         `json:"idle_self_improvement_target_queue,omitempty"`
	IdleSelfImprovementCheck              string                                      `json:"idle_self_improvement_check,omitempty"`
	IdleSelfImprovementReason             string                                      `json:"idle_self_improvement_reason,omitempty"`
	IdleSelfImprovementOperationalMessage *ServerOperationalMessageV0                 `json:"idle_self_improvement_operational_message,omitempty"`
	IdleSelfImprovementGoal               *ServerPublicIdleSelfImprovementGoalStateV0 `json:"idle_self_improvement_goal,omitempty"`
	IdleSelfImprovementFlight             bool                                        `json:"idle_self_improvement_in_flight,omitempty"`
	IdleSelfImprovementRuns               int                                         `json:"idle_self_improvement_runs,omitempty"`
	IdleSelfImprovementOK                 int                                         `json:"idle_self_improvement_ok,omitempty"`
	ResidentDirectorStatus                string                                      `json:"resident_director_status,omitempty"`
	ResidentDirectorTickActive            bool                                        `json:"resident_director_tick_active,omitempty"`
	ResidentDirectorLastTickAt            string                                      `json:"resident_director_last_tick_at,omitempty"`
	ResidentDirectorLastSuccessAt         string                                      `json:"resident_director_last_success_at,omitempty"`
	ResidentDirectorLastErrorAt           string                                      `json:"resident_director_last_error_at,omitempty"`
	ResidentDirectorLastError             string                                      `json:"resident_director_last_error,omitempty"`
	ResidentDirectorLastRunRef            string                                      `json:"resident_director_last_run_ref,omitempty"`
	ResidentDirectorLastResult            string                                      `json:"resident_director_last_result,omitempty"`
	ResidentDirectorTicks                 int                                         `json:"resident_director_ticks,omitempty"`
	ResidentDirectorErrorTicks            int                                         `json:"resident_director_error_ticks,omitempty"`
	ResidentDirectorExecutedActions       int                                         `json:"resident_director_executed_actions,omitempty"`
	ResidentDirectorOperationalMessage    *ServerOperationalMessageV0                 `json:"resident_director_operational_message,omitempty"`
	GoalObserverStatus                    string                                      `json:"goal_observer_status,omitempty"`
	GoalObserverTickActive                bool                                        `json:"goal_observer_tick_active,omitempty"`
	GoalObserverLastTickAt                string                                      `json:"goal_observer_last_tick_at,omitempty"`
	GoalObserverLastSuccessAt             string                                      `json:"goal_observer_last_success_at,omitempty"`
	GoalObserverLastErrorAt               string                                      `json:"goal_observer_last_error_at,omitempty"`
	GoalObserverLastError                 string                                      `json:"goal_observer_last_error,omitempty"`
	GoalObserverTicks                     int                                         `json:"goal_observer_ticks,omitempty"`
	GoalObserverErrorTicks                int                                         `json:"goal_observer_error_ticks,omitempty"`
	GoalObserverObserved                  int                                         `json:"goal_observer_observed,omitempty"`
	GoalObserverTerminal                  int                                         `json:"goal_observer_terminal,omitempty"`
	GoalObserverIssues                    int                                         `json:"goal_observer_issues,omitempty"`
	GoalObserverOperationalMessage        *ServerOperationalMessageV0                 `json:"goal_observer_operational_message,omitempty"`
	ExternalBridgeComponent               string                                      `json:"external_bridge_component,omitempty"`
	ExternalBridgeStatus                  string                                      `json:"external_bridge_status,omitempty"`
	ExternalBridgeTickActive              bool                                        `json:"external_bridge_tick_active,omitempty"`
	ExternalBridgeLastTickRef             string                                      `json:"external_bridge_last_tick_ref,omitempty"`
	ExternalBridgeLastTickAt              string                                      `json:"external_bridge_last_tick_at,omitempty"`
	ExternalBridgeLastSuccess             string                                      `json:"external_bridge_last_success_at,omitempty"`
	ExternalBridgeLastErrorAt             string                                      `json:"external_bridge_last_error_at,omitempty"`
	ExternalBridgeLastError               string                                      `json:"external_bridge_last_error_code,omitempty"`
	ExternalBridgeOperationalMessage      *ServerOperationalMessageV0                 `json:"external_bridge_operational_message,omitempty"`
	ExternalBridgeStopReason              string                                      `json:"external_bridge_stop_reason,omitempty"`
	ExternalBridgeTicks                   int                                         `json:"external_bridge_ticks,omitempty"`
	ExternalBridgeErrorTicks              int                                         `json:"external_bridge_error_ticks,omitempty"`
	ExternalBridgeFilters                 []string                                    `json:"external_bridge_filters,omitempty"`
	ExternalBridgeCounters                map[string]int                              `json:"external_bridge_counters,omitempty"`
	ExternalBridgeEvidenceRefs            []string                                    `json:"external_bridge_evidence_refs,omitempty"`
	SelfWatchdogStatus                    string                                      `json:"self_watchdog_status,omitempty"`
	SelfWatchdogReason                    string                                      `json:"self_watchdog_reason,omitempty"`
	SelfWatchdogObservedAt                string                                      `json:"self_watchdog_observed_at,omitempty"`
	SelfWatchdogCPUPercent                int                                         `json:"self_watchdog_cpu_percent,omitempty"`
	SelfWatchdogShutdownRequested         bool                                        `json:"self_watchdog_shutdown_requested,omitempty"`
	SelfWatchdogOperationalMessage        *ServerOperationalMessageV0                 `json:"self_watchdog_operational_message,omitempty"`
	LastError                             string                                      `json:"last_error,omitempty"`
	LastErrorOperationalMessage           *ServerOperationalMessageV0                 `json:"last_error_operational_message,omitempty"`
	StartupEvidenceRefs                   []string                                    `json:"startup_evidence_refs,omitempty"`
	StatePersistStatus                    string                                      `json:"state_persist_status,omitempty"`
	StatePersistFailures                  int                                         `json:"state_persist_failures,omitempty"`
	StatePersistLastFailedAt              string                                      `json:"state_persist_last_failed_at,omitempty"`
	StatePersistLastCode                  string                                      `json:"state_persist_last_code,omitempty"`
	StatePersistLastTransition            string                                      `json:"state_persist_last_transition,omitempty"`
	StatePersistLastConfirmed             string                                      `json:"state_persist_last_confirmed_at,omitempty"`
	AuditStatus                           string                                      `json:"audit_status,omitempty"`
	AuditFailures                         int                                         `json:"audit_failures,omitempty"`
	AuditLastFailedAt                     string                                      `json:"audit_last_failed_at,omitempty"`
	AuditLastCode                         string                                      `json:"audit_last_code,omitempty"`
	AuditLastEvent                        string                                      `json:"audit_last_event,omitempty"`
	AuditLastSeverity                     string                                      `json:"audit_last_severity,omitempty"`
	AuditLastConfirmed                    string                                      `json:"audit_last_confirmed_at,omitempty"`
	ResponseWriteFailures                 int                                         `json:"response_write_failures,omitempty"`
	ResponseWriteLastFailedAt             string                                      `json:"response_write_last_failed_at,omitempty"`
	ResponseWriteLastCode                 string                                      `json:"response_write_last_code,omitempty"`
	ResponseWriteLastStage                string                                      `json:"response_write_last_stage,omitempty"`
	RecentErrors                          []ServerDiagnosticV0                        `json:"recent_errors,omitempty"`
}

func NewServerPublicStatusV0(state StateV0) ServerPublicStatusV0 {
	hidden := []string{}
	if strings.TrimSpace(state.ProjectWorkDir) != "" {
		hidden = append(hidden, "project_work_dir")
	}
	if strings.TrimSpace(state.RuntimeWorkDir) != "" {
		hidden = append(hidden, "runtime_work_dir")
	}
	config, configHidden := PublicServerEffectiveConfigV0(state.EffectiveConfig)
	hidden = append(hidden, configHidden...)
	residentDirectorStatus := state.ResidentDirectorStatus
	residentDirectorTickActive := state.ResidentDirectorTickActive
	residentDirectorOperationalMessage := copyServerOperationalMessageV0(state.ResidentDirectorOperationalMessage)
	if !serverPublicResidentDirectorEnabledV0(config) {
		residentDirectorStatus = "disabled"
		residentDirectorTickActive = false
		residentDirectorOperationalMessage = serverPublicResidentDirectorDisabledOperationalMessageV0(state)
	}
	availability := NewServerAvailabilityV0(state)
	return ServerPublicStatusV0{
		SchemaVersion:                         StateSchemaVersionV0,
		Status:                                strings.TrimSpace(state.Status),
		AvailabilityStatus:                    availability.Status,
		AvailabilityReason:                    availability.Reason,
		AvailabilityNextActions:               append([]string(nil), availability.NextActions...),
		AvailabilityEvidenceRefs:              append([]string(nil), availability.EvidenceRefs...),
		Addr:                                  strings.TrimSpace(state.Addr),
		ProcessRef:                            strings.TrimSpace(state.ProcessRef),
		DaemonEpochRef:                        strings.TrimSpace(state.DaemonEpochRef),
		StartedAt:                             state.StartedAt,
		LastHeartbeatAt:                       state.LastHeartbeatAt,
		StartupStatus:                         state.StartupStatus,
		StartupReady:                          state.StartupReady,
		StartupMessage:                        state.StartupMessage,
		StartupOperationalMessage:             copyServerOperationalMessageV0(state.StartupOperationalMessage),
		StartupRevision:                       normalizeStartupRevisionSummaryV0(state.StartupRevision),
		RuntimeIdentity:                       NewServerPublicRuntimeIdentityV0(state),
		DaemonLogPolicy:                       NormalizeDaemonLogPolicyV0(state.DaemonLogPolicy),
		ShutdownSignalPolicy:                  NormalizeShutdownSignalPolicyV0(state.ShutdownSignalPolicy),
		EffectiveConfig:                       config,
		ConfigVisibility:                      ServerStatusConfigVisibilityPublicRedactedV0,
		HiddenConfigFields:                    compactConfigStringsV0(hidden),
		ProjectWorkDirRef:                     publicStatusRefIfConfiguredV0(state.ProjectWorkDir, "server-config-ref-project-work-dir"),
		RuntimeWorkDirRef:                     publicStatusRefIfConfiguredV0(state.RuntimeWorkDir, "server-config-ref-runtime-work-dir"),
		LastStartupCheckAt:                    state.LastStartupCheckAt,
		LastSupervisorAt:                      state.LastSupervisorAt,
		LastSupervisorStatus:                  state.LastSupervisorStatus,
		LastSupervisorStopPublic:              state.LastSupervisorStopPublic,
		LastSupervisorStopCategory:            state.LastSupervisorStopCategory,
		LastSupervisorError:                   state.LastSupervisorError,
		LastSupervisorOperationalMessage:      copyServerOperationalMessageV0(state.LastSupervisorOperationalMessage),
		SupervisorLastErrorAt:                 state.SupervisorLastErrorAt,
		SupervisorLastError:                   state.SupervisorLastError,
		LastSupervisorQueueRef:                state.LastSupervisorQueueRef,
		LastSupervisorQueueSize:               state.LastSupervisorQueueSize,
		LastSupervisorTickNumber:              state.LastSupervisorTickNumber,
		LastSupervisorResultTicks:             state.LastSupervisorResultTicks,
		LastSupervisorExecutions:              state.LastSupervisorExecutions,
		LastSupervisorSkips:                   state.LastSupervisorSkips,
		SupervisorTickActive:                  state.SupervisorTickActive,
		SupervisorFrozen:                      state.SupervisorFrozen,
		SupervisorTicks:                       state.SupervisorTicks,
		SupervisorErrorTicks:                  state.SupervisorErrorTicks,
		SupervisorExecutions:                  state.SupervisorExecutions,
		SupervisorSkips:                       state.SupervisorSkips,
		ShutdownInProgress:                    state.ShutdownInProgress,
		ShutdownStatus:                        state.ShutdownStatus,
		ShutdownReady:                         state.ShutdownReady,
		ShutdownHTTPStatus:                    state.ShutdownHTTPStatus,
		ShutdownRunsRequested:                 state.ShutdownRunsRequested,
		ShutdownRunsStopped:                   state.ShutdownRunsStopped,
		ShutdownAgentsInFlight:                state.ShutdownAgentsInFlight,
		ShutdownCheckpointsPending:            state.ShutdownCheckpointsPending,
		ShutdownCheckpointAgentsPending:       state.ShutdownCheckpointAgentsPending,
		ShutdownActiveWorkCount:               state.ShutdownActiveWorkCount,
		ShutdownActiveWorkRefs:                compactServerStringsV0(state.ShutdownActiveWorkRefs),
		ShutdownAsyncWorkActive:               state.ShutdownAsyncWorkActive,
		ShutdownStopTimeoutAt:                 state.ShutdownStopTimeoutAt,
		ShutdownSignalName:                    state.ShutdownSignalName,
		ShutdownSignalCount:                   state.ShutdownSignalCount,
		ShutdownSignalEscalated:               state.ShutdownSignalEscalated,
		LastShutdownAt:                        state.LastShutdownAt,
		IdleSelfImprovementAfter:              state.IdleSelfImprovementAfter,
		IdleSelfImprovementTarget:             state.IdleSelfImprovementTarget,
		IdleSelfImprovementCheck:              state.IdleSelfImprovementCheck,
		IdleSelfImprovementReason:             state.IdleSelfImprovementReason,
		IdleSelfImprovementOperationalMessage: copyServerOperationalMessageV0(state.IdleSelfImprovementOperationalMessage),
		IdleSelfImprovementGoal:               NewServerPublicIdleSelfImprovementGoalStateV0(state),
		IdleSelfImprovementFlight:             state.IdleSelfImprovementFlight,
		IdleSelfImprovementRuns:               state.IdleSelfImprovementRuns,
		IdleSelfImprovementOK:                 state.IdleSelfImprovementOK,
		ResidentDirectorStatus:                residentDirectorStatus,
		ResidentDirectorTickActive:            residentDirectorTickActive,
		ResidentDirectorLastTickAt:            state.ResidentDirectorLastTickAt,
		ResidentDirectorLastSuccessAt:         state.ResidentDirectorLastSuccessAt,
		ResidentDirectorLastErrorAt:           state.ResidentDirectorLastErrorAt,
		ResidentDirectorLastError:             state.ResidentDirectorLastError,
		ResidentDirectorLastRunRef:            state.ResidentDirectorLastRunRef,
		ResidentDirectorLastResult:            state.ResidentDirectorLastResult,
		ResidentDirectorTicks:                 state.ResidentDirectorTicks,
		ResidentDirectorErrorTicks:            state.ResidentDirectorErrorTicks,
		ResidentDirectorExecutedActions:       state.ResidentDirectorExecutedActions,
		ResidentDirectorOperationalMessage:    residentDirectorOperationalMessage,
		GoalObserverStatus:                    strings.TrimSpace(state.GoalObserverStatus),
		GoalObserverTickActive:                state.GoalObserverTickActive,
		GoalObserverLastTickAt:                state.GoalObserverLastTickAt,
		GoalObserverLastSuccessAt:             state.GoalObserverLastSuccessAt,
		GoalObserverLastErrorAt:               state.GoalObserverLastErrorAt,
		GoalObserverLastError:                 state.GoalObserverLastError,
		GoalObserverTicks:                     state.GoalObserverTicks,
		GoalObserverErrorTicks:                state.GoalObserverErrorTicks,
		GoalObserverObserved:                  state.GoalObserverObserved,
		GoalObserverTerminal:                  state.GoalObserverTerminal,
		GoalObserverIssues:                    state.GoalObserverIssues,
		GoalObserverOperationalMessage:        copyServerOperationalMessageV0(state.GoalObserverOperationalMessage),
		ExternalBridgeComponent:               state.ExternalBridgeComponent,
		ExternalBridgeStatus:                  state.ExternalBridgeStatus,
		ExternalBridgeTickActive:              state.ExternalBridgeTickActive,
		ExternalBridgeLastTickRef:             state.ExternalBridgeLastTickRef,
		ExternalBridgeLastTickAt:              state.ExternalBridgeLastTickAt,
		ExternalBridgeLastSuccess:             state.ExternalBridgeLastSuccess,
		ExternalBridgeLastErrorAt:             state.ExternalBridgeLastErrorAt,
		ExternalBridgeLastError:               state.ExternalBridgeLastError,
		ExternalBridgeOperationalMessage:      copyServerOperationalMessageV0(state.ExternalBridgeOperationalMessage),
		ExternalBridgeStopReason:              state.ExternalBridgeStopReason,
		ExternalBridgeTicks:                   state.ExternalBridgeTicks,
		ExternalBridgeErrorTicks:              state.ExternalBridgeErrorTicks,
		ExternalBridgeFilters:                 compactConfigStringsV0(state.ExternalBridgeFilters),
		ExternalBridgeCounters:                copyServerIntMapV0(state.ExternalBridgeCounters),
		ExternalBridgeEvidenceRefs:            compactConfigStringsV0(state.ExternalBridgeEvidenceRefs),
		SelfWatchdogStatus:                    strings.TrimSpace(state.SelfWatchdogStatus),
		SelfWatchdogReason:                    strings.TrimSpace(state.SelfWatchdogReason),
		SelfWatchdogObservedAt:                strings.TrimSpace(state.SelfWatchdogObservedAt),
		SelfWatchdogCPUPercent:                state.SelfWatchdogCPUPercent,
		SelfWatchdogShutdownRequested:         state.SelfWatchdogShutdownRequested,
		SelfWatchdogOperationalMessage:        copyServerOperationalMessageV0(state.SelfWatchdogOperationalMessage),
		LastError:                             state.LastError,
		LastErrorOperationalMessage:           copyServerOperationalMessageV0(state.LastErrorOperationalMessage),
		StartupEvidenceRefs:                   append([]string(nil), state.StartupEvidenceRefs...),
		StatePersistStatus:                    state.StatePersistStatus,
		StatePersistFailures:                  state.StatePersistFailures,
		StatePersistLastFailedAt:              state.StatePersistLastFailedAt,
		StatePersistLastCode:                  state.StatePersistLastCode,
		StatePersistLastTransition:            state.StatePersistLastTransition,
		StatePersistLastConfirmed:             state.StatePersistLastConfirmed,
		AuditStatus:                           state.AuditStatus,
		AuditFailures:                         state.AuditFailures,
		AuditLastFailedAt:                     state.AuditLastFailedAt,
		AuditLastCode:                         state.AuditLastCode,
		AuditLastEvent:                        state.AuditLastEvent,
		AuditLastSeverity:                     state.AuditLastSeverity,
		AuditLastConfirmed:                    state.AuditLastConfirmed,
		ResponseWriteFailures:                 state.ResponseWriteFailures,
		ResponseWriteLastFailedAt:             state.ResponseWriteLastFailedAt,
		ResponseWriteLastCode:                 state.ResponseWriteLastCode,
		ResponseWriteLastStage:                state.ResponseWriteLastStage,
		RecentErrors:                          append([]ServerDiagnosticV0(nil), state.RecentErrors...),
	}
}

func serverPublicResidentDirectorDisabledOperationalMessageV0(
	state StateV0,
) *ServerOperationalMessageV0 {
	counters := map[string]int{
		"resident_enabled": 0,
	}
	if strings.TrimSpace(state.LastSupervisorStatus) == SupervisorPublicStatusWaitingOutboxV0 {
		counters["waiting_outbox"] = 1
	}
	if state.LastSupervisorQueueSize > 0 {
		counters["queue_size"] = state.LastSupervisorQueueSize
	}
	return projectServerOperationalMessageRecordV0(serverOperationalMessageInputV0{
		Scope:      "resident_director",
		ReasonCode: "resident_director_disabled",
		Status:     "disabled",
		Message:    "activar ORQUESTA_SERVER_RESIDENT_DIRECTOR_ENABLED=true y reiniciar; supervise legacy solo para intervencion puntual",
		EvidenceRefs: []string{
			"evidence-ref-resident-director-disabled-config",
		},
		Counters: counters,
	})
}

func serverPublicResidentDirectorEnabledV0(config ServerEffectiveConfigV0) bool {
	config = NormalizeServerEffectiveConfigV0(config)
	for _, setting := range config.Settings {
		if strings.TrimSpace(setting.Key) != serverPublicResidentDirectorEnabledKeyV0 {
			continue
		}
		switch strings.ToLower(strings.TrimSpace(setting.Value)) {
		case "0", "false", "no", "off":
			return false
		default:
			return true
		}
	}
	return true
}

func PublicServerEffectiveConfigV0(config ServerEffectiveConfigV0) (ServerEffectiveConfigV0, []string) {
	config = NormalizeServerEffectiveConfigV0(config)
	hidden := []string{}
	for index, setting := range config.Settings {
		if serverConfigSettingMustRedactV0(setting) {
			config.Settings[index].Value = ServerStatusConfigHiddenValueV0
			config.Settings[index].Sensitive = true
			hidden = append(hidden, "effective_config."+setting.Key)
		}
	}
	return config, compactConfigStringsV0(hidden)
}

func serverConfigSettingMustRedactV0(setting ServerConfigSettingV0) bool {
	if setting.Sensitive {
		return true
	}
	key := strings.ToUpper(strings.TrimSpace(setting.Key))
	for _, fragment := range []string{"TOKEN", "SECRET", "PASSWORD", "HOME", "WORKDIR", "WORK_DIR", "STATE_DIR", "RUNTIME_WORKDIR", "COMMAND", "PROVIDER", "MODEL"} {
		if strings.Contains(key, fragment) {
			return true
		}
	}
	value := strings.TrimSpace(setting.Value)
	return filepath.IsAbs(value) || strings.Contains(value, `:\`)
}

func publicStatusRefIfConfiguredV0(value string, ref string) string {
	if strings.TrimSpace(value) == "" {
		return ""
	}
	return ref
}
