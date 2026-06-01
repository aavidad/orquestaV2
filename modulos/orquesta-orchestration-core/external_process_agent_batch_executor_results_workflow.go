package orquestacionnucleoapp

import (
	"context"
	"strings"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestaoutboxdispatch "orquesta/modulos/orquesta-outbox-dispatch"
	orquestaruntime "orquesta/modulos/orquesta-runtime"
)

func (executor ExternalProcessAgentBatchExecutorV0) registerAgentProcessV0(
	ctx context.Context,
	inbound orquestaruntime.AgentLauncherInboundV0,
	snapshot orquestaruntime.ProcessRuntimeSnapshotV0,
	launch AgentLaunchResultV0,
) error {
	if executor.ProcessRegistry == nil {
		return nil
	}
	record := agentProcessRegistryRecordFromLaunchV0(inbound, snapshot, launch)
	return executor.ProcessRegistry.RecordAgentProcessV0(ctx, record)
}

func (executor ExternalProcessAgentBatchExecutorV0) registerAgentStartedV0(
	ctx context.Context,
	intent orquestaoutboxdispatch.DispatchIntentV0,
	inbound orquestaruntime.AgentLauncherInboundV0,
	launch AgentLaunchResultV0,
) error {
	launcherExecutor := AgentLauncherExecutorV0{
		RunStore:      executor.RunStore,
		EventSink:     executor.EventSink,
		OccurredAt:    executor.OccurredAt,
		CorrelationID: executor.CorrelationID,
		RequestedBy:   executor.RequestedBy,
		EvidenceRefs:  executor.EvidenceRefs,
	}
	command, err := launcherExecutor.agentStartedCommandV0(intent, inbound, launch)
	if err != nil {
		return err
	}
	workflow := storedWorkflowPortV0{
		RunStore:  executor.RunStore,
		EventSink: executor.EventSink,
	}
	_, err = workflow.HandleWorkflowCommandV0(ctx, command)
	return err
}

func (executor ExternalProcessAgentBatchExecutorV0) registerAgentFailedV0(
	ctx context.Context,
	intent orquestaoutboxdispatch.DispatchIntentV0,
	inbound orquestaruntime.AgentLauncherInboundV0,
	started orquestaruntime.ExternalAgentProcessLaunchResultV0,
) error {
	command, err := executor.agentFailedCommandV0(intent, inbound, started)
	if err != nil {
		return err
	}
	workflow := storedWorkflowPortV0{
		RunStore:  executor.RunStore,
		EventSink: executor.EventSink,
	}
	_, err = workflow.HandleWorkflowCommandV0(ctx, command)
	return err
}

func (executor ExternalProcessAgentBatchExecutorV0) agentFailedCommandV0(
	intent orquestaoutboxdispatch.DispatchIntentV0,
	inbound orquestaruntime.AgentLauncherInboundV0,
	started orquestaruntime.ExternalAgentProcessLaunchResultV0,
) (orquestacoreworkflow.OrchestrationCommandV0, error) {
	payload := orquestacoreworkflow.RegisterAgentFailedCommandPayloadV0{
		AgentRequestID: externalProcessAgentRequestIDV0(inbound),
		ReasonCode:     "agent_launch_blocked",
		Retryable:      externalProcessBatchRetryableV0(started.Issues),
		EvidenceRefs:   executor.agentFailedEvidenceRefsV0(inbound),
	}
	return orquestacoreworkflow.NewRegisterAgentFailedCommandV0(
		executor.agentFailedCommandMetaV0(intent),
		payload,
	)
}

func (executor ExternalProcessAgentBatchExecutorV0) agentFailedCommandMetaV0(
	intent orquestaoutboxdispatch.DispatchIntentV0,
) orquestacoreworkflow.OrchestrationCommandMetaV0 {
	return orquestacoreworkflow.OrchestrationCommandMetaV0{
		CommandID:      "cmd-agent-failed-" + strings.TrimSpace(intent.MessageID),
		RunID:          intent.RunID,
		IdempotencyKey: "idem-agent-failed-" + strings.TrimSpace(intent.MessageID),
		CorrelationID:  agentLauncherCorrelationIDV0(executor.CorrelationID, intent),
		RequestedBy:    agentLauncherRequestedByV0(executor.RequestedBy),
		OccurredAt:     strings.TrimSpace(executor.OccurredAt),
	}
}

func (executor ExternalProcessAgentBatchExecutorV0) agentFailedEvidenceRefsV0(
	inbound orquestaruntime.AgentLauncherInboundV0,
) []string {
	refs := append([]string(nil), executor.EvidenceRefs...)
	if inbound.Payload != nil {
		refs = append(refs, inbound.Payload.EvidenceRefs...)
	}
	refs = append(refs, "evidence-ref-external-process-batch-blocked")
	return compactStringsV0(refs)
}
