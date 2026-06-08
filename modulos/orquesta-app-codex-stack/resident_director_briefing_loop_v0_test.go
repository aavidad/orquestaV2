package orquestaappcodexstack

import (
	"context"
	"testing"
	"time"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectorsupervisedburst "orquesta/modulos/orquesta-director-supervised-burst"
	orquestadirectorsupervisor "orquesta/modulos/orquesta-director-supervisor"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	orquestarunqueue "orquesta/modulos/orquesta-run-queue"
)

func TestCodexStackResidentDirectorV0IdleSinCandidatosV0(t *testing.T) {
	stack := mustBuildCodexStackForTestV0(t, newFakeCodexStackRuntimeV0())

	result, err := stack.RunCodexStackResidentDirectorV0(context.Background(), CodexStackResidentDirectorCommandV0{
		OccurredAt:    "2026-06-08T10:00:00Z",
		CorrelationID: "corr-resident-idle",
		MaxActions:    2,
		EvidenceRefs:  []string{"evidence-ref-resident-idle"},
	})
	if err != nil {
		t.Fatalf("RunCodexStackResidentDirectorV0: %v", err)
	}
	if result.Status != CodexStackResidentDirectorStatusIdleV0 ||
		result.RunRef != "" ||
		result.ExecutedActions != 0 {
		t.Fatalf("result=%+v", result)
	}
}

func TestCodexStackResidentDirectorV0EjecutaRunEnColaConBriefingLoopV0(t *testing.T) {
	ctx := context.Background()
	runtime := newFakeCodexStackRuntimeV0()
	stack := mustBuildCodexStackForTestV0(t, runtime)
	runRef := "run-resident-codex-stack-001"
	taskRef := "task-resident-codex-stack-001"
	now := time.Date(2026, 6, 8, 10, 0, 0, 0, time.UTC)
	seedCodexStackResidentDirectorRunV0(t, ctx, stack, runRef, taskRef)
	if _, err := stack.Stores.RunQueue.SetRunPriorityV0(ctx, orquestarunqueue.RunQueuePriorityCommandV0{
		RunRef:        runRef,
		QueueRef:      normalizeRunQueueConfigV0(stack.RunQueue).QueueRef,
		AppRef:        "app-resident-codex-stack",
		Status:        orquestarunqueue.RunStatusReadyV0,
		PriorityScore: 80,
		UpdatedAt:     now,
		RequestedBy:   "resident-director-test",
		Reason:        "resident-director-test",
		EvidenceRefs:  []string{"evidence-ref-resident-candidate"},
	}); err != nil {
		t.Fatalf("SetRunPriorityV0: %v", err)
	}

	result, err := stack.RunCodexStackResidentDirectorV0(ctx, CodexStackResidentDirectorCommandV0{
		OccurredAt:            "2026-06-08T10:00:00Z",
		CorrelationID:         "corr-resident-codex-stack",
		MaxRunsPerTick:        1,
		MaxExecutions:         1,
		MaxActions:            8,
		MaxDispatchesPerWait:  4,
		MaxCommands:           6,
		MaxOutboxPerCycle:     6,
		QueueRankingPolicyNow: now,
		EvidenceRefs:          []string{"evidence-ref-resident-command"},
	})
	if err != nil {
		t.Fatalf("RunCodexStackResidentDirectorV0: %v result=%+v", err, result)
	}
	if result.RunRef != runRef ||
		result.ExecutedActions == 0 ||
		result.Status == CodexStackResidentDirectorStatusIdleV0 ||
		result.Status == orquestacionnucleoapp.ResidentDirectorBriefingLoopStatusNoProgressV0 ||
		runtime.launchCountV0() != 1 {
		t.Fatalf("result=%+v launches=%d", result, runtime.launchCountV0())
	}
	run, err := stack.Stores.RunStore.LoadRunV0(ctx, runRef)
	if err != nil {
		t.Fatalf("LoadRunV0: %v", err)
	}
	agentRef := orquestacionnucleoapp.WorkflowTaskAgentRequestRefV0(taskRef)
	if !codexStackStringInSetForTestV0(run.StartedAgents, agentRef) {
		t.Fatalf("started=%v want %s", run.StartedAgents, agentRef)
	}
}

