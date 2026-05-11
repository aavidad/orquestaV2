package orquestacoreworkflow

import (
	"reflect"
	"testing"
)

func TestCommandFlowFromDeliveryToCloseRunV0ReplaysToClosed(t *testing.T) {
	seedWithDelivery := mustDeliveryReplayEventsV0(t)
	seed := append([]OrchestrationEventV0(nil), seedWithDelivery[:len(seedWithDelivery)-1]...)
	run, err := ReplayDurableEventsV0(seed)
	if err != nil {
		t.Fatalf("replay seed before delivery: %v", err)
	}
	if len(run.Deliveries) != 0 {
		t.Fatalf("seed deliveries=%v, want empty before RegisterDelivery", run.Deliveries)
	}

	emitted := make([]OrchestrationEventV0, 0, 9)
	for _, step := range closeRunCommandFlowStepsV0(t) {
		var event OrchestrationEventV0
		run, event = mustApplyCommandFlowStepNoOutboxV0(t, run, step)
		emitted = append(emitted, event)
	}

	history := append(append([]OrchestrationEventV0(nil), seed...), emitted...)
	replayed, err := ReplayDurableEventsV0(history)
	if err != nil {
		t.Fatalf("replay command flow to close run: %v", err)
	}
	assertClosedCommandFlowProjectionV0(t, run)
	assertClosedCommandFlowProjectionV0(t, replayed)
}

type commandFlowStepV0 struct {
	name      string
	command   OrchestrationCommandV0
	wantEvent string
}

func closeRunCommandFlowStepsV0(t *testing.T) []commandFlowStepV0 {
	t.Helper()
	return []commandFlowStepV0{
		{"RegisterDelivery", mustRegisterDeliveryCommandV0(t, "cmd-flow-delivery", "idem-flow-delivery", "delivery-001"), OrchestrationEventDeliveryRegisteredV0},
		{"OpenPhaseRevision", mustOpenPhaseCommandV0(t, "cmd-flow-open-revision", "idem-flow-open-revision", OrchestrationPhaseRevisionV0), OrchestrationEventPhaseOpenedV0},
		{"RequestReview", mustRequestReviewCommandV0(t, "cmd-flow-review", "idem-flow-review", "review-request-001"), OrchestrationEventReviewRequestedV0},
		{"RecordReviewResult", mustRecordReviewResultCommandV0(t, "cmd-flow-record-review-result", "idem-flow-record-review-result", "review-result-flow-accepted", ReviewResultStatusAcceptedV0), OrchestrationEventReviewResultRecordedV0},
		{"AcceptReview", mustAcceptReviewCommandV0(t, "cmd-flow-accept-review", "idem-flow-accept-review", "accepted-review-001"), OrchestrationEventReviewAcceptedV0},
		{"CloseTask", mustCloseTaskCommandV0(t, "cmd-flow-close-task", "idem-flow-close-task", "task-ncw-009"), OrchestrationEventTaskClosedV0},
		{"OpenPhaseFinalValidation", mustOpenPhaseCommandV0(t, "cmd-flow-open-final-validation", "idem-flow-open-final-validation", OrchestrationPhaseValidacionFinalV0), OrchestrationEventPhaseOpenedV0},
		{"RegisterFinalValidation", mustRegisterFinalValidationCommandV0(t, "cmd-flow-final-validation", "idem-flow-final-validation", "validation-001"), OrchestrationEventFinalValidationRegisteredV0},
		{"OpenPhaseClosure", mustOpenPhaseCommandV0(t, "cmd-flow-open-closure", "idem-flow-open-closure", OrchestrationPhaseCierreV0), OrchestrationEventPhaseOpenedV0},
		{"CloseRun", mustCloseRunCommandV0(t, "cmd-flow-close-run", "idem-flow-close-run", "closure-001"), OrchestrationEventRunClosedV0},
	}
}

func mustApplyCommandFlowStepNoOutboxV0(t *testing.T, current OrchestrationRunV0, step commandFlowStepV0) (OrchestrationRunV0, OrchestrationEventV0) {
	t.Helper()
	result, err := HandleCommandV0(current, step.command)
	if err != nil {
		t.Fatalf("handle %s: %v", step.name, err)
	}
	assertSingleEventTypeV0(t, result, step.wantEvent)
	if len(result.Outbox) != 0 {
		t.Fatalf("%s outbox=%d, want empty", step.name, len(result.Outbox))
	}
	event := result.Events[0]
	if event.Sequence != current.LastSequence+1 {
		t.Fatalf("%s sequence=%d, want %d", step.name, event.Sequence, current.LastSequence+1)
	}
	next := mustApplyReducerEventV0(t, current, event)
	assertReducerRunValidV0(t, next)
	return next, event
}

func assertClosedCommandFlowProjectionV0(t *testing.T, run OrchestrationRunV0) {
	t.Helper()
	if run.Status != OrchestrationRunStatusClosedV0 {
		t.Fatalf("status=%q, want closed", run.Status)
	}
	if run.CurrentPhase != OrchestrationPhaseCierreV0 || run.LastSequence != 22 {
		t.Fatalf("run close projection mismatch: phase=%q sequence=%d", run.CurrentPhase, run.LastSequence)
	}
	if reducerPhaseByIDV0(t, run, OrchestrationPhaseCierreV0).Status != OrchestrationPhaseStatusActiveV0 {
		t.Fatalf("cierre phase not active after CloseRun")
	}
	if !reflect.DeepEqual(run.Deliveries, []string{"delivery-001"}) ||
		!reflect.DeepEqual(run.Reviews, []string{"review-request-001"}) ||
		!reflect.DeepEqual(run.AcceptedReviews, []string{"accepted-review-001"}) ||
		!reflect.DeepEqual(run.ClosedTasks, []string{"task-ncw-009"}) ||
		!reflect.DeepEqual(run.Validations, []string{"validation-001"}) ||
		!reflect.DeepEqual(run.Closures, []string{"closure-001"}) {
		t.Fatalf("unexpected close flow refs: %+v", run)
	}
}
