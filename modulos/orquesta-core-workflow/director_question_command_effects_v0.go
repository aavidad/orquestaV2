package orquestacoreworkflow

func ensureDirectorQuestionEffectMatchesV0(
	run OrchestrationRunV0,
	command OrchestrationCommandV0,
	question DirectorQuestionV0,
) error {
	meta := commandQuestionEventMetaV0(run, command)
	record, err := commandEffectRecordFromExpectedEventV0(
		OrchestrationEventDirectorQuestionRaisedV0,
		question.QuestionID,
		meta.IdempotencyKey,
		command.CommandID,
		meta.EventID,
		questionRaisedPayloadV0(question),
	)
	if err != nil {
		return err
	}
	return ensureCommandEffectRecordMatchesV0(
		run,
		OrchestrationEventDirectorQuestionRaisedV0,
		question.QuestionID,
		record,
	)
}
