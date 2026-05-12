package orquestastatefileoutbox

import (
	"encoding/json"
	"errors"
	"strconv"
	"strings"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
)

func normalizeOutboxMessageV0(
	message orquestacoreworkflow.OutboxMessageV0,
) (orquestacoreworkflow.OutboxMessageV0, []byte, error) {
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
	fingerprint, err := json.Marshal(normalized)
	if err != nil {
		return orquestacoreworkflow.OutboxMessageV0{}, nil, err
	}
	return cloneOutboxMessageV0(normalized), fingerprint, nil
}

func canonicalJSONV0(raw json.RawMessage) ([]byte, error) {
	var decoded any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return nil, err
	}
	return json.Marshal(decoded)
}

func cloneOutboxMessageV0(
	message orquestacoreworkflow.OutboxMessageV0,
) orquestacoreworkflow.OutboxMessageV0 {
	message.Payload = append([]byte(nil), message.Payload...)
	return message
}

func outboxMessageIssueFromErrorV0(index int, err error) ledgerIssueV0 {
	var publicErr orquestacoreworkflow.OutboxMessageErrorV0
	if errors.As(err, &publicErr) {
		return ledgerIssueV0{
			Code:  publicErr.Code,
			Field: indexedFieldV0("messages", index, publicErr.Field),
		}
	}
	return ledgerIssueV0{
		Code: errPayloadInvalidV0, Field: indexedFieldV0("messages", index, "payload"),
		Message: err.Error(),
	}
}

func indexedFieldV0(prefix string, index int, field string) string {
	path := prefix + "." + strconv.Itoa(index)
	if field == "" {
		return path
	}
	return path + "." + field
}

func trimV0(value string) string {
	return strings.TrimSpace(value)
}

func compactStringsV0(values []string) []string {
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
