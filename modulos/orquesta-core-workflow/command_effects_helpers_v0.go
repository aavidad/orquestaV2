package orquestacoreworkflow

import (
	"encoding/json"
	"strings"
)

func commandEffectForSubjectV0(
	run OrchestrationRunV0,
	eventType string,
	subjectRef string,
) (commandEffectRecordV0, bool, error) {
	key := commandEffectKeyV0(eventType, subjectRef)
	for _, encoded := range run.CommandEffects {
		record, ok := decodeCommandEffectRecordV0(encoded)
		if !ok {
			return commandEffectRecordV0{}, false, eventErrorV0(ErrPayloadInvalidoV0, "command_effects")
		}
		if commandEffectRecordKeyV0(record) == key {
			return record, true, nil
		}
	}
	return commandEffectRecordV0{}, false, nil
}

func commandEffectRecordFromCommandV0(
	command OrchestrationCommandV0,
	eventType string,
	subjectRef string,
	eventPayload any,
) (commandEffectRecordV0, error) {
	return commandEffectRecordFromExpectedEventV0(
		eventType,
		subjectRef,
		command.IdempotencyKey,
		command.CommandID,
		commandEventIDV0(command, eventType),
		eventPayload,
	)
}

func commandEffectRecordFromExpectedEventV0(
	eventType string,
	subjectRef string,
	idempotencyKey string,
	causationID string,
	eventID string,
	eventPayload any,
) (commandEffectRecordV0, error) {
	payloadHash, err := commandEffectPayloadHashFromValueV0(eventPayload)
	if err != nil {
		return commandEffectRecordV0{}, err
	}
	return commandEffectRecordV0{
		EventType:      strings.TrimSpace(eventType),
		SubjectRef:     strings.TrimSpace(subjectRef),
		IdempotencyKey: strings.TrimSpace(idempotencyKey),
		CausationID:    strings.TrimSpace(causationID),
		EventID:        strings.TrimSpace(eventID),
		PayloadHash:    payloadHash,
	}, nil
}

func commandEffectRecordFromEventV0(
	event OrchestrationEventV0,
	subjectRef string,
) (commandEffectRecordV0, error) {
	payloadHash, err := commandEffectPayloadHashFromRawV0(event.Payload)
	if err != nil {
		return commandEffectRecordV0{}, err
	}
	return commandEffectRecordV0{
		EventType:      strings.TrimSpace(event.EventType),
		SubjectRef:     strings.TrimSpace(subjectRef),
		IdempotencyKey: strings.TrimSpace(event.IdempotencyKey),
		CausationID:    strings.TrimSpace(event.CausationID),
		EventID:        strings.TrimSpace(event.EventID),
		PayloadHash:    payloadHash,
	}, nil
}

func encodeCommandEffectRecordV0(record commandEffectRecordV0) (string, error) {
	data, err := json.Marshal(record)
	if err != nil {
		return "", eventErrorV0(ErrPayloadInvalidoV0, "command_effects")
	}
	return string(data), nil
}

func decodeCommandEffectRecordV0(encoded string) (commandEffectRecordV0, bool) {
	var record commandEffectRecordV0
	if err := json.Unmarshal([]byte(strings.TrimSpace(encoded)), &record); err != nil {
		return commandEffectRecordV0{}, false
	}
	return record, commandEffectRecordValidV0(record)
}
