package orquestacionnucleoapp

import (
	"context"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectorcycle "orquesta/modulos/orquesta-director-cycle"
	orquestaruntime "orquesta/modulos/orquesta-runtime"
)

type AgentProgressObservationProviderPortV0 interface {
	BuildAgentProgressObservationsV0(
		ctx context.Context,
		request AgentProgressObservationRequestV0,
	) ([]AgentProgressObservationV0, error)
}

type AgentProgressObservationRequestV0 struct {
	Run              orquestacoreworkflow.OrchestrationRunV0
	StepNumber       int
	MaxSteps         int
	OccurredAt       string
	PreviousStep     *orquestadirectorcycle.DirectorCycleStepResultV0
	CorrelationID    string
	EvidenceRefs     []string
	PreviousDecision string
}

type AgentProgressObservationV0 struct {
	CandidateRef     string                                `json:"candidate_ref,omitempty"`
	Report           orquestaruntime.AgentProgressReportV0 `json:"report"`
	PhaseID          string                                `json:"phase_id,omitempty"`
	TaskRef          string                                `json:"task_ref,omitempty"`
	DeliveryRef      string                                `json:"delivery_ref,omitempty"`
	AssessmentRef    string                                `json:"assessment_ref,omitempty"`
	QuestionID       string                                `json:"question_id,omitempty"`
	DecisionRequired bool                                  `json:"decision_required,omitempty"`
	EvidenceRefs     []string                              `json:"evidence_refs,omitempty"`
}
