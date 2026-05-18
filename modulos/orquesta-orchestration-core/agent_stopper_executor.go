package orquestacionnucleoapp

import (
	"context"
	"encoding/json"
	"strings"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestaoutboxdispatch "orquesta/modulos/orquesta-outbox-dispatch"
	orquestaruntime "orquesta/modulos/orquesta-runtime"
)

type AgentStopperExecutorV0 struct {
	RunStore      RunStorePortV0
	EventSink     EventSinkPortV0
	Stopper       AgentStopperPortV0
	ObservedAt    string
	CorrelationID string
	RequestedBy   string
	Summary       string
	EvidenceRefs  []string
}

func (executor AgentStopperExecutorV0) ExecuteOutboxDispatchV0(
	intent orquestaoutboxdispatch.DispatchIntentV0,
) (orquestaoutboxdispatch.OutboxDispatchExecutionResultV0, error) {
	intent = normalizeAgentStopperIntentV0(intent)
	if err := executor.validateV0(intent); err != nil {
		return orquestaoutboxdispatch.OutboxDispatchExecutionResultV0{}, err
	}
	inbound, err := agentStopperInboundFromIntentV0(intent)
	if err != nil {
		return orquestaoutboxdispatch.OutboxDispatchExecutionResultV0{}, err
	}
	result, err := executor.Stopper.StopAgentV0(context.Background(), inbound)
	if err != nil {
		if stopperRuntimeLostV0(err) {
			return executor.registerAgentLostAfterUnconfirmedStopV0(intent, inbound)
		}
		return orquestaoutboxdispatch.OutboxDispatchExecutionResultV0{}, err
	}
	command, err := executor.agentStopConfirmedCommandV0(intent, inbound, result)
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
		DispatchRef:  strings.TrimSpace(result.ConfirmationRef),
		EvidenceRefs: executor.agentStopperEvidenceRefsV0(inbound, result),
	}, nil
}

func (executor AgentStopperExecutorV0) validateV0(
	intent orquestaoutboxdispatch.DispatchIntentV0,
) error {
	if executor.RunStore == nil {
		return errorV0(ErrNucleoOrquestacionInvalidoV0, "run_store", "run_store requerido")
	}
	if executor.Stopper == nil {
		return errorV0(ErrNucleoOrquestacionInvalidoV0, "agent_stopper", "agent_stopper requerido")
	}
	if strings.TrimSpace(executor.ObservedAt) == "" {
		return errorV0(ErrNucleoOrquestacionInvalidoV0, "observed_at", "observed_at requerido")
	}
	if intent.MessageType != orquestacoreworkflow.OutboxMessageStopRuntimeAgentV0 {
		return errorV0(ErrNucleoOrquestacionInvalidoV0, "message_type", "tipo de outbox no soportado")
	}
	if intent.TargetPort != orquestacoreworkflow.OutboxTargetAgentLauncherV0 {
		return errorV0(ErrNucleoOrquestacionInvalidoV0, "target_port", "puerto agent_launcher requerido")
	}
	if len(intent.Payload) == 0 {
		return errorV0(ErrNucleoOrquestacionInvalidoV0, "payload", "payload requerido")
	}
	return nil
}

func agentStopperInboundFromIntentV0(
	intent orquestaoutboxdispatch.DispatchIntentV0,
) (orquestaruntime.AgentStopperInboundV0, error) {
	var payload orquestaruntime.StopRuntimeAgentRequestV0
	if err := json.Unmarshal(intent.Payload, &payload); err != nil {
		return orquestaruntime.AgentStopperInboundV0{},
			errorV0(ErrNucleoOrquestacionInvalidoV0, "payload", err.Error())
	}
	inbound := orquestaruntime.AgentStopperInboundV0{
		TargetPort:     orquestaruntime.AgentStopperTargetPortV0,
		MessageType:    orquestaruntime.AgentStopperMessageTypeV0,
		CorrelationID:  intent.CorrelationID,
		IdempotencyKey: intent.IdempotencyKey,
		Payload:        &payload,
	}
	if issues := orquestaruntime.ValidateAgentStopperInboundV0(inbound); len(issues) > 0 {
		return inbound, errorV0(
			ErrNucleoOrquestacionInvalidoV0,
			"agent_stopper_inbound."+issues[0].Field,
			string(issues[0].Code),
		)
	}
	return inbound, nil
}

func normalizeAgentStopperIntentV0(
	intent orquestaoutboxdispatch.DispatchIntentV0,
) orquestaoutboxdispatch.DispatchIntentV0 {
	intent.MessageID = strings.TrimSpace(intent.MessageID)
	intent.RunID = strings.TrimSpace(intent.RunID)
	intent.TargetPort = strings.TrimSpace(intent.TargetPort)
	intent.MessageType = strings.TrimSpace(intent.MessageType)
	intent.IdempotencyKey = strings.TrimSpace(intent.IdempotencyKey)
	intent.CorrelationID = strings.TrimSpace(intent.CorrelationID)
	intent.PayloadVersion = strings.TrimSpace(intent.PayloadVersion)
	intent.Payload = append(intent.Payload[:0:0], intent.Payload...)
	return intent
}
