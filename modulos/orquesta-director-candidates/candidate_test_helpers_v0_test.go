package orquestadirectorcandidates

import (
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
)

func validCandidateInputV0() SchedulableWorkCandidateInputV0 {
	return SchedulableWorkCandidateInputV0{
		CandidateRef:     "candidate-work-001",
		RunRef:           "run-candidates-001",
		PhaseID:          string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0),
		TaskRef:          "task-candidates-001",
		SubjectClaimRefs: []string{"claim-candidates-001"},
		ScopeClaims: []WorkCandidateScopeClaimV0{{
			ClaimRef:       "claim-candidates-001",
			AgentRequestID: "agent-candidates-001",
			WriteScopes:    []string{"modulos/demo/main.go"},
			EvidenceRefs:   []string{"evidence-claim-001"},
		}},
		Commands: WorkCandidateCommandsV0{
			CapacityCommandID:      "cmd-capacity-candidates-001",
			CapacityIdempotencyKey: "idem-capacity-candidates-001",
			GateCommandID:          "cmd-gate-candidates-001",
			GateIdempotencyKey:     "idem-gate-candidates-001",
			AgentCommandID:         "cmd-agent-candidates-001",
			AgentIdempotencyKey:    "idem-agent-candidates-001",
			CorrelationID:          "corr-candidates-001",
			RequestedBy:            "director-candidates-test",
			OccurredAt:             "2026-05-06T11:00:00Z",
		},
		Capacity: WorkCandidateCapacityInputV0{
			CapacityRequestID:          "capacity-candidates-001",
			ReasonCode:                 "programacion_siguiente_paso",
			Summary:                    "Capacidad explicita para trabajo listo.",
			MinimumRecommendedCapacity: orquestacoreworkflow.OrchestrationCapacityMediumV0,
			EvidenceRefs:               []string{"evidence-capacity-001"},
		},
		Agent: WorkCandidateAgentInputV0{
			AgentRequestID: "agent-candidates-001",
			ClaimRef:       "claim-candidates-001",
			Role:           "implementacion",
			Summary:        "Agente explicito para trabajo listo.",
			EvidenceRefs:   []string{"evidence-agent-001"},
		},
		GateEvidenceRefs: []string{"evidence-gate-001"},
		EvidenceRefs:     []string{"evidence-candidate-001"},
	}
}
