package orquestacoreworkflow

import "testing"

func TestRequestReworkCommandV0RejectsNonReviewPhase(t *testing.T) {
	run := mustRequestReworkReadyRunV0(t, "review-result-phase", ReviewResultStatusChangesRequestedV0)
	open := mustOpenPhaseCommandV0(t, "cmd-open-doc-after-rework", "idem-open-doc-after-rework", OrchestrationPhaseDocumentacionV0)
	run = mustApplySingleCommandEventV0(t, run, open)
	command := mustRequestReworkCommandV0(t, "cmd-request-rework-phase", "idem-request-rework-phase", "rework-request-phase", "review-result-phase")

	_, err := HandleCommandV0(run, command)
	assertRequestReworkCommandErrorV0(t, err, ErrTransicionInvalidaV0)
}

func TestRequestReworkCommandV0RejectsMissingReviewRequest(t *testing.T) {
	run := mustReviewReadyRunV0(t)
	command := mustRequestReworkCommandV0(t, "cmd-request-rework-missing-review", "idem-request-rework-missing-review", "rework-request-missing-review", "review-result-missing-review")

	_, err := HandleCommandV0(run, command)
	assertRequestReworkCommandErrorV0(t, err, ErrTransicionInvalidaV0)
}

func TestRequestReworkCommandV0RejectsMissingDelivery(t *testing.T) {
	run := mustReviewActiveRunWithoutDeliveryV0(t)
	run.Reviews = []string{"review-request-001"}
	command := mustRequestReworkCommandV0(t, "cmd-request-rework-missing-delivery", "idem-request-rework-missing-delivery", "rework-request-missing-delivery", "review-result-missing-delivery")

	_, err := HandleCommandV0(run, command)
	assertRequestReworkCommandErrorV0(t, err, ErrTransicionInvalidaV0)
}

func TestRequestReworkCommandV0RejectsMissingReviewResult(t *testing.T) {
	run := mustRecordReviewResultReadyRunV0(t)
	command := mustRequestReworkCommandV0(t, "cmd-request-rework-missing-result", "idem-request-rework-missing-result", "rework-request-missing-result", "review-result-missing-result")

	_, err := HandleCommandV0(run, command)
	assertRequestReworkCommandErrorV0(t, err, ErrTransicionInvalidaV0)
}

func TestRequestReworkCommandV0RejectsAcceptedReviewResult(t *testing.T) {
	run := mustRequestReworkReadyRunV0(t, "review-result-accepted-no-rework", ReviewResultStatusAcceptedV0)
	command := mustRequestReworkCommandV0(t, "cmd-request-rework-accepted-result", "idem-request-rework-accepted-result", "rework-request-accepted-result", "review-result-accepted-no-rework")

	_, err := HandleCommandV0(run, command)
	assertRequestReworkCommandErrorV0(t, err, ErrTransicionInvalidaV0)
}

func TestRequestReworkCommandV0RejectsMismatchedReviewResult(t *testing.T) {
	run := mustRequestReworkReadyRunV0(t, "review-result-mismatch", ReviewResultStatusChangesRequestedV0)
	run.Deliveries = append(run.Deliveries, "delivery-002")
	payload := validRequestReworkPayloadV0("rework-request-mismatch", "review-result-mismatch")
	payload.DeliveryRef = "delivery-002"
	command, err := NewRequestReworkCommandV0(validCommandMetaV0("cmd-request-rework-mismatch", "idem-request-rework-mismatch"), payload)
	command = mustCommandV0(t, command, err)

	_, err = HandleCommandV0(run, command)
	assertRequestReworkCommandErrorV0(t, err, ErrTransicionInvalidaV0)
}

func TestReworkRequestedEventV0RejectsMissingReviewResult(t *testing.T) {
	run := mustRecordReviewResultReadyRunV0(t)
	event := mustReworkRequestedEventV0(t, "evt-request-rework-missing-result", run.LastSequence+1, validReworkRequestedPayloadV0("rework-request-missing-result", "review-result-missing-result"))

	_, err := ApplyEventV0(run, event)
	assertReworkRequestedEventErrorV0(t, err, ErrSecuenciaInvalidaV0)
}

func TestReworkRequestedEventV0RejectsAcceptedReviewResult(t *testing.T) {
	run := mustRequestReworkReadyRunV0(t, "review-result-accepted-event-no-rework", ReviewResultStatusAcceptedV0)
	event := mustReworkRequestedEventV0(t, "evt-request-rework-accepted-result", run.LastSequence+1, validReworkRequestedPayloadV0("rework-request-accepted-result", "review-result-accepted-event-no-rework"))

	_, err := ApplyEventV0(run, event)
	assertReworkRequestedEventErrorV0(t, err, ErrSecuenciaInvalidaV0)
}

