package orquestapersistence

import orquestaoutboxdispatch "orquesta/modulos/orquesta-outbox-dispatch"

func (ledger *FileOutboxLedgerV0) ListPendingOutboxV0(
	filter orquestaoutboxdispatch.PendingOutboxFilterV0,
) ([]orquestaoutboxdispatch.OutboxPendingEntryV0, []orquestaoutboxdispatch.DispatchIssueV0) {
	if ledger == nil {
		return nil, []orquestaoutboxdispatch.DispatchIssueV0{{
			Code: ErrPersistenciaNoDisponibleV0, Field: "ledger", Message: "ledger no disponible",
		}}
	}
	filter, issues := normalizeFileOutboxDispatchFilterV0(filter)
	if len(issues) > 0 {
		return nil, issues
	}
	ledger.mu.Lock()
	defer ledger.mu.Unlock()
	entries := make([]orquestaoutboxdispatch.OutboxPendingEntryV0, 0, len(ledger.state.order))
	for _, messageID := range ledger.state.order {
		record := ledger.state.recordsByMessageID[messageID]
		if fileOutboxRecordPendingForDispatchV0(record, filter) {
			entries = append(entries, fileOutboxEntryFromRecordV0(record))
		}
	}
	return entries, nil
}

func (ledger *FileOutboxLedgerV0) ClaimOutboxDispatchV0(
	claim orquestaoutboxdispatch.OutboxDispatchClaimV0,
) (orquestaoutboxdispatch.OutboxDispatchClaimResultV0, []orquestaoutboxdispatch.DispatchIssueV0) {
	if ledger == nil {
		return orquestaoutboxdispatch.OutboxDispatchClaimResultV0{}, []orquestaoutboxdispatch.DispatchIssueV0{{
			Code: ErrPersistenciaNoDisponibleV0, Field: "ledger", Message: "ledger no disponible",
		}}
	}
	claim = normalizeFileDispatchClaimV0(claim)
	if issues := validateFileDispatchClaimV0(claim); len(issues) > 0 {
		return orquestaoutboxdispatch.OutboxDispatchClaimResultV0{}, issues
	}
	ledger.mu.Lock()
	defer ledger.mu.Unlock()
	next := ledger.state.cloneV0()
	record := next.recordsByMessageID[claim.MessageID]
	if record == nil {
		return fileUnclaimedResultV0(claim), []orquestaoutboxdispatch.DispatchIssueV0{{
			Code: ErrPayloadInvalidoV0, Field: "message_id", Message: "mensaje pendiente no encontrado",
		}}
	}
	if issues := fileClaimConflictsV0(record, claim); len(issues) > 0 {
		return fileUnclaimedResultV0(claim), issues
	}
	if record.Ack != nil || (record.Claim != nil && !record.Claim.Recovered) {
		return orquestaoutboxdispatch.OutboxDispatchClaimResultV0{
			AlreadyClaimed: true, MessageID: claim.MessageID, TargetPort: claim.TargetPort,
		}, nil
	}
	storedClaim := fileOutboxClaimFromDispatchV0(claim)
	record.Claim = &storedClaim
	if err := persistFileOutboxLedgerStateV0(ledger.path, next); err != nil {
		return fileUnclaimedResultV0(claim), []orquestaoutboxdispatch.DispatchIssueV0{{
			Code: ErrPersistenciaNoDisponibleV0, Field: "ledger", Message: "persistencia no disponible",
		}}
	}
	ledger.state = next
	return orquestaoutboxdispatch.OutboxDispatchClaimResultV0{
		Claimed: true, MessageID: claim.MessageID, TargetPort: claim.TargetPort,
	}, nil
}

func (ledger *FileOutboxLedgerV0) ReleaseOutboxDispatchClaimV0(
	claim orquestaoutboxdispatch.OutboxDispatchClaimV0,
) []orquestaoutboxdispatch.DispatchIssueV0 {
	claim = normalizeFileDispatchClaimV0(claim)
	if ledger == nil {
		return []orquestaoutboxdispatch.DispatchIssueV0{{
			Code: ErrPersistenciaNoDisponibleV0, Field: "ledger", Message: "ledger no disponible",
		}}
	}
	if issues := validateFileDispatchClaimV0(claim); len(issues) > 0 {
		return issues
	}
	ledger.mu.Lock()
	defer ledger.mu.Unlock()
	next := ledger.state.cloneV0()
	record := next.recordsByMessageID[claim.MessageID]
	if record == nil {
		return []orquestaoutboxdispatch.DispatchIssueV0{{
			Code: ErrPayloadInvalidoV0, Field: "message_id", Message: "mensaje pendiente no encontrado",
		}}
	}
	if issues := fileClaimConflictsV0(record, claim); len(issues) > 0 {
		return issues
	}
	if record.Ack != nil || record.Claim == nil {
		return nil
	}
	record.Claim = nil
	if err := persistFileOutboxLedgerStateV0(ledger.path, next); err != nil {
		return []orquestaoutboxdispatch.DispatchIssueV0{{
			Code: ErrPersistenciaNoDisponibleV0, Field: "ledger", Message: "persistencia no disponible",
		}}
	}
	ledger.state = next
	return nil
}

func (ledger *FileOutboxLedgerV0) AckOutboxDispatchV0(
	ack orquestaoutboxdispatch.OutboxDispatchAckV0,
) []orquestaoutboxdispatch.DispatchIssueV0 {
	return ledger.ackFileOutboxV0(fileOutboxAckFromSuccessV0(ack))
}

func (ledger *FileOutboxLedgerV0) AckOutboxDispatchObservationV0(
	ack orquestaoutboxdispatch.OutboxDispatchAckObservationV0,
) []orquestaoutboxdispatch.DispatchIssueV0 {
	return ledger.ackFileOutboxV0(fileOutboxAckFromObservationV0(ack))
}

func (ledger *FileOutboxLedgerV0) ackFileOutboxV0(
	ack fileOutboxLedgerAckV0,
) []orquestaoutboxdispatch.DispatchIssueV0 {
	if ledger == nil {
		return []orquestaoutboxdispatch.DispatchIssueV0{{
			Code: ErrPersistenciaNoDisponibleV0, Field: "ledger", Message: "ledger no disponible",
		}}
	}
	if issues := validateFileOutboxAckV0(ack); len(issues) > 0 {
		return issues
	}
	ledger.mu.Lock()
	defer ledger.mu.Unlock()
	next := ledger.state.cloneV0()
	record := next.recordsByMessageID[ack.MessageID]
	if record == nil {
		return []orquestaoutboxdispatch.DispatchIssueV0{{
			Code: ErrPayloadInvalidoV0, Field: "message_id", Message: "mensaje pendiente no encontrado",
		}}
	}
	if issues := fileAckConflictsV0(record, ack); len(issues) > 0 {
		return issues
	}
	if record.Ack != nil {
		if fileOutboxAckEqualV0(*record.Ack, ack) {
			return nil
		}
		return []orquestaoutboxdispatch.DispatchIssueV0{{
			Code: ErrConflictoIdempotenciaV0, Field: "ack", Message: "ack incompatible",
		}}
	}
	record.Ack = &ack
	record.Claim = nil
	if err := persistFileOutboxLedgerStateV0(ledger.path, next); err != nil {
		return []orquestaoutboxdispatch.DispatchIssueV0{{
			Code: ErrPersistenciaNoDisponibleV0, Field: "ledger", Message: "persistencia no disponible",
		}}
	}
	ledger.state = next
	return nil
}
