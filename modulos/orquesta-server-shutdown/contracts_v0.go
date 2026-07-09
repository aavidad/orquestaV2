package orquestaservershutdown

import (
	"context"
	"time"

	orquestaruncontrol "orquesta/modulos/orquesta-run-control"
	orquestarunqueue "orquesta/modulos/orquesta-run-queue"
	orquestarunsupervisor "orquesta/modulos/orquesta-run-supervisor"
)

const ServerShutdownSchemaVersionV0 = "server_shutdown.v0"

const (
	ServerShutdownStatusReadyV0               = "ready"
	ServerShutdownStatusWaitingDrainV0        = "waiting_drain"
	ServerShutdownStatusWaitingCheckpointV0   = "waiting_checkpoint"
	ServerShutdownStatusNoQueueReaderV0       = "queue_reader_required"
	ServerShutdownStatusNoRunControlReaderV0  = "run_control_reader_required"
	ServerShutdownStatusNoRunControlWriterV0  = "run_control_writer_required"
	ServerShutdownStatusRequesterDeniedV0     = "requester_not_authorized"
	ServerShutdownStatusActiveGoalsPresentV0  = "active_goals_present"
	ServerShutdownStatusBackendStillRunningV0 = "backend_still_running"
)

const (
	ServerShutdownGoalActionObserveActiveV0       = "observe_active_goal"
	ServerShutdownGoalActionWaitCheckpointV0      = "wait_checkpoint"
	ServerShutdownGoalActionStopRequestedWaitV0   = "stop_requested_wait"
	ServerShutdownGoalActionForcedStopRequestedV0 = "forced_stop_requested"
	ServerShutdownGoalActionCleanupRequiredV0     = "cleanup_required"
	ServerShutdownGoalActionCleanupRequestedV0    = "cleanup_requested"
	ServerShutdownGoalActionCleanupAttemptedV0    = "cleanup_attempted_wait"
	ServerShutdownGoalActionCleanupCompletedV0    = "cleanup_completed"
)

type ServerShutdownCommandV0 struct {
	RequestID            string    `json:"request_id,omitempty"`
	CorrelationID        string    `json:"correlation_id,omitempty"`
	QueueRef             string    `json:"queue_ref,omitempty"`
	AppRefs              []string  `json:"app_refs,omitempty"`
	QueueLimit           int       `json:"queue_limit,omitempty"`
	MaxTicks             int       `json:"max_ticks,omitempty"`
	MaxRunsPerTick       int       `json:"max_runs_per_tick,omitempty"`
	MaxExecutions        int       `json:"max_executions,omitempty"`
	Forced               bool      `json:"forced,omitempty"`
	CleanupGoalBackends  bool      `json:"cleanup_goal_backends,omitempty"`
	RequestedBy          string    `json:"requested_by,omitempty"`
	Reason               string    `json:"reason,omitempty"`
	IdempotencyKey       string    `json:"idempotency_key,omitempty"`
	EvidenceRefs         []string  `json:"evidence_refs,omitempty"`
	OccurredAt           time.Time `json:"occurred_at"`
	CheckpointDeadlineAt time.Time `json:"checkpoint_deadline_at,omitempty"`
	StopOnNoExecution    bool      `json:"stop_on_no_execution,omitempty"`
}

type ServerShutdownDepsV0 struct {
	QueueReader         orquestarunqueue.RunQueueReaderPortV0
	RunControlReader    orquestaruncontrol.RunControlReaderPortV0
	RunControlWriter    orquestaruncontrol.RunControlWriterPortV0
	RunCheckpointWriter orquestaruncontrol.RunControlCheckpointWriterPortV0
	CheckpointPreparer  PrepareAgentShutdownPortV0
	Supervisor          RunSupervisorPortV0
	StatsReader         RunStatsReaderPortV0
	ActiveWorkReader    ActiveShutdownWorkReaderPortV0
	ActiveWorkCleaner   ActiveShutdownWorkCleanerPortV0
}

type RunSupervisorPortV0 interface {
	RunGlobalSupervisorV0(
		context.Context,
		orquestarunsupervisor.RunSupervisorCommandV0,
	) (orquestarunsupervisor.RunSupervisorResultV0, error)
}

type RunStatsReaderPortV0 interface {
	ReadRunShutdownStatsV0(
		context.Context,
		RunShutdownStatsRequestV0,
	) (RunShutdownStatsV0, error)
}

