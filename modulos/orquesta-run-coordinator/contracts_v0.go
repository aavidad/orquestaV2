package orquestaruncoordinator

import (
	"context"
	"time"

	orquestaruncontrol "orquesta/modulos/orquesta-run-control"
	orquestarunqueue "orquesta/modulos/orquesta-run-queue"
)

const RunCoordinatorSchemaVersionV0 = "run_coordinator.v0"

type RunDrainerPortV0 interface {
	DrainRunV0(context.Context, RunDrainRequestV0) (RunDrainResultV0, error)
}

type RunCoordinatorDepsV0 struct {
	QueueReader   orquestarunqueue.RunQueueReaderPortV0
	QueueUpdater  orquestarunqueue.RunQueuePriorityWriterPortV0
	ControlReader orquestaruncontrol.RunControlReaderPortV0
	Drainer       RunDrainerPortV0
}

type RunCoordinatorTickCommandV0 struct {
	QueueRef             string                                   `json:"queue_ref,omitempty"`
	AppRefs              []string                                 `json:"app_refs,omitempty"`
	ExcludeRunRefs       []string                                 `json:"exclude_run_refs,omitempty"`
	QueueLimit           int                                      `json:"queue_limit,omitempty"`
	MaxRuns              int                                      `json:"max_runs,omitempty"`
	OccurredAt           time.Time                                `json:"occurred_at"`
	CorrelationID        string                                   `json:"correlation_id,omitempty"`
	DrainLimits          RunDrainLimitsV0                         `json:"drain_limits,omitempty"`
	RankingPolicy        orquestarunqueue.RunQueueRankingPolicyV0 `json:"ranking_policy,omitempty"`
	ContinueOnDrainError bool                                     `json:"continue_on_drain_error,omitempty"`
}

type RunDrainLimitsV0 struct {
	MaxBursts            int `json:"max_bursts,omitempty"`
	MaxStepsPerBurst     int `json:"max_steps_per_burst,omitempty"`
	MaxDispatchesPerWait int `json:"max_dispatches_per_wait,omitempty"`
	MaxCommands          int `json:"max_commands,omitempty"`
	MaxOutboxPerCycle    int `json:"max_outbox_per_cycle,omitempty"`
	MaxDecisionCycles    int `json:"max_decision_cycles,omitempty"`
	MaxExternalWaits     int `json:"max_external_waits,omitempty"`
}

type RunCoordinatorTickResultV0 struct {
	Executions []RunExecutionSummaryV0 `json:"executions,omitempty"`
	Skips      []RunSkipSummaryV0      `json:"skips,omitempty"`
	Ranked     []RankedRunSummaryV0    `json:"ranked,omitempty"`
}

type RunDrainRequestV0 struct {
	RunRef        string           `json:"run_ref"`
	AppRef        string           `json:"app_ref,omitempty"`
	Rank          int              `json:"rank"`
	OccurredAt    time.Time        `json:"occurred_at"`
	CorrelationID string           `json:"correlation_id,omitempty"`
	Limits        RunDrainLimitsV0 `json:"limits,omitempty"`
}

type RunDrainResultV0 struct {
	RunRef       string                 `json:"run_ref"`
	AppRef       string                 `json:"app_ref,omitempty"`
	Outcome      string                 `json:"outcome,omitempty"`
	QueueStatus  string                 `json:"queue_status,omitempty"`
	EvidenceRefs []string               `json:"evidence_refs,omitempty"`
	Diagnostics  []RunDrainDiagnosticV0 `json:"diagnostics,omitempty"`
}

type RunDrainDiagnosticV0 struct {
	Kind               string   `json:"kind,omitempty"`
	Status             string   `json:"status,omitempty"`
	RunRef             string   `json:"run_ref,omitempty"`
	Error              string   `json:"error,omitempty"`
	AttemptNumber      int      `json:"attempt_number,omitempty"`
	BurstNumber        int      `json:"burst_number,omitempty"`
	ExecutedSteps      int      `json:"executed_steps,omitempty"`
	FinalAction        string   `json:"final_action,omitempty"`
	TargetPort         string   `json:"target_port,omitempty"`
	MessageType        string   `json:"message_type,omitempty"`
	MessageID          string   `json:"message_id,omitempty"`
	DispatchRef        string   `json:"dispatch_ref,omitempty"`
	PlannedCount       int      `json:"planned_count,omitempty"`
	AckedCount         int      `json:"acked_count,omitempty"`
	PendingCount       int      `json:"pending_count,omitempty"`
	FailedCount        int      `json:"failed_count,omitempty"`
	Issues             int      `json:"issues,omitempty"`
	ClaimedMessages    []string `json:"claimed_messages,omitempty"`
	AckedMessages      []string `json:"acked_messages,omitempty"`
	FirstPendingCount  int      `json:"first_pending_count,omitempty"`
	FirstPendingRefs   []string `json:"first_pending_refs,omitempty"`
	PendingOutboxCount int      `json:"pending_outbox_count,omitempty"`
	PendingOutboxRefs  []string `json:"pending_outbox_refs,omitempty"`
	EvidenceRefs       []string `json:"evidence_refs,omitempty"`
}

type RunExecutionSummaryV0 struct {
	RunRef       string                 `json:"run_ref"`
	AppRef       string                 `json:"app_ref,omitempty"`
	Rank         int                    `json:"rank"`
	Outcome      string                 `json:"outcome,omitempty"`
	QueueStatus  string                 `json:"queue_status,omitempty"`
	EvidenceRefs []string               `json:"evidence_refs,omitempty"`
	Diagnostics  []RunDrainDiagnosticV0 `json:"diagnostics,omitempty"`
}

type RunSkipSummaryV0 struct {
	RunRef string `json:"run_ref"`
	AppRef string `json:"app_ref,omitempty"`
	Rank   int    `json:"rank"`
	Reason string `json:"reason"`
	Status string `json:"status,omitempty"`
}

type RankedRunSummaryV0 struct {
	RunRef        string `json:"run_ref"`
	AppRef        string `json:"app_ref,omitempty"`
	Rank          int    `json:"rank"`
	PriorityScore int    `json:"priority_score"`
	AgingBoost    int    `json:"aging_boost"`
}
