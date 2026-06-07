package orquestacoreworkflow

import "testing"

func mustReducerStartedRunV0(t *testing.T) OrchestrationRunV0 {
	t.Helper()
	return mustApplyReducerEventV0(t, OrchestrationRunV0{}, mustReducerRunStartedEventV0(t, "evt-run-started-reducer-base", 1))
}

func mustApplyReducerEventV0(t *testing.T, current OrchestrationRunV0, event OrchestrationEventV0) OrchestrationRunV0 {
	t.Helper()
	next, err := ApplyEventV0(current, event)
	if err != nil {
		t.Fatalf("apply %s: %v", event.EventType, err)
	}
	return next
}

func mustReducerRunStartedEventV0(t *testing.T, eventID string, sequence int64) OrchestrationEventV0 {
	t.Helper()
	event, err := NewRunStartedEventV0(reducerEventMetaV0(eventID, sequence), RunStartedPayloadV0{
		ProjectRef:  "project:ventas",
		AppSpecRef:  "appspec:req-001",
		RequestedBy: "director",
	})
	return mustReducerEventV0(t, event, err)
}

func mustReducerPhaseOpenedEventV0(t *testing.T, eventID string, sequence int64, phaseID OrchestrationPhaseIDV0) OrchestrationEventV0 {
	t.Helper()
	event, err := NewPhaseOpenedEventV0(reducerEventMetaV0(eventID, sequence), PhaseOpenedPayloadV0{
		PhaseID:  string(phaseID),
		OpenedBy: "director",
		Reason:   "apertura de fase canonica",
	})
	return mustReducerEventV0(t, event, err)
}

func mustReducerRunBlockedEventV0(t *testing.T, eventID string, sequence int64) OrchestrationEventV0 {
	t.Helper()
	return mustReducerRunBlockedWithRefEventV0(t, eventID, sequence, "blocker-001")
}

func mustReducerRunBlockedWithRefEventV0(t *testing.T, eventID string, sequence int64, blockerID string) OrchestrationEventV0 {
	t.Helper()
	event, err := NewRunBlockedEventV0(reducerEventMetaV0(eventID, sequence), RunBlockedPayloadV0{
		BlockerID:    blockerID,
		ReasonCode:   "consulta_director_requerida",
		Summary:      "Falta una decision de contrato antes de continuar.",
		SourceGroup:  "workflow",
		EvidenceRefs: []string{"docs/contratos.md#OrchestrationEventV0"},
	})
	return mustReducerEventV0(t, event, err)
}

func mustReducerRunBlockerResolvedEventV0(t *testing.T, eventID string, sequence int64, blockerID string) OrchestrationEventV0 {
	t.Helper()
	event, err := NewRunBlockerResolvedEventV0(reducerEventMetaV0(eventID, sequence), RunBlockerResolvedPayloadV0{
		BlockerID:    blockerID,
		ReasonCode:   "bloqueo_resuelto",
		Summary:      "El criterio pendiente queda resuelto con evidencia durable.",
		EvidenceRefs: []string{"docs/contratos.md#RunBlockerResolved"},
	})
	return mustReducerEventV0(t, event, err)
}

func mustReducerDirectorQuestionRaisedEventV0(t *testing.T, eventID string, sequence int64, blocking bool) OrchestrationEventV0 {
	t.Helper()
	payload := DirectorQuestionRaisedPayloadV0{
		QuestionID:   "question-001",
		SourceGroup:  "workflow",
		TargetGroup:  "director",
		Summary:      "Falta una decision de contrato antes de continuar.",
		EvidenceRefs: []string{"docs/contratos.md#DirectorQuestionRaised"},
		Blocking:     blocking,
	}
	if blocking {
		payload.BlockerID = "director-question-question-001"
	}
	event, err := NewDirectorQuestionRaisedEventV0(reducerEventMetaV0(eventID, sequence), payload)
	return mustReducerEventV0(t, event, err)
}

func mustReducerDirectorQuestionAnsweredEventV0(t *testing.T, eventID string, sequence int64, unblocks bool) OrchestrationEventV0 {
	t.Helper()
	payload := DirectorQuestionAnsweredPayloadV0{
		AnswerID:     "answer-001",
		QuestionID:   "question-001",
		Decision:     DirectorAnswerDecisionContinueV0,
		Summary:      "Continuar con decision compacta y evidencia acotada.",
		EvidenceRefs: []string{"docs/contratos.md#DirectorQuestionAnswered"},
		Unblocks:     unblocks,
	}
	if unblocks {
		payload.BlockerID = "director-question-question-001"
	}
	event, err := NewDirectorQuestionAnsweredEventV0(reducerEventMetaV0(eventID, sequence), payload)
	return mustReducerEventV0(t, event, err)
}

func mustReducerEventV0(t *testing.T, event OrchestrationEventV0, err error) OrchestrationEventV0 {
	t.Helper()
	if err != nil {
		t.Fatalf("event constructor failed: %v", err)
	}
	return event
}

func reducerEventMetaV0(eventID string, sequence int64) OrchestrationEventMetaV0 {
	return OrchestrationEventMetaV0{
		EventID:        eventID,
		RunID:          "run-001",
		Sequence:       sequence,
		IdempotencyKey: "idem-" + eventID,
		CorrelationID:  "corr-reducer-001",
		CausationID:    "cmd-reducer-001",
		OccurredAt:     "2026-05-04T10:00:00Z",
	}
}

func reducerUnknownEventV0() OrchestrationEventV0 {
	meta := reducerEventMetaV0("evt-unknown-reducer-001", 2)
	return OrchestrationEventV0{
		EventID:        meta.EventID,
		EventType:      "AgentUnknown",
		RunID:          meta.RunID,
		Sequence:       meta.Sequence,
		IdempotencyKey: meta.IdempotencyKey,
		CorrelationID:  meta.CorrelationID,
		CausationID:    meta.CausationID,
		OccurredAt:     meta.OccurredAt,
		PayloadVersion: OrchestrationEventPayloadVersionV0,
		Payload:        []byte(`{}`),
	}
}

func activePhaseIDsV0(run OrchestrationRunV0) []OrchestrationPhaseIDV0 {
	var active []OrchestrationPhaseIDV0
	for _, phase := range run.Phases {
		if phase.Status == OrchestrationPhaseStatusActiveV0 {
			active = append(active, phase.ID)
		}
	}
	return active
}

func reducerPhaseByIDV0(t *testing.T, run OrchestrationRunV0, phaseID OrchestrationPhaseIDV0) OrchestrationPhaseV0 {
	t.Helper()
	for _, phase := range run.Phases {
		if normalizePhaseIDV0(phase.ID) == phaseID {
			return phase
		}
	}
	t.Fatalf("phase %s not found", phaseID)
	return OrchestrationPhaseV0{}
}

func assertReducerRunValidV0(t *testing.T, run OrchestrationRunV0) {
	t.Helper()
	if issues := ValidateOrchestrationRunV0(run); len(issues) > 0 {
		t.Fatalf("run invalido: %+v", issues)
	}
}
