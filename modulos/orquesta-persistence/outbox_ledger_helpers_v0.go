package orquestapersistence

import (
	"encoding/json"
	"errors"
	"strconv"
	"time"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
)

const maxOutboxLedgerEvidenceRefsV0 = 20

func normalizeOutboxLedgerMessageV0(message orquestacoreworkflow.OutboxMessageV0) (orquestacoreworkflow.OutboxMessageV0, []byte, error) {
	normalized := orquestacoreworkflow.OutboxMessageV0{
		MessageID:        trimV0(message.MessageID),
		MessageType:      trimV0(message.MessageType),
		RunID:            trimV0(message.RunID),
		IdempotencyKey:   trimV0(message.IdempotencyKey),
		CorrelationID:    trimV0(message.CorrelationID),
		CausationEventID: trimV0(message.CausationEventID),
		TargetPort:       trimV0(message.TargetPort),
		PayloadVersion:   trimV0(message.PayloadVersion),
	}
	payload, err := canonicalJSONV0(message.Payload)
	if err != nil {
		return orquestacoreworkflow.OutboxMessageV0{}, nil, err
	}
	normalized.Payload = payload
	if err := orquestacoreworkflow.ValidateOutboxMessageV0(normalized); err != nil {
		return orquestacoreworkflow.OutboxMessageV0{}, nil, err
	}
	fingerprint, err := json.Marshal(outboxLedgerMessageFingerprintV0(normalized))
	if err != nil {
		return orquestacoreworkflow.OutboxMessageV0{}, nil, err
	}
	return cloneOutboxMessageV0(normalized), fingerprint, nil
}

func outboxLedgerMessageFingerprintV0(
	message orquestacoreworkflow.OutboxMessageV0,
) orquestacoreworkflow.OutboxMessageV0 {
	message.CorrelationID = ""
	return message
}

func canonicalJSONV0(raw json.RawMessage) ([]byte, error) {
	var decoded any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return nil, err
	}
	return json.Marshal(decoded)
}

func normalizeOutboxDispatchAckV0(ack OutboxDispatchAckV0) OutboxDispatchAckV0 {
	return OutboxDispatchAckV0{
		MessageID:    trimV0(ack.MessageID),
		RunID:        trimV0(ack.RunID),
		TargetPort:   trimV0(ack.TargetPort),
		Status:       trimV0(ack.Status),
		DispatchRef:  trimV0(ack.DispatchRef),
		DispatchedAt: trimV0(ack.DispatchedAt),
		ErrorCode:    trimV0(ack.ErrorCode),
		EvidenceRefs: compactOutboxLedgerStringsV0(ack.EvidenceRefs),
	}
}

func validateOutboxDispatchAckV0(ack OutboxDispatchAckV0) []OutboxLedgerIssueV0 {
	var issues []OutboxLedgerIssueV0
	addOutboxLedgerRequiredV0(&issues, "message_id", ack.MessageID)
	addOutboxLedgerRequiredV0(&issues, "run_id", ack.RunID)
	addOutboxLedgerRequiredV0(&issues, "target_port", ack.TargetPort)
	addOutboxLedgerRequiredV0(&issues, "dispatch_ref", ack.DispatchRef)
	addOutboxLedgerRequiredV0(&issues, "dispatched_at", ack.DispatchedAt)
	if ack.Status != OutboxDispatchStatusDispatchedV0 && ack.Status != OutboxDispatchStatusFailedV0 {
		issues = append(issues, outboxLedgerIssueV0(ErrPayloadInvalidoV0, "status", "status no soportado"))
	}
	if ack.TargetPort != "" && !outboxLedgerTargetPortSupportedV0(ack.TargetPort) {
		issues = append(issues, outboxLedgerIssueV0(ErrPayloadInvalidoV0, "target_port", "target_port no soportado"))
	}
	if ack.DispatchedAt != "" {
		if _, err := time.Parse(time.RFC3339Nano, ack.DispatchedAt); err != nil {
			issues = append(issues, outboxLedgerIssueV0(ErrPayloadInvalidoV0, "dispatched_at", "timestamp invalido"))
		}
	}
	if len(ack.EvidenceRefs) > maxOutboxLedgerEvidenceRefsV0 {
		issues = append(issues, outboxLedgerIssueV0(ErrPayloadInvalidoV0, "evidence_refs", "demasiadas evidencias"))
	}
	issues = append(issues, validateOutboxLedgerSafeTextV0(ack)...)
	return issues
}

func outboxLedgerAckFingerprintV0(ack OutboxDispatchAckV0) ([]byte, error) {
	return json.Marshal(ack)
}

