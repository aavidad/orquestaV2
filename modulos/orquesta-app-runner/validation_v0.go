package orquestaapprunner

import (
	"errors"
	"strings"

	orquestaappplanner "orquesta/modulos/orquesta-app-planner"
)

func validatePrepareAppOrchestrationRequestV0(
	request PrepareAppOrchestrationRequestV0,
) error {
	if strings.TrimSpace(request.RunRef) == "" {
		return AppRunnerIssueV0{Field: "run_ref"}
	}
	if strings.TrimSpace(request.ProjectRef) == "" {
		return AppRunnerIssueV0{Field: "project_ref"}
	}
	if strings.TrimSpace(request.OccurredAt) == "" {
		return AppRunnerIssueV0{Field: "occurred_at"}
	}
	if _, err := orquestaappplanner.AppPlanRequestFromAppSpecV0(request.RunRef, request.AppSpec); err != nil {
		return AppRunnerIssueV0{Field: appRunnerPlannerFieldV0(err)}
	}
	return nil
}

func appRunnerPlannerFieldV0(err error) string {
	var plannerIssue orquestaappplanner.AppPlannerIssueV0
	if errors.As(err, &plannerIssue) && strings.TrimSpace(plannerIssue.Field) != "" {
		if strings.HasPrefix(plannerIssue.Field, "app_spec.") {
			return plannerIssue.Field
		}
		return "app_spec." + plannerIssue.Field
	}
	return "app_spec"
}
