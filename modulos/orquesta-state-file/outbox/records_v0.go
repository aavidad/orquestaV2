package orquestastatefileoutbox

import (
	"bytes"
	"fmt"
)

func rebuildOutboxLedgerStateV0(records []outboxLedgerRecordV0) (outboxLedgerStateV0, error) {
	state := newOutboxLedgerStateV0()
	for _, stored := range records {
		record, err := normalizeOutboxRecordV0(stored)
		if err != nil {
			return outboxLedgerStateV0{}, err
		}
		messageID := record.Message.MessageID
		if state.recordsByMessageID[messageID] != nil {
			return outboxLedgerStateV0{}, fmt.Errorf("file_outbox_ledger: duplicate_message")
		}
		if state.messageIDByIdemKey[record.Message.IdempotencyKey] != "" {
			return outboxLedgerStateV0{}, fmt.Errorf("file_outbox_ledger: duplicate_idempotency")
		}
		state.recordsByMessageID[messageID] = record
		state.messageIDByIdemKey[record.Message.IdempotencyKey] = messageID
		state.order = append(state.order, messageID)
	}
	return state, nil
}

func normalizeOutboxRecordV0(stored outboxLedgerRecordV0) (*outboxLedgerRecordV0, error) {
	message, fingerprint, err := normalizeOutboxMessageV0(stored.Message)
	if err != nil {
		return nil, fmt.Errorf("file_outbox_ledger: message_invalid")
	}
	record := &outboxLedgerRecordV0{
		Message:            message,
		messageFingerprint: append([]byte(nil), fingerprint...),
	}
	if stored.Claim != nil && stored.Ack == nil {
		claim := normalizeStoredClaimV0(*stored.Claim)
		if !claimMatchesRecordV0(claim, record) {
			return nil, fmt.Errorf("file_outbox_ledger: claim_invalid")
		}
		// A claim without ack is process-local reservation. After reload it must
		// be retried idempotently instead of blocking the message forever.
	}
	if stored.Ack != nil {
		ack := normalizeStoredAckV0(*stored.Ack)
		if !ackMatchesRecordV0(ack, record) {
			return nil, fmt.Errorf("file_outbox_ledger: ack_invalid")
		}
		record.Ack = &ack
	} else {
		record.Claim = nil
	}
	return record, nil
}

func (state outboxLedgerStateV0) cloneV0() outboxLedgerStateV0 {
	cloned := newOutboxLedgerStateV0()
	cloned.order = append([]string(nil), state.order...)
	for messageID, record := range state.recordsByMessageID {
		cloned.recordsByMessageID[messageID] = cloneOutboxRecordV0(record)
	}
	for idemKey, messageID := range state.messageIDByIdemKey {
		cloned.messageIDByIdemKey[idemKey] = messageID
	}
	return cloned
}

func cloneOutboxRecordV0(record *outboxLedgerRecordV0) *outboxLedgerRecordV0 {
	if record == nil {
		return nil
	}
	cloned := &outboxLedgerRecordV0{
		Message:            cloneOutboxMessageV0(record.Message),
		messageFingerprint: append([]byte(nil), record.messageFingerprint...),
	}
	if record.Claim != nil {
		claim := *record.Claim
		cloned.Claim = &claim
	}
	if record.Ack != nil {
		ack := *record.Ack
		ack.EvidenceRefs = append([]string(nil), record.Ack.EvidenceRefs...)
		ack.Issues = append([]outboxLedgerAckIssueV0(nil), record.Ack.Issues...)
		cloned.Ack = &ack
	}
	return cloned
}

func outboxLedgerRecordsForSnapshotV0(state outboxLedgerStateV0) []outboxLedgerRecordV0 {
	records := make([]outboxLedgerRecordV0, 0, len(state.order))
	for _, messageID := range state.order {
		record := cloneOutboxRecordV0(state.recordsByMessageID[messageID])
		if record == nil {
			continue
		}
		record.messageFingerprint = nil
		records = append(records, *record)
	}
	return records
}

func (state outboxLedgerStateV0) compatibleRecordV0(
	candidate outboxLedgerCandidateV0,
) (*outboxLedgerRecordV0, []ledgerIssueV0) {
	byMessageID := state.recordsByMessageID[candidate.message.MessageID]
	byIdemKey := state.recordsByMessageID[state.messageIDByIdemKey[candidate.message.IdempotencyKey]]
	if byMessageID != nil && byIdemKey != nil && byMessageID != byIdemKey {
		return nil, []ledgerIssueV0{{
			Code: errIdempotencyConflictV0, Field: "idempotency_key",
			Message: "idempotency_key apunta a otro mensaje",
		}}
	}
	record := byMessageID
	if record == nil {
		record = byIdemKey
	}
	if record == nil {
		return nil, nil
	}
	if !bytes.Equal(record.messageFingerprint, candidate.fingerprint) {
		return nil, []ledgerIssueV0{{
			Code: errIdempotencyConflictV0, Field: "message_id", Message: "mensaje incompatible",
		}}
	}
	return record, nil
}
