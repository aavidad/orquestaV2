package orquestacoreworkflow

import "testing"

func TestRecordReplanDecisionCommandV0RejectsMissingReworkSource(t *testing.T) {
	run := mustRequestReworkReadyRunV0(t, "review-result-replan-missing-source", ReviewResultStatusChangesRequestedV0)
	command := mustRecordReplanDecisionCommandV0(t, "cmd-record-replan-missing-source", "idem-record-replan-missing-source", "replan-decision-missing-source")

	_, err := HandleCommandV0(run, command)
	assertRecordReplanDecisionCommandErrorV0(t, err, ErrTransicionInvalidaV0)
}

func TestRecordReplanDecisionCommandV0RejectsMissingTask(t *testing.T) {
	run := mustReplanDecisionReadyRunV0(t)
	run.Tasks = nil
	run.ReplanDecisions = nil
	command := mustRecordReplanDecisionCommandV0(t, "cmd-record-replan-missing-task", "idem-record-replan-missing-task", "replan-decision-missing-task")

	_, err := HandleCommandV0(run, command)
	assertRecordReplanDecisionCommandErrorV0(t, err, ErrTransicionInvalidaV0)
}

func TestRecordReplanDecisionCommandV0RejectsNonReviewPhase(t *testing.T) {
	run := mustReplanDecisionReadyRunV0(t)
	open := mustOpenPhaseCommandV0(t, "cmd-open-doc-after-replan", "idem-open-doc-after-replan", OrchestrationPhaseDocumentacionV0)
	run = mustApplySingleCommandEventV0(t, run, open)
	command := mustRecordReplanDecisionCommandV0(t, "cmd-record-replan-phase", "idem-record-replan-phase", "replan-decision-phase")

	_, err := HandleCommandV0(run, command)
	assertRecordReplanDecisionCommandErrorV0(t, err, ErrTransicionInvalidaV0)
}

func TestRecordReplanDecisionCommandV0RejectsConflictingRef(t *testing.T) {
	run := mustReplanDecisionReadyRunV0(t)
	first := mustRecordReplanDecisionCommandV0(t, "cmd-record-replan-conflict-first", "idem-record-replan-conflict-first", "replan-decision-conflict")
	run = mustApplySingleCommandEventV0(t, run, first)
	payload := validReplanDecisionPayloadV0("replan-decision-conflict")
	payload.AcceptedAction = ReplanDecisionActionAskDirectorV0
	payload.FollowupRefs = []string{"followup-director-question-001"}
	conflict, err := NewRecordReplanDecisionCommandV0(validCommandMetaV0("cmd-record-replan-conflict-second", "idem-record-replan-conflict-second"), payload)
	conflict = mustCommandV0(t, conflict, err)

	_, err = HandleCommandV0(run, conflict)
	assertRecordReplanDecisionCommandErrorV0(t, err, ErrTransicionInvalidaV0)
}

func TestReplanDecisionRecordedEventV0RejectsMissingSource(t *testing.T) {
	run := mustRequestReworkReadyRunV0(t, "review-result-replan-event-missing-source", ReviewResultStatusRejectedV0)
	event := mustReplanDecisionRecordedEventV0(t, "evt-record-replan-missing-source", run.LastSequence+1, validReplanDecisionPayloadV0("replan-decision-event-missing-source"))

	_, err := ApplyEventV0(run, event)
	assertReplanDecisionRecordedEventErrorV0(t, err, ErrSecuenciaInvalidaV0)
}

func TestRecordReplanDecisionCommandV0RejectsForbiddenDetails(t *testing.T) {
	payload := validReplanDecisionPayloadV0("replan-decision-forbidden")
	payload.Summary = "usar provider externo"

	_, err := NewRecordReplanDecisionCommandV0(validCommandMetaV0("cmd-record-replan-forbidden", "idem-record-replan-forbidden"), payload)
	assertRecordReplanDecisionCommandErrorV0(t, err, ErrDetalleProhibidoV0)
}

func TestRecordReplanDecisionCommandV0RejectsProjectionSeparators(t *testing.T) {
	payload := validReplanDecisionPayloadV0("replan-decision#source:unsafe")

	_, err := NewRecordReplanDecisionCommandV0(validCommandMetaV0("cmd-record-replan-unsafe", "idem-record-replan-unsafe"), payload)
	assertRecordReplanDecisionCommandErrorV0(t, err, ErrPayloadInvalidoV0)
}

func TestReplanDecisionRecordedEventV0RejectsForbiddenDetails(t *testing.T) {
	payload := validReplanDecisionPayloadV0("replan-decision-event-forbidden")
	payload.Summary = "usar Claude"

	_, err := NewReplanDecisionRecordedEventV0(reducerEventMetaV0("evt-record-replan-forbidden", 18), payload)
	assertReplanDecisionRecordedEventErrorV0(t, err, ErrDetalleProhibidoV0)
}

func TestValidateOrchestrationRunV0RejectsInvalidReplanDecisionProjection(t *testing.T) {
	run := validRunForSerializationTestV0()
	payload := validReplanDecisionPayloadV0("replan-decision-invalid")
	run.Tasks = []string{payload.TaskRef}
	run.ReworkRequests = []string{"broken-rework-projection"}
	run.ReplanDecisions = []string{replanDecisionProjectionRefV0(payload)}

	issues := ValidateOrchestrationRunV0(run)
	if len(issues) == 0 {
		t.Fatal("run con replan decision sobre rework inexistente aceptado")
	}
	if issues[0].Code != OrchestrationEstadoInconsistenteV0 || issues[0].Field != "rework_requests" {
		t.Fatalf("issue=%+v, want estado_inconsistente rework_requests", issues[0])
	}
}
