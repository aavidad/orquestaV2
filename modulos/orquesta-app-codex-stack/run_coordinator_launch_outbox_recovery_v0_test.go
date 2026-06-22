package orquestaappcodexstack

import (
	"context"
	"encoding/json"
	"testing"

	orquestaagentprocessregistrymemory "orquesta/modulos/orquesta-agent-process-registry-memory"
	orquestaappdirectorservice "orquesta/modulos/orquesta-app-director-service"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	orquestaoutboxdispatch "orquesta/modulos/orquesta-outbox-dispatch"
	orquestapersistence "orquesta/modulos/orquesta-persistence"
)

func TestRecoverMissingLaunchOutboxForRequestedAgentsV0ReconstruyeOutboxTrasPersistParcial(t *testing.T) {
	ctx := context.Background()
	runRef := "run-ref-launch-outbox-recovery-001"
	runStore := orquestacionnucleoapp.NewInMemoryRunStoreV0()
	eventSink := orquestacionnucleoapp.NewInMemoryEventSinkV0()
	ledger := orquestacionnucleoapp.NewInMemoryOutboxLedgerV0()
	run := codexStackLaunchOutboxRecoveryRunV0(t, ctx, runStore, eventSink, runRef)
	stack := StackV0{
		Stores: StoresV0{
			RunStore:     runStore,
			EventSink:    eventSink,
			OutboxLedger: ledger,
		},
		Ports: orquestaappdirectorservice.StartAppDirectorPortsV0{
			RunStore:     runStore,
			EventSink:    eventSink,
			EventReader:  eventSink,
			OutboxLedger: ledger,
		},
	}

	recovered, err := stack.recoverMissingLaunchOutboxForRequestedAgentsV0(ctx, run)
	if err != nil {
		t.Fatalf("recoverMissingLaunchOutboxForRequestedAgentsV0: %v", err)
	}
	if !recovered {
		t.Fatalf("recovery no reconstruyo outbox")
	}
	pending, issues := ledger.ListPendingOutboxV0(orquestaoutboxdispatch.PendingOutboxFilterV0{
		RunID:       runRef,
		TargetPort:  orquestacoreworkflow.OutboxTargetAgentLauncherV0,
		MessageType: orquestacoreworkflow.OutboxMessageLaunchRuntimeAgentV0,
	})
	if len(issues) > 0 || len(pending) != 1 {
		t.Fatalf("pending=%+v issues=%+v", pending, issues)
	}
	if pending[0].MessageID != "outbox-launchruntimeagent-idem-agent-launch-outbox-recovery-001" {
		t.Fatalf("message_id=%s", pending[0].MessageID)
	}
}

func TestRecoverMissingCapacityOutboxForPendingCapacityRequestsV0ReconstruyeOutboxTrasPersistParcial(t *testing.T) {
	ctx := context.Background()
	runRef := "run-ref-capacity-outbox-recovery-001"
	runStore := orquestacionnucleoapp.NewInMemoryRunStoreV0()
	eventSink := orquestacionnucleoapp.NewInMemoryEventSinkV0()
	ledger := orquestacionnucleoapp.NewInMemoryOutboxLedgerV0()
	run := codexStackCapacityOutboxRecoveryRunV0(t, ctx, runStore, eventSink, runRef)
	stack := StackV0{
		Stores: StoresV0{
			RunStore:     runStore,
			EventSink:    eventSink,
			OutboxLedger: ledger,
		},
		Ports: orquestaappdirectorservice.StartAppDirectorPortsV0{
			RunStore:     runStore,
			EventSink:    eventSink,
			EventReader:  eventSink,
			OutboxLedger: ledger,
		},
	}

	recovered, err := stack.recoverMissingCapacityOutboxForPendingCapacityRequestsV0(ctx, run)
	if err != nil {
		t.Fatalf("recoverMissingCapacityOutboxForPendingCapacityRequestsV0: %v", err)
	}
	if !recovered {
		t.Fatalf("recovery no reconstruyo outbox")
	}
	pending, issues := ledger.ListPendingOutboxV0(orquestaoutboxdispatch.PendingOutboxFilterV0{
		RunID:       runRef,
		TargetPort:  orquestacoreworkflow.OutboxTargetCapacityV0,
		MessageType: orquestacoreworkflow.OutboxMessageRequestCapacityDecisionV0,
	})
	if len(issues) > 0 || len(pending) != 1 {
		t.Fatalf("pending=%+v issues=%+v", pending, issues)
	}
	if pending[0].MessageID != "outbox-requestcapacitydecision-idem-capacity-recovery-001" {
		t.Fatalf("message_id=%s", pending[0].MessageID)
	}
}

