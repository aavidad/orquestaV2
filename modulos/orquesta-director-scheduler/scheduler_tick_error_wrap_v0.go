package orquestadirectorscheduler

import (
	"errors"
	"strings"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirector "orquesta/modulos/orquesta-director"
)

func schedulerTickWrappedExternalErrorV0(prefix string, err error) error {
	if err == nil {
		return nil
	}
	field := strings.TrimSpace(prefix)
	var schedulerErr DirectorSchedulerTickErrorV0
	if errors.As(err, &schedulerErr) {
		return schedulerTickErrorV0(schedulerTickJoinFieldV0(field, schedulerErr.Field))
	}
	var replanErr orquestadirector.ReplanFollowupsErrorV0
	if errors.As(err, &replanErr) {
		return schedulerTickErrorV0(schedulerTickJoinFieldV0(field, replanErr.Field))
	}
	var progressErr orquestadirector.AgentProgressSupervisionErrorV0
	if errors.As(err, &progressErr) {
		return schedulerTickErrorV0(schedulerTickJoinFieldV0(field, progressErr.Field))
	}
	var commandErr orquestacoreworkflow.OrchestrationCommandErrorV0
	if errors.As(err, &commandErr) {
		return schedulerTickErrorV0(schedulerTickJoinFieldV0(field, commandErr.Field))
	}
	return schedulerTickErrorV0(field)
}

func schedulerTickJoinFieldV0(prefix string, field string) string {
	prefix = strings.Trim(strings.TrimSpace(prefix), ".")
	field = strings.Trim(strings.TrimSpace(field), ".")
	switch {
	case prefix == "":
		return field
	case field == "":
		return prefix
	default:
		return prefix + "." + field
	}
}
