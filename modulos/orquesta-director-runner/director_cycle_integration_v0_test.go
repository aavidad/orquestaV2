package orquestadirectorrunner

import (
	"context"
	"testing"

	orquestacoreconcurrency "orquesta/modulos/orquesta-core-concurrency"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectorscheduler "orquesta/modulos/orquesta-director-scheduler"
)

func TestRunDirectorCycleV0ConSchedulerRealYWorkflowEnMemoriaPideCapacidad(t *testing.T) {
	store, workflow, run := newRunnerWorkflowEventStoreProgramacionV0(t)
	input := DirectorCycleInputV0{
		Scheduler:      DirectDirectorSchedulerPortV0{},
		Workflow:       workflow,
		CycleRef:       "cycle-ref-runner-real-001",
		RunRef:         run.RunID,
		SchedulerInput: runnerCapacitySchedulerInputV0(run),
	}

	result, err := RunDirectorCycleV0(context.Background(), input)
	if err != nil {
		t.Fatalf("run cycle: %v", err)
	}
	if result.Status != DirectorCycleStatusOutboxPendingV0 {
		t.Fatalf("unexpected status: %+v", result)
	}
	if result.SchedulerStatus != orquestadirectorscheduler.SchedulerTickStatusCommandsReadyV0 {
		t.Fatalf("unexpected scheduler status: %+v", result)
	}
	if len(result.AppliedCommands) != 1 || result.AppliedCommands[0].CommandType != orquestacoreworkflow.OrchestrationCommandRequestCapacityV0 {
		t.Fatalf("unexpected commands: %+v", result.AppliedCommands)
	}
	reloaded, err := LoadWorkflowRunFromEventStoreV0(context.Background(), store, run.RunID)
	if err != nil {
		t.Fatalf("reload workflow after cycle: %v", err)
	}
	if !runnerContainsRefV0(reloaded.CapacityRequests, "capacity-ref-runner-real-001") {
		t.Fatalf("capacity request not reflected: %+v", reloaded.CapacityRequests)
	}
	if len(result.Outbox) != 1 || result.Outbox[0].MessageType != orquestacoreworkflow.OutboxMessageRequestCapacityDecisionV0 {
		t.Fatalf("unexpected outbox: %+v", result.Outbox)
	}
}

func runnerCapacitySchedulerInputV0(
	run orquestacoreworkflow.OrchestrationRunV0,
) orquestadirectorscheduler.DirectorSchedulerTickInputV0 {
	return orquestadirectorscheduler.DirectorSchedulerTickInputV0{
		TickRef:    "tick-ref-runner-real-001",
		RunRef:     run.RunID,
		OccurredAt: "2026-05-06T11:00:00Z",
		Snapshot: orquestadirectorscheduler.RunSchedulingSnapshotV0{
			RunRef:           run.RunID,
			CurrentPhaseID:   string(run.CurrentPhase),
			CapacityRequests: run.CapacityRequests,
		},
		WorkCandidates: []orquestadirectorscheduler.SchedulableWorkCandidateV0{
			runnerCapacityWorkCandidateV0(run.RunID),
		},
		EvidenceRefs: []string{"evidence-ref-runner-real-001"},
	}
}

func runnerCapacityWorkCandidateV0(runRef string) orquestadirectorscheduler.SchedulableWorkCandidateV0 {
	return orquestadirectorscheduler.SchedulableWorkCandidateV0{
		CandidateRef:     "candidate-ref-runner-real-001",
		SubjectClaimRefs: []string{"claim-ref-runner-real-001"},
		Claims: []orquestacoreconcurrency.WorksetClaimV0{
			{
				SchemaVersion:  orquestacoreconcurrency.WorksetClaimSchemaVersionV0,
				ClaimRef:       "claim-ref-runner-real-001",
				RunRef:         runRef,
				TaskRef:        "task-ref-runner-real-001",
				AgentRequestID: "agent-ref-runner-real-001",
				WriteSet:       []orquestacoreconcurrency.ScopeRefV0{{Ref: "modulos/app/main.go"}},
				EvidenceRefs:   []string{"evidence-ref-claim-runner-real-001"},
			},
		},
		CapacityCandidate: runnerCapacityCandidateV0(),
		GateCommandMeta:   runnerCommandMetaV0("cmd-runner-gate-real-001", "gate-real"),
		GateEvidenceRefs:  []string{"evidence-ref-gate-runner-real-001"},
		EvidenceRefs:      []string{"evidence-ref-candidate-runner-real-001"},
	}
}

func runnerCapacityCandidateV0() *orquestadirectorscheduler.SchedulerCapacityCommandCandidateV0 {
	return &orquestadirectorscheduler.SchedulerCapacityCommandCandidateV0{
		CommandMeta: runnerCommandMetaV0("cmd-runner-capacity-real-001", "capacity-real"),
		Payload: orquestacoreworkflow.RequestCapacityCommandPayloadV0{
			CapacityRequestID:          "capacity-ref-runner-real-001",
			PhaseID:                    string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0),
			TaskRef:                    "task-ref-runner-real-001",
			ReasonCode:                 "programacion_siguiente_paso",
			Summary:                    "Decision de nivel para siguiente tarea.",
			MinimumRecommendedCapacity: orquestacoreworkflow.OrchestrationCapacityMediumV0,
			EvidenceRefs:               []string{"evidence-ref-capacity-runner-real-001"},
		},
	}
}

func runnerContainsRefV0(values []string, ref string) bool {
	for _, value := range values {
		if value == ref {
			return true
		}
	}
	return false
}
