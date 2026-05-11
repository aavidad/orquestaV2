package orquestacoreworkflow

func ensureAgentWorkAssessedEffectMatchesV0(
	run OrchestrationRunV0,
	command OrchestrationCommandV0,
	payload AssessAgentWorkCommandPayloadV0,
) error {
	return ensureCommandEffectMatchesV0(
		run,
		command,
		OrchestrationEventAgentWorkAssessedV0,
		payload.AssessmentRef,
		agentWorkAssessedPayloadFromCommandV0(payload),
	)
}

func assessmentStopEffectMatchesV0(
	run OrchestrationRunV0,
	command OrchestrationCommandV0,
	payload AssessAgentWorkCommandPayloadV0,
) (bool, error) {
	record, err := assessmentStopEffectRecordV0(command, payload)
	if err != nil {
		return false, err
	}
	return commandEffectRecordMatchesV0(
		run,
		OrchestrationEventAgentStopRequestedV0,
		payload.AgentRequestID,
		record,
	)
}

func assessmentStopEffectRecordV0(
	command OrchestrationCommandV0,
	payload AssessAgentWorkCommandPayloadV0,
) (commandEffectRecordV0, error) {
	stopKey := assessmentStopIdempotencyKeyV0(command)
	stopCommand := commandWithIdempotencyKeyV0(command, stopKey)
	stopPayload := stopAgentPayloadFromAssessmentV0(payload)
	return commandEffectRecordFromExpectedEventV0(
		OrchestrationEventAgentStopRequestedV0,
		payload.AgentRequestID,
		stopKey,
		command.CommandID,
		commandEventIDV0(stopCommand, OrchestrationEventAgentStopRequestedV0),
		agentStopRequestedPayloadFromCommandV0(stopPayload),
	)
}
