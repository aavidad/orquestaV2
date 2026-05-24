package orquestacoreworkflow

import (
	"encoding/json"
	"errors"
	"strings"
)

func ValidateOutboxMessageV0(message OutboxMessageV0) error {
	if strings.TrimSpace(message.MessageID) == "" {
		return outboxErrorV0(ErrOutboxInvalidoV0, "message_id")
	}
	if !isSupportedOutboxMessageTypeV0(message.MessageType) {
		return outboxErrorV0(ErrOutboxTipoNoSoportadoV0, "message_type")
	}
	if strings.TrimSpace(message.RunID) == "" {
		return outboxErrorV0(ErrOutboxInvalidoV0, "run_id")
	}
	if strings.TrimSpace(message.IdempotencyKey) == "" {
		return outboxErrorV0(ErrOutboxInvalidoV0, "idempotency_key")
	}
	if !targetPortMatchesMessageTypeV0(message.MessageType, message.TargetPort) {
		return outboxErrorV0(ErrOutboxInvalidoV0, "target_port")
	}
	if message.PayloadVersion != OutboxPayloadVersionV0 {
		return outboxErrorV0(ErrOutboxPayloadInvalidoV0, "payload_version")
	}
	if outboxEnvelopeHasForbiddenDetailV0(message) {
		return outboxErrorV0(ErrOutboxDetalleProhibidoV0, "envelope")
	}
	return validateOutboxPayloadV0(message)
}

func validateOutboxPayloadV0(message OutboxMessageV0) error {
	if len(message.Payload) == 0 || len(message.Payload) > maxOutboxPayloadBytesV0 {
		return outboxErrorV0(ErrOutboxPayloadInvalidoV0, "payload")
	}
	var decoded any
	if err := json.Unmarshal(message.Payload, &decoded); err != nil {
		return outboxErrorV0(ErrOutboxPayloadInvalidoV0, "payload")
	}
	if containsForbiddenOutboxDetailV0(decoded) {
		return outboxErrorV0(ErrOutboxDetalleProhibidoV0, "payload")
	}
	if message.MessageType == OutboxMessageSendDirectorQuestionV0 {
		return validateDirectorQuestionPayloadV0(message.RunID, message.Payload)
	}
	if message.MessageType == OutboxMessageRequestCapacityDecisionV0 {
		return validateCapacityDecisionRequestPayloadV0(message.RunID, message.Payload)
	}
	if message.MessageType == OutboxMessageLaunchRuntimeAgentV0 {
		return validateLaunchRuntimeAgentPayloadV0(message.RunID, message.Payload)
	}
	if message.MessageType == OutboxMessageStopRuntimeAgentV0 {
		return validateStopRuntimeAgentPayloadV0(message.RunID, message.Payload)
	}
	return nil
}

func validateDirectorQuestionPayloadV0(runID string, raw json.RawMessage) error {
	var question DirectorQuestionV0
	if err := json.Unmarshal(raw, &question); err != nil {
		return outboxErrorV0(ErrOutboxPayloadInvalidoV0, "payload")
	}
	if err := ValidateDirectorQuestionV0(question); err != nil {
		return outboxPayloadErrorFromDirectorQuestionV0(err)
	}
	if strings.TrimSpace(question.RunID) != strings.TrimSpace(runID) {
		return outboxErrorV0(ErrOutboxPayloadInvalidoV0, "payload.run_id")
	}
	return nil
}

func outboxPayloadErrorFromDirectorQuestionV0(err error) error {
	var questionErr DirectorQuestionErrorV0
	if errors.As(err, &questionErr) {
		field := strings.TrimSpace(questionErr.Field)
		if field == "" {
			return outboxErrorV0(ErrOutboxPayloadInvalidoV0, "payload")
		}
		return outboxErrorV0(ErrOutboxPayloadInvalidoV0, "payload."+field)
	}
	return outboxErrorV0(ErrOutboxPayloadInvalidoV0, "payload")
}

func isSupportedOutboxMessageTypeV0(messageType string) bool {
	switch strings.TrimSpace(messageType) {
	case OutboxMessagePersistRunEventsV0, OutboxMessagePublishOrquestaEventV0,
		OutboxMessageRequestCapacityDecisionV0, OutboxMessageLaunchRuntimeAgentV0,
		OutboxMessageStopRuntimeAgentV0, OutboxMessageRequestDeployPlanV0,
		OutboxMessageSendDirectorQuestionV0:
		return true
	default:
		return false
	}
}

func targetPortMatchesMessageTypeV0(messageType string, targetPort string) bool {
	return expectedOutboxTargetPortV0(messageType) == strings.TrimSpace(targetPort)
}

func expectedOutboxTargetPortV0(messageType string) string {
	switch strings.TrimSpace(messageType) {
	case OutboxMessagePersistRunEventsV0:
		return OutboxTargetPersistenceV0
	case OutboxMessagePublishOrquestaEventV0:
		return OutboxTargetObservabilityV0
	case OutboxMessageRequestCapacityDecisionV0:
		return OutboxTargetCapacityV0
	case OutboxMessageLaunchRuntimeAgentV0:
		return OutboxTargetAgentLauncherV0
	case OutboxMessageStopRuntimeAgentV0:
		return OutboxTargetAgentLauncherV0
	case OutboxMessageRequestDeployPlanV0:
		return OutboxTargetDeployPlannerV0
	case OutboxMessageSendDirectorQuestionV0:
		return OutboxTargetDirectorV0
	default:
		return ""
	}
}

func outboxEnvelopeHasForbiddenDetailV0(message OutboxMessageV0) bool {
	values := []string{
		message.MessageID,
		message.RunID,
		message.IdempotencyKey,
		message.CorrelationID,
		message.CausationEventID,
		message.TargetPort,
	}
	for _, value := range values {
		if containsForbiddenOutboxTextV0(value) {
			return true
		}
	}
	return false
}

func containsForbiddenOutboxDetailV0(value any) bool {
	if !detailProhibitedRailsEnabledV0() {
		return false
	}
	switch typed := value.(type) {
	case map[string]any:
		return mapContainsForbiddenOutboxDetailV0(typed)
	case []any:
		if len(typed) > maxOutboxCollectionLenV0 {
			return true
		}
		for _, item := range typed {
			if containsForbiddenOutboxDetailV0(item) {
				return true
			}
		}
	case string:
		return len(typed) > maxOutboxStringValueV0 || containsForbiddenOutboxTextV0(typed)
	}
	return false
}

func mapContainsForbiddenOutboxDetailV0(values map[string]any) bool {
	if !detailProhibitedRailsEnabledV0() {
		return false
	}
	if len(values) > maxOutboxCollectionLenV0 {
		return true
	}
	for key, item := range values {
		if containsForbiddenOutboxKeyV0(key) || containsForbiddenOutboxDetailV0(item) {
			return true
		}
	}
	return false
}

func containsForbiddenOutboxKeyV0(value string) bool {
	if !detailProhibitedRailsEnabledV0() {
		return false
	}
	lower := strings.ToLower(value)
	for _, fragment := range forbiddenOutboxPayloadKeysV0 {
		if strings.Contains(lower, fragment) {
			return true
		}
	}
	return containsForbiddenOutboxTextV0(value)
}

func containsForbiddenOutboxTextV0(value string) bool {
	return textContainsForbiddenOperationalSensitiveDetailV0(value)
}
