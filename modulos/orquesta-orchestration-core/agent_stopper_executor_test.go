package orquestacionnucleoapp

import (
	"context"
	"strings"
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectorscheduler "orquesta/modulos/orquesta-director-scheduler"
	orquestaoutboxdispatch "orquesta/modulos/orquesta-outbox-dispatch"
	orquestaruntime "orquesta/modulos/orquesta-runtime"
)

func TestAgentStopperExecutorV0ConfirmsStopAndAcksOutbox(t *testing.T) {
	runRef := "run-nucleo-agent-stopper-001"
	store := NewInMemoryRunStoreV0(mustActiveProgrammingRunV0(t, runRef))
	ledger := NewInMemoryOutboxLedgerV0()
	sink := NewInMemoryEventSinkV0()
	runtime := orquestaruntime.NewRuntimeFakeLifecycleV0()
	service := ServiceV0{
		RunStore:  store,
		EventSink: sink,
		CandidateProvider: StaticCandidateProviderV0{Candidates: SchedulerCandidateSetV0{
			WorkCandidates: []orquestadirectorscheduler.SchedulableWorkCandidateV0{
				workCandidateWithAgentV0(runRef),
			},
		}},
		OutboxLedger: ledger,
		MaxCommands:  4,
	}

	launched := runProgressiveLoopWithFakeAgentRuntimeV0(t, service, store, sink, ledger, runtime, runRef)
	if !containsNucleoRefV0(launched.Run.StartedAgents, "agent-ref-001") {
		t.Fatalf("started agents=%v", launched.Run.StartedAgents)
	}
	saveStopOutboxV0(t, store, ledger, launched.Run, "agent-ref-001")

	dispatch, err := RunOutboxDispatchOnceV0(context.Background(), OutboxDispatchOnceRequestV0{
		RunRef:     runRef,
		TargetPort: orquestacoreworkflow.OutboxTargetAgentLauncherV0,
		Reader:     ledger,
		Claimer:    ledger,
		Executor: AgentStopperExecutorV0{
			RunStore:   store,
			EventSink:  sink,
			Stopper:    &FakeLifecycleAgentStopperV0{Runtime: runtime},
			ObservedAt: "2026-05-08T10:40:00Z",
		},
		Acker: ledger,
	})
	if err != nil {
		t.Fatalf("dispatch stop: %v", err)
	}
	if dispatch.Status != "dispatched" {
		t.Fatalf("dispatch=%+v", dispatch)
	}
	if dispatch.DispatchRef == "" {
		t.Fatalf("dispatch ref vacio")
	}

	finalRun, err := store.LoadRunV0(context.Background(), runRef)
	if err != nil {
		t.Fatalf("load final run: %v", err)
	}
	if !containsNucleoRefV0(finalRun.ConfirmedStoppedAgents, "agent-ref-001") {
		t.Fatalf("confirmed stopped agents=%v", finalRun.ConfirmedStoppedAgents)
	}
	if pending := pendingOutboxRefsForTargetV0(t, ledger, runRef, orquestacoreworkflow.OutboxTargetAgentLauncherV0); len(pending) != 0 {
		t.Fatalf("pending agent_launcher=%v", pending)
	}
	if !sinkHasEventTypeV0(sink, orquestacoreworkflow.OrchestrationEventAgentStopConfirmedV0) {
		t.Fatalf("sink no contiene AgentStopConfirmed: %+v", sink.EventsV0())
	}
}

