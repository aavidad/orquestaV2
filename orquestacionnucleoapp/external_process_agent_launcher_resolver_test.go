package orquestacionnucleoapp

import (
	"context"
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestaruntime "orquesta/modulos/orquesta-runtime"
)

func externalProcessAgentDispatcherForTestV0(
	store *InMemoryRunStoreV0,
	sink *InMemoryEventSinkV0,
	ledger *InMemoryOutboxLedgerV0,
	runtime *recordingExternalProcessRuntimeV0,
	t *testing.T,
) OutboxDispatcherBindingV0 {
	t.Helper()
	registry := NewInMemoryAgentProcessRegistryV0()
	return OutboxDispatcherBindingV0{
		TargetPort: orquestacoreworkflow.OutboxTargetAgentLauncherV0,
		Reader:     ledger,
		Claimer:    ledger,
		Executor: AgentLauncherExecutorV0{
			RunStore:  store,
			EventSink: sink,
			Launcher: &ExternalProcessAgentLauncherV0{
				SpecResolver: externalAgentLaunchSpecResolverForTestV0(
					externalProcessRuntimeRequestForTestV0(t, "exit"),
				),
				Runtime:         runtime,
				ProcessStopper:  runtime,
				ProcessRegistry: registry,
			},
			FailureStopper: ProcessAgentStopperV0{Registry: registry, Runtime: runtime},
			OccurredAt:     "2026-05-08T11:02:00Z",
		},
		Acker: ledger,
	}
}

type externalProcessCommandResolverForTestV0 struct {
	processRequest orquestaruntime.ProcessRuntimeLaunchRequestV0
}

func (resolver externalProcessCommandResolverForTestV0) ResolveExternalAgentProcessCommandV0(
	context.Context,
	orquestaruntime.ExternalAgentLaunchSpecV0,
) (orquestaruntime.ProcessRuntimeLaunchRequestV0, []orquestaruntime.ExternalAgentConnectorErrorV0) {
	return resolver.processRequest, nil
}

type recordingExternalProcessRuntimeV0 struct {
	connector    *orquestaruntime.ProcessRuntimeConnectorV0
	lastRequest  orquestaruntime.ProcessRuntimeLaunchRequestV0
	lastSnapshot orquestaruntime.ProcessRuntimeSnapshotV0
}

func (runtime *recordingExternalProcessRuntimeV0) LaunchV0(
	ctx context.Context,
	request orquestaruntime.ProcessRuntimeLaunchRequestV0,
) (orquestaruntime.ProcessRuntimeSnapshotV0, error) {
	runtime.lastRequest = request
	snapshot, err := runtime.connector.LaunchV0(ctx, request)
	runtime.lastSnapshot = snapshot
	return snapshot, err
}

func (runtime *recordingExternalProcessRuntimeV0) StopV0(
	ctx context.Context,
	processRef string,
) (orquestaruntime.ProcessRuntimeSnapshotV0, error) {
	return runtime.connector.StopV0(ctx, processRef)
}
