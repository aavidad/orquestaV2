package orquestadirectorcycle

import "errors"

func cycleStepsIssueV0(code string, field string, message string) DirectorCycleStepsIssueV0 {
	return DirectorCycleStepsIssueV0{Code: code, Field: field, Message: message}
}

func cycleStepsErrorV0(
	input DirectorCycleStepsInputV0,
	code string,
	message string,
	field string,
	retryable bool,
	issues []DirectorCycleStepsIssueV0,
) DirectorCycleStepsErrorV0 {
	if len(issues) == 0 {
		issues = []DirectorCycleStepsIssueV0{cycleStepsIssueV0(code, field, message)}
	}
	return DirectorCycleStepsErrorV0{
		Code:          code,
		Message:       message,
		Field:         field,
		Retryable:     retryable,
		Issues:        append([]DirectorCycleStepsIssueV0(nil), issues...),
		CorrelationID: input.CorrelationID,
	}
}

func resultWithCycleStepsErrorV0(
	result DirectorCycleStepsResultV0,
	err error,
) (DirectorCycleStepsResultV0, error) {
	var publicErr DirectorCycleStepsErrorV0
	if errors.As(err, &publicErr) {
		result.LastErrorCode = publicErr.Code
		result.Issues = compactCycleStepsIssuesV0(append(result.Issues, publicErr.Issues...))
		return result, publicErr
	}
	var stepErr DirectorCycleStepErrorV0
	if errors.As(err, &stepErr) {
		result.LastErrorCode = stepErr.Code
		issues := cycleStepsIssuesFromStepV0(stepErr.Issues)
		if len(issues) == 0 {
			issues = []DirectorCycleStepsIssueV0{
				cycleStepsIssueV0(stepErr.Code, "step."+stepErr.Field, stepErr.Message),
			}
		}
		result.Issues = compactCycleStepsIssuesV0(append(result.Issues, issues...))
		return result, cycleStepsErrorV0(
			DirectorCycleStepsInputV0{CorrelationID: result.CorrelationID},
			ErrDirectorCycleStepsStepV0,
			"step fallo",
			"step",
			stepErr.Retryable,
			result.Issues,
		)
	}
	return result, err
}

func cycleStepsIssuesFromStepV0(
	issues []DirectorCycleStepIssueV0,
) []DirectorCycleStepsIssueV0 {
	if len(issues) == 0 {
		return nil
	}
	out := make([]DirectorCycleStepsIssueV0, 0, len(issues))
	for _, issue := range issues {
		out = append(out, cycleStepsIssueV0(issue.Code, "step."+issue.Field, issue.Message))
	}
	return out
}

func compactCycleStepsIssuesV0(
	issues []DirectorCycleStepsIssueV0,
) []DirectorCycleStepsIssueV0 {
	seen := map[DirectorCycleStepsIssueV0]bool{}
	out := make([]DirectorCycleStepsIssueV0, 0, len(issues))
	for _, issue := range issues {
		if issue.Code == "" || seen[issue] {
			continue
		}
		seen[issue] = true
		out = append(out, issue)
	}
	return out
}
