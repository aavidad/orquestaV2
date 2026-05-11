package orquestadirectoragentworkflow

import orquestadirectoragent "orquesta/modulos/orquesta-director-agent"

type DirectorAgentWorkflowCommandRequestV0 struct {
	Decision      orquestadirectoragent.DirectorAgentDecisionV0
	OccurredAt    string
	CorrelationID string
	RequestedBy   string
}

type DirectorAgentWorkflowIssueV0 struct {
	Code  string
	Field string
}