func TestReconcileOrphanCapacityOutboxForRunV0ReconstruyeRequestYLiberaClaim(t *testing.T) {
	ctx := context.Background()
	runRef := "run-ref-capacity-outbox-orphan-001"
	runStore := orquestacionnucleoapp.NewInMemoryRunStoreV0()
	eventSink := orquestacionnucleoapp.NewInMemoryEventSinkV0()
	ledger := orquestacionnucleoapp.NewInMemoryOutboxLedgerV0()
	run := codexStackStartOpenRunForOutboxRecoveryV0(t, ctx, runStore, eventSink, runRef)
	message := codexStackCapacityDecisionOutboxMessageForTestV0(
		t,
		runRef,
		"task-ref-capacity-outbox-orphan-001",
		"capacity-ref-capacity-outbox-orphan-001",
	)
	if _, issues := ledger.SavePending(ctx, []orquestacoreworkflow.OutboxMessageV0{message}); len(issues) > 0 {
		t.Fatalf("SavePending orphan: %+v", issues)
	}
	claim := orquestaoutboxdispatch.OutboxDispatchClaimV0{
		MessageID:      message.MessageID,
		RunID:          message.RunID,
		TargetPort:     message.TargetPort,
		IdempotencyKey: message.IdempotencyKey,
	}
	if claimed, issues := ledger.ClaimOutboxDispatchV0(claim); len(issues) > 0 || !claimed.Claimed {
		t.Fatalf("ClaimOutboxDispatchV0: claimed=%+v issues=%+v", claimed, issues)
	}
	stack := StackV0{
		Stores: StoresV0{
			RunStore:     runStore,
			EventSink:    eventSink,
			OutboxLedger: ledger,
		},
		Ports: stackLaunchOutboxRecoveryPortsV0(runStore, eventSink, ledger),
	}

	reconciled, err := stack.reconcileOrphanCapacityOutboxForRunV0(ctx, run, "2026-06-22T10:00:00Z")
	if err != nil {
		t.Fatalf("reconcileOrphanCapacityOutboxForRunV0: %v", err)
	}
	if !reconciled {
		t.Fatalf("orphan capacity outbox no reconciliado")
	}
	loaded, err := runStore.LoadRunV0(ctx, runRef)
	if err != nil {
		t.Fatalf("LoadRunV0: %v", err)
	}
	if !codexStackStringInSetV0(loaded.CapacityRequests, "capacity-ref-capacity-outbox-orphan-001") {
		t.Fatalf("capacity_requests=%v", loaded.CapacityRequests)
	}
	reclaimed, issues := ledger.ClaimOutboxDispatchV0(claim)
	if len(issues) > 0 || !reclaimed.Claimed || reclaimed.AlreadyClaimed {
		t.Fatalf("claim no liberado: reclaimed=%+v issues=%+v", reclaimed, issues)
	}
}

func TestReconcileOrphanCapacityOutboxForRunV0NoReutilizaIdempotencyParcial(t *testing.T) {
	ctx := context.Background()
	runRef := "run-ref-capacity-outbox-orphan-partial-001"
	runStore := orquestacionnucleoapp.NewInMemoryRunStoreV0()
	eventSink := orquestacionnucleoapp.NewInMemoryEventSinkV0()
	ledger := orquestacionnucleoapp.NewInMemoryOutboxLedgerV0()
	run := codexStackStartOpenRunForOutboxRecoveryV0(t, ctx, runStore, eventSink, runRef)
	message := codexStackCapacityDecisionOutboxMessageForTestV0(
		t,
		runRef,
		"task-ref-capacity-outbox-orphan-partial-001",
		"capacity-ref-capacity-outbox-orphan-partial-001",
	)
	originalEvent := codexStackCapacityRequestedEventFromOutboxMessageForTestV0(t, message, run.LastSequence+1)
	if err := eventSink.AppendRunEventsV0(ctx, runRef, []orquestacoreworkflow.OrchestrationEventV0{originalEvent}); err != nil {
		t.Fatalf("AppendRunEventsV0 original parcial: %v", err)
	}
	if _, issues := ledger.SavePending(ctx, []orquestacoreworkflow.OutboxMessageV0{message}); len(issues) > 0 {
		t.Fatalf("SavePending orphan: %+v", issues)
	}

	stack := StackV0{
		Stores: StoresV0{
			RunStore:     runStore,
			EventSink:    eventSink,
			OutboxLedger: ledger,
		},
		Ports: stackLaunchOutboxRecoveryPortsV0(runStore, eventSink, ledger),
	}
	reconciled, err := stack.reconcileOrphanCapacityOutboxForRunV0(ctx, run, "2026-06-22T10:00:00Z")
	if err != nil {
		t.Fatalf("reconcileOrphanCapacityOutboxForRunV0: %v", err)
	}
	if !reconciled {
		t.Fatalf("orphan capacity outbox no reconciliado")
	}
	loaded, err := runStore.LoadRunV0(ctx, runRef)
	if err != nil {
		t.Fatalf("LoadRunV0: %v", err)
	}
	if !codexStackStringInSetV0(loaded.CapacityRequests, "capacity-ref-capacity-outbox-orphan-partial-001") {
		t.Fatalf("capacity_requests=%v", loaded.CapacityRequests)
	}
	events := eventSink.EventsV0()
	if len(events) < 2 {
		t.Fatalf("events=%d, want evento original y evento reconciliado", len(events))
	}
	if events[len(events)-1].EventID == originalEvent.EventID {
		t.Fatalf("evento reconciliado reutilizo idempotency parcial: %s", events[len(events)-1].EventID)
	}
}

