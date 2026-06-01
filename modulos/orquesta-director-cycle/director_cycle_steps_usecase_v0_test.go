package orquestadirectorcycle

import (
	"context"
	"errors"
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectorrunner "orquesta/modulos/orquesta-director-runner"
	orquestadirectorscheduler "orquesta/modulos/orquesta-director-scheduler"
)

func TestExecuteDirectorCycleStepsV0ContinuaConSnapshotActualizadoHastaOutbox(t *testing.T) {
	workflow := newCycleStepWorkflowV0(t)
	ledger := newCycleStepLedgerAdapterV0()
	input := validCycleStepInputV0(workflow, ledger, "cycle-ref-steps-001", "tick-ref-steps-001")
	input.Scheduler = &cycleStepsSequenceSchedulerV0{commands: []orquestacoreworkflow.OrchestrationCommandV0{
		cycleStepsGateCommandV0(t, workflow.run.RunID, "multi-001"),
		cycleStepsCapacityCommandV0(t, workflow.run.RunID, "multi-002"),
	}}
	input.WorkCandidates = nil
	input.MaxOutbox = 1
	snapshot := &cycleStepsSnapshotFuncV0{
		load: func(_ context.Context, request DirectorCycleStepSnapshotRequestV0) (DirectorCycleStepSnapshotV0, error) {
			if request.StepNumber != 2 || request.PreviousStepResult.Status != orquestadirectorrunner.DirectorCycleStatusCommandsAppliedV0 {
				t.Fatalf("request snapshot inesperada: %+v", request)
			}
			return DirectorCycleStepSnapshotV0{
				CycleRef:     "cycle-ref-steps-002",
				TickRef:      "tick-ref-steps-002",
				OccurredAt:   "2026-05-06T13:00:01Z",
				Run:          workflow.run,
				EvidenceRefs: []string{"evidence-ref-cycle-steps-snapshot-001"},
			}, nil
		},
	}

	result, err := ExecuteDirectorCycleStepsV0(context.Background(), DirectorCycleStepsInputV0{
		InitialStep:  input,
		SnapshotPort: snapshot,
		MaxSteps:     3,
	})
	if err != nil {
		t.Fatalf("cycle steps: %v result=%+v", err, result)
	}
	if result.ExecutedSteps != 2 ||
		result.StopReason != DirectorCycleStepsStopWaitOutboxV0 ||
		result.Status != orquestadirectorrunner.DirectorCycleStatusOutboxPendingV0 {
		t.Fatalf("result=%+v", result)
	}
	if snapshot.calls != 1 {
		t.Fatalf("snapshot calls=%d", snapshot.calls)
	}
	assertCycleStepAppliedCommandTypesV0(t, result.StepResults[0], orquestacoreworkflow.OrchestrationCommandRecordConcurrencyGateV0)
	assertCycleStepAppliedCommandTypesV0(t, result.StepResults[1], orquestacoreworkflow.OrchestrationCommandRequestCapacityV0)
	if result.StepResults[1].OutboxSavedCount != 1 || result.StepResults[1].OutboxPendingAfterCount != 1 {
		t.Fatalf("outbox segundo paso=%+v", result.StepResults[1])
	}
}

func TestExecuteDirectorCycleStepsV0ParaEnMaxSteps(t *testing.T) {
	workflow := newCycleStepWorkflowV0(t)
	ledger := newCycleStepLedgerAdapterV0()
	input := validCycleStepInputV0(workflow, ledger, "cycle-ref-steps-max-001", "tick-ref-steps-max-001")
	input.Scheduler = &cycleStepsSequenceSchedulerV0{commands: []orquestacoreworkflow.OrchestrationCommandV0{
		cycleStepsGateCommandV0(t, workflow.run.RunID, "max-001"),
	}}
	input.WorkCandidates = nil

	result, err := ExecuteDirectorCycleStepsV0(context.Background(), DirectorCycleStepsInputV0{
		InitialStep: input,
		MaxSteps:    1,
	})
	if err != nil {
		t.Fatalf("cycle steps max: %v result=%+v", err, result)
	}
	if result.ExecutedSteps != 1 ||
		result.StopReason != DirectorCycleStepsStopMaxStepsV0 ||
		result.Status != orquestadirectorrunner.DirectorCycleStatusCommandsAppliedV0 {
		t.Fatalf("result=%+v", result)
	}
}

