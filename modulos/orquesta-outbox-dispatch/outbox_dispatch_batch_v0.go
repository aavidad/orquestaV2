package orquestaoutboxdispatch

import "strings"

const defaultDispatchBatchMaxReadyV0 = 1

func ChooseDispatchBatchV0(selection DispatchBatchSelectionV0) DispatchBatchDecisionV0 {
	normalized := normalizeBatchSelectionV0(selection)
	if normalized.TargetPort == "" {
		return DispatchBatchDecisionV0{
			Kind:   DispatchDecisionInvalidRequestV0,
			Reason: "target_port requerido",
			Issues: []DispatchIssueV0{issueV0(issueInvalidRequestV0, "target_port", "target_port requerido")},
		}
	}

	claimed := claimedSetV0(normalized.ClaimedMessageIDs)
	intents := make([]DispatchIntentV0, 0, normalized.MaxReady)
	issues := make([]DispatchIssueV0, 0)
	for _, entry := range normalized.Pending {
		entry = normalizeEntryV0(entry)
		if !entryMatchesBatchSelectionV0(entry, normalized) {
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
		intents = append(intents, intentFromEntryV0(entry))
		claimed[entry.MessageID] = true
		if len(intents) >= normalized.MaxReady {
			break
		}
	}
	if len(intents) == 0 {
		return DispatchBatchDecisionV0{
			Kind:   DispatchDecisionNoPendingV0,
			Reason: "sin entradas pendientes elegibles",
			Issues: issues,
		}
	}
	return DispatchBatchDecisionV0{
		Kind:    DispatchDecisionReadyV0,
		Reason:  "entradas pendientes seleccionadas",
		Intents: intents,
		Issues:  issues,
	}
}

func normalizeBatchSelectionV0(selection DispatchBatchSelectionV0) DispatchBatchSelectionV0 {
	normalized := DispatchBatchSelectionV0{
		RunID:             strings.TrimSpace(selection.RunID),
		TargetPort:        strings.TrimSpace(selection.TargetPort),
		MessageType:       strings.TrimSpace(selection.MessageType),
		Pending:           cloneEntriesV0(selection.Pending),
		ClaimedMessageIDs: compactStringsV0(selection.ClaimedMessageIDs),
		MaxReady:          selection.MaxReady,
	}
	if normalized.MaxReady <= 0 {
		normalized.MaxReady = defaultDispatchBatchMaxReadyV0
	}
	return normalized
}

func entryMatchesBatchSelectionV0(
	entry OutboxPendingEntryV0,
	selection DispatchBatchSelectionV0,
) bool {
	if entry.TargetPort != selection.TargetPort {
		return false
	}
	if selection.MessageType != "" && entry.MessageType != selection.MessageType {
		return false
	}
	return selection.RunID == "" || entry.RunID == selection.RunID
}
