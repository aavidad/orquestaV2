package orquestacionnucleoapp

import (
	"context"
	"errors"
	"fmt"
	"hash/fnv"
	"strings"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestaoutboxdispatch "orquesta/modulos/orquesta-outbox-dispatch"
	orquestaruntime "orquesta/modulos/orquesta-runtime"
)

const agentStopLostReasonCodeV0 = "agent_stop_process_lost"

func stopperRuntimeLostV0(err error) bool {
	var runtimeErr orquestaruntime.ProcessRuntimeErrorV0
	return errors.As(err, &runtimeErr) &&
		runtimeErr.Code == orquestaruntime.ProcessRuntimeNoEncontradoV0
}

func (executor AgentStopperExecutorV0) registerAgentLostAfterUnconfirmedStopV0(
	intent orquestaoutboxdispatch.DispatchIntentV0,
	inbound orquestaruntime.AgentStopperInboundV0,
) (orquestaoutboxdispatch.OutboxDispatchExecutionResultV0, error) {
	lossRef := agentLostRefFromStopIntentV0(intent)
	command, err := orquestacoreworkflow.NewRegisterAgentLostCommandV0(
		executor.agentLostCommandMetaV0(intent, lossRef),
		orquestacoreworkflow.RegisterAgentLostCommandPayloadV0{
			AgentRequestID: agentStopAgentRequestIDV0(inbound, AgentStopResultV0{}),
			LossRef:        lossRef,
			ReasonCode:     agentStopLostReasonCodeV0,
			ObservedAt:     strings.TrimSpace(executor.ObservedAt),
			Retryable:      false,
			EvidenceRefs:   executor.agentLostEvidenceRefsV0(inbound, lossRef),
		},
	)
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
		DispatchRef:  lossRef,
		EvidenceRefs: executor.agentLostEvidenceRefsV0(inbound, lossRef),
	}, nil
}

func (executor AgentStopperExecutorV0) agentLostCommandMetaV0(
	intent orquestaoutboxdispatch.DispatchIntentV0,
	lossRef string,
) orquestacoreworkflow.OrchestrationCommandMetaV0 {
	return orquestacoreworkflow.OrchestrationCommandMetaV0{
		CommandID:      "cmd-agent-lost-" + strings.TrimSpace(intent.MessageID),
		RunID:          intent.RunID,
		IdempotencyKey: "idem-agent-lost-" + strings.TrimSpace(lossRef),
		CorrelationID:  agentStopperCorrelationIDV0(executor.CorrelationID, intent),
		RequestedBy:    agentStopperRequestedByV0(executor.RequestedBy),
		OccurredAt:     strings.TrimSpace(executor.ObservedAt),
	}
}

func (executor AgentStopperExecutorV0) agentLostEvidenceRefsV0(
	inbound orquestaruntime.AgentStopperInboundV0,
	lossRef string,
) []string {
	refs := append([]string(nil), executor.EvidenceRefs...)
	if inbound.Payload != nil {
		refs = append(refs, inbound.Payload.EvidenceRefs...)
	}
	refs = append(refs, lossRef, "evidence-ref-agent-lost")
	return compactStringsV0(refs)
}

func agentLostRefFromStopIntentV0(
	intent orquestaoutboxdispatch.DispatchIntentV0,
) string {
	hash := fnv.New32a()
	_, _ = hash.Write([]byte(strings.TrimSpace(intent.MessageID)))
	return fmt.Sprintf("agent-loss-ref-%08x", hash.Sum32())
}
