package orquestaserver

const StateSchemaVersionV0 = "orquesta_server_state.v0"

type StateV0 struct {
	SchemaVersion             string   `json:"schema_version"`
	Status                    string   `json:"status"`
	PID                       int      `json:"pid"`
	Addr                      string   `json:"addr"`
	ProjectWorkDir            string   `json:"project_work_dir,omitempty"`
	RuntimeWorkDir            string   `json:"runtime_work_dir,omitempty"`
	StartedAt                 string   `json:"started_at,omitempty"`
	LastHeartbeatAt           string   `json:"last_heartbeat_at,omitempty"`
	StartupStatus             string   `json:"startup_status,omitempty"`
	StartupReady              bool     `json:"startup_ready,omitempty"`
	StartupMessage            string   `json:"startup_message,omitempty"`
	LastStartupCheckAt        string   `json:"last_startup_check_at,omitempty"`
	LastSupervisorAt          string   `json:"last_supervisor_at,omitempty"`
	LastSupervisorStatus      string   `json:"last_supervisor_status,omitempty"`
	LastSupervisorStop        string   `json:"last_supervisor_stop,omitempty"`
	LastSupervisorError       string   `json:"last_supervisor_error,omitempty"`
	SupervisorLastErrorAt     string   `json:"supervisor_last_error_at,omitempty"`
	SupervisorLastError       string   `json:"supervisor_last_error,omitempty"`
	LastSupervisorQueueRef    string   `json:"last_supervisor_queue_ref,omitempty"`
	LastSupervisorQueueSize   int      `json:"last_supervisor_queue_size,omitempty"`
	LastSupervisorTickNumber  int      `json:"last_supervisor_tick_number,omitempty"`
	LastSupervisorResultTicks int      `json:"last_supervisor_result_ticks,omitempty"`
	LastSupervisorExecutions  int      `json:"last_supervisor_executions,omitempty"`
	LastSupervisorSkips       int      `json:"last_supervisor_skips,omitempty"`
	SupervisorTicks           int      `json:"supervisor_ticks"`
	SupervisorErrorTicks      int      `json:"supervisor_error_ticks,omitempty"`
	SupervisorExecutions      int      `json:"supervisor_executions,omitempty"`
	SupervisorSkips           int      `json:"supervisor_skips,omitempty"`
	IdleSelfImprovementAfter  string   `json:"idle_self_improvement_after,omitempty"`
	IdleSelfImprovementTarget int      `json:"idle_self_improvement_target_queue,omitempty"`
	IdleSelfImprovementCheck  string   `json:"idle_self_improvement_check,omitempty"`
	IdleSelfImprovementReason string   `json:"idle_self_improvement_reason,omitempty"`
	IdleSelfImprovementFlight bool     `json:"idle_self_improvement_in_flight,omitempty"`
	IdleSelfImprovementRuns   int      `json:"idle_self_improvement_runs,omitempty"`
	IdleSelfImprovementOK     int      `json:"idle_self_improvement_ok,omitempty"`
	LastError                 string   `json:"last_error,omitempty"`
	StartupEvidenceRefs       []string `json:"startup_evidence_refs,omitempty"`
}
