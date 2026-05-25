package orquestacionnucleoapp

import (
	"context"
	"errors"
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestaoutboxdispatch "orquesta/modulos/orquesta-outbox-dispatch"
)

func TestRunOutboxDispatchBatchV0ExecutesBatchThroughPortAndAcksSuccesses(t *testing.T) {
	ledger := NewInMemoryOutboxLedgerV0()
	saveBatchPlanOutboxV0(t, ledger,
		batchPlanMessageV0("outbox-agent-run-001"),
		batchPlanMessageV0("outbox-agent-run-002"),
	)

	executor := recordingBatchExecutorV0{mode: "success"}
	result, err := RunOutboxDispatchBatchV0(context.Background(), OutboxDispatchBatchRunRequestV0{
		RunRef:     "run-batch-plan-001",
		TargetPort: orquestacoreworkflow.OutboxTargetAgentLauncherV0,
		MaxReady:   2,
		Reader:     ledger,
		Claimer:    ledger,
		Executor:   &executor,
		Acker:      ledger,
	})
	if err != nil {
		t.Fatalf("RunOutboxDispatchBatchV0: %v", err)
	}
	if result.Status != OutboxDispatchBatchRunDispatchedV0 ||
		result.PlannedCount != 2 ||
		result.AckedCount != 2 ||
		len(result.AckedMessages) != 2 {
		t.Fatalf("result=%+v", result)
	}
	if executor.gotCount != 2 {
		t.Fatalf("executor got count=%d", executor.gotCount)
	}
	pending, issues := ledger.ListPending(context.Background(), directorPendingFilterForBatchAckV0())
	if len(issues) > 0 || len(pending) != 0 {
		t.Fatalf("pending=%+v issues=%+v", pending, issues)
	}
}

func TestRunOutboxDispatchBatchV0KeepsPartialBatchVisible(t *testing.T) {
	ledger := NewInMemoryOutboxLedgerV0()
	saveBatchPlanOutboxV0(t, ledger,
		batchPlanMessageV0("outbox-agent-run-011"),
		batchPlanMessageV0("outbox-agent-run-012"),
	)

	executor := recordingBatchExecutorV0{mode: "partial"}
	result, err := RunOutboxDispatchBatchV0(context.Background(), OutboxDispatchBatchRunRequestV0{
		RunRef:     "run-batch-plan-001",
		TargetPort: orquestacoreworkflow.OutboxTargetAgentLauncherV0,
		MaxReady:   2,
		Reader:     ledger,
		Claimer:    ledger,
		Executor:   &executor,
		Acker:      ledger,
	})
	if err != nil {
		t.Fatalf("RunOutboxDispatchBatchV0: %v", err)
	}
	if result.Status != OutboxDispatchBatchRunAckFailedV0 ||
		result.AckedCount != 1 ||
		result.FailedCount != 1 {
		t.Fatalf("result=%+v", result)
	}
	pending, issues := ledger.ListPending(context.Background(), directorPendingFilterForBatchAckV0())
	if len(issues) > 0 || len(pending) != 1 {
		t.Fatalf("pending=%+v issues=%+v", pending, issues)
	}

	retry := recordingBatchExecutorV0{mode: "success"}
	retried, err := RunOutboxDispatchBatchV0(context.Background(), OutboxDispatchBatchRunRequestV0{
		RunRef:     "run-batch-plan-001",
		TargetPort: orquestacoreworkflow.OutboxTargetAgentLauncherV0,
		MaxReady:   2,
		Reader:     ledger,
		Claimer:    ledger,
		Executor:   &retry,
		Acker:      ledger,
	})
	if err != nil {
		t.Fatalf("retry RunOutboxDispatchBatchV0: %v", err)
	}
	if retried.Status != OutboxDispatchBatchRunDispatchedV0 ||
		retried.AckedCount != 1 ||
		len(retried.AckedMessages) != 1 {
		t.Fatalf("retried=%+v", retried)
	}
}

func TestRunOutboxDispatchBatchV0RejectsMissingExecutor(t *testing.T) {
	result, err := RunOutboxDispatchBatchV0(context.Background(), OutboxDispatchBatchRunRequestV0{})
	if err != nil {
		t.Fatalf("RunOutboxDispatchBatchV0: %v", err)
	}
	if result.Status != OutboxDispatchBatchRunInvalidV0 || result.Issues != 1 {
		t.Fatalf("result=%+v", result)
	}
}

