package orquestacionnucleoapp

import (
	"context"
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestaoutboxdispatch "orquesta/modulos/orquesta-outbox-dispatch"
)

func TestRunOutboxDispatchBatchPlanV0ClaimsSeveralIntents(t *testing.T) {
	ledger := NewInMemoryOutboxLedgerV0()
	saveBatchPlanOutboxV0(t, ledger,
		batchPlanMessageV0("outbox-agent-batch-001"),
		batchPlanMessageV0("outbox-agent-batch-002"),
		batchPlanMessageV0("outbox-agent-batch-003"),
	)

	result, err := RunOutboxDispatchBatchPlanV0(context.Background(), OutboxDispatchBatchPlanRequestV0{
		RunRef:     "run-batch-plan-001",
		TargetPort: orquestacoreworkflow.OutboxTargetAgentLauncherV0,
		MaxReady:   2,
		Reader:     ledger,
		Claimer:    ledger,
	})
	if err != nil {
		t.Fatalf("RunOutboxDispatchBatchPlanV0: %v", err)
	}
	if result.Status != OutboxDispatchBatchPlannedV0 {
		t.Fatalf("status=%s result=%+v", result.Status, result)
	}
	if len(result.Intents) != 2 ||
		result.Intents[0].MessageID != "outbox-agent-batch-001" ||
		result.Intents[1].MessageID != "outbox-agent-batch-002" {
		t.Fatalf("intents=%+v", result.Intents)
	}
	if len(result.ClaimedMessageIDs) != 2 {
		t.Fatalf("claimed=%+v", result.ClaimedMessageIDs)
	}

	next, err := RunOutboxDispatchBatchPlanV0(context.Background(), OutboxDispatchBatchPlanRequestV0{
		RunRef:     "run-batch-plan-001",
		TargetPort: orquestacoreworkflow.OutboxTargetAgentLauncherV0,
		MaxReady:   2,
		Reader:     ledger,
		Claimer:    ledger,
	})
	if err != nil {
		t.Fatalf("RunOutboxDispatchBatchPlanV0 second: %v", err)
	}
	if next.Status != OutboxDispatchBatchPlannedV0 ||
		len(next.Intents) != 1 ||
		next.Intents[0].MessageID != "outbox-agent-batch-003" {
		t.Fatalf("next=%+v", next)
	}
}

func TestRunOutboxDispatchBatchPlanV0SkipsAlreadyClaimedInsideBatch(t *testing.T) {
	ledger := NewInMemoryOutboxLedgerV0()
	saveBatchPlanOutboxV0(t, ledger,
		batchPlanMessageV0("outbox-agent-batch-011"),
		batchPlanMessageV0("outbox-agent-batch-012"),
		batchPlanMessageV0("outbox-agent-batch-013"),
	)
	_, issues := ledger.ClaimOutboxDispatchV0(orquestaoutboxdispatch.OutboxDispatchClaimV0{
		MessageID:      "outbox-agent-batch-011",
		RunID:          "run-batch-plan-001",
		TargetPort:     orquestacoreworkflow.OutboxTargetAgentLauncherV0,
		IdempotencyKey: "idem-outbox-agent-batch-011",
	})
	if len(issues) > 0 {
		t.Fatalf("preclaim issues=%+v", issues)
	}

	result, err := RunOutboxDispatchBatchPlanV0(context.Background(), OutboxDispatchBatchPlanRequestV0{
		RunRef:     "run-batch-plan-001",
		TargetPort: orquestacoreworkflow.OutboxTargetAgentLauncherV0,
		MaxReady:   3,
		Reader:     ledger,
		Claimer:    ledger,
	})
	if err != nil {
		t.Fatalf("RunOutboxDispatchBatchPlanV0: %v", err)
	}
	if result.Status != OutboxDispatchBatchPlannedV0 {
		t.Fatalf("status=%s result=%+v", result.Status, result)
	}
	if len(result.Intents) != 2 ||
		result.Intents[0].MessageID != "outbox-agent-batch-012" ||
		result.Intents[1].MessageID != "outbox-agent-batch-013" {
		t.Fatalf("intents=%+v", result.Intents)
	}
}

func TestRunOutboxDispatchBatchPlanV0InvalidWithoutPorts(t *testing.T) {
	result, err := RunOutboxDispatchBatchPlanV0(context.Background(), OutboxDispatchBatchPlanRequestV0{})
	if err != nil {
		t.Fatalf("RunOutboxDispatchBatchPlanV0: %v", err)
	}
	if result.Status != OutboxDispatchBatchInvalidV0 || result.Issues != 3 {
		t.Fatalf("result=%+v", result)
	}
}

func TestRunOutboxDispatchBatchPlanV0AlreadyClaimedWhenNoClaimableIntent(t *testing.T) {
	ledger := NewInMemoryOutboxLedgerV0()
	saveBatchPlanOutboxV0(t, ledger,
		batchPlanMessageV0("outbox-agent-batch-021"),
		batchPlanMessageV0("outbox-agent-batch-022"),
	)
	for _, messageID := range []string{"outbox-agent-batch-021", "outbox-agent-batch-022"} {
		_, issues := ledger.ClaimOutboxDispatchV0(orquestaoutboxdispatch.OutboxDispatchClaimV0{
			MessageID:      messageID,
			RunID:          "run-batch-plan-001",
			TargetPort:     orquestacoreworkflow.OutboxTargetAgentLauncherV0,
			IdempotencyKey: "idem-" + messageID,
		})
		if len(issues) > 0 {
			t.Fatalf("preclaim issues=%+v", issues)
		}
	}

	result, err := RunOutboxDispatchBatchPlanV0(context.Background(), OutboxDispatchBatchPlanRequestV0{
		RunRef:     "run-batch-plan-001",
		TargetPort: orquestacoreworkflow.OutboxTargetAgentLauncherV0,
		MaxReady:   2,
		Reader:     ledger,
		Claimer:    ledger,
	})
	if err != nil {
		t.Fatalf("RunOutboxDispatchBatchPlanV0: %v", err)
	}
	if result.Status != OutboxDispatchBatchAlreadyClaimedV0 ||
		len(result.Intents) != 0 {
		t.Fatalf("result=%+v", result)
	}
}

func saveBatchPlanOutboxV0(
	t *testing.T,
	ledger *InMemoryOutboxLedgerV0,
	messages ...orquestacoreworkflow.OutboxMessageV0,
) {
	t.Helper()
	_, issues := ledger.SavePending(context.Background(), messages)
	if len(issues) > 0 {
		t.Fatalf("save pending issues=%+v", issues)
	}
}

func batchPlanMessageV0(messageID string) orquestacoreworkflow.OutboxMessageV0 {
	return orquestacoreworkflow.OutboxMessageV0{
		MessageID:      messageID,
		MessageType:    orquestacoreworkflow.OutboxMessageLaunchRuntimeAgentV0,
		RunID:          "run-batch-plan-001",
		IdempotencyKey: "idem-" + messageID,
		CorrelationID:  "corr-" + messageID,
		TargetPort:     orquestacoreworkflow.OutboxTargetAgentLauncherV0,
		PayloadVersion: orquestacoreworkflow.OutboxPayloadVersionV0,
		Payload:        []byte(`{"ref":"` + messageID + `"}`),
	}
}
