package orquestadirectorsupervisedburst

import (
	"context"
	"errors"
	"testing"

	orquestadirectorrunner "orquesta/modulos/orquesta-director-runner"
	orquestadirectorsupervisor "orquesta/modulos/orquesta-director-supervisor"
)

func TestRunDirectorSupervisedBurstV0ContinuaYParaEnOutbox(t *testing.T) {
	builder := &burstRecordingBuilderV0{}
	executor := &burstScriptedExecutorV0{statuses: []orquestadirectorrunner.DirectorCycleStatusV0{
		orquestadirectorrunner.DirectorCycleStatusCommandsAppliedV0,
		orquestadirectorrunner.DirectorCycleStatusOutboxPendingV0,
	}}

	result, err := RunDirectorSupervisedBurstV0(context.TODO(), burstValidInputV0(builder, executor, 3))
	if err != nil {
		t.Fatalf("run burst: %v", err)
	}
	if result.ExecutedSteps != 2 || executor.calls != 2 || len(builder.requests) != 2 {
		t.Fatalf("unexpected counters: result=%+v calls=%d builder=%d", result, executor.calls, len(builder.requests))
	}
	if result.FinalAction != orquestadirectorsupervisor.DirectorSupervisorActionWaitOutboxV0 {
		t.Fatalf("final action=%s", result.FinalAction)
	}
	if result.FinalBriefing == nil ||
		result.FinalBriefing.NextAction == nil ||
		result.FinalBriefing.NextAction.Kind != orquestadirectorsupervisor.DirectorSupervisorActionKindDispatchOutboxV0 {
		t.Fatalf("final briefing=%+v", result.FinalBriefing)
	}
	if result.Steps[0].Action != orquestadirectorsupervisor.DirectorSupervisorActionContinueV0 ||
		result.Steps[1].Action != orquestadirectorsupervisor.DirectorSupervisorActionWaitOutboxV0 {
		t.Fatalf("steps=%+v", result.Steps)
	}
	if result.Steps[0].Briefing == nil ||
		result.Steps[0].Briefing.NextAction == nil ||
		result.Steps[0].Briefing.NextAction.Kind != orquestadirectorsupervisor.DirectorSupervisorActionKindRunStepV0 ||
		result.Steps[1].Briefing == nil ||
		result.Steps[1].Briefing.NextAction == nil ||
		len(result.Steps[1].Briefing.NextAction.TargetRefs) != 1 ||
		result.Steps[1].Briefing.NextAction.TargetRefs[0] != "outbox-ref-burst-001" {
		t.Fatalf("step briefings=%+v", result.Steps)
	}
	if builder.requests[1].PreviousStepResult == nil || builder.requests[1].PreviousDecision == nil {
		t.Fatalf("builder did not receive previous step/decision: %+v", builder.requests[1])
	}
}

func TestRunDirectorSupervisedBurstV0CortaEnMaxSteps(t *testing.T) {
	builder := &burstRecordingBuilderV0{}
	executor := &burstScriptedExecutorV0{statuses: []orquestadirectorrunner.DirectorCycleStatusV0{
		orquestadirectorrunner.DirectorCycleStatusCommandsAppliedV0,
		orquestadirectorrunner.DirectorCycleStatusCommandsAppliedV0,
	}}

	result, err := RunDirectorSupervisedBurstV0(context.TODO(), burstValidInputV0(builder, executor, 2))
	if err != nil {
		t.Fatalf("run burst: %v", err)
	}
	if result.ExecutedSteps != 2 || executor.calls != 2 {
		t.Fatalf("unexpected result: %+v calls=%d", result, executor.calls)
	}
	if result.FinalAction != orquestadirectorsupervisor.DirectorSupervisorActionStopMaxStepsV0 {
		t.Fatalf("final action=%s", result.FinalAction)
	}
	if result.FinalBriefing == nil ||
		result.FinalBriefing.NextAction == nil ||
		result.FinalBriefing.NextAction.Kind != orquestadirectorsupervisor.DirectorSupervisorActionKindStopBudgetV0 {
		t.Fatalf("final briefing=%+v", result.FinalBriefing)
	}
	if result.Steps[1].ShouldRepeat ||
		result.StopProjection.PublicReason != "budget_max_steps" {
		t.Fatalf("max steps should not repeat: %+v", result.Steps[1])
	}
}

