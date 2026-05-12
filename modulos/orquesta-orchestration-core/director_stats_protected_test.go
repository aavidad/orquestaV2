package orquestacionnucleoapp

import (
	"context"
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
)

func TestDirectorRunStatsWithObservationsV0MarksProtectedDirectorLoopCanStop(t *testing.T) {
	runRef := "run-nucleo-director-stats-protected-001"
	agentRef := "agent-ref-stats-protected-director-001"
	run := mustActiveBrainstormingRunWithStartedAgentV0(t, runRef, agentRef)
	registry := NewInMemoryAgentProcessRegistryV0()
	if err := registry.RecordAgentProcessV0(context.Background(), AgentProcessRecordV0{
		RunID:          run.RunID,
		AgentRequestID: agentRef,
		ProcessRef:     "process-ref-stats-protected-director-001",
		SessionRef:     "session-ref-stats-protected-director-001",
		LaunchRef:      "launch-ref-stats-protected-director-001",
		ReadinessRef:   "readiness-ref-stats-protected-director-001",
		EvidenceRefs:   []string{"evidence-ref-stats-protected-director-001"},
	}); err != nil {
		t.Fatalf("record process: %v", err)
	}

	stats := BuildDirectorRunStatsWithProcessRegistryV0(context.Background(), run, registry)
	ApplyDirectorProgressObservationsV0(&stats, []AgentProgressObservationV0{{
		Report:       progressLoopReportV0(runRef, agentRef),
		PhaseID:      string(orquestacoreworkflow.OrchestrationPhaseBrainstormingArquitecturaV0),
		TaskRef:      "task-ref-stats-protected-director-001",
		EvidenceRefs: []string{"evidence-ref-stats-protected-observation-001"},
	}}, nil)

	agent := findDirectorAgentStatsForTestV0(t, stats, agentRef)
	if !agent.ControlRegistered || agent.Process == nil || !agent.CanStop {
		t.Fatalf("agent protegido en loop=%+v", agent)
	}
}
