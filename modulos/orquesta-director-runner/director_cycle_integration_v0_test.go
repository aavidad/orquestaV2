package orquestadirectorrunner

import (
	"context"
	"testing"

	orquestacoreconcurrency "orquesta/modulos/orquesta-core-concurrency"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectorscheduler "orquesta/modulos/orquesta-director-scheduler"
)

func TestRunDirectorCycleV0ConSchedulerRealYWorkflowEnMemoriaPideCapacidad(t *testing.T) {
	workflow := newRunnerMemoryWorkflowProgramacionV0(t)
	input := DirectorCycleInputV0{
		Scheduler:      DirectDirectorSchedulerPortV0{},
		Workflow:       workflow,
		CycleRef:       "cycle-ref-runner-real-001",
		RunRef:         workflow.run.RunID,
		SchedulerInput: runnerCapacitySchedulerInputV0(workflow.run),
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
	if !runnerContainsRefV0(workflow.run.CapacityRequests, "capacity-ref-runner-real-001") {
		t.Fatalf("capacity request not reflected: %+v", workflow.run.CapacityRequests)
	}
	if len(result.Outbox) != 1 || result.Outbox[0].MessageType != orquestacoreworkflow.OutboxMessageRequestCapacityDecisionV0 {
		t.Fatalf("unexpected outbox: %+v", result.Outbox)
	}
}

type runnerMemoryWorkflowPortV0 struct {
	run   orquestacoreworkflow.OrchestrationRunV0
	calls []string
}

func newRunnerMemoryWorkflowProgramacionV0(t *testing.T) *runnerMemoryWorkflowPortV0 {
	t.Helper()
	port := &runnerMemoryWorkflowPortV0{}
	port.mustApplyCommandV0(t, mustRunnerStartCommandV0(t, "memory-start"))
	port.mustApplyCommandV0(t, mustRunnerOpenProgramacionCommandV0(t, "memory-open"))
	return port
}

func (port *runnerMemoryWorkflowPortV0) HandleWorkflowCommandV0(
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
	if err := port.applyEventsV0(result.Events); err != nil {
		return result, err
	}
	port.calls = append(port.calls, command.CommandType)
	return result, nil
}

func (port *runnerMemoryWorkflowPortV0) mustApplyCommandV0(
	t *testing.T,
	command orquestacoreworkflow.OrchestrationCommandV0,
) {
	t.Helper()
	result, err := port.HandleWorkflowCommandV0(context.Background(), command)
	if err != nil {
		t.Fatalf("apply command %s: %v", command.CommandType, err)
	}
	if len(result.Events) == 0 {
		t.Fatalf("command %s produced no events", command.CommandType)
	}
}

func (port *runnerMemoryWorkflowPortV0) applyEventsV0(events []orquestacoreworkflow.OrchestrationEventV0) error {
	var err error
	for _, event := range events {
		port.run, err = orquestacoreworkflow.ApplyEventV0(port.run, event)
		if err != nil {
			return err
		}
	}
	return nil
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
