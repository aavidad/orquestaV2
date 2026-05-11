package orquestadirectorcycle

import (
	"context"
	"reflect"
	"strings"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
)

func validateDirectorCycleStepInputV0(ctx context.Context, input DirectorCycleStepInputV0) error {
	if ctx == nil {
		return cycleStepErrorV0(input, ErrDirectorCycleStepInvalidoV0, "context requerido", "context", false, nil)
	}
	if isNilCycleStepPortV0(input.Scheduler) {
		return cycleStepErrorV0(input, ErrDirectorCycleStepInvalidoV0, "scheduler requerido", "scheduler", false, nil)
	}
	if isNilCycleStepPortV0(input.Workflow) {
		return cycleStepErrorV0(input, ErrDirectorCycleStepInvalidoV0, "workflow requerido", "workflow", false, nil)
	}
	if isNilCycleStepPortV0(input.OutboxLedger) {
		return cycleStepErrorV0(input, ErrDirectorCycleStepInvalidoV0, "outbox_ledger requerido", "outbox_ledger", false, nil)
	}
	if err := validateCycleStepRequiredV0(input); err != nil {
		return err
	}
	if issues := orquestacoreworkflow.ValidateOrchestrationRunV0(input.Run); len(issues) > 0 {
		return cycleStepErrorV0(input, ErrDirectorCycleStepInvalidoV0, "run invalido", "run."+issues[0].Field, false, nil)
	}
	return nil
}

func validateCycleStepRequiredV0(input DirectorCycleStepInputV0) error {
	fields := map[string]string{
		"cycle_ref":   input.CycleRef,
		"tick_ref":    input.TickRef,
		"run_ref":     input.RunRef,
		"occurred_at": input.OccurredAt,
		"run.run_id":  input.Run.RunID,
	}
	for field, value := range fields {
		if strings.TrimSpace(value) == "" {
			return cycleStepErrorV0(input, ErrDirectorCycleStepInvalidoV0, "campo requerido", field, false, nil)
		}
	}
	if strings.TrimSpace(input.Run.RunID) != input.RunRef {
		return cycleStepErrorV0(input, ErrDirectorCycleStepInvalidoV0, "run_ref no coincide", "run.run_id", false, nil)
	}
	return nil
}

func isNilCycleStepPortV0(port any) bool {
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
