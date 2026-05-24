package orquestacoreworkflow

import (
	"errors"
	"testing"
)

func TestCloseTaskCommandV0RejectsNonReviewPhase(t *testing.T) {
	run := mustCloseTaskReadyRunV0(t)
	open := mustOpenPhaseCommandV0(t, "cmd-open-doc-after-close-task", "idem-open-doc-after-close-task", OrchestrationPhaseDocumentacionV0)
	run = mustApplySingleCommandEventV0(t, run, open)
	command := mustCloseTaskCommandV0(t, "cmd-close-task-phase", "idem-close-task-phase", "task-ncw-009")

	_, err := HandleCommandV0(run, command)
	assertCloseTaskCommandErrorV0(t, err, ErrTransicionInvalidaV0)
}

func TestCloseTaskCommandV0RejectsMissingTask(t *testing.T) {
	run := mustCloseTaskReadyRunV0(t)
	run.Tasks = nil
	command := mustCloseTaskCommandV0(t, "cmd-close-task-missing-task", "idem-close-task-missing-task", "task-ncw-009")

	_, err := HandleCommandV0(run, command)
	assertCloseTaskCommandErrorV0(t, err, ErrTransicionInvalidaV0)
}

func TestCloseTaskCommandV0RejectsMissingDelivery(t *testing.T) {
	run := mustCloseTaskReadyRunV0(t)
	run.Deliveries = nil
	run.ReviewResults = nil
	command := mustCloseTaskCommandV0(t, "cmd-close-task-missing-delivery", "idem-close-task-missing-delivery", "task-ncw-009")

	_, err := HandleCommandV0(run, command)
	assertCloseTaskCommandErrorV0(t, err, ErrTransicionInvalidaV0)
}

func TestCloseTaskCommandV0RejectsMissingAcceptedReview(t *testing.T) {
	run := mustAcceptReviewReadyRunV0(t)
	command := mustCloseTaskCommandV0(t, "cmd-close-task-missing-review", "idem-close-task-missing-review", "task-ncw-009")

	_, err := HandleCommandV0(run, command)
	assertCloseTaskCommandErrorV0(t, err, ErrTransicionInvalidaV0)
}

func TestTaskClosedEventV0RejectsMissingTask(t *testing.T) {
	run := mustCloseTaskReadyRunV0(t)
	run.Tasks = nil
	event := mustTaskClosedEventV0(t, "evt-close-task-missing-task", run.LastSequence+1, "task-ncw-009")

	_, err := ApplyEventV0(run, event)
	assertTaskClosedEventErrorV0(t, err, ErrSecuenciaInvalidaV0)
}

func TestTaskClosedEventV0RejectsMissingDelivery(t *testing.T) {
	run := mustCloseTaskReadyRunV0(t)
	run.Deliveries = nil
	run.ReviewResults = nil
	event := mustTaskClosedEventV0(t, "evt-close-task-missing-delivery", run.LastSequence+1, "task-ncw-009")

	_, err := ApplyEventV0(run, event)
	assertTaskClosedEventErrorV0(t, err, ErrSecuenciaInvalidaV0)
}

func TestTaskClosedEventV0RejectsMissingAcceptedReview(t *testing.T) {
	run := mustAcceptReviewReadyRunV0(t)
	event := mustTaskClosedEventV0(t, "evt-close-task-missing-review", run.LastSequence+1, "task-ncw-009")

	_, err := ApplyEventV0(run, event)
	assertTaskClosedEventErrorV0(t, err, ErrSecuenciaInvalidaV0)
}

func TestCloseTaskCommandV0RejectsForbiddenDetails(t *testing.T) {
	payload := validCloseTaskPayloadV0("task-close-forbidden")
	payload.Summary = "usar api_key=valor"

	_, err := NewCloseTaskCommandV0(validCommandMetaV0("cmd-close-task-forbidden", "idem-close-task-forbidden"), payload)
	assertCloseTaskCommandErrorV0(t, err, ErrDetalleProhibidoV0)
}

func TestTaskClosedEventV0RejectsForbiddenDetails(t *testing.T) {
	payload := taskClosedPayloadFromCommandV0(validCloseTaskPayloadV0("task-close-event-forbidden"))
	payload.Summary = "usar authorization: bearer valor"

	_, err := NewTaskClosedEventV0(reducerEventMetaV0("evt-close-task-forbidden", 14), payload)
	assertTaskClosedEventErrorV0(t, err, ErrDetalleProhibidoV0)
}

func assertTaskClosedEventErrorV0(t *testing.T, err error, code string) {
	t.Helper()
	var publicErr OrchestrationEventErrorV0
	if !errors.As(err, &publicErr) {
		t.Fatalf("expected public event error, got %T %v", err, err)
	}
	if publicErr.Code != code {
		t.Fatalf("code=%q, want %q", publicErr.Code, code)
	}
}
