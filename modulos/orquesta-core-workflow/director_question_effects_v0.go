package orquestacoreworkflow

import "strings"

func questionFromAskDirectorCommandV0(command OrchestrationCommandV0, payload AskDirectorCommandPayloadV0) DirectorQuestionV0 {
	return DirectorQuestionV0{
		QuestionID:   payload.QuestionID,
		RunID:        strings.TrimSpace(command.RunID),
		SourceGroup:  payload.SourceGroup,
		TargetGroup:  payload.TargetGroup,
		Summary:      payload.Summary,
		Options:      cloneStringsV0(payload.Options),
		EvidenceRefs: cloneStringsV0(payload.EvidenceRefs),
		Blocking:     payload.Blocking,
		RequestedAt:  strings.TrimSpace(command.OccurredAt),
	}
}

func questionRaisedPayloadV0(question DirectorQuestionV0) DirectorQuestionRaisedPayloadV0 {
	payload := DirectorQuestionRaisedPayloadV0{
		QuestionID:   question.QuestionID,
		SourceGroup:  question.SourceGroup,
		TargetGroup:  question.TargetGroup,
		Summary:      question.Summary,
		Options:      cloneStringsV0(question.Options),
		EvidenceRefs: cloneStringsV0(question.EvidenceRefs),
		Blocking:     question.Blocking,
	}
	if question.Blocking {
		payload.BlockerID = directorQuestionBlockerIDV0(question.QuestionID)
	}
	return payload
}

func askDirectorEffectsV0(current OrchestrationRunV0, command OrchestrationCommandV0, question DirectorQuestionV0) ([]OrchestrationEventV0, []OutboxMessageV0, error) {
	events := []OrchestrationEventV0{}
	outbox := []OutboxMessageV0{}

	if !directorQuestionRefAlreadyReflectedV0(current, question.QuestionID) {
		event, err := NewDirectorQuestionRaisedEventV0(commandQuestionEventMetaV0(current, command), questionRaisedPayloadV0(question))
		if err != nil {
			return nil, nil, err
		}
		message, err := NewSendDirectorQuestionOutboxV0(commandQuestionOutboxMetaV0(command, event), question)
		if err != nil {
			return nil, nil, err
		}
		events = append(events, event)
		outbox = append(outbox, message)
	} else {
		message, err := directorQuestionOutboxForReflectedQuestionV0(current, command, question)
		if err != nil {
			return nil, nil, err
		}
		outbox = append(outbox, message)
	}

	if !question.Blocking || blockerAlreadyReflectedV0(current, directorQuestionBlockerIDV0(question.QuestionID)) {
		return events, outbox, nil
	}

	meta := commandEventMetaV0(current, command, OrchestrationEventRunBlockedV0)
	if len(events) > 0 {
		meta.Sequence = events[len(events)-1].Sequence + 1
	}
	meta.IdempotencyKey = strings.TrimSpace(command.IdempotencyKey) + ":block"
	block, err := NewRunBlockedEventV0(meta, RunBlockedPayloadV0{
		BlockerID:   directorQuestionBlockerIDV0(question.QuestionID),
		ReasonCode:  DirectorQuestionBlockReasonCodeV0,
		Summary:     question.Summary,
		SourceGroup: question.SourceGroup,
	})
	if err != nil {
		return nil, nil, err
	}
	events = append(events, block)
	return events, outbox, nil
}

func pendingDirectorQuestionOutboxResultV0(current OrchestrationRunV0, command OrchestrationCommandV0, question DirectorQuestionV0) (OrchestrationCommandResultV0, error) {
	message, err := directorQuestionOutboxForReflectedQuestionV0(current, command, question)
	if err != nil {
		return emptyCommandResultV0(), err
	}
	return OrchestrationCommandResultV0{Outbox: []OutboxMessageV0{message}}, nil
}

func directorQuestionOutboxForReflectedQuestionV0(current OrchestrationRunV0, command OrchestrationCommandV0, question DirectorQuestionV0) (OutboxMessageV0, error) {
	event, err := NewDirectorQuestionRaisedEventV0(commandQuestionEventMetaV0(current, command), questionRaisedPayloadV0(question))
	if err != nil {
		return OutboxMessageV0{}, err
	}
	return NewSendDirectorQuestionOutboxV0(commandQuestionOutboxMetaV0(command, event), question)
}

func commandQuestionEventMetaV0(current OrchestrationRunV0, command OrchestrationCommandV0) OrchestrationEventMetaV0 {
	meta := commandEventMetaV0(current, command, OrchestrationEventDirectorQuestionRaisedV0)
	meta.IdempotencyKey = strings.TrimSpace(command.IdempotencyKey) + ":question"
	return meta
}

func commandQuestionOutboxMetaV0(command OrchestrationCommandV0, event OrchestrationEventV0) OutboxMessageMetaV0 {
	return OutboxMessageMetaV0{
		MessageID:        "outbox-send-director-question-" + strings.TrimSpace(command.IdempotencyKey),
		RunID:            strings.TrimSpace(command.RunID),
		IdempotencyKey:   strings.TrimSpace(command.IdempotencyKey),
		CorrelationID:    strings.TrimSpace(command.CorrelationID),
		CausationEventID: strings.TrimSpace(event.EventID),
	}
}

func directorQuestionFullyReflectedV0(current OrchestrationRunV0, question DirectorQuestionV0) bool {
	if !directorQuestionRefAlreadyReflectedV0(current, question.QuestionID) {
		return false
	}
	return !question.Blocking || blockerAlreadyReflectedV0(current, directorQuestionBlockerIDV0(question.QuestionID))
}

func directorQuestionAnsweredAlreadyReflectedV0(current OrchestrationRunV0, questionID string) bool {
	return compactRefInListV0(current.DirectorAnsweredQuestions, questionID)
}

func directorQuestionRefAlreadyReflectedV0(current OrchestrationRunV0, questionID string) bool {
	for _, ref := range current.DirectorQuestions {
		if strings.TrimSpace(ref) == strings.TrimSpace(questionID) {
			return true
		}
	}
	return false
}

func directorQuestionBlockerIDV0(questionID string) string {
	return "director-question-" + strings.TrimSpace(questionID)
}
