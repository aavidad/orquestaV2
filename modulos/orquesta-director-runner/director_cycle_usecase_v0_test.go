package orquestadirectorrunner

import (
	"context"
	"errors"
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectorscheduler "orquesta/modulos/orquesta-director-scheduler"
)

func TestRunDirectorCycleV0WaitingNoAplicaWorkflow(t *testing.T) {
	calls := 0
	scheduler := DirectorSchedulerFuncV0(func(context.Context, orquestadirectorscheduler.DirectorSchedulerTickInputV0) (orquestadirectorscheduler.DirectorSchedulerTickPlanV0, error) {
		plan := runnerSchedulerPlanV0(orquestadirectorscheduler.SchedulerTickStatusWaitingV0, nil)
		plan.WaitingReasons = []orquestadirectorscheduler.SchedulerWaitingReasonV0{orquestadirectorscheduler.SchedulerWaitingOutboxPendingV0}
		return plan, nil
	})
	workflow := WorkflowCommandFuncV0(func(context.Context, orquestacoreworkflow.OrchestrationCommandV0) (orquestacoreworkflow.OrchestrationCommandResultV0, error) {
		calls++
		return orquestacoreworkflow.OrchestrationCommandResultV0{}, nil
	})

	result, err := RunDirectorCycleV0(context.Background(), runnerValidInputV0(scheduler, workflow))
	if err != nil {
		t.Fatalf("run cycle: %v", err)
	}
	if result.Status != DirectorCycleStatusWaitingV0 || result.StopReason != DirectorCycleStopSchedulerWaitingV0 {
		t.Fatalf("unexpected result: %+v", result)
	}
	if calls != 0 {
		t.Fatalf("workflow called %d times", calls)
	}
}

func TestRunDirectorCycleV0AplicaComandosYParaEnOutbox(t *testing.T) {
	commands := []orquestacoreworkflow.OrchestrationCommandV0{
		mustRunnerStartCommandV0(t, "001"),
		mustRunnerStartCommandV0(t, "002"),
		mustRunnerStartCommandV0(t, "003"),
	}
	scheduler := DirectorSchedulerFuncV0(func(context.Context, orquestadirectorscheduler.DirectorSchedulerTickInputV0) (orquestadirectorscheduler.DirectorSchedulerTickPlanV0, error) {
		return runnerSchedulerPlanV0(orquestadirectorscheduler.SchedulerTickStatusCommandsReadyV0, commands), nil
	})
	calls := 0
	workflow := WorkflowCommandFuncV0(func(context.Context, orquestacoreworkflow.OrchestrationCommandV0) (orquestacoreworkflow.OrchestrationCommandResultV0, error) {
		calls++
		if calls == 2 {
			return orquestacoreworkflow.OrchestrationCommandResultV0{Outbox: []orquestacoreworkflow.OutboxMessageV0{mustRunnerCapacityOutboxV0(t)}}, nil
		}
		return orquestacoreworkflow.OrchestrationCommandResultV0{Events: []orquestacoreworkflow.OrchestrationEventV0{{EventID: "event-ref-runner-001"}}}, nil
	})

	result, err := RunDirectorCycleV0(context.Background(), runnerValidInputV0(scheduler, workflow))
	if err != nil {
		t.Fatalf("run cycle: %v", err)
	}
	if result.Status != DirectorCycleStatusOutboxPendingV0 || len(result.Outbox) != 1 {
		t.Fatalf("unexpected result: %+v", result)
	}
	if calls != 2 || len(result.AppliedCommands) != 2 || result.EventsCount != 1 {
		t.Fatalf("unexpected counters calls=%d result=%+v", calls, result)
	}
}

