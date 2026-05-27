package orquestadirectorrunner

import (
	"context"
	"reflect"
	"strings"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectorscheduler "orquesta/modulos/orquesta-director-scheduler"
	orquestarails "orquesta/modulos/orquesta-rails"
)

const (
	DirectorCycleMaxCommandsV0 = 20
	DirectorCycleMaxOutboxV0   = DirectorCycleMaxCommandsV0
	maxDirectorCycleRefsV0     = 40
	maxDirectorCycleStringV0   = 600
)

func validateDirectorCycleInputV0(ctx context.Context, input DirectorCycleInputV0) error {
	if ctx == nil {
		return directorCycleErrorV0(input, ErrDirectorRunnerCycleInvalidoV0, "context requerido", "context", false, nil)
	}
	if isNilDirectorCyclePortV0(input.Scheduler) {
		return directorCycleErrorV0(input, ErrDirectorRunnerCycleInvalidoV0, "scheduler requerido", "scheduler", false, nil)
	}
	if isNilDirectorCyclePortV0(input.Workflow) {
		return directorCycleErrorV0(input, ErrDirectorRunnerCycleInvalidoV0, "workflow requerido", "workflow", false, nil)
	}
	for field, value := range map[string]string{"cycle_ref": input.CycleRef, "run_ref": input.RunRef} {
		if strings.TrimSpace(value) == "" {
			return directorCycleErrorV0(input, ErrDirectorRunnerCycleInvalidoV0, "campo requerido", field, false, nil)
		}
	}
	if input.SchedulerInput.RunRef != input.RunRef {
		return directorCycleErrorV0(input, ErrDirectorRunnerCycleInvalidoV0, "run_ref no coincide", "scheduler_input.run_ref", false, nil)
	}
	if input.MaxCommands < 1 || input.MaxCommands > DirectorCycleMaxCommandsV0 {
		return directorCycleErrorV0(input, ErrDirectorRunnerCycleInvalidoV0, "max_commands fuera de rango", "max_commands", false, nil)
	}
	if input.MaxOutbox < 1 || input.MaxOutbox > DirectorCycleMaxOutboxV0 {
		return directorCycleErrorV0(input, ErrDirectorRunnerCycleInvalidoV0, "max_outbox fuera de rango", "max_outbox", false, nil)
	}
	if refsInvalidDirectorCycleV0(input.EvidenceRefs) && !orquestarails.SecurityModeProgrammingEnabledV0() {
		return directorCycleErrorV0(input, ErrDirectorRunnerCycleInvalidoV0, "refs invalidas", "evidence_refs", false, nil)
	}
	if err := orquestadirectorscheduler.ValidateDirectorSchedulerTickInputV0(input.SchedulerInput); err != nil {
		return directorCycleErrorV0(input, ErrDirectorRunnerCycleInvalidoV0, "scheduler_input invalido", "scheduler_input", false, nil)
	}
	return nil
}

func validateDirectorCyclePlanV0(
	input DirectorCycleInputV0,
	plan orquestadirectorscheduler.DirectorSchedulerTickPlanV0,
) error {
	if strings.TrimSpace(plan.TickRef) == "" {
		return directorCycleErrorV0(input, ErrDirectorRunnerCycleInvalidoV0, "plan sin tick_ref", "plan.tick_ref", false, nil)
	}
	if strings.TrimSpace(plan.RunRef) != input.RunRef {
		return directorCycleErrorV0(input, ErrDirectorRunnerCycleInvalidoV0, "plan de otro run", "plan.run_ref", false, nil)
	}
	if !directorCycleSchedulerStatusKnownV0(plan.Status) {
		return directorCycleErrorV0(input, ErrDirectorRunnerCycleInvalidoV0, "status scheduler no soportado", "plan.status", false, nil)
	}
	if len(plan.Commands) > input.MaxCommands && !orquestarails.SecurityModeProgrammingEnabledV0() {
		return directorCycleErrorV0(input, ErrDirectorRunnerCycleInvalidoV0, "demasiados comandos", "plan.commands", false, nil)
	}
	return validateDirectorCyclePlanCommandsV0(input, plan)
}

func validateDirectorCyclePlanCommandsV0(
	input DirectorCycleInputV0,
	plan orquestadirectorscheduler.DirectorSchedulerTickPlanV0,
) error {
	if len(plan.Commands) == 0 {
		if plan.Status == orquestadirectorscheduler.SchedulerTickStatusCommandsReadyV0 {
			return directorCycleErrorV0(input, ErrDirectorRunnerCycleInvalidoV0, "commands_ready sin comandos", "plan.commands", false, nil)
		}
		return nil
	}
	if !directorCycleStatusAllowsCommandsV0(plan.Status) {
		return directorCycleErrorV0(input, ErrDirectorRunnerCycleInvalidoV0, "status no permite comandos", "plan.status", false, nil)
	}
	for _, command := range plan.Commands {
		if strings.TrimSpace(command.RunID) != input.RunRef {
			return directorCycleErrorV0(input, ErrDirectorRunnerCycleInvalidoV0, "comando de otro run", "plan.commands.run_id", false, nil)
		}
		if err := orquestacoreworkflow.ValidateOrchestrationCommandV0(command); err != nil {
			return directorCycleErrorV0(input, ErrDirectorRunnerCycleInvalidoV0, "comando invalido", "plan.commands", false, nil)
		}
	}
	return nil
}

func directorCycleSchedulerStatusKnownV0(status orquestadirectorscheduler.DirectorSchedulerTickStatusV0) bool {
	switch status {
	case orquestadirectorscheduler.SchedulerTickStatusCommandsReadyV0,
		orquestadirectorscheduler.SchedulerTickStatusWaitingV0,
		orquestadirectorscheduler.SchedulerTickStatusBlockedV0,
		orquestadirectorscheduler.SchedulerTickStatusNeedsDirectorV0,
		orquestadirectorscheduler.SchedulerTickStatusQuiescentV0:
		return true
	default:
		return false
	}
}

func directorCycleStatusAllowsCommandsV0(status orquestadirectorscheduler.DirectorSchedulerTickStatusV0) bool {
	return status == orquestadirectorscheduler.SchedulerTickStatusCommandsReadyV0 ||
		status == orquestadirectorscheduler.SchedulerTickStatusBlockedV0 ||
		status == orquestadirectorscheduler.SchedulerTickStatusNeedsDirectorV0
}

func refsInvalidDirectorCycleV0(refs []string) bool {
	if len(refs) > maxDirectorCycleRefsV0 {
		return true
	}
	for _, ref := range refs {
		trimmed := strings.TrimSpace(ref)
		if trimmed == "" || len(trimmed) > maxDirectorCycleStringV0 {
			return true
		}
	}
	return false
}

func isNilDirectorCyclePortV0(port any) bool {
	if port == nil {
		return true
	}
	value := reflect.ValueOf(port)
	switch value.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return value.IsNil()
	default:
		return false
	}
}
