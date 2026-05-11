package orquestacionnucleoapp

import (
	"context"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectorcycle "orquesta/modulos/orquesta-director-cycle"
)

type AgentDeliveryObservationProviderPortV0 interface {
	BuildAgentDeliveryObservationsV0(
		ctx context.Context,
		request AgentDeliveryObservationRequestV0,
	) ([]AgentDeliveryObservationV0, error)
}

type AgentDeliveryObservationRequestV0 struct {
	Run              orquestacoreworkflow.OrchestrationRunV0
	StepNumber       int
	MaxSteps         int
	OccurredAt       string
	PreviousStep     *orquestadirectorcycle.DirectorCycleStepResultV0
	CorrelationID    string
	EvidenceRefs     []string
	PreviousDecision string
}

type AgentDeliveryObservationV0 struct {
	CandidateRef string   `json:"candidate_ref,omitempty"`
	ArtifactRef  string   `json:"artifact_ref,omitempty"`
	DeliveryRef  string   `json:"delivery_ref"`
	PhaseID      string   `json:"phase_id,omitempty"`
	TaskID       string   `json:"task_id"`
	AgentRef     string   `json:"agent_ref"`
	Summary      string   `json:"summary"`
	EvidenceRefs []string `json:"evidence_refs,omitempty"`
}
