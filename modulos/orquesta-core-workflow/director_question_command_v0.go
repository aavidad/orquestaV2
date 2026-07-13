package orquestacoreworkflow

import (
	"encoding/json"
	"strings"
)

const (
	DirectorQuestionBlockReasonCodeV0 = "director_question_blocking"
)

type AskDirectorCommandPayloadV0 struct {
	QuestionID   string   `json:"question_id"`
	SourceGroup  string   `json:"source_group"`
	TargetGroup  string   `json:"target_group,omitempty"`
	Summary      string   `json:"summary"`
	Options      []string `json:"options,omitempty"`
	EvidenceRefs []string `json:"evidence_refs,omitempty"`
	Blocking     bool     `json:"blocking"`
}

type DirectorQuestionRaisedPayloadV0 struct {
	QuestionID   string   `json:"question_id"`
	SourceGroup  string   `json:"source_group"`
	TargetGroup  string   `json:"target_group,omitempty"`
	Summary      string   `json:"summary"`
	Options      []string `json:"options,omitempty"`
	EvidenceRefs []string `json:"evidence_refs,omitempty"`
	Blocking     bool     `json:"blocking"`
	BlockerID    string   `json:"blocker_id,omitempty"`
}

func NewAskDirectorCommandV0(meta OrchestrationCommandMetaV0, payload AskDirectorCommandPayloadV0) (OrchestrationCommandV0, error) {
	payloadJSON, err := json.Marshal(normalizeAskDirectorPayloadV0(payload))
	if err != nil {
		return OrchestrationCommandV0{}, commandErrorV0(ErrPayloadInvalidoV0, "payload")
	}
	command := OrchestrationCommandV0{
		CommandID:      strings.TrimSpace(meta.CommandID),
		CommandType:    OrchestrationCommandAskDirectorV0,
		RunID:          strings.TrimSpace(meta.RunID),
		IdempotencyKey: strings.TrimSpace(meta.IdempotencyKey),
		CorrelationID:  strings.TrimSpace(meta.CorrelationID),
		RequestedBy:    strings.TrimSpace(meta.RequestedBy),
		OccurredAt:     strings.TrimSpace(meta.OccurredAt),
		PayloadVersion: OrchestrationCommandPayloadVersionV0,
		Payload:        payloadJSON,
	}
	if err := ValidateAskDirectorCommandV0(command); err != nil {
		return OrchestrationCommandV0{}, err
	}
	return command, nil
}

func HandleAskDirectorCommandV0(current OrchestrationRunV0, command OrchestrationCommandV0) (OrchestrationCommandResultV0, error) {
	if err := ValidateAskDirectorCommandV0(command); err != nil {
		return emptyCommandResultV0(), err
	}
	if err := ensureExistingRunForCommandV0(current, command); err != nil {
		return emptyCommandResultV0(), err
	}
	payload, err := decodeAskDirectorCommandPayloadV0(command.Payload)
	if err != nil {
		return emptyCommandResultV0(), err
	}
	question, err := NewDirectorQuestionV0(questionFromAskDirectorCommandV0(command, payload))
	if err != nil {
		return emptyCommandResultV0(), err
	}
	if directorQuestionAlreadyKnownV0(current, question.QuestionID) {
		if err := ensureDirectorQuestionEffectMatchesV0(current, command, question); err != nil {
			return emptyCommandResultV0(), err
		}
	}
	if directorQuestionAnsweredAlreadyReflectedV0(current, question.QuestionID) {
		return idempotentCommandResultV0(), nil
	}
	if directorQuestionFullyReflectedV0(current, question) {
		return pendingDirectorQuestionOutboxResultV0(current, command, question)
	}

	events, outbox, err := askDirectorEffectsV0(current, command, question)
	if err != nil {
		return emptyCommandResultV0(), err
	}
	return OrchestrationCommandResultV0{Events: events, Outbox: outbox}, nil
}

func NewDirectorQuestionRaisedEventV0(meta OrchestrationEventMetaV0, payload DirectorQuestionRaisedPayloadV0) (OrchestrationEventV0, error) {
	return newOrchestrationEventV0(meta, OrchestrationEventDirectorQuestionRaisedV0, normalizeQuestionRaisedPayloadV0(payload))
}

// ValidateDirectorQuestionRaisedEventV0 preserves the public question-event boundary.
func ValidateDirectorQuestionRaisedEventV0(event OrchestrationEventV0) error {
	return ValidateOrchestrationEventV0(event)
}

func directorQuestionAlreadyKnownV0(current OrchestrationRunV0, questionID string) bool {
	return directorQuestionRefAlreadyReflectedV0(current, questionID) ||
		directorQuestionAnsweredAlreadyReflectedV0(current, questionID)
}