func TestCodexStackResidentDirectorV0ProcesaLoteSegunMaxRunsPerTickV0(t *testing.T) {
	ctx := context.Background()
	runtime := newFakeCodexStackRuntimeV0()
	stack := mustBuildCodexStackForTestV0(t, runtime)
	now := time.Date(2026, 6, 8, 10, 30, 0, 0, time.UTC)
	firstRunRef := "run-resident-codex-stack-batch-001"
	secondRunRef := "run-resident-codex-stack-batch-002"
	seedCodexStackResidentDirectorRunV0(t, ctx, stack, firstRunRef, "task-resident-codex-stack-batch-001")
	seedCodexStackResidentDirectorRunV0(t, ctx, stack, secondRunRef, "task-resident-codex-stack-batch-002")
	for _, runRef := range []string{firstRunRef, secondRunRef} {
		if _, err := stack.Stores.RunQueue.SetRunPriorityV0(ctx, orquestarunqueue.RunQueuePriorityCommandV0{
			RunRef:        runRef,
			QueueRef:      normalizeRunQueueConfigV0(stack.RunQueue).QueueRef,
			AppRef:        "app-resident-codex-stack-batch",
			Status:        orquestarunqueue.RunStatusReadyV0,
			PriorityScore: 80,
			UpdatedAt:     now,
			RequestedBy:   "resident-director-test",
			Reason:        "resident-director-batch-test",
			EvidenceRefs:  []string{"evidence-ref-resident-batch-candidate"},
		}); err != nil {
			t.Fatalf("SetRunPriorityV0 %s: %v", runRef, err)
		}
	}

	result, err := stack.RunCodexStackResidentDirectorV0(ctx, CodexStackResidentDirectorCommandV0{
		OccurredAt:            "2026-06-08T10:30:00Z",
		CorrelationID:         "corr-resident-codex-stack-batch",
		MaxRunsPerTick:        2,
		MaxExecutions:         2,
		RunQueueReadLimit:     2,
		MaxActions:            1,
		MaxDispatchesPerWait:  2,
		MaxCommands:           6,
		MaxOutboxPerCycle:     6,
		QueueRankingPolicyNow: now,
		EvidenceRefs:          []string{"evidence-ref-resident-batch-command"},
	})
	if err != nil {
		t.Fatalf("RunCodexStackResidentDirectorV0: %v result=%+v", err, result)
	}
	if len(result.Runs) != 2 ||
		!codexStackStringInSetForTestV0(result.RunRefs, firstRunRef) ||
		!codexStackStringInSetForTestV0(result.RunRefs, secondRunRef) ||
		result.ExecutedActions != 2 {
		t.Fatalf("result=%+v launches=%d", result, runtime.launchCountV0())
	}
}

func TestCodexStackResidentBriefingSourceV0NoReutilizaStopMaxStepsInternoV0(t *testing.T) {
	runRef := "run-ref-resident-stop-max-001"
	stopBudget := mustCodexStackResidentBriefingForTestV0(
		t,
		runRef,
		orquestadirectorsupervisor.DirectorSupervisorActionStopMaxStepsV0,
		orquestadirectorsupervisor.DirectorSupervisorReasonMaxStepsV0,
		false,
	)
	source := codexStackResidentBriefingSourceV0{}

	briefing, err := source.BuildResidentDirectorBriefingV0(
		context.Background(),
		orquestacionnucleoapp.ResidentDirectorBriefingBuildRequestV0{
			RunRef:     runRef,
			StepNumber: 2,
			PreviousStep: &orquestacionnucleoapp.ResidentDirectorBriefingLoopStepV0{
				Execution: &orquestacionnucleoapp.DirectorBriefingExecutionResultV0{
					Status: orquestacionnucleoapp.DirectorBriefingExecutionStatusRunStepV0,
					Burst: &orquestadirectorsupervisedburst.DirectorSupervisedBurstResultV0{
						RunRef:        runRef,
						MaxSteps:      1,
						ExecutedSteps: 1,
						FinalAction:   orquestadirectorsupervisor.DirectorSupervisorActionStopMaxStepsV0,
						FinalBriefing: &stopBudget,
					},
				},
			},
		},
	)
	if err != nil {
		t.Fatalf("BuildResidentDirectorBriefingV0: %v", err)
	}
	if briefing.DecisionAction != orquestadirectorsupervisor.DirectorSupervisorActionContinueV0 ||
		briefing.NextAction == nil ||
		briefing.NextAction.Kind != orquestadirectorsupervisor.DirectorSupervisorActionKindRunStepV0 {
		t.Fatalf("briefing=%+v", briefing)
	}
}

