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
	ServerShutdownStatusReadyV0              = "ready"
	ServerShutdownStatusWaitingDrainV0       = "waiting_drain"
	ServerShutdownStatusWaitingCheckpointV0  = "waiting_checkpoint"
	ServerShutdownStatusNoQueueReaderV0      = "queue_reader_required"
	ServerShutdownStatusNoRunControlReaderV0 = "run_control_reader_required"
	ServerShutdownStatusNoRunControlWriterV0 = "run_control_writer_required"
)

type ServerShutdownCommandV0 struct {
	RequestID         string    `json:"request_id,omitempty"`
	CorrelationID     string    `json:"correlation_id,omitempty"`
	QueueRef          string    `json:"queue_ref,omitempty"`
	AppRefs           []string  `json:"app_refs,omitempty"`
	QueueLimit        int       `json:"queue_limit,omitempty"`
	MaxTicks          int       `json:"max_ticks,omitempty"`
	MaxRunsPerTick    int       `json:"max_runs_per_tick,omitempty"`
	MaxExecutions     int       `json:"max_executions,omitempty"`
	Forced            bool      `json:"forced,omitempty"`
	RequestedBy       string    `json:"requested_by,omitempty"`
	Reason            string    `json:"reason,omitempty"`
	IdempotencyKey    string    `json:"idempotency_key,omitempty"`
	EvidenceRefs      []string  `json:"evidence_refs,omitempty"`
	OccurredAt        time.Time `json:"occurred_at"`
	StopOnNoExecution bool      `json:"stop_on_no_execution,omitempty"`
}

type ServerShutdownDepsV0 struct {
	QueueReader      orquestarunqueue.RunQueueReaderPortV0
	RunControlReader orquestaruncontrol.RunControlReaderPortV0
	RunControlWriter orquestaruncontrol.RunControlWriterPortV0
	Supervisor       RunSupervisorPortV0
	StatsReader      RunStatsReaderPortV0
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

type RunShutdownStatsRequestV0 struct {
	RunRef        string   `json:"run_ref"`
	CorrelationID string   `json:"correlation_id,omitempty"`
	EvidenceRefs  []string `json:"evidence_refs,omitempty"`
}

type RunShutdownStatsV0 struct {
	RunRef              string   `json:"run_ref"`
	AgentsInFlight      int      `json:"agents_in_flight"`
	AgentsStopRequested int      `json:"agents_stop_requested"`
	AgentsStopConfirmed int      `json:"agents_stop_confirmed"`
	EvidenceRefs        []string `json:"evidence_refs,omitempty"`
}

type ServerShutdownResultV0 struct {
	SchemaVersion      string                                       `json:"schema_version"`
	Status             string                                       `json:"status"`
	ShutdownReady      bool                                         `json:"shutdown_ready"`
	RunsRequested      int                                          `json:"runs_requested"`
	RunsStopped        int                                          `json:"runs_stopped"`
	AgentsInFlight     int                                          `json:"agents_in_flight"`
	CheckpointsPending int                                          `json:"checkpoints_pending"`
	Runs               []ServerShutdownRunResultV0                  `json:"runs,omitempty"`
	Supervisor         *orquestarunsupervisor.RunSupervisorResultV0 `json:"supervisor,omitempty"`
	EvidenceRefs       []string                                     `json:"evidence_refs,omitempty"`
}

type ServerShutdownRunResultV0 struct {
	RunRef              string `json:"run_ref"`
	AppRef              string `json:"app_ref,omitempty"`
	ControlStatus       string `json:"control_status,omitempty"`
	CheckpointRequired  bool   `json:"checkpoint_required,omitempty"`
	Terminal            bool   `json:"terminal,omitempty"`
	StopRequested       bool   `json:"stop_requested,omitempty"`
	AgentsInFlight      int    `json:"agents_in_flight"`
	AgentsStopRequested int    `json:"agents_stop_requested"`
	AgentsStopConfirmed int    `json:"agents_stop_confirmed"`
	Ready               bool   `json:"ready"`
}
