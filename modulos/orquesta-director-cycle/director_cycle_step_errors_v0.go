package orquestadirectorcycle

func cycleStepIssueV0(code string, field string, message string) DirectorCycleStepIssueV0 {
	return DirectorCycleStepIssueV0{Code: code, Field: field, Message: message}
}

func cycleStepErrorV0(
	input DirectorCycleStepInputV0,
	code string,
	message string,
	field string,
	retryable bool,
	issues []DirectorCycleStepIssueV0,
) DirectorCycleStepErrorV0 {
	if len(issues) == 0 {
		issues = []DirectorCycleStepIssueV0{cycleStepIssueV0(code, field, message)}
	}
	return DirectorCycleStepErrorV0{
		Code:          code,
		Message:       message,
		Field:         field,
		Retryable:     retryable,
		Issues:        append([]DirectorCycleStepIssueV0(nil), issues...),
		CorrelationID: input.CorrelationID,
	}
}

func resultWithCycleStepErrorV0(
	result DirectorCycleStepResultV0,
	err error,
) (DirectorCycleStepResultV0, error) {
	publicErr, ok := err.(DirectorCycleStepErrorV0)
	if !ok {
		return result, err
	}
	result.Issues = append(result.Issues, publicErr.Issues...)
	return result, publicErr
}

func resultWithCycleStepIssueV0(
	result DirectorCycleStepResultV0,
	input DirectorCycleStepInputV0,
	code string,
	field string,
	message string,
	retryable bool,
) (DirectorCycleStepResultV0, error) {
	issue := cycleStepIssueV0(code, field, message)
	result.Issues = append(result.Issues, issue)
	return result, cycleStepErrorV0(input, code, message, field, retryable, result.Issues)
}
