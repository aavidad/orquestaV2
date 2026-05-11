package orquestacionnucleoapp

import (
	"strings"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestaoutboxdispatch "orquesta/modulos/orquesta-outbox-dispatch"
	orquestaruntime "orquesta/modulos/orquesta-runtime"
)

func (executor AgentLauncherExecutorV0) agentStartedCommandV0(
	intent orquestaoutboxdispatch.DispatchIntentV0,
	inbound orquestaruntime.AgentLauncherInboundV0,
	launch AgentLaunchResultV0,
) (orquestacoreworkflow.OrchestrationCommandV0, error) {
	payload := orquestacoreworkflow.RegisterAgentStartedCommandPayloadV0{
		AgentRequestID: agentLaunchAgentRequestIDV0(inbound, launch),
		LaunchRef:      strings.TrimSpace(launch.LaunchRef),
		AckRef:         strings.TrimSpace(launch.AckRef),
		ReadinessRef:   strings.TrimSpace(launch.ReadinessRef),
		EvidenceRefs:   executor.agentLauncherEvidenceRefsV0(inbound, launch),
	}
	return orquestacoreworkflow.NewRegisterAgentStartedCommandV0(
		executor.agentStartedCommandMetaV0(intent),
		payload,
	)
}

func (executor AgentLauncherExecutorV0) agentStartedCommandMetaV0(
	intent orquestaoutboxdispatch.DispatchIntentV0,
) orquestacoreworkflow.OrchestrationCommandMetaV0 {
	return orquestacoreworkflow.OrchestrationCommandMetaV0{
		CommandID:      "cmd-agent-started-" + strings.TrimSpace(intent.MessageID),
		RunID:          intent.RunID,
		IdempotencyKey: "idem-agent-started-" + strings.TrimSpace(intent.MessageID),
		CorrelationID:  agentLauncherCorrelationIDV0(executor.CorrelationID, intent),
		RequestedBy:    agentLauncherRequestedByV0(executor.RequestedBy),
		OccurredAt:     strings.TrimSpace(executor.OccurredAt),
	}
}

func (executor AgentLauncherExecutorV0) agentLauncherEvidenceRefsV0(
	inbound orquestaruntime.AgentLauncherInboundV0,
	launch AgentLaunchResultV0,
) []string {
	refs := append([]string(nil), executor.EvidenceRefs...)
	if inbound.Payload != nil {
		refs = append(refs, inbound.Payload.EvidenceRefs...)
	}
	refs = append(refs, launch.EvidenceRefs...)
	refs = append(refs, launch.LaunchRef, launch.AckRef, launch.ReadinessRef)
	return compactStringsV0(refs)
}

func agentLaunchAgentRequestIDV0(
	inbound orquestaruntime.AgentLauncherInboundV0,
	launch AgentLaunchResultV0,
) string {
	if strings.TrimSpace(launch.AgentRequestID) != "" {
		return strings.TrimSpace(launch.AgentRequestID)
	}
	if inbound.Payload == nil {
		return ""
	}
	return strings.TrimSpace(inbound.Payload.AgentRequestID)
}

func agentLauncherCorrelationIDV0(
	configured string,
	intent orquestaoutboxdispatch.DispatchIntentV0,
) string {
	if strings.TrimSpace(configured) != "" {
		return strings.TrimSpace(configured)
	}
	return strings.TrimSpace(intent.CorrelationID)
}

func agentLauncherRequestedByV0(requestedBy string) string {
	if strings.TrimSpace(requestedBy) != "" {
		return strings.TrimSpace(requestedBy)
	}
	return "orquesta-agent-launcher"
}
