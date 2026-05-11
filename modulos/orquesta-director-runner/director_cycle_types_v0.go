package orquestadirectorrunner

import (
	"context"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectorscheduler "orquesta/modulos/orquesta-director-scheduler"
)

const (
	DirectorCycleStatusQuiescentV0       DirectorCycleStatusV0 = "quiescent"
	DirectorCycleStatusWaitingV0         DirectorCycleStatusV0 = "waiting"
	DirectorCycleStatusBlockedV0         DirectorCycleStatusV0 = "blocked"
	DirectorCycleStatusNeedsDirectorV0   DirectorCycleStatusV0 = "needs_director"
	DirectorCycleStatusCommandsAppliedV0 DirectorCycleStatusV0 = "commands_applied"
	DirectorCycleStatusOutboxPendingV0   DirectorCycleStatusV0 = "outbox_pending"
)

const (
	ErrDirectorRunnerCycleInvalidoV0 = "director_runner_cycle_invalido"
	ErrDirectorRunnerSchedulerV0     = "director_runner_scheduler"
	ErrDirectorRunnerWorkflowV0      = "director_runner_workflow"
)

const (
	DirectorCycleStopSchedulerWaitingV0  = "scheduler_waiting"
	DirectorCycleStopSchedulerBlockedV0  = "scheduler_blocked"
	DirectorCycleStopNeedsDirectorV0     = "scheduler_needs_director"
	DirectorCycleStopQuiescentV0         = "scheduler_quiescent"
	DirectorCycleStopOutboxGeneratedV0   = "outbox_generated"
	DirectorCycleStopCommandsExhaustedV0 = "commands_exhausted"
)

type DirectorCycleStatusV0 string

type DirectorSchedulerPortV0 interface {
	BuildDirectorSchedulerTickV0(
		ctx context.Context,
		input orquestadirectorscheduler.DirectorSchedulerTickInputV0,
	) (orquestadirectorscheduler.DirectorSchedulerTickPlanV0, error)
}

type WorkflowCommandPortV0 interface {
	HandleWorkflowCommandV0(
		ctx context.Context,
		command orquestacoreworkflow.OrchestrationCommandV0,
	) (orquestacoreworkflow.OrchestrationCommandResultV0, error)
}

type DirectorCycleInputV0 struct {
	Scheduler      DirectorSchedulerPortV0                                `json:"-"`
	Workflow       WorkflowCommandPortV0                                  `json:"-"`
	CycleRef       string                                                 `json:"cycle_ref"`
	RunRef         string                                                 `json:"run_ref"`
	SchedulerInput orquestadirectorscheduler.DirectorSchedulerTickInputV0 `json:"scheduler_input"`
	MaxCommands    int                                                    `json:"max_commands,omitempty"`
	MaxOutbox      int                                                    `json:"max_outbox,omitempty"`
	CorrelationID  string                                                 `json:"correlation_id,omitempty"`
	EvidenceRefs   []string                                               `json:"evidence_refs,omitempty"`
}

type DirectorCycleResultV0 struct {
	CycleRef         string                                                  `json:"cycle_ref"`
	RunRef           string                                                  `json:"run_ref"`
	Status           DirectorCycleStatusV0                                   `json:"status"`
	SchedulerStatus  orquestadirectorscheduler.DirectorSchedulerTickStatusV0 `json:"scheduler_status,omitempty"`
	SchedulerSummary string                                                  `json:"scheduler_summary,omitempty"`
	AppliedCommands  []DirectorCycleAppliedCommandV0                         `json:"applied_commands,omitempty"`
	EventsCount      int                                                     `json:"events_count,omitempty"`
	Outbox           []orquestacoreworkflow.OutboxMessageV0                  `json:"outbox,omitempty"`
	WaitingReasons   []orquestadirectorscheduler.SchedulerWaitingReasonV0    `json:"waiting_reasons,omitempty"`
	BlockedRefs      []string                                                `json:"blocked_refs,omitempty"`
	StopReason       string                                                  `json:"stop_reason,omitempty"`
	Issues           []DirectorCycleIssueV0                                  `json:"issues,omitempty"`
	EvidenceRefs     []string                                                `json:"evidence_refs,omitempty"`
}

type DirectorCycleAppliedCommandV0 struct {
	CommandID   string `json:"command_id"`
	CommandType string `json:"command_type"`
	EventCount  int    `json:"event_count"`
	OutboxCount int    `json:"outbox_count"`
}

type DirectorCycleIssueV0 struct {
	Code    string `json:"code"`
	Field   string `json:"field,omitempty"`
	Message string `json:"message,omitempty"`
}

type DirectorCycleErrorV0 struct {
	Code          string                 `json:"code"`
	Message       string                 `json:"message"`
	Field         string                 `json:"field,omitempty"`
	Retryable     bool                   `json:"retryable"`
	Issues        []DirectorCycleIssueV0 `json:"issues,omitempty"`
	CorrelationID string                 `json:"correlation_id,omitempty"`
}

func (err DirectorCycleErrorV0) Error() string {
	return err.Code
}
