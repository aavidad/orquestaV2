package orquestacionnucleoapp

import (
	"context"
	"errors"
	"testing"

	orquestacoreconcurrency "orquesta/modulos/orquesta-core-concurrency"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectorcycleoutbox "orquesta/modulos/orquesta-director-cycle-outbox"
	orquestadirectorscheduler "orquesta/modulos/orquesta-director-scheduler"
	orquestadirectorsupervisor "orquesta/modulos/orquesta-director-supervisor"
)

func TestRunSupervisedBurstV0StopsAtOutbox(t *testing.T) {
	runRef := "run-nucleo-001"
	store := newMemoryRunStoreV0(mustActiveProgrammingRunV0(t, runRef))
	ledger := &memoryOutboxLedgerV0{}
	sink := &recordingEventSinkV0{}
	service := ServiceV0{
		RunStore:  store,
		EventSink: sink,
		CandidateProvider: StaticCandidateProviderV0{Candidates: SchedulerCandidateSetV0{
			WorkCandidates: []orquestadirectorscheduler.SchedulableWorkCandidateV0{
				workCandidateV0(runRef),
			},
		}},
		OutboxLedger: ledger,
	}

	result, err := service.RunSupervisedBurstV0(context.Background(), SupervisedBurstRequestV0{
		RunRef:        runRef,
		OccurredAt:    "2026-05-08T10:00:00Z",
		MaxSteps:      3,
		CorrelationID: "corr-nucleo-001",
		EvidenceRefs:  []string{"evidence-ref-nucleo-001"},
	})
	if err != nil {
		t.Fatalf("run burst: %v", err)
	}
	if result.Burst.ExecutedSteps != 1 {
		t.Fatalf("executed steps=%d", result.Burst.ExecutedSteps)
	}
	if result.Burst.FinalAction != orquestadirectorsupervisor.DirectorSupervisorActionWaitOutboxV0 {
		t.Fatalf("final action=%s", result.Burst.FinalAction)
	}
	if len(result.Run.CapacityRequests) != 1 || result.Run.CapacityRequests[0] != "capacity-ref-001" {
		t.Fatalf("capacity requests=%v", result.Run.CapacityRequests)
	}
	if len(sink.events) != 1 || sink.events[0].EventType != orquestacoreworkflow.OrchestrationEventCapacityRequestedV0 {
		t.Fatalf("events=%+v", sink.events)
	}
	pending, issues := ledger.ListPending(context.Background(), orquestadirectorcycleoutbox.DirectorCycleOutboxPendingFilterV0{RunRef: runRef})
	if len(issues) > 0 || len(pending) != 1 {
		t.Fatalf("pending=%+v issues=%+v", pending, issues)
	}
	if pending[0].MessageType != orquestacoreworkflow.OutboxMessageRequestCapacityDecisionV0 {
		t.Fatalf("message type=%s", pending[0].MessageType)
	}
}

func TestRunSupervisedBurstV0StopsQuiescentWithoutCandidates(t *testing.T) {
	runRef := "run-nucleo-quiet-001"
	service := ServiceV0{
		RunStore:          newMemoryRunStoreV0(mustActiveProgrammingRunV0(t, runRef)),
		CandidateProvider: StaticCandidateProviderV0{},
		OutboxLedger:      &memoryOutboxLedgerV0{},
	}

	result, err := service.RunSupervisedBurstV0(context.Background(), SupervisedBurstRequestV0{
		RunRef:     runRef,
		OccurredAt: "2026-05-08T10:05:00Z",
		MaxSteps:   2,
	})
	if err != nil {
		t.Fatalf("run burst: %v", err)
	}
	if result.Burst.ExecutedSteps != 1 {
		t.Fatalf("executed steps=%d", result.Burst.ExecutedSteps)
	}
	if result.Burst.FinalAction != orquestadirectorsupervisor.DirectorSupervisorActionStopQuiescentV0 {
		t.Fatalf("final action=%s", result.Burst.FinalAction)
	}
	if len(result.Run.CapacityRequests) != 0 {
		t.Fatalf("unexpected capacity requests=%v", result.Run.CapacityRequests)
	}
}

func TestRunSupervisedBurstV0RejectsMissingPorts(t *testing.T) {
	_, err := (ServiceV0{}).RunSupervisedBurstV0(context.Background(), SupervisedBurstRequestV0{})
	assertNucleoErrorV0(t, err, ErrNucleoOrquestacionInvalidoV0, "run_store")
}

func assertNucleoErrorV0(t *testing.T, err error, code string, field string) {
	t.Helper()
	if err == nil {
		t.Fatal("expected error")
	}
	var publicErr ErrorV0
	if !errors.As(err, &publicErr) {
		t.Fatalf("unexpected error type %T: %v", err, err)
	}
	if publicErr.Code != code || publicErr.Field != field {
		t.Fatalf("error=%+v, want code=%s field=%s", publicErr, code, field)
	}
}