func TestRequestReworkCommandV0RejectsConflictingRef(t *testing.T) {
	run := mustRequestReworkReadyRunV0(t, "review-result-conflict-a", ReviewResultStatusChangesRequestedV0)
	result := mustRecordReviewResultCommandV0(t, "cmd-record-result-conflict-b", "idem-record-result-conflict-b", "review-result-conflict-b", ReviewResultStatusRejectedV0)
	run = mustApplySingleCommandEventV0(t, run, result)
	first := mustRequestReworkCommandV0(t, "cmd-request-rework-conflict-first", "idem-request-rework-conflict-first", "rework-request-conflict", "review-result-conflict-a")
	run = mustApplySingleCommandEventV0(t, run, first)
	conflict := mustRequestReworkCommandV0(t, "cmd-request-rework-conflict-second", "idem-request-rework-conflict-second", "rework-request-conflict", "review-result-conflict-b")

	_, err := HandleCommandV0(run, conflict)
	assertRequestReworkCommandErrorV0(t, err, ErrTransicionInvalidaV0)
}

func TestReworkRequestedEventV0RejectsConflictingRef(t *testing.T) {
	run := mustRequestReworkReadyRunV0(t, "review-result-event-conflict-a", ReviewResultStatusChangesRequestedV0)
	result := mustRecordReviewResultCommandV0(t, "cmd-record-result-event-conflict-b", "idem-record-result-event-conflict-b", "review-result-event-conflict-b", ReviewResultStatusRejectedV0)
	run = mustApplySingleCommandEventV0(t, run, result)
	first := mustReworkRequestedEventV0(t, "evt-request-rework-conflict-first", run.LastSequence+1, validReworkRequestedPayloadV0("rework-request-event-conflict", "review-result-event-conflict-a"))
	run = mustApplyReducerEventV0(t, run, first)
	conflict := mustReworkRequestedEventV0(t, "evt-request-rework-conflict-second", run.LastSequence+1, validReworkRequestedPayloadV0("rework-request-event-conflict", "review-result-event-conflict-b"))

	_, err := ApplyEventV0(run, conflict)
	assertReworkRequestedEventErrorV0(t, err, ErrSecuenciaInvalidaV0)
}

func TestRequestReworkCommandV0RejectsForbiddenDetails(t *testing.T) {
	payload := validRequestReworkPayloadV0("rework-request-forbidden", "review-result-forbidden")
	payload.Summary = "usar provider externo"

	_, err := NewRequestReworkCommandV0(validCommandMetaV0("cmd-request-rework-forbidden", "idem-request-rework-forbidden"), payload)
	assertRequestReworkCommandErrorV0(t, err, ErrDetalleProhibidoV0)
}

func TestRequestReworkCommandV0RejectsProjectionSeparators(t *testing.T) {
	payload := validRequestReworkPayloadV0("rework-request#review_result:unsafe", "review-result-unsafe")

	_, err := NewRequestReworkCommandV0(validCommandMetaV0("cmd-request-rework-unsafe", "idem-request-rework-unsafe"), payload)
	assertRequestReworkCommandErrorV0(t, err, ErrPayloadInvalidoV0)
}

func TestReworkRequestedEventV0RejectsForbiddenDetails(t *testing.T) {
	payload := validReworkRequestedPayloadV0("rework-request-event-forbidden", "review-result-event-forbidden")
	payload.Summary = "usar Claude"

	_, err := NewReworkRequestedEventV0(reducerEventMetaV0("evt-request-rework-forbidden", 18), payload)
	assertReworkRequestedEventErrorV0(t, err, ErrDetalleProhibidoV0)
}

func TestValidateOrchestrationRunV0RejectsInvalidReworkProjection(t *testing.T) {
	run := validRunForSerializationTestV0()
	result := validRecordReviewResultPayloadV0("review-result-invalid-rework", ReviewResultStatusAcceptedV0)
	run.Reviews = []string{result.ReviewRequestID}
	run.Deliveries = []string{result.DeliveryRef}
	run.ReviewResults = []string{reviewResultProjectionRefV0(result)}
	run.ReworkRequests = []string{reworkRequestProjectionRefV0(validReworkRequestedPayloadV0("rework-request-invalid", result.ReviewResultRef))}

	issues := ValidateOrchestrationRunV0(run)
	if len(issues) == 0 {
		t.Fatal("run con rework sobre resultado accepted aceptado")
	}
	if issues[0].Code != OrchestrationEstadoInconsistenteV0 || issues[0].Field != "rework_requests" {
		t.Fatalf("issue=%+v, want estado_inconsistente rework_requests", issues[0])
	}
}
