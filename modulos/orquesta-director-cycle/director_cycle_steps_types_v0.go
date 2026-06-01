package orquestadirectorcycle

import (
	"context"
	"strings"

	orquestacoreconcurrency "orquesta/modulos/orquesta-core-concurrency"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectorrunner "orquesta/modulos/orquesta-director-runner"
	orquestadirectorscheduler "orquesta/modulos/orquesta-director-scheduler"
)

const (
	ErrDirectorCycleStepsInvalidoV0 = "director_cycle_steps_invalido"
	ErrDirectorCycleStepsSnapshotV0 = "director_cycle_steps_snapshot"
	ErrDirectorCycleStepsStepV0     = "director_cycle_steps_step"

	DirectorCycleStepsMaxStepsLimitV0 = 50
)

const (
	DirectorCycleStepsStopWaitOutboxV0    = "wait_outbox"
	DirectorCycleStepsStopWaitExternalV0  = "wait_external"
	DirectorCycleStepsStopBlockedV0       = "blocked"
	DirectorCycleStepsStopNeedsDirectorV0 = "needs_director"
	DirectorCycleStepsStopQuiescentV0     = "stop_quiescent"
	DirectorCycleStepsStopMaxStepsV0      = "stop_max_steps"
	DirectorCycleStepsStopErrorV0         = "stop_error"
)

type DirectorCycleStepsSnapshotPortV0 interface {
	LoadDirectorCycleStepSnapshotV0(
		ctx context.Context,
		request DirectorCycleStepSnapshotRequestV0,
	) (DirectorCycleStepSnapshotV0, error)
}

type DirectorCycleRunSnapshotPortV0 interface {
	LoadDirectorCycleRunSnapshotV0(
		ctx context.Context,
		runRef string,
	) (orquestacoreworkflow.OrchestrationRunV0, error)
}

type DirectorCycleStepsInputV0 struct {
	InitialStep   DirectorCycleStepInputV0         `json:"initial_step"`
	StepTemplate  DirectorCycleStepInputV0         `json:"step_template,omitempty"`
	SnapshotPort  DirectorCycleStepsSnapshotPortV0 `json:"-"`
	RunSnapshot   DirectorCycleRunSnapshotPortV0   `json:"-"`
	MaxSteps      int                              `json:"max_steps"`
	CorrelationID string                           `json:"correlation_id,omitempty"`
	EvidenceRefs  []string                         `json:"evidence_refs,omitempty"`
}

type DirectorCycleStepSnapshotRequestV0 struct {
	RunRef             string                    `json:"run_ref"`
	StepNumber         int                       `json:"step_number"`
	MaxSteps           int                       `json:"max_steps"`
	PreviousStepResult DirectorCycleStepResultV0 `json:"previous_step_result"`
	CorrelationID      string                    `json:"correlation_id,omitempty"`
	EvidenceRefs       []string                  `json:"evidence_refs,omitempty"`
}

type DirectorCycleStepSnapshotV0 struct {
	CycleRef                      string                                                                `json:"cycle_ref"`
	TickRef                       string                                                                `json:"tick_ref"`
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
	EvidenceRefs                  []string                                                              `json:"evidence_refs,omitempty"`
}

type DirectorCycleStepsResultV0 struct {
	RunRef            string                                               `json:"run_ref"`
	MaxSteps          int                                                  `json:"max_steps"`
	ExecutedSteps     int                                                  `json:"executed_steps"`
	StepsExecuted     int                                                  `json:"steps_executed,omitempty"`
	MaxStepsReached   bool                                                 `json:"max_steps_reached,omitempty"`
	StopReason        string                                               `json:"stop_reason,omitempty"`
	Status            orquestadirectorrunner.DirectorCycleStatusV0         `json:"status,omitempty"`
	LastStepResult    DirectorCycleStepResultV0                            `json:"last_step_result,omitempty"`
	StepResults       []DirectorCycleStepResultV0                          `json:"step_results,omitempty"`
	PendingOutboxRefs []string                                             `json:"pending_outbox_refs,omitempty"`
	WaitingReasons    []orquestadirectorscheduler.SchedulerWaitingReasonV0 `json:"waiting_reasons,omitempty"`
	BlockedRefs       []string                                             `json:"blocked_refs,omitempty"`
	LastErrorCode     string                                               `json:"last_error_code,omitempty"`
	Issues            []DirectorCycleStepsIssueV0                          `json:"issues,omitempty"`
	CorrelationID     string                                               `json:"correlation_id,omitempty"`
	EvidenceRefs      []string                                             `json:"evidence_refs,omitempty"`
}

type DirectorCycleStepsIssueV0 struct {
	Code    string `json:"code"`
	Field   string `json:"field,omitempty"`
	Message string `json:"message,omitempty"`
}

type DirectorCycleStepsErrorV0 struct {
	Code          string                      `json:"code"`
	Message       string                      `json:"message"`
	Field         string                      `json:"field,omitempty"`
	Retryable     bool                        `json:"retryable"`
	Issues        []DirectorCycleStepsIssueV0 `json:"issues,omitempty"`
	CorrelationID string                      `json:"correlation_id,omitempty"`
}

func (err DirectorCycleStepsErrorV0) Error() string {
	parts := []string{strings.TrimSpace(err.Code)}
	if field := strings.TrimSpace(err.Field); field != "" {
		parts = append(parts, "field="+field)
	}
	if message := strings.TrimSpace(err.Message); message != "" {
		parts = append(parts, message)
	}
	return strings.Join(compactDirectorCycleStepStringsV0(parts), ": ")
}
