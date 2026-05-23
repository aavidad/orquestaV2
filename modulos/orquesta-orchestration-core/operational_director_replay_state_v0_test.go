package orquestacionnucleoapp

import (
	"context"
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectoroperativo "orquesta/modulos/orquesta-director-operativo"
)

func TestCheckOperationalDirectorReplayStateV0VerificaStoresVivos(t *testing.T) {
	run, state, task, waitState := replayStateFixturesV0(t)
	result, err := CheckOperationalDirectorReplayStateV0(context.Background(), OperationalDirectorReplayStateCheckRequestV0{
		Run:            run,
		PlanRef:        state.PlanRef,
		PlanStateStore: NewInMemoryOperationalDirectorPlanStateStoreV0(state),
		TaskStore:      NewInMemoryWorkflowTaskStoreV0(task),
		WaitStateStore: NewInMemoryWorkflowTaskWaitStateStoreV0(waitState),
	})
	if err != nil {
		t.Fatalf("CheckOperationalDirectorReplayStateV0: %v", err)
	}
	if !result.Restored || len(result.Issues) != 0 || len(result.Tasks) != 1 || len(result.WaitStates) != 1 {
		t.Fatalf("result=%+v", result)
	}
}

func TestCheckOperationalDirectorReplayStateV0NoReconstruyeMetadataFaltante(t *testing.T) {
	run, state, _, waitState := replayStateFixturesV0(t)
	result, err := CheckOperationalDirectorReplayStateV0(context.Background(), OperationalDirectorReplayStateCheckRequestV0{
		Run:            run,
		PlanRef:        state.PlanRef,
		PlanStateStore: NewInMemoryOperationalDirectorPlanStateStoreV0(state),
		TaskStore:      NewInMemoryWorkflowTaskStoreV0(),
		WaitStateStore: NewInMemoryWorkflowTaskWaitStateStoreV0(waitState),
	})
	if err != nil {
		t.Fatalf("CheckOperationalDirectorReplayStateV0: %v", err)
	}
	if result.Restored || !replayStateHasIssueFieldV0(result.Issues, "workflow_task_store") {
		t.Fatalf("issues=%+v", result.Issues)
	}
}

func TestCheckOperationalDirectorReplayStateV0DetectaMetadataVivaIncompleta(t *testing.T) {
	run, state, task, waitState := replayStateFixturesV0(t)
	task.WaveRef = ""
	task.CohortRef = ""
	result, err := CheckOperationalDirectorReplayStateV0(context.Background(), OperationalDirectorReplayStateCheckRequestV0{
		Run:            run,
		PlanRef:        state.PlanRef,
		PlanStateStore: NewInMemoryOperationalDirectorPlanStateStoreV0(state),
		TaskStore:      NewInMemoryWorkflowTaskStoreV0(task),
		WaitStateStore: NewInMemoryWorkflowTaskWaitStateStoreV0(waitState),
	})
	if err != nil {
		t.Fatalf("CheckOperationalDirectorReplayStateV0: %v", err)
	}
	if result.Restored || !replayStateHasIssueFieldV0(result.Issues, "task.wave_ref") {
		t.Fatalf("issues=%+v", result.Issues)
	}
}

func replayStateFixturesV0(
	t *testing.T,
) (
	orquestacoreworkflow.OrchestrationRunV0,
	OperationalDirectorPlanStateV0,
	orquestacoreworkflow.WorkflowTaskV0,
	WorkflowTaskWaitStateV0,
) {
	t.Helper()
	run := mustActiveProgrammingRunV0(t, "run-operational-director-replay-state-001")
	task := replayWorkflowTaskV0(t, run.RunID)
	run.Tasks = []string{task.TaskID}
	state := replayPlanStateV0(run.RunID, task.TaskID)
	waitState := replayWaitStateV0(run.RunID, task.TaskID)
	return run, state, task, waitState
}

