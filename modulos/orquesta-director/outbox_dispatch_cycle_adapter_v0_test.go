package orquestadirector

import (
	"context"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestapersistence "orquesta/modulos/orquesta-persistence"
)

type cyclePersistenceLedgerAdapterV0 struct {
	ledger *orquestapersistence.InMemoryOutboxLedgerV0
}

func newCyclePersistenceLedgerAdapterV0() *cyclePersistenceLedgerAdapterV0 {
	return &cyclePersistenceLedgerAdapterV0{ledger: orquestapersistence.NewInMemoryOutboxLedgerV0()}
}

func (a *cyclePersistenceLedgerAdapterV0) SavePending(
	ctx context.Context,
	messages []orquestacoreworkflow.OutboxMessageV0,
) ([]orquestacoreworkflow.OutboxMessageV0, []OutboxDispatchCycleIssueV0) {
	saved, issues := a.ledger.SavePending(ctx, messages)
	return saved, cycleIssuesFromPersistenceV0(issues)
}

func (a *cyclePersistenceLedgerAdapterV0) ListPending(
	ctx context.Context,
	filter OutboxPendingFilterV0,
) ([]orquestacoreworkflow.OutboxMessageV0, []OutboxDispatchCycleIssueV0) {
	pending, issues := a.ledger.ListPending(ctx, orquestapersistence.OutboxPendingFilterV0{
		RunID:      filter.RunID,
		TargetPort: filter.TargetPort,
	})
	return pending, cycleIssuesFromPersistenceV0(issues)
}

func (a *cyclePersistenceLedgerAdapterV0) MarkDispatched(
	ctx context.Context,
	ack OutboxDispatchAckV0,
) (OutboxDispatchSnapshotV0, []OutboxDispatchCycleIssueV0) {
	snapshot, issues := a.ledger.MarkDispatched(ctx, orquestapersistence.OutboxDispatchAckV0{
		MessageID:    ack.MessageID,
		RunID:        ack.RunID,
		TargetPort:   ack.TargetPort,
		Status:       ack.Status,
		DispatchRef:  ack.DispatchRef,
		DispatchedAt: ack.DispatchedAt,
		ErrorCode:    ack.ErrorCode,
		EvidenceRefs: ack.EvidenceRefs,
	})
	return cycleSnapshotFromPersistenceV0(snapshot), cycleIssuesFromPersistenceV0(issues)
}

func cycleIssuesFromPersistenceV0(
	issues []orquestapersistence.OutboxLedgerIssueV0,
) []OutboxDispatchCycleIssueV0 {
	result := make([]OutboxDispatchCycleIssueV0, 0, len(issues))
	for _, issue := range issues {
		result = append(result, OutboxDispatchCycleIssueV0{
			Code:    issue.Code,
			Field:   issue.Field,
			Message: issue.Message,
		})
	}
	return result
}

func cycleSnapshotFromPersistenceV0(
	snapshot orquestapersistence.OutboxDispatchSnapshotV0,
) OutboxDispatchSnapshotV0 {
	return OutboxDispatchSnapshotV0{
		MessageID:    snapshot.MessageID,
		RunID:        snapshot.RunID,
		TargetPort:   snapshot.TargetPort,
		Status:       snapshot.Status,
		DispatchRef:  snapshot.DispatchRef,
		DispatchedAt: snapshot.DispatchedAt,
		ErrorCode:    snapshot.ErrorCode,
		EvidenceRefs: append([]string(nil), snapshot.EvidenceRefs...),
	}
}
