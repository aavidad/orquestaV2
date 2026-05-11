package orquestadirectorcycleoutbox

import (
	"context"
	"testing"

	orquestacoreconcurrency "orquesta/modulos/orquesta-core-concurrency"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectorrunner "orquesta/modulos/orquesta-director-runner"
	orquestadirectorscheduler "orquesta/modulos/orquesta-director-scheduler"
	orquestadirectortickinput "orquesta/modulos/orquesta-director-tick-input"
)

func TestRecordDirectorCycleOutboxV0BloqueaSiguienteTick(t *testing.T) {
	workflow := newCycleOutboxWorkflowV0(t)
	schedulerInput := cycleOutboxBuildTickInputV0(t, workflow.run, nil)
	cycleResult, err := orquestadirectorrunner.RunDirectorCycleV0(context.Background(), orquestadirectorrunner.DirectorCycleInputV0{
		Scheduler:      orquestadirectorrunner.DirectDirectorSchedulerPortV0{},
		Workflow:       workflow,
		CycleRef:       "cycle-ref-outbox-integration-001",
		RunRef:         workflow.run.RunID,
		SchedulerInput: schedulerInput,
	})
	if err != nil {
		t.Fatalf("run director cycle: %v", err)
	}
	if cycleResult.Status != orquestadirectorrunner.DirectorCycleStatusOutboxPendingV0 || len(cycleResult.Outbox) != 1 {
		t.Fatalf("unexpected cycle result: %+v", cycleResult)
	}

	record, err := RecordDirectorCycleOutboxV0(context.Background(), DirectorCycleOutboxRecordInputV0{
		Ledger:   newCycleOutboxPersistenceLedgerAdapterV0(),
		RunRef:   workflow.run.RunID,
		Messages: cycleResult.Outbox,
	})
	if err != nil {
		t.Fatalf("record outbox: %v", err)
	}
	nextInput := cycleOutboxBuildTickInputV0(t, workflow.run, record.PendingOutboxRefs)
	nextPlan, err := orquestadirectorscheduler.BuildDirectorSchedulerTickV0(nextInput)
	if err != nil {
		t.Fatalf("next scheduler tick: %v", err)
	}
	if nextPlan.Status != orquestadirectorscheduler.SchedulerTickStatusWaitingV0 ||
		len(nextPlan.WaitingReasons) != 1 ||
		nextPlan.WaitingReasons[0] != orquestadirectorscheduler.SchedulerWaitingOutboxPendingV0 {
		t.Fatalf("unexpected next plan: %+v", nextPlan)
	}
}

type cycleOutboxWorkflowPortV0 struct {
	run orquestacoreworkflow.OrchestrationRunV0
}

func newCycleOutboxWorkflowV0(t *testing.T) *cycleOutboxWorkflowPortV0 {
	t.Helper()
	port := &cycleOutboxWorkflowPortV0{}
	port.mustApplyCommandV0(t, cycleOutboxStartCommandV0(t))
	port.mustApplyCommandV0(t, cycleOutboxOpenProgramacionCommandV0(t))
	return port
}

func (port *cycleOutboxWorkflowPortV0) HandleWorkflowCommandV0(
	ctx context.Context,
	command orquestacoreworkflow.OrchestrationCommandV0,
) (orquestacoreworkflow.OrchestrationCommandResultV0, error) {
	if err := ctx.Err(); err != nil {
		return orquestacoreworkflow.OrchestrationCommandResultV0{}, err
	}
	result, err := orquestacoreworkflow.HandleCommandV0(port.run, command)
	if err != nil {
		return result, err
	}
	for _, event := range result.Events {
		port.run, err = orquestacoreworkflow.ApplyEventV0(port.run, event)
		if err != nil {
			return result, err
		}
	}
	return result, nil
}

func (port *cycleOutboxWorkflowPortV0) mustApplyCommandV0(
	t *testing.T,
	command orquestacoreworkflow.OrchestrationCommandV0,
) {
	t.Helper()
	if _, err := port.HandleWorkflowCommandV0(context.Background(), command); err != nil {
		t.Fatalf("apply command %s: %v", command.CommandType, err)
	}
}

func cycleOutboxBuildTickInputV0(
	t *testing.T,
	run orquestacoreworkflow.OrchestrationRunV0,
	pendingOutboxRefs []string,
) orquestadirectorscheduler.DirectorSchedulerTickInputV0 {
	t.Helper()
	input, err := orquestadirectortickinput.BuildDirectorSchedulerTickInputV0(orquestadirectortickinput.DirectorTickInputBuildRequestV0{
		TickRef:           "tick-ref-cycle-outbox-001",
		OccurredAt:        "2026-05-06T12:30:00Z",
		Run:               run,
		PendingOutboxRefs: pendingOutboxRefs,
		WorkCandidates:    []orquestadirectorscheduler.SchedulableWorkCandidateV0{cycleOutboxWorkCandidateV0(run.RunID)},
		EvidenceRefs:      []string{"evidence-ref-cycle-outbox-integration-001"},
	})
	if err != nil {
		t.Fatalf("build tick input: %v", err)
	}
	return input
}

func cycleOutboxWorkCandidateV0(runRef string) orquestadirectorscheduler.SchedulableWorkCandidateV0 {
	claimRef := "claim-ref-cycle-outbox-001"
	return orquestadirectorscheduler.SchedulableWorkCandidateV0{
		CandidateRef:      "candidate-ref-cycle-outbox-001",
		SubjectClaimRefs:  []string{claimRef},
		Claims:            []orquestacoreconcurrency.WorksetClaimV0{cycleOutboxClaimV0(runRef, claimRef)},
		CapacityCandidate: cycleOutboxCapacityCandidateV0(),
		GateCommandMeta:   cycleOutboxCommandMetaV0("cmd-cycle-outbox-gate", "gate"),
		EvidenceRefs:      []string{"evidence-ref-cycle-outbox-candidate-001"},
	}
}

func cycleOutboxClaimV0(runRef string, claimRef string) orquestacoreconcurrency.WorksetClaimV0 {
	return orquestacoreconcurrency.WorksetClaimV0{
		SchemaVersion:  orquestacoreconcurrency.WorksetClaimSchemaVersionV0,
		ClaimRef:       claimRef,
		RunRef:         runRef,
		TaskRef:        "task-ref-cycle-outbox-001",
		AgentRequestID: "agent-ref-cycle-outbox-001",
		WriteSet:       []orquestacoreconcurrency.ScopeRefV0{{Ref: "modulos/app/main.go"}},
		EvidenceRefs:   []string{"evidence-ref-cycle-outbox-claim-001"},
	}
}

func cycleOutboxCapacityCandidateV0() *orquestadirectorscheduler.SchedulerCapacityCommandCandidateV0 {
	return &orquestadirectorscheduler.SchedulerCapacityCommandCandidateV0{
		CommandMeta: cycleOutboxCommandMetaV0("cmd-cycle-outbox-capacity", "capacity"),
		Payload: orquestacoreworkflow.RequestCapacityCommandPayloadV0{
			CapacityRequestID:          "capacity-ref-cycle-outbox-001",
			PhaseID:                    string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0),
			TaskRef:                    "task-ref-cycle-outbox-001",
			ReasonCode:                 "programacion_siguiente_paso",
			Summary:                    "Capacidad para siguiente tarea.",
			MinimumRecommendedCapacity: orquestacoreworkflow.OrchestrationCapacityMediumV0,
			EvidenceRefs:               []string{"evidence-ref-cycle-outbox-capacity-001"},
		},
	}
}
