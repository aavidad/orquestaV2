package orquestadirectoragentworkflow

import "strings"

func normalizeDirectorAgentWorkflowRequestV0(
	request DirectorAgentWorkflowCommandRequestV0,
) DirectorAgentWorkflowCommandRequestV0 {
	request.OccurredAt = strings.TrimSpace(request.OccurredAt)
	request.CorrelationID = strings.TrimSpace(request.CorrelationID)
	request.RequestedBy = strings.TrimSpace(request.RequestedBy)
	if request.CorrelationID == "" {
		request.CorrelationID = strings.TrimSpace(request.Decision.DecisionRef)
	}
	if request.RequestedBy == "" {
		request.RequestedBy = "orquesta-director-agent-workflow"
	}
	return request
}

func directorAgentWorkflowIssueV0(code string, field string) DirectorAgentWorkflowIssueV0 {
	return DirectorAgentWorkflowIssueV0{Code: code, Field: field}
}
