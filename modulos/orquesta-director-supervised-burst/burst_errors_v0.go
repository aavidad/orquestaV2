package orquestadirectorsupervisedburst

func burstErrorV0(
	input DirectorSupervisedBurstInputV0,
	code string,
	message string,
	field string,
	retryable bool,
) DirectorSupervisedBurstErrorV0 {
	issue := DirectorSupervisedBurstIssueV0{
		Code:    code,
		Field:   field,
		Message: message,
	}
	return DirectorSupervisedBurstErrorV0{
		Code:          code,
		Message:       message,
		Field:         field,
		Retryable:     retryable,
		Issues:        []DirectorSupervisedBurstIssueV0{issue},
		CorrelationID: input.CorrelationID,
	}
}

func resultWithBurstErrorV0(
	result DirectorSupervisedBurstResultV0,
	err DirectorSupervisedBurstErrorV0,
) (DirectorSupervisedBurstResultV0, error) {
	result.Issues = append(result.Issues, err.Issues...)
	return result, err
}