func TestRunOutboxDispatchBatchV0PropagatesExecutorError(t *testing.T) {
	ledger := NewInMemoryOutboxLedgerV0()
	saveBatchPlanOutboxV0(t, ledger, batchPlanMessageV0("outbox-agent-run-021"))

	_, err := RunOutboxDispatchBatchV0(context.Background(), OutboxDispatchBatchRunRequestV0{
		RunRef:     "run-batch-plan-001",
		TargetPort: orquestacoreworkflow.OutboxTargetAgentLauncherV0,
		MaxReady:   1,
		Reader:     ledger,
		Claimer:    ledger,
		Executor:   &recordingBatchExecutorV0{err: errors.New("batch executor unavailable")},
		Acker:      ledger,
	})
	if err == nil {
		t.Fatalf("expected executor error")
	}

	retry := recordingBatchExecutorV0{mode: "success"}
	result, err := RunOutboxDispatchBatchV0(context.Background(), OutboxDispatchBatchRunRequestV0{
		RunRef:     "run-batch-plan-001",
		TargetPort: orquestacoreworkflow.OutboxTargetAgentLauncherV0,
		MaxReady:   1,
		Reader:     ledger,
		Claimer:    ledger,
		Executor:   &retry,
		Acker:      ledger,
	})
	if err != nil {
		t.Fatalf("retry RunOutboxDispatchBatchV0: %v", err)
	}
	if result.Status != OutboxDispatchBatchRunDispatchedV0 || result.AckedCount != 1 {
		t.Fatalf("retry result=%+v", result)
	}
}

func TestRunOutboxDispatchBatchV0NoReclamaSiCapacidadVivaEstaLlena(t *testing.T) {
	ledger := NewInMemoryOutboxLedgerV0()
	saveBatchPlanOutboxV0(t, ledger,
		batchPlanMessageV0("outbox-agent-run-capacity-001"),
		batchPlanMessageV0("outbox-agent-run-capacity-002"),
	)
	executor := recordingBatchExecutorV0{mode: "success"}
	result, err := RunOutboxDispatchBatchV0(context.Background(), OutboxDispatchBatchRunRequestV0{
		RunRef:       "run-batch-plan-001",
		TargetPort:   orquestacoreworkflow.OutboxTargetAgentLauncherV0,
		MessageType:  orquestacoreworkflow.OutboxMessageLaunchRuntimeAgentV0,
		MaxReady:     2,
		CapacityGate: &staticLiveProcessCapacityGateForBatchTestV0{granted: 0, live: 10, limit: 10},
		Reader:       ledger,
		Claimer:      ledger,
		Executor:     &executor,
		Acker:        ledger,
	})
	if err != nil {
		t.Fatalf("RunOutboxDispatchBatchV0: %v", err)
	}
	if result.Status != OutboxDispatchBatchRunCapacityBlockedV0 ||
		result.PlannedCount != 0 ||
		result.CapacityLive != 10 ||
		result.CapacityLimit != 10 ||
		executor.gotCount != 0 {
		t.Fatalf("result=%+v executor=%+v", result, executor)
	}
	pending, issues := ledger.ListPending(context.Background(), directorPendingFilterForBatchAckV0())
	if len(issues) > 0 || len(pending) != 2 {
		t.Fatalf("pending=%+v issues=%+v", pending, issues)
	}
}

