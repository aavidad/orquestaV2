package orquestaoutboxdispatch

const (
	issueInvalidRequestV0 = "invalid_request"
	issueInvalidEntryV0   = "invalid_entry"
	issueDuplicateV0      = "duplicate_pending"
)

func ChooseNextDispatchV0(selection DispatchSelectionV0) DispatchDecisionV0 {
	normalized := normalizeSelectionV0(selection)
	if normalized.TargetPort == "" {
		return DispatchDecisionV0{
			Kind:   DispatchDecisionInvalidRequestV0,
			Reason: "target_port requerido",
			Issues: []DispatchIssueV0{issueV0(issueInvalidRequestV0, "target_port", "target_port requerido")},
		}
	}

	claimed := claimedSetV0(normalized.ClaimedMessageIDs)
	issues := make([]DispatchIssueV0, 0)
	for _, entry := range normalized.Pending {
		entry = normalizeEntryV0(entry)
		if !entryMatchesSelectionV0(entry, normalized) {
			continue
		}
		if issue := validateEntryV0(entry); issue.Code != "" {
			issues = append(issues, issue)
			continue
		}
		if claimed[entry.MessageID] {
			issues = append(issues, issueV0(issueDuplicateV0, "message_id", entry.MessageID+" ya reclamado"))
			continue
		}
		return DispatchDecisionV0{
			Kind:   DispatchDecisionReadyV0,
			Reason: "entrada pendiente seleccionada",
			Intent: intentFromEntryV0(entry),
			Issues: issues,
		}
	}

	return DispatchDecisionV0{
		Kind:   DispatchDecisionNoPendingV0,
		Reason: "sin entrada pendiente elegible",
		Issues: issues,
	}
}

func entryMatchesSelectionV0(entry OutboxPendingEntryV0, selection DispatchSelectionV0) bool {
	if entry.TargetPort != selection.TargetPort {
		return false
	}
	if selection.MessageType != "" && entry.MessageType != selection.MessageType {
		return false
	}
	return selection.RunID == "" || entry.RunID == selection.RunID
}

func intentFromEntryV0(entry OutboxPendingEntryV0) DispatchIntentV0 {
	return DispatchIntentV0{
		MessageID:      entry.MessageID,
		RunID:          entry.RunID,
		TargetPort:     entry.TargetPort,
		MessageType:    entry.MessageType,
		IdempotencyKey: entry.IdempotencyKey,
		CorrelationID:  entry.CorrelationID,
		PayloadVersion: entry.PayloadVersion,
		Payload:        cloneBytesV0(entry.Payload),
	}
}
