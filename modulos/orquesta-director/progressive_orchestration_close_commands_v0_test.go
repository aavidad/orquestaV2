package orquestadirector

import orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"

func (h *progressiveHarnessV0) registerDelivery(deliveryRef string, taskRef string, agentRef string) orquestacoreworkflow.OrchestrationCommandV0 {
	h.t.Helper()
	cmd, err := orquestacoreworkflow.NewRegisterDeliveryCommandV0(h.meta("delivery-"+deliveryRef), orquestacoreworkflow.RegisterDeliveryCommandPayloadV0{
		DeliveryRef:  deliveryRef,
		PhaseID:      string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0),
		TaskID:       taskRef,
		AgentRef:     agentRef,
		Summary:      "Entrega compacta de tarea lista para revision.",
		EvidenceRefs: []string{"evidence-ref-delivery-001"},
	})
	if err != nil {
		h.t.Fatalf("NewRegisterDeliveryCommandV0: %v", err)
	}
	return cmd
}

func (h *progressiveHarnessV0) requestReview(reviewRef string, deliveryRef string) orquestacoreworkflow.OrchestrationCommandV0 {
	h.t.Helper()
	cmd, err := orquestacoreworkflow.NewRequestReviewCommandV0(h.meta("review-"+reviewRef), orquestacoreworkflow.RequestReviewCommandPayloadV0{
		ReviewRequestID: reviewRef,
		PhaseID:         string(orquestacoreworkflow.OrchestrationPhaseRevisionV0),
		DeliveryRef:     deliveryRef,
		Summary:         "Solicitar revision compacta de entrega.",
		EvidenceRefs:    []string{"evidence-ref-review-001"},
	})
	if err != nil {
		h.t.Fatalf("NewRequestReviewCommandV0: %v", err)
	}
	return cmd
}

func (h *progressiveHarnessV0) recordAcceptedReviewResult(reviewResultRef string, reviewRef string, deliveryRef string) orquestacoreworkflow.OrchestrationCommandV0 {
	h.t.Helper()
	cmd, err := orquestacoreworkflow.NewRecordReviewResultCommandV0(h.meta("record-review-result-"+reviewResultRef), orquestacoreworkflow.RecordReviewResultCommandPayloadV0{
		ReviewResultRef: reviewResultRef,
		ReviewRequestID: reviewRef,
		DeliveryRef:     deliveryRef,
		Status:          orquestacoreworkflow.ReviewResultStatusAcceptedV0,
		Summary:         "Resultado compacto de revision aceptado.",
		EvidenceRefs:    []string{"evidence-ref-review-result-accepted-001"},
	})
	if err != nil {
		h.t.Fatalf("NewRecordReviewResultCommandV0: %v", err)
	}
	return cmd
}

func (h *progressiveHarnessV0) acceptReview(acceptedReviewRef string, reviewRef string, deliveryRef string) orquestacoreworkflow.OrchestrationCommandV0 {
	h.t.Helper()
	cmd, err := orquestacoreworkflow.NewAcceptReviewCommandV0(h.meta("accept-review-"+acceptedReviewRef), orquestacoreworkflow.AcceptReviewCommandPayloadV0{
		AcceptedReviewRef: acceptedReviewRef,
		PhaseID:           string(orquestacoreworkflow.OrchestrationPhaseRevisionV0),
		ReviewRequestID:   reviewRef,
		DeliveryRef:       deliveryRef,
		Summary:           "Aceptar revision sin cambios pendientes.",
		EvidenceRefs:      []string{"evidence-ref-review-accepted-001"},
	})
	if err != nil {
		h.t.Fatalf("NewAcceptReviewCommandV0: %v", err)
	}
	return cmd
}

func (h *progressiveHarnessV0) closeTask(taskRef string, deliveryRef string, acceptedReviewRef string) orquestacoreworkflow.OrchestrationCommandV0 {
	h.t.Helper()
	cmd, err := orquestacoreworkflow.NewCloseTaskCommandV0(h.meta("close-task-"+taskRef), orquestacoreworkflow.CloseTaskCommandPayloadV0{
		TaskID:            taskRef,
		PhaseID:           string(orquestacoreworkflow.OrchestrationPhaseRevisionV0),
		DeliveryRef:       deliveryRef,
		AcceptedReviewRef: acceptedReviewRef,
		Summary:           "Cerrar tarea revisada con evidencia compacta.",
		EvidenceRefs:      []string{"evidence-ref-task-closed-001"},
	})
	if err != nil {
		h.t.Fatalf("NewCloseTaskCommandV0: %v", err)
	}
	return cmd
}

func (h *progressiveHarnessV0) registerFinalValidation(validationRef string, closedTaskRef string) orquestacoreworkflow.OrchestrationCommandV0 {
	h.t.Helper()
	cmd, err := orquestacoreworkflow.NewRegisterFinalValidationCommandV0(h.meta("final-validation-"+validationRef), orquestacoreworkflow.RegisterFinalValidationCommandPayloadV0{
		ValidationRef: validationRef,
		PhaseID:       string(orquestacoreworkflow.OrchestrationPhaseValidacionFinalV0),
		ClosedTaskRef: closedTaskRef,
		Summary:       "Validacion final compacta de tarea cerrada.",
		EvidenceRefs:  []string{"evidence-ref-final-validation-001"},
	})
	if err != nil {
		h.t.Fatalf("NewRegisterFinalValidationCommandV0: %v", err)
	}
	return cmd
}

func (h *progressiveHarnessV0) closeRun(closureRef string, validationRef string) orquestacoreworkflow.OrchestrationCommandV0 {
	h.t.Helper()
	cmd, err := orquestacoreworkflow.NewCloseRunCommandV0(h.meta("close-run-"+closureRef), orquestacoreworkflow.CloseRunCommandPayloadV0{
		ClosureRef:    closureRef,
		PhaseID:       string(orquestacoreworkflow.OrchestrationPhaseCierreV0),
		ValidationRef: validationRef,
		Summary:       "Cerrar ejecucion con validacion final registrada.",
		EvidenceRefs:  []string{"evidence-ref-run-closed-001"},
	})
	if err != nil {
		h.t.Fatalf("NewCloseRunCommandV0: %v", err)
	}
	return cmd
}
