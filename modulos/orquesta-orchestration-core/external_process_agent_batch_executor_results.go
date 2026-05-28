package orquestacionnucleoapp

import (
	"context"
	"strings"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestaoutboxdispatch "orquesta/modulos/orquesta-outbox-dispatch"
	orquestaruntime "orquesta/modulos/orquesta-runtime"
)

func (executor ExternalProcessAgentBatchExecutorV0) ackBatchResultsV0(
	ctx context.Context,
	prepared externalProcessBatchPreparedV0,
	results []orquestaruntime.ExternalAgentProcessBatchResultV0,
) []orquestaoutboxdispatch.OutboxDispatchAckObservationV0 {
	acks := make([]orquestaoutboxdispatch.OutboxDispatchAckObservationV0, 0, len(results))
	for _, batchResult := range results {
		index := batchResult.Index
		if index < 0 || index >= len(prepared.Intents) {
			continue
		}
		ack := executor.ackBatchResultV0(
			ctx,
			prepared.Intents[index],
			prepared.Inbounds[index],
			prepared.Specs[index],
			batchResult.Result,
		)
		acks = append(acks, ack)
	}
	return acks
}

func (executor ExternalProcessAgentBatchExecutorV0) ackBatchResultV0(
	ctx context.Context,
	intent orquestaoutboxdispatch.DispatchIntentV0,
	inbound orquestaruntime.AgentLauncherInboundV0,
	spec orquestaruntime.ExternalAgentLaunchSpecV0,
	started orquestaruntime.ExternalAgentProcessLaunchResultV0,
) orquestaoutboxdispatch.OutboxDispatchAckObservationV0 {
	if started.Status != orquestaruntime.ExternalAgentProcessLaunchStartedV0 {
		if err := executor.registerAgentFailedV0(ctx, intent, inbound, started); err != nil {
			return failedBatchObservationWithIssuesV0(
				intent,
				"evidence-ref-external-process-batch-blocked",
				externalProcessBatchDispatchIssuesV0(started),
			)
		}
		return handledFailedAgentBatchObservationV0(intent, executor.agentFailedEvidenceRefsV0(inbound))
	}
	readiness, err := ProbeAgentReadinessV0(
		ctx,
		executor.Readiness,
		agentReadinessProbeRequestFromLaunchV0(inbound, spec, started.Snapshot),
	)
	if err != nil {
		cleanupStartedAgentProcessV0(executor.ProcessStopper, started.Snapshot)
		return failedBatchObservationV0(intent, "evidence-ref-external-process-batch-readiness")
	}
	launch := externalProcessAgentLaunchResultV0(inbound, spec, started.Snapshot)
	launch = mergeAgentReadinessEvidenceV0(launch, readiness)
	if err := executor.registerAgentProcessV0(ctx, inbound, started.Snapshot, launch); err != nil {
		cleanupStartedAgentProcessV0(executor.ProcessStopper, started.Snapshot)
		return failedBatchObservationV0(intent, "evidence-ref-external-process-batch-registry")
	}
	if err := executor.registerAgentStartedV0(ctx, intent, inbound, launch); err != nil {
		cleanupStartedAgentProcessV0(executor.ProcessStopper, started.Snapshot)
		return failedBatchObservationV0(intent, "evidence-ref-external-process-batch-workflow")
	}
	return successBatchObservationV0(intent, launch)
}

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

func handledFailedAgentBatchObservationV0(
	intent orquestaoutboxdispatch.DispatchIntentV0,
	evidenceRefs []string,
) orquestaoutboxdispatch.OutboxDispatchAckObservationV0 {
	return orquestaoutboxdispatch.OutboxDispatchAckObservationV0{
		MessageID:   intent.MessageID,
		RunID:       intent.RunID,
		TargetPort:  intent.TargetPort,
		Status:      orquestaoutboxdispatch.OutboxDispatchAckObservationSuccessV0,
		DispatchRef: "dispatch-ref-agent-failed-" + strings.TrimSpace(intent.MessageID),
		EvidenceRefs: compactStringsV0(append(
			append([]string(nil), evidenceRefs...),
			"evidence-ref-external-process-batch-failed-agent-recorded",
		)),
	}
}

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

