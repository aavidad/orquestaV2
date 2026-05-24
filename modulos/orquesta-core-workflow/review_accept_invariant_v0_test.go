package orquestacoreworkflow

import (
	"errors"
	"testing"
)

func TestAcceptReviewCommandV0RejectsNonReviewPhase(t *testing.T) {
	run := mustAcceptReviewReadyRunV0(t)
	open := mustOpenPhaseCommandV0(t, "cmd-open-doc-after-review", "idem-open-doc-after-review", OrchestrationPhaseDocumentacionV0)
	run = mustApplySingleCommandEventV0(t, run, open)
	command := mustAcceptReviewCommandV0(t, "cmd-accept-review-phase", "idem-accept-review-phase", "accepted-review-phase")

	_, err := HandleCommandV0(run, command)
	assertAcceptReviewCommandErrorV0(t, err, ErrTransicionInvalidaV0)
}

func TestAcceptReviewCommandV0RejectsMissingReviewRequest(t *testing.T) {
	run := mustReviewReadyRunV0(t)
	command := mustAcceptReviewCommandV0(t, "cmd-accept-review-missing-review", "idem-accept-review-missing-review", "accepted-review-missing-review")

	_, err := HandleCommandV0(run, command)
	assertAcceptReviewCommandErrorV0(t, err, ErrTransicionInvalidaV0)
}

func TestAcceptReviewCommandV0RejectsMissingDelivery(t *testing.T) {
	run := mustReviewActiveRunWithoutDeliveryV0(t)
	run.Reviews = []string{"review-request-001"}
	command := mustAcceptReviewCommandV0(t, "cmd-accept-review-missing-delivery", "idem-accept-review-missing-delivery", "accepted-review-missing-delivery")

	_, err := HandleCommandV0(run, command)
	assertAcceptReviewCommandErrorV0(t, err, ErrTransicionInvalidaV0)
}

func TestAcceptReviewCommandV0RejectsMissingAcceptedReviewResult(t *testing.T) {
	run := mustRecordReviewResultReadyRunV0(t)
	command := mustAcceptReviewCommandV0(t, "cmd-accept-review-missing-result", "idem-accept-review-missing-result", "accepted-review-missing-result")

	_, err := HandleCommandV0(run, command)
	assertAcceptReviewCommandErrorV0(t, err, ErrTransicionInvalidaV0)
}

func TestAcceptReviewCommandV0RejectsChangesRequestedReviewResult(t *testing.T) {
	run := mustRecordReviewResultReadyRunV0(t)
	result := mustRecordReviewResultCommandV0(t, "cmd-record-result-changes-before-accept", "idem-record-result-changes-before-accept", "review-result-changes-before-accept", ReviewResultStatusChangesRequestedV0)
	run = mustApplySingleCommandEventV0(t, run, result)
	command := mustAcceptReviewCommandV0(t, "cmd-accept-review-changes-result", "idem-accept-review-changes-result", "accepted-review-changes-result")

	_, err := HandleCommandV0(run, command)
	assertAcceptReviewCommandErrorV0(t, err, ErrTransicionInvalidaV0)
}

func TestReviewAcceptedEventV0RejectsMissingReviewRequest(t *testing.T) {
	run := mustReviewReadyRunV0(t)
	event := mustReviewAcceptedEventV0(t, "evt-accept-review-missing-review", run.LastSequence+1, "accepted-review-missing-review")

	_, err := ApplyEventV0(run, event)
	assertReviewAcceptedEventErrorV0(t, err, ErrSecuenciaInvalidaV0)
}

func TestReviewAcceptedEventV0RejectsMissingDelivery(t *testing.T) {
	run := mustReviewActiveRunWithoutDeliveryV0(t)
	run.Reviews = []string{"review-request-001"}
	event := mustReviewAcceptedEventV0(t, "evt-accept-review-missing-delivery", run.LastSequence+1, "accepted-review-missing-delivery")

	_, err := ApplyEventV0(run, event)
	assertReviewAcceptedEventErrorV0(t, err, ErrSecuenciaInvalidaV0)
}

func TestReviewAcceptedEventV0RejectsMissingAcceptedReviewResult(t *testing.T) {
	run := mustRecordReviewResultReadyRunV0(t)
	event := mustReviewAcceptedEventV0(t, "evt-accept-review-missing-result", run.LastSequence+1, "accepted-review-missing-result")

	_, err := ApplyEventV0(run, event)
	assertReviewAcceptedEventErrorV0(t, err, ErrSecuenciaInvalidaV0)
}

func TestAcceptReviewCommandV0RejectsForbiddenDetails(t *testing.T) {
	payload := validAcceptReviewPayloadV0("accepted-review-forbidden")
	payload.Summary = "usar api_key=valor"

	_, err := NewAcceptReviewCommandV0(validCommandMetaV0("cmd-accept-review-forbidden", "idem-accept-review-forbidden"), payload)
	assertAcceptReviewCommandErrorV0(t, err, ErrDetalleProhibidoV0)
}

func TestReviewAcceptedEventV0RejectsForbiddenDetails(t *testing.T) {
	payload := reviewAcceptedPayloadFromCommandV0(validAcceptReviewPayloadV0("accepted-review-event-forbidden"))
	payload.Summary = "usar authorization: bearer valor"

	_, err := NewReviewAcceptedEventV0(reducerEventMetaV0("evt-accept-review-forbidden", 13), payload)
	assertReviewAcceptedEventErrorV0(t, err, ErrDetalleProhibidoV0)
}

func assertReviewAcceptedEventErrorV0(t *testing.T, err error, code string) {
	t.Helper()
	var publicErr OrchestrationEventErrorV0
	if !errors.As(err, &publicErr) {
		t.Fatalf("expected public event error, got %T %v", err, err)
	}
	if publicErr.Code != code {
		t.Fatalf("code=%q, want %q", publicErr.Code, code)
	}
}
