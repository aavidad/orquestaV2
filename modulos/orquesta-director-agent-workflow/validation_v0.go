package orquestadirectoragentworkflow

import orquestadirectoragent "orquesta/modulos/orquesta-director-agent"

func validateDirectorAgentWorkflowRequestV0(
	request DirectorAgentWorkflowCommandRequestV0,
) []DirectorAgentWorkflowIssueV0 {
	issues := make([]DirectorAgentWorkflowIssueV0, 0)
	for _, issue := range orquestadirectoragent.ValidateDirectorAgentDecisionV0(request.Decision) {
		issues = append(issues, DirectorAgentWorkflowIssueV0{
			Code:  issue.Code,
			Field: "decision." + issue.Field,
		})
	}
	if request.OccurredAt == "" {
		issues = append(issues, directorAgentWorkflowIssueV0("director_agent_workflow_required", "occurred_at"))
	}
	return issues
}