func replayWorkflowTaskV0(
	t *testing.T,
	runRef string,
) orquestacoreworkflow.WorkflowTaskV0 {
	t.Helper()
	task, err := orquestacoreworkflow.NewWorkflowTaskV0(orquestacoreworkflow.WorkflowTaskV0{
		TaskID:          "task-ref-operational-director-replay-state-001",
		RunID:           runRef,
		PhaseID:         orquestacoreworkflow.OrchestrationPhaseProgramacionV0,
		WorkProfileKind: orquestacoreworkflow.WorkProfileImplementationV0,
		Title:           "Verificar replay de metadata viva",
		WriteSet:        []string{"modulos/orquesta-orchestration-core"},
		AcceptanceCriteria: []string{
			"WorkflowTaskStore conserva wave_ref y cohort_ref.",
			"El replay no reconstruye metadata viva desde eventos compactos.",
		},
		RequiredTests: []string{"go test -count=1 ./modulos/orquesta-orchestration-core"},
		WaveRef:       "wave-replay-state-001",
		CohortRef:     "cohort-replay-state-001",
	})
	if err != nil {
		t.Fatalf("workflow task: %v", err)
	}
	return task
}

func replayPlanStateV0(runRef string, taskRef string) OperationalDirectorPlanStateV0 {
	return OperationalDirectorPlanStateV0{
		SchemaVersion:   OperationalDirectorPlanStateSchemaVersionV0,
		StateRef:        "plan-state-ref-replay-state-001",
		PlanRef:         "plan-ref-replay-state-001",
		RequestRef:      "request-ref-replay-state-001",
		RunRef:          runRef,
		ProjectRef:      "project-ref-replay-state-001",
		Mode:            orquestadirectoroperativo.OperationalDirectorModeProgrammingV0,
		Status:          OperationalDirectorPlanStateActiveV0,
		ActiveStepID:    "step-wait-subagents",
		ActiveWaveRef:   "wave-replay-state-001",
		ActiveCohortRef: "cohort-replay-state-001",
		ObservedAt:      "2026-05-23T10:00:00Z",
		Steps: []OperationalDirectorPlanStepStateV0{
			{
				StepID:    "step-launch-subagents",
				Kind:      orquestadirectoroperativo.OperationalDirectorStepLaunchSubagentsV0,
				Status:    orquestadirectoroperativo.OperationalDirectorStepAcceptedV0,
				WaveRef:   "wave-replay-state-001",
				CohortRef: "cohort-replay-state-001",
				TaskRefs:  []string{taskRef},
				AgentRefs: []string{"agent-ref-replay-state-001"},
			},
			{
				StepID:           "step-wait-subagents",
				Kind:             orquestadirectoroperativo.OperationalDirectorStepWaitSubagentsV0,
				Status:           orquestadirectoroperativo.OperationalDirectorStepRunningV0,
				WaveRef:          "wave-replay-state-001",
				CohortRef:        "cohort-replay-state-001",
				TaskRefs:         []string{taskRef},
				WaitRefs:         []string{"wait-ref-replay-state-001"},
				AgentRefs:        []string{"agent-ref-replay-state-001"},
				PendingAgentRefs: []string{"agent-ref-replay-state-001"},
			},
		},
	}
}

func replayWaitStateV0(runRef string, taskRef string) WorkflowTaskWaitStateV0 {
	return WorkflowTaskWaitStateV0{
		SchemaVersion:    WorkflowTaskWaitStateSchemaVersionV0,
		WaitRef:          "wait-ref-replay-state-001",
		RunRef:           runRef,
		ReasonCode:       WorkflowTaskWaitReasonCohortInProgressV0,
		WaveRef:          "wave-replay-state-001",
		CohortRef:        "cohort-replay-state-001",
		TaskRefs:         []string{taskRef},
		AgentRefs:        []string{"agent-ref-replay-state-001"},
		PendingAgentRefs: []string{"agent-ref-replay-state-001"},
		Status:           WorkflowTaskWaitStateStatusWaitingV0,
		MaxExternalWaits: 2,
		ObservedAt:       "2026-05-23T10:01:00Z",
	}
}

func replayStateHasIssueFieldV0(issues []ErrorV0, field string) bool {
	for _, issue := range issues {
		if issue.Field == field {
			return true
		}
	}
	return false
}
