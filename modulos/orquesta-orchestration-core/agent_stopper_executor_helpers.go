package orquestacionnucleoapp

import (
	"fmt"
	"hash/fnv"
	"strings"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestaoutboxdispatch "orquesta/modulos/orquesta-outbox-dispatch"
	orquestaruntime "orquesta/modulos/orquesta-runtime"
)

func (executor AgentStopperExecutorV0) agentStopConfirmedCommandV0(
	intent orquestaoutboxdispatch.DispatchIntentV0,
	inbound orquestaruntime.AgentStopperInboundV0,
	stop AgentStopResultV0,
) (orquestacoreworkflow.OrchestrationCommandV0, error) {
	payload := orquestacoreworkflow.RegisterAgentStopConfirmedCommandPayloadV0{
		ConfirmationRef: agentStopConfirmationRefV0(intent, stop),
		AgentRequestID:  agentStopAgentRequestIDV0(inbound, stop),
		ObservedAt:      strings.TrimSpace(executor.ObservedAt),
		Summary:         agentStopConfirmedSummaryV0(executor.Summary),
		EvidenceRefs:    executor.agentStopperEvidenceRefsV0(inbound, stop),
	}
	return orquestacoreworkflow.NewRegisterAgentStopConfirmedCommandV0(
		executor.agentStopConfirmedCommandMetaV0(intent),
		payload,
	)
}

func (executor AgentStopperExecutorV0) agentStopConfirmedCommandMetaV0(
	intent orquestaoutboxdispatch.DispatchIntentV0,
) orquestacoreworkflow.OrchestrationCommandMetaV0 {
	messageRef := agentStopperOpaqueControlRefV0("agent-stop-message", intent.MessageID)
	return orquestacoreworkflow.OrchestrationCommandMetaV0{
		CommandID:      "cmd-agent-stop-confirmed-" + messageRef,
		RunID:          intent.RunID,
		IdempotencyKey: "idem-agent-stop-confirmed-" + messageRef,
		CorrelationID:  agentStopperCorrelationIDV0(executor.CorrelationID, intent),
		RequestedBy:    agentStopperRequestedByV0(executor.RequestedBy),
		OccurredAt:     strings.TrimSpace(executor.ObservedAt),
	}
}

func (executor AgentStopperExecutorV0) agentStopperEvidenceRefsV0(
	inbound orquestaruntime.AgentStopperInboundV0,
	stop AgentStopResultV0,
) []string {
	refs := append([]string(nil), executor.EvidenceRefs...)
	if inbound.Payload != nil {
		refs = append(refs, inbound.Payload.EvidenceRefs...)
	}
	refs = append(refs, stop.EvidenceRefs...)
	refs = append(refs, stop.ConfirmationRef)
	return compactStringsV0(refs)
}

func agentStopAgentRequestIDV0(
	inbound orquestaruntime.AgentStopperInboundV0,
	stop AgentStopResultV0,
) string {
	if inbound.Payload != nil && strings.TrimSpace(inbound.Payload.AgentRequestID) != "" {
		return strings.TrimSpace(inbound.Payload.AgentRequestID)
	}
	return strings.TrimSpace(stop.AgentRequestID)
}

func agentStopConfirmationRefV0(
	intent orquestaoutboxdispatch.DispatchIntentV0,
	stop AgentStopResultV0,
) string {
	if strings.TrimSpace(stop.ConfirmationRef) != "" {
		return strings.TrimSpace(stop.ConfirmationRef)
	}
	hash := fnv.New32a()
	_, _ = hash.Write([]byte(strings.TrimSpace(intent.MessageID)))
	return fmt.Sprintf("agent-stop-confirmation-ref-%08x", hash.Sum32())
}

func agentStopConfirmedSummaryV0(summary string) string {
	if strings.TrimSpace(summary) != "" {
		return strings.TrimSpace(summary)
	}
	return "Parada confirmada."
}

func agentStopperCorrelationIDV0(
	configured string,
	intent orquestaoutboxdispatch.DispatchIntentV0,
) string {
	if strings.TrimSpace(configured) != "" {
		return agentStopperOpaqueControlRefV0("corr-agent-stop", configured)
	}
	return agentStopperOpaqueControlRefV0("corr-agent-stop", intent.CorrelationID)
}

func agentStopperRequestedByV0(requestedBy string) string {
	if strings.TrimSpace(requestedBy) != "" {
		return strings.TrimSpace(requestedBy)
	}
	return "orquesta-agent-stopper"
}

func agentStopperOpaqueControlRefV0(prefix string, value string) string {
	trimmed := strings.TrimSpace(value)
	prefix = strings.TrimSpace(prefix)
	if prefix == "" {
		prefix = "agent-stop-ref"
	}
	if trimmed == "" {
		return prefix + "-empty"
	}
	if !agentStopperControlRefNeedsCompactionV0(trimmed) {
		return trimmed
	}
	hash := fnv.New64a()
	_, _ = hash.Write([]byte(trimmed))
	return fmt.Sprintf("%s-%016x", prefix, hash.Sum64())
}

func agentStopperControlRefNeedsCompactionV0(value string) bool {
	if len(value) < 3 || len(value) > 159 {
		return true
	}
	for _, r := range value {
		if (r >= 'A' && r <= 'Z') ||
			(r >= 'a' && r <= 'z') ||
			(r >= '0' && r <= '9') ||
			r == '.' || r == '_' || r == ':' || r == '-' {
			continue
		}
		return true
	}
	return false
}
