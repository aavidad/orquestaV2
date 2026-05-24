package orquestastatefileoutbox

import orquestaoutboxdispatch "orquesta/modulos/orquesta-outbox-dispatch"

func (ledger *FileOutboxLedgerV0) ListPendingOutboxV0(
	filter orquestaoutboxdispatch.PendingOutboxFilterV0,
) ([]orquestaoutboxdispatch.OutboxPendingEntryV0, []orquestaoutboxdispatch.DispatchIssueV0) {
	if ledger == nil {
		return nil, dispatchPersistenceIssueV0()
	}
	filter, issues := normalizeDispatchFilterV0(filter)
	if len(issues) > 0 {
		return nil, issues
	}

	ledger.mu.Lock()
	defer ledger.mu.Unlock()

	entries := make([]orquestaoutboxdispatch.OutboxPendingEntryV0, 0, len(ledger.state.order))
	for _, messageID := range ledger.state.order {
		record := ledger.state.recordsByMessageID[messageID]
		if recordMatchesDispatchFilterV0(record, filter) {
			entries = append(entries, pendingEntryFromRecordV0(record))
		}
	}
	return entries, nil
}

func (ledger *FileOutboxLedgerV0) ClaimOutboxDispatchV0(
	claim orquestaoutboxdispatch.OutboxDispatchClaimV0,
) (orquestaoutboxdispatch.OutboxDispatchClaimResultV0, []orquestaoutboxdispatch.DispatchIssueV0) {
	if ledger == nil {
		return orquestaoutboxdispatch.OutboxDispatchClaimResultV0{}, dispatchPersistenceIssueV0()
	}
	claim = normalizeClaimV0(claim)
	if issues := validateClaimV0(claim); len(issues) > 0 {
		return orquestaoutboxdispatch.OutboxDispatchClaimResultV0{}, issues
	}

	ledger.mu.Lock()
	defer ledger.mu.Unlock()

	next := ledger.state.cloneV0()
	record := next.recordsByMessageID[claim.MessageID]
	if record == nil {
		return unclaimedResultV0(claim), []orquestaoutboxdispatch.DispatchIssueV0{
			dispatchIssueV0(errPayloadInvalidV0, "message_id", "mensaje pendiente no encontrado"),
		}
	}
	if issues := claimConflictsV0(record, claim); len(issues) > 0 {
		return unclaimedResultV0(claim), issues
	}
	if record.Claim != nil || record.Ack != nil {
		return orquestaoutboxdispatch.OutboxDispatchClaimResultV0{
			AlreadyClaimed: true,
			MessageID:      claim.MessageID,
			TargetPort:     claim.TargetPort,
		}, nil
	}
	record.Claim = &outboxLedgerClaimV0{
		MessageID:      claim.MessageID,
		RunID:          claim.RunID,
		TargetPort:     claim.TargetPort,
		IdempotencyKey: claim.IdempotencyKey,
	}
	if err := persistOutboxLedgerStateV0(ledger.path, next); err != nil {
		return unclaimedResultV0(claim), dispatchPersistenceIssueV0()
	}
	ledger.state = next
	return orquestaoutboxdispatch.OutboxDispatchClaimResultV0{
		Claimed:    true,
		MessageID:  claim.MessageID,
		TargetPort: claim.TargetPort,
	}, nil
}

func (ledger *FileOutboxLedgerV0) ReleaseOutboxDispatchClaimV0(
	claim orquestaoutboxdispatch.OutboxDispatchClaimV0,
) []orquestaoutboxdispatch.DispatchIssueV0 {
	if ledger == nil {
		return dispatchPersistenceIssueV0()
	}
	claim = normalizeClaimV0(claim)
	if issues := validateClaimV0(claim); len(issues) > 0 {
		return issues
	}

	ledger.mu.Lock()
	defer ledger.mu.Unlock()

	next := ledger.state.cloneV0()
	record := next.recordsByMessageID[claim.MessageID]
	if record == nil {
		return []orquestaoutboxdispatch.DispatchIssueV0{
			dispatchIssueV0(errPayloadInvalidV0, "message_id", "mensaje pendiente no encontrado"),
		}
	}
	if issues := claimConflictsV0(record, claim); len(issues) > 0 {
		return issues
	}
	if record.Ack != nil || record.Claim == nil {
		return nil
	}
	record.Claim = nil
	if err := persistOutboxLedgerStateV0(ledger.path, next); err != nil {
		return dispatchPersistenceIssueV0()
	}
	ledger.state = next
	return nil
}

func (ledger *FileOutboxLedgerV0) AckOutboxDispatchV0(
	ack orquestaoutboxdispatch.OutboxDispatchAckV0,
) []orquestaoutboxdispatch.DispatchIssueV0 {
	if ledger == nil {
		return dispatchPersistenceIssueV0()
	}
	ack = normalizeDispatchAckV0(ack)
	if issues := validateAckV0(ack); len(issues) > 0 {
		return issues
	}

	ledger.mu.Lock()
	defer ledger.mu.Unlock()

	next := ledger.state.cloneV0()
	record := next.recordsByMessageID[ack.MessageID]
	if record == nil {
		return []orquestaoutboxdispatch.DispatchIssueV0{
			dispatchIssueV0(errPayloadInvalidV0, "message_id", "mensaje pendiente no encontrado"),
		}
	}
	if issues := ackConflictsV0(record, ack); len(issues) > 0 {
		return issues
	}
	nextAck := storedAckFromDispatchV0(ack)
	if record.Ack != nil {
		if storedAckEqualV0(*record.Ack, nextAck) {
			return nil
		}
		return []orquestaoutboxdispatch.DispatchIssueV0{
			dispatchIssueV0(errIdempotencyConflictV0, "ack", "ack incompatible"),
		}
	}
	record.Ack = &nextAck
	record.Claim = nil
	if err := persistOutboxLedgerStateV0(ledger.path, next); err != nil {
		return dispatchPersistenceIssueV0()
	}
	ledger.state = next
	return nil
}

func pendingEntryFromRecordV0(record *outboxLedgerRecordV0) orquestaoutboxdispatch.OutboxPendingEntryV0 {
	message := record.Message
	return orquestaoutboxdispatch.OutboxPendingEntryV0{
		MessageID:      trimV0(message.MessageID),
		RunID:          trimV0(message.RunID),
		TargetPort:     trimV0(message.TargetPort),
		MessageType:    trimV0(message.MessageType),
		IdempotencyKey: trimV0(message.IdempotencyKey),
		CorrelationID:  trimV0(message.CorrelationID),
		PayloadVersion: trimV0(message.PayloadVersion),
		Payload:        append([]byte(nil), message.Payload...),
	}
}

func unclaimedResultV0(
	claim orquestaoutboxdispatch.OutboxDispatchClaimV0,
) orquestaoutboxdispatch.OutboxDispatchClaimResultV0 {
	return orquestaoutboxdispatch.OutboxDispatchClaimResultV0{
		MessageID:  claim.MessageID,
		TargetPort: claim.TargetPort,
	}
}
