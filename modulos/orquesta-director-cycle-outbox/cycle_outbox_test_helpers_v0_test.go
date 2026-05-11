package orquestadirectorcycleoutbox

import (
	"context"
	"encoding/json"
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestapersistence "orquesta/modulos/orquesta-persistence"
)

const cycleOutboxRunRefV0 = "run-cycle-outbox-001"

type cycleOutboxPersistenceLedgerAdapterV0 struct {
	ledger *orquestapersistence.InMemoryOutboxLedgerV0
}

func newCycleOutboxPersistenceLedgerAdapterV0() *cycleOutboxPersistenceLedgerAdapterV0 {
	return &cycleOutboxPersistenceLedgerAdapterV0{ledger: orquestapersistence.NewInMemoryOutboxLedgerV0()}
}

func (adapter *cycleOutboxPersistenceLedgerAdapterV0) SavePending(
	ctx context.Context,
	messages []orquestacoreworkflow.OutboxMessageV0,
) ([]orquestacoreworkflow.OutboxMessageV0, []DirectorCycleOutboxIssueV0) {
	saved, issues := adapter.ledger.SavePending(ctx, messages)
	return saved, cycleOutboxPersistenceIssuesV0(issues)
}

func (adapter *cycleOutboxPersistenceLedgerAdapterV0) ListPending(
	ctx context.Context,
	filter DirectorCycleOutboxPendingFilterV0,
) ([]orquestacoreworkflow.OutboxMessageV0, []DirectorCycleOutboxIssueV0) {
	pending, issues := adapter.ledger.ListPending(ctx, orquestapersistence.OutboxPendingFilterV0{
		RunID:      filter.RunRef,
		TargetPort: filter.TargetPort,
	})
	return pending, cycleOutboxPersistenceIssuesV0(issues)
}

func cycleOutboxPersistenceIssuesV0(
	issues []orquestapersistence.OutboxLedgerIssueV0,
) []DirectorCycleOutboxIssueV0 {
	if len(issues) == 0 {
		return nil
	}
	out := make([]DirectorCycleOutboxIssueV0, 0, len(issues))
	for _, issue := range issues {
		out = append(out, DirectorCycleOutboxIssueV0{
			Code:    issue.Code,
			Field:   issue.Field,
			Message: issue.Message,
		})
	}
	return out
}

func mustCycleOutboxCapacityMessageV0(
	t *testing.T,
	runRef string,
	messageID string,
) orquestacoreworkflow.OutboxMessageV0 {
	t.Helper()
	payload, err := json.Marshal(orquestacoreworkflow.CapacityDecisionRequestV0{
		CapacityRequestID:          "capacity-ref-cycle-outbox-001",
		RunID:                      runRef,
		PhaseID:                    string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0),
		TaskRef:                    "task-ref-cycle-outbox-001",
		ReasonCode:                 "programacion_siguiente_paso",
		Summary:                    "Decision compacta para siguiente paso.",
		MinimumRecommendedCapacity: orquestacoreworkflow.OrchestrationCapacityMediumV0,
		EvidenceRefs:               []string{"evidence-ref-cycle-outbox-001"},
	})
	if err != nil {
		t.Fatalf("payload: %v", err)
	}
	message := orquestacoreworkflow.OutboxMessageV0{
		MessageID:      messageID,
		MessageType:    orquestacoreworkflow.OutboxMessageRequestCapacityDecisionV0,
		RunID:          runRef,
		IdempotencyKey: "idem-" + messageID,
		CorrelationID:  "corr-cycle-outbox-001",
		TargetPort:     orquestacoreworkflow.OutboxTargetCapacityV0,
		PayloadVersion: orquestacoreworkflow.OutboxPayloadVersionV0,
		Payload:        payload,
	}
	if err := orquestacoreworkflow.ValidateOutboxMessageV0(message); err != nil {
		t.Fatalf("outbox message: %v", err)
	}
	return message
}

func cycleOutboxStartCommandV0(t *testing.T) orquestacoreworkflow.OrchestrationCommandV0 {
	t.Helper()
	command, err := orquestacoreworkflow.NewStartRunCommandV0(
		cycleOutboxCommandMetaV0("cmd-cycle-outbox-start", "start"),
		orquestacoreworkflow.StartRunCommandPayloadV0{
			ProjectRef: "project-ref-cycle-outbox-001",
			AppSpecRef: "appspec-ref-cycle-outbox-001",
		},
	)
	if err != nil {
		t.Fatalf("start command: %v", err)
	}
	return command
}

func cycleOutboxOpenProgramacionCommandV0(t *testing.T) orquestacoreworkflow.OrchestrationCommandV0 {
	t.Helper()
	command, err := orquestacoreworkflow.NewOpenPhaseCommandV0(
		cycleOutboxCommandMetaV0("cmd-cycle-outbox-open-programacion", "open-programacion"),
		orquestacoreworkflow.OpenPhaseCommandPayloadV0{
			PhaseID: string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0),
			Reason:  "Preparar programacion",
		},
	)
	if err != nil {
		t.Fatalf("open command: %v", err)
	}
	return command
}

func cycleOutboxCommandMetaV0(commandID string, suffix string) orquestacoreworkflow.OrchestrationCommandMetaV0 {
	return orquestacoreworkflow.OrchestrationCommandMetaV0{
		CommandID:      commandID,
		RunID:          cycleOutboxRunRefV0,
		IdempotencyKey: "idem-cycle-outbox-" + suffix,
		CorrelationID:  "corr-cycle-outbox-001",
		RequestedBy:    "director-cycle-outbox-test",
		OccurredAt:     "2026-05-06T12:30:00Z",
	}
}

type failingCycleOutboxLedgerV0 struct{}

func (failingCycleOutboxLedgerV0) SavePending(
	context.Context,
	[]orquestacoreworkflow.OutboxMessageV0,
) ([]orquestacoreworkflow.OutboxMessageV0, []DirectorCycleOutboxIssueV0) {
	return nil, []DirectorCycleOutboxIssueV0{{Code: "ledger_failed", Field: "save_pending"}}
}

func (failingCycleOutboxLedgerV0) ListPending(
	context.Context,
	DirectorCycleOutboxPendingFilterV0,
) ([]orquestacoreworkflow.OutboxMessageV0, []DirectorCycleOutboxIssueV0) {
	return nil, []DirectorCycleOutboxIssueV0{{Code: "ledger_failed", Field: "list_pending"}}
}
