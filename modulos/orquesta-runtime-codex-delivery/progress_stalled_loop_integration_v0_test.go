package orquestaruntimecodexdelivery

import (
	"context"
	"path/filepath"
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestaoutboxdispatch "orquesta/modulos/orquesta-outbox-dispatch"
	orquestaruntime "orquesta/modulos/orquesta-runtime"
	orquestaruntimecodex "orquesta/modulos/orquesta-runtime-codex"
	orquestacionnucleoapp "orquesta/orquestacionnucleoapp"
)

func TestCodexProgressObservationV0StalledEscalaYParaPorBucle(t *testing.T) {
	ctx := context.Background()
	runRef := "run-ref-progress-stalled-loop-001"
	spec := codexDeliverySpecForTestV0()
	run := codexProgressStopRunForTestV0(t, runRef, spec)
	store := orquestacionnucleoapp.NewInMemoryRunStoreV0(run)
	ledger := orquestacionnucleoapp.NewInMemoryOutboxLedgerV0()
	sink := orquestacionnucleoapp.NewInMemoryEventSinkV0()
	processRuntime := orquestaruntime.NewProcessRuntimeConnectorV0()
	process := codexProgressStopLaunchProcessV0(t, processRuntime)
	defer func() { _, _ = processRuntime.StopV0(context.Background(), process.ProcessRef) }()
	registry := orquestacionnucleoapp.NewInMemoryAgentProcessRegistryV0()
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
					StalledAfterNoProgressTicks: 1,
					LoopAfterRepeatedActions:    2,
				},
			},
			RequestedBy: "orquesta-progress-stalled-loop-test",
		},
		OutboxLedger: ledger,
		MaxCommands:  4,
	}

	result, err := service.RunManagedProgressiveLoopV0(ctx, orquestacionnucleoapp.ManagedProgressiveLoopRequestV0{
		Loop: orquestacionnucleoapp.ProgressiveLoopRequestV0{
			RunRef:               runRef,
			OccurredAt:           "2026-05-09T19:40:00Z",
			MaxBursts:            6,
			MaxStepsPerBurst:     3,
			MaxDispatchesPerWait: 3,
			CorrelationID:        "corr-progress-stalled-loop-001",
			EvidenceRefs:         []string{"evidence-ref-progress-stalled-loop-001"},
			Dispatchers: []orquestacionnucleoapp.OutboxDispatcherBindingV0{
				codexProgressStopDispatcherV0(store, sink, ledger, registry, processRuntime),
				codexProgressDirectorQuestionDispatcherV0(ledger),
			},
		},
		ExternalWaiter:   codexProgressStopInstantWaiterV0{},
		MaxExternalWaits: 3,
	})
	if err != nil {
		t.Fatalf("RunManagedProgressiveLoopV0: %v status=%s", err, result.Status)
	}
	if result.Status != orquestacionnucleoapp.ProgressiveLoopStatusQuiescentV0 {
		t.Fatalf("status=%s final=%+v", result.Status, result.Final)
	}
	if !codexDeliveryLoopHasEventV0(sink.EventsV0(), orquestacoreworkflow.OrchestrationEventDirectorQuestionRaisedV0) {
		t.Fatalf("sin DirectorQuestionRaised: %+v", sink.EventsV0())
	}
	if !codexDeliveryLoopHasEventV0(sink.EventsV0(), orquestacoreworkflow.OrchestrationEventAgentStopConfirmedV0) {
		t.Fatalf("sin AgentStopConfirmed: %+v", sink.EventsV0())
	}
	if !codexDeliveryLoopContainsRefV0(result.Final.Run.ConfirmedStoppedAgents, spec.RequestID) {
		t.Fatalf("agente no confirmado como parado: %+v", result.Final.Run)
	}
	codexProgressAssertNoPendingTargetV0(t, ledger, runRef, orquestacoreworkflow.OutboxTargetDirectorV0)
	codexProgressAssertNoPendingTargetV0(t, ledger, runRef, orquestacoreworkflow.OutboxTargetAgentLauncherV0)
}

func codexProgressDirectorQuestionDispatcherV0(
	ledger *orquestacionnucleoapp.InMemoryOutboxLedgerV0,
) orquestacionnucleoapp.OutboxDispatcherBindingV0 {
	return orquestacionnucleoapp.OutboxDispatcherBindingV0{
		TargetPort: orquestacoreworkflow.OutboxTargetDirectorV0,
		Reader:     ledger,
		Claimer:    ledger,
		Executor:   codexProgressDirectorQuestionAckExecutorV0{},
		Acker:      ledger,
	}
}

type codexProgressDirectorQuestionAckExecutorV0 struct{}

func (codexProgressDirectorQuestionAckExecutorV0) ExecuteOutboxDispatchV0(
	intent orquestaoutboxdispatch.DispatchIntentV0,
) (orquestaoutboxdispatch.OutboxDispatchExecutionResultV0, error) {
	return orquestaoutboxdispatch.OutboxDispatchExecutionResultV0{
		DispatchRef:  "dispatch-ref-" + intent.MessageID,
		EvidenceRefs: []string{"evidence-ref-director-question-dispatched-001"},
	}, nil
}

func codexProgressAssertNoPendingTargetV0(
	t *testing.T,
	ledger *orquestacionnucleoapp.InMemoryOutboxLedgerV0,
	runRef string,
	targetPort string,
) {
	t.Helper()
	pending, issues := ledger.ListPendingOutboxV0(orquestaoutboxdispatch.PendingOutboxFilterV0{
		RunID:      runRef,
		TargetPort: targetPort,
	})
	if len(issues) > 0 {
		t.Fatalf("pending issues=%+v", issues)
	}
	if len(pending) != 0 {
		t.Fatalf("pending %s=%+v", targetPort, pending)
	}
}
