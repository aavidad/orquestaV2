package orquestacoreworkflow

import "encoding/json"

const maxWorkflowCommandEventPayloadBytesV0 = maxOutboxPayloadBytesV0

func validateCommandPayloadBudgetV0(command OrchestrationCommandV0) error {
	if !payloadWithinBudgetV0(command.Payload, workflowCommandPayloadMaxBytesV0(command.CommandType)) {
		return commandErrorV0(ErrPayloadInvalidoV0, "payload")
	}
	return nil
}

func validateEventPayloadBudgetV0(event OrchestrationEventV0) error {
	if !payloadWithinBudgetV0(event.Payload, workflowEventPayloadMaxBytesV0(event.EventType)) {
		return eventErrorV0(ErrPayloadInvalidoV0, "payload")
	}
	return nil
}

func ValidateOrchestrationEventPayloadBudgetV0(event OrchestrationEventV0) error {
	return validateEventPayloadBudgetV0(event)
}

func workflowCommandPayloadMaxBytesV0(commandType string) int {
	switch commandType {
	case OrchestrationCommandCreateMicrotaskV0:
		return maxCreateMicrotaskCommandPayloadBytesV0
	default:
		return maxWorkflowCommandEventPayloadBytesV0
	}
}

func workflowEventPayloadMaxBytesV0(eventType string) int {
	switch eventType {
	case OrchestrationEventMicrotaskCreatedV0:
		return maxMicrotaskCreatedPayloadBytesV0
	default:
		return maxWorkflowCommandEventPayloadBytesV0
	}
}

func payloadWithinBudgetV0(payload json.RawMessage, maxBytes int) bool {
	return len(payload) > 0 && len(payload) <= maxBytes
}
