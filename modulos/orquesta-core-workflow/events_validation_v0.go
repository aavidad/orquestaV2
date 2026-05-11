package orquestacoreworkflow

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

var forbiddenEventFragmentsV0 = []string{
	"secret",
	"secreto",
	"token",
	"password",
	"credential",
	"credencial",
	"api_key",
	"oauth",
	"transcript",
	"prompt",
	"completion",
	"raw_text",
	"full_text",
	"runtime",
	"sql",
	"dsn",
	"connection",
	"conexion",
	"table",
	"tabla",
	"provider",
	"proveedor",
	"model",
	"home",
	"tmux",
	"docker",
	"git",
}

func validateEventPayloadV0(event OrchestrationEventV0) error {
	var decoded any
	if err := json.Unmarshal(event.Payload, &decoded); err != nil {
		return eventErrorV0(ErrPayloadInvalidoV0, "payload")
	}
	if containsForbiddenEventDetailV0(decoded) {
		return eventErrorV0(ErrDetalleProhibidoV0, "payload")
	}
	switch event.EventType {
	case OrchestrationEventRunStartedV0:
		var payload RunStartedPayloadV0
		if err := decodePayloadV0(event.Payload, &payload); err != nil {
			return err
		}
		return requirePayloadFieldsV0(map[string]string{
			"project_ref":  payload.ProjectRef,
			"app_spec_ref": payload.AppSpecRef,
		})
	case OrchestrationEventPhaseOpenedV0:
		var payload PhaseOpenedPayloadV0
		if err := decodePayloadV0(event.Payload, &payload); err != nil {
			return err
		}
		return requirePayloadFieldsV0(map[string]string{"phase_id": payload.PhaseID})
	case OrchestrationEventPhaseClosedV0:
		return validatePhaseClosedPayloadV0(event)
	case OrchestrationEventRunBlockedV0:
		var payload RunBlockedPayloadV0
		if err := decodePayloadV0(event.Payload, &payload); err != nil {
			return err
		}
		return requirePayloadFieldsV0(map[string]string{
			"blocker_id":  payload.BlockerID,
			"reason_code": payload.ReasonCode,
			"summary":     payload.Summary,
		})
	case OrchestrationEventDirectorQuestionRaisedV0:
		return validateDirectorQuestionRaisedPayloadV0(event)
	case OrchestrationEventDirectorQuestionAnsweredV0:
		return validateDirectorQuestionAnsweredPayloadV0(event)
	case OrchestrationEventBrainstormRequestedV0:
		return validateBrainstormRequestedPayloadV0(event)
	case OrchestrationEventVoteRequestedV0:
		return validateVoteRequestedPayloadV0(event)
	case OrchestrationEventArchitectureDecisionAcceptedV0:
		return validateArchitectureDecisionAcceptedPayloadV0(event)
	case OrchestrationEventFunctionContractPublishedV0:
		return validateFunctionContractPublishedPayloadV0(event)
	case OrchestrationEventMicrotaskCreatedV0:
		return validateMicrotaskCreatedPayloadV0(event)
	case OrchestrationEventCapacityRequestedV0:
		return validateCapacityRequestedPayloadV0(event)
	case OrchestrationEventCapacityDecidedV0:
		return validateCapacityDecidedPayloadV0(event)
	case OrchestrationEventAgentRequestedV0:
		return validateAgentRequestedPayloadV0(event)
	case OrchestrationEventAgentStartedV0:
		return validateAgentStartedPayloadV0(event)
	case OrchestrationEventAgentFailedV0:
		return validateAgentFailedPayloadV0(event)
	case OrchestrationEventAgentLeaseExpiredV0:
		return validateAgentLeaseExpiredPayloadV0(event)
	case OrchestrationEventAgentStopRequestedV0:
		return validateAgentStopRequestedPayloadV0(event)
	case OrchestrationEventAgentStopConfirmedV0:
		return validateAgentStopConfirmedPayloadV0(event)
	case OrchestrationEventAgentWorkAssessedV0:
		return validateAgentWorkAssessedPayloadV0(event)
	case OrchestrationEventConcurrencyGateRecordedV0:
		return validateConcurrencyGateRecordedPayloadV0(event)
	case OrchestrationEventQualityGateRecordedV0:
		return validateQualityGateRecordedPayloadV0(event)
	case OrchestrationEventPhaseArtifactRegisteredV0:
		return validatePhaseArtifactRegisteredPayloadV0(event)
	case OrchestrationEventDeliveryRegisteredV0:
		return validateDeliveryRegisteredPayloadV0(event)
	case OrchestrationEventReviewRequestedV0:
		return validateReviewRequestedPayloadV0(event)
	case OrchestrationEventReviewAcceptedV0:
		return validateReviewAcceptedPayloadV0(event)
	case OrchestrationEventReviewResultRecordedV0:
		return validateReviewResultRecordedPayloadV0(event)
	case OrchestrationEventReworkRequestedV0:
		return validateReworkRequestedPayloadV0(event)
	case OrchestrationEventReplanDecisionRecordedV0:
		return validateReplanDecisionRecordedPayloadV0(event)
	case OrchestrationEventTaskClosedV0:
		return validateTaskClosedPayloadV0(event)
	case OrchestrationEventFinalValidationRegisteredV0:
		return validateFinalValidationRegisteredPayloadV0(event)
	case OrchestrationEventRunClosedV0:
		return validateRunClosedPayloadV0(event)
	default:
		return eventErrorV0(ErrEventoNoSoportadoV0, "event_type")
	}
}

