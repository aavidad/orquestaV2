package orquestadirectorsupervisor

import (
	orquestadirectorcycle "orquesta/modulos/orquesta-director-cycle"
	orquestadirectorscheduler "orquesta/modulos/orquesta-director-scheduler"
)

const (
	ErrDirectorSupervisorDecisionInvalidaV0 = "director_supervisor_decision_invalida"
	DirectorSupervisorMaxStepsLimitV0       = 50
)

type DirectorSupervisorActionV0 string
type DirectorSupervisorAutonomousRecommendationV0 string

const (
	DirectorSupervisorActionContinueV0      DirectorSupervisorActionV0 = "continue"
	DirectorSupervisorActionWaitOutboxV0    DirectorSupervisorActionV0 = "wait_outbox"
	DirectorSupervisorActionWaitExternalV0  DirectorSupervisorActionV0 = "wait_external"
	DirectorSupervisorActionNeedsDirectorV0 DirectorSupervisorActionV0 = "needs_director"
	DirectorSupervisorActionBlockedV0       DirectorSupervisorActionV0 = "blocked"
	DirectorSupervisorActionStopQuiescentV0 DirectorSupervisorActionV0 = "stop_quiescent"
	DirectorSupervisorActionStopMaxStepsV0  DirectorSupervisorActionV0 = "stop_max_steps"
	DirectorSupervisorActionStopErrorV0     DirectorSupervisorActionV0 = "stop_error"
)

const (
	DirectorSupervisorAutonomousContinueV0      DirectorSupervisorAutonomousRecommendationV0 = "continue"
	DirectorSupervisorAutonomousWaitV0          DirectorSupervisorAutonomousRecommendationV0 = "wait"
	DirectorSupervisorAutonomousNeedsDirectorV0 DirectorSupervisorAutonomousRecommendationV0 = "needs_director"
	DirectorSupervisorAutonomousStopV0          DirectorSupervisorAutonomousRecommendationV0 = "stop"
)

const (
	DirectorSupervisorReasonContinueV0      = "commands_applied_budget_available"
	DirectorSupervisorReasonOutboxPendingV0 = "outbox_pending"
	DirectorSupervisorReasonCandidateWaitV0 = "candidate_pending"
	DirectorSupervisorReasonExternalWaitV0  = "external_wait_required"
	DirectorSupervisorReasonNeedsDirectorV0 = "director_decision_required"
	DirectorSupervisorReasonBlockedV0       = "blocked"
	DirectorSupervisorReasonQuiescentV0     = "quiescent"
	DirectorSupervisorReasonMaxStepsV0      = "max_steps_reached"
	DirectorSupervisorReasonLastErrorV0     = "last_step_error"
)

type DirectorSupervisorDecisionInputV0 struct {
	RunRef         string                                          `json:"run_ref"`
	StepNumber     int                                             `json:"step_number"`
	MaxSteps       int                                             `json:"max_steps"`
	LastStepResult orquestadirectorcycle.DirectorCycleStepResultV0 `json:"last_step_result"`
	LastErrorCode  string                                          `json:"last_error_code,omitempty"`
	CorrelationID  string                                          `json:"correlation_id,omitempty"`
	EvidenceRefs   []string                                        `json:"evidence_refs,omitempty"`
}

type DirectorSupervisorDecisionV0 struct {
	RunRef                   string                                               `json:"run_ref"`
	Action                   DirectorSupervisorActionV0                           `json:"action"`
	ShouldContinue           bool                                                 `json:"should_continue"`
	AutonomousRecommendation DirectorSupervisorAutonomousRecommendationV0         `json:"autonomous_recommendation"`
	ReasonCode               string                                               `json:"reason_code"`
	StepNumber               int                                                  `json:"step_number"`
	MaxSteps                 int                                                  `json:"max_steps"`
	PendingOutboxRefs        []string                                             `json:"pending_outbox_refs,omitempty"`
	WaitingReasons           []orquestadirectorscheduler.SchedulerWaitingReasonV0 `json:"waiting_reasons,omitempty"`
	BlockedRefs              []string                                             `json:"blocked_refs,omitempty"`
	Issues                   []DirectorSupervisorIssueV0                          `json:"issues,omitempty"`
	EvidenceRefs             []string                                             `json:"evidence_refs,omitempty"`
}

type DirectorSupervisorIssueV0 struct {
	Code    string `json:"code"`
	Field   string `json:"field,omitempty"`
	Message string `json:"message,omitempty"`
}

type DirectorSupervisorErrorV0 struct {
	Code          string                      `json:"code"`
	Message       string                      `json:"message"`
	Field         string                      `json:"field,omitempty"`
	Retryable     bool                        `json:"retryable"`
	Issues        []DirectorSupervisorIssueV0 `json:"issues,omitempty"`
	CorrelationID string                      `json:"correlation_id,omitempty"`
}

func (err DirectorSupervisorErrorV0) Error() string {
	return err.Code
}