func TestRunDirectorCycleV0AcumulaOutboxHastaLimiteExplicito(t *testing.T) {
	commands := []orquestacoreworkflow.OrchestrationCommandV0{
		mustRunnerStartCommandV0(t, "batch-001"),
		mustRunnerStartCommandV0(t, "batch-002"),
		mustRunnerStartCommandV0(t, "batch-003"),
	}
	scheduler := DirectorSchedulerFuncV0(func(context.Context, orquestadirectorscheduler.DirectorSchedulerTickInputV0) (orquestadirectorscheduler.DirectorSchedulerTickPlanV0, error) {
		return runnerSchedulerPlanV0(orquestadirectorscheduler.SchedulerTickStatusCommandsReadyV0, commands), nil
	})
	calls := 0
	workflow := WorkflowCommandFuncV0(func(context.Context, orquestacoreworkflow.OrchestrationCommandV0) (orquestacoreworkflow.OrchestrationCommandResultV0, error) {
		calls++
		return orquestacoreworkflow.OrchestrationCommandResultV0{
			Outbox: []orquestacoreworkflow.OutboxMessageV0{mustRunnerCapacityOutboxV0(t)},
		}, nil
	})
	input := runnerValidInputV0(scheduler, workflow)
	input.MaxOutbox = 2

	result, err := RunDirectorCycleV0(context.Background(), input)
	if err != nil {
		t.Fatalf("run cycle: %v", err)
	}
	if result.Status != DirectorCycleStatusOutboxPendingV0 || len(result.Outbox) != 2 {
		t.Fatalf("unexpected result: %+v", result)
	}
	if calls != 2 || len(result.AppliedCommands) != 2 {
		t.Fatalf("unexpected counters calls=%d result=%+v", calls, result)
	}
}

func TestRunDirectorCycleV0WorkflowErrorStops(t *testing.T) {
	commands := []orquestacoreworkflow.OrchestrationCommandV0{
		mustRunnerStartCommandV0(t, "error-001"),
		mustRunnerStartCommandV0(t, "error-002"),
	}
	scheduler := DirectorSchedulerFuncV0(func(context.Context, orquestadirectorscheduler.DirectorSchedulerTickInputV0) (orquestadirectorscheduler.DirectorSchedulerTickPlanV0, error) {
		return runnerSchedulerPlanV0(orquestadirectorscheduler.SchedulerTickStatusCommandsReadyV0, commands), nil
	})
	workflow := WorkflowCommandFuncV0(func(context.Context, orquestacoreworkflow.OrchestrationCommandV0) (orquestacoreworkflow.OrchestrationCommandResultV0, error) {
		return orquestacoreworkflow.OrchestrationCommandResultV0{}, errors.New("workflow test failure")
	})

	result, err := RunDirectorCycleV0(context.Background(), runnerValidInputV0(scheduler, workflow))
	if err == nil {
		t.Fatal("expected workflow error")
	}
	var cycleErr DirectorCycleErrorV0
	if !errors.As(err, &cycleErr) || cycleErr.Code != ErrDirectorRunnerWorkflowV0 {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.AppliedCommands) != 0 {
		t.Fatalf("commands should stop before marking applied: %+v", result.AppliedCommands)
	}
}

func TestRunDirectorCycleV0NeedsDirectorAplicaComandoDurablePrevio(t *testing.T) {
	command := mustRunnerStartCommandV0(t, "needs-director")
	scheduler := DirectorSchedulerFuncV0(func(context.Context, orquestadirectorscheduler.DirectorSchedulerTickInputV0) (orquestadirectorscheduler.DirectorSchedulerTickPlanV0, error) {
		return runnerSchedulerPlanV0(orquestadirectorscheduler.SchedulerTickStatusNeedsDirectorV0, []orquestacoreworkflow.OrchestrationCommandV0{command}), nil
	})
	calls := 0
	workflow := WorkflowCommandFuncV0(func(context.Context, orquestacoreworkflow.OrchestrationCommandV0) (orquestacoreworkflow.OrchestrationCommandResultV0, error) {
		calls++
		return orquestacoreworkflow.OrchestrationCommandResultV0{}, nil
	})

	result, err := RunDirectorCycleV0(context.Background(), runnerValidInputV0(scheduler, workflow))
	if err != nil {
		t.Fatalf("run cycle: %v", err)
	}
	if result.Status != DirectorCycleStatusNeedsDirectorV0 || calls != 1 {
		t.Fatalf("unexpected result calls=%d result=%+v", calls, result)
	}
}

func TestRunDirectorCycleV0RejectsInputIncompleto(t *testing.T) {
	_, err := RunDirectorCycleV0(context.Background(), DirectorCycleInputV0{})
	if err == nil {
		t.Fatal("expected validation error")
	}
	var cycleErr DirectorCycleErrorV0
	if !errors.As(err, &cycleErr) || cycleErr.Code != ErrDirectorRunnerCycleInvalidoV0 {
		t.Fatalf("unexpected error: %v", err)
	}
}