func decodePayloadV0(raw json.RawMessage, target any) error {
	if err := json.Unmarshal(raw, target); err != nil {
		return eventErrorV0(ErrPayloadInvalidoV0, "payload")
	}
	return nil
}

func requirePayloadFieldsV0(fields map[string]string) error {
	for field, value := range fields {
		if strings.TrimSpace(value) == "" {
			return eventErrorV0(ErrPayloadInvalidoV0, fmt.Sprintf("payload.%s", field))
		}
	}
	return nil
}

func eventPayloadErrorV0(err error) error {
	if err == nil {
		return nil
	}
	var eventErr OrchestrationEventErrorV0
	if errors.As(err, &eventErr) {
		return eventErr
	}
	var questionErr DirectorQuestionErrorV0
	if errors.As(err, &questionErr) {
		field := strings.TrimSpace(questionErr.Field)
		if field == "" {
			return eventErrorV0(ErrPayloadInvalidoV0, "payload")
		}
		return eventErrorV0(ErrPayloadInvalidoV0, "payload."+field)
	}
	var taskErr WorkflowTaskErrorV0
	if errors.As(err, &taskErr) {
		field := strings.TrimSpace(taskErr.Field)
		if field == "" {
			return eventErrorV0(ErrPayloadInvalidoV0, "payload")
		}
		if taskErr.Code == ErrDetalleProhibidoV0 {
			return eventErrorV0(ErrDetalleProhibidoV0, "payload."+field)
		}
		return eventErrorV0(ErrPayloadInvalidoV0, "payload."+field)
	}
	return eventErrorV0(ErrPayloadInvalidoV0, "payload")
}

func isSupportedEventTypeV0(eventType string) bool {
	for _, supported := range orchestrationEventTypeCatalogV0 {
		if supported == strings.TrimSpace(eventType) {
			return true
		}
	}
	return false
}

func containsForbiddenEventDetailV0(value any) bool {
	switch typed := value.(type) {
	case map[string]any:
		for key, item := range typed {
			if containsForbiddenEventTextV0(key) || containsForbiddenEventDetailV0(item) {
				return true
			}
		}
	case []any:
		for _, item := range typed {
			if containsForbiddenEventDetailV0(item) {
				return true
			}
		}
	case string:
		return containsForbiddenEventTextV0(typed)
	}
	return false
}

func containsForbiddenEventTextV0(value string) bool {
	lower := strings.ToLower(value)
	for _, fragment := range forbiddenEventFragmentsV0 {
		if containsForbiddenFragmentV0(lower, fragment) {
			return true
		}
	}
	return false
}

func containsForbiddenFragmentV0(lowerValue string, fragment string) bool {
	fragment = strings.ToLower(strings.TrimSpace(fragment))
	if fragment == "" {
		return false
	}
	start := 0
	for {
		index := strings.Index(lowerValue[start:], fragment)
		if index < 0 {
			return false
		}
		absolute := start + index
		if hasTokenBoundaryV0(lowerValue, absolute, absolute+len(fragment)) {
			return true
		}
		start = absolute + len(fragment)
	}
}

func hasTokenBoundaryV0(value string, start int, end int) bool {
	before := start == 0 || !isAsciiLetterOrDigitV0(value[start-1])
	after := end >= len(value) || !isAsciiLetterOrDigitV0(value[end])
	return before && after
}

func isAsciiLetterOrDigitV0(ch byte) bool {
	return (ch >= 'a' && ch <= 'z') || (ch >= '0' && ch <= '9')
}

func eventErrorV0(code string, field string) OrchestrationEventErrorV0 {
	return OrchestrationEventErrorV0{Code: code, Field: field}
}
