package orquestacoreworkflow

import "encoding/json"

func validateAnswerDirectorQuestionPayloadForCommandV0(command OrchestrationCommandV0) error {
	var decoded any
	if err := json.Unmarshal(command.Payload, &decoded); err != nil {
		return commandErrorV0(ErrPayloadInvalidoV0, "payload")
	}
	if containsForbiddenEventDetailV0(decoded) {
		return commandErrorV0(ErrDetalleProhibidoV0, "payload")
	}
	payload, err := decodeAnswerDirectorQuestionCommandPayloadV0(command.Payload)
	if err != nil {
		return err
	}
	return validateAnswerDirectorQuestionPayloadDataV0(payload)
}

func validateDirectorQuestionAnsweredPayloadV0(event OrchestrationEventV0) error {
	var payload DirectorQuestionAnsweredPayloadV0
	if err := decodePayloadV0(event.Payload, &payload); err != nil {
		return err
	}
	payload = normalizeDirectorQuestionAnsweredPayloadV0(payload)
	if err := validateDirectorQuestionAnsweredPayloadDataV0(payload); err != nil {
		return eventPayloadErrorV0(err)
	}
	return nil
}

func validateAnswerDirectorQuestionPayloadDataV0(
	payload AnswerDirectorQuestionCommandPayloadV0,
) error {
	return validateDirectorAnswerFieldsV0(
		payload.AnswerID,
		payload.QuestionID,
		payload.Decision,
		payload.Summary,
		payload.EvidenceRefs,
		payload.Unblocks,
		"",
		false,
	)
}

func validateDirectorQuestionAnsweredPayloadDataV0(
	payload DirectorQuestionAnsweredPayloadV0,
) error {
	return validateDirectorAnswerFieldsV0(
		payload.AnswerID,
		payload.QuestionID,
		payload.Decision,
		payload.Summary,
		payload.EvidenceRefs,
		payload.Unblocks,
		payload.BlockerID,
		true,
	)
}

func validateDirectorAnswerFieldsV0(
	answerID string,
	questionID string,
	decision string,
	summary string,
	evidenceRefs []string,
	unblocks bool,
	blockerID string,
	requireBlockerID bool,
) error {
	if answerID == "" {
		return commandErrorV0(ErrPayloadInvalidoV0, "payload.answer_id")
	}
	if questionID == "" {
		return commandErrorV0(ErrPayloadInvalidoV0, "payload.question_id")
	}
	if !directorAnswerDecisionSupportedV0(decision) {
		return commandErrorV0(ErrPayloadInvalidoV0, "payload.decision")
	}
	if summary == "" || len(summary) > maxDirectorQuestionSummaryLenV0 {
		return commandErrorV0(ErrPayloadInvalidoV0, "payload.summary")
	}
	if !compactListWithinLimitV0(evidenceRefs, maxDirectorQuestionRefsV0, maxDirectorQuestionRefLenV0) {
		return commandErrorV0(ErrPayloadInvalidoV0, "payload.evidence_refs")
	}
	if unblocks {
		expectedBlockerID := directorQuestionBlockerIDV0(questionID)
		if requireBlockerID && blockerID == "" {
			return commandErrorV0(ErrPayloadInvalidoV0, "payload.blocker_id")
		}
		if blockerID != "" && blockerID != expectedBlockerID {
			return commandErrorV0(ErrPayloadInvalidoV0, "payload.blocker_id")
		}
	}
	if !unblocks && blockerID != "" {
		return commandErrorV0(ErrPayloadInvalidoV0, "payload.blocker_id")
	}
	return nil
}

func directorAnswerDecisionSupportedV0(decision string) bool {
	switch decision {
	case DirectorAnswerDecisionContinueV0, DirectorAnswerDecisionReplanV0, DirectorAnswerDecisionStopAgentV0:
		return true
	default:
		return false
	}
}