func successBatchObservationV0(
	intent orquestaoutboxdispatch.DispatchIntentV0,
	launch AgentLaunchResultV0,
) orquestaoutboxdispatch.OutboxDispatchAckObservationV0 {
	return orquestaoutboxdispatch.OutboxDispatchAckObservationV0{
		MessageID:    intent.MessageID,
		RunID:        intent.RunID,
		TargetPort:   intent.TargetPort,
		Status:       orquestaoutboxdispatch.OutboxDispatchAckObservationSuccessV0,
		DispatchRef:  launch.LaunchRef,
		EvidenceRefs: launch.EvidenceRefs,
	}
}

func failedBatchObservationV0(
	intent orquestaoutboxdispatch.DispatchIntentV0,
	evidenceRef string,
) orquestaoutboxdispatch.OutboxDispatchAckObservationV0 {
	return failedBatchObservationWithIssuesV0(intent, evidenceRef, nil)
}

func failedBatchObservationFromErrorV0(
	intent orquestaoutboxdispatch.DispatchIntentV0,
	evidenceRef string,
	code string,
	field string,
	err error,
) orquestaoutboxdispatch.OutboxDispatchAckObservationV0 {
	message := ""
	if err != nil {
		message = strings.TrimSpace(err.Error())
	}
	return failedBatchObservationWithIssuesV0(
		intent,
		evidenceRef,
		[]orquestaoutboxdispatch.DispatchIssueV0{{
			Code:    strings.TrimSpace(code),
			Field:   strings.TrimSpace(field),
			Message: message,
		}},
	)
}

func failedBatchObservationWithIssuesV0(
	intent orquestaoutboxdispatch.DispatchIntentV0,
	evidenceRef string,
	issues []orquestaoutboxdispatch.DispatchIssueV0,
) orquestaoutboxdispatch.OutboxDispatchAckObservationV0 {
	return orquestaoutboxdispatch.OutboxDispatchAckObservationV0{
		MessageID:    intent.MessageID,
		RunID:        intent.RunID,
		TargetPort:   intent.TargetPort,
		Status:       orquestaoutboxdispatch.OutboxDispatchAckObservationFailedV0,
		EvidenceRefs: compactStringsV0([]string{evidenceRef}),
		Issues:       issues,
	}
}

func externalProcessBatchDispatchIssuesV0(
	started orquestaruntime.ExternalAgentProcessLaunchResultV0,
) []orquestaoutboxdispatch.DispatchIssueV0 {
	if len(started.Issues) == 0 {
		return []orquestaoutboxdispatch.DispatchIssueV0{{
			Code:    "external_agent_launch_blocked",
			Field:   "external_agent_process",
			Message: string(started.Status),
		}}
	}
	issues := make([]orquestaoutboxdispatch.DispatchIssueV0, 0, len(started.Issues))
	for _, issue := range started.Issues {
		code := strings.TrimSpace(string(issue.Code))
		if code == "" {
			code = "external_agent_launch_blocked"
		}
		field := strings.TrimSpace(issue.Field)
		if field == "" {
			field = "external_agent_process"
		}
		message := strings.TrimSpace(issue.MessageKey)
		if message == "" {
			message = "external_agent_launch_blocked"
		}
		issues = append(issues, orquestaoutboxdispatch.DispatchIssueV0{
			Code:    code,
			Field:   field,
			Message: message,
		})
	}
	return issues
}

func externalProcessBatchRetryableV0(
	issues []orquestaruntime.ExternalAgentConnectorErrorV0,
) bool {
	for _, issue := range issues {
		if issue.Retryable {
			return true
		}
	}
	return false
}
