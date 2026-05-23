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
	LastSupervisorQueueRef    string   `json:"last_supervisor_queue_ref,omitempty"`
	LastSupervisorResultTicks int      `json:"last_supervisor_result_ticks,omitempty"`
	LastSupervisorExecutions  int      `json:"last_supervisor_executions,omitempty"`
	LastSupervisorSkips       int      `json:"last_supervisor_skips,omitempty"`
	SupervisorTicks           int      `json:"supervisor_ticks"`
	SupervisorErrorTicks      int      `json:"supervisor_error_ticks,omitempty"`
	LastError                 string   `json:"last_error,omitempty"`
	StartupEvidenceRefs       []string `json:"startup_evidence_refs,omitempty"`
}
