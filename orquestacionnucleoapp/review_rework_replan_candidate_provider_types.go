package orquestacionnucleoapp

import (
	"context"

	orquestacorereplanner "orquesta/modulos/orquesta-core-replanner"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectorcycle "orquesta/modulos/orquesta-director-cycle"
)

type ReviewReworkReplanPlanProviderPortV0 interface {
	BuildReviewReworkReplanPlansV0(
		context.Context,
		ReviewReworkReplanPlanRequestV0,
	) ([]ReviewReworkReplanPlanV0, error)
}

type ReviewReworkReplanPlanRequestV0 struct {
	Run              orquestacoreworkflow.OrchestrationRunV0
	StepNumber       int
	MaxSteps         int
	OccurredAt       string
	PreviousStep     *orquestadirectorcycle.DirectorCycleStepResultV0
	CorrelationID    string
	EvidenceRefs     []string
	PreviousDecision string
}

type ReviewReworkReplanPlanV0 struct {
	CandidateRef               string
	ReplanRef                  string
	SignalRef                  string
	ReworkRequestRef           string
	TaskRef                    string
	ReasonRef                  string
	RequestedAction            orquestacorereplanner.ReplanRecommendedActionV0
	CapacityRequestRef         string
	AgentRequestID             string
	AskDirectorQuestionID      string
	AgentRole                  string
	MinimumRecommendedCapacity orquestacoreworkflow.OrchestrationCapacityRecommendationV0
	Summary                    string
	EvidenceRefs               []string
	ReviewResult               orquestacoreworkflow.ReviewResultV0
	SplitTasks                 []orquestacoreworkflow.WorkflowTaskV0
}
