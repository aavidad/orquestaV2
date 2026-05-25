package orquestapersistence

func (ledger *FileOutboxLedgerV0) ListOutboxDispatchSnapshotsV0(
	filter OutboxPendingFilterV0,
) ([]OutboxDispatchSnapshotV0, []OutboxLedgerIssueV0) {
	if ledger == nil {
		return nil, []OutboxLedgerIssueV0{outboxLedgerIssueV0(
			ErrPersistenciaNoDisponibleV0, "ledger", "ledger no disponible",
		)}
	}
	filter.RunID = trimV0(filter.RunID)
	filter.TargetPort = trimV0(filter.TargetPort)
	if filter.TargetPort != "" && !outboxLedgerTargetPortSupportedV0(filter.TargetPort) {
		return nil, []OutboxLedgerIssueV0{outboxLedgerIssueV0(
			ErrPayloadInvalidoV0, "target_port", "target_port no soportado",
		)}
	}
	ledger.mu.Lock()
	defer ledger.mu.Unlock()
	snapshots := []OutboxDispatchSnapshotV0{}
	for _, messageID := range ledger.state.order {
		record := ledger.state.recordsByMessageID[messageID]
		if record == nil || record.Ack == nil {
			continue
		}
		if filter.RunID != "" && record.Message.RunID != filter.RunID {
			continue
		}
		if filter.TargetPort != "" && record.Message.TargetPort != filter.TargetPort {
			continue
		}
		snapshots = append(snapshots, fileOutboxSnapshotFromAckV0(*record.Ack))
	}
	return snapshots, nil
}
