package orquestacoreworkflow

import "strings"

func directorQuestionAnsweredPayloadFromCommandV0(
	payload AnswerDirectorQuestionCommandPayloadV0,
) DirectorQuestionAnsweredPayloadV0 {
	answer := DirectorQuestionAnsweredPayloadV0{
		AnswerID:     payload.AnswerID,
		QuestionID:   payload.QuestionID,
		Decision:     payload.Decision,
		Summary:      payload.Summary,
		EvidenceRefs: cloneStringsV0(payload.EvidenceRefs),
		Unblocks:     payload.Unblocks,
	}
	if payload.Unblocks {
		answer.BlockerID = directorQuestionBlockerIDV0(payload.QuestionID)
	}
	return answer
}

func normalizeDirectorQuestionAnsweredPayloadV0(
	payload DirectorQuestionAnsweredPayloadV0,
) DirectorQuestionAnsweredPayloadV0 {
	payload.AnswerID = strings.TrimSpace(payload.AnswerID)
	payload.QuestionID = strings.TrimSpace(payload.QuestionID)
	payload.Decision = strings.TrimSpace(payload.Decision)
	payload.Summary = strings.TrimSpace(payload.Summary)
	payload.EvidenceRefs = compactStringsV0(payload.EvidenceRefs)
	payload.BlockerID = strings.TrimSpace(payload.BlockerID)
	return payload
}

func directorQuestionAlreadyReflectedV0(current OrchestrationRunV0, questionID string) bool {
	for _, ref := range current.DirectorQuestions {
		if strings.TrimSpace(ref) == strings.TrimSpace(questionID) {
			return true
		}
	}
	return false
}

func directorAnswerAlreadyReflectedV0(current OrchestrationRunV0, answerID string) bool {
	for _, ref := range current.DirectorAnswers {
		if strings.TrimSpace(ref) == strings.TrimSpace(answerID) {
			return true
		}
	}
	return false
}

func removeCompactRefV0(refs []string, ref string) []string {
	compact := strings.TrimSpace(ref)
	result := make([]string, 0, len(refs))
	for _, existing := range refs {
		if strings.TrimSpace(existing) != compact {
			result = append(result, existing)
		}
	}
	return result
}

func ensureAnswerDirectorQuestionAllowedV0(
	current OrchestrationRunV0,
	command OrchestrationCommandV0,
	payload AnswerDirectorQuestionCommandPayloadV0,
) error {
	if err := ensureExistingRunForCommandV0(current, command); err != nil {
		return err
	}
	if !directorQuestionAlreadyReflectedV0(current, payload.QuestionID) {
		return commandErrorV0(ErrTransicionInvalidaV0, "payload.question_id")
	}
	if payload.Unblocks && !blockerAlreadyReflectedV0(current, directorQuestionBlockerIDV0(payload.QuestionID)) {
		return commandErrorV0(ErrTransicionInvalidaV0, "payload.unblocks")
	}
	return nil
}
