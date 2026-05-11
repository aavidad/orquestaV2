package orquestadirectorrunner

import (
	"errors"
	"strings"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
)

func resultWithDirectorCycleWorkflowErrorV0(
	result DirectorCycleResultV0,
	input DirectorCycleInputV0,
	command orquestacoreworkflow.OrchestrationCommandV0,
	err error,
) (DirectorCycleResultV0, error) {
	issues := []DirectorCycleIssueV0{
		directorCycleIssueV0(ErrDirectorRunnerWorkflowV0, "workflow", "workflow fallo"),
	}
	var commandErr orquestacoreworkflow.OrchestrationCommandErrorV0
	if errors.As(err, &commandErr) {
		issues = append(issues, DirectorCycleIssueV0{
			Code:    commandErr.Code,
			Field:   directorCycleWorkflowFieldV0(command, commandErr.Field),
			Message: "workflow comando rechazado",
		})
	} else if err != nil {
		issues = append(issues, DirectorCycleIssueV0{
			Code:    ErrDirectorRunnerWorkflowV0,
			Field:   directorCycleWorkflowFieldV0(command, ""),
			Message: directorCycleWorkflowMessageV0(err),
		})
	}
	result.Issues = compactDirectorCycleIssuesV0(append(result.Issues, issues...))
	return result, directorCycleErrorV0(
		input,
		ErrDirectorRunnerWorkflowV0,
		"workflow fallo",
		"workflow",
		true,
		result.Issues,
	)
}

func directorCycleWorkflowFieldV0(
	command orquestacoreworkflow.OrchestrationCommandV0,
	field string,
) string {
	parts := []string{"workflow", strings.TrimSpace(command.CommandType)}
	if trimmed := strings.TrimSpace(field); trimmed != "" {
		parts = append(parts, trimmed)
	}
	return strings.Join(parts, ".")
}

func directorCycleWorkflowMessageV0(err error) string {
	if err == nil {
		return "workflow error no tipado"
	}
	message := strings.TrimSpace(err.Error())
	if message == "" {
		return "workflow error no tipado"
	}
	if len(message) > 180 {
		return message[:180]
	}
	return message
}

func compactDirectorCycleIssuesV0(issues []DirectorCycleIssueV0) []DirectorCycleIssueV0 {
	seen := map[DirectorCycleIssueV0]bool{}
	out := make([]DirectorCycleIssueV0, 0, len(issues))
	for _, issue := range issues {
		if issue.Code == "" || seen[issue] {
			continue
		}
		seen[issue] = true
		out = append(out, issue)
	}
	return out
}