func TestAgentStopperExecutorV0MarcaLostSiStopNoPuedeConfirmarProcess(t *testing.T) {
	runRef := "run-nucleo-agent-stopper-lost-001"
	store := NewInMemoryRunStoreV0(mustActiveProgrammingRunV0(t, runRef))
	ledger := NewInMemoryOutboxLedgerV0()
	sink := NewInMemoryEventSinkV0()
	runtime := orquestaruntime.NewRuntimeFakeLifecycleV0()
	service := ServiceV0{
		RunStore:  store,
		EventSink: sink,
		CandidateProvider: StaticCandidateProviderV0{Candidates: SchedulerCandidateSetV0{
			WorkCandidates: []orquestadirectorscheduler.SchedulableWorkCandidateV0{
				workCandidateWithAgentV0(runRef),
			},
		}},
		OutboxLedger: ledger,
		MaxCommands:  4,
	}

	launched := runProgressiveLoopWithFakeAgentRuntimeV0(t, service, store, sink, ledger, runtime, runRef)
	saveStopOutboxV0(t, store, ledger, launched.Run, "agent-ref-001")

	dispatch, err := RunOutboxDispatchOnceV0(context.Background(), OutboxDispatchOnceRequestV0{
		RunRef:     runRef,
		TargetPort: orquestacoreworkflow.OutboxTargetAgentLauncherV0,
		Reader:     ledger,
		Claimer:    ledger,
		Executor: AgentStopperExecutorV0{
			RunStore:   store,
			EventSink:  sink,
			Stopper:    lostAgentStopperForTestV0{},
			ObservedAt: "2026-05-08T10:45:00Z",
		},
		Acker: ledger,
	})
	if err != nil {
		t.Fatalf("dispatch lost stop: %v", err)
	}
	if dispatch.Status != "dispatched" || dispatch.DispatchRef == "" {
		t.Fatalf("dispatch=%+v", dispatch)
	}
	finalRun, err := store.LoadRunV0(context.Background(), runRef)
	if err != nil {
		t.Fatalf("load final run: %v", err)
	}
	if !containsNucleoRefV0(finalRun.LostAgents, "agent-ref-001") ||
		containsNucleoRefV0(finalRun.ConfirmedStoppedAgents, "agent-ref-001") {
		t.Fatalf("lost=%v confirmed=%v", finalRun.LostAgents, finalRun.ConfirmedStoppedAgents)
	}
	if pending := pendingOutboxRefsForTargetV0(t, ledger, runRef, orquestacoreworkflow.OutboxTargetAgentLauncherV0); len(pending) != 0 {
		t.Fatalf("pending agent_launcher=%v", pending)
	}
	if !sinkHasEventTypeV0(sink, orquestacoreworkflow.OrchestrationEventAgentLostV0) ||
		sinkHasEventTypeV0(sink, orquestacoreworkflow.OrchestrationEventAgentStopConfirmedV0) {
		t.Fatalf("sink events=%+v", sink.EventsV0())
	}
}

func TestAgentStopperExecutorV0ConsumeParadaObsoletaSiAgenteYaEntrego(t *testing.T) {
	runRef := "run-nucleo-agent-stopper-delivered-001"
	agentRef := "agent-ref-delivered-001"
	run := mustActiveProgrammingRunV0(t, runRef)
	run.Agents = []string{agentRef}
	run.StartedAgents = []string{agentRef}
	run.DeliveredAgents = []string{agentRef}
	store := NewInMemoryRunStoreV0(run)
	ledger := NewInMemoryOutboxLedgerV0()
	sink := NewInMemoryEventSinkV0()
	if _, issues := ledger.SavePending(context.Background(), []orquestacoreworkflow.OutboxMessageV0{
		stopRuntimeAgentOutboxForStopperTestV0(runRef, agentRef),
	}); len(issues) > 0 {
		t.Fatalf("save stop outbox issues=%+v", issues)
	}

	dispatch, err := RunOutboxDispatchOnceV0(context.Background(), OutboxDispatchOnceRequestV0{
		RunRef:     runRef,
		TargetPort: orquestacoreworkflow.OutboxTargetAgentLauncherV0,
		Reader:     ledger,
		Claimer:    ledger,
		Executor: AgentStopperExecutorV0{
			RunStore:   store,
			EventSink:  sink,
			Stopper:    successfulAgentStopperForTestV0{},
			ObservedAt: "2026-05-08T10:50:00Z",
		},
		Acker: ledger,
	})
	if err != nil {
		t.Fatalf("dispatch obsolete stop: %v", err)
	}
	if dispatch.Status != "dispatched" || dispatch.DispatchRef == "" {
		t.Fatalf("dispatch=%+v", dispatch)
	}
	if pending := pendingOutboxRefsForTargetV0(t, ledger, runRef, orquestacoreworkflow.OutboxTargetAgentLauncherV0); len(pending) != 0 {
		t.Fatalf("pending agent_launcher=%v", pending)
	}
	finalRun, err := store.LoadRunV0(context.Background(), runRef)
	if err != nil {
		t.Fatalf("load final run: %v", err)
	}
	if containsNucleoRefV0(finalRun.ConfirmedStoppedAgents, agentRef) ||
		containsNucleoRefV0(finalRun.LostAgents, agentRef) {
		t.Fatalf("no debia mutar terminales: confirmed=%v lost=%v", finalRun.ConfirmedStoppedAgents, finalRun.LostAgents)
	}
}