func TestReconcileOrphanCapacityOutboxForRunV0IncluyeClaimsPersistentesRecientes(t *testing.T) {
	ctx := context.Background()
	runRef := "run-ref-capacity-outbox-orphan-file-001"
	runStore := orquestacionnucleoapp.NewInMemoryRunStoreV0()
	eventSink := orquestacionnucleoapp.NewInMemoryEventSinkV0()
	ledger, err := orquestapersistence.NewFileOutboxLedgerV0(t.TempDir())
	if err != nil {
		t.Fatalf("NewFileOutboxLedgerV0: %v", err)
	}
	run := codexStackStartOpenRunForOutboxRecoveryV0(t, ctx, runStore, eventSink, runRef)
	message := codexStackCapacityDecisionOutboxMessageForTestV0(
		t,
		runRef,
		"task-ref-capacity-outbox-orphan-file-001",
		"capacity-ref-capacity-outbox-orphan-file-001",
	)
	if _, issues := ledger.SavePending(ctx, []orquestacoreworkflow.OutboxMessageV0{message}); len(issues) > 0 {
		t.Fatalf("SavePending orphan: %+v", issues)
	}
	claim := orquestaoutboxdispatch.OutboxDispatchClaimV0{
		MessageID:      message.MessageID,
		RunID:          message.RunID,
		TargetPort:     message.TargetPort,
		IdempotencyKey: message.IdempotencyKey,
	}
	if claimed, issues := ledger.ClaimOutboxDispatchV0(claim); len(issues) > 0 || !claimed.Claimed {
		t.Fatalf("ClaimOutboxDispatchV0: claimed=%+v issues=%+v", claimed, issues)
	}
	hiddenFromDispatcher, issues := ledger.ListPendingOutboxV0(orquestaoutboxdispatch.PendingOutboxFilterV0{
		RunID:       runRef,
		TargetPort:  orquestacoreworkflow.OutboxTargetCapacityV0,
		MessageType: orquestacoreworkflow.OutboxMessageRequestCapacityDecisionV0,
	})
	if len(issues) > 0 {
		t.Fatalf("ListPendingOutboxV0: %+v", issues)
	}
	if len(hiddenFromDispatcher) != 0 {
		t.Fatalf("ledger persistente no oculto claim reciente: %+v", hiddenFromDispatcher)
	}
	stack := StackV0{
		Stores: StoresV0{
			RunStore:     runStore,
			EventSink:    eventSink,
			OutboxLedger: ledger,
		},
		Ports: orquestaappdirectorservice.StartAppDirectorPortsV0{
			RunStore:     runStore,
			EventSink:    eventSink,
			EventReader:  eventSink,
			OutboxLedger: ledger,
		},
	}

	reconciled, err := stack.reconcileOrphanCapacityOutboxForRunV0(ctx, run, "2026-06-22T10:00:00Z")
	if err != nil {
		t.Fatalf("reconcileOrphanCapacityOutboxForRunV0: %v", err)
	}
	if !reconciled {
		t.Fatalf("orphan capacity outbox no reconciliado")
	}
	loaded, err := runStore.LoadRunV0(ctx, runRef)
	if err != nil {
		t.Fatalf("LoadRunV0: %v", err)
	}
	if !codexStackStringInSetV0(loaded.CapacityRequests, "capacity-ref-capacity-outbox-orphan-file-001") {
		t.Fatalf("capacity_requests=%v", loaded.CapacityRequests)
	}
	reclaimed, issues := ledger.ClaimOutboxDispatchV0(claim)
	if len(issues) > 0 || !reclaimed.Claimed || reclaimed.AlreadyClaimed {
		t.Fatalf("claim no liberado: reclaimed=%+v issues=%+v", reclaimed, issues)
	}
}

func TestReconcileOrphanCapacityOutboxForRunV0AckSupersededDecisionPersistente(t *testing.T) {
	ctx := context.Background()
	runRef := "run-ref-capacity-outbox-superseded-file-001"
	runStore := orquestacionnucleoapp.NewInMemoryRunStoreV0()
	eventSink := orquestacionnucleoapp.NewInMemoryEventSinkV0()
	ledger, err := orquestapersistence.NewFileOutboxLedgerV0(t.TempDir())
	if err != nil {
		t.Fatalf("NewFileOutboxLedgerV0: %v", err)
	}
	run := codexStackStartOpenRunForOutboxRecoveryV0(t, ctx, runStore, eventSink, runRef)
	capacityRef := "capacity-ref-capacity-outbox-superseded-file-001"
	requestCommand, err := orquestacoreworkflow.NewRequestCapacityCommandV0(
		launchOutboxRecoveryCommandMetaV0(runRef, "cmd-reconciled-capacity-superseded-file-001", "idem-reconciled-capacity-superseded-file-001"),
		orquestacoreworkflow.RequestCapacityCommandPayloadV0{
			CapacityRequestID:          capacityRef,
			PhaseID:                    string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0),
			TaskRef:                    "task-ref-capacity-outbox-superseded-file-001",
			ReasonCode:                 "programacion_siguiente_paso",
			Summary:                    "Capacidad ya reconciliada.",
			MinimumRecommendedCapacity: orquestacoreworkflow.OrchestrationCapacityMediumV0,
			EvidenceRefs:               []string{"evidence-ref-reconciled-capacity-superseded-file-001"},
		},
	)
	if err != nil {
		t.Fatalf("NewRequestCapacityCommandV0: %v", err)
	}
	if _, err := orquestacionnucleoapp.HandleStoredWorkflowCommandV0(ctx, runStore, eventSink, requestCommand); err != nil {
		t.Fatalf("HandleStoredWorkflowCommandV0 request: %v", err)
	}
	decisionCommand, err := orquestacoreworkflow.NewRegisterCapacityDecisionCommandV0(
		launchOutboxRecoveryCommandMetaV0(runRef, "cmd-reconciled-capacity-decision-superseded-file-001", "idem-reconciled-capacity-decision-superseded-file-001"),
		orquestacoreworkflow.RegisterCapacityDecisionCommandPayloadV0{
			CapacityRequestID: capacityRef,
			DecisionRef:       "capacity-decision-ref-reconciled-superseded-file-001",
			Tier:              orquestacoreworkflow.OrchestrationCapacityMediumV0,
			ReasoningEffort:   orquestacoreworkflow.OrchestrationCapacityMediumV0,
			Summary:           "Decisión ya reconciliada.",
			EvidenceRefs:      []string{"evidence-ref-reconciled-capacity-decision-superseded-file-001"},
		},
	)
	if err != nil {
		t.Fatalf("NewRegisterCapacityDecisionCommandV0: %v", err)
	}
	if _, err := orquestacionnucleoapp.HandleStoredWorkflowCommandV0(ctx, runStore, eventSink, decisionCommand); err != nil {
		t.Fatalf("HandleStoredWorkflowCommandV0 decision: %v", err)
	}
	run, err = runStore.LoadRunV0(ctx, runRef)
	if err != nil {
		t.Fatalf("LoadRunV0: %v", err)
	}
	message := codexStackCapacityDecisionOutboxMessageForTestV0(
		t,
		runRef,
		"task-ref-capacity-outbox-superseded-file-001",
		capacityRef,
	)
	if _, issues := ledger.SavePending(ctx, []orquestacoreworkflow.OutboxMessageV0{message}); len(issues) > 0 {
		t.Fatalf("SavePending orphan: %+v", issues)
	}
	claim := orquestaoutboxdispatch.OutboxDispatchClaimV0{
		MessageID:      message.MessageID,
		RunID:          message.RunID,
		TargetPort:     message.TargetPort,
		IdempotencyKey: message.IdempotencyKey,
	}
	if claimed, issues := ledger.ClaimOutboxDispatchV0(claim); len(issues) > 0 || !claimed.Claimed {
		t.Fatalf("ClaimOutboxDispatchV0: claimed=%+v issues=%+v", claimed, issues)
	}
	stack := StackV0{
		Stores: StoresV0{
			RunStore:     runStore,
			EventSink:    eventSink,
			OutboxLedger: ledger,
		},
		Ports: orquestaappdirectorservice.StartAppDirectorPortsV0{
			RunStore:     runStore,
			EventSink:    eventSink,
			EventReader:  eventSink,
			OutboxLedger: ledger,
		},
	}

	reconciled, err := stack.reconcileOrphanCapacityOutboxForRunV0(ctx, run, "2026-06-22T10:00:00Z")
	if err != nil {
		t.Fatalf("reconcileOrphanCapacityOutboxForRunV0: %v", err)
	}
	if !reconciled {
		t.Fatalf("outbox supersedido no reconciliado")
	}
	pending, issues := ledger.ListPendingOutboxV0(orquestaoutboxdispatch.PendingOutboxFilterV0{
		RunID:       runRef,
		TargetPort:  orquestacoreworkflow.OutboxTargetCapacityV0,
		MessageType: orquestacoreworkflow.OutboxMessageRequestCapacityDecisionV0,
	})
	if len(issues) > 0 || len(pending) != 0 {
		t.Fatalf("pending=%+v issues=%+v", pending, issues)
	}
}

