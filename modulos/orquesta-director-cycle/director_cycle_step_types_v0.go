package orquestadirectorcycle

import (
	"strings"

	orquestacoreconcurrency "orquesta/modulos/orquesta-core-concurrency"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectorcycleoutbox "orquesta/modulos/orquesta-director-cycle-outbox"
	orquestadirectorrunner "orquesta/modulos/orquesta-director-runner"
	orquestadirectorscheduler "orquesta/modulos/orquesta-director-scheduler"
)

const (
	ErrDirectorCycleStepInvalidoV0  = "director_cycle_step_invalido"
	ErrDirectorCycleStepOutboxV0    = "director_cycle_step_outbox"
	ErrDirectorCycleStepTickInputV0 = "director_cycle_step_tick_input"
	ErrDirectorCycleStepRunnerV0    = "director_cycle_step_runner"
)

type DirectorCycleStepInputV0 struct {
	Scheduler                     orquestadirectorrunner.DirectorSchedulerPortV0                        `json:"-"`
	Workflow                      orquestadirectorrunner.WorkflowCommandPortV0                          `json:"-"`
	OutboxLedger                  orquestadirectorcycleoutbox.DirectorCycleOutboxLedgerPortV0           `json:"-"`
	CycleRef                      string                                                                `json:"cycle_ref"`
	TickRef                       string                                                                `json:"tick_ref"`
	RunRef                        string                                                                `json:"run_ref"`
	OccurredAt                    string                                                                `json:"occurred_at"`
	Run                           orquestacoreworkflow.OrchestrationRunV0                               `json:"run"`
	LeaseActionCandidates         []orquestadirectorscheduler.SchedulableLeaseActionCandidateV0         `json:"lease_action_candidates,omitempty"`
	PhaseArtifactCandidates       []orquestadirectorscheduler.SchedulablePhaseArtifactCandidateV0       `json:"phase_artifact_candidates,omitempty"`
	DeliveryCandidates            []orquestadirectorscheduler.SchedulableDeliveryCandidateV0            `json:"delivery_candidates,omitempty"`
	ReviewGateCandidates          []orquestadirectorscheduler.SchedulableReviewGateCandidateV0          `json:"review_gate_candidates,omitempty"`
	ProgressSupervisionCandidates []orquestadirectorscheduler.SchedulableProgressSupervisionCandidateV0 `json:"progress_supervision_candidates,omitempty"`
	ReplanFollowupCandidates      []orquestadirectorscheduler.SchedulableReplanFollowupCandidateV0      `json:"replan_followup_candidates,omitempty"`
	WorkClaims                    []orquestacoreconcurrency.WorksetClaimV0                              `json:"work_claims,omitempty"`
	WorkCandidates                []orquestadirectorscheduler.SchedulableWorkCandidateV0                `json:"work_candidates,omitempty"`
	MaxCommands                   int                                                                   `json:"max_commands,omitempty"`
	MaxOutbox                     int                                                                   `json:"max_outbox,omitempty"`
	CorrelationID                 string                                                                `json:"correlation_id,omitempty"`
	EvidenceRefs                  []string                                                              `json:"evidence_refs,omitempty"`
}

type DirectorCycleStepResultV0 struct {
	CycleRef                string                                                  `json:"cycle_ref"`
	TickRef                 string                                                  `json:"tick_ref"`
	RunRef                  string                                                  `json:"run_ref"`
	Status                  orquestadirectorrunner.DirectorCycleStatusV0            `json:"status"`
	SchedulerStatus         orquestadirectorscheduler.DirectorSchedulerTickStatusV0 `json:"scheduler_status,omitempty"`
	StopReason              string                                                  `json:"stop_reason,omitempty"`
	PendingOutboxBeforeRefs []string                                                `json:"pending_outbox_before_refs,omitempty"`
	PendingOutboxAfterRefs  []string                                                `json:"pending_outbox_after_refs,omitempty"`
	AppliedCommands         []orquestadirectorrunner.DirectorCycleAppliedCommandV0  `json:"applied_commands,omitempty"`
	WaitingReasons          []orquestadirectorscheduler.SchedulerWaitingReasonV0    `json:"waiting_reasons,omitempty"`
	BlockedRefs             []string                                                `json:"blocked_refs,omitempty"`
	EventsCount             int                                                     `json:"events_count,omitempty"`
	OutboxSavedCount        int                                                     `json:"outbox_saved_count,omitempty"`
	OutboxPendingAfterCount int                                                     `json:"outbox_pending_after_count,omitempty"`
	Issues                  []DirectorCycleStepIssueV0                              `json:"issues,omitempty"`
}

type DirectorCycleStepIssueV0 struct {
	Code    string `json:"code"`
	Field   string `json:"field,omitempty"`
	Message string `json:"message,omitempty"`
}

type DirectorCycleStepErrorV0 struct {
	Code          string                     `json:"code"`
	Message       string                     `json:"message"`
	Field         string                     `json:"field,omitempty"`
	Retryable     bool                       `json:"retryable"`
	Issues        []DirectorCycleStepIssueV0 `json:"issues,omitempty"`
	CorrelationID string                     `json:"correlation_id,omitempty"`
}

func (err DirectorCycleStepErrorV0) Error() string {
	parts := []string{strings.TrimSpace(err.Code)}
	if field := strings.TrimSpace(err.Field); field != "" {
		parts = append(parts, "field="+field)
	}
	if message := strings.TrimSpace(err.Message); message != "" {
		parts = append(parts, message)
	}
	if len(err.Issues) > 0 {
		issue := err.Issues[len(err.Issues)-1]
		detail := strings.Join(compactDirectorCycleStepStringsV0([]string{
			strings.TrimSpace(issue.Code),
			strings.TrimSpace(issue.Field),
			strings.TrimSpace(issue.Message),
		}), ": ")
		if detail != "" {
			parts = append(parts, "issue="+detail)
		}
	}
	return strings.Join(compactDirectorCycleStepStringsV0(parts), ": ")
}