func TestRunOutboxDispatchBatchV0CapaPlanAlHuecoDeCapacidadViva(t *testing.T) {
	ledger := NewInMemoryOutboxLedgerV0()
	saveBatchPlanOutboxV0(t, ledger,
		batchPlanMessageV0("outbox-agent-run-capacity-011"),
		batchPlanMessageV0("outbox-agent-run-capacity-012"),
		batchPlanMessageV0("outbox-agent-run-capacity-013"),
	)
	gate := &staticLiveProcessCapacityGateForBatchTestV0{granted: 1, live: 9, limit: 10}
	executor := recordingBatchExecutorV0{mode: "success"}
	result, err := RunOutboxDispatchBatchV0(context.Background(), OutboxDispatchBatchRunRequestV0{
		RunRef:       "run-batch-plan-001",
		TargetPort:   orquestacoreworkflow.OutboxTargetAgentLauncherV0,
		MessageType:  orquestacoreworkflow.OutboxMessageLaunchRuntimeAgentV0,
		MaxReady:     3,
		CapacityGate: gate,
		Reader:       ledger,
		Claimer:      ledger,
		Executor:     &executor,
		Acker:        ledger,
	})
	if err != nil {
		t.Fatalf("RunOutboxDispatchBatchV0: %v", err)
	}
	if result.Status != OutboxDispatchBatchRunDispatchedV0 ||
		result.PlannedCount != 1 ||
		result.AckedCount != 1 ||
		result.CapacityGranted != 1 ||
		executor.gotCount != 1 ||
		gate.released != 1 {
		t.Fatalf("result=%+v executor=%+v gate=%+v", result, executor, gate)
	}
	pending, issues := ledger.ListPending(context.Background(), directorPendingFilterForBatchAckV0())
	if len(issues) > 0 || len(pending) != 2 {
		t.Fatalf("pending=%+v issues=%+v", pending, issues)
	}
}

func TestRunOutboxDispatchBatchV0NoReservaCapacidadParaStopRuntimeAgent(t *testing.T) {
	ledger := NewInMemoryOutboxLedgerV0()
	message := batchPlanMessageV0("outbox-agent-run-stop-capacity-001")
	message.MessageType = orquestacoreworkflow.OutboxMessageStopRuntimeAgentV0
	saveBatchPlanOutboxV0(t, ledger, message)

	gate := &staticLiveProcessCapacityGateForBatchTestV0{granted: 0, live: 10, limit: 10}
	executor := recordingBatchExecutorV0{mode: "success"}
	result, err := RunOutboxDispatchBatchV0(context.Background(), OutboxDispatchBatchRunRequestV0{
		RunRef:       "run-batch-plan-001",
		TargetPort:   orquestacoreworkflow.OutboxTargetAgentLauncherV0,
		MessageType:  orquestacoreworkflow.OutboxMessageStopRuntimeAgentV0,
		MaxReady:     1,
		CapacityGate: gate,
		Reader:       ledger,
		Claimer:      ledger,
		Executor:     &executor,
		Acker:        ledger,
	})
	if err != nil {
		t.Fatalf("RunOutboxDispatchBatchV0: %v", err)
	}
	if result.Status != OutboxDispatchBatchRunDispatchedV0 ||
		result.PlannedCount != 1 ||
		result.CapacityRef != "" ||
		gate.calls != 0 ||
		executor.gotCount != 1 {
		t.Fatalf("result=%+v executor=%+v gate=%+v", result, executor, gate)
	}
}

type recordingBatchExecutorV0 struct {
	mode     string
	err      error
	gotCount int
}

func (executor *recordingBatchExecutorV0) ExecuteOutboxDispatchBatchV0(
	_ context.Context,
	intents []orquestaoutboxdispatch.DispatchIntentV0,
) ([]orquestaoutboxdispatch.OutboxDispatchAckObservationV0, error) {
	executor.gotCount = len(intents)
	if executor.err != nil {
		return nil, executor.err
	}
	acks := make([]orquestaoutboxdispatch.OutboxDispatchAckObservationV0, 0, len(intents))
	for index, intent := range intents {
		if executor.mode == "partial" && index == len(intents)-1 {
			acks = append(acks, failedBatchAckV0(intent.MessageID))
			continue
		}
		acks = append(acks, successBatchAckV0(intent.MessageID))
	}
	return acks, nil
}

type staticLiveProcessCapacityGateForBatchTestV0 struct {
	granted  int
	live     int
	limit    int
	released int
	calls    int
}

func (gate *staticLiveProcessCapacityGateForBatchTestV0) ReserveLiveProcessCapacityV0(
	context.Context,
	LiveProcessCapacityReservationRequestV0,
) (LiveProcessCapacityReservationV0, error) {
	gate.calls++
	return LiveProcessCapacityReservationV0{
		ReservationRef: "live-capacity-reservation-test",
		Granted:        gate.granted,
		Live:           gate.live,
		Limit:          gate.limit,
	}, nil
}

func (gate *staticLiveProcessCapacityGateForBatchTestV0) ReleaseLiveProcessCapacityV0(
	context.Context,
	LiveProcessCapacityReservationV0,
) error {
	gate.released++
	return nil
}
