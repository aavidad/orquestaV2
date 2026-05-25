package orquestacionnucleoapp

import (
	"context"
	"encoding/json"
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectorscheduler "orquesta/modulos/orquesta-director-scheduler"
	orquestaoutboxdispatch "orquesta/modulos/orquesta-outbox-dispatch"
	orquestaruntime "orquesta/modulos/orquesta-runtime"
)

func TestExternalProcessAgentBatchExecutorV0ProgressiveLoopStartsTwoProcesses(t *testing.T) {
	runRef := "run-nucleo-process-batch-001"
	store := NewInMemoryRunStoreV0(mustActiveProgrammingRunV0(t, runRef))
	ledger := NewInMemoryOutboxLedgerV0()
	sink := NewInMemoryEventSinkV0()
	processRuntime := orquestaruntime.NewProcessRuntimeConnectorV0()
	registry := NewInMemoryAgentProcessRegistryV0()
	service := ServiceV0{
		RunStore:  store,
		EventSink: sink,
		CandidateProvider: StaticCandidateProviderV0{Candidates: SchedulerCandidateSetV0{
			WorkCandidates: []orquestadirectorscheduler.SchedulableWorkCandidateV0{
				workCandidateWithAgentVariantV0(runRef, "001", "app/process_worker_a.go"),
				workCandidateWithAgentVariantV0(runRef, "002", "app/process_worker_b.go"),
			},
		}},
		OutboxLedger:      ledger,
		MaxCommands:       8,
		MaxOutboxPerCycle: 2,
	}

	result, err := service.RunProgressiveLoopV0(context.Background(), ProgressiveLoopRequestV0{
		RunRef:               runRef,
		OccurredAt:           "2026-05-09T11:00:00Z",
		MaxBursts:            5,
		MaxStepsPerBurst:     4,
		MaxDispatchesPerWait: 4,
		CorrelationID:        "corr-process-batch-001",
		EvidenceRefs:         []string{"evidence-ref-process-batch-001"},
		Dispatchers: []OutboxDispatcherBindingV0{
			capacityDecisionDispatcherForTestV0(store, sink, ledger),
		},
		BatchDispatchers: []OutboxBatchDispatcherBindingV0{{
			TargetPort: orquestacoreworkflow.OutboxTargetAgentLauncherV0,
			MaxReady:   2,
			Reader:     ledger,
			Claimer:    ledger,
			Executor: ExternalProcessAgentBatchExecutorV0{
				RunStore:        store,
				EventSink:       sink,
				SpecResolver:    externalAgentLaunchSpecResolverForTestV0(externalProcessRuntimeRequestForTestV0(t, "exit")),
				Runtime:         processRuntime,
				ProcessStopper:  processRuntime,
				ProcessRegistry: registry,
				MaxConcurrency:  2,
				OccurredAt:      "2026-05-09T11:01:00Z",
			},
			Acker: ledger,
		}},
	})
	if err != nil {
		t.Fatalf("progressive loop process batch: %v", err)
	}
	if result.Status != ProgressiveLoopStatusWaitExternalV0 {
		t.Fatalf("status=%s result=%+v", result.Status, result)
	}
	requireBatchStartedAgentsV0(t, result.Run.StartedAgents, "agent-ref-001", "agent-ref-002")
	dispatched := dispatchedProcessBatchResultV0(result.BatchDispatches)
	if dispatched.PlannedCount != 2 || dispatched.AckedCount != 2 {
		t.Fatalf("batch dispatches=%+v", result.BatchDispatches)
	}
	for _, agentRef := range []string{"agent-ref-001", "agent-ref-002"} {
		record, err := registry.ResolveAgentProcessV0(context.Background(), runRef, agentRef)
		if err != nil {
			t.Fatalf("resolve process %s: %v", agentRef, err)
		}
		waitExternalProcessRuntimeStatusV0(t, processRuntime, record.ProcessRef, orquestaruntime.ProcessRuntimeStoppedV0)
	}
	if result.PendingOutboxCount != 0 {
		t.Fatalf("pending=%d refs=%v", result.PendingOutboxCount, result.PendingOutboxRefs)
	}
}

