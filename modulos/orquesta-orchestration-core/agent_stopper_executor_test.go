package orquestacionnucleoapp

import (
	"context"
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
