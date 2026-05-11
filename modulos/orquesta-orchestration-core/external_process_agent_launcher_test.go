package orquestacionnucleoapp

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectorscheduler "orquesta/modulos/orquesta-director-scheduler"
	orquestaoutboxdispatch "orquesta/modulos/orquesta-outbox-dispatch"
	orquestaruntime "orquesta/modulos/orquesta-runtime"
)

func TestExternalProcessAgentLauncherV0LaunchAgentStartsRealProcess(t *testing.T) {
	inbound := externalProcessLauncherInboundV0("run-nucleo-process-direct-001", "agent-ref-process-direct-001")
	resolver := externalAgentLaunchSpecResolverForTestV0(
		externalProcessRuntimeRequestForTestV0(t, "exit"),
	)
	runtime := &recordingExternalProcessRuntimeV0{
		connector: orquestaruntime.NewProcessRuntimeConnectorV0(),
	}
	registry := &recordingAgentProcessRegistryV0{}
	launcher := &ExternalProcessAgentLauncherV0{
		SpecResolver:    resolver,
		Runtime:         runtime,
		ProcessStopper:  runtime,
		ProcessRegistry: registry,
	}

	result, err := launcher.LaunchAgentV0(context.Background(), inbound)
	if err != nil {
		t.Fatalf("launch agent: %v", err)
	}

	if result.AgentRequestID != inbound.Payload.AgentRequestID {
		t.Fatalf("agent_request_id=%q", result.AgentRequestID)
	}
	if result.LaunchRef == "" || result.LaunchRef != runtime.lastSnapshot.LaunchRef {
		t.Fatalf("launch_ref=%q snapshot=%+v", result.LaunchRef, runtime.lastSnapshot)
	}
	if result.AckRef != "ack-ref-"+inbound.Payload.AgentRequestID ||
		result.ReadinessRef != "readiness-ref-"+inbound.Payload.AgentRequestID {
		t.Fatalf("delivery refs invalidas: %+v", result)
	}
	if len(registry.records) != 1 {
		t.Fatalf("registry records=%+v", registry.records)
	}
	record := registry.records[0]
	if record.RunID != inbound.Payload.RunID ||
		record.AgentRequestID != inbound.Payload.AgentRequestID ||
		record.ProcessRef != runtime.lastSnapshot.ProcessRef ||
		record.LaunchRef != runtime.lastSnapshot.LaunchRef ||
		record.ReadinessRef != result.ReadinessRef {
		t.Fatalf("registry record invalido: %+v snapshot=%+v result=%+v", record, runtime.lastSnapshot, result)
	}
	waitExternalProcessRuntimeStatusV0(t, runtime.connector, runtime.lastSnapshot.ProcessRef, orquestaruntime.ProcessRuntimeStoppedV0)
}

