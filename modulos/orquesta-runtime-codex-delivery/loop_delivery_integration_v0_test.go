package orquestaruntimecodexdelivery

import (
	"context"
	"path/filepath"
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectorscheduler "orquesta/modulos/orquesta-director-scheduler"
	orquestaruntimecodex "orquesta/modulos/orquesta-runtime-codex"
	orquestacionnucleoapp "orquesta/orquestacionnucleoapp"
)

func TestCodexReceiptDeliveryLoopV0RegistraEntregaDesdeACKSimulado(t *testing.T) {
	runRef := "run-ref-receipt-loop-001"
	agentRef := "agent-ref-001"
	taskRef := "task-ref-001"
	spec := codexDeliveryLoopSpecForTestV0(agentRef, taskRef)
	ackPath := filepath.Join(t.TempDir(), orquestaruntimecodex.CodexAgentAckFileNameV0)
	commandPath := filepath.Join(t.TempDir(), "agentcmd")
	workDir := t.TempDir()
	receiptStore := NewInMemoryCodexReceiptDescriptorStoreV0()
	run := codexDeliveryLoopRunForTestV0(t, runRef, taskRef)
	store := orquestacionnucleoapp.NewInMemoryRunStoreV0(run)
	sink := orquestacionnucleoapp.NewInMemoryEventSinkV0()
	ledger := orquestacionnucleoapp.NewInMemoryOutboxLedgerV0()
	runtime := &ackWritingProcessRuntimeV0{
		AckPath: ackPath,
		Ack:     codexDeliveryAckForTestV0(spec),
	}
	service := orquestacionnucleoapp.ServiceV0{
		RunStore:  store,
		EventSink: sink,
		CandidateProvider: orquestacionnucleoapp.DeliveryCandidateProviderV0{
			Base: orquestacionnucleoapp.StaticCandidateProviderV0{
				Candidates: orquestacionnucleoapp.SchedulerCandidateSetV0{
					WorkCandidates: []orquestadirectorscheduler.SchedulableWorkCandidateV0{
						codexDeliveryLoopWorkCandidateV0(runRef, agentRef, taskRef),
					},
				},
			},
			DeliverySource: CodexDeliveryObservationSourceV0{Store: receiptStore},
			RequestedBy:    "orquesta-receipt-loop-test",
		},
		OutboxLedger: ledger,
		MaxCommands:  8,
	}

	result, err := service.RunProgressiveLoopV0(context.Background(), orquestacionnucleoapp.ProgressiveLoopRequestV0{
		RunRef:               runRef,
		OccurredAt:           "2026-05-09T18:00:00Z",
		MaxBursts:            6,
		MaxStepsPerBurst:     4,
		MaxDispatchesPerWait: 2,
		CorrelationID:        "corr-receipt-loop-001",
		EvidenceRefs:         []string{"evidence-ref-receipt-loop-001"},
		Dispatchers: []orquestacionnucleoapp.OutboxDispatcherBindingV0{
			codexDeliveryCapacityDispatcherForTestV0(store, sink, ledger),
			codexDeliveryAgentDispatcherForTestV0(
				store,
				sink,
				ledger,
				receiptStore,
				runtime,
				spec,
				commandPath,
				workDir,
				ackPath,
			),
		},
	})
	if err != nil {
		t.Fatalf("RunProgressiveLoopV0: %v", err)
	}
	if result.Status != orquestacionnucleoapp.ProgressiveLoopStatusQuiescentV0 {
		t.Fatalf("status=%s result=%+v", result.Status, result)
	}
	if !codexDeliveryLoopContainsRefV0(result.Run.StartedAgents, agentRef) {
		t.Fatalf("started_agents=%v", result.Run.StartedAgents)
	}
	if !codexDeliveryLoopContainsRefV0(result.Run.Deliveries, spec.AgentPacket.DeliveryRefs.AckRef) {
		t.Fatalf("deliveries=%v", result.Run.Deliveries)
	}
	if result.PendingOutboxCount != 0 {
		t.Fatalf("pending_outbox=%d refs=%v", result.PendingOutboxCount, result.PendingOutboxRefs)
	}
	if !codexDeliveryLoopHasEventV0(sink.EventsV0(), orquestacoreworkflow.OrchestrationEventDeliveryRegisteredV0) {
		t.Fatalf("events sin DeliveryRegistered: %+v", sink.EventsV0())
	}
	descriptors, err := receiptStore.ListCodexReceiptDescriptorsV0(context.Background(), CodexReceiptDescriptorRequestV0{
		RunID:         runRef,
		StartedAgents: []string{agentRef},
		Deliveries:    result.Run.Deliveries,
	})
	if err != nil {
		t.Fatalf("ListCodexReceiptDescriptorsV0: %v", err)
	}
	if len(descriptors) != 0 {
		t.Fatalf("descriptors deben quedar filtrados por delivery registrada: %+v", descriptors)
	}
	if !runtime.Launched {
		t.Fatalf("runtime simulado no fue lanzado")
	}
}
