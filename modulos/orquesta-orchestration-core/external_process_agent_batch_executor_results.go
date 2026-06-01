package orquestacionnucleoapp

import (
	"context"

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
