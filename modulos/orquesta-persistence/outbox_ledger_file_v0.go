package orquestapersistence

import (
	"bytes"
	"context"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectorcycleoutbox "orquesta/modulos/orquesta-director-cycle-outbox"
	orquestaoutboxdispatch "orquesta/modulos/orquesta-outbox-dispatch"
)

func (ledger *FileOutboxLedgerV0) SavePending(
	ctx context.Context,
	messages []orquestacoreworkflow.OutboxMessageV0,
) ([]orquestacoreworkflow.OutboxMessageV0, []orquestadirectorcycleoutbox.DirectorCycleOutboxIssueV0) {
	if ledger == nil {
		return nil, []orquestadirectorcycleoutbox.DirectorCycleOutboxIssueV0{{
			Code: ErrPersistenciaNoDisponibleV0, Field: "ledger", Message: "ledger no disponible",
		}}
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return nil, []orquestadirectorcycleoutbox.DirectorCycleOutboxIssueV0{{
			Code: ErrPersistenciaNoDisponibleV0, Field: "contexto", Message: err.Error(),
		}}
	}
	candidates, issues := fileOutboxCandidatesV0(messages)
	if len(issues) > 0 {
		return nil, directorIssuesFromOutboxLedgerV0(issues)
	}
	ledger.mu.Lock()
	defer ledger.mu.Unlock()
	next := ledger.state.cloneV0()
	accepted, changed, issues := saveFileOutboxCandidatesV0(&next, candidates)
	if len(issues) > 0 {
		return nil, directorIssuesFromOutboxLedgerV0(issues)
	}
	if changed {
		if err := persistFileOutboxLedgerStateV0(ledger.path, next); err != nil {
			return nil, []orquestadirectorcycleoutbox.DirectorCycleOutboxIssueV0{{
				Code: ErrPersistenciaNoDisponibleV0, Field: "ledger", Message: "persistencia no disponible",
			}}
		}
		ledger.state = next
	}
	return accepted, nil
}

func (ledger *FileOutboxLedgerV0) ListPending(
	ctx context.Context,
	filter orquestadirectorcycleoutbox.DirectorCycleOutboxPendingFilterV0,
) ([]orquestacoreworkflow.OutboxMessageV0, []orquestadirectorcycleoutbox.DirectorCycleOutboxIssueV0) {
	if ledger == nil {
		return nil, []orquestadirectorcycleoutbox.DirectorCycleOutboxIssueV0{{
			Code: ErrPersistenciaNoDisponibleV0, Field: "ledger", Message: "ledger no disponible",
		}}
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return nil, []orquestadirectorcycleoutbox.DirectorCycleOutboxIssueV0{{
			Code: ErrPersistenciaNoDisponibleV0, Field: "contexto", Message: err.Error(),
		}}
	}
	filter, issues := normalizeFileOutboxDirectorFilterV0(filter)
	if len(issues) > 0 {
		return nil, issues
	}
	ledger.mu.Lock()
	defer ledger.mu.Unlock()
	pending := make([]orquestacoreworkflow.OutboxMessageV0, 0, len(ledger.state.order))
	for _, messageID := range ledger.state.order {
		record := ledger.state.recordsByMessageID[messageID]
		if fileOutboxRecordPendingForDirectorV0(record, filter) {
			pending = append(pending, cloneOutboxMessageV0(record.Message))
		}
	}
	return pending, nil
}

type fileOutboxCandidateV0 struct {
	message     orquestacoreworkflow.OutboxMessageV0
	fingerprint []byte
}

func fileOutboxCandidatesV0(
	messages []orquestacoreworkflow.OutboxMessageV0,
) ([]fileOutboxCandidateV0, []OutboxLedgerIssueV0) {
	candidates := make([]fileOutboxCandidateV0, 0, len(messages))
	byMessageID := map[string]fileOutboxCandidateV0{}
	byIdemKey := map[string]fileOutboxCandidateV0{}
	for _, message := range messages {
		normalized, fingerprint, err := normalizeOutboxLedgerMessageV0(message)
		if err != nil {
			return nil, []OutboxLedgerIssueV0{outboxLedgerIssueV0(ErrPayloadInvalidoV0, "message", err.Error())}
		}
		candidate := fileOutboxCandidateV0{message: normalized, fingerprint: fingerprint}
		if issue, conflict := fileOutboxCandidateConflictV0(byMessageID[normalized.MessageID], candidate, "message_id"); conflict {
			return nil, []OutboxLedgerIssueV0{issue}
		}
		if issue, conflict := fileOutboxCandidateConflictV0(byIdemKey[normalized.IdempotencyKey], candidate, "idempotency_key"); conflict {
			return nil, []OutboxLedgerIssueV0{issue}
		}
		byMessageID[normalized.MessageID] = candidate
		byIdemKey[normalized.IdempotencyKey] = candidate
		candidates = append(candidates, candidate)
	}
	return candidates, nil
}

func saveFileOutboxCandidatesV0(
	state *fileOutboxLedgerStateV0,
	candidates []fileOutboxCandidateV0,
) ([]orquestacoreworkflow.OutboxMessageV0, bool, []OutboxLedgerIssueV0) {
	accepted := make([]orquestacoreworkflow.OutboxMessageV0, 0, len(candidates))
	changed := false
	for _, candidate := range candidates {
		record, issues := state.compatibleRecordV0(candidate.message, candidate.fingerprint)
		if len(issues) > 0 {
			return nil, false, issues
		}
		if record != nil {
			accepted = append(accepted, cloneOutboxMessageV0(record.Message))
			continue
		}
		state.recordsByMessageID[candidate.message.MessageID] = &fileOutboxLedgerRecordV0{
			Message:            cloneOutboxMessageV0(candidate.message),
			messageFingerprint: append([]byte(nil), candidate.fingerprint...),
		}
		state.messageIDByIdemKey[candidate.message.IdempotencyKey] = candidate.message.MessageID
		state.order = append(state.order, candidate.message.MessageID)
		accepted = append(accepted, cloneOutboxMessageV0(candidate.message))
		changed = true
	}
	return accepted, changed, nil
}

func fileOutboxCandidateConflictV0(
	existing fileOutboxCandidateV0,
	candidate fileOutboxCandidateV0,
	field string,
) (OutboxLedgerIssueV0, bool) {
	if existing.message.MessageID == "" || bytes.Equal(existing.fingerprint, candidate.fingerprint) {
		return OutboxLedgerIssueV0{}, false
	}
	return outboxLedgerIssueV0(ErrConflictoIdempotenciaV0, field, "mensaje incompatible en lote"), true
}

var _ orquestaoutboxdispatch.PendingOutboxReaderPortV0 = (*FileOutboxLedgerV0)(nil)
var _ orquestaoutboxdispatch.OutboxDispatchClaimerPortV0 = (*FileOutboxLedgerV0)(nil)
var _ orquestaoutboxdispatch.OutboxDispatchAckPortV0 = (*FileOutboxLedgerV0)(nil)
var _ orquestaoutboxdispatch.OutboxDispatchAckObservationPortV0 = (*FileOutboxLedgerV0)(nil)
