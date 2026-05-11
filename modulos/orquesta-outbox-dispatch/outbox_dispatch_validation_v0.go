package orquestaoutboxdispatch

import "strings"

func normalizeSelectionV0(selection DispatchSelectionV0) DispatchSelectionV0 {
	return DispatchSelectionV0{
		RunID:             strings.TrimSpace(selection.RunID),
		TargetPort:        strings.TrimSpace(selection.TargetPort),
		MessageType:       strings.TrimSpace(selection.MessageType),
		Pending:           cloneEntriesV0(selection.Pending),
		ClaimedMessageIDs: compactStringsV0(selection.ClaimedMessageIDs),
	}
}

func normalizeEntryV0(entry OutboxPendingEntryV0) OutboxPendingEntryV0 {
	return OutboxPendingEntryV0{
		MessageID:      strings.TrimSpace(entry.MessageID),
		RunID:          strings.TrimSpace(entry.RunID),
		TargetPort:     strings.TrimSpace(entry.TargetPort),
		MessageType:    strings.TrimSpace(entry.MessageType),
		IdempotencyKey: strings.TrimSpace(entry.IdempotencyKey),
		CorrelationID:  strings.TrimSpace(entry.CorrelationID),
		PayloadVersion: strings.TrimSpace(entry.PayloadVersion),
		Payload:        cloneBytesV0(entry.Payload),
	}
}

func validateEntryV0(entry OutboxPendingEntryV0) DispatchIssueV0 {
	if entry.MessageID == "" {
		return issueV0(issueInvalidEntryV0, "message_id", "message_id requerido")
	}
	if entry.RunID == "" {
		return issueV0(issueInvalidEntryV0, "run_id", "run_id requerido")
	}
	if entry.TargetPort == "" {
		return issueV0(issueInvalidEntryV0, "target_port", "target_port requerido")
	}
	if entry.MessageType == "" {
		return issueV0(issueInvalidEntryV0, "message_type", "message_type requerido")
	}
	if entry.PayloadVersion == "" {
		return issueV0(issueInvalidEntryV0, "payload_version", "payload_version requerido")
	}
	return DispatchIssueV0{}
}

func issueV0(code string, field string, message string) DispatchIssueV0 {
	return DispatchIssueV0{Code: code, Field: field, Message: message}
}

func claimedSetV0(ids []string) map[string]bool {
	claimed := make(map[string]bool, len(ids))
	for _, id := range ids {
		trimmed := strings.TrimSpace(id)
		if trimmed != "" {
			claimed[trimmed] = true
		}
	}
	return claimed
}
