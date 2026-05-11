package orquestadirectorsupervisor

func supervisorErrorV0(
	input DirectorSupervisorDecisionInputV0,
	message string,
	field string,
) DirectorSupervisorErrorV0 {
	issue := DirectorSupervisorIssueV0{
		Code:    ErrDirectorSupervisorDecisionInvalidaV0,
		Field:   field,
		Message: message,
	}
	return DirectorSupervisorErrorV0{
		Code:          ErrDirectorSupervisorDecisionInvalidaV0,
		Message:       message,
		Field:         field,
		Retryable:     false,
		Issues:        []DirectorSupervisorIssueV0{issue},
		CorrelationID: input.CorrelationID,
	}
}

func resultWithSupervisorErrorV0(
	result DirectorSupervisorDecisionV0,
	err error,
) (DirectorSupervisorDecisionV0, error) {
	if publicErr, ok := err.(DirectorSupervisorErrorV0); ok {
		result.Issues = append(result.Issues, publicErr.Issues...)
	}
	return result, err
}
