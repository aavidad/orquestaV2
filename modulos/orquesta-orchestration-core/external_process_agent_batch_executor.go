package orquestacionnucleoapp

import (
	"context"
	"strings"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestaoutboxdispatch "orquesta/modulos/orquesta-outbox-dispatch"
	orquestaruntime "orquesta/modulos/orquesta-runtime"
)

type ExternalProcessAgentBatchExecutorV0 struct {
	RunStore        RunStorePortV0
	EventSink       EventSinkPortV0
	SpecResolver    ExternalAgentLaunchSpecResolverPortV0
	Runtime         orquestaruntime.ExternalAgentProcessRuntimePortV0
	ProcessStopper  ProcessRuntimeStopPortV0
	Readiness       AgentReadinessProbePortV0
	ProcessRegistry AgentProcessRegistryPortV0
	MaxConcurrency  int
	OccurredAt      string
	CorrelationID   string
	RequestedBy     string
	EvidenceRefs    []string
}

var _ OutboxDispatchBatchExecutorPortV0 = ExternalProcessAgentBatchExecutorV0{}
var _ AutonomousBatchExecutorTunerV0 = ExternalProcessAgentBatchExecutorV0{}

func (executor ExternalProcessAgentBatchExecutorV0) WithAutonomousBatchConcurrencyV0(
	maxConcurrency int,
) OutboxDispatchBatchExecutorPortV0 {
	executor.MaxConcurrency = maxConcurrency
	return executor
}

func (executor ExternalProcessAgentBatchExecutorV0) ExecuteOutboxDispatchBatchV0(
	ctx context.Context,
	intents []orquestaoutboxdispatch.DispatchIntentV0,
) ([]orquestaoutboxdispatch.OutboxDispatchAckObservationV0, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := executor.validateV0(); err != nil {
		return nil, err
	}
	prepared := executor.prepareBatchV0(ctx, intents)
	acks := append([]orquestaoutboxdispatch.OutboxDispatchAckObservationV0(nil), prepared.Failed...)
	acks = append(acks, prepared.Closed...)
	if len(prepared.Items) == 0 {
		return acks, nil
	}
	results := orquestaruntime.LaunchExternalAgentProcessBatchV0(
		ctx,
		prepared.Items,
		executor.MaxConcurrency,
	)
	acks = append(acks, executor.ackBatchResultsV0(ctx, prepared, results)...)
	return acks, nil
}

func (executor ExternalProcessAgentBatchExecutorV0) validateV0() error {
	switch {
	case executor.RunStore == nil:
		return errorV0(ErrNucleoOrquestacionInvalidoV0, "run_store", "run_store requerido")
	case executor.SpecResolver == nil:
		return errorV0(ErrNucleoOrquestacionInvalidoV0, "external_agent_launch_spec_resolver", "external_agent_launch_spec_resolver requerido")
	case executor.Runtime == nil:
		return errorV0(ErrNucleoOrquestacionInvalidoV0, "external_agent_process_runtime", "external_agent_process_runtime requerido")
	case executor.ProcessStopper == nil:
		return errorV0(ErrNucleoOrquestacionInvalidoV0, "external_agent_process_stopper", "external_agent_process_stopper requerido")
	case executor.ProcessRegistry == nil:
		return errorV0(ErrNucleoOrquestacionInvalidoV0, "agent_process_registry", "agent_process_registry requerido")
	case strings.TrimSpace(executor.OccurredAt) == "":
		return errorV0(ErrNucleoOrquestacionInvalidoV0, "occurred_at", "occurred_at requerido")
	default:
		return nil
	}
}

func (executor ExternalProcessAgentBatchExecutorV0) prepareBatchV0(
	ctx context.Context,
	intents []orquestaoutboxdispatch.DispatchIntentV0,
) externalProcessBatchPreparedV0 {
	prepared := externalProcessBatchPreparedV0{}
	for _, raw := range intents {
		intent := normalizeAgentLauncherIntentV0(raw)
		if err := validateExternalProcessBatchIntentV0(intent); err != nil {
			prepared.Failed = append(prepared.Failed, failedBatchObservationFromErrorV0(
				intent,
				"evidence-ref-external-process-batch-invalid",
				"external_process_batch_invalid",
				"dispatch_intent",
				err,
			))
			continue
		}
		inbound, err := agentLauncherInboundFromIntentV0(intent)
		if err != nil {
			prepared.Failed = append(prepared.Failed, failedBatchObservationFromErrorV0(
				intent,
				"evidence-ref-external-process-batch-inbound",
				"external_process_batch_inbound_failed",
				"agent_launcher_inbound",
				err,
			))
			continue
		}
		if executor.agentLaunchAlreadyTerminalV0(ctx, inbound) {
			prepared.Closed = append(prepared.Closed, handledTerminalAgentBatchObservationV0(intent, inbound))
			continue
		}
		resolution, err := executor.SpecResolver.ResolveExternalAgentLaunchSpecV0(ctx, inbound)
		if err != nil {
			prepared.Failed = append(prepared.Failed, failedBatchObservationFromErrorV0(
				intent,
				"evidence-ref-external-process-batch-spec",
				"external_process_batch_spec_failed",
				"external_agent_launch_spec",
				err,
			))
			continue
		}
		prepared.Intents = append(prepared.Intents, intent)
		prepared.Inbounds = append(prepared.Inbounds, inbound)
		prepared.Specs = append(prepared.Specs, resolution.Spec)
		prepared.Items = append(prepared.Items, orquestaruntime.ExternalAgentProcessBatchItemV0{
			ItemRef:  intent.MessageID,
			Spec:     resolution.Spec,
			Resolver: resolution.CommandResolver,
			Runtime:  executor.Runtime,
		})
	}
	return prepared
}

func validateExternalProcessBatchIntentV0(
	intent orquestaoutboxdispatch.DispatchIntentV0,
) error {
	if intent.MessageType != orquestacoreworkflow.OutboxMessageLaunchRuntimeAgentV0 {
		return errorV0(ErrNucleoOrquestacionInvalidoV0, "message_type", "tipo de outbox no soportado")
	}
	if intent.TargetPort != orquestacoreworkflow.OutboxTargetAgentLauncherV0 {
		return errorV0(ErrNucleoOrquestacionInvalidoV0, "target_port", "puerto agent_launcher requerido")
	}
	if len(intent.Payload) == 0 {
		return errorV0(ErrNucleoOrquestacionInvalidoV0, "payload", "payload requerido")
	}
	return nil
}

type externalProcessBatchPreparedV0 struct {
	Intents  []orquestaoutboxdispatch.DispatchIntentV0
	Inbounds []orquestaruntime.AgentLauncherInboundV0
	Specs    []orquestaruntime.ExternalAgentLaunchSpecV0
	Items    []orquestaruntime.ExternalAgentProcessBatchItemV0
	Failed   []orquestaoutboxdispatch.OutboxDispatchAckObservationV0
	Closed   []orquestaoutboxdispatch.OutboxDispatchAckObservationV0
}
