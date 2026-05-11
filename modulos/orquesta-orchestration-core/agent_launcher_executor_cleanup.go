package orquestacionnucleoapp

import (
	"context"
	"strings"

	orquestaruntime "orquesta/modulos/orquesta-runtime"
)

func (executor AgentLauncherExecutorV0) stopLaunchedAgentBestEffortV0(
	inbound orquestaruntime.AgentLauncherInboundV0,
) {
	if executor.FailureStopper == nil || inbound.Payload == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), externalProcessCleanupTimeoutV0)
	defer cancel()
	_, _ = executor.FailureStopper.StopAgentV0(
		ctx,
		agentLauncherFailureStopInboundV0(inbound),
	)
}

func agentLauncherFailureStopInboundV0(
	inbound orquestaruntime.AgentLauncherInboundV0,
) orquestaruntime.AgentStopperInboundV0 {
	payload := inbound.Payload
	return orquestaruntime.AgentStopperInboundV0{
		TargetPort:     orquestaruntime.AgentStopperTargetPortV0,
		MessageType:    orquestaruntime.AgentStopperMessageTypeV0,
		CorrelationID:  cleanupOpaqueRefV0(inbound.CorrelationID, "stop"),
		IdempotencyKey: cleanupOpaqueRefV0(inbound.IdempotencyKey, "stop"),
		Payload: &orquestaruntime.StopRuntimeAgentRequestV0{
			AgentRequestID: strings.TrimSpace(payload.AgentRequestID),
			RunID:          strings.TrimSpace(payload.RunID),
			ReasonCode:     "launch_workflow_failed",
			Summary:        "Parada por fallo al registrar AgentStarted.",
			EvidenceRefs:   []string{"evidence-ref-launch-cleanup"},
		},
	}
}

func cleanupOpaqueRefV0(value string, suffix string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return suffix
	}
	return trimmed + "-" + suffix
}
