package orquestadirectorsupervisedburst

import (
	"context"
	"errors"
	"strconv"
	"testing"

	orquestadirectorcycle "orquesta/modulos/orquesta-director-cycle"
	orquestadirectorrunner "orquesta/modulos/orquesta-director-runner"
	orquestadirectorsupervisor "orquesta/modulos/orquesta-director-supervisor"
)

const burstTestRunRefV0 = "run-burst-001"

type burstRecordingBuilderV0 struct {
	err      error
	requests []DirectorSupervisedBurstStepRequestV0
}

func (builder *burstRecordingBuilderV0) BuildDirectorCycleStepInputV0(
	_ context.Context,
	request DirectorSupervisedBurstStepRequestV0,
) (orquestadirectorcycle.DirectorCycleStepInputV0, error) {
	builder.requests = append(builder.requests, request)
	if builder.err != nil {
		return orquestadirectorcycle.DirectorCycleStepInputV0{}, builder.err
	}
	step := strconv.Itoa(request.StepNumber)
	return orquestadirectorcycle.DirectorCycleStepInputV0{
		RunRef:   request.RunRef,
		CycleRef: "cycle-ref-burst-" + step,
		TickRef:  "tick-ref-burst-" + step,
	}, nil
}

type burstScriptedExecutorV0 struct {
	statuses []orquestadirectorrunner.DirectorCycleStatusV0
	errs     []error
	calls    int
}

func (executor *burstScriptedExecutorV0) ExecuteDirectorCycleStepV0(
	_ context.Context,
	input orquestadirectorcycle.DirectorCycleStepInputV0,
) (orquestadirectorcycle.DirectorCycleStepResultV0, error) {
	index := executor.calls
	executor.calls++
	status := executor.statuses[min(index, len(executor.statuses)-1)]
	result := orquestadirectorcycle.DirectorCycleStepResultV0{
		RunRef:   input.RunRef,
		CycleRef: input.CycleRef,
		TickRef:  input.TickRef,
		Status:   status,
	}
	if status == orquestadirectorrunner.DirectorCycleStatusOutboxPendingV0 {
		result.PendingOutboxAfterRefs = []string{"outbox-ref-burst-001"}
		result.OutboxPendingAfterCount = 1
	}
	if index < len(executor.errs) && executor.errs[index] != nil {
		return result, executor.errs[index]
	}
	return result, nil
}

func burstValidInputV0(
	builder *burstRecordingBuilderV0,
	executor *burstScriptedExecutorV0,
	maxSteps int,
) DirectorSupervisedBurstInputV0 {
	return DirectorSupervisedBurstInputV0{
		StepInputBuilder: builder,
		StepExecutor:     executor,
		Supervisor:       DirectDirectorSupervisorPolicyV0{},
		RunRef:           burstTestRunRefV0,
		MaxSteps:         maxSteps,
		EvidenceRefs:     []string{"evidence-ref-burst-001"},
	}
}

func assertBurstErrorV0(t *testing.T, err error, code string, field string) {
	t.Helper()
	if err == nil {
		t.Fatal("expected error")
	}
	var publicErr DirectorSupervisedBurstErrorV0
	if !errors.As(err, &publicErr) {
		t.Fatalf("unexpected error type %T %v", err, err)
	}
	if publicErr.Code != code || publicErr.Field != field {
		t.Fatalf("error=%+v, want code=%s field=%s", publicErr, code, field)
	}
}

func burstCycleStepErrorV0() orquestadirectorcycle.DirectorCycleStepErrorV0 {
	return orquestadirectorcycle.DirectorCycleStepErrorV0{
		Code:      orquestadirectorcycle.ErrDirectorCycleStepRunnerV0,
		Message:   "runner fallo",
		Field:     "runner",
		Retryable: true,
	}
}

func burstSupervisorErrorPolicyV0() DirectorSupervisorPolicyFuncV0 {
	return func(orquestadirectorsupervisor.DirectorSupervisorDecisionInputV0) (
		orquestadirectorsupervisor.DirectorSupervisorDecisionV0,
		error,
	) {
		return orquestadirectorsupervisor.DirectorSupervisorDecisionV0{}, errors.New("supervisor failed")
	}
}