func TestExternalProcessAgentBatchExecutorV0MarksInvalidItemFailed(t *testing.T) {
	executor := ExternalProcessAgentBatchExecutorV0{
		RunStore:        NewInMemoryRunStoreV0(),
		SpecResolver:    externalAgentLaunchSpecResolverForTestV0(externalProcessRuntimeRequestForTestV0(t, "exit")),
		Runtime:         orquestaruntime.NewProcessRuntimeConnectorV0(),
		ProcessStopper:  orquestaruntime.NewProcessRuntimeConnectorV0(),
		ProcessRegistry: NewInMemoryAgentProcessRegistryV0(),
		OccurredAt:      "2026-05-09T11:05:00Z",
	}

	acks, err := executor.ExecuteOutboxDispatchBatchV0(context.Background(), []orquestaoutboxdispatch.DispatchIntentV0{
		orquestacoreworkflowIntentV0(
			orquestacoreworkflow.OutboxMessageLaunchRuntimeAgentV0,
			orquestacoreworkflow.OutboxTargetAgentLauncherV0,
		),
	})
	if err != nil {
		t.Fatalf("ExecuteOutboxDispatchBatchV0: %v", err)
	}
	if len(acks) != 1 || acks[0].Status != orquestaoutboxdispatch.OutboxDispatchAckObservationFailedV0 {
		t.Fatalf("acks=%+v", acks)
	}
}

func TestExternalProcessAgentBatchExecutorV0RegistersFailedAgentOnBlockedLaunch(t *testing.T) {
	runRef := "run-nucleo-process-batch-blocked-001"
	store := NewInMemoryRunStoreV0(mustActiveProgrammingRunV0(t, runRef))
	ledger := NewInMemoryOutboxLedgerV0()
	sink := NewInMemoryEventSinkV0()
	badRequest := externalProcessRuntimeRequestForTestV0(t, "exit")
	badRequest.Env = nil
	service := ServiceV0{
		RunStore:  store,
		EventSink: sink,
		CandidateProvider: StaticCandidateProviderV0{Candidates: SchedulerCandidateSetV0{
			WorkCandidates: []orquestadirectorscheduler.SchedulableWorkCandidateV0{
				workCandidateWithAgentVariantV0(runRef, "blocked", "app/process_worker_blocked.go"),
			},
		}},
		OutboxLedger:      ledger,
		MaxCommands:       8,
		MaxOutboxPerCycle: 2,
	}

	result, err := service.RunProgressiveLoopV0(context.Background(), ProgressiveLoopRequestV0{
		RunRef:               runRef,
		OccurredAt:           "2026-05-09T11:10:00Z",
		MaxBursts:            5,
		MaxStepsPerBurst:     4,
		MaxDispatchesPerWait: 4,
		CorrelationID:        "corr-process-batch-blocked-001",
		EvidenceRefs:         []string{"evidence-ref-process-batch-blocked-001"},
		Dispatchers: []OutboxDispatcherBindingV0{
			capacityDecisionDispatcherForTestV0(store, sink, ledger),
		},
		BatchDispatchers: []OutboxBatchDispatcherBindingV0{{
			TargetPort: orquestacoreworkflow.OutboxTargetAgentLauncherV0,
			MaxReady:   1,
			Reader:     ledger,
			Claimer:    ledger,
			Executor: ExternalProcessAgentBatchExecutorV0{
				RunStore:        store,
				EventSink:       sink,
				SpecResolver:    externalAgentLaunchSpecResolverForTestV0(badRequest),
				Runtime:         orquestaruntime.NewProcessRuntimeConnectorV0(),
				ProcessStopper:  orquestaruntime.NewProcessRuntimeConnectorV0(),
				ProcessRegistry: NewInMemoryAgentProcessRegistryV0(),
				MaxConcurrency:  1,
				OccurredAt:      "2026-05-09T11:11:00Z",
				RequestedBy:     "orquesta-test",
				EvidenceRefs:    []string{"evidence-ref-process-batch-blocked-executor"},
			},
			Acker: ledger,
		}},
	})
	if err != nil {
		t.Fatalf("progressive loop process batch blocked: %v", err)
	}
	if containsNucleoRefV0(result.Run.StartedAgents, "agent-ref-blocked") {
		t.Fatalf("started agents=%v", result.Run.StartedAgents)
	}
	if !containsNucleoRefV0(result.Run.FailedAgents, "agent-ref-blocked") {
		t.Fatalf("failed agents=%v", result.Run.FailedAgents)
	}
	dispatched := dispatchedProcessBatchResultV0(result.BatchDispatches)
	if dispatched.AckedCount != 1 || dispatched.FailedCount != 0 {
		t.Fatalf("batch dispatches=%+v", result.BatchDispatches)
	}
	if result.PendingOutboxCount != 0 {
		t.Fatalf("pending=%d refs=%v", result.PendingOutboxCount, result.PendingOutboxRefs)
	}
}

