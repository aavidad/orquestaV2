package orquestapersistence

import orquestaoutboxdispatch "orquesta/modulos/orquesta-outbox-dispatch"

func normalizeFileDispatchClaimV0(
	claim orquestaoutboxdispatch.OutboxDispatchClaimV0,
) orquestaoutboxdispatch.OutboxDispatchClaimV0 {
	return orquestaoutboxdispatch.OutboxDispatchClaimV0{
		MessageID:      trimV0(claim.MessageID),
		RunID:          trimV0(claim.RunID),
		TargetPort:     trimV0(claim.TargetPort),
		IdempotencyKey: trimV0(claim.IdempotencyKey),
	}
}

func validateFileDispatchClaimV0(
	claim orquestaoutboxdispatch.OutboxDispatchClaimV0,
) []orquestaoutboxdispatch.DispatchIssueV0 {
	if claim.MessageID == "" {
		return []orquestaoutboxdispatch.DispatchIssueV0{{
			Code: ErrPayloadInvalidoV0, Field: "message_id", Message: "message_id requerido",
		}}
	}
	if claim.TargetPort != "" && !outboxLedgerTargetPortSupportedV0(claim.TargetPort) {
		return []orquestaoutboxdispatch.DispatchIssueV0{{
			Code: ErrPayloadInvalidoV0, Field: "target_port", Message: "target_port no soportado",
		}}
	}
	return nil
}

func validateFileOutboxAckV0(
	ack fileOutboxLedgerAckV0,
) []orquestaoutboxdispatch.DispatchIssueV0 {
	var issues []orquestaoutboxdispatch.DispatchIssueV0
	if ack.MessageID == "" {
		issues = append(issues, orquestaoutboxdispatch.DispatchIssueV0{
			Code: ErrPayloadInvalidoV0, Field: "message_id", Message: "message_id requerido",
		})
	}
	if ack.Status != OutboxDispatchStatusDispatchedV0 && ack.Status != OutboxDispatchStatusFailedV0 {
		issues = append(issues, orquestaoutboxdispatch.DispatchIssueV0{
			Code: ErrPayloadInvalidoV0, Field: "status", Message: "status no soportado",
		})
	}
	if ack.TargetPort != "" && !outboxLedgerTargetPortSupportedV0(ack.TargetPort) {
		issues = append(issues, orquestaoutboxdispatch.DispatchIssueV0{
			Code: ErrPayloadInvalidoV0, Field: "target_port", Message: "target_port no soportado",
		})
	}
	return issues
}

func fileClaimConflictsV0(
	record *fileOutboxLedgerRecordV0,
	claim orquestaoutboxdispatch.OutboxDispatchClaimV0,
) []orquestaoutboxdispatch.DispatchIssueV0 {
	if claim.RunID != "" && claim.RunID != record.Message.RunID {
		return []orquestaoutboxdispatch.DispatchIssueV0{{
			Code: ErrConflictoIdempotenciaV0, Field: "run_id", Message: "claim no corresponde al mensaje",
		}}
	}
	if claim.TargetPort != "" && claim.TargetPort != record.Message.TargetPort {
		return []orquestaoutboxdispatch.DispatchIssueV0{{
			Code: ErrConflictoIdempotenciaV0, Field: "target_port", Message: "claim no corresponde al mensaje",
		}}
	}
	if claim.IdempotencyKey != "" && claim.IdempotencyKey != record.Message.IdempotencyKey {
		return []orquestaoutboxdispatch.DispatchIssueV0{{
			Code: ErrConflictoIdempotenciaV0, Field: "idempotency_key", Message: "claim no corresponde al mensaje",
		}}
	}
	return nil
}

func fileAckConflictsV0(
	record *fileOutboxLedgerRecordV0,
	ack fileOutboxLedgerAckV0,
) []orquestaoutboxdispatch.DispatchIssueV0 {
	if ack.RunID != "" && ack.RunID != record.Message.RunID {
		return []orquestaoutboxdispatch.DispatchIssueV0{{
			Code: ErrConflictoIdempotenciaV0, Field: "run_id", Message: "ack no corresponde al mensaje",
		}}
	}
	if ack.TargetPort != "" && ack.TargetPort != record.Message.TargetPort {
		return []orquestaoutboxdispatch.DispatchIssueV0{{
			Code: ErrConflictoIdempotenciaV0, Field: "target_port", Message: "ack no corresponde al mensaje",
		}}
	}
	return nil
}

func fileUnclaimedResultV0(
	claim orquestaoutboxdispatch.OutboxDispatchClaimV0,
) orquestaoutboxdispatch.OutboxDispatchClaimResultV0 {
	return orquestaoutboxdispatch.OutboxDispatchClaimResultV0{
		MessageID: claim.MessageID, TargetPort: claim.TargetPort,
	}
}