type PrepareAgentShutdownPortV0 interface {
	PrepareAgentShutdownV0(
		context.Context,
		PrepareAgentShutdownCommandV0,
	) (PrepareAgentShutdownResultV0, error)
}

type ActiveShutdownWorkReaderPortV0 interface {
	ReadActiveShutdownWorkV0(
		context.Context,
		ActiveShutdownWorkRequestV0,
	) (ActiveShutdownWorkResultV0, error)
}

type ActiveShutdownWorkCleanerPortV0 interface {
	CleanupActiveShutdownWorkV0(
		context.Context,
		ActiveShutdownWorkCleanupCommandV0,
	) (ActiveShutdownWorkCleanupResultV0, error)
}

type ActiveShutdownWorkIdentityPortV0 interface {
	ActiveShutdownWorkIdentityV0() string
}

type ActiveShutdownWorkRequestV0 struct {
	QueueRef      string   `json:"queue_ref,omitempty"`
	AppRefs       []string `json:"app_refs,omitempty"`
	CorrelationID string   `json:"correlation_id,omitempty"`
	EvidenceRefs  []string `json:"evidence_refs,omitempty"`
	MaxItems      int      `json:"max_items,omitempty"`
}

type ActiveShutdownWorkResultV0 struct {
	ActiveWorks  []ActiveShutdownWorkV0 `json:"active_works,omitempty"`
	EvidenceRefs []string               `json:"evidence_refs,omitempty"`
}

type ActiveShutdownWorkCleanupCommandV0 struct {
	QueueRef            string                 `json:"queue_ref,omitempty"`
	AppRefs             []string               `json:"app_refs,omitempty"`
	CorrelationID       string                 `json:"correlation_id,omitempty"`
	EvidenceRefs        []string               `json:"evidence_refs,omitempty"`
	ActiveWorks         []ActiveShutdownWorkV0 `json:"active_works,omitempty"`
	CleanupGoalBackends bool                   `json:"cleanup_goal_backends,omitempty"`
}

type ActiveShutdownWorkCleanupResultV0 struct {
	CleanedWorkCount int      `json:"cleaned_work_count,omitempty"`
	EvidenceRefs     []string `json:"evidence_refs,omitempty"`
}

type ActiveShutdownWorkV0 struct {
	Kind               string   `json:"kind,omitempty"`
	RunRef             string   `json:"run_ref,omitempty"`
	WorkRef            string   `json:"work_ref,omitempty"`
	ExternalWorkRef    string   `json:"external_work_ref,omitempty"`
	Status             string   `json:"status,omitempty"`
	ActionTaken        string   `json:"action_taken,omitempty"`
	ActionEvidenceRefs []string `json:"action_evidence_refs,omitempty"`
	EvidenceRefs       []string `json:"evidence_refs,omitempty"`
}

type ServerShutdownGoalActionV0 struct {
	Kind               string   `json:"kind,omitempty"`
	RunRef             string   `json:"run_ref,omitempty"`
	WorkRef            string   `json:"work_ref,omitempty"`
	ExternalWorkRef    string   `json:"external_work_ref,omitempty"`
	Status             string   `json:"status,omitempty"`
	ActionTaken        string   `json:"action_taken"`
	ActionEvidenceRefs []string `json:"action_evidence_refs,omitempty"`
	EvidenceRefs       []string `json:"evidence_refs,omitempty"`
}

type PrepareAgentShutdownCommandV0 struct {
	RunRef        string    `json:"run_ref"`
	AppRef        string    `json:"app_ref,omitempty"`
	RequestedBy   string    `json:"requested_by,omitempty"`
	Reason        string    `json:"reason,omitempty"`
	CorrelationID string    `json:"correlation_id,omitempty"`
	EvidenceRefs  []string  `json:"evidence_refs,omitempty"`
	OccurredAt    time.Time `json:"occurred_at"`
}

type PrepareAgentShutdownResultV0 struct {
	RunRef             string   `json:"run_ref"`
	CheckpointRecorded bool     `json:"checkpoint_recorded"`
	CheckpointRef      string   `json:"checkpoint_ref,omitempty"`
	PendingAgentRefs   []string `json:"pending_agent_refs,omitempty"`
	EvidenceRefs       []string `json:"evidence_refs,omitempty"`
}