func TestExternalProcessAgentBatchExecutorV0CierraOutboxSiAgenteYaEsTerminal(t *testing.T) {
	runRef := "run-nucleo-process-batch-terminal-001"
	agentRef := "agent-ref-process-batch-terminal-001"
	run := mustActiveProgrammingRunV0(t, runRef)
	run.FailedAgents = []string{agentRef}
	store := NewInMemoryRunStoreV0(run)
	inbound := externalProcessLauncherInboundV0(runRef, agentRef)
	payload, err := json.Marshal(inbound.Payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	acks, err := (ExternalProcessAgentBatchExecutorV0{
		RunStore:        store,
		EventSink:       NewInMemoryEventSinkV0(),
		SpecResolver:    batchSpecResolverMustNotRunV0{t: t},
		Runtime:         orquestaruntime.NewProcessRuntimeConnectorV0(),
		ProcessStopper:  orquestaruntime.NewProcessRuntimeConnectorV0(),
		ProcessRegistry: NewInMemoryAgentProcessRegistryV0(),
		OccurredAt:      "2026-05-09T11:12:00Z",
	}).ExecuteOutboxDispatchBatchV0(context.Background(), []orquestaoutboxdispatch.DispatchIntentV0{{
		MessageID:      "outbox-process-batch-terminal-001",
		RunID:          runRef,
		TargetPort:     orquestacoreworkflow.OutboxTargetAgentLauncherV0,
		MessageType:    orquestacoreworkflow.OutboxMessageLaunchRuntimeAgentV0,
		IdempotencyKey: inbound.IdempotencyKey,
		CorrelationID:  inbound.CorrelationID,
		PayloadVersion: orquestacoreworkflow.OutboxPayloadVersionV0,
		Payload:        payload,
	}})
	if err != nil {
		t.Fatalf("ExecuteOutboxDispatchBatchV0: %v", err)
	}
	if len(acks) != 1 ||
		acks[0].Status != orquestaoutboxdispatch.OutboxDispatchAckObservationSuccessV0 ||
		!containsNucleoRefV0(acks[0].EvidenceRefs, "evidence-ref-external-process-batch-terminal-agent-preserved") {
		t.Fatalf("acks=%+v", acks)
	}
}

type batchSpecResolverMustNotRunV0 struct {
	t *testing.T
}

func (resolver batchSpecResolverMustNotRunV0) ResolveExternalAgentLaunchSpecV0(
	context.Context,
	orquestaruntime.AgentLauncherInboundV0,
) (ExternalAgentLaunchSpecResolutionV0, error) {
	resolver.t.Fatalf("spec resolver no debe ejecutarse para agente terminal")
	return ExternalAgentLaunchSpecResolutionV0{}, nil
}

func dispatchedProcessBatchResultV0(
	results []OutboxDispatchBatchRunResultV0,
) OutboxDispatchBatchRunResultV0 {
	for _, result := range results {
		if result.Status == OutboxDispatchBatchRunDispatchedV0 {
			return result
		}
	}
	return OutboxDispatchBatchRunResultV0{}
}

func failedProcessBatchResultV0(
	results []OutboxDispatchBatchRunResultV0,
) OutboxDispatchBatchRunResultV0 {
	for _, result := range results {
		if result.Status == OutboxDispatchBatchRunAckFailedV0 {
			return result
		}
	}
	return OutboxDispatchBatchRunResultV0{}
}

func requireBatchStartedAgentsV0(
	t *testing.T,
	started []string,
	agentRefs ...string,
) {
	t.Helper()
	for _, agentRef := range agentRefs {
		if !containsNucleoRefV0(started, agentRef) {
			t.Fatalf("started agents=%v missing=%s", started, agentRef)
		}
	}
}
