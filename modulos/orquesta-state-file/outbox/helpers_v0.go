package orquestastatefileoutbox

import (
	"bytes"
	"context"
	"path/filepath"
	"reflect"
	"strings"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectorcycleoutbox "orquesta/modulos/orquesta-director-cycle-outbox"
	orquestaoutboxdispatch "orquesta/modulos/orquesta-outbox-dispatch"
)

func contextIssueV0(ctx context.Context) []orquestadirectorcycleoutbox.DirectorCycleOutboxIssueV0 {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return []orquestadirectorcycleoutbox.DirectorCycleOutboxIssueV0{
			directorIssueV0(errPersistenceUnavailableV0, "contexto", err.Error()),
		}
	}
	return nil
}

func buildCandidatesV0(
	messages []orquestacoreworkflow.OutboxMessageV0,
) ([]outboxLedgerCandidateV0, []orquestadirectorcycleoutbox.DirectorCycleOutboxIssueV0) {
	candidates := make([]outboxLedgerCandidateV0, 0, len(messages))
	byMessageID := map[string]outboxLedgerCandidateV0{}
	byIdemKey := map[string]outboxLedgerCandidateV0{}
	for index, message := range messages {
		normalized, fingerprint, err := normalizeOutboxMessageV0(message)
		if err != nil {
			return nil, directorIssuesFromLedgerV0([]ledgerIssueV0{outboxMessageIssueFromErrorV0(index, err)})
		}
		candidate := outboxLedgerCandidateV0{message: normalized, fingerprint: fingerprint}
		if issue, conflict := candidateConflictV0(byMessageID[normalized.MessageID], candidate, "message_id"); conflict {
			return nil, []orquestadirectorcycleoutbox.DirectorCycleOutboxIssueV0{issue}
		}
		if issue, conflict := candidateConflictV0(byIdemKey[normalized.IdempotencyKey], candidate, "idempotency_key"); conflict {
			return nil, []orquestadirectorcycleoutbox.DirectorCycleOutboxIssueV0{issue}
		}
		byMessageID[normalized.MessageID] = candidate
		byIdemKey[normalized.IdempotencyKey] = candidate
		candidates = append(candidates, candidate)
	}
	return candidates, nil
}

func candidateConflictV0(
	existing outboxLedgerCandidateV0,
	candidate outboxLedgerCandidateV0,
	field string,
) (orquestadirectorcycleoutbox.DirectorCycleOutboxIssueV0, bool) {
	if existing.message.MessageID == "" || bytes.Equal(existing.fingerprint, candidate.fingerprint) {
		return orquestadirectorcycleoutbox.DirectorCycleOutboxIssueV0{}, false
	}
	return directorIssueV0(errIdempotencyConflictV0, field, "mensaje incompatible en lote"), true
}

func ledgerPathV0(dir string) (string, error) {
	dir = strings.TrimSpace(dir)
	if dir == "" {
		return "", filepath.ErrBadPattern
	}
	return filepath.Join(filepath.Clean(dir), fileOutboxLedgerNameV0), nil
}

func normalizeDispatchFilterV0(
	filter orquestaoutboxdispatch.PendingOutboxFilterV0,
) (orquestaoutboxdispatch.PendingOutboxFilterV0, []orquestaoutboxdispatch.DispatchIssueV0) {
	filter.RunID = trimV0(filter.RunID)
	filter.TargetPort = trimV0(filter.TargetPort)
	filter.MessageType = trimV0(filter.MessageType)
	if filter.TargetPort != "" && !targetPortSupportedV0(filter.TargetPort) {
		return filter, []orquestaoutboxdispatch.DispatchIssueV0{
			dispatchIssueV0(errPayloadInvalidV0, "target_port", "target_port no soportado"),
		}
	}
	return filter, nil
}

func normalizeClaimV0(
	claim orquestaoutboxdispatch.OutboxDispatchClaimV0,
) orquestaoutboxdispatch.OutboxDispatchClaimV0 {
	return orquestaoutboxdispatch.OutboxDispatchClaimV0{
		MessageID:      trimV0(claim.MessageID),
		RunID:          trimV0(claim.RunID),
		TargetPort:     trimV0(claim.TargetPort),
		IdempotencyKey: trimV0(claim.IdempotencyKey),
	}
}

func normalizeDispatchAckV0(
	ack orquestaoutboxdispatch.OutboxDispatchAckV0,
) orquestaoutboxdispatch.OutboxDispatchAckV0 {
	return orquestaoutboxdispatch.OutboxDispatchAckV0{
		MessageID:    trimV0(ack.MessageID),
		RunID:        trimV0(ack.RunID),
		TargetPort:   trimV0(ack.TargetPort),
		DispatchRef:  trimV0(ack.DispatchRef),
		EvidenceRefs: compactStringsV0(ack.EvidenceRefs),
	}
}

func storedAckFromDispatchV0(ack orquestaoutboxdispatch.OutboxDispatchAckV0) outboxLedgerAckV0 {
	return outboxLedgerAckV0{
		MessageID:    ack.MessageID,
		RunID:        ack.RunID,
		TargetPort:   ack.TargetPort,
		DispatchRef:  ack.DispatchRef,
		EvidenceRefs: append([]string(nil), ack.EvidenceRefs...),
	}
}

func dispatchAckFromStoredV0(ack outboxLedgerAckV0) orquestaoutboxdispatch.OutboxDispatchAckV0 {
	return orquestaoutboxdispatch.OutboxDispatchAckV0{
		MessageID:    trimV0(ack.MessageID),
		RunID:        trimV0(ack.RunID),
		TargetPort:   trimV0(ack.TargetPort),
		DispatchRef:  trimV0(ack.DispatchRef),
		EvidenceRefs: compactStringsV0(ack.EvidenceRefs),
	}
}

func storedAckEqualV0(a, b outboxLedgerAckV0) bool {
	return reflect.DeepEqual(dispatchAckFromStoredV0(a), dispatchAckFromStoredV0(b))
}
