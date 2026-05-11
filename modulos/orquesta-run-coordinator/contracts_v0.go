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
	ControlReader orquestaruncontrol.RunControlReaderPortV0
	Drainer       RunDrainerPortV0
}

type RunCoordinatorTickCommandV0 struct {
	QueueRef       string                                   `json:"queue_ref,omitempty"`
	AppRefs        []string                                 `json:"app_refs,omitempty"`
	ExcludeRunRefs []string                                 `json:"exclude_run_refs,omitempty"`
	QueueLimit     int                                      `json:"queue_limit,omitempty"`
	MaxRuns        int                                      `json:"max_runs,omitempty"`
	OccurredAt     time.Time                                `json:"occurred_at"`
	CorrelationID  string                                   `json:"correlation_id,omitempty"`
	DrainLimits    RunDrainLimitsV0                         `json:"drain_limits,omitempty"`
	RankingPolicy  orquestarunqueue.RunQueueRankingPolicyV0 `json:"ranking_policy,omitempty"`
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
	RunRef       string   `json:"run_ref"`
	AppRef       string   `json:"app_ref,omitempty"`
	Outcome      string   `json:"outcome,omitempty"`
	EvidenceRefs []string `json:"evidence_refs,omitempty"`
}

type RunExecutionSummaryV0 struct {
	RunRef       string   `json:"run_ref"`
	AppRef       string   `json:"app_ref,omitempty"`
	Rank         int      `json:"rank"`
	Outcome      string   `json:"outcome,omitempty"`
	EvidenceRefs []string `json:"evidence_refs,omitempty"`
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
