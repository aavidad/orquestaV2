package orquestaoutboxdispatch

import (
	"errors"
	"testing"
)

func TestRunOutboxDispatchOnceV0SuccessExecuteAndAck(t *testing.T) {
	reader, claimer, executor, acker := runOncePortsV0(
		[]OutboxPendingEntryV0{validPendingEntryV0("outbox-101")},
		true,
		OutboxDispatchExecutionResultV0{
			DispatchRef:  "dispatch-101",
			EvidenceRefs: []string{"evidence-101"},
		},
		nil,
	)

	result, err := RunOutboxDispatchOnceV0(runOnceInputV0(reader, claimer, executor, acker))

	if err != nil {
		t.Fatalf("err=%v, want nil", err)
	}
	if result.Status != RunOutboxDispatchOnceDispatchedV0 {
		t.Fatalf("status=%s, want %s", result.Status, RunOutboxDispatchOnceDispatchedV0)
	}
	if len(reader.calls) != 1 || reader.calls[0].TargetPort != "agent_launcher" {
		t.Fatalf("reader calls=%+v", reader.calls)
	}
	if len(claimer.calls) != 1 || claimer.calls[0].TargetPort != "agent_launcher" {
		t.Fatalf("claimer calls=%+v", claimer.calls)
	}
	if len(executor.calls) != 1 || executor.calls[0].MessageID != "outbox-101" {
		t.Fatalf("executor calls=%+v", executor.calls)
	}
	if len(acker.calls) != 1 || acker.calls[0].DispatchRef != "dispatch-101" {
		t.Fatalf("ack calls=%+v", acker.calls)
	}
	if acker.calls[0].TargetPort != "agent_launcher" {
		t.Fatalf("ack target_port=%q", acker.calls[0].TargetPort)
	}
}

func TestRunOutboxDispatchOnceV0ExecutorErrorDoesNotAck(t *testing.T) {
	reader, claimer, executor, acker := runOncePortsV0(
		[]OutboxPendingEntryV0{validPendingEntryV0("outbox-102")},
		true,
		OutboxDispatchExecutionResultV0{},
		errors.New("executor failed"),
	)

	result, err := RunOutboxDispatchOnceV0(runOnceInputV0(reader, claimer, executor, acker))

	if err == nil {
		t.Fatal("err=nil, want executor error")
	}
	if result.Status != RunOutboxDispatchOnceDispatchFailedV0 {
		t.Fatalf("status=%s, want %s", result.Status, RunOutboxDispatchOnceDispatchFailedV0)
	}
	if len(executor.calls) != 1 {
		t.Fatalf("executor calls=%d, want 1", len(executor.calls))
	}
	if len(acker.calls) != 0 {
		t.Fatalf("ack calls=%+v, want none", acker.calls)
	}
}

func TestRunOutboxDispatchOnceV0NoPendingDoesNotExecute(t *testing.T) {
	reader, claimer, executor, acker := runOncePortsV0(nil, true, OutboxDispatchExecutionResultV0{}, nil)

	result, err := RunOutboxDispatchOnceV0(runOnceInputV0(reader, claimer, executor, acker))

	if err != nil {
		t.Fatalf("err=%v, want nil", err)
	}
	if result.Status != RunOutboxDispatchOnceNoPendingV0 {
		t.Fatalf("status=%s, want %s", result.Status, RunOutboxDispatchOnceNoPendingV0)
	}
	if len(claimer.calls) != 0 || len(executor.calls) != 0 || len(acker.calls) != 0 {
		t.Fatalf("calls claimer=%d executor=%d ack=%d",
			len(claimer.calls), len(executor.calls), len(acker.calls))
	}
}

func TestRunOutboxDispatchOnceV0ClaimedDoesNotDuplicate(t *testing.T) {
	reader, claimer, executor, acker := runOncePortsV0(
		[]OutboxPendingEntryV0{validPendingEntryV0("outbox-103")},
		false,
		OutboxDispatchExecutionResultV0{},
		nil,
	)

	result, err := RunOutboxDispatchOnceV0(runOnceInputV0(reader, claimer, executor, acker))

	if err != nil {
		t.Fatalf("err=%v, want nil", err)
	}
	if result.Status != RunOutboxDispatchOnceAlreadyClaimedV0 {
		t.Fatalf("status=%s, want %s", result.Status, RunOutboxDispatchOnceAlreadyClaimedV0)
	}
	if len(claimer.calls) != 1 {
		t.Fatalf("claimer calls=%d, want 1", len(claimer.calls))
	}
	if len(executor.calls) != 0 || len(acker.calls) != 0 {
		t.Fatalf("calls executor=%d ack=%d, want none", len(executor.calls), len(acker.calls))
	}
}

func runOnceInputV0(
	reader *fakePendingReaderV0,
	claimer *fakeClaimerPortV0,
	executor *fakeExecutorPortV0,
	acker *fakeAckPortV0,
) RunOutboxDispatchOnceInputV0 {
	return RunOutboxDispatchOnceInputV0{
		RunID:      "run-001",
		TargetPort: "agent_launcher",
		Reader:     reader,
		Claimer:    claimer,
		Executor:   executor,
		Acker:      acker,
	}
}
