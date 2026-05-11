package orquestadirectorcycle

import (
	"context"
	"errors"
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectorrunner "orquesta/modulos/orquesta-director-runner"
	orquestadirectorscheduler "orquesta/modulos/orquesta-director-scheduler"
)

func TestExecuteDirectorCycleStepV0RegistraOutboxYSegundoPasoEspera(t *testing.T) {
	workflow := newCycleStepWorkflowV0(t)
	ledger := newCycleStepLedgerAdapterV0()

	first, err := ExecuteDirectorCycleStepV0(context.Background(), validCycleStepInputV0(workflow, ledger, "cycle-ref-step-001", "tick-ref-step-001"))
	if err != nil {
		t.Fatalf("first step: %v", err)
	}
	if first.Status != orquestadirectorrunner.DirectorCycleStatusOutboxPendingV0 {
		t.Fatalf("first status: %+v", first)
	}
	if first.OutboxSavedCount != 1 || first.OutboxPendingAfterCount != 1 {
		t.Fatalf("first outbox counters: %+v", first)
	}
	if !cycleStepContainsRefV0(workflow.run.CapacityRequests, "capacity-ref-cycle-step-001") {
		t.Fatalf("capacity not reflected: %+v", workflow.run.CapacityRequests)
	}

	second, err := ExecuteDirectorCycleStepV0(context.Background(), validCycleStepInputV0(workflow, ledger, "cycle-ref-step-002", "tick-ref-step-002"))
	if err != nil {
		t.Fatalf("second step: %v", err)
	}
	if second.Status != orquestadirectorrunner.DirectorCycleStatusWaitingV0 ||
		second.SchedulerStatus != orquestadirectorscheduler.SchedulerTickStatusWaitingV0 {
		t.Fatalf("second status: %+v", second)
	}
	if len(second.AppliedCommands) != 0 || second.OutboxSavedCount != 0 {
		t.Fatalf("second duplicated work: %+v", second)
	}
	if len(second.WaitingReasons) != 1 ||
		second.WaitingReasons[0] != orquestadirectorscheduler.SchedulerWaitingOutboxPendingV0 {
		t.Fatalf("second waiting reasons: %+v", second.WaitingReasons)
	}
}

func TestExecuteDirectorCycleStepV0SinCandidatesQuiescent(t *testing.T) {
	workflow := newCycleStepWorkflowV0(t)
	ledger := newCycleStepLedgerAdapterV0()
	input := validCycleStepInputV0(workflow, ledger, "cycle-ref-step-quiescent", "tick-ref-step-quiescent")
	input.WorkCandidates = nil

	result, err := ExecuteDirectorCycleStepV0(context.Background(), input)
	if err != nil {
		t.Fatalf("cycle step: %v", err)
	}
	if result.Status != orquestadirectorrunner.DirectorCycleStatusQuiescentV0 ||
		len(result.AppliedCommands) != 0 ||
		result.OutboxPendingAfterCount != 0 {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestExecuteDirectorCycleStepV0RejectsInputIncompleto(t *testing.T) {
	_, err := ExecuteDirectorCycleStepV0(context.Background(), DirectorCycleStepInputV0{})
	if err == nil {
		t.Fatal("expected validation error")
	}
	var publicErr DirectorCycleStepErrorV0
	if !errors.As(err, &publicErr) || publicErr.Code != ErrDirectorCycleStepInvalidoV0 {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestExecuteDirectorCycleStepV0PropagaErrorWorkflow(t *testing.T) {
	workflow := newCycleStepWorkflowV0(t)
	input := validCycleStepInputV0(workflow, newCycleStepLedgerAdapterV0(), "cycle-ref-step-error", "tick-ref-step-error")
	input.Workflow = failingCycleStepWorkflowV0{}

	_, err := ExecuteDirectorCycleStepV0(context.Background(), input)
	if err == nil {
		t.Fatal("expected runner error")
	}
	var publicErr DirectorCycleStepErrorV0
	if !errors.As(err, &publicErr) || publicErr.Code != ErrDirectorCycleStepRunnerV0 {
		t.Fatalf("unexpected error: %v", err)
	}
}

type failingCycleStepWorkflowV0 struct{}

func (failingCycleStepWorkflowV0) HandleWorkflowCommandV0(
	context.Context,
	orquestacoreworkflow.OrchestrationCommandV0,
) (orquestacoreworkflow.OrchestrationCommandResultV0, error) {
	return orquestacoreworkflow.OrchestrationCommandResultV0{}, errors.New("workflow failed")
}

func cycleStepContainsRefV0(values []string, ref string) bool {
	for _, value := range values {
		if value == ref {
			return true
		}
	}
	return false
}
