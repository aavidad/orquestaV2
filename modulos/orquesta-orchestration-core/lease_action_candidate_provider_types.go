package orquestacionnucleoapp

import (
	"context"

	orquestacoreleases "orquesta/modulos/orquesta-core-leases"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectorcycle "orquesta/modulos/orquesta-director-cycle"
)

type AgentLeaseAssessmentProviderPortV0 interface {
	BuildAgentLeaseAssessmentsV0(
		ctx context.Context,
		request AgentLeaseAssessmentRequestV0,
	) ([]orquestacoreleases.AgentTimeoutAssessmentV0, error)
}

type AgentLeaseAssessmentRequestV0 struct {
	Run              orquestacoreworkflow.OrchestrationRunV0
	StepNumber       int
	MaxSteps         int
	OccurredAt       string
	PreviousStep     *orquestadirectorcycle.DirectorCycleStepResultV0
	CorrelationID    string
	EvidenceRefs     []string
	PreviousDecision string
}
