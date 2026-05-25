package orquestaserver

const StateSchemaVersionV0 = "orquesta_server_state.v0"

type StateV0 struct {
	SchemaVersion              string                  `json:"schema_version"`
	Status                     string                  `json:"status"`
	PID                        int                     `json:"pid"`
	Addr                       string                  `json:"addr"`
	ProjectWorkDir             string                  `json:"project_work_dir,omitempty"`
	RuntimeWorkDir             string                  `json:"runtime_work_dir,omitempty"`
	StartedAt                  string                  `json:"started_at,omitempty"`
	LastHeartbeatAt            string                  `json:"last_heartbeat_at,omitempty"`
	StartupStatus              string                  `json:"startup_status,omitempty"`
	StartupReady               bool                    `json:"startup_ready,omitempty"`
	StartupMessage             string                  `json:"startup_message,omitempty"`
	EffectiveConfig            ServerEffectiveConfigV0 `json:"effective_config,omitempty"`
	LastStartupCheckAt         string                  `json:"last_startup_check_at,omitempty"`
	LastSupervisorAt           string                  `json:"last_supervisor_at,omitempty"`
	LastSupervisorStatus       string                  `json:"last_supervisor_status,omitempty"`
	LastSupervisorStop         string                  `json:"last_supervisor_stop,omitempty"`
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
	RecentErrors               []ServerDiagnosticV0    `json:"recent_errors,omitempty"`
}

type ServerEffectiveConfigV0 struct {
	SchemaVersion string                  `json:"schema_version,omitempty"`
	RestartNote   string                  `json:"restart_note,omitempty"`
	Settings      []ServerConfigSettingV0 `json:"settings,omitempty"`
}

type ServerConfigSettingV0 struct {
	Key             string `json:"key"`
	Value           string `json:"value,omitempty"`
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
