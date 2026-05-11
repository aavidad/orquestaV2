package orquestadirector

import (
	"encoding/json"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestaruntime "orquesta/modulos/orquesta-runtime"
)

func progressiveLaunchInboundFromOutboxV0(
	message orquestacoreworkflow.OutboxMessageV0,
) (orquestaruntime.AgentLauncherInboundV0, error) {
	var payload orquestaruntime.LaunchRuntimeAgentRequestV0
	if err := json.Unmarshal(message.Payload, &payload); err != nil {
		return orquestaruntime.AgentLauncherInboundV0{}, progressivePayloadErrorV0(message, err)
	}
	inbound := orquestaruntime.AgentLauncherInboundV0{
		TargetPort:     message.TargetPort,
		MessageType:    message.MessageType,
		CorrelationID:  message.CorrelationID,
		IdempotencyKey: message.IdempotencyKey,
		Payload:        &payload,
	}
	if issues := orquestaruntime.ValidateAgentLauncherInboundV0(inbound); len(issues) != 0 {
		return orquestaruntime.AgentLauncherInboundV0{}, progressiveLaunchContractErrorV0(message, issues)
	}
	return inbound, nil
}

func progressiveStopInboundFromOutboxV0(
	message orquestacoreworkflow.OutboxMessageV0,
) (orquestaruntime.AgentStopperInboundV0, error) {
	var payload orquestaruntime.StopRuntimeAgentRequestV0
	if err := json.Unmarshal(message.Payload, &payload); err != nil {
		return orquestaruntime.AgentStopperInboundV0{}, progressivePayloadErrorV0(message, err)
	}
	inbound := orquestaruntime.AgentStopperInboundV0{
		TargetPort:     message.TargetPort,
		MessageType:    message.MessageType,
		CorrelationID:  message.CorrelationID,
		IdempotencyKey: message.IdempotencyKey,
		Payload:        &payload,
	}
	if issues := orquestaruntime.ValidateAgentStopperInboundV0(inbound); len(issues) != 0 {
		return orquestaruntime.AgentStopperInboundV0{}, progressiveStopContractErrorV0(message, issues)
	}
	return inbound, nil
}
