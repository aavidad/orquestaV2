package orquestacoreworkflow

func ensureAcceptReviewCommandAllowedV0(current OrchestrationRunV0, command OrchestrationCommandV0, payload AcceptReviewCommandPayloadV0) error {
	if err := ensureActiveRunForCommandV0(current, command); err != nil {
		return err
	}
	if err := ensureAcceptReviewCommandPhaseCurrentV0(current, payload.PhaseID); err != nil {
		return err
	}
	if !reviewRequestAlreadyReflectedV0(current, payload.ReviewRequestID) {
		return commandErrorV0(ErrTransicionInvalidaV0, "payload.review_request_id")
	}
	if !deliveryAlreadyReflectedV0(current, payload.DeliveryRef) {
		return commandErrorV0(ErrTransicionInvalidaV0, "payload.delivery_ref")
	}
	if !acceptedReviewResultAlreadyReflectedV0(current, payload.ReviewRequestID, payload.DeliveryRef) {
		return commandErrorV0(ErrTransicionInvalidaV0, "payload.review_result_ref")
	}
	return nil
}

func ensureReviewAcceptedEventAllowedV0(current OrchestrationRunV0, event OrchestrationEventV0, payload ReviewAcceptedPayloadV0) error {
	if err := ensureRunCanApplyEventV0(current, event); err != nil {
		return err
	}
	if err := ensureReviewAcceptedEventPhaseCurrentV0(current, payload.PhaseID); err != nil {
		return err
	}
	if !reviewRequestAlreadyReflectedV0(current, payload.ReviewRequestID) {
		return eventErrorV0(ErrSecuenciaInvalidaV0, "payload.review_request_id")
	}
	if !deliveryAlreadyReflectedV0(current, payload.DeliveryRef) {
		return eventErrorV0(ErrSecuenciaInvalidaV0, "payload.delivery_ref")
	}
	if !acceptedReviewResultAlreadyReflectedV0(current, payload.ReviewRequestID, payload.DeliveryRef) {
		return eventErrorV0(ErrSecuenciaInvalidaV0, "payload.review_result_ref")
	}
	return nil
}

func ensureAcceptReviewCommandPhaseCurrentV0(run OrchestrationRunV0, phaseID string) error {
	if !reviewPhaseCurrentV0(run, phaseID) {
		return commandErrorV0(ErrTransicionInvalidaV0, "phase.status")
	}
	return nil
}

func ensureReviewAcceptedEventPhaseCurrentV0(run OrchestrationRunV0, phaseID string) error {
	if !reviewPhaseCurrentV0(run, phaseID) {
		return eventErrorV0(ErrSecuenciaInvalidaV0, "phase.status")
	}
	return nil
}
