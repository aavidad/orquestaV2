package orquestacionnucleoapp

import (
	"context"
	"strings"

	orquestaoutboxdispatch "orquesta/modulos/orquesta-outbox-dispatch"
	orquestaruntime "orquesta/modulos/orquesta-runtime"
)

func (executor ExternalProcessAgentBatchExecutorV0) agentLaunchAlreadyTerminalV0(
	ctx context.Context,
	inbound orquestaruntime.AgentLauncherInboundV0,
) bool {
	if executor.RunStore == nil || inbound.Payload == nil {
		return false
	}
	runID := strings.TrimSpace(inbound.Payload.RunID)
	agentRef := strings.TrimSpace(inbound.Payload.AgentRequestID)
	if runID == "" || agentRef == "" {
		return false
	}
	run, err := executor.RunStore.LoadRunV0(ctx, runID)
	if err != nil {
		return false
	}
	return stringInSetV0(agentRef, run.DeliveredAgents) ||
		stringInSetV0(agentRef, run.FailedAgents) ||
		stringInSetV0(agentRef, run.LostAgents) ||
		stringInSetV0(agentRef, run.StoppedAgents) ||
		stringInSetV0(agentRef, run.ConfirmedStoppedAgents)
}

func handledTerminalAgentBatchObservationV0(
	intent orquestaoutboxdispatch.DispatchIntentV0,
	inbound orquestaruntime.AgentLauncherInboundV0,
) orquestaoutboxdispatch.OutboxDispatchAckObservationV0 {
	evidenceRefs := []string{
		"evidence-ref-external-process-batch-terminal-agent-preserved",
	}
	if inbound.Payload != nil {
		evidenceRefs = append(evidenceRefs, inbound.Payload.EvidenceRefs...)
	}
	return orquestaoutboxdispatch.OutboxDispatchAckObservationV0{
		MessageID:    intent.MessageID,
		RunID:        intent.RunID,
		TargetPort:   intent.TargetPort,
		Status:       orquestaoutboxdispatch.OutboxDispatchAckObservationSuccessV0,
		DispatchRef:  "dispatch-ref-terminal-agent-" + strings.TrimSpace(intent.MessageID),
		EvidenceRefs: compactStringsV0(evidenceRefs),
	}
}
