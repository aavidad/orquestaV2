package orquestadirectorsupervisedburst

import (
	"reflect"
	"strings"

	orquestadirectorsupervisor "orquesta/modulos/orquesta-director-supervisor"
)

func validateDirectorSupervisedBurstInputV0(input DirectorSupervisedBurstInputV0) error {
	if isNilBurstPortV0(input.StepInputBuilder) {
		return burstErrorV0(input, ErrDirectorSupervisedBurstInvalidoV0, "step_input_builder requerido", "step_input_builder", false)
	}
	if isNilBurstPortV0(input.StepExecutor) {
		return burstErrorV0(input, ErrDirectorSupervisedBurstInvalidoV0, "step_executor requerido", "step_executor", false)
	}
	if isNilBurstPortV0(input.Supervisor) {
		return burstErrorV0(input, ErrDirectorSupervisedBurstInvalidoV0, "supervisor requerido", "supervisor", false)
	}
	if strings.TrimSpace(input.RunRef) == "" {
		return burstErrorV0(input, ErrDirectorSupervisedBurstInvalidoV0, "run_ref requerido", "run_ref", false)
	}
	if input.MaxSteps < 1 {
		return burstErrorV0(input, ErrDirectorSupervisedBurstInvalidoV0, "max_steps debe ser mayor que cero", "max_steps", false)
	}
	if input.MaxSteps > orquestadirectorsupervisor.DirectorSupervisorMaxStepsLimitV0 {
		return burstErrorV0(input, ErrDirectorSupervisedBurstInvalidoV0, "max_steps supera el limite", "max_steps", false)
	}
	return nil
}

func isNilBurstPortV0(port any) bool {
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
