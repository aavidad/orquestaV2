package orquestacoreworkflow

import "strings"

func ensureRecordConcurrencyGateCommandAllowedV0(current OrchestrationRunV0, command OrchestrationCommandV0, payload RecordConcurrencyGateCommandPayloadV0) error {
	if err := ensureActiveRunForCommandV0(current, command); err != nil {
		return err
	}
	if !concurrencyGateRunMatchesV0(current, payload.RunRef) {
		return commandErrorV0(ErrTransicionInvalidaV0, "payload.run_ref")
	}
	return nil
}

func ensureConcurrencyGateRecordedEventAllowedV0(current OrchestrationRunV0, event OrchestrationEventV0, payload ConcurrencyGateRecordedPayloadV0) error {
	if err := ensureRunCanApplyEventV0(current, event); err != nil {
		return err
	}
	if !concurrencyGateRunMatchesV0(current, payload.RunRef) {
		return eventErrorV0(ErrSecuenciaInvalidaV0, "payload.run_ref")
	}
	return nil
}

func concurrencyGateRunMatchesV0(current OrchestrationRunV0, runRef string) bool {
	return strings.TrimSpace(current.RunID) == strings.TrimSpace(runRef)
}
