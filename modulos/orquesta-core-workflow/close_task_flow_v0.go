package orquestacoreworkflow

func ensureCloseTaskCommandAllowedV0(current OrchestrationRunV0, command OrchestrationCommandV0, payload CloseTaskCommandPayloadV0) error {
	if err := ensureActiveRunForCommandV0(current, command); err != nil {
		return err
	}
	if err := ensureCloseTaskCommandPhaseCurrentV0(current, payload.PhaseID); err != nil {
		return err
	}
	if !microtaskAlreadyReflectedV0(current, payload.TaskID) {
		return commandErrorV0(ErrTransicionInvalidaV0, "payload.task_id")
	}
	if !deliveryAlreadyReflectedV0(current, payload.DeliveryRef) {
		return commandErrorV0(ErrTransicionInvalidaV0, "payload.delivery_ref")
	}
	if !acceptedReviewAlreadyReflectedV0(current, payload.AcceptedReviewRef) {
		return commandErrorV0(ErrTransicionInvalidaV0, "payload.accepted_review_ref")
	}
	return nil
}

func ensureTaskClosedEventAllowedV0(current OrchestrationRunV0, event OrchestrationEventV0, payload TaskClosedPayloadV0) error {
	if err := ensureRunCanApplyEventV0(current, event); err != nil {
		return err
	}
	if err := ensureTaskClosedEventPhaseCurrentV0(current, payload.PhaseID); err != nil {
		return err
	}
	if !microtaskAlreadyReflectedV0(current, payload.TaskID) {
		return eventErrorV0(ErrSecuenciaInvalidaV0, "payload.task_id")
	}
	if !deliveryAlreadyReflectedV0(current, payload.DeliveryRef) {
		return eventErrorV0(ErrSecuenciaInvalidaV0, "payload.delivery_ref")
	}
	if !acceptedReviewAlreadyReflectedV0(current, payload.AcceptedReviewRef) {
		return eventErrorV0(ErrSecuenciaInvalidaV0, "payload.accepted_review_ref")
	}
	return nil
}

func ensureCloseTaskCommandPhaseCurrentV0(run OrchestrationRunV0, phaseID string) error {
	if !reviewPhaseCurrentV0(run, phaseID) {
		return commandErrorV0(ErrTransicionInvalidaV0, "phase.status")
	}
	return nil
}

func ensureTaskClosedEventPhaseCurrentV0(run OrchestrationRunV0, phaseID string) error {
	if !reviewPhaseCurrentV0(run, phaseID) {
		return eventErrorV0(ErrSecuenciaInvalidaV0, "phase.status")
	}
	return nil
}
