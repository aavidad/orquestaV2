package orquestacionnucleoapp

import (
	"context"
	"encoding/json"
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectorcycleoutbox "orquesta/modulos/orquesta-director-cycle-outbox"
	orquestadirectorscheduler "orquesta/modulos/orquesta-director-scheduler"
	orquestadirectorsupervisor "orquesta/modulos/orquesta-director-supervisor"
)

func TestRunSupervisedBurstV0WaitsWhenOutboxAlreadyPending(t *testing.T) {
	runRef := "run-nucleo-pending-001"
	ledger := &memoryOutboxLedgerV0{}
	_, issues := ledger.SavePending(context.Background(), []orquestacoreworkflow.OutboxMessageV0{
		mustCapacityOutboxMessageV0(t, runRef, "outbox-ref-pending-001"),
	})
	if len(issues) > 0 {
		t.Fatalf("seed outbox issues=%+v", issues)
	}
	service := ServiceV0{
		RunStore: newMemoryRunStoreV0(mustActiveProgrammingRunV0(t, runRef)),
		CandidateProvider: StaticCandidateProviderV0{Candidates: SchedulerCandidateSetV0{
			WorkCandidates: []orquestadirectorscheduler.SchedulableWorkCandidateV0{
				workCandidateV0(runRef),
			},
		}},
		OutboxLedger: ledger,
	}

	result, err := service.RunSupervisedBurstV0(context.Background(), SupervisedBurstRequestV0{
		RunRef:     runRef,
		OccurredAt: "2026-05-08T10:10:00Z",
		MaxSteps:   3,
	})
	if err != nil {
		t.Fatalf("run burst: %v", err)
	}
	if result.Burst.FinalAction != orquestadirectorsupervisor.DirectorSupervisorActionWaitOutboxV0 {
		t.Fatalf("final action=%s", result.Burst.FinalAction)
	}
	if len(result.Run.CapacityRequests) != 0 {
		t.Fatalf("no debe pedir capacidad nueva con outbox pendiente: %v", result.Run.CapacityRequests)
	}
	pending, pendingIssues := ledger.ListPending(context.Background(), orquestadirectorcycleoutbox.DirectorCycleOutboxPendingFilterV0{RunRef: runRef})
	if len(pendingIssues) > 0 || len(pending) != 1 {
		t.Fatalf("pending=%+v issues=%+v", pending, pendingIssues)
	}
}

func mustCapacityOutboxMessageV0(
	t *testing.T,
	runRef string,
	messageID string,
) orquestacoreworkflow.OutboxMessageV0 {
	t.Helper()
	payload, err := json.Marshal(orquestacoreworkflow.CapacityDecisionRequestV0{
		CapacityRequestID:          "capacity-ref-pending-001",
		RunID:                      runRef,
		PhaseID:                    string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0),
		TaskRef:                    "task-ref-pending-001",
		ReasonCode:                 "programacion_siguiente_paso",
		Summary:                    "Decision de capacidad pendiente.",
		MinimumRecommendedCapacity: orquestacoreworkflow.OrchestrationCapacityMediumV0,
		EvidenceRefs:               []string{"evidence-ref-pending-001"},
	})
	if err != nil {
		t.Fatalf("payload: %v", err)
	}
	message := orquestacoreworkflow.OutboxMessageV0{
		MessageID:      messageID,
		MessageType:    orquestacoreworkflow.OutboxMessageRequestCapacityDecisionV0,
		RunID:          runRef,
		IdempotencyKey: "idem-" + messageID,
		CorrelationID:  "corr-nucleo-pending-001",
		TargetPort:     orquestacoreworkflow.OutboxTargetCapacityV0,
		PayloadVersion: orquestacoreworkflow.OutboxPayloadVersionV0,
		Payload:        payload,
	}
	if err := orquestacoreworkflow.ValidateOutboxMessageV0(message); err != nil {
		t.Fatalf("outbox message: %v", err)
	}
	return message
}
