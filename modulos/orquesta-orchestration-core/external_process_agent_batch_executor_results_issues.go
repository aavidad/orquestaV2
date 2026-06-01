package orquestacionnucleoapp

import (
	"strings"

	orquestaoutboxdispatch "orquesta/modulos/orquesta-outbox-dispatch"
	orquestaruntime "orquesta/modulos/orquesta-runtime"
)

func externalProcessBatchDispatchIssuesV0(
	started orquestaruntime.ExternalAgentProcessLaunchResultV0,
) []orquestaoutboxdispatch.DispatchIssueV0 {
	if len(started.Issues) == 0 {
		return []orquestaoutboxdispatch.DispatchIssueV0{{
			Code:    "external_agent_launch_blocked",
			Field:   "external_agent_process",
			Message: string(started.Status),
		}}
	}
	issues := make([]orquestaoutboxdispatch.DispatchIssueV0, 0, len(started.Issues))
	for _, issue := range started.Issues {
		code := strings.TrimSpace(string(issue.Code))
		if code == "" {
			code = "external_agent_launch_blocked"
		}
		field := strings.TrimSpace(issue.Field)
		if field == "" {
			field = "external_agent_process"
		}
		message := strings.TrimSpace(issue.MessageKey)
		if message == "" {
			message = "external_agent_launch_blocked"
		}
		issues = append(issues, orquestaoutboxdispatch.DispatchIssueV0{
			Code:    code,
			Field:   field,
			Message: message,
		})
	}
	return issues
}

func externalProcessBatchRetryableV0(
	issues []orquestaruntime.ExternalAgentConnectorErrorV0,
) bool {
	for _, issue := range issues {
		if issue.Retryable {
			return true
		}
	}
	return false
}
