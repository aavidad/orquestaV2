package orquestacionnucleoapp

import (
	"context"
	"fmt"
	"sync"
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestaoutboxdispatch "orquesta/modulos/orquesta-outbox-dispatch"
)

func TestInMemoryRunStoreV0SoportaRunsConcurrentesBasicos(t *testing.T) {
	store := NewInMemoryRunStoreV0()
	ctx := context.Background()
	runs := make([]orquestacoreworkflow.OrchestrationRunV0, 12)
	for index := range runs {
		runs[index] = mustActiveProgrammingRunV0(t, fmt.Sprintf("run-memory-concurrent-%03d", index))
	}
	var wg sync.WaitGroup

	for index := range runs {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			run := runs[index]
			if err := store.SaveRunV0(ctx, run); err != nil {
				t.Errorf("save run %d: %v", index, err)
				return
			}
			loaded, err := store.LoadRunV0(ctx, run.RunID)
			if err != nil {
				t.Errorf("load run %d: %v", index, err)
				return
			}
			if loaded.RunID != run.RunID {
				t.Errorf("loaded run=%s want=%s", loaded.RunID, run.RunID)
			}
		}(index)
	}
	wg.Wait()
}

func TestInMemoryOutboxLedgerV0ClaimsConcurrentesNoDuplican(t *testing.T) {
	ledger := NewInMemoryOutboxLedgerV0()
	message := orquestacoreworkflow.OutboxMessageV0{
		MessageID:   "message-ref-memory-concurrent-001",
		RunID:       "run-memory-concurrent-outbox-001",
		TargetPort:  "agent_launcher",
		MessageType: "LaunchRuntimeAgent",
	}
	if _, issues := ledger.SavePending(context.Background(), []orquestacoreworkflow.OutboxMessageV0{message}); len(issues) > 0 {
		t.Fatalf("save issues=%+v", issues)
	}

	var wg sync.WaitGroup
	claimed := make(chan bool, 12)
	for index := 0; index < 12; index++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			result, issues := ledger.ClaimOutboxDispatchV0(orquestaoutboxdispatch.OutboxDispatchClaimV0{
				MessageID:  message.MessageID,
				TargetPort: message.TargetPort,
			})
			if len(issues) > 0 {
				t.Errorf("claim issues=%+v", issues)
				return
			}
			claimed <- result.Claimed
		}()
	}
	wg.Wait()
	close(claimed)

	totalClaimed := 0
	for value := range claimed {
		if value {
			totalClaimed++
		}
	}
	if totalClaimed != 1 {
		t.Fatalf("claimed=%d want=1", totalClaimed)
	}
}
