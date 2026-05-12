package orquestacionnucleoapp

import (
	"context"
	"errors"
	"testing"
	"time"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectorscheduler "orquesta/modulos/orquesta-director-scheduler"
	orquestaoutboxdispatch "orquesta/modulos/orquesta-outbox-dispatch"
	orquestaruntime "orquesta/modulos/orquesta-runtime"
)

func TestProcessAgentStopperV0StopsRegisteredProcessAndConfirmsWorkflow(t *testing.T) {
	runRef := "run-nucleo-process-stopper-001"
	store := NewInMemoryRunStoreV0(mustActiveProgrammingRunV0(t, runRef))
	ledger := NewInMemoryOutboxLedgerV0()
	sink := NewInMemoryEventSinkV0()
	fakeRuntime := orquestaruntime.NewRuntimeFakeLifecycleV0()
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

	launched := runProgressiveLoopWithFakeAgentRuntimeV0(
		t,
		service,
		store,
		sink,
		ledger,
		fakeRuntime,
		runRef,
	)
	processRuntime := orquestaruntime.NewProcessRuntimeConnectorV0()
	process := launchWaitProcessForStopperTestV0(t, processRuntime)
	registry := NewInMemoryAgentProcessRegistryV0()
	recordProcessForStopperTestV0(t, registry, runRef, "agent-ref-001", process)
	saveStopOutboxV0(t, store, ledger, launched.Run, "agent-ref-001")

	dispatch, err := RunOutboxDispatchOnceV0(context.Background(), OutboxDispatchOnceRequestV0{
		RunRef:     runRef,
		TargetPort: orquestacoreworkflow.OutboxTargetAgentLauncherV0,
		Reader:     ledger,
		Claimer:    ledger,
		Executor: AgentStopperExecutorV0{
			RunStore:   store,
			EventSink:  sink,
			Stopper:    ProcessAgentStopperV0{Registry: registry, Runtime: processRuntime},
			ObservedAt: "2026-05-08T11:20:00Z",
		},
		Acker: ledger,
	})
	if err != nil {
		t.Fatalf("dispatch stop real: %v", err)
	}
	if dispatch.Status != "dispatched" || dispatch.DispatchRef == "" {
		t.Fatalf("dispatch=%+v", dispatch)
	}

	stopped, err := processRuntime.SnapshotV0(process.ProcessRef)
	if err != nil {
		t.Fatalf("snapshot stopped: %v", err)
	}
	if stopped.Status != orquestaruntime.ProcessRuntimeStoppedV0 || stopped.StopRef == "" {
		t.Fatalf("process no parado: %+v", stopped)
	}
	if dispatch.DispatchRef != stopped.StopRef {
		t.Fatalf("dispatch_ref=%q stop_ref=%q", dispatch.DispatchRef, stopped.StopRef)
	}
	if !containsNucleoRefV0(dispatch.EvidenceRefs, stopped.StopRef) {
		t.Fatalf("evidencias sin stop_ref: %+v stopped=%+v", dispatch.EvidenceRefs, stopped)
	}
	assertAgentStopConfirmedForProcessTestV0(t, store, sink, runRef, "agent-ref-001")
	assertNoPendingAgentLauncherOutboxForProcessTestV0(t, ledger, runRef)
}

func TestProcessAgentStopperV0DoesNotStopUnregisteredProcess(t *testing.T) {
	processRuntime := orquestaruntime.NewProcessRuntimeConnectorV0()
	process := launchWaitProcessForStopperTestV0(t, processRuntime)
	stopper := ProcessAgentStopperV0{
		Registry: NewInMemoryAgentProcessRegistryV0(),
		Runtime:  processRuntime,
	}

	_, err := stopper.StopAgentV0(context.Background(), validAgentStopperInboundForProcessTestV0())

	assertNucleoErrorV0(t, err, ErrNucleoOrquestacionStoreV0, "agent_process_registry")
	running, snapErr := processRuntime.SnapshotV0(process.ProcessRef)
	if snapErr != nil {
		t.Fatalf("snapshot process: %v", snapErr)
	}
	if running.Status != orquestaruntime.ProcessRuntimeRunningV0 || running.StopRef != "" {
		t.Fatalf("unregistered process was stopped: %+v", running)
	}
}

func TestProcessAgentStopperV0RejectsMismatchedSessionRefBeforeStop(t *testing.T) {
	processRuntime := orquestaruntime.NewProcessRuntimeConnectorV0()
	process := launchWaitProcessForStopperTestV0(t, processRuntime)
	registry := NewInMemoryAgentProcessRegistryV0()
	err := registry.RecordAgentProcessV0(context.Background(), AgentProcessRecordV0{
		RunID:          "run-nucleo-process-stopper-001",
		AgentRequestID: "agent-ref-001",
		ProcessRef:     process.ProcessRef,
		SessionRef:     "session-ref-stopper-mismatch-001",
		LaunchRef:      process.LaunchRef,
		ReadinessRef:   "readiness-ref-agent-ref-001",
	})
	if err != nil {
		t.Fatalf("record mismatched process identity: %v", err)
	}
	stopper := ProcessAgentStopperV0{
		Registry: registry,
		Runtime:  processRuntime,
	}

	_, err = stopper.StopAgentV0(context.Background(), validAgentStopperInboundForProcessTestV0())

	assertNucleoErrorV0(t, err, ErrNucleoOrquestacionInvalidoV0, "agent_process.session_ref")
	running, snapErr := processRuntime.SnapshotV0(process.ProcessRef)
	if snapErr != nil {
		t.Fatalf("snapshot process: %v", snapErr)
	}
	if running.Status != orquestaruntime.ProcessRuntimeRunningV0 || running.StopRef != "" {
		t.Fatalf("mismatched process was stopped: %+v", running)
	}
}

