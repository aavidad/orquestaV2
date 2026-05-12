package orquestadirectortickinput

import (
	"context"
	"testing"

	orquestacoreconcurrency "orquesta/modulos/orquesta-core-concurrency"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectorrunner "orquesta/modulos/orquesta-director-runner"
	orquestadirectorscheduler "orquesta/modulos/orquesta-director-scheduler"
)

func TestBuildDirectorSchedulerTickInputV0IntegraConSchedulerYRunner(t *testing.T) {
	workflow := &tickInputMemoryWorkflowPortV0{run: tickInputProgramacionRunV0(t)}
	schedulerInput, err := BuildDirectorSchedulerTickInputV0(DirectorTickInputBuildRequestV0{
		TickRef:        "tick-ref-integration-001",
		OccurredAt:     "2026-05-06T12:05:00Z",
		Run:            workflow.run,
		WorkCandidates: []orquestadirectorscheduler.SchedulableWorkCandidateV0{tickInputWorkCandidateV0(workflow.run.RunID)},
		EvidenceRefs:   []string{"evidence-ref-tick-input-integration-001"},
	})
	if err != nil {
		t.Fatalf("build scheduler input: %v", err)
	}

	result, err := orquestadirectorrunner.RunDirectorCycleV0(context.Background(), orquestadirectorrunner.DirectorCycleInputV0{
		Scheduler:      orquestadirectorrunner.DirectDirectorSchedulerPortV0{},
		Workflow:       workflow,
		CycleRef:       "cycle-ref-tick-input-integration-001",
		RunRef:         workflow.run.RunID,
		SchedulerInput: schedulerInput,
	})
	if err != nil {
		t.Fatalf("run director cycle: %v", err)
	}
	if result.Status != orquestadirectorrunner.DirectorCycleStatusOutboxPendingV0 {
		t.Fatalf("unexpected status: %+v", result)
	}
	if len(result.AppliedCommands) != 1 ||
		result.AppliedCommands[0].CommandType != orquestacoreworkflow.OrchestrationCommandRequestCapacityV0 {
		t.Fatalf("unexpected applied commands: %+v", result.AppliedCommands)
	}
	if !tickInputContainsRefV0(workflow.run.CapacityRequests, "capacity-ref-tick-input-001") {
		t.Fatalf("capacity not reflected: %+v", workflow.run.CapacityRequests)
	}
}

func TestBuildDirectorSchedulerTickInputV0PendingOutboxHaceEsperarScheduler(t *testing.T) {
	run := tickInputProgramacionRunV0(t)
	input, err := BuildDirectorSchedulerTickInputV0(DirectorTickInputBuildRequestV0{
		TickRef:           "tick-ref-pending-outbox-001",
		OccurredAt:        "2026-05-06T12:06:00Z",
		Run:               run,
		PendingOutboxRefs: []string{"outbox-ref-pending-001"},
		WorkCandidates:    []orquestadirectorscheduler.SchedulableWorkCandidateV0{tickInputWorkCandidateV0(run.RunID)},
	})
	if err != nil {
		t.Fatalf("build scheduler input: %v", err)
	}
	if len(input.WorkCandidates) != 0 ||
		len(input.WorkClaims) != 0 ||
		len(input.DeliveryCandidates) != 0 ||
		len(input.ReviewGateCandidates) != 0 ||
		len(input.ProgressSupervisionCandidates) != 0 ||
		len(input.ReplanFollowupCandidates) != 0 {
		t.Fatalf("pending outbox debe vaciar candidatos: %+v", input)
	}
	plan, err := orquestadirectorscheduler.BuildDirectorSchedulerTickV0(input)
	if err != nil {
		t.Fatalf("scheduler: %v", err)
	}
	if plan.Status != orquestadirectorscheduler.SchedulerTickStatusWaitingV0 ||
		len(plan.WaitingReasons) != 1 ||
		plan.WaitingReasons[0] != orquestadirectorscheduler.SchedulerWaitingOutboxPendingV0 {
		t.Fatalf("unexpected plan: %+v", plan)
	}
}

func tickInputWorkCandidateV0(runRef string) orquestadirectorscheduler.SchedulableWorkCandidateV0 {
	claimRef := "claim-ref-tick-input-001"
	return orquestadirectorscheduler.SchedulableWorkCandidateV0{
		CandidateRef:      "candidate-ref-tick-input-001",
		SubjectClaimRefs:  []string{claimRef},
		Claims:            []orquestacoreconcurrency.WorksetClaimV0{tickInputClaimV0(runRef, claimRef)},
		CapacityCandidate: tickInputCapacityCandidateV0(),
		AgentCandidate:    tickInputAgentCandidateV0(claimRef),
		GateCommandMeta:   tickInputCommandMetaV0("cmd-tick-input-gate", "gate"),
		GateEvidenceRefs:  []string{"evidence-ref-tick-input-gate-001"},
		EvidenceRefs:      []string{"evidence-ref-tick-input-candidate-001"},
	}
}

func tickInputClaimV0(runRef string, claimRef string) orquestacoreconcurrency.WorksetClaimV0 {
	return orquestacoreconcurrency.WorksetClaimV0{
		SchemaVersion:  orquestacoreconcurrency.WorksetClaimSchemaVersionV0,
		ClaimRef:       claimRef,
		RunRef:         runRef,
		TaskRef:        "task-ref-tick-input-001",
		AgentRequestID: "agent-ref-tick-input-001",
		WriteSet:       []orquestacoreconcurrency.ScopeRefV0{{Ref: "modulos/app/main.go"}},
		EvidenceRefs:   []string{"evidence-ref-tick-input-claim-001"},
	}
}

func tickInputAgentCandidateV0(claimRef string) *orquestadirectorscheduler.SchedulerAgentCommandCandidateV0 {
	return &orquestadirectorscheduler.SchedulerAgentCommandCandidateV0{
		ClaimRef:    claimRef,
		CommandMeta: tickInputCommandMetaV0("cmd-tick-input-agent", "agent"),
		Payload: orquestacoreworkflow.RequestAgentCommandPayloadV0{
			AgentRequestID:     "agent-ref-tick-input-001",
			PhaseID:            string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0),
			TaskRef:            "task-ref-tick-input-001",
			CapacityRequestRef: "capacity-ref-tick-input-001",
			Role:               "implementacion",
			Summary:            "Agente para siguiente tarea.",
			EvidenceRefs:       []string{"evidence-ref-tick-input-agent-001"},
		},
	}
}

func tickInputCapacityCandidateV0() *orquestadirectorscheduler.SchedulerCapacityCommandCandidateV0 {
	return &orquestadirectorscheduler.SchedulerCapacityCommandCandidateV0{
		CommandMeta: tickInputCommandMetaV0("cmd-tick-input-capacity", "capacity"),
		Payload: orquestacoreworkflow.RequestCapacityCommandPayloadV0{
			CapacityRequestID:          "capacity-ref-tick-input-001",
			PhaseID:                    string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0),
			TaskRef:                    "task-ref-tick-input-001",
			ReasonCode:                 "programacion_siguiente_paso",
			Summary:                    "Capacidad para siguiente tarea.",
			MinimumRecommendedCapacity: orquestacoreworkflow.OrchestrationCapacityMediumV0,
			EvidenceRefs:               []string{"evidence-ref-tick-input-capacity-001"},
		},
	}
}

func tickInputContainsRefV0(values []string, ref string) bool {
	for _, value := range values {
		if value == ref {
			return true
		}
	}
	return false
}