func TestReconcileDurableCapacityDecisionsForRunV0ProyectaDecisionPerdida(t *testing.T) {
	ctx := context.Background()
	runRef := "run-ref-capacity-decision-durable-missing-001"
	runStore := orquestacionnucleoapp.NewInMemoryRunStoreV0()
	eventSink := orquestacionnucleoapp.NewInMemoryEventSinkV0()
	ledger := orquestacionnucleoapp.NewInMemoryOutboxLedgerV0()
	run := codexStackCapacityOutboxRecoveryRunV0(t, ctx, runStore, eventSink, runRef)
	decisionCommand, err := orquestacoreworkflow.NewRegisterCapacityDecisionCommandV0(
		launchOutboxRecoveryCommandMetaV0(runRef, "cmd-capacity-decision-durable-missing-001", "idem-capacity-decision-durable-missing-001"),
		orquestacoreworkflow.RegisterCapacityDecisionCommandPayloadV0{
			CapacityRequestID: "capacity-ref-capacity-recovery-001",
			DecisionRef:       "capacity-decision-ref-durable-missing-001",
			Tier:              orquestacoreworkflow.OrchestrationCapacityMediumV0,
			ReasoningEffort:   orquestacoreworkflow.OrchestrationCapacityHighV0,
			Summary:           "Decision duradera no proyectada.",
			EvidenceRefs:      []string{"evidence-ref-capacity-decision-durable-missing-001"},
		},
	)
	if err != nil {
		t.Fatalf("NewRegisterCapacityDecisionCommandV0: %v", err)
	}
	result, err := orquestacoreworkflow.HandleCommandV0(run, decisionCommand)
	if err != nil {
		t.Fatalf("HandleCommandV0 decision duradera: %v", err)
	}
	if len(result.Events) != 1 {
		t.Fatalf("events=%d", len(result.Events))
	}
	if err := eventSink.AppendRunEventsV0(ctx, runRef, result.Events); err != nil {
		t.Fatalf("AppendRunEventsV0 decision duradera: %v", err)
	}
	stack := StackV0{
		Stores: StoresV0{
			RunStore:     runStore,
			EventSink:    eventSink,
			OutboxLedger: ledger,
		},
		Ports: orquestaappdirectorservice.StartAppDirectorPortsV0{
			RunStore:     runStore,
			EventSink:    eventSink,
			EventReader:  eventSink,
			OutboxLedger: ledger,
		},
	}

	reconciled, err := stack.reconcileDurableCapacityDecisionsForRunV0(ctx, run)
	if err != nil {
		t.Fatalf("reconcileDurableCapacityDecisionsForRunV0: %v", err)
	}
	if !reconciled {
		t.Fatalf("decision duradera no reconciliada")
	}
	loaded, err := runStore.LoadRunV0(ctx, runRef)
	if err != nil {
		t.Fatalf("LoadRunV0: %v", err)
	}
	if !codexStackStringInSetV0(loaded.CapacityDecisions, "capacity-ref-capacity-recovery-001#capacity_decision:capacity-decision-ref-durable-missing-001") {
		t.Fatalf("capacity_decisions=%v", loaded.CapacityDecisions)
	}
	if loaded.LastSequence != result.Events[0].Sequence {
		t.Fatalf("last_sequence=%d, want %d", loaded.LastSequence, result.Events[0].Sequence)
	}
}

