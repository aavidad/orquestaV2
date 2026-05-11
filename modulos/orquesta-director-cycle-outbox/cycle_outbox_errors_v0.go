package orquestadirectorcycleoutbox

func cycleOutboxIssueV0(code string, field string, message string) DirectorCycleOutboxIssueV0 {
	return DirectorCycleOutboxIssueV0{Code: code, Field: field, Message: message}
}

func cycleOutboxErrorV0(
	input DirectorCycleOutboxRecordInputV0,
	code string,
	message string,
	field string,
	retryable bool,
	issues []DirectorCycleOutboxIssueV0,
) DirectorCycleOutboxErrorV0 {
	if len(issues) == 0 {
		issues = []DirectorCycleOutboxIssueV0{cycleOutboxIssueV0(code, field, message)}
	}
	return DirectorCycleOutboxErrorV0{
		Code:          code,
		Message:       message,
		Field:         field,
		Retryable:     retryable,
		Issues:        append([]DirectorCycleOutboxIssueV0(nil), issues...),
		CorrelationID: input.CorrelationID,
	}
}

func resultWithCycleOutboxErrorV0(
	result DirectorCycleOutboxRecordResultV0,
	err error,
) (DirectorCycleOutboxRecordResultV0, error) {
	publicErr, ok := err.(DirectorCycleOutboxErrorV0)
	if !ok {
		return result, err
	}
	result.Issues = append(result.Issues, publicErr.Issues...)
	return result, publicErr
}

func resultWithCycleOutboxIssueV0(
	result DirectorCycleOutboxRecordResultV0,
	input DirectorCycleOutboxRecordInputV0,
	code string,
	field string,
	message string,
	retryable bool,
) (DirectorCycleOutboxRecordResultV0, error) {
	issue := cycleOutboxIssueV0(code, field, message)
	result.Issues = append(result.Issues, issue)
	return result, cycleOutboxErrorV0(input, code, message, field, retryable, result.Issues)
}
