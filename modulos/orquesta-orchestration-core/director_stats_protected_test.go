package orquestacionnucleoapp

import (
	"context"
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestaruntime "orquesta/modulos/orquesta-runtime"
)

func TestDirectorRunStatsWithObservationsV0MarksProtectedDirectorLoopCanStop(t *testing.T) {
	runRef := "run-nucleo-director-stats-protected-001"
	agentRef := "agent-spec-stats-protected-director"
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

func TestDirectorRunStatsWithObservationsV0DoesNotProtectSpecializedDirectorAgent(t *testing.T) {
	runRef := "run-nucleo-director-stats-specialized-001"
	agentRef := "agent-spec-agenda-api-web-req-agenda-api-web-001-web"
	run := mustActiveBrainstormingRunWithStartedAgentV0(t, runRef, agentRef)
	registry := NewInMemoryAgentProcessRegistryV0()
	if err := registry.RecordAgentProcessV0(context.Background(), AgentProcessRecordV0{
		RunID:          run.RunID,
		AgentRequestID: agentRef,
		ProcessRef:     "process-ref-stats-specialized-web-001",
		SessionRef:     "session-ref-stats-specialized-web-001",
		LaunchRef:      "launch-ref-stats-specialized-web-001",
		ReadinessRef:   "readiness-ref-stats-specialized-web-001",
		EvidenceRefs:   []string{"evidence-ref-stats-specialized-web-001"},
	}); err != nil {
		t.Fatalf("record process: %v", err)
	}

	stats := BuildDirectorRunStatsWithProcessRegistryV0(context.Background(), run, registry)
	ApplyDirectorProgressObservationsV0(&stats, []AgentProgressObservationV0{{
		Report: directorBudgetProgressReportForTestV0(
			runRef,
			agentRef,
			orquestaruntime.AgentProgressBudgetOverBudgetNoActivityV0,
		),
		PhaseID:      string(orquestacoreworkflow.OrchestrationPhaseBrainstormingArquitecturaV0),
		TaskRef:      "task-ref-director-web-001",
		EvidenceRefs: []string{"evidence-ref-stats-specialized-web-observation-001"},
	}}, nil)

	agent := findDirectorAgentStatsForTestV0(t, stats, agentRef)
	if !agent.ControlRegistered || agent.Process == nil || !agent.CanStop {
		t.Fatalf("agente especializado no parable: %+v", agent)
	}
}
