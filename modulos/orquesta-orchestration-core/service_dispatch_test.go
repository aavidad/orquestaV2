package orquestacionnucleoapp

import (
	"context"
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectorcycleoutbox "orquesta/modulos/orquesta-director-cycle-outbox"
	orquestaoutboxdispatch "orquesta/modulos/orquesta-outbox-dispatch"
)

func TestRunOutboxDispatchOnceV0DispatchesPendingCapacityMessage(t *testing.T) {
	runRef := "run-nucleo-dispatch-001"
	ledger := NewInMemoryOutboxLedgerV0()
	_, issues := ledger.SavePending(context.Background(), []orquestacoreworkflow.OutboxMessageV0{
		mustCapacityOutboxMessageV0(t, runRef, "outbox-ref-dispatch-001"),
	})
	if len(issues) > 0 {
		t.Fatalf("seed outbox issues=%+v", issues)
	}

	result, err := RunOutboxDispatchOnceV0(context.Background(), OutboxDispatchOnceRequestV0{
		RunRef:     runRef,
		TargetPort: orquestacoreworkflow.OutboxTargetCapacityV0,
		Reader:     ledger,
		Claimer:    ledger,
		Executor:   fakeDispatchExecutorV0{},
		Acker:      ledger,
	})
	if err != nil {
		t.Fatalf("dispatch once: %v", err)
	}
	if result.Status != string(orquestaoutboxdispatch.RunOutboxDispatchOnceDispatchedV0) {
		t.Fatalf("status=%s result=%+v", result.Status, result)
	}
	if result.MessageID != "outbox-ref-dispatch-001" || result.DispatchRef == "" {
		t.Fatalf("dispatch result incompleto: %+v", result)
	}

	pending, pendingIssues := ledger.ListPending(context.Background(),
		orquestadirectorcycleoutbox.DirectorCycleOutboxPendingFilterV0{RunRef: runRef},
	)
	if len(pendingIssues) > 0 || len(pending) != 0 {
		t.Fatalf("pending=%+v issues=%+v", pending, pendingIssues)
	}
}

type fakeDispatchExecutorV0 struct{}

func (fakeDispatchExecutorV0) ExecuteOutboxDispatchV0(
	intent orquestaoutboxdispatch.DispatchIntentV0,
) (orquestaoutboxdispatch.OutboxDispatchExecutionResultV0, error) {
	return orquestaoutboxdispatch.OutboxDispatchExecutionResultV0{
		DispatchRef:  "dispatch-ref-" + intent.MessageID,
		EvidenceRefs: []string{"evidence-ref-dispatch-001"},
	}, nil
}