func seedCodexStackResidentDirectorRunV0(
	t *testing.T,
	ctx context.Context,
	stack StackV0,
	runRef string,
	taskRef string,
) {
	t.Helper()
	task := orquestacoreworkflow.WorkflowTaskV0{
		SchemaVersion:      orquestacoreworkflow.WorkflowTaskSchemaVersionV0,
		TaskID:             taskRef,
		RunID:              runRef,
		PhaseID:            orquestacoreworkflow.OrchestrationPhaseProgramacionV0,
		WorkProfileKind:    orquestacoreworkflow.WorkProfileImplementationV0,
		Title:              "Resident director codex stack task",
		Summary:            "Fixture de runtime residente para stack Codex.",
		WriteSet:           []string{"internal/resident-director/fixture.txt"},
		AcceptanceCriteria: []string{"el agente fake entrega un ACK durable"},
		RequiredTests:      []string{"go test ./modulos/orquesta-app-codex-stack"},
		ContextRefs:        []string{"context-ref-resident-director-test"},
	}
	if err := stack.Stores.TaskStore.SaveWorkflowTaskV0(ctx, task); err != nil {
		t.Fatalf("SaveWorkflowTaskV0: %v", err)
	}
	run := orquestacoreworkflow.OrchestrationRunV0{
		SchemaVersion: orquestacoreworkflow.OrchestrationRunSchemaVersionV0,
		RunID:         runRef,
		ProjectRef:    "project-ref-resident-codex-stack",
		AppSpecRef:    "app-spec-ref-resident-codex-stack",
		Status:        orquestacoreworkflow.OrchestrationRunStatusActiveV0,
		CurrentPhase:  orquestacoreworkflow.OrchestrationPhaseProgramacionV0,
		Phases: []orquestacoreworkflow.OrchestrationPhaseV0{{
			ID:                  orquestacoreworkflow.OrchestrationPhaseProgramacionV0,
			Status:              orquestacoreworkflow.OrchestrationPhaseStatusActiveV0,
			RecommendedCapacity: orquestacoreworkflow.OrchestrationCapacityXHighV0,
		}},
		Tasks: []string{taskRef},
	}
	if err := stack.Stores.RunStore.SaveRunV0(ctx, run); err != nil {
		t.Fatalf("SaveRunV0: %v", err)
	}
}

func mustCodexStackResidentBriefingForTestV0(
	t *testing.T,
	runRef string,
	action orquestadirectorsupervisor.DirectorSupervisorActionV0,
	reason string,
	shouldContinue bool,
) orquestadirectorsupervisor.DirectorSupervisorBriefingV0 {
	t.Helper()
	briefing, err := orquestadirectorsupervisor.BuildDirectorSupervisorBriefingV0(
		orquestadirectorsupervisor.DirectorSupervisorBriefingInputV0{
			Decision: orquestadirectorsupervisor.DirectorSupervisorDecisionV0{
				RunRef:                   runRef,
				Action:                   action,
				ShouldContinue:           shouldContinue,
				AutonomousRecommendation: residentDirectorSupervisorRecommendationForTestV0(action),
				ReasonCode:               reason,
				StepNumber:               1,
				MaxSteps:                 1,
			},
		},
	)
	if err != nil {
		t.Fatalf("BuildDirectorSupervisorBriefingV0: %v", err)
	}
	return briefing
}

func residentDirectorSupervisorRecommendationForTestV0(
	action orquestadirectorsupervisor.DirectorSupervisorActionV0,
) orquestadirectorsupervisor.DirectorSupervisorAutonomousRecommendationV0 {
	switch action {
	case orquestadirectorsupervisor.DirectorSupervisorActionContinueV0:
		return orquestadirectorsupervisor.DirectorSupervisorAutonomousContinueV0
	case orquestadirectorsupervisor.DirectorSupervisorActionWaitOutboxV0,
		orquestadirectorsupervisor.DirectorSupervisorActionWaitExternalV0:
		return orquestadirectorsupervisor.DirectorSupervisorAutonomousWaitV0
	case orquestadirectorsupervisor.DirectorSupervisorActionNeedsDirectorV0:
		return orquestadirectorsupervisor.DirectorSupervisorAutonomousNeedsDirectorV0
	default:
		return orquestadirectorsupervisor.DirectorSupervisorAutonomousStopV0
	}
}
