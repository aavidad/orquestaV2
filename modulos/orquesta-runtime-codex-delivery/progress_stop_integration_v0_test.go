package orquestaruntimecodexdelivery

import (
	"context"
	"os"
	"os/signal"
	"path/filepath"
	"testing"
	"time"

	orquestaagentprocessregistrymemory "orquesta/modulos/orquesta-agent-process-registry-memory"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	orquestaoutboxdispatch "orquesta/modulos/orquesta-outbox-dispatch"
	orquestaruntime "orquesta/modulos/orquesta-runtime"
	orquestaruntimecodex "orquesta/modulos/orquesta-runtime-codex"
)

const codexProgressStopChildEnvV0 = "ORQUESTA_CODEX_PROGRESS_STOP_CHILD"

func TestMain(m *testing.M) {
	if os.Getenv(codexProgressStopChildEnvV0) == "wait" {
		codexProgressStopChildWaitV0()
	}
	os.Exit(m.Run())
}

func TestCodexProgressObservationV0NoParaProcesoRealSoloPorACKSinCambios(t *testing.T) {
	ctx := context.Background()
	runRef := "run-ref-progress-stop-001"
	spec := codexDeliverySpecForTestV0()
	run := codexProgressStopRunForTestV0(t, runRef, spec)
	store := orquestacionnucleoapp.NewInMemoryRunStoreV0(run)
	ledger := orquestacionnucleoapp.NewInMemoryOutboxLedgerV0()
	sink := orquestacionnucleoapp.NewInMemoryEventSinkV0()
	processRuntime := orquestaruntime.NewProcessRuntimeConnectorV0()
	process := codexProgressStopLaunchProcessV0(t, processRuntime)
	defer func() { _, _ = processRuntime.StopV0(context.Background(), process.ProcessRef) }()
	registry := orquestaagentprocessregistrymemory.NewInMemoryAgentProcessRegistryV0()
	codexProgressStopRecordProcessV0(t, registry, runRef, spec, process)
	ackPath := filepath.Join(t.TempDir(), orquestaruntimecodex.CodexAgentAckFileNameV0)
	receiptStore := NewInMemoryCodexReceiptDescriptorStoreV0(
		codexProgressStopDescriptorV0(runRef, spec, ackPath),
	)

	service := orquestacionnucleoapp.ServiceV0{
		RunStore:  store,
		EventSink: sink,
		CandidateProvider: orquestacionnucleoapp.ProgressSupervisionCandidateProviderV0{
			ProgressSource: CodexProgressObservationSourceV0{
				Store:           receiptStore,
				ProcessRegistry: registry,
				SnapshotSource:  processRuntime,
				State:           NewInMemoryCodexProgressStateStoreV0(),
				Policy: orquestaruntime.AgentProgressHeartbeatPolicyV0{
					StalledAfterNoProgressTicks: 99,
					LoopAfterRepeatedActions:    99,
				},
			},
			RequestedBy: "orquesta-progress-stop-test",
		},
		OutboxLedger: ledger,
		MaxCommands:  4,
	}

	result, err := service.RunManagedProgressiveLoopV0(ctx, orquestacionnucleoapp.ManagedProgressiveLoopRequestV0{
		Loop: orquestacionnucleoapp.ProgressiveLoopRequestV0{
			RunRef:               runRef,
			OccurredAt:           "2026-05-09T19:30:00Z",
			MaxBursts:            6,
			MaxStepsPerBurst:     3,
			MaxDispatchesPerWait: 2,
			CorrelationID:        "corr-progress-stop-001",
			EvidenceRefs:         []string{"evidence-ref-progress-stop-001"},
			Dispatchers: []orquestacionnucleoapp.OutboxDispatcherBindingV0{
				codexProgressStopDispatcherV0(store, sink, ledger, registry, processRuntime),
			},
		},
		ExternalWaiter:   codexProgressStopInstantWaiterV0{},
		MaxExternalWaits: 4,
	})
	if err != nil {
		t.Fatalf("RunManagedProgressiveLoopV0: %v status=%s", err, result.Status)
	}
	if result.Status != orquestacionnucleoapp.ProgressiveLoopStatusWaitExternalV0 {
		t.Fatalf("status=%s final=%+v", result.Status, result.Final)
	}
	if codexDeliveryLoopContainsRefV0(result.Final.Run.StoppedAgents, spec.RequestID) ||
		codexDeliveryLoopContainsRefV0(result.Final.Run.ConfirmedStoppedAgents, spec.RequestID) {
		t.Fatalf("agente parado sin senal de bucle real: %+v", result.Final.Run)
	}
	snapshot, err := processRuntime.SnapshotV0(process.ProcessRef)
	if err != nil {
		t.Fatalf("SnapshotV0: %v", err)
	}
	if snapshot.Status != orquestaruntime.ProcessRuntimeRunningV0 {
		t.Fatalf("proceso parado sin senal de bucle real: %+v", snapshot)
	}
	if codexDeliveryLoopHasEventV0(sink.EventsV0(), orquestacoreworkflow.OrchestrationEventAgentStopConfirmedV0) {
		t.Fatalf("AgentStopConfirmed inesperado: %+v", sink.EventsV0())
	}
	codexProgressStopAssertNoPendingAgentLauncherV0(t, ledger, runRef)
}

type codexProgressStopInstantWaiterV0 struct{}

