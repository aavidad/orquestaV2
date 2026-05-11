package orquestacionnucleoapp

import (
	"context"
	"errors"
	"testing"

	orquestacoreconcurrency "orquesta/modulos/orquesta-core-concurrency"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectorscheduler "orquesta/modulos/orquesta-director-scheduler"
	orquestadirectorsupervisedburst "orquesta/modulos/orquesta-director-supervised-burst"
	orquestadirectorsupervisor "orquesta/modulos/orquesta-director-supervisor"
)

func TestRunSupervisedBurstV0StopsOnWorkflowPathError(t *testing.T) {
	runRef := "run-nucleo-error-001"
	service := ServiceV0{
		RunStore: newMemoryRunStoreV0(mustActiveProgrammingRunV0(t, runRef)),
		CandidateProvider: StaticCandidateProviderV0{Candidates: SchedulerCandidateSetV0{
			WorkCandidates: []orquestadirectorscheduler.SchedulableWorkCandidateV0{invalidRunCandidateV0(runRef)},
		}},
		OutboxLedger: &memoryOutboxLedgerV0{},
	}

	result, err := service.RunSupervisedBurstV0(context.Background(), SupervisedBurstRequestV0{
		RunRef:     runRef,
		OccurredAt: "2026-05-08T10:20:00Z",
		MaxSteps:   2,
	})
	assertBurstErrorCodeV0(t, err, orquestadirectorsupervisedburst.ErrDirectorSupervisedBurstStepV0)
	if result.Burst.ExecutedSteps != 1 {
		t.Fatalf("executed steps=%d", result.Burst.ExecutedSteps)
	}
	if result.Burst.FinalAction != orquestadirectorsupervisor.DirectorSupervisorActionStopErrorV0 {
		t.Fatalf("final action=%s", result.Burst.FinalAction)
	}
}

func assertBurstErrorCodeV0(t *testing.T, err error, code string) {
	t.Helper()
	if err == nil {
		t.Fatal("expected error")
	}
	var publicErr orquestadirectorsupervisedburst.DirectorSupervisedBurstErrorV0
	if !errors.As(err, &publicErr) {
		t.Fatalf("unexpected error type %T: %v", err, err)
	}
	if publicErr.Code != code {
		t.Fatalf("error code=%s, want %s", publicErr.Code, code)
	}
}

func invalidRunCandidateV0(runRef string) orquestadirectorscheduler.SchedulableWorkCandidateV0 {
	candidate := workCandidateV0(runRef)
	candidate.CapacityCandidate = &orquestadirectorscheduler.SchedulerCapacityCommandCandidateV0{
		CommandMeta: commandMetaV0("run-nucleo-other-001", "cmd-capacity-bad-001", "idem-capacity-bad-001"),
		Payload: orquestacoreworkflow.RequestCapacityCommandPayloadV0{
			CapacityRequestID:          "capacity-ref-bad-001",
			PhaseID:                    string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0),
			TaskRef:                    "task-ref-bad-001",
			ReasonCode:                 "programacion_siguiente_paso",
			Summary:                    "Capacidad invalida por run_ref.",
			MinimumRecommendedCapacity: orquestacoreworkflow.OrchestrationCapacityMediumV0,
			EvidenceRefs:               []string{"evidence-ref-bad-001"},
		},
	}
	candidate.Claims = []orquestacoreconcurrency.WorksetClaimV0{candidate.Claims[0]}
	return candidate
}
