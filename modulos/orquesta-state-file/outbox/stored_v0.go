package orquestastatefileoutbox

func normalizeStoredClaimV0(claim outboxLedgerClaimV0) outboxLedgerClaimV0 {
	return outboxLedgerClaimV0{
		MessageID:      trimV0(claim.MessageID),
		RunID:          trimV0(claim.RunID),
		TargetPort:     trimV0(claim.TargetPort),
		IdempotencyKey: trimV0(claim.IdempotencyKey),
	}
}

func normalizeStoredAckV0(ack outboxLedgerAckV0) outboxLedgerAckV0 {
	return outboxLedgerAckV0{
		MessageID:    trimV0(ack.MessageID),
		RunID:        trimV0(ack.RunID),
		TargetPort:   trimV0(ack.TargetPort),
		Status:       trimV0(ack.Status),
		DispatchRef:  trimV0(ack.DispatchRef),
		EvidenceRefs: compactStringsV0(ack.EvidenceRefs),
		Issues:       normalizeStoredAckIssuesV0(ack.Issues),
	}
}

func normalizeStoredAckIssuesV0(issues []outboxLedgerAckIssueV0) []outboxLedgerAckIssueV0 {
	out := make([]outboxLedgerAckIssueV0, 0, len(issues))
	for _, issue := range issues {
		normalized := outboxLedgerAckIssueV0{
			Code:    trimV0(issue.Code),
			Field:   trimV0(issue.Field),
			Message: trimV0(issue.Message),
		}
		if normalized.Code == "" && normalized.Field == "" && normalized.Message == "" {
			continue
		}
		out = append(out, normalized)
	}
	return out
}

func claimMatchesRecordV0(claim outboxLedgerClaimV0, record *outboxLedgerRecordV0) bool {
	return claim.MessageID == record.Message.MessageID &&
		(claim.RunID == "" || claim.RunID == record.Message.RunID) &&
		(claim.TargetPort == "" || claim.TargetPort == record.Message.TargetPort) &&
		(claim.IdempotencyKey == "" || claim.IdempotencyKey == record.Message.IdempotencyKey)
}

func ackMatchesRecordV0(ack outboxLedgerAckV0, record *outboxLedgerRecordV0) bool {
	return ack.MessageID == record.Message.MessageID &&
		(ack.RunID == "" || ack.RunID == record.Message.RunID) &&
		(ack.TargetPort == "" || ack.TargetPort == record.Message.TargetPort)
}
