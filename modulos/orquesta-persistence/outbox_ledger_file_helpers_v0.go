package orquestapersistence

import (
	"time"

	orquestadirectorcycleoutbox "orquesta/modulos/orquesta-director-cycle-outbox"
	orquestaoutboxdispatch "orquesta/modulos/orquesta-outbox-dispatch"
)

const fileOutboxClaimLeaseV0 = 5 * time.Minute

func directorIssuesFromOutboxLedgerV0(
	issues []OutboxLedgerIssueV0,
) []orquestadirectorcycleoutbox.DirectorCycleOutboxIssueV0 {
	out := make([]orquestadirectorcycleoutbox.DirectorCycleOutboxIssueV0, 0, len(issues))
	for _, issue := range issues {
		out = append(out, orquestadirectorcycleoutbox.DirectorCycleOutboxIssueV0(issue))
	}
	return out
}

func dispatchIssuesFromOutboxLedgerV0(
	issues []OutboxLedgerIssueV0,
) []orquestaoutboxdispatch.DispatchIssueV0 {
	out := make([]orquestaoutboxdispatch.DispatchIssueV0, 0, len(issues))
	for _, issue := range issues {
		out = append(out, orquestaoutboxdispatch.DispatchIssueV0(issue))
	}
	return out
}

func normalizeFileOutboxDirectorFilterV0(
	filter orquestadirectorcycleoutbox.DirectorCycleOutboxPendingFilterV0,
) (orquestadirectorcycleoutbox.DirectorCycleOutboxPendingFilterV0, []orquestadirectorcycleoutbox.DirectorCycleOutboxIssueV0) {
	filter.RunRef = trimV0(filter.RunRef)
	filter.TargetPort = trimV0(filter.TargetPort)
	if filter.TargetPort != "" && !outboxLedgerTargetPortSupportedV0(filter.TargetPort) {
		return filter, []orquestadirectorcycleoutbox.DirectorCycleOutboxIssueV0{{
			Code: ErrPayloadInvalidoV0, Field: "target_port", Message: "target_port no soportado",
		}}
	}
	return filter, nil
}

func normalizeFileOutboxDispatchFilterV0(
	filter orquestaoutboxdispatch.PendingOutboxFilterV0,
) (orquestaoutboxdispatch.PendingOutboxFilterV0, []orquestaoutboxdispatch.DispatchIssueV0) {
	filter.RunID = trimV0(filter.RunID)
	filter.TargetPort = trimV0(filter.TargetPort)
	filter.MessageType = trimV0(filter.MessageType)
	if filter.TargetPort != "" && !outboxLedgerTargetPortSupportedV0(filter.TargetPort) {
		return filter, []orquestaoutboxdispatch.DispatchIssueV0{{
			Code: ErrPayloadInvalidoV0, Field: "target_port", Message: "target_port no soportado",
		}}
	}
	return filter, nil
}

func normalizeStoredFileOutboxClaimV0(claim fileOutboxLedgerClaimV0) fileOutboxLedgerClaimV0 {
	claim.MessageID = trimV0(claim.MessageID)
	claim.RunID = trimV0(claim.RunID)
	claim.TargetPort = trimV0(claim.TargetPort)
	claim.IdempotencyKey = trimV0(claim.IdempotencyKey)
	claim.ClaimRef = trimV0(claim.ClaimRef)
	claim.LeaseRef = trimV0(claim.LeaseRef)
	claim.ClaimedAt = trimV0(claim.ClaimedAt)
	if claim.ClaimRef == "" {
		claim.ClaimRef = "claim-ref-" + claim.MessageID
	}
	if claim.LeaseRef == "" {
		claim.LeaseRef = "lease-ref-" + claim.MessageID
	}
	return claim
}

func normalizeStoredFileOutboxAckV0(ack fileOutboxLedgerAckV0) fileOutboxLedgerAckV0 {
	ack.MessageID = trimV0(ack.MessageID)
	ack.RunID = trimV0(ack.RunID)
	ack.TargetPort = trimV0(ack.TargetPort)
	ack.Status = trimV0(ack.Status)
	ack.DispatchRef = trimV0(ack.DispatchRef)
	ack.EvidenceRefs = compactOutboxLedgerStringsV0(ack.EvidenceRefs)
	ack.IssueCodes = compactOutboxLedgerStringsV0(ack.IssueCodes)
	ack.Issues = compactFileOutboxIssuesV0(ack.Issues)
	if len(ack.EvidenceRefs) == 0 {
		ack.EvidenceRefs = nil
	}
	if len(ack.IssueCodes) == 0 {
		ack.IssueCodes = nil
	}
	if len(ack.Issues) == 0 {
		ack.Issues = nil
	}
	return ack
}

func fileOutboxClaimFromDispatchV0(
	claim orquestaoutboxdispatch.OutboxDispatchClaimV0,
) fileOutboxLedgerClaimV0 {
	return normalizeStoredFileOutboxClaimV0(fileOutboxLedgerClaimV0{
		MessageID:      claim.MessageID,
		RunID:          claim.RunID,
		TargetPort:     claim.TargetPort,
		IdempotencyKey: claim.IdempotencyKey,
		ClaimedAt:      fileOutboxClaimedAtNowV0(),
	})
}

func fileOutboxClaimRecoverableInProcessV0(claim *fileOutboxLedgerClaimV0) bool {
	if claim == nil || claim.Recovered {
		return true
	}
	claimedAt := trimV0(claim.ClaimedAt)
	if claimedAt == "" {
		return true
	}
	parsed, err := time.Parse(time.RFC3339, claimedAt)
	if err != nil {
		return true
	}
	return time.Since(parsed) >= fileOutboxClaimLeaseV0
}

