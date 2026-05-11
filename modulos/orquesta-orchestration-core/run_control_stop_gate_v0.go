package orquestacionnucleoapp

import (
	"context"
	"strings"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
)

func (service ServiceV0) stopRunControlAgentsV0(
	ctx context.Context,
	request ProgressiveLoopRequestV0,
	gate progressiveRunControlGateV0,
) (progressiveDispatchWaitResultV0, error) {
	if err := service.materializeRunControlStopOutboxV0(ctx, request, gate); err != nil {
		return progressiveDispatchWaitResultV0{}, err
	}
	stopRequest := request
	stopRequest.Dispatchers = runControlStopDispatchersV0(request.Dispatchers)
	stopRequest.BatchDispatchers = runControlStopBatchDispatchersV0(request.BatchDispatchers)
	return service.dispatchProgressiveWaitV0(ctx, stopRequest)
}

func (service ServiceV0) materializeRunControlStopOutboxV0(
	ctx context.Context,
	request ProgressiveLoopRequestV0,
	gate progressiveRunControlGateV0,
) error {
	run, err := service.RunStore.LoadRunV0(ctx, request.RunRef)
	if err != nil {
		return err
	}
	for _, agentRef := range runControlStoppableAgentRefsV0(run) {
		if err := service.materializeRunControlStopAgentV0(ctx, request, gate, agentRef); err != nil {
			return err
		}
	}
	return nil
}

func (service ServiceV0) materializeRunControlStopAgentV0(
	ctx context.Context,
	request ProgressiveLoopRequestV0,
	gate progressiveRunControlGateV0,
	agentRef string,
) error {
	command, err := orquestacoreworkflow.NewStopAgentCommandV0(
		runControlStopCommandMetaV0(request, agentRef),
		orquestacoreworkflow.StopAgentCommandPayloadV0{
			AgentRequestID: agentRef,
			ReasonCode:     runControlStopReasonCodeV0(gate),
			Summary:        "Parada solicitada por control de run.",
			EvidenceRefs:   runControlStopEvidenceRefsV0(agentRef),
		},
	)
	if err != nil {
		return err
	}
	result, err := HandleStoredWorkflowCommandV0(ctx, service.RunStore, service.EventSink, command)
	if err != nil {
		return err
	}
	if len(result.Outbox) == 0 {
		return nil
	}
	if _, issues := service.OutboxLedger.SavePending(ctx, result.Outbox); len(issues) > 0 {
		return errorV0(ErrNucleoOrquestacionStoreV0, "outbox_ledger", issues[0].Code)
	}
	return nil
}

func runControlStoppableAgentRefsV0(
	run orquestacoreworkflow.OrchestrationRunV0,
) []string {
	failed := autonomousStringSetV0(run.FailedAgents)
	stopped := autonomousStringSetV0(run.StoppedAgents)
	confirmed := autonomousStringSetV0(run.ConfirmedStoppedAgents)
	out := make([]string, 0, len(run.StartedAgents))
	for _, agentRef := range compactStringsV0(run.StartedAgents) {
		if failed[agentRef] || stopped[agentRef] || confirmed[agentRef] {
			continue
		}
		out = append(out, agentRef)
	}
	return out
}

func runControlStopCommandMetaV0(
	request ProgressiveLoopRequestV0,
	agentRef string,
) orquestacoreworkflow.OrchestrationCommandMetaV0 {
	suffix := runControlStopSafeRefPartV0(agentRef)
	return orquestacoreworkflow.OrchestrationCommandMetaV0{
		CommandID:      "cmd-run-control-stop-" + suffix,
		RunID:          request.RunRef,
		IdempotencyKey: "idem-run-control-stop-" + suffix,
		CorrelationID:  request.CorrelationID,
		RequestedBy:    "orquesta-run-control",
		OccurredAt:     request.OccurredAt,
	}
}

func runControlStopReasonCodeV0(gate progressiveRunControlGateV0) string {
	if strings.TrimSpace(gate.ReasonCode) != "" {
		return strings.TrimSpace(gate.ReasonCode)
	}
	return "run_stop_requested"
}

func runControlStopEvidenceRefsV0(agentRef string) []string {
	return []string{"evidence-ref-run-control-stop-" + runControlStopSafeRefPartV0(agentRef)}
}

func runControlStopDispatchersV0(
	dispatchers []OutboxDispatcherBindingV0,
) []OutboxDispatcherBindingV0 {
	out := make([]OutboxDispatcherBindingV0, 0, len(dispatchers))
	for _, dispatcher := range dispatchers {
		if strings.TrimSpace(dispatcher.MessageType) != orquestacoreworkflow.OutboxMessageStopRuntimeAgentV0 {
			continue
		}
		out = append(out, dispatcher)
	}
	return out
}

func runControlStopBatchDispatchersV0(
	dispatchers []OutboxBatchDispatcherBindingV0,
) []OutboxBatchDispatcherBindingV0 {
	out := make([]OutboxBatchDispatcherBindingV0, 0, len(dispatchers))
	for _, dispatcher := range dispatchers {
		if strings.TrimSpace(dispatcher.MessageType) != orquestacoreworkflow.OutboxMessageStopRuntimeAgentV0 {
			continue
		}
		out = append(out, dispatcher)
	}
	return out
}

func runControlStopSafeRefPartV0(value string) string {
	value = strings.TrimSpace(value)
	value = strings.ReplaceAll(value, "\\", "-")
	value = strings.ReplaceAll(value, "/", "-")
	value = strings.ReplaceAll(value, " ", "-")
	return value
}
