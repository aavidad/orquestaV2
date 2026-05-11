package orquestacoreworkflow

import (
	"errors"
	"testing"
)

func TestRequestReviewCommandV0RejectsNonReviewPhase(t *testing.T) {
	run := mustDeliveryReadyRunV0(t)
	delivery := mustRegisterDeliveryCommandV0(t, "cmd-delivery-for-review-phase", "idem-delivery-for-review-phase", "delivery-001")
	run = mustApplySingleCommandEventV0(t, run, delivery)
	command := mustRequestReviewCommandV0(t, "cmd-review-phase", "idem-review-phase", "review-request-phase")

	_, err := HandleCommandV0(run, command)
	assertRequestReviewCommandErrorV0(t, err, ErrTransicionInvalidaV0)
}

func TestRequestReviewCommandV0RejectsMissingDelivery(t *testing.T) {
	run := mustReviewActiveRunWithoutDeliveryV0(t)
	command := mustRequestReviewCommandV0(t, "cmd-review-missing-delivery", "idem-review-missing-delivery", "review-request-missing-delivery")

	_, err := HandleCommandV0(run, command)
	assertRequestReviewCommandErrorV0(t, err, ErrTransicionInvalidaV0)
}

func TestReviewRequestedEventV0RejectsMissingDelivery(t *testing.T) {
	run := mustReviewActiveRunWithoutDeliveryV0(t)
	event := mustReviewRequestedEventV0(t, "evt-review-missing-delivery", run.LastSequence+1, "review-request-missing-delivery")

	_, err := ApplyEventV0(run, event)
	assertReviewRequestedEventErrorV0(t, err, ErrSecuenciaInvalidaV0)
}

func TestRequestReviewCommandV0RejectsForbiddenDetails(t *testing.T) {
	payload := validRequestReviewPayloadV0("review-request-forbidden")
	payload.Summary = "usar provider externo"

	_, err := NewRequestReviewCommandV0(validCommandMetaV0("cmd-review-forbidden", "idem-review-forbidden"), payload)
	assertRequestReviewCommandErrorV0(t, err, ErrDetalleProhibidoV0)
}

func TestReviewRequestedEventV0RejectsForbiddenDetails(t *testing.T) {
	payload := reviewRequestedPayloadFromCommandV0(validRequestReviewPayloadV0("review-request-event-forbidden"))
	payload.Summary = "usar Claude"

	_, err := NewReviewRequestedEventV0(reducerEventMetaV0("evt-review-forbidden", 12), payload)
	assertReviewRequestedEventErrorV0(t, err, ErrDetalleProhibidoV0)
}

func mustReviewActiveRunWithoutDeliveryV0(t *testing.T) OrchestrationRunV0 {
	t.Helper()
	run := mustHandlerStartedRunV0(t)
	open := mustOpenPhaseCommandV0(t, "cmd-open-review-without-delivery", "idem-open-review-without-delivery", OrchestrationPhaseRevisionV0)
	return mustApplySingleCommandEventV0(t, run, open)
}

func assertReviewRequestedEventErrorV0(t *testing.T, err error, code string) {
	t.Helper()
	var publicErr OrchestrationEventErrorV0
	if !errors.As(err, &publicErr) {
		t.Fatalf("expected public event error, got %T %v", err, err)
	}
	if publicErr.Code != code {
		t.Fatalf("code=%q, want %q", publicErr.Code, code)
	}
}
