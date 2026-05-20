package orquestadirectorrunner

import (
	"context"
	"errors"
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirector "orquesta/modulos/orquesta-director"
	orquestadirectorscheduler "orquesta/modulos/orquesta-director-scheduler"
	orquestaruntime "orquesta/modulos/orquesta-runtime"
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

func TestRunDirectorCycleV0PropagaCampoPublicoDelScheduler(t *testing.T) {
	scheduler := DirectorSchedulerFuncV0(func(context.Context, orquestadirectorscheduler.DirectorSchedulerTickInputV0) (orquestadirectorscheduler.DirectorSchedulerTickPlanV0, error) {
		return orquestadirectorscheduler.DirectorSchedulerTickPlanV0{}, orquestadirectorscheduler.DirectorSchedulerTickErrorV0{
			Code:  orquestadirectorscheduler.ErrDirectorSchedulerTickInvalidoV0,
			Field: "replan_followup_candidates.payload",
		}
	})
	workflow := WorkflowCommandFuncV0(func(context.Context, orquestacoreworkflow.OrchestrationCommandV0) (orquestacoreworkflow.OrchestrationCommandResultV0, error) {
		t.Fatal("workflow should not be called")
		return orquestacoreworkflow.OrchestrationCommandResultV0{}, nil
	})

	result, err := RunDirectorCycleV0(context.Background(), runnerValidInputV0(scheduler, workflow))
	if err == nil {
		t.Fatal("expected scheduler error")
	}
	var cycleErr DirectorCycleErrorV0
	if !errors.As(err, &cycleErr) || cycleErr.Code != ErrDirectorRunnerSchedulerV0 {
		t.Fatalf("unexpected error: %v", err)
	}
	if cycleErr.Field != "scheduler.replan_followup_candidates.payload" ||
		cycleErr.Message != "scheduler fallo: director_scheduler_tick_invalido:replan_followup_candidates.payload" {
		t.Fatalf("unexpected scheduler error details: %+v", cycleErr)
	}
	if len(result.Issues) != 1 || result.Issues[0].Field != cycleErr.Field {
		t.Fatalf("unexpected result issues: %+v", result.Issues)
	}
}

func TestRunDirectorCycleV0FiltraProgressCandidatesAjenosAntesDeScheduler(t *testing.T) {
	workflowCalls := 0
	workflow := WorkflowCommandFuncV0(func(context.Context, orquestacoreworkflow.OrchestrationCommandV0) (orquestacoreworkflow.OrchestrationCommandResultV0, error) {
		workflowCalls++
		return orquestacoreworkflow.OrchestrationCommandResultV0{}, nil
	})
	input := runnerValidInputV0(DirectDirectorSchedulerPortV0{}, workflow)
	input.SchedulerInput.ProgressSupervisionCandidates = []orquestadirectorscheduler.SchedulableProgressSupervisionCandidateV0{
		runnerProgressCandidateV0("progress-candidate-ref-foreign-001", runnerRunRefV0, "run-runner-foreign-001"),
	}

	result, err := RunDirectorCycleV0(context.Background(), input)
	if err != nil {
		t.Fatalf("run cycle: %v", err)
	}
	if result.Status != DirectorCycleStatusQuiescentV0 || workflowCalls != 0 {
		t.Fatalf("unexpected result calls=%d result=%+v", workflowCalls, result)
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

func runnerProgressCandidateV0(
	candidateRef string,
	commandRunRef string,
	reportRunRef string,
) orquestadirectorscheduler.SchedulableProgressSupervisionCandidateV0 {
	return orquestadirectorscheduler.SchedulableProgressSupervisionCandidateV0{
		CandidateRef: candidateRef,
		SupervisionInput: orquestadirector.AgentProgressSupervisionInputV0{
			CommandMeta: orquestacoreworkflow.OrchestrationCommandMetaV0{
				CommandID:      "cmd-" + candidateRef,
				RunID:          commandRunRef,
				IdempotencyKey: "idem-" + candidateRef,
				CorrelationID:  "corr-runner-001",
				RequestedBy:    "director-runner-test",
				OccurredAt:     "2026-05-06T11:00:00Z",
			},
			Report: orquestaruntime.AgentProgressReportV0{
				ReportID:         "report-ref-" + candidateRef,
				RunID:            reportRunRef,
				AgentRequestID:   "agent-ref-runner-progress-001",
				Status:           orquestaruntime.AgentLoopDetectedV0,
				Summary:          "Progreso de otro run persistido en estado durable.",
				DecisionRequired: true,
			},
			PhaseID:       string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0),
			AssessmentRef: "assessment-ref-" + candidateRef,
		},
		EvidenceRefs: []string{"evidence-ref-" + candidateRef},
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
