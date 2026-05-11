package orquestadirector

import (
	"context"
	"encoding/json"
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
)

func validCycleInputV0() OutboxDispatchCycleInputV0 {
	return OutboxDispatchCycleInputV0{
		Ledger:       newCyclePersistenceLedgerAdapterV0(),
		Dispatcher:   &cycleFakeOutboxDispatcherV0{},
		RunID:        cycleRunIDV0,
		TargetPort:   cycleTargetPortV0,
		DispatchedAt: cycleDispatchedAtV0,
	}
}

func validCycleOutboxMessageV0(t *testing.T, suffix string) orquestacoreworkflow.OutboxMessageV0 {
	t.Helper()
	payload := orquestacoreworkflow.LaunchRuntimeAgentRequestV0{
		AgentRequestID:     "agent-request-cycle-" + suffix,
		RunID:              cycleRunIDV0,
		PhaseID:            string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0),
		TaskRef:            "task-ref-cycle-" + suffix,
		CapacityRequestRef: "capacity-ref-cycle-" + suffix,
		Role:               "builder",
		Summary:            "Implementar microtarea compacta.",
		EvidenceRefs:       []string{"evidence-ref-cycle-" + suffix},
	}
	return orquestacoreworkflow.OutboxMessageV0{
		MessageID:        "outbox-cycle-" + suffix,
		MessageType:      orquestacoreworkflow.OutboxMessageLaunchRuntimeAgentV0,
		RunID:            cycleRunIDV0,
		IdempotencyKey:   "idem-cycle-" + suffix,
		CorrelationID:    "corr-cycle-001",
		CausationEventID: "event-cycle-" + suffix,
		TargetPort:       cycleTargetPortV0,
		PayloadVersion:   orquestacoreworkflow.OutboxPayloadVersionV0,
		Payload:          mustCyclePayloadV0(t, payload),
	}
}

func mustCyclePayloadV0(t *testing.T, payload any) json.RawMessage {
	t.Helper()
	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	return data
}

func cycleListPendingV0(
	t *testing.T,
	ctx context.Context,
	ledger *cyclePersistenceLedgerAdapterV0,
) []orquestacoreworkflow.OutboxMessageV0 {
	t.Helper()
	pending, issues := ledger.ListPending(ctx, OutboxPendingFilterV0{
		RunID:      cycleRunIDV0,
		TargetPort: cycleTargetPortV0,
	})
	if len(issues) != 0 {
		t.Fatalf("list pending issues=%+v", issues)
	}
	return pending
}

func cycleMessageIDsV0(messages []orquestacoreworkflow.OutboxMessageV0) []string {
	ids := make([]string, 0, len(messages))
	for _, message := range messages {
		ids = append(ids, message.MessageID)
	}
	return ids
}

func cycleSnapshotStatusesV0(snapshots []OutboxDispatchSnapshotV0) []string {
	statuses := make([]string, 0, len(snapshots))
	for _, snapshot := range snapshots {
		statuses = append(statuses, snapshot.Status)
	}
	return statuses
}
