package orquestapersistence

import (
	"bytes"
	"fmt"
	"reflect"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
)

func rebuildFileOutboxLedgerStateV0(
	records []fileOutboxLedgerRecordV0,
) (fileOutboxLedgerStateV0, error) {
	state := newFileOutboxLedgerStateV0()
	for _, stored := range records {
		record, err := normalizeFileOutboxRecordV0(stored)
		if err != nil {
			return fileOutboxLedgerStateV0{}, err
		}
		messageID := record.Message.MessageID
		if state.recordsByMessageID[messageID] != nil {
			return fileOutboxLedgerStateV0{}, fmt.Errorf("file_outbox_ledger: duplicate_message")
		}
		if state.messageIDByIdemKey[record.Message.IdempotencyKey] != "" {
			return fileOutboxLedgerStateV0{}, fmt.Errorf("file_outbox_ledger: duplicate_idempotency")
		}
		state.recordsByMessageID[messageID] = record
		state.messageIDByIdemKey[record.Message.IdempotencyKey] = messageID
		state.order = append(state.order, messageID)
	}
	return state, nil
}

func normalizeFileOutboxRecordV0(stored fileOutboxLedgerRecordV0) (*fileOutboxLedgerRecordV0, error) {
	message, fingerprint, err := normalizeOutboxLedgerMessageV0(stored.Message)
	if err != nil {
		return nil, fmt.Errorf("file_outbox_ledger: message_invalid")
	}
	record := &fileOutboxLedgerRecordV0{
		Message:            message,
		messageFingerprint: append([]byte(nil), fingerprint...),
	}
	if stored.Claim != nil && stored.Ack == nil {
		claim := normalizeStoredFileOutboxClaimV0(*stored.Claim)
		if !fileOutboxClaimMatchesRecordV0(claim, record) {
			return nil, fmt.Errorf("file_outbox_ledger: claim_invalid")
		}
		claim.Recovered = true
		record.Claim = &claim
	}
	if stored.Ack != nil {
		ack := normalizeStoredFileOutboxAckV0(*stored.Ack)
		if !fileOutboxAckMatchesRecordV0(ack, record) {
			return nil, fmt.Errorf("file_outbox_ledger: ack_invalid")
		}
		record.Ack = &ack
	}
	return record, nil
}

func (state fileOutboxLedgerStateV0) compatibleRecordV0(
	message orquestacoreworkflow.OutboxMessageV0,
	fingerprint []byte,
) (*fileOutboxLedgerRecordV0, []OutboxLedgerIssueV0) {
	byMessageID := state.recordsByMessageID[message.MessageID]
	byIdemKey := state.recordsByMessageID[state.messageIDByIdemKey[message.IdempotencyKey]]
	if byMessageID != nil && byIdemKey != nil && byMessageID != byIdemKey {
		return nil, []OutboxLedgerIssueV0{outboxLedgerIssueV0(
			ErrConflictoIdempotenciaV0, "idempotency_key", "idempotency_key apunta a otro mensaje",
		)}
	}
	record := byMessageID
	if record == nil {
		record = byIdemKey
	}
	if record == nil {
		return nil, nil
	}
	if !bytes.Equal(record.messageFingerprint, fingerprint) {
		return nil, []OutboxLedgerIssueV0{outboxLedgerIssueV0(
			ErrConflictoIdempotenciaV0, "message_id", "mensaje incompatible",
		)}
	}
	return record, nil
}

func (state fileOutboxLedgerStateV0) cloneV0() fileOutboxLedgerStateV0 {
	cloned := newFileOutboxLedgerStateV0()
	cloned.order = append([]string(nil), state.order...)
	for messageID, record := range state.recordsByMessageID {
		cloned.recordsByMessageID[messageID] = cloneFileOutboxRecordV0(record)
	}
	for idemKey, messageID := range state.messageIDByIdemKey {
		cloned.messageIDByIdemKey[idemKey] = messageID
	}
	return cloned
}

func cloneFileOutboxRecordV0(record *fileOutboxLedgerRecordV0) *fileOutboxLedgerRecordV0 {
	if record == nil {
		return nil
	}
	cloned := &fileOutboxLedgerRecordV0{
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
		ack.IssueCodes = append([]string(nil), record.Ack.IssueCodes...)
		ack.Issues = append([]OutboxLedgerIssueV0(nil), record.Ack.Issues...)
		cloned.Ack = &ack
	}
	return cloned
}

func fileOutboxLedgerRecordsForSnapshotV0(
	state fileOutboxLedgerStateV0,
) []fileOutboxLedgerRecordV0 {
	records := make([]fileOutboxLedgerRecordV0, 0, len(state.order))
	for _, messageID := range state.order {
		record := cloneFileOutboxRecordV0(state.recordsByMessageID[messageID])
		if record == nil {
			continue
		}
		record.messageFingerprint = nil
		records = append(records, *record)
	}
	return records
}

func fileOutboxAckEqualV0(a, b fileOutboxLedgerAckV0) bool {
	return reflect.DeepEqual(normalizeStoredFileOutboxAckV0(a), normalizeStoredFileOutboxAckV0(b))
}
