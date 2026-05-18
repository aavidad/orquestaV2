package orquestacionnucleoapp

import (
	"context"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectorcycle "orquesta/modulos/orquesta-director-cycle"
)

type ReviewGateObservationProviderPortV0 interface {
	BuildReviewGateObservationsV0(
		ctx context.Context,
		request ReviewGateObservationRequestV0,
	) ([]ReviewGateObservationV0, error)
}

type ReviewGateObservationRequestV0 struct {
	Run              orquestacoreworkflow.OrchestrationRunV0
	StepNumber       int
	MaxSteps         int
	OccurredAt       string
	PreviousStep     *orquestadirectorcycle.DirectorCycleStepResultV0
	CorrelationID    string
	EvidenceRefs     []string
	PreviousDecision string
	WaitAgentRefs    []string
}

type ReviewGateObservationV0 struct {
	CandidateRef      string                                    `json:"candidate_ref,omitempty"`
	ReviewRequestID   string                                    `json:"review_request_id"`
	ReviewResultRef   string                                    `json:"review_result_ref"`
	AcceptedReviewRef string                                    `json:"accepted_review_ref,omitempty"`
	ReworkRequestRef  string                                    `json:"rework_request_ref,omitempty"`
	DeliveryRef       string                                    `json:"delivery_ref"`
	PhaseID           string                                    `json:"phase_id,omitempty"`
	Status            orquestacoreworkflow.ReviewResultStatusV0 `json:"status"`
	Summary           string                                    `json:"summary"`
	QualityGateRef    string                                    `json:"quality_gate_ref,omitempty"`
	EvidenceRefs      []string                                  `json:"evidence_refs,omitempty"`
}
