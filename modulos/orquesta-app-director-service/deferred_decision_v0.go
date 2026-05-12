package orquestaappdirectorservice

import (
	"errors"
	"strings"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectoragent "orquesta/modulos/orquesta-director-agent"
)

func startAppDirectorDecisionDeferredV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	decision orquestadirectoragent.DirectorAgentDecisionV0,
) bool {
	if decision.CommandType != orquestadirectoragent.DirectorAgentCommandOpenPhaseV0 ||
		decision.OpenPhase == nil {
		return false
	}
	next := orquestacoreworkflow.OrchestrationPhaseIDV0(decision.OpenPhase.PhaseID)
	switch next {
	case orquestacoreworkflow.OrchestrationPhaseProgramacionV0:
		return !startAppDirectorProgrammingTasksReadyV0(run)
	case orquestacoreworkflow.OrchestrationPhaseRevisionV0:
		return !startAppDirectorProgrammingDeliveriesReadyV0(run)
	default:
		return false
	}
}

func startAppDirectorProgrammingTasksReadyV0(
	run orquestacoreworkflow.OrchestrationRunV0,
) bool {
	return len(compactServiceRefsV0(run.Tasks)) > 0
}

func startAppDirectorProgrammingDeliveriesReadyV0(
	run orquestacoreworkflow.OrchestrationRunV0,
) bool {
	if run.CurrentPhase != orquestacoreworkflow.OrchestrationPhaseProgramacionV0 {
		return false
	}
	tasks := compactServiceRefsV0(run.Tasks)
	deliveries := compactServiceRefsV0(run.Deliveries)
	return len(tasks) > 0 && len(deliveries) >= len(tasks)
}

func startAppDirectorDecisionTransitionPendingV0(err error) bool {
	var commandErr orquestacoreworkflow.OrchestrationCommandErrorV0
	return errors.As(err, &commandErr) &&
		commandErr.Code == orquestacoreworkflow.ErrTransicionInvalidaV0
}

func compactServiceRefsV0(values []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	return out
}
