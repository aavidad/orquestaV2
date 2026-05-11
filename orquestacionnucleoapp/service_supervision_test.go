package orquestacionnucleoapp

import (
	"context"
	"testing"

	orquestadirectorcycle "orquesta/modulos/orquesta-director-cycle"
	orquestadirectorrunner "orquesta/modulos/orquesta-director-runner"
	orquestadirectorsupervisor "orquesta/modulos/orquesta-director-supervisor"
)

func TestRunSupervisedBurstV0StopsAtMaxSteps(t *testing.T) {
	runRef := "run-nucleo-max-001"
	executor := &commandsAppliedExecutorV0{}
	service := ServiceV0{
		RunStore:          newMemoryRunStoreV0(mustActiveProgrammingRunV0(t, runRef)),
		CandidateProvider: StaticCandidateProviderV0{},
		OutboxLedger:      &memoryOutboxLedgerV0{},
		StepExecutor:      executor,
	}

	result, err := service.RunSupervisedBurstV0(context.Background(), SupervisedBurstRequestV0{
		RunRef:     runRef,
		OccurredAt: "2026-05-08T10:15:00Z",
		MaxSteps:   2,
	})
	if err != nil {
		t.Fatalf("run burst: %v", err)
	}
	if executor.calls != 2 || result.Burst.ExecutedSteps != 2 {
		t.Fatalf("calls=%d result=%+v", executor.calls, result.Burst)
	}
	if result.Burst.FinalAction != orquestadirectorsupervisor.DirectorSupervisorActionStopMaxStepsV0 {
		t.Fatalf("final action=%s", result.Burst.FinalAction)
	}
	if result.Burst.Steps[1].ShouldRepeat {
		t.Fatalf("max_steps debe cortar repeticion: %+v", result.Burst.Steps)
	}
}

type commandsAppliedExecutorV0 struct {
	calls int
}

func (executor *commandsAppliedExecutorV0) ExecuteDirectorCycleStepV0(
	ctx context.Context,
	input orquestadirectorcycle.DirectorCycleStepInputV0,
) (orquestadirectorcycle.DirectorCycleStepResultV0, error) {
	if err := ctx.Err(); err != nil {
		return orquestadirectorcycle.DirectorCycleStepResultV0{}, err
	}
	executor.calls++
	return orquestadirectorcycle.DirectorCycleStepResultV0{
		CycleRef: input.CycleRef,
		TickRef:  input.TickRef,
		RunRef:   input.RunRef,
		Status:   orquestadirectorrunner.DirectorCycleStatusCommandsAppliedV0,
	}, nil
}
