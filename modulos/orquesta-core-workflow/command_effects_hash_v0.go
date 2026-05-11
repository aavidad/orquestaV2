package orquestacoreworkflow

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
)

func commandEffectPayloadHashFromValueV0(value any) (string, error) {
	raw, err := json.Marshal(value)
	if err != nil {
		return "", commandErrorV0(ErrPayloadInvalidoV0, "payload")
	}
	return commandEffectPayloadHashFromRawV0(raw)
}

func commandEffectPayloadHashFromRawV0(raw json.RawMessage) (string, error) {
	canonical, err := canonicalEventPayloadV0(raw)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256([]byte(canonical))
	return hex.EncodeToString(sum[:]), nil
}

func sameCommandEffectV0(left commandEffectRecordV0, right commandEffectRecordV0) bool {
	return left.EventType == right.EventType &&
		left.SubjectRef == right.SubjectRef &&
		left.IdempotencyKey == right.IdempotencyKey &&
		left.CausationID == right.CausationID &&
		left.EventID == right.EventID &&
		left.PayloadHash == right.PayloadHash
}

func commandEffectConflictFieldV0(left commandEffectRecordV0, right commandEffectRecordV0) string {
	if left.IdempotencyKey != right.IdempotencyKey ||
		left.CausationID != right.CausationID ||
		left.EventID != right.EventID {
		return "idempotency_key"
	}
	return "payload"
}
