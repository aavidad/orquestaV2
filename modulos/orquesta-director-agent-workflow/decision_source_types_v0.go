package orquestadirectoragentworkflow

import (
	"context"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectoragent "orquesta/modulos/orquesta-director-agent"
)

type DirectorAgentDecisionSourceRequestV0 struct {
	Run            orquestacoreworkflow.OrchestrationRunV0
	Stats          orquestadirectoragent.DirectorAgentCompactStatsV0
	RequestKind    string
	ExecutionMode  string
	ObjectiveHints []string
	OccurredAt     string
	CorrelationID  string
	RequestedBy    string
}

type DirectorAgentDecisionSourcePortV0 interface {
	ListDirectorAgentDecisionsV0(
		ctx context.Context,
		request DirectorAgentDecisionSourceRequestV0,
	) ([]orquestadirectoragent.DirectorAgentDecisionV0, error)
}