func TestReconcileClaimedLaunchOutboxForProcessRegistryV0RegistraStartedYAck(t *testing.T) {
	ctx := context.Background()
	runRef := "run-ref-launch-outbox-claimed-process-001"
	runStore := orquestacionnucleoapp.NewInMemoryRunStoreV0()
	eventSink := orquestacionnucleoapp.NewInMemoryEventSinkV0()
	ledger, err := orquestapersistence.NewFileOutboxLedgerV0(t.TempDir())
	if err != nil {
		t.Fatalf("NewFileOutboxLedgerV0: %v", err)
	}
	processRegistry := orquestaagentprocessregistrymemory.NewInMemoryAgentProcessRegistryV0()
	run := codexStackStartOpenRunForOutboxRecoveryV0(t, ctx, runStore, eventSink, runRef)
	capacityRef := "capacity-ref-launch-outbox-claimed-process-001"
	capacityCommand, err := orquestacoreworkflow.NewRequestCapacityCommandV0(
		launchOutboxRecoveryCommandMetaV0(runRef, "cmd-launch-claimed-capacity-001", "idem-launch-claimed-capacity-001"),
		orquestacoreworkflow.RequestCapacityCommandPayloadV0{
			CapacityRequestID:          capacityRef,
			PhaseID:                    string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0),
			TaskRef:                    "task-ref-launch-outbox-claimed-process-001",
			ReasonCode:                 "programacion_siguiente_paso",
			Summary:                    "Capacidad para agente parcial.",
			MinimumRecommendedCapacity: orquestacoreworkflow.OrchestrationCapacityMediumV0,
		},
	)
	if err != nil {
		t.Fatalf("NewRequestCapacityCommandV0: %v", err)
	}
	if _, err := orquestacionnucleoapp.HandleStoredWorkflowCommandV0(ctx, runStore, eventSink, capacityCommand); err != nil {
		t.Fatalf("HandleStoredWorkflowCommandV0 capacity: %v", err)
	}
	decisionCommand, err := orquestacoreworkflow.NewRegisterCapacityDecisionCommandV0(
		launchOutboxRecoveryCommandMetaV0(runRef, "cmd-launch-claimed-capacity-decision-001", "idem-launch-claimed-capacity-decision-001"),
		orquestacoreworkflow.RegisterCapacityDecisionCommandPayloadV0{
			CapacityRequestID: capacityRef,
			DecisionRef:       "capacity-decision-ref-launch-outbox-claimed-process-001",
			Tier:              orquestacoreworkflow.OrchestrationCapacityMediumV0,
			ReasoningEffort:   orquestacoreworkflow.OrchestrationCapacityMediumV0,
			Summary:           "Decision para agente parcial.",
		},
	)
	if err != nil {
		t.Fatalf("NewRegisterCapacityDecisionCommandV0: %v", err)
	}
	if _, err := orquestacionnucleoapp.HandleStoredWorkflowCommandV0(ctx, runStore, eventSink, decisionCommand); err != nil {
		t.Fatalf("HandleStoredWorkflowCommandV0 decision: %v", err)
	}
	run, err = runStore.LoadRunV0(ctx, runRef)
	if err != nil {
		t.Fatalf("LoadRunV0: %v", err)
	}
	agentRef := "agent-ref-launch-outbox-claimed-process-001"
	requestAgentCommand, err := orquestacoreworkflow.NewRequestAgentCommandV0(
		launchOutboxRecoveryCommandMetaV0(runRef, "cmd-agent-launch-claimed-process-001", "idem-agent-launch-claimed-process-001"),
		orquestacoreworkflow.RequestAgentCommandPayloadV0{
			AgentRequestID:     agentRef,
			PhaseID:            string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0),
			TaskRef:            "task-ref-launch-outbox-claimed-process-001",
			CapacityRequestRef: capacityRef,
			Role:               "implementacion",
			Summary:            "Agente parcial con proceso registrado.",
			EvidenceRefs:       []string{"evidence-ref-agent-launch-claimed-process-001"},
		},
	)
	if err != nil {
		t.Fatalf("NewRequestAgentCommandV0: %v", err)
	}
	result, err := orquestacoreworkflow.HandleCommandV0(run, requestAgentCommand)
	if err != nil {
		t.Fatalf("HandleCommandV0 request agent parcial: %v", err)
	}
	if len(result.Events) != 1 || len(result.Outbox) != 1 {
		t.Fatalf("request agent parcial events=%d outbox=%d", len(result.Events), len(result.Outbox))
	}
	if err := eventSink.AppendRunEventsV0(ctx, runRef, result.Events); err != nil {
		t.Fatalf("AppendRunEventsV0 request agent parcial: %v", err)
	}
	if _, issues := ledger.SavePending(ctx, result.Outbox); len(issues) > 0 {
		t.Fatalf("SavePending launch parcial: %+v", issues)
	}
	message := result.Outbox[0]
	claim := orquestaoutboxdispatch.OutboxDispatchClaimV0{
		MessageID:      message.MessageID,
		RunID:          message.RunID,
		TargetPort:     message.TargetPort,
		IdempotencyKey: message.IdempotencyKey,
	}
	if claimed, issues := ledger.ClaimOutboxDispatchV0(claim); len(issues) > 0 || !claimed.Claimed {
		t.Fatalf("ClaimOutboxDispatchV0: claimed=%+v issues=%+v", claimed, issues)
	}
	if err := processRegistry.RecordAgentProcessV0(ctx, orquestacionnucleoapp.AgentProcessRegistryRecordV0{
		RunID:          runRef,
		AgentRequestID: agentRef,
		ProcessRef:     "process-ref-launch-outbox-claimed-process-001",
		SessionRef:     "session-ref-launch-outbox-claimed-process-001",
		LaunchRef:      "launch-ref-launch-outbox-claimed-process-001",
		ReadinessRef:   "readiness-ref-launch-outbox-claimed-process-001",
		EvidenceRefs: []string{
			"process-ref-launch-outbox-claimed-process-001",
			"ack-ref-launch-outbox-claimed-process-001",
			"readiness-ref-launch-outbox-claimed-process-001",
		},
	}); err != nil {
		t.Fatalf("RecordAgentProcessV0: %v", err)
	}
	stack := StackV0{
		Stores: StoresV0{
			RunStore:        runStore,
			EventSink:       eventSink,
			OutboxLedger:    ledger,
			ProcessRegistry: processRegistry,
		},
		Ports: orquestaappdirectorservice.StartAppDirectorPortsV0{
			RunStore:     runStore,
			EventSink:    eventSink,
			EventReader:  eventSink,
			OutboxLedger: ledger,
		},
	}

	reconciled, err := stack.reconcileClaimedLaunchOutboxForProcessRegistryV0(ctx, DrainRunRequestV0{
		RunRef:        runRef,
		CorrelationID: "corr-launch-outbox-claimed-process-001",
		OccurredAt:    "2026-06-22T20:00:00Z",
	}, run)
	if err != nil {
		t.Fatalf("reconcileClaimedLaunchOutboxForProcessRegistryV0: %v", err)
	}
	if !reconciled {
		t.Fatalf("launch outbox reclamado no reconciliado")
	}
	loaded, err := runStore.LoadRunV0(ctx, runRef)
	if err != nil {
		t.Fatalf("LoadRunV0 final: %v", err)
	}
	if !codexStackStringInSetV0(loaded.Agents, agentRef) || !codexStackStringInSetV0(loaded.StartedAgents, agentRef) {
		t.Fatalf("agents=%v started=%v", loaded.Agents, loaded.StartedAgents)
	}
	pending, issues := ledger.ListPendingOutboxV0(orquestaoutboxdispatch.PendingOutboxFilterV0{
		RunID:       runRef,
		TargetPort:  orquestacoreworkflow.OutboxTargetAgentLauncherV0,
		MessageType: orquestacoreworkflow.OutboxMessageLaunchRuntimeAgentV0,
	})
	if len(issues) > 0 || len(pending) != 0 {
		t.Fatalf("pending=%+v issues=%+v", pending, issues)
	}
}

