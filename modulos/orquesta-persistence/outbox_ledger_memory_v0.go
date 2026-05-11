package orquestapersistence

import (
	"bytes"
	"context"
	"sync"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
)

// InMemoryOutboxLedgerV0 is a RAM-only ledger for contract tests.
// It does not dispatch messages and is not a production persistence adapter.
type InMemoryOutboxLedgerV0 struct {
	mu                 sync.Mutex
	order              []string
	recordsByMessageID map[string]*outboxLedgerRecordV0
	recordsByIdemKey   map[string]*outboxLedgerRecordV0
}

func NewInMemoryOutboxLedgerV0() *InMemoryOutboxLedgerV0 {
	return &InMemoryOutboxLedgerV0{
		recordsByMessageID: make(map[string]*outboxLedgerRecordV0),
		recordsByIdemKey:   make(map[string]*outboxLedgerRecordV0),
	}
}

func (ledger *InMemoryOutboxLedgerV0) GuardarPendientes(
	ctx context.Context,
	messages []orquestacoreworkflow.OutboxMessageV0,
) ([]orquestacoreworkflow.OutboxMessageV0, []OutboxLedgerIssueV0) {
	return ledger.SavePending(ctx, messages)
}

func (ledger *InMemoryOutboxLedgerV0) SavePending(
	ctx context.Context,
	messages []orquestacoreworkflow.OutboxMessageV0,
) ([]orquestacoreworkflow.OutboxMessageV0, []OutboxLedgerIssueV0) {
	if err := ctx.Err(); err != nil {
		return nil, []OutboxLedgerIssueV0{outboxLedgerIssueV0(ErrPersistenciaNoDisponibleV0, "contexto", err.Error())}
	}

	type candidateV0 struct {
		message     orquestacoreworkflow.OutboxMessageV0
		fingerprint []byte
	}
	candidates := make([]candidateV0, 0, len(messages))
	candidateByMessageID := map[string]candidateV0{}
	candidateByIdemKey := map[string]candidateV0{}
	for index, message := range messages {
		normalized, fingerprint, err := normalizeOutboxLedgerMessageV0(message)
		if err != nil {
			return nil, []OutboxLedgerIssueV0{outboxLedgerIssueFromMessageErrorV0(index, err)}
		}
		candidate := candidateV0{message: normalized, fingerprint: fingerprint}
		if issue, conflict := candidateConflictsV0(candidateByMessageID[normalized.MessageID], candidate, "message_id"); conflict {
			return nil, []OutboxLedgerIssueV0{issue}
		}
		if issue, conflict := candidateConflictsV0(candidateByIdemKey[normalized.IdempotencyKey], candidate, "idempotency_key"); conflict {
			return nil, []OutboxLedgerIssueV0{issue}
		}
		candidateByMessageID[normalized.MessageID] = candidate
		candidateByIdemKey[normalized.IdempotencyKey] = candidate
		candidates = append(candidates, candidate)
	}

	ledger.mu.Lock()
	defer ledger.mu.Unlock()

	accepted := make([]orquestacoreworkflow.OutboxMessageV0, 0, len(candidates))
	for _, candidate := range candidates {
		record, issues := ledger.findCompatibleRecordV0(candidate)
		if len(issues) > 0 {
			return nil, issues
		}
		if record != nil {
			accepted = append(accepted, cloneOutboxMessageV0(record.message))
			continue
		}
		record = &outboxLedgerRecordV0{
			message:            cloneOutboxMessageV0(candidate.message),
			messageFingerprint: append([]byte(nil), candidate.fingerprint...),
		}
		ledger.recordsByMessageID[candidate.message.MessageID] = record
		ledger.recordsByIdemKey[candidate.message.IdempotencyKey] = record
		ledger.order = append(ledger.order, candidate.message.MessageID)
		accepted = append(accepted, cloneOutboxMessageV0(record.message))
	}
	return accepted, nil
}

func (ledger *InMemoryOutboxLedgerV0) ListarPendientes(
	ctx context.Context,
	filter OutboxPendingFilterV0,
) ([]orquestacoreworkflow.OutboxMessageV0, []OutboxLedgerIssueV0) {
	return ledger.ListPending(ctx, filter)
}

func (ledger *InMemoryOutboxLedgerV0) ListPending(
	ctx context.Context,
	filter OutboxPendingFilterV0,
) ([]orquestacoreworkflow.OutboxMessageV0, []OutboxLedgerIssueV0) {
	if err := ctx.Err(); err != nil {
		return nil, []OutboxLedgerIssueV0{outboxLedgerIssueV0(ErrPersistenciaNoDisponibleV0, "contexto", err.Error())}
	}
	filter.RunID = trimV0(filter.RunID)
	filter.TargetPort = trimV0(filter.TargetPort)
	if filter.TargetPort != "" && !outboxLedgerTargetPortSupportedV0(filter.TargetPort) {
		return nil, []OutboxLedgerIssueV0{outboxLedgerIssueV0(ErrPayloadInvalidoV0, "target_port", "target_port no soportado")}
	}

	ledger.mu.Lock()
	defer ledger.mu.Unlock()

	pending := []orquestacoreworkflow.OutboxMessageV0{}
	for _, messageID := range ledger.order {
		record := ledger.recordsByMessageID[messageID]
		if record == nil || record.ack != nil || !recordMatchesFilterV0(record, filter) {
			continue
		}
		pending = append(pending, cloneOutboxMessageV0(record.message))
	}
	return pending, nil
}