func TestAgentStopperExecutorV0ConsumeParadaObsoletaSiAgenteYaEntregoYRuntimeNoExiste(t *testing.T) {
	runRef := "run-nucleo-agent-stopper-delivered-lost-001"
	agentRef := "agent-ref-delivered-lost-001"
	run := mustActiveProgrammingRunV0(t, runRef)
	run.Agents = []string{agentRef}
	run.StartedAgents = []string{agentRef}
	run.DeliveredAgents = []string{agentRef}
	store := NewInMemoryRunStoreV0(run)
	ledger := NewInMemoryOutboxLedgerV0()
	sink := NewInMemoryEventSinkV0()
	if _, issues := ledger.SavePending(context.Background(), []orquestacoreworkflow.OutboxMessageV0{
		stopRuntimeAgentOutboxForStopperTestV0(runRef, agentRef),
	}); len(issues) > 0 {
		t.Fatalf("save stop outbox issues=%+v", issues)
	}

	dispatch, err := RunOutboxDispatchOnceV0(context.Background(), OutboxDispatchOnceRequestV0{
		RunRef:     runRef,
		TargetPort: orquestacoreworkflow.OutboxTargetAgentLauncherV0,
		Reader:     ledger,
		Claimer:    ledger,
		Executor: AgentStopperExecutorV0{
			RunStore:   store,
			EventSink:  sink,
			Stopper:    lostAgentStopperForTestV0{},
			ObservedAt: "2026-05-08T10:51:00Z",
		},
		Acker: ledger,
	})
	if err != nil {
		t.Fatalf("dispatch obsolete lost stop: %v", err)
	}
	if dispatch.Status != "dispatched" || dispatch.DispatchRef == "" {
		t.Fatalf("dispatch=%+v", dispatch)
	}
	if pending := pendingOutboxRefsForTargetV0(t, ledger, runRef, orquestacoreworkflow.OutboxTargetAgentLauncherV0); len(pending) != 0 {
		t.Fatalf("pending agent_launcher=%v", pending)
	}
	finalRun, err := store.LoadRunV0(context.Background(), runRef)
	if err != nil {
		t.Fatalf("load final run: %v", err)
	}
	if containsNucleoRefV0(finalRun.LostAgents, agentRef) {
		t.Fatalf("no debia marcar perdido un agente ya reflejado por entrega: lost=%v", finalRun.LostAgents)
	}
}

func TestAgentStopperExecutorV0RejectsNonStopRuntimeAgentOutbox(t *testing.T) {
	executor := AgentStopperExecutorV0{
		RunStore:   NewInMemoryRunStoreV0(),
		Stopper:    NewFakeLifecycleAgentStopperV0(),
		ObservedAt: "2026-05-08T10:41:00Z",
	}

	_, err := executor.ExecuteOutboxDispatchV0(orquestacoreworkflowIntentV0(
		orquestacoreworkflow.OutboxMessageLaunchRuntimeAgentV0,
		orquestacoreworkflow.OutboxTargetAgentLauncherV0,
	))
	if err == nil {
		t.Fatalf("esperaba error")
	}
	got, ok := err.(ErrorV0)
	if !ok || got.Field != "message_type" {
		t.Fatalf("error=%+v", err)
	}
}

func TestAgentStopperExecutorV0KeepsWorkflowAgentRequestID(t *testing.T) {
	inbound := orquestaruntime.AgentStopperInboundV0{
		Payload: &orquestaruntime.StopRuntimeAgentRequestV0{
			AgentRequestID: "agent-ref-workflow-001",
		},
	}
	got := agentStopAgentRequestIDV0(inbound, AgentStopResultV0{
		AgentRequestID: "agent-ref-adapter-wrong-001",
	})

	if got != "agent-ref-workflow-001" {
		t.Fatalf("agent_request_id=%q", got)
	}
}

