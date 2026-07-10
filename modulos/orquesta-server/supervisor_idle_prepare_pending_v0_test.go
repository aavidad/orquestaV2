package orquestaserver

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestRuntimeV0PrepareIdleSelfImprovementProcesoVivoNoEmiteErrorTerminalV0(t *testing.T) {
	now := time.Date(2026, 6, 25, 10, 0, 0, 0, time.UTC)
	store := &memoryStateStoreV0{}
	supervisor := &fakeSupervisorV0{
		selfResults: []IdleSelfImprovementResultV0{{
			Accepted:   false,
			RunRef:     "request-ref-autoprogramming-backlog-srv-task-022-live",
			RequestRef: "request-ref-autoprogramming-backlog-srv-task-022",
			Status:     "runtime_error",
			Message:    "supervise_timeout_waiting_for_ack",
			NextActions: []string{
				"wait_for_live_agent_ack",
				"reconcile_late_ack",
			},
			EvidenceRefs: []string{
				"external_wait_live_process",
				"ack_pending",
			},
		}},
	}
	runtime, err := NewRuntimeV0(ConfigV0{
		StateDir:      t.TempDir(),
		TickInterval:  time.Hour,
		AuditDisabled: true,
	}, RuntimeDepsV0{
		Supervisor: supervisor,
		StateStore: store,
		Clock:      fixedClockV0{now: now},
	})
	if err != nil {
		t.Fatalf("NewRuntimeV0: %v", err)
	}

	runtime.prepareIdleSelfImprovementBatchV0(context.Background(), supervisor, []IdleSelfImprovementRequestV0{{
		RequestRef: "request-ref-autoprogramming-backlog-srv-task-022",
	}})

	reason := store.snapshotV0().IdleSelfImprovementReason
	if !strings.HasPrefix(reason, "reconcile_pending") ||
		!strings.Contains(reason, "external_wait_live_process") ||
		!strings.Contains(reason, "ack_pending") {
		t.Fatalf("reason=%q", reason)
	}
	if store.snapshotV0().LastError != "" || len(store.snapshotV0().RecentErrors) != 0 {
		t.Fatalf("terminal error persisted: last_error=%q recent=%+v", store.snapshotV0().LastError, store.snapshotV0().RecentErrors)
	}
	if store.snapshotV0().IdleSelfImprovementOperationalMessage == nil ||
		store.snapshotV0().IdleSelfImprovementOperationalMessage.ReasonCode != "reconcile_pending" ||
		store.snapshotV0().IdleSelfImprovementOperationalMessage.Status != "reconcile_pending" {
		t.Fatalf("operational_message=%+v", store.snapshotV0().IdleSelfImprovementOperationalMessage)
	}
	if store.snapshotV0().IdleSelfImprovementRuns != 1 || store.snapshotV0().IdleSelfImprovementOK != 0 {
		t.Fatalf("counters runs=%d ok=%d", store.snapshotV0().IdleSelfImprovementRuns, store.snapshotV0().IdleSelfImprovementOK)
	}
}
