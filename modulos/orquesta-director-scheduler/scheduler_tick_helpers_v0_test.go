package orquestadirectorscheduler

import (
	orquestacoreconcurrency "orquesta/modulos/orquesta-core-concurrency"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
)

func validSchedulerTickInputV0() DirectorSchedulerTickInputV0 {
	return DirectorSchedulerTickInputV0{
		TickRef:    "scheduler-tick-001",
		RunRef:     "run-scheduler-001",
		OccurredAt: "2026-05-06T11:00:00Z",
		Snapshot: RunSchedulingSnapshotV0{
			RunRef:         "run-scheduler-001",
			CurrentPhaseID: string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0),
		},
		WorkCandidates: []SchedulableWorkCandidateV0{validSchedulableCandidateV0()},
		EvidenceRefs:   []string{"evidence-ref-scheduler-001"},
	}
}

func validSchedulerTickInputWithConflictV0() DirectorSchedulerTickInputV0 {
	input := validSchedulerTickInputV0()
	conflict := validWorksetClaimV0("claim-ref-scheduler-conflict-001", "agent-ref-scheduler-conflict-001", "modulos/demo/main.go")
	input.WorkCandidates[0].Claims = append(input.WorkCandidates[0].Claims, conflict)
	input.WorkCandidates[0].SubjectClaimRefs = []string{"claim-ref-scheduler-001", "claim-ref-scheduler-conflict-001"}
	return input
}

func validSchedulableCandidateV0() SchedulableWorkCandidateV0 {
	return SchedulableWorkCandidateV0{
		CandidateRef:      "schedulable-ref-001",
		SubjectClaimRefs:  []string{"claim-ref-scheduler-001"},
		Claims:            []orquestacoreconcurrency.WorksetClaimV0{validWorksetClaimV0("claim-ref-scheduler-001", "agent-ref-scheduler-001", "modulos/demo/main.go")},
		CapacityCandidate: validSchedulerCapacityCandidateV0(),
		AgentCandidate:    validSchedulerAgentCandidateV0(),
		GateCommandMeta:   schedulerMetaV0("cmd-gate-scheduler-001", "gate"),
		GateEvidenceRefs:  []string{"evidence-ref-scheduler-gate-001"},
		EvidenceRefs:      []string{"evidence-ref-schedulable-001"},
	}
}

func schedulableCandidateVariantV0(suffix string, writeRef string) SchedulableWorkCandidateV0 {
	claimRef := "claim-ref-scheduler-" + suffix
	agentRef := "agent-ref-scheduler-" + suffix
	capacityRef := "capacity-ref-scheduler-" + suffix
	taskRef := "task-ref-scheduler-" + suffix
	capacity := validSchedulerCapacityCandidateV0()
	capacity.CommandMeta = schedulerMetaV0("cmd-capacity-scheduler-"+suffix, "capacity-"+suffix)
	capacity.Payload.CapacityRequestID = capacityRef
	capacity.Payload.TaskRef = taskRef
	capacity.Payload.EvidenceRefs = []string{"evidence-ref-capacity-" + suffix}
	agent := validSchedulerAgentCandidateV0()
	agent.ClaimRef = claimRef
	agent.CommandMeta = schedulerMetaV0("cmd-agent-scheduler-"+suffix, "agent-"+suffix)
	agent.Payload.AgentRequestID = agentRef
	agent.Payload.TaskRef = taskRef
	agent.Payload.CapacityRequestRef = capacityRef
	agent.Payload.EvidenceRefs = []string{"evidence-ref-agent-" + suffix}
	return SchedulableWorkCandidateV0{
		CandidateRef:      "schedulable-ref-" + suffix,
		SubjectClaimRefs:  []string{claimRef},
		Claims:            []orquestacoreconcurrency.WorksetClaimV0{validWorksetClaimV0(claimRef, agentRef, writeRef)},
		CapacityCandidate: capacity,
		AgentCandidate:    agent,
		GateCommandMeta:   schedulerMetaV0("cmd-gate-scheduler-"+suffix, "gate-"+suffix),
		GateEvidenceRefs:  []string{"evidence-ref-scheduler-gate-" + suffix},
		EvidenceRefs:      []string{"evidence-ref-schedulable-" + suffix},
	}
}

func validWorksetClaimV0(claimRef string, agentRef string, writeRef string) orquestacoreconcurrency.WorksetClaimV0 {
	return orquestacoreconcurrency.WorksetClaimV0{
		SchemaVersion:  orquestacoreconcurrency.WorksetClaimSchemaVersionV0,
		ClaimRef:       claimRef,
		RunRef:         "run-scheduler-001",
		TaskRef:        "task-ref-scheduler-001",
		AgentRequestID: agentRef,
		WriteSet:       []orquestacoreconcurrency.ScopeRefV0{{Ref: writeRef}},
		EvidenceRefs:   []string{"evidence-ref-claim-001"},
	}
}

func validSchedulerCapacityCandidateV0() *SchedulerCapacityCommandCandidateV0 {
	return &SchedulerCapacityCommandCandidateV0{
		CommandMeta: schedulerMetaV0("cmd-capacity-scheduler-001", "capacity"),
		Payload: orquestacoreworkflow.RequestCapacityCommandPayloadV0{
			CapacityRequestID:          "capacity-ref-scheduler-001",
			PhaseID:                    string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0),
			TaskRef:                    "task-ref-scheduler-001",
			ReasonCode:                 "programacion_siguiente_paso",
			Summary:                    "Capacidad explicita para microtarea lista.",
			MinimumRecommendedCapacity: orquestacoreworkflow.OrchestrationCapacityMediumV0,
			EvidenceRefs:               []string{"evidence-ref-capacity-001"},
		},
	}
}

func validSchedulerAgentCandidateV0() *SchedulerAgentCommandCandidateV0 {
	return &SchedulerAgentCommandCandidateV0{
		ClaimRef:    "claim-ref-scheduler-001",
		CommandMeta: schedulerMetaV0("cmd-agent-scheduler-001", "agent"),
		Payload: orquestacoreworkflow.RequestAgentCommandPayloadV0{
			AgentRequestID:     "agent-ref-scheduler-001",
			PhaseID:            string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0),
			TaskRef:            "task-ref-scheduler-001",
			CapacityRequestRef: "capacity-ref-scheduler-001",
			Role:               "implementacion",
			Summary:            "Agente explicito para microtarea lista.",
			EvidenceRefs:       []string{"evidence-ref-agent-001"},
		},
	}
}

func schedulerMetaV0(commandID string, suffix string) orquestacoreworkflow.OrchestrationCommandMetaV0 {
	return orquestacoreworkflow.OrchestrationCommandMetaV0{
		CommandID:      commandID,
		RunID:          "run-scheduler-001",
		IdempotencyKey: "idem-" + suffix,
		CorrelationID:  "corr-scheduler-001",
		RequestedBy:    "director-scheduler-test",
		OccurredAt:     "2026-05-06T11:00:00Z",
	}
}
