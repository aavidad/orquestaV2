package orquestacoreworkflow

import "testing"

func TestRecordReviewResultCommandV0RejectsNonReviewPhase(t *testing.T) {
	run := mustRecordReviewResultReadyRunV0(t)
	open := mustOpenPhaseCommandV0(t, "cmd-open-doc-after-result", "idem-open-doc-after-result", OrchestrationPhaseDocumentacionV0)
	run = mustApplySingleCommandEventV0(t, run, open)
	command := mustRecordReviewResultCommandV0(t, "cmd-record-result-phase", "idem-record-result-phase", "review-result-phase", ReviewResultStatusAcceptedV0)

	_, err := HandleCommandV0(run, command)
	assertRecordReviewResultCommandErrorV0(t, err, ErrTransicionInvalidaV0)
}

func TestRecordReviewResultCommandV0RejectsMissingReviewRequest(t *testing.T) {
	run := mustReviewReadyRunV0(t)
	command := mustRecordReviewResultCommandV0(t, "cmd-record-result-missing-review", "idem-record-result-missing-review", "review-result-missing-review", ReviewResultStatusAcceptedV0)

	_, err := HandleCommandV0(run, command)
	assertRecordReviewResultCommandErrorV0(t, err, ErrTransicionInvalidaV0)
}

func TestRecordReviewResultCommandV0RejectsMissingDelivery(t *testing.T) {
	run := mustReviewActiveRunWithoutDeliveryV0(t)
	run.Reviews = []string{"review-request-001"}
	command := mustRecordReviewResultCommandV0(t, "cmd-record-result-missing-delivery", "idem-record-result-missing-delivery", "review-result-missing-delivery", ReviewResultStatusAcceptedV0)

	_, err := HandleCommandV0(run, command)
	assertRecordReviewResultCommandErrorV0(t, err, ErrTransicionInvalidaV0)
}

func TestReviewResultRecordedEventV0RejectsMissingReviewRequest(t *testing.T) {
	run := mustReviewReadyRunV0(t)
	event := mustReviewResultRecordedEventV0(t, "evt-record-result-missing-review", run.LastSequence+1, validRecordReviewResultPayloadV0("review-result-missing-review", ReviewResultStatusAcceptedV0))

	_, err := ApplyEventV0(run, event)
	assertReviewResultRecordedEventErrorV0(t, err, ErrSecuenciaInvalidaV0)
}

func TestReviewResultRecordedEventV0RejectsMissingDelivery(t *testing.T) {
	run := mustReviewActiveRunWithoutDeliveryV0(t)
	run.Reviews = []string{"review-request-001"}
	event := mustReviewResultRecordedEventV0(t, "evt-record-result-missing-delivery", run.LastSequence+1, validRecordReviewResultPayloadV0("review-result-missing-delivery", ReviewResultStatusAcceptedV0))

	_, err := ApplyEventV0(run, event)
	assertReviewResultRecordedEventErrorV0(t, err, ErrSecuenciaInvalidaV0)
}

func TestRecordReviewResultCommandV0RejectsConflictingRef(t *testing.T) {
	run := mustRecordReviewResultReadyRunV0(t)
	first := mustRecordReviewResultCommandV0(t, "cmd-record-result-conflict-first", "idem-record-result-conflict-first", "review-result-conflict", ReviewResultStatusAcceptedV0)
	run = mustApplySingleCommandEventV0(t, run, first)
	conflict := mustRecordReviewResultCommandV0(t, "cmd-record-result-conflict-second", "idem-record-result-conflict-second", "review-result-conflict", ReviewResultStatusRejectedV0)

	_, err := HandleCommandV0(run, conflict)
	assertRecordReviewResultCommandErrorV0(t, err, ErrTransicionInvalidaV0)
}

func TestReviewResultRecordedEventV0RejectsConflictingRef(t *testing.T) {
	run := mustRecordReviewResultReadyRunV0(t)
	first := mustReviewResultRecordedEventV0(t, "evt-record-result-conflict-first", run.LastSequence+1, validRecordReviewResultPayloadV0("review-result-conflict", ReviewResultStatusAcceptedV0))
	run = mustApplyReducerEventV0(t, run, first)
	conflict := mustReviewResultRecordedEventV0(t, "evt-record-result-conflict-second", run.LastSequence+1, validRecordReviewResultPayloadV0("review-result-conflict", ReviewResultStatusRejectedV0))

	_, err := ApplyEventV0(run, conflict)
	assertReviewResultRecordedEventErrorV0(t, err, ErrSecuenciaInvalidaV0)
}

func TestRecordReviewResultCommandV0RejectsForbiddenDetails(t *testing.T) {
	payload := validRecordReviewResultPayloadV0("review-result-forbidden", ReviewResultStatusChangesRequestedV0)
	payload.Summary = "usar api_key=valor"

	_, err := NewRecordReviewResultCommandV0(validCommandMetaV0("cmd-record-result-forbidden", "idem-record-result-forbidden"), payload)
	assertRecordReviewResultCommandErrorV0(t, err, ErrDetalleProhibidoV0)
}

func TestReviewResultRecordedEventV0RejectsForbiddenDetails(t *testing.T) {
	payload := validRecordReviewResultPayloadV0("review-result-event-forbidden", ReviewResultStatusRejectedV0)
	payload.Summary = "usar authorization: bearer valor"

	_, err := NewReviewResultRecordedEventV0(reducerEventMetaV0("evt-record-result-forbidden", 15), payload)
	assertReviewResultRecordedEventErrorV0(t, err, ErrDetalleProhibidoV0)
}

func TestValidateOrchestrationRunV0RejectsInvalidReviewResultProjection(t *testing.T) {
	run := validRunForSerializationTestV0()
	run.ReviewResults = []string{reviewResultProjectionRefV0(validRecordReviewResultPayloadV0("review-result-invalid", ReviewResultStatusAcceptedV0))}

	issues := ValidateOrchestrationRunV0(run)
	if len(issues) == 0 {
		t.Fatal("run con review_result sin review_request_id asociado aceptado")
	}
	if issues[0].Code != OrchestrationEstadoInconsistenteV0 || issues[0].Field != "review_results" {
		t.Fatalf("issue=%+v, want estado_inconsistente review_results", issues[0])
	}
}
