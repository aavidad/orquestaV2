package orquestadirectorcycleoutbox

import (
	"context"
	"reflect"
	"strings"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
)

func validateDirectorCycleOutboxInputV0(
	ctx context.Context,
	input DirectorCycleOutboxRecordInputV0,
) error {
	if ctx == nil {
		return cycleOutboxErrorV0(input, ErrDirectorCycleOutboxInvalidoV0, "context requerido", "context", false, nil)
	}
	if isNilCycleOutboxPortV0(input.Ledger) {
		return cycleOutboxErrorV0(input, ErrDirectorCycleOutboxInvalidoV0, "ledger requerido", "ledger", false, nil)
	}
	if strings.TrimSpace(input.RunRef) == "" {
		return cycleOutboxErrorV0(input, ErrDirectorCycleOutboxInvalidoV0, "run_ref requerido", "run_ref", false, nil)
	}
	for _, message := range input.Messages {
		if err := orquestacoreworkflow.ValidateOutboxMessageV0(message); err != nil {
			return cycleOutboxErrorV0(input, ErrDirectorCycleOutboxInvalidoV0, "outbox invalido", "messages", false, nil)
		}
		if strings.TrimSpace(message.RunID) != input.RunRef {
			return cycleOutboxErrorV0(input, ErrDirectorCycleOutboxInvalidoV0, "outbox de otro run", "messages.run_id", false, nil)
		}
		if input.TargetPort != "" && strings.TrimSpace(message.TargetPort) != input.TargetPort {
			return cycleOutboxErrorV0(input, ErrDirectorCycleOutboxInvalidoV0, "target_port no coincide", "messages.target_port", false, nil)
		}
	}
	return nil
}

func isNilCycleOutboxPortV0(port any) bool {
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
