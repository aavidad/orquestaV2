package orquestacoreworkflow

func ensureRequestReworkCommandAllowedV0(current OrchestrationRunV0, command OrchestrationCommandV0, payload RequestReworkCommandPayloadV0) error {
	if err := ensureActiveRunForCommandV0(current, command); err != nil {
		return err
	}
	if err := ensureRequestReworkCommandPhaseCurrentV0(current, payload.PhaseID); err != nil {
		return err
	}
	if !reviewRequestAlreadyReflectedV0(current, payload.ReviewRequestID) {
		return commandErrorV0(ErrTransicionInvalidaV0, "payload.review_request_id")
	}
	if !deliveryAlreadyReflectedV0(current, payload.DeliveryRef) {
		return commandErrorV0(ErrTransicionInvalidaV0, "payload.delivery_ref")
	}
	if !reviewResultSupportsReworkV0(current, payload.ReviewResultRef, payload.ReviewRequestID, payload.DeliveryRef) {
		return commandErrorV0(ErrTransicionInvalidaV0, "payload.review_result_ref")
	}
	return nil
}

func ensureReworkRequestedEventAllowedV0(current OrchestrationRunV0, event OrchestrationEventV0, payload ReworkRequestedPayloadV0) error {
	if err := ensureRunCanApplyEventV0(current, event); err != nil {
		return err
	}
	if err := ensureReworkRequestedEventPhaseCurrentV0(current, payload.PhaseID); err != nil {
		return err
	}
	if !reviewRequestAlreadyReflectedV0(current, payload.ReviewRequestID) {
		return eventErrorV0(ErrSecuenciaInvalidaV0, "payload.review_request_id")
	}
	if !deliveryAlreadyReflectedV0(current, payload.DeliveryRef) {
		return eventErrorV0(ErrSecuenciaInvalidaV0, "payload.delivery_ref")
	}
	if !reviewResultSupportsReworkV0(current, payload.ReviewResultRef, payload.ReviewRequestID, payload.DeliveryRef) {
		return eventErrorV0(ErrSecuenciaInvalidaV0, "payload.review_result_ref")
	}
	return nil
}

func ensureRequestReworkCommandPhaseCurrentV0(run OrchestrationRunV0, phaseID string) error {
	if !reviewPhaseCurrentV0(run, phaseID) {
		return commandErrorV0(ErrTransicionInvalidaV0, "phase.status")
	}
	return nil
}

func ensureReworkRequestedEventPhaseCurrentV0(run OrchestrationRunV0, phaseID string) error {
	if !reviewPhaseCurrentV0(run, phaseID) {
		return eventErrorV0(ErrSecuenciaInvalidaV0, "phase.status")
	}
	return nil
}
