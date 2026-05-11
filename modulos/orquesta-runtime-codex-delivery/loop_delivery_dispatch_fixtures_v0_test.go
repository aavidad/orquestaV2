package orquestaruntimecodexdelivery

import (
	"context"
	"encoding/json"
	"os"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestaruntime "orquesta/modulos/orquesta-runtime"
	orquestaruntimecodex "orquesta/modulos/orquesta-runtime-codex"
	orquestacionnucleoapp "orquesta/orquestacionnucleoapp"
)

func codexDeliveryCapacityDispatcherForTestV0(
	store orquestacionnucleoapp.RunStorePortV0,
	sink orquestacionnucleoapp.EventSinkPortV0,
	ledger *orquestacionnucleoapp.InMemoryOutboxLedgerV0,
) orquestacionnucleoapp.OutboxDispatcherBindingV0 {
	return orquestacionnucleoapp.OutboxDispatcherBindingV0{
		TargetPort: orquestacoreworkflow.OutboxTargetCapacityV0,
		Reader:     ledger,
		Claimer:    ledger,
		Executor: orquestacionnucleoapp.CapacityDecisionExecutorV0{
			RunStore:        store,
			EventSink:       sink,
			Tier:            orquestacoreworkflow.OrchestrationCapacityMediumV0,
			ReasoningEffort: orquestacoreworkflow.OrchestrationCapacityMediumV0,
			OccurredAt:      "2026-05-09T18:00:30Z",
			RequestedBy:     "orquesta-receipt-loop-test",
			Summary:         "Capacidad validada para prueba de receipt.",
			EvidenceRefs:    []string{"evidence-ref-receipt-capacity-001"},
		},
		Acker: ledger,
	}
}

func codexDeliveryAgentDispatcherForTestV0(
	store orquestacionnucleoapp.RunStorePortV0,
	sink orquestacionnucleoapp.EventSinkPortV0,
	ledger *orquestacionnucleoapp.InMemoryOutboxLedgerV0,
	receiptStore *InMemoryCodexReceiptDescriptorStoreV0,
	runtime *ackWritingProcessRuntimeV0,
	spec orquestaruntime.ExternalAgentLaunchSpecV0,
	commandPath string,
	workDir string,
	ackPath string,
) orquestacionnucleoapp.OutboxDispatcherBindingV0 {
	specResolver := CodexReceiptRecordingSpecResolverV0{
		Inner: staticExternalAgentSpecResolverV0{
			Spec: spec,
			CommandResolver: staticProcessCommandResolverV0{
				CommandPath: commandPath,
				WorkingDir:  workDir,
			},
		},
		Recorder:        receiptStore,
		AckPathResolver: StaticCodexReceiptAckPathResolverV0{AckPath: ackPath},
	}
	registry := orquestacionnucleoapp.NewInMemoryAgentProcessRegistryV0()
	return orquestacionnucleoapp.OutboxDispatcherBindingV0{
		TargetPort: orquestacoreworkflow.OutboxTargetAgentLauncherV0,
		Reader:     ledger,
		Claimer:    ledger,
		Executor: orquestacionnucleoapp.AgentLauncherExecutorV0{
			RunStore:  store,
			EventSink: sink,
			Launcher: orquestacionnucleoapp.ExternalProcessAgentLauncherV0{
				SpecResolver:    specResolver,
				Runtime:         runtime,
				ProcessStopper:  runtime,
				ProcessRegistry: registry,
			},
			FailureStopper: orquestacionnucleoapp.ProcessAgentStopperV0{Registry: registry, Runtime: runtime},
			OccurredAt:     "2026-05-09T18:01:00Z",
			RequestedBy:    "orquesta-receipt-loop-test",
			EvidenceRefs: []string{
				"evidence-ref-receipt-agent-001",
			},
		},
		Acker: ledger,
	}
}

type staticProcessCommandResolverV0 struct {
	CommandPath string
	WorkingDir  string
}

func (resolver staticProcessCommandResolverV0) ResolveExternalAgentProcessCommandV0(
	context.Context,
	orquestaruntime.ExternalAgentLaunchSpecV0,
) (orquestaruntime.ProcessRuntimeLaunchRequestV0, []orquestaruntime.ExternalAgentConnectorErrorV0) {
	return orquestaruntime.ProcessRuntimeLaunchRequestV0{
		CommandPath: resolver.CommandPath,
		Env:         []string{},
		WorkingDir:  resolver.WorkingDir,
	}, nil
}

func (runtime *ackWritingProcessRuntimeV0) StopV0(
	_ context.Context,
	processRef string,
) (orquestaruntime.ProcessRuntimeSnapshotV0, error) {
	return orquestaruntime.ProcessRuntimeSnapshotV0{
		SchemaVersion: orquestaruntime.ProcessRuntimeConnectorVersionV0,
		ProcessRef:    processRef,
		SessionRef:    "session-ref-receipt-loop-001",
		LaunchRef:     "launch-ref-receipt-loop-001",
		StopRef:       "stop-ref-receipt-loop-001",
		Status:        orquestaruntime.ProcessRuntimeStoppedV0,
	}, nil
}

type ackWritingProcessRuntimeV0 struct {
	AckPath  string
	Ack      orquestaruntimecodex.CodexAgentAckV0
	Launched bool
}

func (runtime *ackWritingProcessRuntimeV0) LaunchV0(
	_ context.Context,
	_ orquestaruntime.ProcessRuntimeLaunchRequestV0,
) (orquestaruntime.ProcessRuntimeSnapshotV0, error) {
	data, err := json.Marshal(runtime.Ack)
	if err != nil {
		return orquestaruntime.ProcessRuntimeSnapshotV0{}, err
	}
	if err := os.WriteFile(runtime.AckPath, data, 0o600); err != nil {
		return orquestaruntime.ProcessRuntimeSnapshotV0{}, err
	}
	runtime.Launched = true
	return orquestaruntime.ProcessRuntimeSnapshotV0{
		SchemaVersion: orquestaruntime.ProcessRuntimeConnectorVersionV0,
		ProcessRef:    "process-ref-receipt-loop-001",
		SessionRef:    "session-ref-receipt-loop-001",
		LaunchRef:     "launch-ref-receipt-loop-001",
		Status:        orquestaruntime.ProcessRuntimeRunningV0,
	}, nil
}
