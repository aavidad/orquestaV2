package orquestacoreworkflow

import "strings"

func ensureRegisterFinalValidationCommandAllowedV0(current OrchestrationRunV0, command OrchestrationCommandV0, payload RegisterFinalValidationCommandPayloadV0) error {
	if err := ensureActiveRunForCommandV0(current, command); err != nil {
		return err
	}
	if err := ensureRegisterFinalValidationCommandPhaseCurrentV0(current, payload.PhaseID); err != nil {
		return err
	}
	if !taskClosedAlreadyReflectedV0(current, payload.ClosedTaskRef) {
		return commandErrorV0(ErrTransicionInvalidaV0, "payload.closed_task_ref")
	}
	return nil
}

func ensureFinalValidationRegisteredEventAllowedV0(current OrchestrationRunV0, event OrchestrationEventV0, payload FinalValidationRegisteredPayloadV0) error {
	if err := ensureRunCanApplyEventV0(current, event); err != nil {
		return err
	}
	if err := ensureFinalValidationRegisteredEventPhaseCurrentV0(current, payload.PhaseID); err != nil {
		return err
	}
	if !taskClosedAlreadyReflectedV0(current, payload.ClosedTaskRef) {
		return eventErrorV0(ErrSecuenciaInvalidaV0, "payload.closed_task_ref")
	}
	return nil
}

func ensureRegisterFinalValidationCommandPhaseCurrentV0(run OrchestrationRunV0, phaseID string) error {
	if !finalValidationPhaseCurrentV0(run, phaseID) {
		return commandErrorV0(ErrTransicionInvalidaV0, "phase.status")
	}
	return nil
}

func ensureFinalValidationRegisteredEventPhaseCurrentV0(run OrchestrationRunV0, phaseID string) error {
	if !finalValidationPhaseCurrentV0(run, phaseID) {
		return eventErrorV0(ErrSecuenciaInvalidaV0, "phase.status")
	}
	return nil
}

func finalValidationPhaseCurrentV0(run OrchestrationRunV0, phaseID string) bool {
	phase := OrchestrationPhaseIDV0(strings.TrimSpace(phaseID))
	if phase != OrchestrationPhaseValidacionFinalV0 {
		return false
	}
	if normalizePhaseIDV0(run.CurrentPhase) != phase {
		return false
	}
	return phaseIsCurrentAndActiveV0(run, phase)
}