func (ledger *InMemoryOutboxLedgerV0) RegistrarAck(
	ctx context.Context,
	ack OutboxDispatchAckV0,
) (OutboxDispatchSnapshotV0, []OutboxLedgerIssueV0) {
	return ledger.MarkDispatched(ctx, ack)
}

func (ledger *InMemoryOutboxLedgerV0) MarkDispatched(
	ctx context.Context,
	ack OutboxDispatchAckV0,
) (OutboxDispatchSnapshotV0, []OutboxLedgerIssueV0) {
	if err := ctx.Err(); err != nil {
		return OutboxDispatchSnapshotV0{}, []OutboxLedgerIssueV0{outboxLedgerIssueV0(ErrPersistenciaNoDisponibleV0, "contexto", err.Error())}
	}
	ack = normalizeOutboxDispatchAckV0(ack)
	if issues := validateOutboxDispatchAckV0(ack); len(issues) > 0 {
		return OutboxDispatchSnapshotV0{}, issues
	}
	fingerprint, err := outboxLedgerAckFingerprintV0(ack)
	if err != nil {
		return OutboxDispatchSnapshotV0{}, []OutboxLedgerIssueV0{outboxLedgerIssueV0(ErrPayloadInvalidoV0, "ack", err.Error())}
	}

	ledger.mu.Lock()
	defer ledger.mu.Unlock()

	record := ledger.recordsByMessageID[ack.MessageID]
	if record == nil {
		return OutboxDispatchSnapshotV0{}, []OutboxLedgerIssueV0{outboxLedgerIssueV0(ErrPayloadInvalidoV0, "message_id", "mensaje pendiente no encontrado")}
	}
	if ack.RunID != record.message.RunID || ack.TargetPort != record.message.TargetPort {
		return OutboxDispatchSnapshotV0{}, []OutboxLedgerIssueV0{outboxLedgerIssueV0(ErrConflictoIdempotenciaV0, "ack", "ack no corresponde al mensaje")}
	}
	if record.ack != nil {
		if !bytes.Equal(record.ackFingerprint, fingerprint) {
			return OutboxDispatchSnapshotV0{}, []OutboxLedgerIssueV0{outboxLedgerIssueV0(ErrConflictoIdempotenciaV0, "ack", "ack incompatible")}
		}
		return outboxLedgerSnapshotFromAckV0(cloneOutboxAckV0(*record.ack)), nil
	}
	record.ack = &ack
	record.ackFingerprint = append([]byte(nil), fingerprint...)
	return outboxLedgerSnapshotFromAckV0(cloneOutboxAckV0(ack)), nil
}

func (ledger *InMemoryOutboxLedgerV0) findCompatibleRecordV0(candidate struct {
	message     orquestacoreworkflow.OutboxMessageV0
	fingerprint []byte
}) (*outboxLedgerRecordV0, []OutboxLedgerIssueV0) {
	byMessageID := ledger.recordsByMessageID[candidate.message.MessageID]
	byIdemKey := ledger.recordsByIdemKey[candidate.message.IdempotencyKey]
	if byMessageID != nil && byIdemKey != nil && byMessageID != byIdemKey {
		return nil, []OutboxLedgerIssueV0{outboxLedgerIssueV0(ErrConflictoIdempotenciaV0, "idempotency_key", "idempotency_key apunta a otro mensaje")}
	}
	record := byMessageID
	if record == nil {
		record = byIdemKey
	}
	if record == nil {
		return nil, nil
	}
	if !bytes.Equal(record.messageFingerprint, candidate.fingerprint) {
		return nil, []OutboxLedgerIssueV0{outboxLedgerIssueV0(ErrConflictoIdempotenciaV0, "message_id", "mensaje incompatible")}
	}
	return record, nil
}

func candidateConflictsV0(existing struct {
	message     orquestacoreworkflow.OutboxMessageV0
	fingerprint []byte
}, candidate struct {
	message     orquestacoreworkflow.OutboxMessageV0
	fingerprint []byte
}, field string) (OutboxLedgerIssueV0, bool) {
	if existing.message.MessageID == "" {
		return OutboxLedgerIssueV0{}, false
	}
	if bytes.Equal(existing.fingerprint, candidate.fingerprint) {
		return OutboxLedgerIssueV0{}, false
	}
	return outboxLedgerIssueV0(ErrConflictoIdempotenciaV0, field, "mensaje incompatible en lote"), true
}

func recordMatchesFilterV0(record *outboxLedgerRecordV0, filter OutboxPendingFilterV0) bool {
	if filter.RunID != "" && record.message.RunID != filter.RunID {
		return false
	}
	return filter.TargetPort == "" || record.message.TargetPort == filter.TargetPort
}