func fileOutboxClaimedAtNowV0() string {
	return time.Now().UTC().Format(time.RFC3339)
}

func fileOutboxAckFromSuccessV0(
	ack orquestaoutboxdispatch.OutboxDispatchAckV0,
) fileOutboxLedgerAckV0 {
	return normalizeStoredFileOutboxAckV0(fileOutboxLedgerAckV0{
		MessageID:    ack.MessageID,
		RunID:        ack.RunID,
		TargetPort:   ack.TargetPort,
		Status:       OutboxDispatchStatusDispatchedV0,
		DispatchRef:  ack.DispatchRef,
		EvidenceRefs: ack.EvidenceRefs,
	})
}

func fileOutboxAckFromObservationV0(
	ack orquestaoutboxdispatch.OutboxDispatchAckObservationV0,
) fileOutboxLedgerAckV0 {
	status := OutboxDispatchStatusDispatchedV0
	if ack.Status == orquestaoutboxdispatch.OutboxDispatchAckObservationFailedV0 {
		status = OutboxDispatchStatusFailedV0
	}
	return normalizeStoredFileOutboxAckV0(fileOutboxLedgerAckV0{
		MessageID:    ack.MessageID,
		RunID:        ack.RunID,
		TargetPort:   ack.TargetPort,
		Status:       status,
		DispatchRef:  ack.DispatchRef,
		EvidenceRefs: ack.EvidenceRefs,
		IssueCodes:   fileOutboxIssueCodesV0(ack.Issues),
		Issues:       fileOutboxIssuesV0(ack.Issues),
	})
}

func fileOutboxIssueCodesV0(issues []orquestaoutboxdispatch.DispatchIssueV0) []string {
	codes := make([]string, 0, len(issues))
	for _, issue := range issues {
		if code := trimV0(issue.Code); code != "" {
			codes = append(codes, code)
		}
	}
	return compactOutboxLedgerStringsV0(codes)
}

func fileOutboxIssuesV0(issues []orquestaoutboxdispatch.DispatchIssueV0) []OutboxLedgerIssueV0 {
	out := make([]OutboxLedgerIssueV0, 0, len(issues))
	for _, issue := range issues {
		normalized := OutboxLedgerIssueV0{
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

func compactFileOutboxIssuesV0(issues []OutboxLedgerIssueV0) []OutboxLedgerIssueV0 {
	out := make([]OutboxLedgerIssueV0, 0, len(issues))
	for _, issue := range issues {
		normalized := OutboxLedgerIssueV0{
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

func fileOutboxClaimMatchesRecordV0(
	claim fileOutboxLedgerClaimV0,
	record *fileOutboxLedgerRecordV0,
) bool {
	return claim.MessageID == record.Message.MessageID &&
		(claim.RunID == "" || claim.RunID == record.Message.RunID) &&
		(claim.TargetPort == "" || claim.TargetPort == record.Message.TargetPort) &&
		(claim.IdempotencyKey == "" || claim.IdempotencyKey == record.Message.IdempotencyKey)
}

func fileOutboxAckMatchesRecordV0(
	ack fileOutboxLedgerAckV0,
	record *fileOutboxLedgerRecordV0,
) bool {
	return ack.MessageID == record.Message.MessageID &&
		(ack.RunID == "" || ack.RunID == record.Message.RunID) &&
		(ack.TargetPort == "" || ack.TargetPort == record.Message.TargetPort)
}

func fileOutboxRecordPendingForDirectorV0(
	record *fileOutboxLedgerRecordV0,
	filter orquestadirectorcycleoutbox.DirectorCycleOutboxPendingFilterV0,
) bool {
	if record == nil || record.Ack != nil {
		return false
	}
	if filter.RunRef != "" && record.Message.RunID != filter.RunRef {
		return false
	}
	return filter.TargetPort == "" || record.Message.TargetPort == filter.TargetPort
}

func fileOutboxRecordPendingForDispatchV0(
	record *fileOutboxLedgerRecordV0,
	filter orquestaoutboxdispatch.PendingOutboxFilterV0,
) bool {
	if record == nil || record.Ack != nil {
		return false
	}
	if record.Claim != nil && !fileOutboxClaimRecoverableInProcessV0(record.Claim) {
		return false
	}
	message := record.Message
	if filter.RunID != "" && message.RunID != filter.RunID {
		return false
	}
	if filter.TargetPort != "" && message.TargetPort != filter.TargetPort {
		return false
	}
	return filter.MessageType == "" || message.MessageType == filter.MessageType
}

func fileOutboxEntryFromRecordV0(record *fileOutboxLedgerRecordV0) orquestaoutboxdispatch.OutboxPendingEntryV0 {
	message := record.Message
	return orquestaoutboxdispatch.OutboxPendingEntryV0{
		MessageID:      message.MessageID,
		RunID:          message.RunID,
		TargetPort:     message.TargetPort,
		MessageType:    message.MessageType,
		IdempotencyKey: message.IdempotencyKey,
		CorrelationID:  message.CorrelationID,
		PayloadVersion: message.PayloadVersion,
		Payload:        append([]byte(nil), message.Payload...),
	}
}

func fileOutboxSnapshotFromAckV0(ack fileOutboxLedgerAckV0) OutboxDispatchSnapshotV0 {
	return OutboxDispatchSnapshotV0{
		MessageID:    ack.MessageID,
		RunID:        ack.RunID,
		TargetPort:   ack.TargetPort,
		Status:       ack.Status,
		DispatchRef:  ack.DispatchRef,
		EvidenceRefs: append([]string(nil), ack.EvidenceRefs...),
		Issues:       append([]OutboxLedgerIssueV0(nil), ack.Issues...),
	}
}
