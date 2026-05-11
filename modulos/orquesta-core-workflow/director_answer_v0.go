package orquestacoreworkflow

import (
	"encoding/json"
	"strings"
)

const (
	DirectorAnswerDecisionContinueV0  = "continue"
	DirectorAnswerDecisionReplanV0    = "replan"
	DirectorAnswerDecisionStopAgentV0 = "stop_agent"
)

type AnswerDirectorQuestionCommandPayloadV0 struct {
	AnswerID     string   `json:"answer_id"`
	QuestionID   string   `json:"question_id"`
	Decision     string   `json:"decision"`
	Summary      string   `json:"summary"`
	EvidenceRefs []string `json:"evidence_refs,omitempty"`
	Unblocks     bool     `json:"unblocks"`
}

type DirectorQuestionAnsweredPayloadV0 struct {
	AnswerID     string   `json:"answer_id"`
	QuestionID   string   `json:"question_id"`
	Decision     string   `json:"decision"`
	Summary      string   `json:"summary"`
	EvidenceRefs []string `json:"evidence_refs,omitempty"`
	Unblocks     bool     `json:"unblocks"`
	BlockerID    string   `json:"blocker_id,omitempty"`
}

func NewAnswerDirectorQuestionCommandV0(
	meta OrchestrationCommandMetaV0,
	payload AnswerDirectorQuestionCommandPayloadV0,
) (OrchestrationCommandV0, error) {
	return newOrchestrationCommandV0(
		meta,
		OrchestrationCommandAnswerDirectorQuestionV0,
		normalizeAnswerDirectorQuestionPayloadV0(payload),
	)
}

func HandleAnswerDirectorQuestionCommandV0(
	current OrchestrationRunV0,
	command OrchestrationCommandV0,
) (OrchestrationCommandResultV0, error) {
	if err := ValidateOrchestrationCommandV0(command); err != nil {
		return emptyCommandResultV0(), err
	}
	payload, err := decodeAnswerDirectorQuestionCommandPayloadV0(command.Payload)
	if err != nil {
		return emptyCommandResultV0(), err
	}
	if directorAnswerAlreadyReflectedV0(current, payload.AnswerID) {
		if err := ensureExistingRunForCommandV0(current, command); err != nil {
			return emptyCommandResultV0(), err
		}
		if err := ensureCommandEffectMatchesV0(current, command, OrchestrationEventDirectorQuestionAnsweredV0, payload.AnswerID, directorQuestionAnsweredPayloadFromCommandV0(payload)); err != nil {
			return emptyCommandResultV0(), err
		}
		return idempotentCommandResultV0(), nil
	}
	if err := ensureAnswerDirectorQuestionAllowedV0(current, command, payload); err != nil {
		return emptyCommandResultV0(), err
	}
	event, err := NewDirectorQuestionAnsweredEventV0(
		commandEventMetaV0(current, command, OrchestrationEventDirectorQuestionAnsweredV0),
		directorQuestionAnsweredPayloadFromCommandV0(payload),
	)
	return eventCommandResultV0(event, err)
}

func NewDirectorQuestionAnsweredEventV0(
	meta OrchestrationEventMetaV0,
	payload DirectorQuestionAnsweredPayloadV0,
) (OrchestrationEventV0, error) {
	return newOrchestrationEventV0(meta, OrchestrationEventDirectorQuestionAnsweredV0, normalizeDirectorQuestionAnsweredPayloadV0(payload))
}

func decodeAnswerDirectorQuestionCommandPayloadV0(
	raw json.RawMessage,
) (AnswerDirectorQuestionCommandPayloadV0, error) {
	var payload AnswerDirectorQuestionCommandPayloadV0
	if err := json.Unmarshal(raw, &payload); err != nil {
		return AnswerDirectorQuestionCommandPayloadV0{}, commandErrorV0(ErrPayloadInvalidoV0, "payload")
	}
	return normalizeAnswerDirectorQuestionPayloadV0(payload), nil
}

func normalizeAnswerDirectorQuestionPayloadV0(
	payload AnswerDirectorQuestionCommandPayloadV0,
) AnswerDirectorQuestionCommandPayloadV0 {
	payload.AnswerID = strings.TrimSpace(payload.AnswerID)
	payload.QuestionID = strings.TrimSpace(payload.QuestionID)
	payload.Decision = strings.TrimSpace(payload.Decision)
	payload.Summary = strings.TrimSpace(payload.Summary)
	payload.EvidenceRefs = compactStringsV0(payload.EvidenceRefs)
	return payload
}