func codexStackLaunchOutboxRecoveryRunV0(
	t *testing.T,
	ctx context.Context,
	runStore *orquestacionnucleoapp.InMemoryRunStoreV0,
	eventSink *orquestacionnucleoapp.InMemoryEventSinkV0,
	runRef string,
) orquestacoreworkflow.OrchestrationRunV0 {
	t.Helper()
	run := orquestacoreworkflow.OrchestrationRunV0{}
	commands := codexStackLaunchOutboxRecoveryCommandsV0(t, runRef)
	for _, command := range commands {
		result, err := orquestacoreworkflow.HandleCommandV0(run, command)
		if err != nil {
			t.Fatalf("HandleCommandV0 %s: %v", command.CommandType, err)
		}
		for _, event := range result.Events {
			var applyErr error
			run, applyErr = orquestacoreworkflow.ApplyEventV0(run, event)
			if applyErr != nil {
				t.Fatalf("ApplyEventV0 %s: %v", event.EventType, applyErr)
			}
		}
		if len(result.Events) > 0 {
			if err := eventSink.AppendRunEventsV0(ctx, runRef, result.Events); err != nil {
				t.Fatalf("AppendRunEventsV0: %v", err)
			}
		}
	}
	if err := runStore.SaveRunV0(ctx, run); err != nil {
		t.Fatalf("SaveRunV0: %v", err)
	}
	return run
}

func codexStackCapacityOutboxRecoveryRunV0(
	t *testing.T,
	ctx context.Context,
	runStore *orquestacionnucleoapp.InMemoryRunStoreV0,
	eventSink *orquestacionnucleoapp.InMemoryEventSinkV0,
	runRef string,
) orquestacoreworkflow.OrchestrationRunV0 {
	t.Helper()
	run := orquestacoreworkflow.OrchestrationRunV0{}
	commands := codexStackCapacityOutboxRecoveryCommandsV0(t, runRef)
	for _, command := range commands {
		result, err := orquestacoreworkflow.HandleCommandV0(run, command)
		if err != nil {
			t.Fatalf("HandleCommandV0 %s: %v", command.CommandType, err)
		}
		for _, event := range result.Events {
			var applyErr error
			run, applyErr = orquestacoreworkflow.ApplyEventV0(run, event)
			if applyErr != nil {
				t.Fatalf("ApplyEventV0 %s: %v", event.EventType, applyErr)
			}
		}
		if len(result.Events) > 0 {
			if err := eventSink.AppendRunEventsV0(ctx, runRef, result.Events); err != nil {
				t.Fatalf("AppendRunEventsV0: %v", err)
			}
		}
	}
	if err := runStore.SaveRunV0(ctx, run); err != nil {
		t.Fatalf("SaveRunV0: %v", err)
	}
	return run
}