func (codexProgressStopInstantWaiterV0) WaitExternalProgressV0(
	_ context.Context,
	_ orquestacionnucleoapp.ExternalProgressWaitRequestV0,
) (orquestacionnucleoapp.ExternalProgressWaitResultV0, error) {
	return orquestacionnucleoapp.ExternalProgressWaitResultV0{
		Continue:     true,
		EvidenceRefs: []string{"evidence-ref-progress-stop-wait-001"},
	}, nil
}

func codexProgressStopRunForTestV0(
	t *testing.T,
	runRef string,
	spec orquestaruntime.ExternalAgentLaunchSpecV0,
) orquestacoreworkflow.OrchestrationRunV0 {
	t.Helper()
	run := codexDeliveryLoopRunForTestV0(t, runRef, spec.AgentPacket.Task.TaskRef)
	run.Agents = append(run.Agents, spec.RequestID)
	run.StartedAgents = append(run.StartedAgents, spec.RequestID)
	if issues := orquestacoreworkflow.ValidateOrchestrationRunV0(run); len(issues) > 0 {
		t.Fatalf("run invalido: %+v", issues)
	}
	return run
}

func codexProgressStopLaunchProcessV0(
	t *testing.T,
	runtime *orquestaruntime.ProcessRuntimeConnectorV0,
) orquestaruntime.ProcessRuntimeSnapshotV0 {
	t.Helper()
	executable, err := os.Executable()
	if err != nil {
		t.Fatalf("test executable: %v", err)
	}
	process, err := runtime.LaunchV0(context.Background(), orquestaruntime.ProcessRuntimeLaunchRequestV0{
		CommandPath: executable,
		Env:         []string{codexProgressStopChildEnvV0 + "=wait"},
		WorkingDir:  t.TempDir(),
	})
	if err != nil {
		t.Fatalf("LaunchV0: %v", err)
	}
	if process.Status != orquestaruntime.ProcessRuntimeRunningV0 {
		t.Fatalf("process status=%q", process.Status)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_, _ = runtime.StopV0(ctx, process.ProcessRef)
	})
	return process
}

func codexProgressStopRecordProcessV0(
	t *testing.T,
	registry orquestacionnucleoapp.AgentProcessRegistryPortV0,
	runRef string,
	spec orquestaruntime.ExternalAgentLaunchSpecV0,
	process orquestaruntime.ProcessRuntimeSnapshotV0,
) {
	t.Helper()
	err := registry.RecordAgentProcessV0(context.Background(), orquestacionnucleoapp.AgentProcessRegistryRecordV0{
		RunID:          runRef,
		AgentRequestID: spec.RequestID,
		ProcessRef:     process.ProcessRef,
		SessionRef:     process.SessionRef,
		LaunchRef:      process.LaunchRef,
		ReadinessRef:   spec.AgentPacket.DeliveryRefs.ReadinessRef,
		EvidenceRefs: []string{
			process.ProcessRef,
			process.SessionRef,
			process.LaunchRef,
			spec.AgentPacket.DeliveryRefs.ReadinessRef,
		},
	})
	if err != nil {
		t.Fatalf("RecordAgentProcessV0: %v", err)
	}
}

func codexProgressStopDescriptorV0(
	runRef string,
	spec orquestaruntime.ExternalAgentLaunchSpecV0,
	ackPath string,
) CodexReceiptDescriptorV0 {
	return CodexReceiptDescriptorV0{
		DescriptorRef: "receipt-ref-progress-stop-001",
		RunID:         runRef,
		AgentRef:      spec.RequestID,
		Spec:          spec,
		AckPath:       ackPath,
	}
}

func codexProgressStopDispatcherV0(
	store *orquestacionnucleoapp.InMemoryRunStoreV0,
	sink *orquestacionnucleoapp.InMemoryEventSinkV0,
	ledger *orquestacionnucleoapp.InMemoryOutboxLedgerV0,
	registry orquestacionnucleoapp.AgentProcessRegistryPortV0,
	runtime *orquestaruntime.ProcessRuntimeConnectorV0,
) orquestacionnucleoapp.OutboxDispatcherBindingV0 {
	return orquestacionnucleoapp.OutboxDispatcherBindingV0{
		TargetPort: orquestacoreworkflow.OutboxTargetAgentLauncherV0,
		Reader:     ledger,
		Claimer:    ledger,
		Executor: orquestacionnucleoapp.AgentStopperExecutorV0{
			RunStore:   store,
			EventSink:  sink,
			Stopper:    orquestacionnucleoapp.ProcessAgentStopperV0{Registry: registry, Runtime: runtime},
			ObservedAt: "2026-05-09T19:31:00Z",
		},
		Acker: ledger,
	}
}

func codexProgressStopAssertNoPendingAgentLauncherV0(
	t *testing.T,
	ledger *orquestacionnucleoapp.InMemoryOutboxLedgerV0,
	runRef string,
) {
	t.Helper()
	pending, issues := ledger.ListPendingOutboxV0(orquestaoutboxdispatch.PendingOutboxFilterV0{
		RunID:      runRef,
		TargetPort: orquestacoreworkflow.OutboxTargetAgentLauncherV0,
	})
	if len(issues) > 0 {
		t.Fatalf("pending issues=%+v", issues)
	}
	if len(pending) != 0 {
		t.Fatalf("pending agent_launcher=%+v", pending)
	}
}

func codexProgressStopChildWaitV0() {
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt)
	defer signal.Stop(signals)

	select {
	case <-signals:
		os.Exit(0)
	case <-time.After(30 * time.Second):
		os.Exit(3)
	}
}
