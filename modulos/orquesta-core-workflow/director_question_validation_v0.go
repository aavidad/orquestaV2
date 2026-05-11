package orquestacoreworkflow

import (
	"encoding/json"
	"strings"
)

func ValidateAskDirectorCommandV0(command OrchestrationCommandV0) error {
	if err := validateAskDirectorEnvelopeV0(command); err != nil {
		return err
	}
	return validateAskDirectorPayloadForCommandV0(command)
}

func validateAskDirectorPayloadForCommandV0(command OrchestrationCommandV0) error {
	var decoded any
	if err := json.Unmarshal(command.Payload, &decoded); err != nil {
		return commandErrorV0(ErrPayloadInvalidoV0, "payload")
	}
	if containsForbiddenEventDetailV0(decoded) {
		return commandErrorV0(ErrDetalleProhibidoV0, "payload")
	}
	payload, err := decodeAskDirectorCommandPayloadV0(command.Payload)
	if err != nil {
		return err
	}
	_, err = NewDirectorQuestionV0(questionFromAskDirectorCommandV0(command, payload))
	return err
}

func validateAskDirectorEnvelopeV0(command OrchestrationCommandV0) error {
	if strings.TrimSpace(command.CommandID) == "" {
		return commandErrorV0(ErrComandoInvalidoV0, "command_id")
	}
	if strings.TrimSpace(command.CommandType) != OrchestrationCommandAskDirectorV0 {
		return commandErrorV0(ErrComandoNoSoportadoV0, "command_type")
	}
	if strings.TrimSpace(command.RunID) == "" {
		return commandErrorV0(ErrComandoInvalidoV0, "run_id")
	}
	if strings.TrimSpace(command.IdempotencyKey) == "" {
		return commandErrorV0(ErrIdempotencyKeyRequeridaV0, "idempotency_key")
	}
	if strings.TrimSpace(command.OccurredAt) == "" {
		return commandErrorV0(ErrComandoInvalidoV0, "occurred_at")
	}
	if command.PayloadVersion != OrchestrationCommandPayloadVersionV0 || len(command.Payload) == 0 {
		return commandErrorV0(ErrPayloadInvalidoV0, "payload")
	}
	return nil
}

func validateDirectorQuestionRaisedPayloadV0(event OrchestrationEventV0) error {
	var decoded any
	if err := json.Unmarshal(event.Payload, &decoded); err != nil {
		return eventErrorV0(ErrPayloadInvalidoV0, "payload")
	}
	if containsForbiddenEventDetailV0(decoded) {
		return eventErrorV0(ErrDetalleProhibidoV0, "payload")
	}
	var payload DirectorQuestionRaisedPayloadV0
	if err := decodePayloadV0(event.Payload, &payload); err != nil {
		return err
	}
	question := DirectorQuestionV0{
		QuestionID:   payload.QuestionID,
		RunID:        event.RunID,
		SourceGroup:  payload.SourceGroup,
		TargetGroup:  payload.TargetGroup,
		Summary:      payload.Summary,
		Options:      payload.Options,
		EvidenceRefs: payload.EvidenceRefs,
		Blocking:     payload.Blocking,
		RequestedAt:  event.OccurredAt,
	}
	if _, err := NewDirectorQuestionV0(question); err != nil {
		return eventPayloadErrorV0(err)
	}
	if payload.Blocking && strings.TrimSpace(payload.BlockerID) == "" {
		return eventErrorV0(ErrPayloadInvalidoV0, "payload.blocker_id")
	}
	if !payload.Blocking && strings.TrimSpace(payload.BlockerID) != "" {
		return eventErrorV0(ErrPayloadInvalidoV0, "payload.blocker_id")
	}
	return nil
}

func decodeAskDirectorCommandPayloadV0(raw json.RawMessage) (AskDirectorCommandPayloadV0, error) {
	var payload AskDirectorCommandPayloadV0
	if err := json.Unmarshal(raw, &payload); err != nil {
		return AskDirectorCommandPayloadV0{}, commandErrorV0(ErrPayloadInvalidoV0, "payload")
	}
	return normalizeAskDirectorPayloadV0(payload), nil
}

func normalizeAskDirectorPayloadV0(payload AskDirectorCommandPayloadV0) AskDirectorCommandPayloadV0 {
	payload.QuestionID = strings.TrimSpace(payload.QuestionID)
	payload.SourceGroup = strings.TrimSpace(payload.SourceGroup)
	payload.TargetGroup = normalizeDirectorTargetGroupV0(payload.TargetGroup)
	payload.Summary = strings.TrimSpace(payload.Summary)
	payload.Options = compactStringsV0(payload.Options)
	payload.EvidenceRefs = compactStringsV0(payload.EvidenceRefs)
	return payload
}

func normalizeQuestionRaisedPayloadV0(payload DirectorQuestionRaisedPayloadV0) DirectorQuestionRaisedPayloadV0 {
	payload.QuestionID = strings.TrimSpace(payload.QuestionID)
	payload.SourceGroup = strings.TrimSpace(payload.SourceGroup)
	payload.TargetGroup = normalizeDirectorTargetGroupV0(payload.TargetGroup)
	payload.Summary = strings.TrimSpace(payload.Summary)
	payload.Options = compactStringsV0(payload.Options)
	payload.EvidenceRefs = compactStringsV0(payload.EvidenceRefs)
	payload.BlockerID = strings.TrimSpace(payload.BlockerID)
	return payload
}