func workCandidateV0(runRef string) orquestadirectorscheduler.SchedulableWorkCandidateV0 {
	return orquestadirectorscheduler.SchedulableWorkCandidateV0{
		CandidateRef:     "candidate-ref-001",
		SubjectClaimRefs: []string{"claim-ref-001"},
		Claims: []orquestacoreconcurrency.WorksetClaimV0{{
			SchemaVersion:  orquestacoreconcurrency.WorksetClaimSchemaVersionV0,
			ClaimRef:       "claim-ref-001",
			RunRef:         runRef,
			TaskRef:        "task-ref-001",
			AgentRequestID: "agent-ref-001",
			WriteSet:       []orquestacoreconcurrency.ScopeRefV0{{Ref: "app/main.go"}},
			EvidenceRefs:   []string{"evidence-ref-claim-001"},
		}},
		CapacityCandidate: &orquestadirectorscheduler.SchedulerCapacityCommandCandidateV0{
			CommandMeta: commandMetaV0(runRef, "cmd-capacity-001", "idem-capacity-001"),
			Payload: orquestacoreworkflow.RequestCapacityCommandPayloadV0{
				CapacityRequestID:          "capacity-ref-001",
				PhaseID:                    string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0),
				TaskRef:                    "task-ref-001",
				ReasonCode:                 "programacion_siguiente_paso",
				Summary:                    "Capacidad para ejecutar el siguiente paso.",
				MinimumRecommendedCapacity: orquestacoreworkflow.OrchestrationCapacityMediumV0,
				EvidenceRefs:               []string{"evidence-ref-capacity-001"},
			},
		},
		GateCommandMeta: commandMetaV0(runRef, "cmd-gate-001", "idem-gate-001"),
		EvidenceRefs:    []string{"evidence-ref-candidate-001"},
	}
}

func mustActiveProgrammingRunV0(t *testing.T, runRef string) orquestacoreworkflow.OrchestrationRunV0 {
	t.Helper()
	var run orquestacoreworkflow.OrchestrationRunV0
	run = mustApplyCommandV0(t, run, mustStartRunCommandV0(t, runRef))
	run = mustApplyCommandV0(t, run, mustOpenProgrammingCommandV0(t, runRef))
	return run
}

func mustApplyCommandV0(
	t *testing.T,
	run orquestacoreworkflow.OrchestrationRunV0,
	command orquestacoreworkflow.OrchestrationCommandV0,
) orquestacoreworkflow.OrchestrationRunV0 {
	t.Helper()
	result, err := orquestacoreworkflow.HandleCommandV0(run, command)
	if err != nil {
		t.Fatalf("handle command %s: %v", command.CommandType, err)
	}
	next, err := applyCommandEventsV0(run, result.Events)
	if err != nil {
		t.Fatalf("apply command %s: %v", command.CommandType, err)
	}
	return next
}

func mustStartRunCommandV0(t *testing.T, runRef string) orquestacoreworkflow.OrchestrationCommandV0 {
	t.Helper()
	command, err := orquestacoreworkflow.NewStartRunCommandV0(
		commandMetaV0(runRef, "cmd-start-001", "idem-start-001"),
		orquestacoreworkflow.StartRunCommandPayloadV0{
			ProjectRef: "project-ref-001",
			AppSpecRef: "appspec-ref-001",
		},
	)
	if err != nil {
		t.Fatalf("start command: %v", err)
	}
	return command
}

func mustOpenProgrammingCommandV0(t *testing.T, runRef string) orquestacoreworkflow.OrchestrationCommandV0 {
	t.Helper()
	command, err := orquestacoreworkflow.NewOpenPhaseCommandV0(
		commandMetaV0(runRef, "cmd-open-programacion-001", "idem-open-programacion-001"),
		orquestacoreworkflow.OpenPhaseCommandPayloadV0{
			PhaseID: string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0),
			Reason:  "Preparar programacion.",
		},
	)
	if err != nil {
		t.Fatalf("open phase command: %v", err)
	}
	return command
}

func commandMetaV0(runRef string, commandID string, idempotencyKey string) orquestacoreworkflow.OrchestrationCommandMetaV0 {
	return orquestacoreworkflow.OrchestrationCommandMetaV0{
		CommandID:      commandID,
		RunID:          runRef,
		IdempotencyKey: idempotencyKey,
		CorrelationID:  "corr-nucleo-test-001",
		RequestedBy:    "orquesta-nucleo-test",
		OccurredAt:     "2026-05-08T09:55:00Z",
	}
}