func TestExecuteDirectorCycleStepsV0RequiereSnapshotParaContinuar(t *testing.T) {
	workflow := newCycleStepWorkflowV0(t)
	ledger := newCycleStepLedgerAdapterV0()
	input := validCycleStepInputV0(workflow, ledger, "cycle-ref-steps-snapshot-001", "tick-ref-steps-snapshot-001")
	input.Scheduler = &cycleStepsSequenceSchedulerV0{commands: []orquestacoreworkflow.OrchestrationCommandV0{
		cycleStepsGateCommandV0(t, workflow.run.RunID, "snapshot-001"),
	}}
	input.WorkCandidates = nil

	result, err := ExecuteDirectorCycleStepsV0(context.Background(), DirectorCycleStepsInputV0{
		InitialStep: input,
		MaxSteps:    2,
	})
	if err == nil {
		t.Fatal("expected snapshot error")
	}
	var publicErr DirectorCycleStepsErrorV0
	if !errors.As(err, &publicErr) || publicErr.Code != ErrDirectorCycleStepsSnapshotV0 {
		t.Fatalf("unexpected error=%v", err)
	}
	if result.ExecutedSteps != 1 ||
		result.StopReason != DirectorCycleStepsStopErrorV0 ||
		result.LastErrorCode != ErrDirectorCycleStepsSnapshotV0 {
		t.Fatalf("result=%+v", result)
	}
}

func TestExecuteDirectorCycleStepsV0ParaEnEstadosTerminales(t *testing.T) {
	cases := []struct {
		name        string
		status      orquestadirectorscheduler.DirectorSchedulerTickStatusV0
		waiting     []orquestadirectorscheduler.SchedulerWaitingReasonV0
		blocked     []string
		stopReason  string
		cycleStatus orquestadirectorrunner.DirectorCycleStatusV0
	}{
		{
			name:        "waiting",
			status:      orquestadirectorscheduler.SchedulerTickStatusWaitingV0,
			waiting:     []orquestadirectorscheduler.SchedulerWaitingReasonV0{orquestadirectorscheduler.SchedulerWaitingAgentDeliveryPendingV0},
			stopReason:  DirectorCycleStepsStopWaitExternalV0,
			cycleStatus: orquestadirectorrunner.DirectorCycleStatusWaitingV0,
		},
		{
			name:        "blocked",
			status:      orquestadirectorscheduler.SchedulerTickStatusBlockedV0,
			blocked:     []string{"blocked-ref-steps-001"},
			stopReason:  DirectorCycleStepsStopBlockedV0,
			cycleStatus: orquestadirectorrunner.DirectorCycleStatusBlockedV0,
		},
		{
			name:        "needs_director",
			status:      orquestadirectorscheduler.SchedulerTickStatusNeedsDirectorV0,
			waiting:     []orquestadirectorscheduler.SchedulerWaitingReasonV0{orquestadirectorscheduler.SchedulerWaitingCandidateMissingV0},
			stopReason:  DirectorCycleStepsStopNeedsDirectorV0,
			cycleStatus: orquestadirectorrunner.DirectorCycleStatusNeedsDirectorV0,
		},
		{
			name:        "quiescent",
			status:      orquestadirectorscheduler.SchedulerTickStatusQuiescentV0,
			stopReason:  DirectorCycleStepsStopQuiescentV0,
			cycleStatus: orquestadirectorrunner.DirectorCycleStatusQuiescentV0,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			workflow := newCycleStepWorkflowV0(t)
			input := validCycleStepInputV0(workflow, newCycleStepLedgerAdapterV0(), "cycle-ref-steps-"+tc.name, "tick-ref-steps-"+tc.name)
			input.Scheduler = cycleStepsStaticSchedulerV0{status: tc.status, waiting: tc.waiting, blocked: tc.blocked}
			input.WorkCandidates = nil
			result, err := ExecuteDirectorCycleStepsV0(context.Background(), DirectorCycleStepsInputV0{
				InitialStep: input,
				MaxSteps:    3,
			})
			if err != nil {
				t.Fatalf("cycle steps terminal: %v", err)
			}
			if result.ExecutedSteps != 1 || result.StopReason != tc.stopReason || result.Status != tc.cycleStatus {
				t.Fatalf("result=%+v", result)
			}
		})
	}
}

type cycleStepsSnapshotFuncV0 struct {
	calls int
	load  func(context.Context, DirectorCycleStepSnapshotRequestV0) (DirectorCycleStepSnapshotV0, error)
}

func (source *cycleStepsSnapshotFuncV0) LoadDirectorCycleStepSnapshotV0(
	ctx context.Context,
	request DirectorCycleStepSnapshotRequestV0,
) (DirectorCycleStepSnapshotV0, error) {
	source.calls++
	return source.load(ctx, request)
}

type cycleStepsStaticSchedulerV0 struct {
	status  orquestadirectorscheduler.DirectorSchedulerTickStatusV0
	waiting []orquestadirectorscheduler.SchedulerWaitingReasonV0
	blocked []string
}

