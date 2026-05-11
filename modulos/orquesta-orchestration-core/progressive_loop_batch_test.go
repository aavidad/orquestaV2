package orquestacionnucleoapp

import (
	"context"
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectorscheduler "orquesta/modulos/orquesta-director-scheduler"
	orquestaoutboxdispatch "orquesta/modulos/orquesta-outbox-dispatch"
)

func TestRunProgressiveLoopV0UsesBatchDispatcherForTwoAgentLaunches(t *testing.T) {
	runRef := "run-nucleo-progressive-batch-001"
	store := NewInMemoryRunStoreV0(mustActiveProgrammingRunV0(t, runRef))
	ledger := NewInMemoryOutboxLedgerV0()
	sink := NewInMemoryEventSinkV0()
	batchExecutor := &singleAgentLauncherBatchExecutorForTestV0{
		executor: AgentLauncherExecutorV0{
			RunStore:   store,
			EventSink:  sink,
			Launcher:   NewFakeLifecycleAgentLauncherV0(),
			OccurredAt: "2026-05-09T10:12:00Z",
		},
	}
	service := ServiceV0{
		RunStore:  store,
		EventSink: sink,
		CandidateProvider: StaticCandidateProviderV0{Candidates: SchedulerCandidateSetV0{
			WorkCandidates: []orquestadirectorscheduler.SchedulableWorkCandidateV0{
				workCandidateWithAgentVariantV0(runRef, "001", "app/worker_a.go"),
				workCandidateWithAgentVariantV0(runRef, "002", "app/worker_b.go"),
			},
		}},
		OutboxLedger:      ledger,
		MaxCommands:       8,
		MaxOutboxPerCycle: 2,
	}

	result, err := service.RunProgressiveLoopV0(context.Background(), ProgressiveLoopRequestV0{
		RunRef:               runRef,
		OccurredAt:           "2026-05-09T10:10:00Z",
		MaxBursts:            5,
		MaxStepsPerBurst:     4,
		MaxDispatchesPerWait: 4,
		CorrelationID:        "corr-progressive-batch-001",
		EvidenceRefs:         []string{"evidence-ref-progressive-batch-001"},
		Dispatchers: []OutboxDispatcherBindingV0{
			capacityDecisionDispatcherForTestV0(store, sink, ledger),
		},
		BatchDispatchers: []OutboxBatchDispatcherBindingV0{{
			TargetPort: orquestacoreworkflow.OutboxTargetAgentLauncherV0,
			MaxReady:   2,
			Reader:     ledger,
			Claimer:    ledger,
			Executor:   batchExecutor,
			Acker:      ledger,
		}},
	})
	if err != nil {
		t.Fatalf("progressive loop batch: %v", err)
	}
	if result.Status != ProgressiveLoopStatusWaitExternalV0 {
		t.Fatalf("status=%s result=%+v", result.Status, result)
	}
	for _, agentRef := range []string{"agent-ref-001", "agent-ref-002"} {
		if !containsNucleoRefV0(result.Run.StartedAgents, agentRef) {
			t.Fatalf("started agents=%v", result.Run.StartedAgents)
		}
	}
	if !hasProgressiveBatchDispatchedV0(result.BatchDispatches) ||
		batchExecutor.lastBatchSize != 2 {
		t.Fatalf("batch dispatches=%+v lastBatchSize=%d", result.BatchDispatches, batchExecutor.lastBatchSize)
	}
	if result.PendingOutboxCount != 0 {
		t.Fatalf("pending=%d refs=%v", result.PendingOutboxCount, result.PendingOutboxRefs)
	}
}

type singleAgentLauncherBatchExecutorForTestV0 struct {
	executor      AgentLauncherExecutorV0
	lastBatchSize int
}

func (executor *singleAgentLauncherBatchExecutorForTestV0) ExecuteOutboxDispatchBatchV0(
	ctx context.Context,
	intents []orquestaoutboxdispatch.DispatchIntentV0,
) ([]orquestaoutboxdispatch.OutboxDispatchAckObservationV0, error) {
	executor.lastBatchSize = len(intents)
	acks := make([]orquestaoutboxdispatch.OutboxDispatchAckObservationV0, 0, len(intents))
	for _, intent := range intents {
		execution, err := executor.executor.ExecuteOutboxDispatchV0(intent)
		if err != nil {
			return nil, err
		}
		acks = append(acks, orquestaoutboxdispatch.OutboxDispatchAckObservationV0{
			MessageID:    intent.MessageID,
			RunID:        intent.RunID,
			TargetPort:   intent.TargetPort,
			Status:       orquestaoutboxdispatch.OutboxDispatchAckObservationSuccessV0,
			DispatchRef:  execution.DispatchRef,
			EvidenceRefs: execution.EvidenceRefs,
		})
	}
	return acks, nil
}

func workCandidateWithAgentVariantV0(
	runRef string,
	suffix string,
	targetFile string,
) orquestadirectorscheduler.SchedulableWorkCandidateV0 {
	candidate := workCandidateWithAgentV0(runRef)
	claimRef := "claim-ref-" + suffix
	taskRef := "task-ref-" + suffix
	capacityRef := "capacity-ref-" + suffix
	agentRef := "agent-ref-" + suffix

	candidate.CandidateRef = "candidate-ref-" + suffix
	candidate.SubjectClaimRefs = []string{claimRef}
	candidate.Claims[0].ClaimRef = claimRef
	candidate.Claims[0].TaskRef = taskRef
	candidate.Claims[0].AgentRequestID = agentRef
	candidate.Claims[0].WriteSet[0].Ref = targetFile
	candidate.Claims[0].EvidenceRefs = []string{"evidence-ref-claim-" + suffix}
	candidate.CapacityCandidate.CommandMeta = commandMetaV0(runRef, "cmd-capacity-"+suffix, "idem-capacity-"+suffix)
	candidate.CapacityCandidate.Payload.CapacityRequestID = capacityRef
	candidate.CapacityCandidate.Payload.TaskRef = taskRef
	candidate.CapacityCandidate.Payload.EvidenceRefs = []string{"evidence-ref-capacity-" + suffix}
	candidate.AgentCandidate.ClaimRef = claimRef
	candidate.AgentCandidate.CommandMeta = commandMetaV0(runRef, "cmd-agent-"+suffix, "idem-agent-"+suffix)
	candidate.AgentCandidate.Payload.AgentRequestID = agentRef
	candidate.AgentCandidate.Payload.TaskRef = taskRef
	candidate.AgentCandidate.Payload.CapacityRequestRef = capacityRef
	candidate.AgentCandidate.Payload.EvidenceRefs = []string{"evidence-ref-agent-" + suffix}
	candidate.GateCommandMeta = commandMetaV0(runRef, "cmd-gate-"+suffix, "idem-gate-"+suffix)
	candidate.GateEvidenceRefs = []string{"evidence-ref-gate-" + suffix}
	candidate.EvidenceRefs = []string{"evidence-ref-candidate-" + suffix}
	return candidate
}