type RunShutdownStatsRequestV0 struct {
	RunRef        string   `json:"run_ref"`
	CorrelationID string   `json:"correlation_id,omitempty"`
	EvidenceRefs  []string `json:"evidence_refs,omitempty"`
}

type RunShutdownStatsV0 struct {
	RunRef                  string   `json:"run_ref"`
	AgentsInFlight          int      `json:"agents_in_flight"`
	ProcessLivenessObserved bool     `json:"process_liveness_observed,omitempty"`
	AgentsRunningLive       int      `json:"agents_running_live,omitempty"`
	AgentsRunningStale      int      `json:"agents_running_stale,omitempty"`
	AgentsLost              int      `json:"agents_lost,omitempty"`
	AgentsStopRequested     int      `json:"agents_stop_requested"`
	AgentsStopConfirmed     int      `json:"agents_stop_confirmed"`
	EvidenceRefs            []string `json:"evidence_refs,omitempty"`
}

type ServerShutdownResultV0 struct {
	SchemaVersion              string                                       `json:"schema_version"`
	Status                     string                                       `json:"status"`
	RecommendedAction          string                                       `json:"recommended_action,omitempty"`
	ShutdownReady              bool                                         `json:"shutdown_ready"`
	ExitPending                bool                                         `json:"exit_pending,omitempty"`
	PID                        int                                          `json:"pid,omitempty"`
	RunsRequested              int                                          `json:"runs_requested"`
	RunsStopped                int                                          `json:"runs_stopped"`
	AgentsInFlight             int                                          `json:"agents_in_flight"`
	ProcessLivenessObserved    bool                                         `json:"process_liveness_observed,omitempty"`
	AgentsRunningLive          int                                          `json:"agents_running_live,omitempty"`
	AgentsRunningStale         int                                          `json:"agents_running_stale,omitempty"`
	AgentsLost                 int                                          `json:"agents_lost,omitempty"`
	CheckpointsPending         int                                          `json:"checkpoints_pending"`
	CheckpointAgentsPending    int                                          `json:"checkpoint_agents_pending,omitempty"`
	CheckpointDeadlinesExpired int                                          `json:"checkpoint_deadlines_expired,omitempty"`
	ActiveWorkCount            int                                          `json:"active_work_count,omitempty"`
	ActiveWorks                []ActiveShutdownWorkV0                       `json:"active_works,omitempty"`
	GoalActions                []ServerShutdownGoalActionV0                 `json:"goal_actions,omitempty"`
	Runs                       []ServerShutdownRunResultV0                  `json:"runs,omitempty"`
	Supervisor                 *orquestarunsupervisor.RunSupervisorResultV0 `json:"supervisor,omitempty"`
	EvidenceRefs               []string                                     `json:"evidence_refs,omitempty"`
}

type ServerShutdownRunResultV0 struct {
	RunRef                        string   `json:"run_ref"`
	AppRef                        string   `json:"app_ref,omitempty"`
	ControlStatus                 string   `json:"control_status,omitempty"`
	CheckpointRequired            bool     `json:"checkpoint_required,omitempty"`
	CheckpointRef                 string   `json:"checkpoint_ref,omitempty"`
	CheckpointDeadlineExpired     bool     `json:"checkpoint_deadline_expired,omitempty"`
	ForcedAfterCheckpointDeadline bool     `json:"forced_after_checkpoint_deadline,omitempty"`
	PendingCheckpointAgentRefs    []string `json:"pending_checkpoint_agent_refs,omitempty"`
	CheckpointEvidenceRefs        []string `json:"checkpoint_evidence_refs,omitempty"`
	Terminal                      bool     `json:"terminal,omitempty"`
	StopRequested                 bool     `json:"stop_requested,omitempty"`
	AgentsInFlight                int      `json:"agents_in_flight"`
	ProcessLivenessObserved       bool     `json:"process_liveness_observed,omitempty"`
	AgentsRunningLive             int      `json:"agents_running_live,omitempty"`
	AgentsRunningStale            int      `json:"agents_running_stale,omitempty"`
	AgentsLost                    int      `json:"agents_lost,omitempty"`
	AgentsStopRequested           int      `json:"agents_stop_requested"`
	AgentsStopConfirmed           int      `json:"agents_stop_confirmed"`
	Ready                         bool     `json:"ready"`
}
