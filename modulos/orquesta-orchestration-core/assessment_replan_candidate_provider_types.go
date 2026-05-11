package orquestacionnucleoapp

import (
	"context"

	orquestacorereplanner "orquesta/modulos/orquesta-core-replanner"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectorcycle "orquesta/modulos/orquesta-director-cycle"
)

type AgentAssessmentReplanPlanProviderPortV0 interface {
	BuildAgentAssessmentReplanPlansV0(
		context.Context,
		AgentAssessmentReplanPlanRequestV0,
	) ([]AgentAssessmentReplanPlanV0, error)
}

type AgentAssessmentReplanPlanRequestV0 struct {
	Run              orquestacoreworkflow.OrchestrationRunV0
	StepNumber       int
	MaxSteps         int
	OccurredAt       string
	PreviousStep     *orquestadirectorcycle.DirectorCycleStepResultV0
	CorrelationID    string
	EvidenceRefs     []string
	PreviousDecision string
}

type AgentAssessmentReplanPlanV0 struct {
	CandidateRef               string
	ReplanRef                  string
	SignalRef                  string
	TaskRef                    string
	ReasonRef                  string
	RequestedAction            orquestacorereplanner.ReplanRecommendedActionV0
	ReplacementRole            string
	CapacityRequestRef         string
	AgentRequestID             string
	MinimumRecommendedCapacity orquestacoreworkflow.OrchestrationCapacityRecommendationV0
	Summary                    string
	EvidenceRefs               []string
	Assessment                 orquestacoreworkflow.AgentWorkAssessedPayloadV0
}
