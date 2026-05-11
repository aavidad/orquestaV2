package orquestadirectorscheduler

import (
	orquestacoreconcurrency "orquesta/modulos/orquesta-core-concurrency"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirector "orquesta/modulos/orquesta-director"
)

const ErrDirectorSchedulerTickInvalidoV0 = "director_scheduler_tick_invalido"

type DirectorSchedulerTickStatusV0 string

const (
	SchedulerTickStatusCommandsReadyV0 DirectorSchedulerTickStatusV0 = "commands_ready"
	SchedulerTickStatusWaitingV0       DirectorSchedulerTickStatusV0 = "waiting"
	SchedulerTickStatusBlockedV0       DirectorSchedulerTickStatusV0 = "blocked"
	SchedulerTickStatusNeedsDirectorV0 DirectorSchedulerTickStatusV0 = "needs_director"
	SchedulerTickStatusQuiescentV0     DirectorSchedulerTickStatusV0 = "quiescent"
)

type SchedulerWaitingReasonV0 string

const (
	SchedulerWaitingOutboxPendingV0           SchedulerWaitingReasonV0 = "outbox_pending"
	SchedulerWaitingCapacityPendingV0         SchedulerWaitingReasonV0 = "capacity_pending"
	SchedulerWaitingAgentLifecyclePendingV0   SchedulerWaitingReasonV0 = "agent_lifecycle_pending"
	SchedulerWaitingAgentDeliveryPendingV0    SchedulerWaitingReasonV0 = "agent_delivery_pending"
	SchedulerWaitingCandidateMissingV0        SchedulerWaitingReasonV0 = "candidate_missing"
	SchedulerWaitingDirectorQuestionPendingV0 SchedulerWaitingReasonV0 = "director_question_pending"
	SchedulerWaitingQualityGateFollowupV0     SchedulerWaitingReasonV0 = "quality_gate_followup_required"
)

type DirectorSchedulerTickInputV0 struct {
	TickRef                       string                                      `json:"tick_ref"`
	RunRef                        string                                      `json:"run_ref"`
	OccurredAt                    string                                      `json:"occurred_at"`
	Snapshot                      RunSchedulingSnapshotV0                     `json:"snapshot"`
	LeaseActionCandidates         []SchedulableLeaseActionCandidateV0         `json:"lease_action_candidates,omitempty"`
	PhaseArtifactCandidates       []SchedulablePhaseArtifactCandidateV0       `json:"phase_artifact_candidates,omitempty"`
	DeliveryCandidates            []SchedulableDeliveryCandidateV0            `json:"delivery_candidates,omitempty"`
	ReviewGateCandidates          []SchedulableReviewGateCandidateV0          `json:"review_gate_candidates,omitempty"`
	ProgressSupervisionCandidates []SchedulableProgressSupervisionCandidateV0 `json:"progress_supervision_candidates,omitempty"`
	ReplanFollowupCandidates      []SchedulableReplanFollowupCandidateV0      `json:"replan_followup_candidates,omitempty"`
	WorkClaims                    []orquestacoreconcurrency.WorksetClaimV0    `json:"work_claims,omitempty"`
	WorkCandidates                []SchedulableWorkCandidateV0                `json:"work_candidates,omitempty"`
	EvidenceRefs                  []string                                    `json:"evidence_refs,omitempty"`
}

type RunSchedulingSnapshotV0 struct {
	RunRef                    string   `json:"run_ref"`
	CurrentPhaseID            string   `json:"current_phase_id"`
	Tasks                     []string `json:"tasks,omitempty"`
	CapacityRequests          []string `json:"capacity_requests,omitempty"`
	CapacityDecisions         []string `json:"capacity_decisions,omitempty"`
	ConcurrencyGates          []string `json:"concurrency_gates,omitempty"`
	Agents                    []string `json:"agents,omitempty"`
	StartedAgents             []string `json:"started_agents,omitempty"`
	FailedAgents              []string `json:"failed_agents,omitempty"`
	StoppedAgents             []string `json:"stopped_agents,omitempty"`
	PhaseArtifacts            []string `json:"phase_artifacts,omitempty"`
	Deliveries                []string `json:"deliveries,omitempty"`
	Reviews                   []string `json:"reviews,omitempty"`
	ReviewResults             []string `json:"review_results,omitempty"`
	AcceptedReviews           []string `json:"accepted_reviews,omitempty"`
	ReworkRequests            []string `json:"rework_requests,omitempty"`
	AgentAssessments          []string `json:"agent_assessments,omitempty"`
	DirectorQuestions         []string `json:"director_questions,omitempty"`
	DirectorAnsweredQuestions []string `json:"director_answered_questions,omitempty"`
	ExpiredLeaseRefs          []string `json:"expired_lease_refs,omitempty"`
	ReplanRefs                []string `json:"replan_refs,omitempty"`
	BlockingQualityGateRefs   []string `json:"blocking_quality_gate_refs,omitempty"`
	PendingOutboxRefs         []string `json:"pending_outbox_refs,omitempty"`
}

type SchedulableWorkCandidateV0 struct {
	CandidateRef      string                                          `json:"candidate_ref"`
	SubjectClaimRefs  []string                                        `json:"subject_claim_refs"`
	Claims            []orquestacoreconcurrency.WorksetClaimV0        `json:"claims"`
	CapacityCandidate *SchedulerCapacityCommandCandidateV0            `json:"capacity_candidate,omitempty"`
	AgentCandidate    *SchedulerAgentCommandCandidateV0               `json:"agent_candidate,omitempty"`
	GateCommandMeta   orquestacoreworkflow.OrchestrationCommandMetaV0 `json:"gate_command_meta"`
	GateEvidenceRefs  []string                                        `json:"gate_evidence_refs,omitempty"`
	EvidenceRefs      []string                                        `json:"evidence_refs,omitempty"`
}

