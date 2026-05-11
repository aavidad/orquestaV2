package orquestacoreworkflow

import "strings"

func ensureRequestReviewCommandAllowedV0(current OrchestrationRunV0, command OrchestrationCommandV0, payload RequestReviewCommandPayloadV0) error {
	if err := ensureActiveRunForCommandV0(current, command); err != nil {
		return err
	}
	if err := ensureRequestReviewCommandPhaseCurrentV0(current, payload.PhaseID); err != nil {
		return err
	}
	if !deliveryAlreadyReflectedV0(current, payload.DeliveryRef) {
		return commandErrorV0(ErrTransicionInvalidaV0, "payload.delivery_ref")
	}
	return nil
}

func ensureReviewRequestedEventAllowedV0(current OrchestrationRunV0, event OrchestrationEventV0, payload ReviewRequestedPayloadV0) error {
	if err := ensureRunCanApplyEventV0(current, event); err != nil {
		return err
	}
	if err := ensureReviewRequestedEventPhaseCurrentV0(current, payload.PhaseID); err != nil {
		return err
	}
	if !deliveryAlreadyReflectedV0(current, payload.DeliveryRef) {
		return eventErrorV0(ErrSecuenciaInvalidaV0, "payload.delivery_ref")
	}
	return nil
}

func ensureRequestReviewCommandPhaseCurrentV0(run OrchestrationRunV0, phaseID string) error {
	if !reviewPhaseCurrentV0(run, phaseID) {
		return commandErrorV0(ErrTransicionInvalidaV0, "phase.status")
	}
	return nil
}

func ensureReviewRequestedEventPhaseCurrentV0(run OrchestrationRunV0, phaseID string) error {
	if !reviewPhaseCurrentV0(run, phaseID) {
		return eventErrorV0(ErrSecuenciaInvalidaV0, "phase.status")
	}
	return nil
}

func reviewPhaseCurrentV0(run OrchestrationRunV0, phaseID string) bool {
	phase := OrchestrationPhaseIDV0(strings.TrimSpace(phaseID))
	if phase != OrchestrationPhaseRevisionV0 {
		return false
	}
	if normalizePhaseIDV0(run.CurrentPhase) != phase {
		return false
	}
	return phaseIsCurrentAndActiveV0(run, phase)
}
