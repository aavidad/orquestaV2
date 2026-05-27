package orquestastatefile

import (
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
)

func mustStoreV0(t *testing.T, rootDir string) *StoreV0 {
	t.Helper()
	store, err := NewStoreV0(ConfigV0{RootDir: rootDir})
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	return store
}

func validRunV0() orquestacoreworkflow.OrchestrationRunV0 {
	return orquestacoreworkflow.OrchestrationRunV0{
		SchemaVersion: "orchestration_run.v0",
		RunID:         "run-state-file-001",
		ProjectRef:    "project-ref-state-file-001",
		AppSpecRef:    "appspec-ref-state-file-001",
		Status:        orquestacoreworkflow.OrchestrationRunStatusActiveV0,
		CurrentPhase:  orquestacoreworkflow.OrchestrationPhaseProgramacionV0,
		Phases: []orquestacoreworkflow.OrchestrationPhaseV0{{
			ID:                  orquestacoreworkflow.OrchestrationPhaseProgramacionV0,
			Status:              orquestacoreworkflow.OrchestrationPhaseStatusActiveV0,
			RecommendedCapacity: orquestacoreworkflow.OrchestrationCapacityMediumV0,
		}},
		Tasks:        []string{"task-ref-state-file-001"},
		LastEventID:  "evt-run-started-state-file-001",
		LastSequence: 1,
	}
}

func mustRunStartedEventV0(
	t *testing.T,
	runRef string,
) orquestacoreworkflow.OrchestrationEventV0 {
	t.Helper()
	event, err := orquestacoreworkflow.NewRunStartedEventV0(
		orquestacoreworkflow.OrchestrationEventMetaV0{
			EventID:    "evt-run-started-state-file-001",
			RunID:      runRef,
			Sequence:   1,
			OccurredAt: "2026-05-12T09:00:00Z",
		},
		orquestacoreworkflow.RunStartedPayloadV0{
			ProjectRef: "project-ref-state-file-001",
			AppSpecRef: "appspec-ref-state-file-001",
		},
	)
	if err != nil {
		t.Fatalf("run started event: %v", err)
	}
	return event
}

func mustWorkflowTaskV0(t *testing.T, runRef string) orquestacoreworkflow.WorkflowTaskV0 {
	t.Helper()
	task, err := orquestacoreworkflow.NewWorkflowTaskV0(orquestacoreworkflow.WorkflowTaskV0{
		TaskID:             "task-ref-state-file-001",
		RunID:              runRef,
		PhaseID:            orquestacoreworkflow.OrchestrationPhaseProgramacionV0,
		Title:              "Implementar adaptador de ficheros",
		WriteSet:           []string{"modulos/orquesta-state-file/store_recovery_v0_test.go"},
		AcceptanceCriteria: []string{"recupera estado tras nueva instancia"},
		RequiredTests:      []string{"go test ./modulos/orquesta-state-file"},
	})
	if err != nil {
		t.Fatalf("workflow task: %v", err)
	}
	return task
}

func validAgentProcessRecordV0(runRef string) orquestacionnucleoapp.AgentProcessRecordV0 {
	return orquestacionnucleoapp.AgentProcessRecordV0{
		RunID:          runRef,
		AgentRequestID: "agent-request-state-file-001",
		ProcessRef:     "process-ref-state-file-001",
		SessionRef:     "session-ref-state-file-001",
		LaunchRef:      "launch-ref-state-file-001",
		PID:            4321,
		ReadinessRef:   "readiness-ref-state-file-001",
		EvidenceRefs:   []string{"evidence-ref-state-file-001"},
	}
}