func codexStackLaunchOutboxRecoveryCommandsV0(
	t *testing.T,
	runRef string,
) []orquestacoreworkflow.OrchestrationCommandV0 {
	t.Helper()
	commands := make([]orquestacoreworkflow.OrchestrationCommandV0, 0, 5)
	command, err := orquestacoreworkflow.NewStartRunCommandV0(
		launchOutboxRecoveryCommandMetaV0(runRef, "cmd-start-launch-outbox-recovery-001", "idem-start-launch-outbox-recovery-001"),
		orquestacoreworkflow.StartRunCommandPayloadV0{
			ProjectRef: "project-ref-launch-outbox-recovery-001",
			AppSpecRef: "appspec-ref-launch-outbox-recovery-001",
		},
	)
	commands = append(commands, mustLaunchOutboxRecoveryCommandV0(t, command, err))
	command, err = orquestacoreworkflow.NewOpenPhaseCommandV0(
		launchOutboxRecoveryCommandMetaV0(runRef, "cmd-open-programacion-launch-outbox-recovery-001", "idem-open-programacion-launch-outbox-recovery-001"),
		orquestacoreworkflow.OpenPhaseCommandPayloadV0{PhaseID: string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0)},
	)
	commands = append(commands, mustLaunchOutboxRecoveryCommandV0(t, command, err))
	command, err = orquestacoreworkflow.NewRequestCapacityCommandV0(
		launchOutboxRecoveryCommandMetaV0(runRef, "cmd-capacity-launch-outbox-recovery-001", "idem-capacity-launch-outbox-recovery-001"),
		orquestacoreworkflow.RequestCapacityCommandPayloadV0{
			CapacityRequestID:          "capacity-ref-launch-outbox-recovery-001",
			PhaseID:                    string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0),
			TaskRef:                    "task-ref-launch-outbox-recovery-001",
			ReasonCode:                 "work_profile_implementation",
			Summary:                    "Capacidad para implementar tarea acotada.",
			MinimumRecommendedCapacity: orquestacoreworkflow.OrchestrationCapacityHighV0,
		},
	)
	commands = append(commands, mustLaunchOutboxRecoveryCommandV0(t, command, err))
	command, err = orquestacoreworkflow.NewRegisterCapacityDecisionCommandV0(
		launchOutboxRecoveryCommandMetaV0(runRef, "cmd-capacity-decision-launch-outbox-recovery-001", "idem-capacity-decision-launch-outbox-recovery-001"),
		orquestacoreworkflow.RegisterCapacityDecisionCommandPayloadV0{
			CapacityRequestID: "capacity-ref-launch-outbox-recovery-001",
			DecisionRef:       "capacity-decision-launch-outbox-recovery-001",
			Tier:              orquestacoreworkflow.OrchestrationCapacityHighV0,
			ReasoningEffort:   orquestacoreworkflow.OrchestrationCapacityHighV0,
			Summary:           "Capacidad aceptada.",
		},
	)
	commands = append(commands, mustLaunchOutboxRecoveryCommandV0(t, command, err))
	command, err = orquestacoreworkflow.NewRequestAgentCommandV0(
		launchOutboxRecoveryCommandMetaV0(runRef, "cmd-agent-launch-outbox-recovery-001", "idem-agent-launch-outbox-recovery-001"),
		orquestacoreworkflow.RequestAgentCommandPayloadV0{
			AgentRequestID:     "agent-ref-launch-outbox-recovery-001",
			PhaseID:            string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0),
			TaskRef:            "task-ref-launch-outbox-recovery-001",
			CapacityRequestRef: "capacity-ref-launch-outbox-recovery-001",
			Role:               "implementacion",
			Summary:            "Microtarea acotada lista.",
			EvidenceRefs:       []string{"evidence-ref-agent-launch-outbox-recovery-001"},
			SkillRefs:          []string{"skill-ref-orquesta-programacion-integracion-v0"},
		},
	)
	commands = append(commands, mustLaunchOutboxRecoveryCommandV0(t, command, err))
	return commands
}

func codexStackStartOpenRunForOutboxRecoveryV0(
	t *testing.T,
	ctx context.Context,
	runStore *orquestacionnucleoapp.InMemoryRunStoreV0,
	eventSink *orquestacionnucleoapp.InMemoryEventSinkV0,
	runRef string,
) orquestacoreworkflow.OrchestrationRunV0 {
	t.Helper()
	run := orquestacoreworkflow.OrchestrationRunV0{}
	commands := codexStackCapacityOutboxRecoveryCommandsV0(t, runRef)[:2]
	for _, command := range commands {
		result, err := orquestacoreworkflow.HandleCommandV0(run, command)
		if err != nil {
			t.Fatalf("HandleCommandV0 %s: %v", command.CommandType, err)
		}
		for _, event := range result.Events {
			var applyErr error
			run, applyErr = orquestacoreworkflow.ApplyEventV0(run, event)
			if applyErr != nil {
				t.Fatalf("ApplyEventV0 %s: %v", event.EventType, applyErr)
			}
		}
		if len(result.Events) > 0 {
			if err := eventSink.AppendRunEventsV0(ctx, runRef, result.Events); err != nil {
				t.Fatalf("AppendRunEventsV0: %v", err)
			}
		}
	}
	if err := runStore.SaveRunV0(ctx, run); err != nil {
		t.Fatalf("SaveRunV0: %v", err)
	}
	return run
}

func codexStackCapacityDecisionOutboxMessageForTestV0(
	t *testing.T,
	runRef string,
	taskRef string,
	capacityRef string,
) orquestacoreworkflow.OutboxMessageV0 {
	t.Helper()
	payload, err := json.Marshal(orquestacoreworkflow.CapacityDecisionRequestV0{
		CapacityRequestID:          capacityRef,
		RunID:                      runRef,
		PhaseID:                    string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0),
		TaskRef:                    taskRef,
		ReasonCode:                 "programacion_siguiente_paso",
		Summary:                    "Capacidad para microtarea acotada.",
		MinimumRecommendedCapacity: orquestacoreworkflow.OrchestrationCapacityMediumV0,
		EvidenceRefs:               []string{"evidence-ref-capacity-outbox-orphan-001"},
	})
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	message := orquestacoreworkflow.OutboxMessageV0{
		MessageID:        "outbox-requestcapacitydecision-idem-capacity-outbox-orphan-001",
		MessageType:      orquestacoreworkflow.OutboxMessageRequestCapacityDecisionV0,
		RunID:            runRef,
		IdempotencyKey:   "idem-capacity-outbox-orphan-001",
		CorrelationID:    "corr-capacity-outbox-orphan-001",
		CausationEventID: "evt-capacityrequested-idem-capacity-outbox-orphan-001",
		TargetPort:       orquestacoreworkflow.OutboxTargetCapacityV0,
		PayloadVersion:   orquestacoreworkflow.OutboxPayloadVersionV0,
		Payload:          payload,
	}
	if err := orquestacoreworkflow.ValidateOutboxMessageV0(message); err != nil {
		t.Fatalf("ValidateOutboxMessageV0: %v", err)
	}
	return message
}

