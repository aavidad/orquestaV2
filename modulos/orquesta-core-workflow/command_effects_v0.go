package orquestacoreworkflow

type commandEffectRecordV0 struct {
	EventType      string `json:"event_type"`
	SubjectRef     string `json:"subject_ref"`
	IdempotencyKey string `json:"idempotency_key"`
	CausationID    string `json:"causation_id"`
	EventID        string `json:"event_id"`
	PayloadHash    string `json:"payload_hash"`
}

func appendCommandEffectFromEventV0(
	effects []string,
	event OrchestrationEventV0,
	subjectRef string,
) ([]string, error) {
	record, err := commandEffectRecordFromEventV0(event, subjectRef)
	if err != nil {
		return nil, err
	}
	encoded, err := encodeCommandEffectRecordV0(record)
	if err != nil {
		return nil, err
	}
	return appendUniqueCompactRefV0(effects, encoded), nil
}

func ensureEventEffectCompatibleV0(
	run OrchestrationRunV0,
	event OrchestrationEventV0,
	subjectRef string,
) error {
	existing, ok, err := commandEffectForSubjectV0(run, event.EventType, subjectRef)
	if err != nil || !ok {
		return err
	}
	next, err := commandEffectRecordFromEventV0(event, subjectRef)
	if err != nil {
		return err
	}
	if !sameCommandEffectV0(existing, next) {
		return eventErrorV0(ErrEventoConflictivoV0, "idempotency")
	}
	return nil
}

func ensureCommandEffectMatchesV0(
	run OrchestrationRunV0,
	command OrchestrationCommandV0,
	eventType string,
	subjectRef string,
	eventPayload any,
) error {
	next, err := commandEffectRecordFromCommandV0(command, eventType, subjectRef, eventPayload)
	if err != nil {
		return err
	}
	return ensureCommandEffectRecordMatchesV0(run, eventType, subjectRef, next)
}

func ensureCommandEffectRecordMatchesV0(
	run OrchestrationRunV0,
	eventType string,
	subjectRef string,
	next commandEffectRecordV0,
) error {
	existing, ok, err := commandEffectForSubjectV0(run, eventType, subjectRef)
	if err != nil {
		return commandErrorV0(ErrTransicionInvalidaV0, "command_effects")
	}
	if !ok {
		return commandErrorV0(ErrTransicionInvalidaV0, "command_effects")
	}
	if !sameCommandEffectV0(existing, next) {
		return commandErrorV0(ErrTransicionInvalidaV0, commandEffectConflictFieldV0(existing, next))
	}
	return nil
}

func commandEffectRecordMatchesV0(
	run OrchestrationRunV0,
	eventType string,
	subjectRef string,
	next commandEffectRecordV0,
) (bool, error) {
	existing, ok, err := commandEffectForSubjectV0(run, eventType, subjectRef)
	if err != nil {
		return false, commandErrorV0(ErrTransicionInvalidaV0, "command_effects")
	}
	if !ok {
		return false, nil
	}
	return sameCommandEffectV0(existing, next), nil
}

func commandEffectRefsInvalidV0(run OrchestrationRunV0) bool {
	seen := map[string]bool{}
	for _, encoded := range run.CommandEffects {
		record, ok := decodeCommandEffectRecordV0(encoded)
		if !ok || commandEffectRecordKeyV0(record) == "" || seen[commandEffectRecordKeyV0(record)] {
			return true
		}
		seen[commandEffectRecordKeyV0(record)] = true
	}
	return false
}
