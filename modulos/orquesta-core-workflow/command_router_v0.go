package orquestacoreworkflow

type commandHandlerV0 func(OrchestrationRunV0, OrchestrationCommandV0) (OrchestrationCommandResultV0, error)

func lookupCommandHandlerV0(commandType string) (commandHandlerV0, bool) {
	handler, ok := commandHandlersV0[commandType]
	return handler, ok
}

var commandHandlersV0 = map[string]commandHandlerV0{
	OrchestrationCommandStartRunV0:                   handleStartRunCommandV0,
	OrchestrationCommandOpenPhaseV0:                  handleOpenPhaseCommandV0,
	OrchestrationCommandClosePhaseV0:                 handleClosePhaseCommandV0,
	OrchestrationCommandBlockRunV0:                   handleBlockRunCommandV0,
	OrchestrationCommandAskDirectorV0:                HandleAskDirectorCommandV0,
	OrchestrationCommandAnswerDirectorQuestionV0:     HandleAnswerDirectorQuestionCommandV0,
	OrchestrationCommandRequestBrainstormV0:          handleRequestBrainstormCommandV0,
	OrchestrationCommandRequestVoteV0:                handleRequestVoteCommandV0,
	OrchestrationCommandAcceptDecisionV0:             handleAcceptDecisionCommandV0,
	OrchestrationCommandPublishFunctionContractV0:    handlePublishFunctionContractCommandV0,
	OrchestrationCommandCreateMicrotaskV0:            handleCreateMicrotaskCommandV0,
	OrchestrationCommandRequestCapacityV0:            handleRequestCapacityCommandV0,
	OrchestrationCommandRegisterCapacityDecisionV0:   handleRegisterCapacityDecisionCommandV0,
	OrchestrationCommandRequestAgentV0:               handleRequestAgentCommandV0,
	OrchestrationCommandRegisterAgentStartedV0:       handleRegisterAgentStartedCommandV0,
	OrchestrationCommandRegisterAgentFailedV0:        handleRegisterAgentFailedCommandV0,
	OrchestrationCommandRegisterAgentLeaseExpiredV0:  handleRegisterAgentLeaseExpiredCommandV0,
	OrchestrationCommandStopAgentV0:                  handleStopAgentCommandV0,
	OrchestrationCommandRegisterAgentStopConfirmedV0: handleRegisterAgentStopConfirmedCommandV0,
	OrchestrationCommandAssessAgentWorkV0:            handleAssessAgentWorkCommandV0,
	OrchestrationCommandRecordConcurrencyGateV0:      handleRecordConcurrencyGateCommandV0,
	OrchestrationCommandRecordQualityGateV0:          handleRecordQualityGateCommandV0,
	OrchestrationCommandRegisterPhaseArtifactV0:      handleRegisterPhaseArtifactCommandV0,
	OrchestrationCommandRegisterDeliveryV0:           handleRegisterDeliveryCommandV0,
	OrchestrationCommandRequestReviewV0:              handleRequestReviewCommandV0,
	OrchestrationCommandAcceptReviewV0:               handleAcceptReviewCommandV0,
	OrchestrationCommandRecordReviewResultV0:         handleRecordReviewResultCommandV0,
	OrchestrationCommandRequestReworkV0:              handleRequestReworkCommandV0,
	OrchestrationCommandRecordReplanDecisionV0:       handleRecordReplanDecisionCommandV0,
	OrchestrationCommandCloseTaskV0:                  handleCloseTaskCommandV0,
	OrchestrationCommandRegisterFinalValidationV0:    handleRegisterFinalValidationCommandV0,
	OrchestrationCommandCloseRunV0:                   handleCloseRunCommandV0,
}