func TestExternalProcessAgentLauncherV0RegistryFailureBlocksAgentStarted(t *testing.T) {
	runRef := "run-nucleo-process-registry-fail-001"
	agentRequestID := "agent-ref-process-registry-fail-001"
	inbound := externalProcessLauncherInboundV0(runRef, agentRequestID)
	payload, err := json.Marshal(inbound.Payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	store := NewInMemoryRunStoreV0(mustActiveProgrammingRunV0(t, runRef))
	sink := NewInMemoryEventSinkV0()
	runtime := &recordingExternalProcessRuntimeV0{
		connector: orquestaruntime.NewProcessRuntimeConnectorV0(),
	}
	executor := AgentLauncherExecutorV0{
		RunStore:  store,
		EventSink: sink,
		Launcher: &ExternalProcessAgentLauncherV0{
			SpecResolver: externalAgentLaunchSpecResolverForTestV0(
				externalProcessRuntimeRequestForTestV0(t, "wait"),
			),
			Runtime:        runtime,
			ProcessStopper: runtime,
			ProcessRegistry: &recordingAgentProcessRegistryV0{
				err: errors.New("registro de proceso no disponible"),
			},
		},
		OccurredAt: "2026-05-08T11:03:00Z",
	}

	_, err = executor.ExecuteOutboxDispatchV0(orquestaoutboxdispatch.DispatchIntentV0{
		MessageID:      "outbox-process-registry-fail-001",
		RunID:          runRef,
		TargetPort:     orquestacoreworkflow.OutboxTargetAgentLauncherV0,
		MessageType:    orquestacoreworkflow.OutboxMessageLaunchRuntimeAgentV0,
		IdempotencyKey: inbound.IdempotencyKey,
		CorrelationID:  inbound.CorrelationID,
		PayloadVersion: orquestacoreworkflow.OutboxPayloadVersionV0,
		Payload:        payload,
	})
	if err == nil {
		t.Fatalf("esperaba error de registry")
	}
	if hasEventTypeV0(sink.EventsV0(), orquestacoreworkflow.OrchestrationEventAgentStartedV0) {
		t.Fatalf("AgentStarted no debe registrarse si falla registry: %+v", sink.EventsV0())
	}
	run, loadErr := store.LoadRunV0(context.Background(), runRef)
	if loadErr != nil {
		t.Fatalf("load run: %v", loadErr)
	}
	if containsNucleoRefV0(run.StartedAgents, agentRequestID) {
		t.Fatalf("started agents=%v", run.StartedAgents)
	}
	waitExternalProcessRuntimeStatusV0(t, runtime.connector, runtime.lastSnapshot.ProcessRef, orquestaruntime.ProcessRuntimeStoppedV0)
}

func TestAgentLauncherExecutorV0WorkflowFailureStopsLaunchedProcess(t *testing.T) {
	runRef := "run-nucleo-process-workflow-fail-001"
	agentRequestID := "agent-ref-process-workflow-fail-001"
	inbound := externalProcessLauncherInboundV0(runRef, agentRequestID)
	payload, err := json.Marshal(inbound.Payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	store := NewInMemoryRunStoreV0(mustActiveProgrammingRunV0(t, runRef))
	runtime := &recordingExternalProcessRuntimeV0{
		connector: orquestaruntime.NewProcessRuntimeConnectorV0(),
	}
	registry := NewInMemoryAgentProcessRegistryV0()
	executor := AgentLauncherExecutorV0{
		RunStore:  store,
		EventSink: failingEventSinkV0{},
		Launcher: &ExternalProcessAgentLauncherV0{
			SpecResolver: externalAgentLaunchSpecResolverForTestV0(
				externalProcessRuntimeRequestForTestV0(t, "wait"),
			),
			Runtime:         runtime,
			ProcessStopper:  runtime,
			ProcessRegistry: registry,
		},
		FailureStopper: ProcessAgentStopperV0{Registry: registry, Runtime: runtime},
		OccurredAt:     "2026-05-08T11:04:00Z",
	}

	_, err = executor.ExecuteOutboxDispatchV0(orquestaoutboxdispatch.DispatchIntentV0{
		MessageID:      "outbox-process-workflow-fail-001",
		RunID:          runRef,
		TargetPort:     orquestacoreworkflow.OutboxTargetAgentLauncherV0,
		MessageType:    orquestacoreworkflow.OutboxMessageLaunchRuntimeAgentV0,
		IdempotencyKey: inbound.IdempotencyKey,
		CorrelationID:  inbound.CorrelationID,
		PayloadVersion: orquestacoreworkflow.OutboxPayloadVersionV0,
		Payload:        payload,
	})
	if err == nil {
		t.Fatalf("esperaba error de workflow")
	}
	waitExternalProcessRuntimeStatusV0(t, runtime.connector, runtime.lastSnapshot.ProcessRef, orquestaruntime.ProcessRuntimeStoppedV0)
}

func TestExternalProcessAgentLauncherV0ProgressiveLoopRegistersAgentStarted(t *testing.T) {
	runRef := "run-nucleo-process-progressive-001"
	store := NewInMemoryRunStoreV0(mustActiveProgrammingRunV0(t, runRef))
	ledger := NewInMemoryOutboxLedgerV0()
	sink := NewInMemoryEventSinkV0()
	runtime := &recordingExternalProcessRuntimeV0{
		connector: orquestaruntime.NewProcessRuntimeConnectorV0(),
	}
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

	result, err := service.RunProgressiveLoopV0(context.Background(), ProgressiveLoopRequestV0{
		RunRef:               runRef,
		OccurredAt:           "2026-05-08T11:00:00Z",
		MaxBursts:            4,
		MaxStepsPerBurst:     3,
		MaxDispatchesPerWait: 2,
		CorrelationID:        "corr-process-progressive-001",
		EvidenceRefs:         []string{"evidence-ref-process-progressive-001"},
		Dispatchers: []OutboxDispatcherBindingV0{
			capacityDecisionDispatcherForTestV0(store, sink, ledger),
			externalProcessAgentDispatcherForTestV0(store, sink, ledger, runtime, t),
		},
	})
	if err != nil {
		t.Fatalf("progressive loop: %v", err)
	}
	if result.Status != ProgressiveLoopStatusWaitExternalV0 {
		t.Fatalf("status=%s result=%+v", result.Status, result)
	}
	if !containsNucleoRefV0(result.Run.StartedAgents, "agent-ref-001") {
		t.Fatalf("started agents=%v", result.Run.StartedAgents)
	}
	for _, eventType := range []string{
		orquestacoreworkflow.OrchestrationEventCapacityRequestedV0,
		orquestacoreworkflow.OrchestrationEventAgentRequestedV0,
		orquestacoreworkflow.OrchestrationEventAgentStartedV0,
	} {
		if !hasEventTypeV0(sink.EventsV0(), eventType) {
			t.Fatalf("events sin %s: %+v", eventType, sink.EventsV0())
		}
	}
	if runtime.lastSnapshot.ProcessRef == "" || runtime.lastRequest.CommandPath == "" {
		t.Fatalf("runtime real no invocado: snapshot=%+v request=%+v", runtime.lastSnapshot, runtime.lastRequest)
	}
	waitExternalProcessRuntimeStatusV0(t, runtime.connector, runtime.lastSnapshot.ProcessRef, orquestaruntime.ProcessRuntimeStoppedV0)
}

type recordingAgentProcessRegistryV0 struct {
	records []AgentProcessRegistryRecordV0
	err     error
}

type failingEventSinkV0 struct{}

func (failingEventSinkV0) AppendRunEventsV0(
	context.Context,
	string,
	[]orquestacoreworkflow.OrchestrationEventV0,
) error {
	return errors.New("event sink no disponible")
}

func (registry *recordingAgentProcessRegistryV0) RecordAgentProcessV0(
	_ context.Context,
	record AgentProcessRegistryRecordV0,
) error {
	if registry.err != nil {
		return registry.err
	}
	registry.records = append(registry.records, record)
	return nil
}

func (registry *recordingAgentProcessRegistryV0) ResolveAgentProcessV0(
	_ context.Context,
	_ string,
	_ string,
) (AgentProcessRegistryRecordV0, error) {
	if registry.err != nil {
		return AgentProcessRegistryRecordV0{}, registry.err
	}
	if len(registry.records) == 0 {
		return AgentProcessRegistryRecordV0{}, errors.New("agent process no registrado")
	}
	return registry.records[0], nil
}