func codexStackCapacityRequestedEventFromOutboxMessageForTestV0(
	t *testing.T,
	message orquestacoreworkflow.OutboxMessageV0,
	sequence int64,
) orquestacoreworkflow.OrchestrationEventV0 {
	t.Helper()
	var payload orquestacoreworkflow.CapacityDecisionRequestV0
	if err := json.Unmarshal(message.Payload, &payload); err != nil {
		t.Fatalf("unmarshal capacity decision request: %v", err)
	}
	event, err := orquestacoreworkflow.NewCapacityRequestedEventV0(
		orquestacoreworkflow.OrchestrationEventMetaV0{
			EventID:        "evt-capacityrequested-" + message.IdempotencyKey,
			RunID:          message.RunID,
			Sequence:       sequence,
			IdempotencyKey: message.IdempotencyKey,
			CorrelationID:  message.CorrelationID,
			CausationID:    "cmd-" + message.IdempotencyKey,
			OccurredAt:     "2026-06-22T10:00:00Z",
		},
		orquestacoreworkflow.CapacityRequestedPayloadV0{
			CapacityRequestID:          payload.CapacityRequestID,
			PhaseID:                    payload.PhaseID,
			TaskRef:                    payload.TaskRef,
			ReasonCode:                 payload.ReasonCode,
			Summary:                    payload.Summary,
			MinimumRecommendedCapacity: payload.MinimumRecommendedCapacity,
			EvidenceRefs:               append([]string(nil), payload.EvidenceRefs...),
		},
	)
	if err != nil {
		t.Fatalf("NewCapacityRequestedEventV0: %v", err)
	}
	return event
}

func codexStackCapacityOutboxRecoveryCommandsV0(
	t *testing.T,
	runRef string,
) []orquestacoreworkflow.OrchestrationCommandV0 {
	t.Helper()
	commands := make([]orquestacoreworkflow.OrchestrationCommandV0, 0, 3)
	command, err := orquestacoreworkflow.NewStartRunCommandV0(
		launchOutboxRecoveryCommandMetaV0(runRef, "cmd-start-capacity-recovery-001", "idem-start-capacity-recovery-001"),
		orquestacoreworkflow.StartRunCommandPayloadV0{
			ProjectRef: "project-ref-capacity-recovery-001",
			AppSpecRef: "appspec-ref-capacity-recovery-001",
		},
	)
	commands = append(commands, mustLaunchOutboxRecoveryCommandV0(t, command, err))
	command, err = orquestacoreworkflow.NewOpenPhaseCommandV0(
		launchOutboxRecoveryCommandMetaV0(runRef, "cmd-open-programacion-capacity-recovery-001", "idem-open-programacion-capacity-recovery-001"),
		orquestacoreworkflow.OpenPhaseCommandPayloadV0{PhaseID: string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0)},
	)
	commands = append(commands, mustLaunchOutboxRecoveryCommandV0(t, command, err))
	command, err = orquestacoreworkflow.NewRequestCapacityCommandV0(
		launchOutboxRecoveryCommandMetaV0(runRef, "cmd-capacity-recovery-001", "idem-capacity-recovery-001"),
		orquestacoreworkflow.RequestCapacityCommandPayloadV0{
			CapacityRequestID:          "capacity-ref-capacity-recovery-001",
			PhaseID:                    string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0),
			TaskRef:                    "task-ref-capacity-recovery-001",
			ReasonCode:                 "work_profile_implementation",
			Summary:                    "Capacidad para implementar tarea acotada.",
			MinimumRecommendedCapacity: orquestacoreworkflow.OrchestrationCapacityHighV0,
		},
	)
	commands = append(commands, mustLaunchOutboxRecoveryCommandV0(t, command, err))
	return commands
}

func stackLaunchOutboxRecoveryPortsV0(
	runStore *orquestacionnucleoapp.InMemoryRunStoreV0,
	eventSink *orquestacionnucleoapp.InMemoryEventSinkV0,
	ledger *orquestacionnucleoapp.InMemoryOutboxLedgerV0,
) orquestaappdirectorservice.StartAppDirectorPortsV0 {
	return orquestaappdirectorservice.StartAppDirectorPortsV0{
		RunStore:     runStore,
		EventSink:    eventSink,
		EventReader:  eventSink,
		OutboxLedger: ledger,
	}
}

func launchOutboxRecoveryCommandMetaV0(
	runRef string,
	commandID string,
	idempotencyKey string,
) orquestacoreworkflow.OrchestrationCommandMetaV0 {
	return orquestacoreworkflow.OrchestrationCommandMetaV0{
		CommandID:      commandID,
		RunID:          runRef,
		IdempotencyKey: idempotencyKey,
		CorrelationID:  "corr-launch-outbox-recovery-001",
		RequestedBy:    "test-launch-outbox-recovery",
		OccurredAt:     "2026-06-13T10:00:00Z",
	}
}

func mustLaunchOutboxRecoveryCommandV0(
	t *testing.T,
	command orquestacoreworkflow.OrchestrationCommandV0,
	err error,
) orquestacoreworkflow.OrchestrationCommandV0 {
	t.Helper()
	if err != nil {
		t.Fatalf("command: %v", err)
	}
	return command
}
