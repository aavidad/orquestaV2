package orquestacoreworkflow

import "strings"

func applyRunStartedEventV0(current OrchestrationRunV0, event OrchestrationEventV0) (OrchestrationRunV0, error) {
	if !orchestrationRunIsEmptyV0(current) {
		return current, eventErrorV0(ErrSecuenciaInvalidaV0, "event_type")
	}

	var payload RunStartedPayloadV0
	if err := decodePayloadV0(event.Payload, &payload); err != nil {
		return OrchestrationRunV0{}, err
	}

	phases := OrchestrationPhaseCatalogV0()
	effects, err := appendCommandEffectFromEventV0([]string{}, event, event.RunID)
	if err != nil {
		return OrchestrationRunV0{}, err
	}
	return OrchestrationRunV0{
		SchemaVersion:             OrchestrationRunSchemaVersionV0,
		RunID:                     strings.TrimSpace(event.RunID),
		ProjectRef:                strings.TrimSpace(payload.ProjectRef),
		AppSpecRef:                strings.TrimSpace(payload.AppSpecRef),
		Status:                    OrchestrationRunStatusActiveV0,
		CurrentPhase:              firstPhaseIDV0(phases),
		Phases:                    phases,
		Brainstorms:               []string{},
		Votes:                     []string{},
		Tasks:                     []string{},
		FunctionContracts:         []string{},
		Decisions:                 []string{},
		CapacityRequests:          []string{},
		CapacityDecisions:         []string{},
		Agents:                    []string{},
		StartedAgents:             []string{},
		FailedAgents:              []string{},
		StoppedAgents:             []string{},
		ConfirmedStoppedAgents:    []string{},
		AgentAssessments:          []string{},
		AgentLeaseExpirations:     []string{},
		ConcurrencyGates:          []string{},
		QualityGates:              []string{},
		PhaseArtifacts:            []string{},
		Deliveries:                []string{},
		DeliveredTasks:            []string{},
		DeliveredAgents:           []string{},
		Reviews:                   []string{},
		ReviewResults:             []string{},
		ReworkRequests:            []string{},
		ReplanDecisions:           []string{},
		AcceptedReviews:           []string{},
		ClosedTasks:               []string{},
		Validations:               []string{},
		Closures:                  []string{},
		DirectorQuestions:         []string{},
		DirectorAnswers:           []string{},
		DirectorAnsweredQuestions: []string{},
		Blockers:                  []string{},
		CommandEffects:            effects,
		LastEventID:               strings.TrimSpace(event.EventID),
		LastSequence:              event.Sequence,
	}, nil
}

func applyPhaseOpenedEventV0(current OrchestrationRunV0, event OrchestrationEventV0) (OrchestrationRunV0, error) {
	var payload PhaseOpenedPayloadV0
	if err := decodePayloadV0(event.Payload, &payload); err != nil {
		return current, err
	}
	if err := ensureRunCanApplyEventV0(current, event); err != nil {
		return current, err
	}

	phaseID := OrchestrationPhaseIDV0(strings.TrimSpace(payload.PhaseID))
	if err := ValidateOrchestrationPhaseIDV0(phaseID); err != nil {
		return current, err
	}
	if !runContainsPhaseV0(current, phaseID) {
		return current, issueV0(OrchestrationFaseInvalidaV0, "phases")
	}
	if err := ensureEventEffectCompatibleV0(current, event, event.EventID); err != nil {
		return current, err
	}

	next := cloneRunForReducerV0(current)
	for index := range next.Phases {
		phase := &next.Phases[index]
		if normalizePhaseIDV0(phase.ID) == phaseID {
			phase.Status = OrchestrationPhaseStatusActiveV0
			if strings.TrimSpace(phase.OpenedAt) == "" {
				phase.OpenedAt = strings.TrimSpace(event.OccurredAt)
			}
			continue
		}
		if phase.Status == OrchestrationPhaseStatusActiveV0 {
			phase.Status = OrchestrationPhaseStatusPendingV0
		}
	}
	effects, err := appendCommandEffectFromEventV0(next.CommandEffects, event, event.EventID)
	if err != nil {
		return current, err
	}
	next.CommandEffects = effects
	next.CurrentPhase = phaseID
	next.LastEventID = strings.TrimSpace(event.EventID)
	next.LastSequence = event.Sequence
	return next, nil
}

func applyRunBlockedEventV0(current OrchestrationRunV0, event OrchestrationEventV0) (OrchestrationRunV0, error) {
	var payload RunBlockedPayloadV0
	if err := decodePayloadV0(event.Payload, &payload); err != nil {
		return current, err
	}
	if err := ensureRunCanApplyEventV0(current, event); err != nil {
		return current, err
	}
	if err := ensureEventEffectCompatibleV0(current, event, payload.BlockerID); err != nil {
		return current, err
	}

	next := cloneRunForReducerV0(current)
	next.Status = OrchestrationRunStatusBlockedV0
	next.Blockers = appendUniqueCompactRefV0(next.Blockers, payload.BlockerID)
	effects, err := appendCommandEffectFromEventV0(next.CommandEffects, event, payload.BlockerID)
	if err != nil {
		return current, err
	}
	next.CommandEffects = effects
	next.LastEventID = strings.TrimSpace(event.EventID)
	next.LastSequence = event.Sequence
	return next, nil
}
