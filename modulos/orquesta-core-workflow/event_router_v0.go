package orquestacoreworkflow

type eventApplierV0 func(OrchestrationRunV0, OrchestrationEventV0) (OrchestrationRunV0, error)

func lookupEventApplierV0(eventType string) (eventApplierV0, bool) {
	applier, ok := eventAppliersV0[eventType]
	return applier, ok
}

var eventAppliersV0 = map[string]eventApplierV0{
	OrchestrationEventRunStartedV0:                   applyRunStartedEventV0,
	OrchestrationEventPhaseOpenedV0:                  applyPhaseOpenedEventV0,
	OrchestrationEventPhaseClosedV0:                  applyPhaseClosedEventV0,
	OrchestrationEventRunBlockedV0:                   applyRunBlockedEventV0,
	OrchestrationEventDirectorQuestionRaisedV0:       applyDirectorQuestionRaisedEventV0,
	OrchestrationEventDirectorQuestionAnsweredV0:     applyDirectorQuestionAnsweredEventV0,
	OrchestrationEventBrainstormRequestedV0:          applyBrainstormRequestedEventV0,
	OrchestrationEventVoteRequestedV0:                applyVoteRequestedEventV0,
	OrchestrationEventArchitectureDecisionAcceptedV0: applyArchitectureDecisionAcceptedEventV0,
	OrchestrationEventFunctionContractPublishedV0:    applyFunctionContractPublishedEventV0,
	OrchestrationEventMicrotaskCreatedV0:             applyMicrotaskCreatedEventV0,
	OrchestrationEventCapacityRequestedV0:            applyCapacityRequestedEventV0,
	OrchestrationEventCapacityDecidedV0:              applyCapacityDecidedEventV0,
	OrchestrationEventAgentRequestedV0:               applyAgentRequestedEventV0,
	OrchestrationEventAgentStartedV0:                 applyAgentStartedEventV0,
	OrchestrationEventAgentFailedV0:                  applyAgentFailedEventV0,
	OrchestrationEventAgentLeaseExpiredV0:            applyAgentLeaseExpiredEventV0,
	OrchestrationEventAgentStopRequestedV0:           applyAgentStopRequestedEventV0,
	OrchestrationEventAgentStopConfirmedV0:           applyAgentStopConfirmedEventV0,
	OrchestrationEventAgentWorkAssessedV0:            applyAgentWorkAssessedEventV0,
	OrchestrationEventConcurrencyGateRecordedV0:      applyConcurrencyGateRecordedEventV0,
	OrchestrationEventQualityGateRecordedV0:          applyQualityGateRecordedEventV0,
	OrchestrationEventPhaseArtifactRegisteredV0:      applyPhaseArtifactRegisteredEventV0,
	OrchestrationEventDeliveryRegisteredV0:           applyDeliveryRegisteredEventV0,
	OrchestrationEventReviewRequestedV0:              applyReviewRequestedEventV0,
	OrchestrationEventReviewAcceptedV0:               applyReviewAcceptedEventV0,
	OrchestrationEventReviewResultRecordedV0:         applyReviewResultRecordedEventV0,
	OrchestrationEventReworkRequestedV0:              applyReworkRequestedEventV0,
	OrchestrationEventReplanDecisionRecordedV0:       applyReplanDecisionRecordedEventV0,
	OrchestrationEventTaskClosedV0:                   applyTaskClosedEventV0,
	OrchestrationEventFinalValidationRegisteredV0:    applyFinalValidationRegisteredEventV0,
	OrchestrationEventRunClosedV0:                    applyRunClosedEventV0,
}
