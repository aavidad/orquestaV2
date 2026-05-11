package orquestadirectortickinput

import (
	orquestacoreconcurrency "orquesta/modulos/orquesta-core-concurrency"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectorscheduler "orquesta/modulos/orquesta-director-scheduler"
)

const ErrDirectorTickInputBuildInvalidoV0 = "director_tick_input_build_invalido"

type DirectorTickInputBuildRequestV0 struct {
	TickRef                       string                                                                `json:"tick_ref"`
	OccurredAt                    string                                                                `json:"occurred_at"`
	Run                           orquestacoreworkflow.OrchestrationRunV0                               `json:"run"`
	PendingOutboxRefs             []string                                                              `json:"pending_outbox_refs,omitempty"`
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

type DirectorTickInputBuildErrorV0 struct {
	Code  string `json:"code"`
	Field string `json:"field,omitempty"`
}

func (err DirectorTickInputBuildErrorV0) Error() string {
	return err.Code
}