type SchedulerCapacityCommandCandidateV0 struct {
	CommandMeta orquestacoreworkflow.OrchestrationCommandMetaV0      `json:"command_meta"`
	Payload     orquestacoreworkflow.RequestCapacityCommandPayloadV0 `json:"payload"`
}

type SchedulerAgentCommandCandidateV0 struct {
	ClaimRef    string                                            `json:"claim_ref"`
	CommandMeta orquestacoreworkflow.OrchestrationCommandMetaV0   `json:"command_meta"`
	Payload     orquestacoreworkflow.RequestAgentCommandPayloadV0 `json:"payload"`
}

type SchedulableLeaseActionCandidateV0 struct {
	CandidateRef         string                                  `json:"candidate_ref"`
	PostLeaseActionInput orquestadirector.PostLeaseActionInputV0 `json:"post_lease_action_input"`
	EvidenceRefs         []string                                `json:"evidence_refs,omitempty"`
}

type SchedulableDeliveryCandidateV0 struct {
	CandidateRef string                                                `json:"candidate_ref"`
	CommandMeta  orquestacoreworkflow.OrchestrationCommandMetaV0       `json:"command_meta"`
	Payload      orquestacoreworkflow.RegisterDeliveryCommandPayloadV0 `json:"payload"`
	EvidenceRefs []string                                              `json:"evidence_refs,omitempty"`
}

type SchedulablePhaseArtifactCandidateV0 struct {
	CandidateRef string                                                     `json:"candidate_ref"`
	CommandMeta  orquestacoreworkflow.OrchestrationCommandMetaV0            `json:"command_meta"`
	Payload      orquestacoreworkflow.RegisterPhaseArtifactCommandPayloadV0 `json:"payload"`
	EvidenceRefs []string                                                   `json:"evidence_refs,omitempty"`
}

type SchedulableReviewGateCandidateV0 struct {
	CandidateRef       string                                  `json:"candidate_ref"`
	RequestReview      *SchedulerRequestReviewCandidateV0      `json:"request_review,omitempty"`
	RecordReviewResult *SchedulerRecordReviewResultCandidateV0 `json:"record_review_result,omitempty"`
	AcceptReview       *SchedulerAcceptReviewCandidateV0       `json:"accept_review,omitempty"`
	RequestRework      *SchedulerRequestReworkCandidateV0      `json:"request_rework,omitempty"`
	EvidenceRefs       []string                                `json:"evidence_refs,omitempty"`
}

type SchedulerRequestReviewCandidateV0 struct {
	CommandMeta orquestacoreworkflow.OrchestrationCommandMetaV0    `json:"command_meta"`
	Payload     orquestacoreworkflow.RequestReviewCommandPayloadV0 `json:"payload"`
}

type SchedulerRecordReviewResultCandidateV0 struct {
	CommandMeta orquestacoreworkflow.OrchestrationCommandMetaV0         `json:"command_meta"`
	Payload     orquestacoreworkflow.RecordReviewResultCommandPayloadV0 `json:"payload"`
}

type SchedulerAcceptReviewCandidateV0 struct {
	CommandMeta orquestacoreworkflow.OrchestrationCommandMetaV0   `json:"command_meta"`
	Payload     orquestacoreworkflow.AcceptReviewCommandPayloadV0 `json:"payload"`
}

type SchedulerRequestReworkCandidateV0 struct {
	CommandMeta orquestacoreworkflow.OrchestrationCommandMetaV0    `json:"command_meta"`
	Payload     orquestacoreworkflow.RequestReworkCommandPayloadV0 `json:"payload"`
}

type SchedulableProgressSupervisionCandidateV0 struct {
	CandidateRef     string                                           `json:"candidate_ref"`
	SupervisionInput orquestadirector.AgentProgressSupervisionInputV0 `json:"supervision_input"`
	EvidenceRefs     []string                                         `json:"evidence_refs,omitempty"`
}

type SchedulableReplanFollowupCandidateV0 struct {
	CandidateRef         string                                  `json:"candidate_ref"`
	ReplanFollowupsInput orquestadirector.ReplanFollowupsInputV0 `json:"replan_followups_input"`
	EvidenceRefs         []string                                `json:"evidence_refs,omitempty"`
}

type DirectorSchedulerTickPlanV0 struct {
	TickRef        string                                        `json:"tick_ref"`
	RunRef         string                                        `json:"run_ref"`
	Status         DirectorSchedulerTickStatusV0                 `json:"status"`
	Commands       []orquestacoreworkflow.OrchestrationCommandV0 `json:"commands,omitempty"`
	WaitingReasons []SchedulerWaitingReasonV0                    `json:"waiting_reasons,omitempty"`
	BlockedRefs    []string                                      `json:"blocked_refs,omitempty"`
	Summary        string                                        `json:"summary"`
	EvidenceRefs   []string                                      `json:"evidence_refs,omitempty"`
}

type DirectorSchedulerTickErrorV0 struct {
	Code  string `json:"code"`
	Field string `json:"field,omitempty"`
}

func (err DirectorSchedulerTickErrorV0) Error() string {
	return err.Code
}