func TestAgentStopperInboundFromIntentV0CompactaRefsInternasLargas(t *testing.T) {
	intent := orquestacoreworkflowIntentV0(
		orquestacoreworkflow.OutboxMessageStopRuntimeAgentV0,
		orquestacoreworkflow.OutboxTargetAgentLauncherV0,
	)
	intent.MessageID = "outbox-stop-runtime-agent-" + strings.Repeat("x", 180)
	intent.IdempotencyKey = "idem-progress-supervision-" + strings.Repeat("y", 180)
	intent.CorrelationID = "corr-progress-supervision-" + strings.Repeat("z", 180)
	intent.Payload = []byte(`{
		"agent_request_id":"agent-ref-stopper-long-001",
		"run_id":"run-test-agent-stopper-001",
		"reason_code":"loop_detected",
		"summary":"Parar agente solicitado por supervision.",
		"evidence_refs":["evidence-ref-stop-agent-long-001"]
	}`)

	inbound, err := agentStopperInboundFromIntentV0(intent)
	if err != nil {
		t.Fatalf("agentStopperInboundFromIntentV0: %v", err)
	}
	if inbound.IdempotencyKey == intent.IdempotencyKey ||
		inbound.CorrelationID == intent.CorrelationID ||
		len(inbound.IdempotencyKey) > 159 ||
		len(inbound.CorrelationID) > 159 {
		t.Fatalf("refs stopper no compactadas: idem=%q corr=%q", inbound.IdempotencyKey, inbound.CorrelationID)
	}
	if issues := orquestaruntime.ValidateAgentStopperInboundV0(inbound); len(issues) > 0 {
		t.Fatalf("inbound invalido: %+v", issues)
	}
	meta := (AgentStopperExecutorV0{}).agentStopConfirmedCommandMetaV0(intent)
	if len(meta.CommandID) > 159 || len(meta.IdempotencyKey) > 159 {
		t.Fatalf("meta stopper no compactada: %+v", meta)
	}
}

type lostAgentStopperForTestV0 struct{}

func (lostAgentStopperForTestV0) StopAgentV0(
	context.Context,
	orquestaruntime.AgentStopperInboundV0,
) (AgentStopResultV0, error) {
	return AgentStopResultV0{}, orquestaruntime.ProcessRuntimeErrorV0{
		Code:       orquestaruntime.ProcessRuntimeNoEncontradoV0,
		MessageKey: "process.no_encontrado",
		Field:      "process_ref",
		Retryable:  false,
	}
}

type successfulAgentStopperForTestV0 struct{}

func (successfulAgentStopperForTestV0) StopAgentV0(
	_ context.Context,
	inbound orquestaruntime.AgentStopperInboundV0,
) (AgentStopResultV0, error) {
	agentRef := ""
	if inbound.Payload != nil {
		agentRef = strings.TrimSpace(inbound.Payload.AgentRequestID)
	}
	return AgentStopResultV0{
		AgentRequestID:  agentRef,
		ConfirmationRef: "agent-stop-confirmation-ref-delivered-001",
		EvidenceRefs:    []string{"evidence-ref-stop-delivered-001"},
	}, nil
}

func runProgressiveLoopWithFakeAgentRuntimeV0(
	t *testing.T,
	service ServiceV0,
	store *InMemoryRunStoreV0,
	sink *InMemoryEventSinkV0,
	ledger *InMemoryOutboxLedgerV0,
	runtime *orquestaruntime.RuntimeFakeLifecycleV0,
	runRef string,
) ProgressiveLoopResultV0 {
	t.Helper()
	result, err := service.RunProgressiveLoopV0(context.Background(), ProgressiveLoopRequestV0{
		RunRef:               runRef,
		OccurredAt:           "2026-05-08T10:35:00Z",
		MaxBursts:            4,
		MaxStepsPerBurst:     3,
		MaxDispatchesPerWait: 2,
		CorrelationID:        "corr-agent-stopper-loop-001",
		EvidenceRefs:         []string{"evidence-ref-agent-stopper-loop-001"},
		Dispatchers: []OutboxDispatcherBindingV0{
			capacityDecisionDispatcherForTestV0(store, sink, ledger),
			{
				TargetPort: orquestacoreworkflow.OutboxTargetAgentLauncherV0,
				Reader:     ledger,
				Claimer:    ledger,
				Executor: AgentLauncherExecutorV0{
					RunStore:   store,
					EventSink:  sink,
					Launcher:   &FakeLifecycleAgentLauncherV0{Runtime: runtime},
					OccurredAt: "2026-05-08T10:36:00Z",
				},
				Acker: ledger,
			},
		},
	})
	if err != nil {
		t.Fatalf("progressive loop: %v", err)
	}
	return result
}