func TestProcessAgentStopperV0NoConfirmaStopSiRuntimeReiniciadoOlvidoProcessRef(t *testing.T) {
	previousRuntime := orquestaruntime.NewProcessRuntimeConnectorV0()
	process := launchWaitProcessForStopperTestV0(t, previousRuntime)
	registry := NewInMemoryAgentProcessRegistryV0()
	recordProcessForStopperTestV0(t, registry, "run-nucleo-process-stopper-001", "agent-ref-001", process)
	restartedRuntime := orquestaruntime.NewProcessRuntimeConnectorV0()
	stopper := ProcessAgentStopperV0{
		Registry: registry,
		Runtime:  restartedRuntime,
	}

	result, err := stopper.StopAgentV0(context.Background(), validAgentStopperInboundForProcessTestV0())

	var runtimeErr orquestaruntime.ProcessRuntimeErrorV0
	if !errors.As(err, &runtimeErr) || runtimeErr.Code != orquestaruntime.ProcessRuntimeNoEncontradoV0 {
		t.Fatalf("err=%v, want process_runtime_no_encontrado", err)
	}
	if result.ConfirmationRef != "" || len(result.EvidenceRefs) != 0 {
		t.Fatalf("stop no confirmado debe devolver resultado vacio: %+v", result)
	}
	running, snapErr := previousRuntime.SnapshotV0(process.ProcessRef)
	if snapErr != nil {
		t.Fatalf("snapshot previous runtime: %v", snapErr)
	}
	if running.Status != orquestaruntime.ProcessRuntimeRunningV0 || running.StopRef != "" {
		t.Fatalf("runtime reiniciado no debe confirmar ni parar por process_ref olvidado: %+v", running)
	}
}

func launchWaitProcessForStopperTestV0(
	t *testing.T,
	connector *orquestaruntime.ProcessRuntimeConnectorV0,
) orquestaruntime.ProcessRuntimeSnapshotV0 {
	t.Helper()
	process, err := connector.LaunchV0(
		context.Background(),
		externalProcessRuntimeRequestForTestV0(t, "wait"),
	)
	if err != nil {
		t.Fatalf("launch wait process: %v", err)
	}
	if process.Status != orquestaruntime.ProcessRuntimeRunningV0 {
		t.Fatalf("process status=%q", process.Status)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_, _ = connector.StopV0(ctx, process.ProcessRef)
	})
	return process
}

func recordProcessForStopperTestV0(
	t *testing.T,
	registry AgentProcessRegistryPortV0,
	runRef string,
	agentRequestID string,
	process orquestaruntime.ProcessRuntimeSnapshotV0,
) {
	t.Helper()
	err := registry.RecordAgentProcessV0(context.Background(), AgentProcessRecordV0{
		RunID:          runRef,
		AgentRequestID: agentRequestID,
		ProcessRef:     process.ProcessRef,
		SessionRef:     process.SessionRef,
		LaunchRef:      process.LaunchRef,
		ReadinessRef:   "readiness-ref-" + agentRequestID,
	})
	if err != nil {
		t.Fatalf("record process: %v", err)
	}
}

func validAgentStopperInboundForProcessTestV0() orquestaruntime.AgentStopperInboundV0 {
	return orquestaruntime.AgentStopperInboundV0{
		TargetPort:     orquestaruntime.AgentStopperTargetPortV0,
		MessageType:    orquestaruntime.AgentStopperMessageTypeV0,
		CorrelationID:  "corr-process-stopper-001",
		IdempotencyKey: "idem-process-stopper-001",
		Payload: &orquestaruntime.StopRuntimeAgentRequestV0{
			AgentRequestID: "agent-ref-001",
			RunID:          "run-nucleo-process-stopper-001",
			ReasonCode:     "supervision_loop_detected",
			Summary:        "Parada solicitada por supervision.",
			EvidenceRefs:   []string{"evidence-ref-stop-process-001"},
		},
	}
}

func assertAgentStopConfirmedForProcessTestV0(
	t *testing.T,
	store *InMemoryRunStoreV0,
	sink *InMemoryEventSinkV0,
	runRef string,
	agentRequestID string,
) {
	t.Helper()
	finalRun, err := store.LoadRunV0(context.Background(), runRef)
	if err != nil {
		t.Fatalf("load final run: %v", err)
	}
	if !containsNucleoRefV0(finalRun.ConfirmedStoppedAgents, agentRequestID) {
		t.Fatalf("confirmed stopped agents=%v", finalRun.ConfirmedStoppedAgents)
	}
	if !sinkHasEventTypeV0(sink, orquestacoreworkflow.OrchestrationEventAgentStopConfirmedV0) {
		t.Fatalf("sink no contiene AgentStopConfirmed: %+v", sink.EventsV0())
	}
}

func assertNoPendingAgentLauncherOutboxForProcessTestV0(
	t *testing.T,
	ledger *InMemoryOutboxLedgerV0,
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
