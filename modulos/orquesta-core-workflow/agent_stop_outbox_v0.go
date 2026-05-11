package orquestacoreworkflow

import (
	"encoding/json"
	"strings"
)

func newStopRuntimeAgentOutboxV0(command OrchestrationCommandV0, event OrchestrationEventV0, payload StopAgentCommandPayloadV0) (OutboxMessageV0, error) {
	outboxPayload := stopRuntimeAgentPayloadFromCommandV0(command, payload)
	payloadJSON, err := json.Marshal(outboxPayload)
	if err != nil {
		return OutboxMessageV0{}, outboxErrorV0(ErrOutboxPayloadInvalidoV0, "payload")
	}
	message := OutboxMessageV0{
		MessageID:        "outbox-stopruntimeagent-" + strings.TrimSpace(command.IdempotencyKey),
		MessageType:      OutboxMessageStopRuntimeAgentV0,
		RunID:            strings.TrimSpace(command.RunID),
		IdempotencyKey:   strings.TrimSpace(command.IdempotencyKey),
		CorrelationID:    strings.TrimSpace(command.CorrelationID),
		CausationEventID: strings.TrimSpace(event.EventID),
		TargetPort:       OutboxTargetAgentLauncherV0,
		PayloadVersion:   OutboxPayloadVersionV0,
		Payload:          payloadJSON,
	}
	if err := ValidateOutboxMessageV0(message); err != nil {
		return OutboxMessageV0{}, err
	}
	return message, nil
}

func validateStopRuntimeAgentPayloadV0(runID string, raw json.RawMessage) error {
	var payload StopRuntimeAgentRequestV0
	if err := json.Unmarshal(raw, &payload); err != nil {
		return outboxErrorV0(ErrOutboxPayloadInvalidoV0, "payload")
	}
	payload = normalizeStopRuntimeAgentPayloadV0(payload)
	if payload.RunID != strings.TrimSpace(runID) {
		return outboxErrorV0(ErrOutboxPayloadInvalidoV0, "payload.run_id")
	}
	return validateStopRuntimeAgentPayloadDataV0(payload)
}
