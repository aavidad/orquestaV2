package orquestacoreworkflow

func ensureRecordReviewResultCommandAllowedV0(current OrchestrationRunV0, command OrchestrationCommandV0, payload ReviewResultV0) error {
	if err := ensureActiveRunForCommandV0(current, command); err != nil {
		return err
	}
	if err := ensureRecordReviewResultCommandPhaseCurrentV0(current); err != nil {
		return err
	}
	if !reviewRequestAlreadyReflectedV0(current, payload.ReviewRequestID) {
		return commandErrorV0(ErrTransicionInvalidaV0, "payload.review_request_id")
	}
	if !deliveryAlreadyReflectedV0(current, payload.DeliveryRef) {
		return commandErrorV0(ErrTransicionInvalidaV0, "payload.delivery_ref")
	}
	return nil
}

func ensureReviewResultRecordedEventAllowedV0(current OrchestrationRunV0, event OrchestrationEventV0, payload ReviewResultV0) error {
	if err := ensureRunCanApplyEventV0(current, event); err != nil {
		return err
	}
	if err := ensureReviewResultRecordedEventPhaseCurrentV0(current); err != nil {
		return err
	}
	if !reviewRequestAlreadyReflectedV0(current, payload.ReviewRequestID) {
		return eventErrorV0(ErrSecuenciaInvalidaV0, "payload.review_request_id")
	}
	if !deliveryAlreadyReflectedV0(current, payload.DeliveryRef) {
		return eventErrorV0(ErrSecuenciaInvalidaV0, "payload.delivery_ref")
	}
	return nil
}

func ensureRecordReviewResultCommandPhaseCurrentV0(run OrchestrationRunV0) error {
	if !reviewPhaseCurrentV0(run, string(OrchestrationPhaseRevisionV0)) {
		return commandErrorV0(ErrTransicionInvalidaV0, "phase.status")
	}
	return nil
}

func ensureReviewResultRecordedEventPhaseCurrentV0(run OrchestrationRunV0) error {
	if !reviewPhaseCurrentV0(run, string(OrchestrationPhaseRevisionV0)) {
		return eventErrorV0(ErrSecuenciaInvalidaV0, "phase.status")
	}
	return nil
}
