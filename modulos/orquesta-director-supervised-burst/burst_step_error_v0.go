package orquestadirectorsupervisedburst

import (
	"errors"

	orquestadirectorcycle "orquesta/modulos/orquesta-director-cycle"
)

func burstStepErrorV0(
	input DirectorSupervisedBurstInputV0,
	stepResult orquestadirectorcycle.DirectorCycleStepResultV0,
	err error,
) DirectorSupervisedBurstErrorV0 {
	issues := []DirectorSupervisedBurstIssueV0{{
		Code:    ErrDirectorSupervisedBurstStepV0,
		Field:   "step",
		Message: "paso del ciclo fallo",
	}}
	issues = append(issues, burstIssuesFromCycleStepV0(stepResult.Issues)...)
	var stepErr orquestadirectorcycle.DirectorCycleStepErrorV0
	if errors.As(err, &stepErr) {
		issues = append(issues, burstIssuesFromCycleStepV0(stepErr.Issues)...)
	}
	issues = compactBurstIssuesV0(issues)
	return DirectorSupervisedBurstErrorV0{
		Code:          ErrDirectorSupervisedBurstStepV0,
		Message:       "paso del ciclo fallo",
		Field:         "step",
		Retryable:     true,
		Issues:        issues,
		CorrelationID: input.CorrelationID,
	}
}

func burstIssuesFromCycleStepV0(
	issues []orquestadirectorcycle.DirectorCycleStepIssueV0,
) []DirectorSupervisedBurstIssueV0 {
	if len(issues) == 0 {
		return nil
	}
	out := make([]DirectorSupervisedBurstIssueV0, 0, len(issues))
	for _, issue := range issues {
		out = append(out, DirectorSupervisedBurstIssueV0{
			Code:    issue.Code,
			Field:   "step." + issue.Field,
			Message: issue.Message,
		})
	}
	return out
}

func compactBurstIssuesV0(
	issues []DirectorSupervisedBurstIssueV0,
) []DirectorSupervisedBurstIssueV0 {
	seen := map[DirectorSupervisedBurstIssueV0]bool{}
	out := make([]DirectorSupervisedBurstIssueV0, 0, len(issues))
	for _, issue := range issues {
		if issue.Code == "" || seen[issue] {
			continue
		}
		seen[issue] = true
		out = append(out, issue)
	}
	return out
}