func (scheduler cycleStepsStaticSchedulerV0) BuildDirectorSchedulerTickV0(
	_ context.Context,
	input orquestadirectorscheduler.DirectorSchedulerTickInputV0,
) (orquestadirectorscheduler.DirectorSchedulerTickPlanV0, error) {
	return orquestadirectorscheduler.DirectorSchedulerTickPlanV0{
		TickRef:        input.TickRef,
		RunRef:         input.RunRef,
		Status:         scheduler.status,
		WaitingReasons: append([]orquestadirectorscheduler.SchedulerWaitingReasonV0(nil), scheduler.waiting...),
		BlockedRefs:    append([]string(nil), scheduler.blocked...),
		Summary:        "static scheduler test plan",
		EvidenceRefs:   append([]string(nil), input.EvidenceRefs...),
	}, nil
}

type cycleStepsSequenceSchedulerV0 struct {
	calls    int
	commands []orquestacoreworkflow.OrchestrationCommandV0
}

func (scheduler *cycleStepsSequenceSchedulerV0) BuildDirectorSchedulerTickV0(
	_ context.Context,
	input orquestadirectorscheduler.DirectorSchedulerTickInputV0,
) (orquestadirectorscheduler.DirectorSchedulerTickPlanV0, error) {
	if scheduler.calls >= len(scheduler.commands) {
		scheduler.calls++
		return orquestadirectorscheduler.DirectorSchedulerTickPlanV0{
			TickRef: input.TickRef,
			RunRef:  input.RunRef,
			Status:  orquestadirectorscheduler.SchedulerTickStatusQuiescentV0,
			Summary: "sequence scheduler quiescent",
		}, nil
	}
	command := scheduler.commands[scheduler.calls]
	scheduler.calls++
	return orquestadirectorscheduler.DirectorSchedulerTickPlanV0{
		TickRef:      input.TickRef,
		RunRef:       input.RunRef,
		Status:       orquestadirectorscheduler.SchedulerTickStatusCommandsReadyV0,
		Commands:     []orquestacoreworkflow.OrchestrationCommandV0{command},
		Summary:      "sequence scheduler command",
		EvidenceRefs: append([]string(nil), input.EvidenceRefs...),
	}, nil
}

func cycleStepsGateCommandV0(
	t *testing.T,
	runRef string,
	suffix string,
) orquestacoreworkflow.OrchestrationCommandV0 {
	t.Helper()
	command, err := orquestacoreworkflow.NewRecordConcurrencyGateCommandV0(
		cycleStepCommandMetaV0("cmd-cycle-steps-gate-"+suffix, "steps-gate-"+suffix),
		orquestacoreworkflow.RecordConcurrencyGateCommandPayloadV0{
			RunRef:           runRef,
			GateRef:          "gate-ref-cycle-steps-" + suffix,
			PlanRef:          "plan-ref-cycle-steps-" + suffix,
			SubjectClaimRefs: []string{"claim-ref-cycle-steps-" + suffix},
			ReadyClaimRefs:   []string{"claim-ref-cycle-steps-" + suffix},
			Decision:         orquestacoreworkflow.ConcurrencyGateDecisionAllowRequestAgentV0,
			Summary:          "Gate de ciclo multi-step.",
			EvidenceRefs:     []string{"evidence-ref-cycle-steps-gate-" + suffix},
		},
	)
	if err != nil {
		t.Fatalf("gate command: %v", err)
	}
	return command
}

func cycleStepsCapacityCommandV0(
	t *testing.T,
	runRef string,
	suffix string,
) orquestacoreworkflow.OrchestrationCommandV0 {
	t.Helper()
	command, err := orquestacoreworkflow.NewRequestCapacityCommandV0(
		cycleStepCommandMetaV0("cmd-cycle-steps-capacity-"+suffix, "steps-capacity-"+suffix),
		orquestacoreworkflow.RequestCapacityCommandPayloadV0{
			CapacityRequestID:          "capacity-ref-cycle-steps-" + suffix,
			PhaseID:                    string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0),
			TaskRef:                    "task-ref-cycle-steps-" + suffix,
			ReasonCode:                 "programacion_siguiente_paso",
			Summary:                    "Capacidad para ciclo multi-step.",
			MinimumRecommendedCapacity: orquestacoreworkflow.OrchestrationCapacityMediumV0,
			EvidenceRefs:               []string{"evidence-ref-cycle-steps-capacity-" + suffix},
		},
	)
	if err != nil {
		t.Fatalf("capacity command: %v", err)
	}
	return command
}
