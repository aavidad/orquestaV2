package orquestacionnucleoapp

import (
	"context"
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectorcycleoutbox "orquesta/modulos/orquesta-director-cycle-outbox"
	orquestaoutboxdispatch "orquesta/modulos/orquesta-outbox-dispatch"
)

func TestRunOutboxDispatchBatchAckClosureV0AcksOnlySuccessfulItems(t *testing.T) {
	ledger := NewInMemoryOutboxLedgerV0()
	saveBatchPlanOutboxV0(t, ledger,
		batchPlanMessageV0("outbox-agent-ack-001"),
		batchPlanMessageV0("outbox-agent-ack-002"),
		batchPlanMessageV0("outbox-agent-ack-003"),
	)
	plan, err := RunOutboxDispatchBatchPlanV0(context.Background(), OutboxDispatchBatchPlanRequestV0{
		RunRef:     "run-batch-plan-001",
		TargetPort: orquestacoreworkflow.OutboxTargetAgentLauncherV0,
		MaxReady:   3,
		Reader:     ledger,
		Claimer:    ledger,
	})
	if err != nil {
		t.Fatalf("plan batch: %v", err)
	}

	result, err := RunOutboxDispatchBatchAckClosureV0(context.Background(), OutboxDispatchBatchAckRequestV0{
		Intents: plan.Intents,
		Acks: []orquestaoutboxdispatch.OutboxDispatchAckObservationV0{
			successBatchAckV0("outbox-agent-ack-001"),
			failedBatchAckV0("outbox-agent-ack-002"),
		},
		Acker: ledger,
	})
	if err != nil {
		t.Fatalf("ack closure: %v", err)
	}
	if result.Status != OutboxDispatchBatchAckFailedV0 ||
		result.AckedCount != 1 ||
		result.FailedCount != 1 ||
		result.PendingCount != 1 {
		t.Fatalf("result=%+v", result)
	}
	if len(result.AckedMessages) != 1 || result.AckedMessages[0] != "outbox-agent-ack-001" {
		t.Fatalf("acked messages=%+v", result.AckedMessages)
	}

	pending, issues := ledger.ListPending(context.Background(), directorPendingFilterForBatchAckV0())
	if len(issues) > 0 {
		t.Fatalf("pending issues=%+v", issues)
	}
	if len(pending) != 2 {
		t.Fatalf("pending=%+v", pending)
	}
	if pending[0].MessageID == "outbox-agent-ack-001" ||
		pending[1].MessageID == "outbox-agent-ack-001" {
		t.Fatalf("success no debe quedar pendiente: %+v", pending)
	}
}

func TestRunOutboxDispatchBatchAckClosureV0ClosedWhenAllAcked(t *testing.T) {
	ledger := NewInMemoryOutboxLedgerV0()
	saveBatchPlanOutboxV0(t, ledger,
		batchPlanMessageV0("outbox-agent-ack-011"),
		batchPlanMessageV0("outbox-agent-ack-012"),
	)
	plan, err := RunOutboxDispatchBatchPlanV0(context.Background(), OutboxDispatchBatchPlanRequestV0{
		RunRef:     "run-batch-plan-001",
		TargetPort: orquestacoreworkflow.OutboxTargetAgentLauncherV0,
		MaxReady:   2,
		Reader:     ledger,
		Claimer:    ledger,
	})
	if err != nil {
		t.Fatalf("plan batch: %v", err)
	}

	result, err := RunOutboxDispatchBatchAckClosureV0(context.Background(), OutboxDispatchBatchAckRequestV0{
		Intents: plan.Intents,
		Acks: []orquestaoutboxdispatch.OutboxDispatchAckObservationV0{
			successBatchAckV0("outbox-agent-ack-011"),
			successBatchAckV0("outbox-agent-ack-012"),
		},
		Acker: ledger,
	})
	if err != nil {
		t.Fatalf("ack closure: %v", err)
	}
	if result.Status != OutboxDispatchBatchAckClosedV0 ||
		result.AckedCount != 2 ||
		result.PendingCount != 0 ||
		result.FailedCount != 0 {
		t.Fatalf("result=%+v", result)
	}
}

func TestRunOutboxDispatchBatchAckClosureV0InvalidWithoutAcker(t *testing.T) {
	result, err := RunOutboxDispatchBatchAckClosureV0(context.Background(), OutboxDispatchBatchAckRequestV0{})
	if err != nil {
		t.Fatalf("ack closure: %v", err)
	}
	if result.Status != OutboxDispatchBatchAckInvalidV0 || result.Issues != 1 {
		t.Fatalf("result=%+v", result)
	}
}

func successBatchAckV0(messageID string) orquestaoutboxdispatch.OutboxDispatchAckObservationV0 {
	return orquestaoutboxdispatch.OutboxDispatchAckObservationV0{
		MessageID:    messageID,
		RunID:        "run-batch-plan-001",
		TargetPort:   orquestacoreworkflow.OutboxTargetAgentLauncherV0,
		Status:       orquestaoutboxdispatch.OutboxDispatchAckObservationSuccessV0,
		DispatchRef:  "dispatch-" + messageID,
		EvidenceRefs: []string{"evidence-" + messageID},
	}
}

func failedBatchAckV0(messageID string) orquestaoutboxdispatch.OutboxDispatchAckObservationV0 {
	ack := successBatchAckV0(messageID)
	ack.Status = orquestaoutboxdispatch.OutboxDispatchAckObservationFailedV0
	return ack
}

func directorPendingFilterForBatchAckV0() orquestadirectorcycleoutbox.DirectorCycleOutboxPendingFilterV0 {
	return orquestadirectorcycleoutbox.DirectorCycleOutboxPendingFilterV0{
		RunRef:     "run-batch-plan-001",
		TargetPort: orquestacoreworkflow.OutboxTargetAgentLauncherV0,
	}
}
