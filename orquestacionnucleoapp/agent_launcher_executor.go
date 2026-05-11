package orquestacionnucleoapp

import (
	"context"
	"encoding/json"
	"strings"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestaoutboxdispatch "orquesta/modulos/orquesta-outbox-dispatch"
	orquestaruntime "orquesta/modulos/orquesta-runtime"
)

type AgentLauncherExecutorV0 struct {
	RunStore       RunStorePortV0
	EventSink      EventSinkPortV0
	Launcher       AgentLauncherPortV0
	FailureStopper AgentStopperPortV0
	OccurredAt     string
	CorrelationID  string
	RequestedBy    string
	EvidenceRefs   []string
}

func (executor AgentLauncherExecutorV0) ExecuteOutboxDispatchV0(
	intent orquestaoutboxdispatch.DispatchIntentV0,
) (orquestaoutboxdispatch.OutboxDispatchExecutionResultV0, error) {
	intent = normalizeAgentLauncherIntentV0(intent)
	if err := executor.validateV0(intent); err != nil {
		return orquestaoutboxdispatch.OutboxDispatchExecutionResultV0{}, err
	}
	inbound, err := agentLauncherInboundFromIntentV0(intent)
	if err != nil {
		return orquestaoutboxdispatch.OutboxDispatchExecutionResultV0{}, err
	}
	result, err := executor.Launcher.LaunchAgentV0(context.Background(), inbound)
	if err != nil {
		return orquestaoutboxdispatch.OutboxDispatchExecutionResultV0{}, err
	}
	command, err := executor.agentStartedCommandV0(intent, inbound, result)
	if err != nil {
		executor.stopLaunchedAgentBestEffortV0(inbound)
		return orquestaoutboxdispatch.OutboxDispatchExecutionResultV0{}, err
	}
	workflow := storedWorkflowPortV0{
		RunStore:  executor.RunStore,
		EventSink: executor.EventSink,
	}
	if _, err := workflow.HandleWorkflowCommandV0(context.Background(), command); err != nil {
		executor.stopLaunchedAgentBestEffortV0(inbound)
		return orquestaoutboxdispatch.OutboxDispatchExecutionResultV0{}, err
	}
	return orquestaoutboxdispatch.OutboxDispatchExecutionResultV0{
		DispatchRef:  strings.TrimSpace(result.LaunchRef),
		EvidenceRefs: executor.agentLauncherEvidenceRefsV0(inbound, result),
	}, nil
}

func (executor AgentLauncherExecutorV0) validateV0(
	intent orquestaoutboxdispatch.DispatchIntentV0,
) error {
	if executor.RunStore == nil {
		return errorV0(ErrNucleoOrquestacionInvalidoV0, "run_store", "run_store requerido")
	}
	if executor.Launcher == nil {
		return errorV0(ErrNucleoOrquestacionInvalidoV0, "agent_launcher", "agent_launcher requerido")
	}
	if strings.TrimSpace(executor.OccurredAt) == "" {
		return errorV0(ErrNucleoOrquestacionInvalidoV0, "occurred_at", "occurred_at requerido")
	}
	if intent.MessageType != orquestacoreworkflow.OutboxMessageLaunchRuntimeAgentV0 {
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

func agentLauncherInboundFromIntentV0(
	intent orquestaoutboxdispatch.DispatchIntentV0,
) (orquestaruntime.AgentLauncherInboundV0, error) {
	var payload orquestaruntime.LaunchRuntimeAgentRequestV0
	if err := json.Unmarshal(intent.Payload, &payload); err != nil {
		return orquestaruntime.AgentLauncherInboundV0{},
			errorV0(ErrNucleoOrquestacionInvalidoV0, "payload", err.Error())
	}
	inbound := orquestaruntime.AgentLauncherInboundV0{
		TargetPort:     orquestaruntime.AgentLauncherTargetPortV0,
		MessageType:    orquestaruntime.AgentLauncherMessageTypeV0,
		CorrelationID:  intent.CorrelationID,
		IdempotencyKey: intent.IdempotencyKey,
		Payload:        &payload,
	}
	if issues := orquestaruntime.ValidateAgentLauncherInboundV0(inbound); len(issues) > 0 {
		return inbound, errorV0(
			ErrNucleoOrquestacionInvalidoV0,
			"agent_launcher_inbound."+issues[0].Field,
			string(issues[0].Code),
		)
	}
	return inbound, nil
}

func normalizeAgentLauncherIntentV0(
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