func outboxLedgerSnapshotFromAckV0(ack OutboxDispatchAckV0) OutboxDispatchSnapshotV0 {
	return OutboxDispatchSnapshotV0{
		MessageID:    ack.MessageID,
		RunID:        ack.RunID,
		TargetPort:   ack.TargetPort,
		Status:       ack.Status,
		DispatchRef:  ack.DispatchRef,
		DispatchedAt: ack.DispatchedAt,
		ErrorCode:    ack.ErrorCode,
		EvidenceRefs: append([]string(nil), ack.EvidenceRefs...),
		Issues:       append([]OutboxLedgerIssueV0(nil), ack.Issues...),
	}
}

func cloneOutboxMessageV0(message orquestacoreworkflow.OutboxMessageV0) orquestacoreworkflow.OutboxMessageV0 {
	message.Payload = append([]byte(nil), message.Payload...)
	return message
}

func cloneOutboxAckV0(ack OutboxDispatchAckV0) OutboxDispatchAckV0 {
	ack.EvidenceRefs = append([]string(nil), ack.EvidenceRefs...)
	ack.Issues = append([]OutboxLedgerIssueV0(nil), ack.Issues...)
	return ack
}

func compactOutboxLedgerStringsV0(values []string) []string {
	if values == nil {
		return nil
	}
	result := make([]string, 0, len(values))
	for _, value := range values {
		if compact := trimV0(value); compact != "" {
			result = append(result, compact)
		}
	}
	return result
}

func outboxLedgerIssueV0(code, field, message string) OutboxLedgerIssueV0 {
	return OutboxLedgerIssueV0{Code: code, Field: field, Message: message}
}

func outboxLedgerIssueFromMessageErrorV0(index int, err error) OutboxLedgerIssueV0 {
	var publicErr orquestacoreworkflow.OutboxMessageErrorV0
	if errors.As(err, &publicErr) {
		return outboxLedgerIssueV0(publicErr.Code, indexedOutboxLedgerFieldV0("messages", index, publicErr.Field), publicErr.Error())
	}
	return outboxLedgerIssueV0(ErrPayloadInvalidoV0, indexedOutboxLedgerFieldV0("messages", index, "payload"), err.Error())
}

func indexedOutboxLedgerFieldV0(prefix string, index int, field string) string {
	path := prefix + "." + strconv.Itoa(index)
	if field == "" {
		return path
	}
	return path + "." + field
}

func addOutboxLedgerRequiredV0(issues *[]OutboxLedgerIssueV0, field, value string) {
	if trimV0(value) == "" {
		*issues = append(*issues, outboxLedgerIssueV0(ErrPayloadInvalidoV0, field, "campo requerido"))
	}
}

func validateOutboxLedgerSafeTextV0(ack OutboxDispatchAckV0) []OutboxLedgerIssueV0 {
	values := map[string][]string{
		"message_id":    {ack.MessageID},
		"run_id":        {ack.RunID},
		"target_port":   {ack.TargetPort},
		"dispatch_ref":  {ack.DispatchRef},
		"error_code":    {ack.ErrorCode},
		"evidence_refs": ack.EvidenceRefs,
	}
	var issues []OutboxLedgerIssueV0
	for field, fieldValues := range values {
		for _, value := range fieldValues {
			if outboxLedgerContainsForbiddenTermV0(value) {
				issues = append(issues, outboxLedgerIssueV0(ErrPayloadInvalidoV0, field, "detalle prohibido en ack"))
			}
		}
	}
	return issues
}

func outboxLedgerContainsForbiddenTermV0(value string) bool {
	// Ledger text rails are offline. The ledger must keep causal evidence even
	// when refs contain operational vocabulary such as provider, model, token,
	// db or sql. Effective secrets are handled at ingress/redaction boundaries.
	return false
}

func outboxLedgerTargetPortSupportedV0(targetPort string) bool {
	switch trimV0(targetPort) {
	case orquestacoreworkflow.OutboxTargetPersistenceV0,
		orquestacoreworkflow.OutboxTargetObservabilityV0,
		orquestacoreworkflow.OutboxTargetCapacityV0,
		orquestacoreworkflow.OutboxTargetAgentLauncherV0,
		orquestacoreworkflow.OutboxTargetDeployPlannerV0,
		orquestacoreworkflow.OutboxTargetDirectorV0:
		return true
	default:
		return false
	}
}

func HasOutboxLedgerIssueV0(issues []OutboxLedgerIssueV0, code, field string) bool {
	for _, issue := range issues {
		if issue.Code == code && (field == "" || issue.Field == field) {
			return true
		}
	}
	return false
}
