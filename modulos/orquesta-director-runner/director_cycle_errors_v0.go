package orquestadirectorrunner

import (
	"errors"
	"strings"

	orquestadirectorscheduler "orquesta/modulos/orquesta-director-scheduler"
)

func directorCycleIssueV0(code string, field string, message string) DirectorCycleIssueV0 {
	return DirectorCycleIssueV0{Code: code, Field: field, Message: message}
}

func directorCycleErrorV0(
	input DirectorCycleInputV0,
	code string,
	message string,
	field string,
	retryable bool,
	issues []DirectorCycleIssueV0,
) DirectorCycleErrorV0 {
	if len(issues) == 0 {
		issues = []DirectorCycleIssueV0{directorCycleIssueV0(code, field, message)}
	}
	return DirectorCycleErrorV0{
		Code:          code,
		Message:       message,
		Field:         field,
		Retryable:     retryable,
		Issues:        append([]DirectorCycleIssueV0(nil), issues...),
		CorrelationID: input.CorrelationID,
	}
}

func resultWithDirectorCycleErrorV0(
	result DirectorCycleResultV0,
	err error,
) (DirectorCycleResultV0, error) {
	cycleErr, ok := err.(DirectorCycleErrorV0)
	if !ok {
		return result, err
	}
	result.Issues = append(result.Issues, cycleErr.Issues...)
	return result, cycleErr
}

func resultWithDirectorCycleIssueV0(
	result DirectorCycleResultV0,
	input DirectorCycleInputV0,
	code string,
	field string,
	message string,
	retryable bool,
) (DirectorCycleResultV0, error) {
	issue := directorCycleIssueV0(code, field, message)
	result.Issues = append(result.Issues, issue)
	return result, directorCycleErrorV0(input, code, message, field, retryable, result.Issues)
}

func resultWithDirectorCycleSchedulerErrorV0(
	result DirectorCycleResultV0,
	input DirectorCycleInputV0,
	err error,
) (DirectorCycleResultV0, error) {
	field := "scheduler"
	message := "scheduler fallo"
	var schedulerErr orquestadirectorscheduler.DirectorSchedulerTickErrorV0
	if errors.As(err, &schedulerErr) {
		if detail := strings.TrimSpace(schedulerErr.Field); detail != "" {
			field = "scheduler." + detail
			message = "scheduler fallo: " + strings.TrimSpace(schedulerErr.Code) + ":" + detail
		}
	}
	return resultWithDirectorCycleIssueV0(result, input, ErrDirectorRunnerSchedulerV0, field, message, true)
}