func TestRunDirectorSupervisedBurstV0PropagaErroresControlados(t *testing.T) {
	builder := &burstRecordingBuilderV0{err: errors.New("builder failed")}
	executor := &burstScriptedExecutorV0{statuses: []orquestadirectorrunner.DirectorCycleStatusV0{
		orquestadirectorrunner.DirectorCycleStatusCommandsAppliedV0,
	}}

	result, err := RunDirectorSupervisedBurstV0(context.TODO(), burstValidInputV0(builder, executor, 2))
	assertBurstErrorV0(t, err, ErrDirectorSupervisedBurstStepInputV0, "step_input_builder")
	if result.ExecutedSteps != 0 || executor.calls != 0 {
		t.Fatalf("builder error should not execute steps: %+v calls=%d", result, executor.calls)
	}
}

func TestRunDirectorSupervisedBurstV0RechazaStepInputDeOtroRun(t *testing.T) {
	builder := &burstRecordingBuilderV0{runRefOverride: "run-burst-externo"}
	executor := &burstScriptedExecutorV0{statuses: []orquestadirectorrunner.DirectorCycleStatusV0{
		orquestadirectorrunner.DirectorCycleStatusCommandsAppliedV0,
	}}

	result, err := RunDirectorSupervisedBurstV0(context.TODO(), burstValidInputV0(builder, executor, 2))

	assertBurstErrorV0(t, err, ErrDirectorSupervisedBurstStepInputV0, "step_input.run_ref")
	if result.ExecutedSteps != 0 || executor.calls != 0 {
		t.Fatalf("foreign step input should not execute: result=%+v calls=%d", result, executor.calls)
	}
}

func TestRunDirectorSupervisedBurstV0RechazaStepResultDeOtroRun(t *testing.T) {
	builder := &burstRecordingBuilderV0{}
	executor := &burstScriptedExecutorV0{
		statuses: []orquestadirectorrunner.DirectorCycleStatusV0{
			orquestadirectorrunner.DirectorCycleStatusCommandsAppliedV0,
		},
		resultRunRefOnce: "run-burst-externo",
	}

	result, err := RunDirectorSupervisedBurstV0(context.TODO(), burstValidInputV0(builder, executor, 2))

	assertBurstErrorV0(t, err, ErrDirectorSupervisedBurstStepV0, "step.run_ref")
	if result.ExecutedSteps != 0 || executor.calls != 1 {
		t.Fatalf("foreign step result should stop before recording step: result=%+v calls=%d", result, executor.calls)
	}
}

func TestRunDirectorSupervisedBurstV0StepErrorRegistraStopError(t *testing.T) {
	builder := &burstRecordingBuilderV0{}
	executor := &burstScriptedExecutorV0{
		statuses: []orquestadirectorrunner.DirectorCycleStatusV0{""},
		errs:     []error{burstCycleStepErrorV0()},
	}

	result, err := RunDirectorSupervisedBurstV0(context.TODO(), burstValidInputV0(builder, executor, 2))
	assertBurstErrorV0(t, err, ErrDirectorSupervisedBurstStepV0, "step")
	if result.ExecutedSteps != 1 || result.FinalAction != orquestadirectorsupervisor.DirectorSupervisorActionStopErrorV0 {
		t.Fatalf("unexpected result: %+v", result)
	}
	if len(result.Steps) != 1 || result.Steps[0].ErrorCode != "director_cycle_step_runner" {
		t.Fatalf("step trace=%+v", result.Steps)
	}
}

func TestRunDirectorSupervisedBurstV0SupervisorError(t *testing.T) {
	builder := &burstRecordingBuilderV0{}
	executor := &burstScriptedExecutorV0{statuses: []orquestadirectorrunner.DirectorCycleStatusV0{
		orquestadirectorrunner.DirectorCycleStatusCommandsAppliedV0,
	}}
	input := burstValidInputV0(builder, executor, 2)
	input.Supervisor = burstSupervisorErrorPolicyV0()

	result, err := RunDirectorSupervisedBurstV0(context.TODO(), input)
	assertBurstErrorV0(t, err, ErrDirectorSupervisedBurstSupervisorV0, "supervisor")
	if result.ExecutedSteps != 0 {
		t.Fatalf("supervisor error should not record step as complete: %+v", result)
	}
}

func TestRunDirectorSupervisedBurstV0RejectsInvalidInput(t *testing.T) {
	_, err := RunDirectorSupervisedBurstV0(context.TODO(), DirectorSupervisedBurstInputV0{})
	assertBurstErrorV0(t, err, ErrDirectorSupervisedBurstInvalidoV0, "step_input_builder")

	input := burstValidInputV0(&burstRecordingBuilderV0{}, &burstScriptedExecutorV0{}, 0)
	_, err = RunDirectorSupervisedBurstV0(context.TODO(), input)
	assertBurstErrorV0(t, err, ErrDirectorSupervisedBurstInvalidoV0, "max_steps")
}
