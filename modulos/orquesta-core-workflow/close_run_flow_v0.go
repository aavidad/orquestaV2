package orquestacoreworkflow

import "strings"

func ensureCloseRunCommandAllowedV0(current OrchestrationRunV0, command OrchestrationCommandV0, payload CloseRunCommandPayloadV0) error {
	if err := ensureActiveRunForCommandV0(current, command); err != nil {
		return err
	}
	if err := ensureCloseRunCommandPhaseCurrentV0(current, payload.PhaseID); err != nil {
		return err
	}
	if !finalValidationAlreadyReflectedV0(current, payload.ValidationRef) {
		return commandErrorV0(ErrTransicionInvalidaV0, "payload.validation_ref")
	}
	return nil
}

func ensureRunClosedEventAllowedV0(current OrchestrationRunV0, event OrchestrationEventV0, payload RunClosedPayloadV0) error {
	if err := ensureRunCanApplyEventV0(current, event); err != nil {
		return err
	}
	if err := ensureRunClosedEventPhaseCurrentV0(current, payload.PhaseID); err != nil {
		return err
	}
	if !finalValidationAlreadyReflectedV0(current, payload.ValidationRef) {
		return eventErrorV0(ErrSecuenciaInvalidaV0, "payload.validation_ref")
	}
	return nil
}

func ensureCloseRunCommandPhaseCurrentV0(run OrchestrationRunV0, phaseID string) error {
	if !closurePhaseCurrentV0(run, phaseID) {
		return commandErrorV0(ErrTransicionInvalidaV0, "phase.status")
	}
	return nil
}

func ensureRunClosedEventPhaseCurrentV0(run OrchestrationRunV0, phaseID string) error {
	if !closurePhaseCurrentV0(run, phaseID) {
		return eventErrorV0(ErrSecuenciaInvalidaV0, "phase.status")
	}
	return nil
}

func closurePhaseCurrentV0(run OrchestrationRunV0, phaseID string) bool {
	phase := OrchestrationPhaseIDV0(strings.TrimSpace(phaseID))
	if phase != OrchestrationPhaseCierreV0 {
		return false
	}
	if normalizePhaseIDV0(run.CurrentPhase) != phase {
		return false
	}
	return phaseIsCurrentAndActiveV0(run, phase)
}
