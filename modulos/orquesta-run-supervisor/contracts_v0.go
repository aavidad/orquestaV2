package orquestarunsupervisor

import (
	"context"
	"time"

	orquestaruncoordinator "orquesta/modulos/orquesta-run-coordinator"
	orquestarunqueue "orquesta/modulos/orquesta-run-queue"
)

const RunSupervisorSchemaVersionV0 = "run_supervisor.v0"

const (
	RunSupervisorStopMaxTicksV0      = "max_ticks"
	RunSupervisorStopMaxExecutionsV0 = "max_executions"
	RunSupervisorStopNoExecutionV0   = "no_execution"
	RunSupervisorStopContextDoneV0   = "context_done"
	RunSupervisorStopTickErrorV0     = "tick_error"
	RunSupervisorStopNoTickerV0      = "ticker_required"
)

type RunSupervisorTickPortV0 interface {
	RunGlobalTickV0(
		context.Context,
		orquestaruncoordinator.RunCoordinatorTickCommandV0,
	) (orquestaruncoordinator.RunCoordinatorTickResultV0, error)
}

type RunSupervisorDepsV0 struct {
	Ticker RunSupervisorTickPortV0
}

type RunSupervisorCommandV0 struct {
	QueueRef          string                                   `json:"queue_ref,omitempty"`
	AppRefs           []string                                 `json:"app_refs,omitempty"`
	QueueLimit        int                                      `json:"queue_limit,omitempty"`
	MaxRunsPerTick    int                                      `json:"max_runs_per_tick,omitempty"`
	MaxTicks          int                                      `json:"max_ticks,omitempty"`
	MaxExecutions     int                                      `json:"max_executions,omitempty"`
	StopOnNoExecution bool                                     `json:"stop_on_no_execution,omitempty"`
	AllowRepeatedRuns bool                                     `json:"allow_repeated_runs,omitempty"`
	OccurredAt        time.Time                                `json:"occurred_at"`
	CorrelationID     string                                   `json:"correlation_id,omitempty"`
	DrainLimits       orquestaruncoordinator.RunDrainLimitsV0  `json:"drain_limits,omitempty"`
	RankingPolicy     orquestarunqueue.RunQueueRankingPolicyV0 `json:"ranking_policy,omitempty"`
}

type RunSupervisorResultV0 struct {
	Ticks           []RunSupervisorTickSummaryV0 `json:"ticks,omitempty"`
	TotalExecutions int                          `json:"total_executions"`
	TotalSkips      int                          `json:"total_skips"`
	StopReason      string                       `json:"stop_reason"`
}

type RunSupervisorTickSummaryV0 struct {
	TickNumber int                                               `json:"tick_number"`
	Excluded   []string                                          `json:"excluded_run_refs,omitempty"`
	Result     orquestaruncoordinator.RunCoordinatorTickResultV0 `json:"result"`
}
