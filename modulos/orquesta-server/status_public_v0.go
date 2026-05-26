package orquestaserver

import (
	"path/filepath"
	"strings"
)

const (
	ServerStatusConfigVisibilityPublicRedactedV0 = "public_redacted"
	ServerStatusConfigHiddenValueV0              = "redacted"
)

type ServerPublicStatusV0 struct {
	SchemaVersion              string                  `json:"schema_version"`
	Status                     string                  `json:"status"`
	Addr                       string                  `json:"addr,omitempty"`
	StartedAt                  string                  `json:"started_at,omitempty"`
	LastHeartbeatAt            string                  `json:"last_heartbeat_at,omitempty"`
	StartupStatus              string                  `json:"startup_status,omitempty"`
	StartupReady               bool                    `json:"startup_ready,omitempty"`
	StartupMessage             string                  `json:"startup_message,omitempty"`
	EffectiveConfig            ServerEffectiveConfigV0 `json:"effective_config,omitempty"`
	ConfigVisibility           string                  `json:"config_visibility,omitempty"`
	HiddenConfigFields         []string                `json:"hidden_config_fields,omitempty"`
	ProjectWorkDirRef          string                  `json:"project_work_dir_ref,omitempty"`
	RuntimeWorkDirRef          string                  `json:"runtime_work_dir_ref,omitempty"`
	LastStartupCheckAt         string                  `json:"last_startup_check_at,omitempty"`
	LastSupervisorAt           string                  `json:"last_supervisor_at,omitempty"`
	LastSupervisorStatus       string                  `json:"last_supervisor_status,omitempty"`
	LastSupervisorStopPublic   string                  `json:"last_supervisor_stop_public,omitempty"`
	LastSupervisorStopCategory string                  `json:"last_supervisor_stop_category,omitempty"`
	LastSupervisorError        string                  `json:"last_supervisor_error,omitempty"`
	SupervisorLastErrorAt      string                  `json:"supervisor_last_error_at,omitempty"`
	SupervisorLastError        string                  `json:"supervisor_last_error,omitempty"`
	LastSupervisorQueueRef     string                  `json:"last_supervisor_queue_ref,omitempty"`
	LastSupervisorQueueSize    int                     `json:"last_supervisor_queue_size,omitempty"`
	LastSupervisorTickNumber   int                     `json:"last_supervisor_tick_number,omitempty"`
	LastSupervisorResultTicks  int                     `json:"last_supervisor_result_ticks,omitempty"`
	LastSupervisorExecutions   int                     `json:"last_supervisor_executions,omitempty"`
	LastSupervisorSkips        int                     `json:"last_supervisor_skips,omitempty"`
	SupervisorTickActive       bool                    `json:"supervisor_tick_active,omitempty"`
	SupervisorFrozen           bool                    `json:"supervisor_frozen,omitempty"`
	SupervisorTicks            int                     `json:"supervisor_ticks"`
	SupervisorErrorTicks       int                     `json:"supervisor_error_ticks,omitempty"`
	SupervisorExecutions       int                     `json:"supervisor_executions,omitempty"`
	SupervisorSkips            int                     `json:"supervisor_skips,omitempty"`
	ShutdownInProgress         bool                    `json:"shutdown_in_progress,omitempty"`
	ShutdownStatus             string                  `json:"shutdown_status,omitempty"`
	ShutdownReady              bool                    `json:"shutdown_ready"`
	ShutdownHTTPStatus         int                     `json:"shutdown_http_status,omitempty"`
	ShutdownRunsRequested      int                     `json:"shutdown_runs_requested,omitempty"`
	ShutdownRunsStopped        int                     `json:"shutdown_runs_stopped,omitempty"`
	ShutdownAgentsInFlight     int                     `json:"shutdown_agents_in_flight,omitempty"`
	ShutdownCheckpointsPending int                     `json:"shutdown_checkpoints_pending,omitempty"`
	LastShutdownAt             string                  `json:"last_shutdown_at,omitempty"`
	IdleSelfImprovementAfter   string                  `json:"idle_self_improvement_after,omitempty"`
	IdleSelfImprovementTarget  int                     `json:"idle_self_improvement_target_queue,omitempty"`
	IdleSelfImprovementCheck   string                  `json:"idle_self_improvement_check,omitempty"`
	IdleSelfImprovementReason  string                  `json:"idle_self_improvement_reason,omitempty"`
	IdleSelfImprovementFlight  bool                    `json:"idle_self_improvement_in_flight,omitempty"`
	IdleSelfImprovementRuns    int                     `json:"idle_self_improvement_runs,omitempty"`
	IdleSelfImprovementOK      int                     `json:"idle_self_improvement_ok,omitempty"`
	LastError                  string                  `json:"last_error,omitempty"`
	StartupEvidenceRefs        []string                `json:"startup_evidence_refs,omitempty"`
	StatePersistStatus         string                  `json:"state_persist_status,omitempty"`
	StatePersistFailures       int                     `json:"state_persist_failures,omitempty"`
	StatePersistLastFailedAt   string                  `json:"state_persist_last_failed_at,omitempty"`
	StatePersistLastCode       string                  `json:"state_persist_last_code,omitempty"`
	StatePersistLastTransition string                  `json:"state_persist_last_transition,omitempty"`
	StatePersistLastConfirmed  string                  `json:"state_persist_last_confirmed_at,omitempty"`
	RecentErrors               []ServerDiagnosticV0    `json:"recent_errors,omitempty"`
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
	return ServerPublicStatusV0{
		SchemaVersion:              StateSchemaVersionV0,
		Status:                     strings.TrimSpace(state.Status),
		Addr:                       strings.TrimSpace(state.Addr),
		StartedAt:                  state.StartedAt,
		LastHeartbeatAt:            state.LastHeartbeatAt,
		StartupStatus:              state.StartupStatus,
		StartupReady:               state.StartupReady,
		StartupMessage:             state.StartupMessage,
		EffectiveConfig:            config,
		ConfigVisibility:           ServerStatusConfigVisibilityPublicRedactedV0,
		HiddenConfigFields:         compactConfigStringsV0(hidden),
		ProjectWorkDirRef:          publicStatusRefIfConfiguredV0(state.ProjectWorkDir, "server-config-ref-project-work-dir"),
		RuntimeWorkDirRef:          publicStatusRefIfConfiguredV0(state.RuntimeWorkDir, "server-config-ref-runtime-work-dir"),
		LastStartupCheckAt:         state.LastStartupCheckAt,
		LastSupervisorAt:           state.LastSupervisorAt,
		LastSupervisorStatus:       state.LastSupervisorStatus,
		LastSupervisorStopPublic:   state.LastSupervisorStopPublic,
		LastSupervisorStopCategory: state.LastSupervisorStopCategory,
		LastSupervisorError:        state.LastSupervisorError,
		SupervisorLastErrorAt:      state.SupervisorLastErrorAt,
		SupervisorLastError:        state.SupervisorLastError,
		LastSupervisorQueueRef:     state.LastSupervisorQueueRef,
		LastSupervisorQueueSize:    state.LastSupervisorQueueSize,
		LastSupervisorTickNumber:   state.LastSupervisorTickNumber,
		LastSupervisorResultTicks:  state.LastSupervisorResultTicks,
		LastSupervisorExecutions:   state.LastSupervisorExecutions,
		LastSupervisorSkips:        state.LastSupervisorSkips,
		SupervisorTickActive:       state.SupervisorTickActive,
		SupervisorFrozen:           state.SupervisorFrozen,
		SupervisorTicks:            state.SupervisorTicks,
		SupervisorErrorTicks:       state.SupervisorErrorTicks,
		SupervisorExecutions:       state.SupervisorExecutions,
		SupervisorSkips:            state.SupervisorSkips,
		ShutdownInProgress:         state.ShutdownInProgress,
		ShutdownStatus:             state.ShutdownStatus,
		ShutdownReady:              state.ShutdownReady,
		ShutdownHTTPStatus:         state.ShutdownHTTPStatus,
		ShutdownRunsRequested:      state.ShutdownRunsRequested,
		ShutdownRunsStopped:        state.ShutdownRunsStopped,
		ShutdownAgentsInFlight:     state.ShutdownAgentsInFlight,
		ShutdownCheckpointsPending: state.ShutdownCheckpointsPending,
		LastShutdownAt:             state.LastShutdownAt,
		IdleSelfImprovementAfter:   state.IdleSelfImprovementAfter,
		IdleSelfImprovementTarget:  state.IdleSelfImprovementTarget,
		IdleSelfImprovementCheck:   state.IdleSelfImprovementCheck,
		IdleSelfImprovementReason:  state.IdleSelfImprovementReason,
		IdleSelfImprovementFlight:  state.IdleSelfImprovementFlight,
		IdleSelfImprovementRuns:    state.IdleSelfImprovementRuns,
		IdleSelfImprovementOK:      state.IdleSelfImprovementOK,
		LastError:                  state.LastError,
		StartupEvidenceRefs:        append([]string(nil), state.StartupEvidenceRefs...),
		StatePersistStatus:         state.StatePersistStatus,
		StatePersistFailures:       state.StatePersistFailures,
		StatePersistLastFailedAt:   state.StatePersistLastFailedAt,
		StatePersistLastCode:       state.StatePersistLastCode,
		StatePersistLastTransition: state.StatePersistLastTransition,
		StatePersistLastConfirmed:  state.StatePersistLastConfirmed,
		RecentErrors:               append([]ServerDiagnosticV0(nil), state.RecentErrors...),
	}
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
