package orquestacoreworkflow

import (
	"encoding/json"
	"strings"
)

const ErrEventoConflictivoV0 = "evento_conflictivo"

type replayIdempotencyIndexV0 struct {
	byEventID        map[string]replayEventFingerprintV0
	byIdempotencyKey map[string]replayEventFingerprintV0
}

type replayEventFingerprintV0 struct {
	EventType   string
	RunID       string
	Sequence    int64
	CausationID string
	OccurredAt  string
	Payload     string
}

func newReplayIdempotencyIndexV0() replayIdempotencyIndexV0 {
	return replayIdempotencyIndexV0{
		byEventID:        map[string]replayEventFingerprintV0{},
		byIdempotencyKey: map[string]replayEventFingerprintV0{},
	}
}

func (index replayIdempotencyIndexV0) registerEventV0(event OrchestrationEventV0) (bool, error) {
	fingerprint, err := replayFingerprintV0(event)
	if err != nil {
		return false, err
	}
	if duplicate, err := index.checkExistingEventV0(event, fingerprint); duplicate || err != nil {
		return duplicate, err
	}
	index.byEventID[strings.TrimSpace(event.EventID)] = fingerprint
	if key := strings.TrimSpace(event.IdempotencyKey); key != "" {
		index.byIdempotencyKey[key] = fingerprint
	}
	return false, nil
}

func (index replayIdempotencyIndexV0) checkExistingEventV0(event OrchestrationEventV0, fingerprint replayEventFingerprintV0) (bool, error) {
	if existing, ok := index.byEventID[strings.TrimSpace(event.EventID)]; ok {
		return sameReplayFingerprintV0(existing, fingerprint), replayConflictIfDifferentV0(existing, fingerprint)
	}
	if key := strings.TrimSpace(event.IdempotencyKey); key != "" {
		if existing, ok := index.byIdempotencyKey[key]; ok {
			return sameReplayFingerprintV0(existing, fingerprint), replayConflictIfDifferentV0(existing, fingerprint)
		}
	}
	return false, nil
}

func replayFingerprintV0(event OrchestrationEventV0) (replayEventFingerprintV0, error) {
	payload, err := canonicalEventPayloadV0(event.Payload)
	if err != nil {
		return replayEventFingerprintV0{}, err
	}
	return replayEventFingerprintV0{
		EventType:   strings.TrimSpace(event.EventType),
		RunID:       strings.TrimSpace(event.RunID),
		Sequence:    event.Sequence,
		CausationID: strings.TrimSpace(event.CausationID),
		OccurredAt:  strings.TrimSpace(event.OccurredAt),
		Payload:     payload,
	}, nil
}

func canonicalEventPayloadV0(raw json.RawMessage) (string, error) {
	var decoded any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return "", eventErrorV0(ErrPayloadInvalidoV0, "payload")
	}
	encoded, err := json.Marshal(decoded)
	if err != nil {
		return "", eventErrorV0(ErrPayloadInvalidoV0, "payload")
	}
	return string(encoded), nil
}

func replayConflictIfDifferentV0(existing replayEventFingerprintV0, next replayEventFingerprintV0) error {
	if sameReplayFingerprintV0(existing, next) {
		return nil
	}
	return eventErrorV0(ErrEventoConflictivoV0, "idempotency")
}

func sameReplayFingerprintV0(left replayEventFingerprintV0, right replayEventFingerprintV0) bool {
	return left.EventType == right.EventType &&
		left.RunID == right.RunID &&
		left.Sequence == right.Sequence &&
		left.CausationID == right.CausationID &&
		left.OccurredAt == right.OccurredAt &&
		left.Payload == right.Payload
}
