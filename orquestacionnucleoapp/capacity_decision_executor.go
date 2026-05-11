package orquestacionnucleoapp

import (
	"context"
	"encoding/json"
	"strings"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestaoutboxdispatch "orquesta/modulos/orquesta-outbox-dispatch"
)

type CapacityDecisionExecutorV0 struct {
	RunStore        RunStorePortV0
	EventSink       EventSinkPortV0
	Tier            orquestacoreworkflow.OrchestrationCapacityRecommendationV0
	ReasoningEffort orquestacoreworkflow.OrchestrationCapacityRecommendationV0
	OccurredAt      string
	CorrelationID   string
	RequestedBy     string
	Summary         string
	EvidenceRefs    []string
}

func (executor CapacityDecisionExecutorV0) ExecuteOutboxDispatchV0(
	intent orquestaoutboxdispatch.DispatchIntentV0,
) (orquestaoutboxdispatch.OutboxDispatchExecutionResultV0, error) {
	intent = normalizeCapacityDecisionIntentV0(intent)
	if err := executor.validateV0(intent); err != nil {
		return orquestaoutboxdispatch.OutboxDispatchExecutionResultV0{}, err
	}
	request, err := decodeCapacityDecisionRequestV0(intent)
	if err != nil {
		return orquestaoutboxdispatch.OutboxDispatchExecutionResultV0{}, err
	}
	command, err := executor.capacityDecisionCommandV0(intent, request)
	if err != nil {
		return orquestaoutboxdispatch.OutboxDispatchExecutionResultV0{}, err
	}
	workflow := storedWorkflowPortV0{
		RunStore:  executor.RunStore,
		EventSink: executor.EventSink,
	}
	if _, err := workflow.HandleWorkflowCommandV0(context.Background(), command); err != nil {
		return orquestaoutboxdispatch.OutboxDispatchExecutionResultV0{}, err
	}
	return orquestaoutboxdispatch.OutboxDispatchExecutionResultV0{
		DispatchRef:  capacityDecisionDispatchRefV0(intent.MessageID),
		EvidenceRefs: executor.capacityDecisionEvidenceRefsV0(intent, request),
	}, nil
}

func (executor CapacityDecisionExecutorV0) validateV0(
	intent orquestaoutboxdispatch.DispatchIntentV0,
) error {
	if executor.RunStore == nil {
		return errorV0(ErrNucleoOrquestacionInvalidoV0, "run_store", "run_store requerido")
	}
	if strings.TrimSpace(executor.OccurredAt) == "" {
		return errorV0(ErrNucleoOrquestacionInvalidoV0, "occurred_at", "occurred_at requerido")
	}
	if intent.MessageType != orquestacoreworkflow.OutboxMessageRequestCapacityDecisionV0 {
		return errorV0(ErrNucleoOrquestacionInvalidoV0, "message_type", "tipo de outbox no soportado")
	}
	if intent.TargetPort != orquestacoreworkflow.OutboxTargetCapacityV0 {
		return errorV0(ErrNucleoOrquestacionInvalidoV0, "target_port", "puerto de capacidad requerido")
	}
	if len(intent.Payload) == 0 {
		return errorV0(ErrNucleoOrquestacionInvalidoV0, "payload", "payload requerido")
	}
	return nil
}

func decodeCapacityDecisionRequestV0(
	intent orquestaoutboxdispatch.DispatchIntentV0,
) (orquestacoreworkflow.CapacityDecisionRequestV0, error) {
	var request orquestacoreworkflow.CapacityDecisionRequestV0
	if err := json.Unmarshal(intent.Payload, &request); err != nil {
		return request, errorV0(ErrNucleoOrquestacionInvalidoV0, "payload", err.Error())
	}
	if strings.TrimSpace(request.RunID) != intent.RunID {
		return request, errorV0(ErrNucleoOrquestacionInvalidoV0, "payload.run_id", "run_id no coincide")
	}
	return request, nil
}

func normalizeCapacityDecisionIntentV0(
	intent orquestaoutboxdispatch.DispatchIntentV0,
) orquestaoutboxdispatch.DispatchIntentV0 {
	intent.MessageID = strings.TrimSpace(intent.MessageID)
	intent.RunID = strings.TrimSpace(intent.RunID)
	intent.TargetPort = strings.TrimSpace(intent.TargetPort)
	intent.MessageType = strings.TrimSpace(intent.MessageType)
	intent.IdempotencyKey = strings.TrimSpace(intent.IdempotencyKey)
	intent.PayloadVersion = strings.TrimSpace(intent.PayloadVersion)
	intent.Payload = append(intent.Payload[:0:0], intent.Payload...)
	return intent
}
