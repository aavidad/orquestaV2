package orquestacoreworkflow

import (
	"errors"
	"reflect"
	"testing"
)

func TestRecordQualityGateCommandV0ProjectsGateAndNoOutbox(t *testing.T) {
	run := mustRunWithPhaseV0(t, OrchestrationPhaseBrainstormingArquitecturaV0)
	command := mustRecordQualityGateCommandV0(
		t,
		"cmd-quality-gate-001",
		"idem-quality-gate-001",
		validQualityGatePayloadV0("quality-gate-001", QualityGateDecisionReworkRequiredV0),
	)

	result, err := HandleCommandV0(run, command)
	if err != nil {
		t.Fatalf("HandleCommandV0 quality gate: %v", err)
	}
	assertSingleEventTypeV0(t, result, OrchestrationEventQualityGateRecordedV0)
	if len(result.Outbox) != 0 {
		t.Fatalf("outbox=%+v, want none", result.Outbox)
	}

	next := mustApplyReducerEventV0(t, run, result.Events[0])
	want := []string{qualityGateProjectionRefV0(qualityGateRecordedPayloadFromCommandV0(validQualityGatePayloadV0("quality-gate-001", QualityGateDecisionReworkRequiredV0)))}
	if !reflect.DeepEqual(next.QualityGates, want) {
		t.Fatalf("quality_gates=%v, want %v", next.QualityGates, want)
	}
}

func TestRecordQualityGateCommandV0IdempotenteYConflicto(t *testing.T) {
	run := mustRunWithPhaseV0(t, OrchestrationPhaseBrainstormingArquitecturaV0)
	payload := validQualityGatePayloadV0("quality-gate-repeat", QualityGateDecisionAcceptedV0)
	command := mustRecordQualityGateCommandV0(t, "cmd-quality-repeat", "idem-quality-repeat", payload)
	applied := mustApplySingleCommandEventV0(t, run, command)

	repeated, err := HandleCommandV0(applied, command)
	assertIdempotentNoEventsV0(t, repeated, err)

	conflictPayload := payload
	conflictPayload.Decision = QualityGateDecisionBlockedV0
	conflictPayload.IssueRefs = []string{"quality-issue-blocked"}
	conflict := mustRecordQualityGateCommandV0(t, "cmd-quality-conflict", "idem-quality-conflict", conflictPayload)
	_, err = HandleCommandV0(applied, conflict)
	assertQualityGateCommandErrorV0(t, err, ErrTransicionInvalidaV0, "payload.gate_ref")
}

func TestRecordQualityGateCommandV0RejectsIssueRefsMissingForRework(t *testing.T) {
	payload := validQualityGatePayloadV0("quality-gate-no-issues", QualityGateDecisionReworkRequiredV0)
	payload.IssueRefs = nil

	_, err := NewRecordQualityGateCommandV0(
		validCommandMetaV0("cmd-quality-no-issues", "idem-quality-no-issues"),
		payload,
	)

	assertQualityGateCommandErrorV0(t, err, ErrPayloadInvalidoV0, "payload.issue_refs")
}

func TestReplayDurableEventsV0AcceptsQualityGateRecordedV0(t *testing.T) {
	payload := validQualityGatePayloadV0("quality-gate-replay", QualityGateDecisionAcceptedV0)
	event := mustQualityGateRecordedEventWithKeyV0(
		t,
		"evt-quality-gate-replay",
		3,
		"idem-quality-gate-replay",
		payload,
	)

	got, err := ReplayDurableEventsV0([]OrchestrationEventV0{
		mustReplayRunStartedEventWithKeyV0(t, "evt-quality-start", 1, "idem-quality-start"),
		mustReplayPhaseEventWithKeyV0(t, "evt-quality-phase", 2, "idem-quality-phase", OrchestrationPhaseBrainstormingArquitecturaV0),
		event,
		event,
	})
	if err != nil {
		t.Fatalf("ReplayDurableEventsV0 quality gate: %v", err)
	}
	want := []string{qualityGateProjectionRefV0(qualityGateRecordedPayloadFromCommandV0(payload))}
	if !reflect.DeepEqual(got.QualityGates, want) {
		t.Fatalf("quality_gates=%v, want %v", got.QualityGates, want)
	}
}

func TestValidateOrchestrationRunV0RejectsInvalidQualityGateProjection(t *testing.T) {
	run := mustRunWithPhaseV0(t, OrchestrationPhaseBrainstormingArquitecturaV0)
	run.QualityGates = []string{"quality-gate-invalid"}

	issues := ValidateOrchestrationRunV0(run)

	assertValidationIssueV0(t, issues, OrchestrationEstadoInconsistenteV0, "quality_gates")
}

func mustRecordQualityGateCommandV0(
	t *testing.T,
	commandID string,
	idempotencyKey string,
	payload RecordQualityGateCommandPayloadV0,
) OrchestrationCommandV0 {
	t.Helper()
	command, err := NewRecordQualityGateCommandV0(validCommandMetaV0(commandID, idempotencyKey), payload)
	return mustCommandV0(t, command, err)
}

func mustQualityGateRecordedEventWithKeyV0(
	t *testing.T,
	eventID string,
	sequence int64,
	idempotencyKey string,
	payload QualityGateRecordedPayloadV0,
) OrchestrationEventV0 {
	t.Helper()
	meta := reducerEventMetaV0(eventID, sequence)
	meta.IdempotencyKey = idempotencyKey
	event, err := NewQualityGateRecordedEventV0(meta, payload)
	if err != nil {
		t.Fatalf("NewQualityGateRecordedEventV0: %v", err)
	}
	return event
}

func validQualityGatePayloadV0(
	gateRef string,
	decision QualityGateDecisionV0,
) RecordQualityGateCommandPayloadV0 {
	payload := RecordQualityGateCommandPayloadV0{
		RunRef:       "run-001",
		GateRef:      gateRef,
		PhaseID:      string(OrchestrationPhaseBrainstormingArquitecturaV0),
		SubjectRef:   "subject-council-artifacts-001",
		Decision:     decision,
		Summary:      "Quality gate registrado con evidencia compacta.",
		EvidenceRefs: []string{"evidence-quality-gate-001"},
	}
	if decision != QualityGateDecisionAcceptedV0 {
		payload.IssueRefs = []string{"quality-issue-001"}
	}
	return payload
}

func assertQualityGateCommandErrorV0(t *testing.T, err error, code string, field string) {
	t.Helper()
	var publicErr OrchestrationCommandErrorV0
	if !errors.As(err, &publicErr) {
		t.Fatalf("expected OrchestrationCommandErrorV0, got %T %v", err, err)
	}
	if publicErr.Code != code || publicErr.Field != field {
		t.Fatalf("error=%+v, want code=%s field=%s", publicErr, code, field)
	}
}

func mustRunWithPhaseV0(t *testing.T, phaseID OrchestrationPhaseIDV0) OrchestrationRunV0 {
	t.Helper()
	run := mustHandlerStartedRunV0(t)
	return mustApplySingleCommandEventV0(t, run, mustOpenPhaseCommandV0(t, "cmd-quality-open-"+string(phaseID), "idem-quality-open-"+string(phaseID), phaseID))
}

func assertValidationIssueV0(
	t *testing.T,
	issues []OrchestrationValidationIssueV0,
	code OrchestrationValidationCodeV0,
	field string,
) {
	t.Helper()
	for _, issue := range issues {
		if issue.Code == code && issue.Field == field {
			return
		}
	}
	t.Fatalf("issue %s/%s no encontrada en %+v", code, field, issues)
}