func saveStopOutboxV0(
	t *testing.T,
	store *InMemoryRunStoreV0,
	ledger *InMemoryOutboxLedgerV0,
	run orquestacoreworkflow.OrchestrationRunV0,
	agentRequestID string,
) {
	t.Helper()
	command, err := orquestacoreworkflow.NewStopAgentCommandV0(
		commandMetaV0(run.RunID, "cmd-stop-agent-stopper-001", "idem-stop-agent-stopper-001"),
		orquestacoreworkflow.StopAgentCommandPayloadV0{
			AgentRequestID: agentRequestID,
			ReasonCode:     "loop_detected",
			Summary:        "Parar agente solicitado.",
			EvidenceRefs:   []string{"evidence-ref-stop-agent-001"},
		},
	)
	if err != nil {
		t.Fatalf("stop command: %v", err)
	}
	result, err := orquestacoreworkflow.HandleCommandV0(run, command)
	if err != nil {
		t.Fatalf("handle stop: %v", err)
	}
	next, err := applyCommandEventsV0(run, result.Events)
	if err != nil {
		t.Fatalf("apply stop: %v", err)
	}
	if err := store.SaveRunV0(context.Background(), next); err != nil {
		t.Fatalf("save stop run: %v", err)
	}
	if len(result.Outbox) != 1 || result.Outbox[0].MessageType != orquestacoreworkflow.OutboxMessageStopRuntimeAgentV0 {
		t.Fatalf("stop outbox=%+v", result.Outbox)
	}
	if _, issues := ledger.SavePending(context.Background(), result.Outbox); len(issues) > 0 {
		t.Fatalf("save stop outbox issues=%+v", issues)
	}
}

func pendingOutboxRefsForTargetV0(
	t *testing.T,
	ledger *InMemoryOutboxLedgerV0,
	runRef string,
	targetPort string,
) []string {
	t.Helper()
	pending, issues := ledger.ListPendingOutboxV0(orquestaoutboxdispatch.PendingOutboxFilterV0{
		RunID:      runRef,
		TargetPort: targetPort,
	})
	if len(issues) > 0 {
		t.Fatalf("pending issues=%+v", issues)
	}
	refs := make([]string, 0, len(pending))
	for _, entry := range pending {
		refs = append(refs, entry.MessageID)
	}
	return refs
}

func sinkHasEventTypeV0(sink *InMemoryEventSinkV0, eventType string) bool {
	for _, event := range sink.EventsV0() {
		if event.EventType == eventType {
			return true
		}
	}
	return false
}

func orquestacoreworkflowIntentV0(
	messageType string,
	targetPort string,
) orquestaoutboxdispatch.DispatchIntentV0 {
	return orquestaoutboxdispatch.DispatchIntentV0{
		MessageID:      "outbox-test-agent-stopper-001",
		RunID:          "run-test-agent-stopper-001",
		TargetPort:     targetPort,
		MessageType:    messageType,
		IdempotencyKey: "idem-test-agent-stopper-001",
		CorrelationID:  "corr-test-agent-stopper-001",
		PayloadVersion: orquestacoreworkflow.OutboxPayloadVersionV0,
		Payload:        []byte(`{}`),
	}
}

func stopRuntimeAgentOutboxForStopperTestV0(
	runRef string,
	agentRef string,
) orquestacoreworkflow.OutboxMessageV0 {
	return orquestacoreworkflow.OutboxMessageV0{
		MessageID:      "outbox-stopruntimeagent-test-" + agentRef,
		MessageType:    orquestacoreworkflow.OutboxMessageStopRuntimeAgentV0,
		RunID:          runRef,
		IdempotencyKey: "idem-stopruntimeagent-test-" + agentRef,
		CorrelationID:  "corr-stopruntimeagent-test-" + agentRef,
		TargetPort:     orquestacoreworkflow.OutboxTargetAgentLauncherV0,
		PayloadVersion: orquestacoreworkflow.OutboxPayloadVersionV0,
		Payload: []byte(`{"agent_request_id":"` + agentRef + `","run_id":"` + runRef + `",` +
			`"reason_code":"run_stop_requested","summary":"Parada obsoleta de agente ya reflejado.",` +
			`"evidence_refs":["evidence-ref-stopruntimeagent-test"]}`),
	}
}
