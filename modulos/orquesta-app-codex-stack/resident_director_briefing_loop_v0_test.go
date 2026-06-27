package orquestaappcodexstack

import (
	"context"
	"strings"
	"testing"
	"time"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadecisioncouncil "orquesta/modulos/orquesta-decision-council"
	orquestadirectorsupervisedburst "orquesta/modulos/orquesta-director-supervised-burst"
	orquestadirectorsupervisor "orquesta/modulos/orquesta-director-supervisor"
	orquestagoal "orquesta/modulos/orquesta-goal"
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

func TestCodexStackDirectorPortsV0ReconstruyeClosureSourceSiFaltaEnPortsV0(t *testing.T) {
	stack := mustBuildCodexStackForTestV0(t, newFakeCodexStackRuntimeV0())
	ports := stack.Ports
	ports.OperationalClosureSource = nil

	recovered := stack.directorPortsWithClosureSourceV0(ports)

	if recovered.OperationalClosureSource == nil {
		t.Fatalf("resident director debe reconstruir OperationalClosureSource desde stores del stack")
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

func TestCodexStackResidentDirectorV0NoDrenaLegacySiGoalFirstEnCola(t *testing.T) {
	ctx := context.Background()
	runtime := newFakeCodexStackRuntimeV0()
	stack := mustBuildCodexStackForTestV0(t, runtime)
	goalStates := newGoalFirstQueueStateStoreForTestV0()
	stack.Ports.GoalStateStore = goalStates
	now := time.Date(2026, 6, 27, 12, 0, 0, 0, time.UTC)
	runRef := "run-resident-goal-first-guard-001"
	saveCodexStackGoalFirstStateForTestV0(t, ctx, goalStates, runRef)
	if err := stack.Ports.RunStore.SaveRunV0(ctx, orquestacoreworkflow.OrchestrationRunV0{
		SchemaVersion: orquestacoreworkflow.OrchestrationRunSchemaVersionV0,
		RunID:         runRef,
		ProjectRef:    "app-resident-goal-first",
		Status:        orquestacoreworkflow.OrchestrationRunStatusActiveV0,
		CurrentPhase:  orquestacoreworkflow.OrchestrationPhaseProgramacionV0,
	}); err != nil {
		t.Fatalf("SaveRunV0: %v", err)
	}
	if _, err := stack.Stores.RunQueue.SetRunPriorityV0(ctx, orquestarunqueue.RunQueuePriorityCommandV0{
		RunRef:        runRef,
		QueueRef:      normalizeRunQueueConfigV0(stack.RunQueue).QueueRef,
		AppRef:        "app-resident-goal-first",
		Status:        orquestarunqueue.RunStatusReadyV0,
		PriorityScore: 80,
		UpdatedAt:     now,
		RequestedBy:   "resident-director-goal-first-test",
		Reason:        "candidato goal-first accidental para guard residente",
		EvidenceRefs:  []string{"evidence-ref-resident-goal-first-candidate"},
	}); err != nil {
		t.Fatalf("SetRunPriorityV0: %v", err)
	}

	result, err := stack.RunCodexStackResidentDirectorV0(ctx, CodexStackResidentDirectorCommandV0{
		OccurredAt:            now.Format(time.RFC3339),
		CorrelationID:         "corr-resident-goal-first-guard-001",
		MaxRunsPerTick:        1,
		MaxExecutions:         1,
		MaxActions:            8,
		MaxDispatchesPerWait:  4,
		MaxCommands:           6,
		MaxOutboxPerCycle:     6,
		QueueRankingPolicyNow: now,
		EvidenceRefs:          []string{"evidence-ref-resident-goal-first-command"},
	})
	if err != nil {
		t.Fatalf("RunCodexStackResidentDirectorV0: %v result=%+v", err, result)
	}
	if result.RunRef != runRef ||
		result.Status != codexStackGoalFirstObserveRequiredOutcomeV0 ||
		result.ExecutedActions != 0 ||
		runtime.launchCountV0() != 0 {
		t.Fatalf("result=%+v launches=%d", result, runtime.launchCountV0())
	}
	candidates, err := stack.Stores.RunQueue.ListRunSchedulingCandidatesV0(ctx, orquestarunqueue.RunQueueReadRequestV0{
		QueueRef:             normalizeRunQueueConfigV0(stack.RunQueue).QueueRef,
		IncludeNonExecutable: true,
	})
	if err != nil {
		t.Fatalf("ListRunSchedulingCandidatesV0: %v", err)
	}
	if len(candidates) != 1 ||
		candidates[0].RunRef != runRef ||
		candidates[0].Status != orquestarunqueue.RunStatusDeliveredV0 {
		t.Fatalf("candidates=%+v", candidates)
	}
	run, err := stack.Ports.RunStore.LoadRunV0(ctx, runRef)
	if err != nil {
		t.Fatalf("LoadRunV0: %v", err)
	}
	if len(run.Tasks) != 0 || len(run.StartedAgents) != 0 {
		t.Fatalf("resident guard materializo legacy: %+v", run)
	}
}

func TestCodexStackResidentDirectorV0NoDrenaLegacySiGoalFirstNoTieneState(t *testing.T) {
	ctx := context.Background()
	runtime := newFakeCodexStackRuntimeV0()
	stack := mustBuildCodexStackForTestV0(t, runtime)
	stack.Ports.GoalStateStore = newGoalFirstQueueStateStoreForTestV0()
	now := time.Date(2026, 6, 27, 12, 15, 0, 0, time.UTC)
	runRef := "run-resident-goal-first-missing-state-001"
	if err := stack.Ports.RunStore.SaveRunV0(ctx, orquestacoreworkflow.OrchestrationRunV0{
		SchemaVersion: orquestacoreworkflow.OrchestrationRunSchemaVersionV0,
		RunID:         runRef,
		ProjectRef:    "app-resident-goal-first-missing-state",
		AppSpecRef:    "app-spec-ref-autoprogramming-missing-state",
		Status:        orquestacoreworkflow.OrchestrationRunStatusActiveV0,
		CurrentPhase:  orquestacoreworkflow.OrchestrationPhaseProgramacionV0,
	}); err != nil {
		t.Fatalf("SaveRunV0: %v", err)
	}
	if _, err := stack.Stores.RunQueue.SetRunPriorityV0(ctx, orquestarunqueue.RunQueuePriorityCommandV0{
		RunRef:        runRef,
		QueueRef:      normalizeRunQueueConfigV0(stack.RunQueue).QueueRef,
		AppRef:        "app-resident-goal-first-missing-state",
		Status:        orquestarunqueue.RunStatusReadyV0,
		PriorityScore: 80,
		UpdatedAt:     now,
		RequestedBy:   "resident-director-goal-first-missing-state-test",
		Reason:        "candidato goal-first sin state no debe caer a legacy",
		EvidenceRefs:  []string{"evidence-ref-resident-goal-first-missing-state-candidate"},
	}); err != nil {
		t.Fatalf("SetRunPriorityV0: %v", err)
	}

	result, err := stack.RunCodexStackResidentDirectorV0(ctx, CodexStackResidentDirectorCommandV0{
		OccurredAt:            now.Format(time.RFC3339),
		CorrelationID:         "corr-resident-goal-first-missing-state-001",
		MaxRunsPerTick:        1,
		MaxExecutions:         1,
		MaxActions:            8,
		MaxDispatchesPerWait:  4,
		MaxCommands:           6,
		MaxOutboxPerCycle:     6,
		QueueRankingPolicyNow: now,
		EvidenceRefs:          []string{"evidence-ref-resident-goal-first-missing-state-command"},
	})
	if err != nil {
		t.Fatalf("RunCodexStackResidentDirectorV0: %v result=%+v", err, result)
	}
	if result.RunRef != runRef ||
		result.Status != codexStackGoalFirstStateMissingOutcomeV0 ||
		result.ExecutedActions != 0 ||
		runtime.launchCountV0() != 0 {
		t.Fatalf("result=%+v launches=%d", result, runtime.launchCountV0())
	}
	candidates, err := stack.Stores.RunQueue.ListRunSchedulingCandidatesV0(ctx, orquestarunqueue.RunQueueReadRequestV0{
		QueueRef:             normalizeRunQueueConfigV0(stack.RunQueue).QueueRef,
		IncludeNonExecutable: true,
	})
	if err != nil {
		t.Fatalf("ListRunSchedulingCandidatesV0: %v", err)
	}
	if len(candidates) != 1 ||
		candidates[0].RunRef != runRef ||
		candidates[0].Status != orquestarunqueue.RunStatusStoppedV0 {
		t.Fatalf("candidates=%+v", candidates)
	}
	run, err := stack.Ports.RunStore.LoadRunV0(ctx, runRef)
	if err != nil {
		t.Fatalf("LoadRunV0: %v", err)
	}
	if len(run.Tasks) != 0 || len(run.StartedAgents) != 0 {
		t.Fatalf("resident missing-state guard materializo legacy: %+v", run)
	}
}

func saveCodexStackGoalFirstStateForTestV0(
	t *testing.T,
	ctx context.Context,
	store *goalFirstQueueStateStoreForTestV0,
	runRef string,
) {
	t.Helper()
	goalRef := "goal-ref-" + runRef
	if err := store.SaveGoalWorkStateV0(ctx, orquestagoal.GoalWorkStateV0{
		RunRef:          runRef,
		GoalRef:         goalRef,
		ExternalGoalRef: "external-" + goalRef,
		Status:          orquestagoal.GoalStatusRunningV0,
		Spec: orquestagoal.GoalWorkSpecV0{
			GoalRef:      goalRef,
			RunRef:       runRef,
			ProjectRef:   "app-resident-goal-first",
			Objective:    "Probar que el residente no drena un goal-first con loop legacy.",
			DirectorKind: orquestagoal.GoalDirectorKindCodexGoalV0,
			WriteSet: []orquestagoal.GoalWriteScopeV0{{
				Path: "modulos/orquesta-app-codex-stack/goal-first-resident.md",
			}},
			EvidenceRefs: []string{"evidence-ref-resident-goal-first-state"},
		},
		LaunchReceipt: orquestagoal.GoalLaunchReceiptV0{
			Status:          orquestagoal.GoalStatusRunningV0,
			GoalRef:         goalRef,
			ExternalGoalRef: "external-" + goalRef,
			EvidenceRefs:    []string{"evidence-ref-resident-goal-first-launch"},
		},
		EvidenceRefs: []string{"evidence-ref-resident-goal-first-state"},
	}); err != nil {
		t.Fatalf("SaveGoalWorkStateV0: %v", err)
	}
}

func TestCodexStackResidentDirectorV0RecuperaRunNoEjecutableAntesDeCoordinarV0(t *testing.T) {
	ctx := context.Background()
	runtime := newFakeCodexStackRuntimeV0()
	stack := mustBuildCodexStackForTestV0(t, runtime)
	now := time.Date(2026, 6, 25, 12, 0, 0, 0, time.UTC)
	runRef := "run-resident-codex-stack-recover-001"
	taskRef := "task-resident-codex-stack-recover-001"
	queueRef := normalizeRunQueueConfigV0(stack.RunQueue).QueueRef
	seedCodexStackResidentDirectorRunV0(t, ctx, stack, runRef, taskRef)
	claim := orquestarunqueue.WorksetClaimV0{
		SchemaVersion: orquestarunqueue.WorksetClaimSchemaVersionV0,
		ClaimRef:      "claim-ref-resident-codex-stack-recover-001",
		RunRef:        runRef,
		TaskRef:       taskRef,
		WriteSet: []orquestarunqueue.ScopeRefV0{{
			Ref: "external/opes/job-demo",
		}},
	}
	if _, err := stack.Stores.RunQueue.SetRunPriorityV0(ctx, orquestarunqueue.RunQueuePriorityCommandV0{
		RunRef:        runRef,
		QueueRef:      queueRef,
		AppRef:        "opes",
		Status:        orquestarunqueue.RunStatusCanceledV0,
		PriorityScore: 80,
		UpdatedAt:     now,
		AttemptGroup: orquestarunqueue.RunQueueAttemptGroupV0{
			GroupRef:     "attempt-group-resident-codex-stack-recover-001",
			ConsumerRef:  "opes",
			ObjectiveRef: "objective-ref-opes-job-demo",
			WorkItemRef:  taskRef,
			WriteSetRefs: []string{"external/opes/job-demo"},
		},
		ParentRunRef:  "parent-run-ref-resident-codex-stack-recover",
		RescueReason:  "resident_director_pending_recovery",
		RequestedBy:   "resident-director-test",
		Reason:        "fixture no ejecutable previo al residente",
		EvidenceRefs:  []string{"evidence-ref-resident-recover-candidate"},
		WorksetClaims: []orquestarunqueue.WorksetClaimV0{claim},
	}); err != nil {
		t.Fatalf("SetRunPriorityV0: %v", err)
	}
	visibleBefore, err := stack.Stores.RunQueue.ListRunSchedulingCandidatesV0(
		ctx,
		orquestarunqueue.RunQueueReadRequestV0{QueueRef: queueRef},
	)
	if err != nil {
		t.Fatalf("ListRunSchedulingCandidatesV0 before: %v", err)
	}
	if len(visibleBefore) != 0 {
		t.Fatalf("visibleBefore=%+v", visibleBefore)
	}

	result, err := stack.RunCodexStackResidentDirectorV0(ctx, CodexStackResidentDirectorCommandV0{
		OccurredAt:            now.Format(time.RFC3339),
		CorrelationID:         "corr-resident-codex-stack-recover",
		MaxRunsPerTick:        1,
		MaxExecutions:         1,
		RunQueueReadLimit:     1,
		MaxActions:            8,
		MaxDispatchesPerWait:  4,
		MaxCommands:           6,
		MaxOutboxPerCycle:     6,
		QueueRankingPolicyNow: now,
		EvidenceRefs:          []string{"evidence-ref-resident-recover-command"},
	})
	if err != nil {
		t.Fatalf("RunCodexStackResidentDirectorV0: %v result=%+v", err, result)
	}
	if result.RunRef != runRef ||
		result.Status == CodexStackResidentDirectorStatusIdleV0 ||
		runtime.launchCountV0() != 1 {
		t.Fatalf("result=%+v launches=%d", result, runtime.launchCountV0())
	}
	all, err := stack.Stores.RunQueue.ListRunSchedulingCandidatesV0(
		ctx,
		orquestarunqueue.RunQueueReadRequestV0{QueueRef: queueRef, IncludeNonExecutable: true},
	)
	if err != nil {
		t.Fatalf("ListRunSchedulingCandidatesV0 after: %v", err)
	}
	if len(all) != 1 ||
		all[0].RunRef != runRef ||
		all[0].AppRef != "opes" ||
		all[0].AttemptGroup.WorkItemRef != taskRef ||
		all[0].ParentRunRef != "parent-run-ref-resident-codex-stack-recover" ||
		all[0].RescueReason != "resident_director_pending_recovery" ||
		len(all[0].WorksetClaims) != 1 ||
		all[0].WorksetClaims[0].ClaimRef != claim.ClaimRef ||
		!codexStackStringInSetForTestV0(all[0].EvidenceRefs, "evidence-ref-run-queue-non-executable-active-reconciled") {
		t.Fatalf("all=%+v", all)
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

func TestCodexStackResidentBriefingSourceV0EmiteConsejoConEstadoEstructuralV0(t *testing.T) {
	ctx := context.Background()
	stack := mustBuildCodexStackForTestV0(t, newFakeCodexStackRuntimeV0())
	run := codexStackResidentCouncilRunForTestV0(
		"run-resident-council-source-001",
		orquestacoreworkflow.OrchestrationPhasePlanificacionMicrotareasV0,
		[]string{"contract:function:decision-council:v0"},
	)
	if err := stack.Stores.RunStore.SaveRunV0(ctx, run); err != nil {
		t.Fatalf("SaveRunV0: %v", err)
	}
	source := codexStackResidentBriefingSourceV0{
		RunStore:     stack.Ports.RunStore,
		TaskStore:    stack.Ports.DirectorTaskStore,
		ObjectiveRef: "resident-director:" + run.RunID,
		ContextRefs:  []string{"context-ref-codex-stack-resident-director"},
	}

	briefing, err := source.BuildResidentDirectorBriefingV0(
		ctx,
		orquestacionnucleoapp.ResidentDirectorBriefingBuildRequestV0{
			RunRef:       run.RunID,
			StepNumber:   1,
			EvidenceRefs: []string{"evidence-ref-resident-council-source"},
		},
	)
	if err != nil {
		t.Fatalf("BuildResidentDirectorBriefingV0: %v", err)
	}
	if briefing.NextAction == nil ||
		briefing.NextAction.Kind != codexStackResidentActionKindMaterializeDecisionCouncilV0 ||
		!briefing.NextAction.SafeToApply ||
		briefing.NextAction.RequiresDirector {
		t.Fatalf("briefing=%+v", briefing)
	}
}

func TestCodexStackResidentBriefingSourceV0EsperaAgentesExternosPendientesV0(t *testing.T) {
	ctx := context.Background()
	stack := mustBuildCodexStackForTestV0(t, newFakeCodexStackRuntimeV0())
	run := codexStackResidentCouncilRunForTestV0(
		"run-resident-pending-agent-source-001",
		orquestacoreworkflow.OrchestrationPhaseProgramacionV0,
		[]string{"contract:function:decision-council:v0"},
	)
	run.StartedAgents = []string{"agent-ref-resident-pending-001"}
	if err := stack.Stores.RunStore.SaveRunV0(ctx, run); err != nil {
		t.Fatalf("SaveRunV0: %v", err)
	}
	source := codexStackResidentBriefingSourceV0{
		RunStore:        stack.Ports.RunStore,
		TaskStore:       stack.Ports.DirectorTaskStore,
		DecisionCouncil: DecisionCouncilConfigV0{VoteSource: fakeDecisionCouncilVoteSourceForTestV0{}},
		ObjectiveRef:    "resident-director:" + run.RunID,
		ContextRefs:     []string{"context-ref-codex-stack-resident-director"},
	}

	briefing, err := source.BuildResidentDirectorBriefingV0(
		ctx,
		orquestacionnucleoapp.ResidentDirectorBriefingBuildRequestV0{
			RunRef:       run.RunID,
			StepNumber:   1,
			EvidenceRefs: []string{"evidence-ref-resident-pending-agent-source"},
		},
	)
	if err != nil {
		t.Fatalf("BuildResidentDirectorBriefingV0: %v", err)
	}
	if briefing.AutonomousRecommendation != orquestadirectorsupervisor.DirectorSupervisorAutonomousWaitV0 ||
		briefing.ReasonCode != "resident_director_external_agents_pending" ||
		briefing.NextAction != nil ||
		briefing.DecisionAction != "" {
		t.Fatalf("briefing=%+v", briefing)
	}
}

func TestCodexStackResidentBriefingSourceV0NoEmiteConsejoConProgramacionAbiertaV0(t *testing.T) {
	ctx := context.Background()
	stack := mustBuildCodexStackForTestV0(t, newFakeCodexStackRuntimeV0())
	run := codexStackResidentCouncilRunForTestV0(
		"run-resident-open-programming-source-001",
		orquestacoreworkflow.OrchestrationPhasePlanificacionMicrotareasV0,
		[]string{"contract:function:decision-council:v0"},
	)
	task := orquestacoreworkflow.WorkflowTaskV0{
		SchemaVersion:      orquestacoreworkflow.WorkflowTaskSchemaVersionV0,
		TaskID:             "task-open-programming-resident-001",
		RunID:              run.RunID,
		PhaseID:            orquestacoreworkflow.OrchestrationPhaseProgramacionV0,
		WorkProfileKind:    orquestacoreworkflow.WorkProfileImplementationV0,
		Title:              "Programming open task",
		Summary:            "Tarea abierta que debe impedir consejo paralelo.",
		WriteSet:           []string{"internal/open/task.go"},
		AcceptanceCriteria: []string{"no abrir consejo mientras siga abierta"},
		RequiredTests:      []string{"go test ./..."},
	}
	run.Tasks = []string{task.TaskID}
	if err := stack.Stores.TaskStore.SaveWorkflowTaskV0(ctx, task); err != nil {
		t.Fatalf("SaveWorkflowTaskV0: %v", err)
	}
	if err := stack.Stores.RunStore.SaveRunV0(ctx, run); err != nil {
		t.Fatalf("SaveRunV0: %v", err)
	}
	source := codexStackResidentBriefingSourceV0{
		RunStore:        stack.Ports.RunStore,
		TaskStore:       stack.Ports.DirectorTaskStore,
		DecisionCouncil: DecisionCouncilConfigV0{VoteSource: fakeDecisionCouncilVoteSourceForTestV0{}},
		ObjectiveRef:    "resident-director:" + run.RunID,
		ContextRefs:     []string{"context-ref-codex-stack-resident-director"},
	}

	briefing, err := source.BuildResidentDirectorBriefingV0(
		ctx,
		orquestacionnucleoapp.ResidentDirectorBriefingBuildRequestV0{
			RunRef:       run.RunID,
			StepNumber:   1,
			EvidenceRefs: []string{"evidence-ref-resident-open-autoprogramming-source"},
		},
	)
	if err != nil {
		t.Fatalf("BuildResidentDirectorBriefingV0: %v", err)
	}
	if briefing.NextAction == nil ||
		briefing.NextAction.Kind == codexStackResidentActionKindMaterializeDecisionCouncilV0 ||
		briefing.NextAction.Kind != orquestadirectorsupervisor.DirectorSupervisorActionKindRunStepV0 {
		t.Fatalf("briefing=%+v", briefing)
	}
}

func TestCodexStackResidentBriefingSourceV0NoEmiteConsejoFueraDeFaseDeCreacionV0(t *testing.T) {
	ctx := context.Background()
	stack := mustBuildCodexStackForTestV0(t, newFakeCodexStackRuntimeV0())
	run := codexStackResidentCouncilRunForTestV0(
		"run-resident-council-source-phase-001",
		orquestacoreworkflow.OrchestrationPhaseBrainstormingArquitecturaV0,
		[]string{"contract:function:decision-council:v0"},
	)
	if err := stack.Stores.RunStore.SaveRunV0(ctx, run); err != nil {
		t.Fatalf("SaveRunV0: %v", err)
	}
	source := codexStackResidentBriefingSourceV0{
		RunStore:        stack.Ports.RunStore,
		TaskStore:       stack.Ports.DirectorTaskStore,
		DecisionCouncil: DecisionCouncilConfigV0{VoteSource: fakeDecisionCouncilVoteSourceForTestV0{}},
	}

	briefing, err := source.BuildResidentDirectorBriefingV0(
		ctx,
		orquestacionnucleoapp.ResidentDirectorBriefingBuildRequestV0{
			RunRef:     run.RunID,
			StepNumber: 1,
		},
	)
	if err != nil {
		t.Fatalf("BuildResidentDirectorBriefingV0: %v", err)
	}
	if briefing.NextAction == nil ||
		briefing.NextAction.Kind == codexStackResidentActionKindMaterializeDecisionCouncilV0 ||
		briefing.NextAction.Kind != orquestadirectorsupervisor.DirectorSupervisorActionKindRunStepV0 {
		t.Fatalf("briefing=%+v", briefing)
	}
}

func TestCodexStackResidentCouncilHandlerV0MaterializaUnaVezConContratoPublicadoV0(t *testing.T) {
	ctx := context.Background()
	stack := mustBuildCodexStackForTestV0(t, newFakeCodexStackRuntimeV0())
	run := codexStackResidentCouncilRunForTestV0(
		"run-resident-council-handler-001",
		orquestacoreworkflow.OrchestrationPhasePlanificacionMicrotareasV0,
		[]string{"contract:function:decision-council:v0"},
	)
	if err := stack.Stores.RunStore.SaveRunV0(ctx, run); err != nil {
		t.Fatalf("SaveRunV0: %v", err)
	}
	handler := codexStackResidentExternalActionHandlerV0{Ports: stack.Ports}
	request := codexStackResidentCouncilExternalActionRequestForTestV0(run.RunID)

	first, err := handler.ExecuteDirectorBriefingExternalActionV0(ctx, request)
	if err != nil {
		t.Fatalf("ExecuteDirectorBriefingExternalActionV0 first: %v", err)
	}
	if first.Status != orquestacionnucleoapp.DirectorBriefingExecutionStatusExternalAppliedV0 {
		t.Fatalf("first=%+v", first)
	}
	afterFirst := codexStackResidentCouncilRunFromStoreForTestV0(t, ctx, stack, run.RunID)
	if afterFirst.CurrentPhase != orquestacoreworkflow.OrchestrationPhaseBrainstormingArquitecturaV0 {
		t.Fatalf("current_phase=%s, want %s", afterFirst.CurrentPhase, orquestacoreworkflow.OrchestrationPhaseBrainstormingArquitecturaV0)
	}
	firstTaskRefs := codexStackResidentCouncilTaskRefsFromRunForTestV0(afterFirst)
	if len(firstTaskRefs) == 0 {
		t.Fatalf("no se materializaron tareas de consejo: %+v", afterFirst.Tasks)
	}
	tasks, err := stack.Ports.DirectorTaskStore.LoadWorkflowTasksV0(ctx, run.RunID, firstTaskRefs)
	if err != nil {
		t.Fatalf("LoadWorkflowTasksV0: %v", err)
	}
	for _, task := range tasks {
		if len(task.FunctionContractRefs) == 0 ||
			task.FunctionContractRefs[0].ContractRef != "contract:function:decision-council:v0" {
			t.Fatalf("task sin contrato esperado: %+v", task)
		}
	}
	candidateTaskRefs := codexStackResidentCouncilCandidateTaskRefsForTestV0(t, ctx, stack, afterFirst)
	if !codexStackResidentCouncilHasTaskRefWithPrefixForTestV0(candidateTaskRefs, codexStackResidentCouncilProposalTaskPrefixV0) ||
		codexStackResidentCouncilHasTaskRefWithPrefixForTestV0(candidateTaskRefs, codexStackResidentCouncilVoteTaskPrefixV0) {
		t.Fatalf("candidate_task_refs=%v", candidateTaskRefs)
	}

	second, err := handler.ExecuteDirectorBriefingExternalActionV0(ctx, request)
	if err != nil {
		t.Fatalf("ExecuteDirectorBriefingExternalActionV0 second: %v", err)
	}
	afterSecond := codexStackResidentCouncilRunFromStoreForTestV0(t, ctx, stack, run.RunID)
	secondTaskRefs := codexStackResidentCouncilTaskRefsFromRunForTestV0(afterSecond)
	if second.Status != orquestacionnucleoapp.DirectorBriefingExecutionStatusExternalAppliedV0 ||
		len(secondTaskRefs) != len(firstTaskRefs) {
		t.Fatalf("second=%+v first_tasks=%v second_tasks=%v", second, firstTaskRefs, secondTaskRefs)
	}
}

func TestCodexStackResidentCouncilHandlerV0NoMaterializaConProgramacionAbiertaV0(t *testing.T) {
	ctx := context.Background()
	stack := mustBuildCodexStackForTestV0(t, newFakeCodexStackRuntimeV0())
	run := codexStackResidentCouncilRunForTestV0(
		"run-resident-council-handler-open-programming-001",
		orquestacoreworkflow.OrchestrationPhasePlanificacionMicrotareasV0,
		[]string{"contract:function:decision-council:v0"},
	)
	task := orquestacoreworkflow.WorkflowTaskV0{
		SchemaVersion:      orquestacoreworkflow.WorkflowTaskSchemaVersionV0,
		TaskID:             "task-open-programming-council-handler-001",
		RunID:              run.RunID,
		PhaseID:            orquestacoreworkflow.OrchestrationPhaseProgramacionV0,
		WorkProfileKind:    orquestacoreworkflow.WorkProfileImplementationV0,
		Title:              "Programacion abierta",
		Summary:            "Tarea de implementacion en curso.",
		WriteSet:           []string{"internal/app"},
		AcceptanceCriteria: []string{"entrega implementada"},
		RequiredTests:      []string{"go test ./..."},
	}
	run.Tasks = []string{task.TaskID}
	if err := stack.Stores.TaskStore.SaveWorkflowTaskV0(ctx, task); err != nil {
		t.Fatalf("SaveWorkflowTaskV0: %v", err)
	}
	if err := stack.Stores.RunStore.SaveRunV0(ctx, run); err != nil {
		t.Fatalf("SaveRunV0: %v", err)
	}
	handler := codexStackResidentExternalActionHandlerV0{Ports: stack.Ports}

	result, err := handler.ExecuteDirectorBriefingExternalActionV0(
		ctx,
		codexStackResidentCouncilExternalActionRequestForTestV0(run.RunID),
	)
	if err != nil {
		t.Fatalf("ExecuteDirectorBriefingExternalActionV0: %v", err)
	}
	after := codexStackResidentCouncilRunFromStoreForTestV0(t, ctx, stack, run.RunID)
	if result.Status != orquestacionnucleoapp.DirectorBriefingExecutionStatusExternalPendingV0 ||
		!codexStackStringInSetForTestV0(result.EvidenceRefs, "evidence-ref-codex-stack-resident-council-open-programming-hold") ||
		len(after.Tasks) != 1 ||
		codexStackResidentCouncilHasTaskRefWithPrefixForTestV0(after.Tasks, codexStackResidentCouncilTaskPrefixV0) {
		t.Fatalf("result=%+v after_tasks=%v", result, after.Tasks)
	}
}

func TestCodexStackResidentCouncilV0AbreVotacionCuandoTerminaBrainstormV0(t *testing.T) {
	ctx := context.Background()
	stack := mustBuildCodexStackForTestV0(t, newFakeCodexStackRuntimeV0())
	run := codexStackResidentCouncilRunForTestV0(
		"run-resident-council-vote-phase-001",
		orquestacoreworkflow.OrchestrationPhasePlanificacionMicrotareasV0,
		[]string{"contract:function:decision-council:v0"},
	)
	if err := stack.Stores.RunStore.SaveRunV0(ctx, run); err != nil {
		t.Fatalf("SaveRunV0: %v", err)
	}
	handler := codexStackResidentExternalActionHandlerV0{Ports: stack.Ports}
	if _, err := handler.ExecuteDirectorBriefingExternalActionV0(
		ctx,
		codexStackResidentCouncilExternalActionRequestForTestV0(run.RunID),
	); err != nil {
		t.Fatalf("materialize council: %v", err)
	}
	brainstorm := codexStackResidentCouncilRunFromStoreForTestV0(t, ctx, stack, run.RunID)
	tasks, err := stack.Ports.DirectorTaskStore.LoadWorkflowTasksV0(
		ctx,
		run.RunID,
		codexStackResidentCouncilTaskRefsFromRunForTestV0(brainstorm),
	)
	if err != nil {
		t.Fatalf("LoadWorkflowTasksV0: %v", err)
	}
	for _, task := range tasks {
		if codexStackResidentCouncilTaskRoleV0(task) != "v" {
			brainstorm.DeliveredTasks = append(brainstorm.DeliveredTasks, task.TaskID)
		}
	}
	if err := stack.Stores.RunStore.SaveRunV0(ctx, brainstorm); err != nil {
		t.Fatalf("SaveRunV0 brainstorm delivered: %v", err)
	}
	source := codexStackResidentBriefingSourceV0{
		RunStore:        stack.Ports.RunStore,
		TaskStore:       stack.Ports.DirectorTaskStore,
		DecisionCouncil: DecisionCouncilConfigV0{VoteSource: fakeDecisionCouncilVoteSourceForTestV0{}},
	}
	briefing, err := source.BuildResidentDirectorBriefingV0(
		ctx,
		orquestacionnucleoapp.ResidentDirectorBriefingBuildRequestV0{
			RunRef:     run.RunID,
			StepNumber: 2,
		},
	)
	if err != nil {
		t.Fatalf("BuildResidentDirectorBriefingV0: %v", err)
	}
	if briefing.NextAction == nil ||
		briefing.NextAction.Kind != codexStackResidentActionKindOpenCouncilVoteV0 {
		t.Fatalf("briefing=%+v", briefing)
	}
	result, err := handler.ExecuteDirectorBriefingExternalActionV0(ctx, orquestacionnucleoapp.DirectorBriefingExternalActionRequestV0{
		Briefing:      briefing,
		Action:        *briefing.NextAction,
		OccurredAt:    "2026-06-08T12:10:00Z",
		CorrelationID: "corr-" + run.RunID + "-vote",
		EvidenceRefs:  []string{"evidence-ref-" + run.RunID + "-vote-test"},
	})
	if err != nil {
		t.Fatalf("open vote phase: %v", err)
	}
	voteRun := codexStackResidentCouncilRunFromStoreForTestV0(t, ctx, stack, run.RunID)
	voteCandidates := codexStackResidentCouncilCandidateTaskRefsForTestV0(t, ctx, stack, voteRun)
	if result.Status != orquestacionnucleoapp.DirectorBriefingExecutionStatusExternalAppliedV0 ||
		voteRun.CurrentPhase != orquestacoreworkflow.OrchestrationPhaseVotacionYDecisionV0 ||
		!codexStackResidentCouncilHasTaskRefWithPrefixForTestV0(voteCandidates, codexStackResidentCouncilVoteTaskPrefixV0) {
		t.Fatalf("result=%+v phase=%s candidates=%v", result, voteRun.CurrentPhase, voteCandidates)
	}
}

func TestCodexStackResidentCouncilV0AceptaDecisionConVotosEstructuradosV0(t *testing.T) {
	ctx := context.Background()
	stack := mustBuildCodexStackForTestV0(t, newFakeCodexStackRuntimeV0())
	run := codexStackResidentCouncilRunForTestV0(
		"run-resident-council-accept-001",
		orquestacoreworkflow.OrchestrationPhasePlanificacionMicrotareasV0,
		[]string{"contract:function:decision-council:v0"},
	)
	if err := stack.Stores.RunStore.SaveRunV0(ctx, run); err != nil {
		t.Fatalf("SaveRunV0: %v", err)
	}
	handler := codexStackResidentExternalActionHandlerV0{
		Ports: stack.Ports,
		DecisionCouncil: DecisionCouncilConfigV0{
			VoteSource: fakeDecisionCouncilVoteSourceForTestV0{},
		},
	}
	if _, err := handler.ExecuteDirectorBriefingExternalActionV0(
		ctx,
		codexStackResidentCouncilExternalActionRequestForTestV0(run.RunID),
	); err != nil {
		t.Fatalf("materialize council: %v", err)
	}
	brainstorm := codexStackResidentCouncilRunFromStoreForTestV0(t, ctx, stack, run.RunID)
	tasks := codexStackResidentCouncilTasksForTestV0(t, ctx, stack, brainstorm)
	brainstorm = codexStackResidentCouncilMarkRolesDeliveredForTestV0(brainstorm, tasks, "p", "c")
	if err := stack.Stores.RunStore.SaveRunV0(ctx, brainstorm); err != nil {
		t.Fatalf("SaveRunV0 brainstorm delivered: %v", err)
	}
	source := codexStackResidentBriefingSourceV0{
		RunStore:        stack.Ports.RunStore,
		TaskStore:       stack.Ports.DirectorTaskStore,
		DecisionCouncil: DecisionCouncilConfigV0{VoteSource: fakeDecisionCouncilVoteSourceForTestV0{}},
	}
	openVote, err := source.BuildResidentDirectorBriefingV0(
		ctx,
		orquestacionnucleoapp.ResidentDirectorBriefingBuildRequestV0{
			RunRef:     run.RunID,
			StepNumber: 2,
		},
	)
	if err != nil {
		t.Fatalf("BuildResidentDirectorBriefingV0 open vote: %v", err)
	}
	if openVote.NextAction == nil || openVote.NextAction.Kind != codexStackResidentActionKindOpenCouncilVoteV0 {
		t.Fatalf("openVote=%+v", openVote)
	}
	if _, err := handler.ExecuteDirectorBriefingExternalActionV0(ctx, orquestacionnucleoapp.DirectorBriefingExternalActionRequestV0{
		Briefing:      openVote,
		Action:        *openVote.NextAction,
		OccurredAt:    "2026-06-08T12:20:00Z",
		CorrelationID: "corr-" + run.RunID + "-open-vote",
		EvidenceRefs:  []string{"evidence-ref-" + run.RunID + "-open-vote-test"},
	}); err != nil {
		t.Fatalf("open vote phase: %v", err)
	}
	voteRun := codexStackResidentCouncilRunFromStoreForTestV0(t, ctx, stack, run.RunID)
	voteRun = codexStackResidentCouncilMarkRolesDeliveredForTestV0(voteRun, tasks, "v")
	if err := stack.Stores.RunStore.SaveRunV0(ctx, voteRun); err != nil {
		t.Fatalf("SaveRunV0 votes delivered: %v", err)
	}
	accept, err := source.BuildResidentDirectorBriefingV0(
		ctx,
		orquestacionnucleoapp.ResidentDirectorBriefingBuildRequestV0{
			RunRef:     run.RunID,
			StepNumber: 3,
		},
	)
	if err != nil {
		t.Fatalf("BuildResidentDirectorBriefingV0 accept: %v", err)
	}
	if accept.NextAction == nil || accept.NextAction.Kind != codexStackResidentActionKindAcceptCouncilDecisionV0 {
		t.Fatalf("accept=%+v", accept)
	}
	first, err := handler.ExecuteDirectorBriefingExternalActionV0(ctx, orquestacionnucleoapp.DirectorBriefingExternalActionRequestV0{
		Briefing:      accept,
		Action:        *accept.NextAction,
		OccurredAt:    "2026-06-08T12:30:00Z",
		CorrelationID: "corr-" + run.RunID + "-accept",
		EvidenceRefs:  []string{"evidence-ref-" + run.RunID + "-accept-test"},
	})
	if err != nil {
		t.Fatalf("accept decision first: %v", err)
	}
	acceptedRun := codexStackResidentCouncilRunFromStoreForTestV0(t, ctx, stack, run.RunID)
	decisionRef := codexStackResidentCouncilDecisionRefV0(run.RunID)
	if first.Status != orquestacionnucleoapp.DirectorBriefingExecutionStatusExternalAppliedV0 ||
		!codexStackStringInSetForTestV0(acceptedRun.Decisions, decisionRef) {
		t.Fatalf("first=%+v decisions=%v want %s", first, acceptedRun.Decisions, decisionRef)
	}
	second, err := handler.ExecuteDirectorBriefingExternalActionV0(ctx, orquestacionnucleoapp.DirectorBriefingExternalActionRequestV0{
		Briefing:      accept,
		Action:        *accept.NextAction,
		OccurredAt:    "2026-06-08T12:31:00Z",
		CorrelationID: "corr-" + run.RunID + "-accept-repeat",
		EvidenceRefs:  []string{"evidence-ref-" + run.RunID + "-accept-repeat-test"},
	})
	if err != nil {
		t.Fatalf("accept decision second: %v", err)
	}
	afterRepeat := codexStackResidentCouncilRunFromStoreForTestV0(t, ctx, stack, run.RunID)
	if second.Status != orquestacionnucleoapp.DirectorBriefingExecutionStatusExternalAppliedV0 ||
		len(afterRepeat.Decisions) != len(acceptedRun.Decisions) {
		t.Fatalf("second=%+v before=%v after=%v", second, acceptedRun.Decisions, afterRepeat.Decisions)
	}
}

func TestCodexStackResidentCouncilV0NoProponeAceptarSinVoteSourceV0(t *testing.T) {
	ctx := context.Background()
	stack := mustBuildCodexStackForTestV0(t, newFakeCodexStackRuntimeV0())
	run := codexStackResidentCouncilRunForTestV0(
		"run-resident-council-no-vote-source-001",
		orquestacoreworkflow.OrchestrationPhasePlanificacionMicrotareasV0,
		[]string{"contract:function:decision-council:v0"},
	)
	if err := stack.Stores.RunStore.SaveRunV0(ctx, run); err != nil {
		t.Fatalf("SaveRunV0: %v", err)
	}
	handler := codexStackResidentExternalActionHandlerV0{Ports: stack.Ports}
	if _, err := handler.ExecuteDirectorBriefingExternalActionV0(
		ctx,
		codexStackResidentCouncilExternalActionRequestForTestV0(run.RunID),
	); err != nil {
		t.Fatalf("materialize council: %v", err)
	}
	brainstorm := codexStackResidentCouncilRunFromStoreForTestV0(t, ctx, stack, run.RunID)
	tasks := codexStackResidentCouncilTasksForTestV0(t, ctx, stack, brainstorm)
	brainstorm = codexStackResidentCouncilMarkRolesDeliveredForTestV0(brainstorm, tasks, "p", "c")
	if err := stack.Stores.RunStore.SaveRunV0(ctx, brainstorm); err != nil {
		t.Fatalf("SaveRunV0 brainstorm delivered: %v", err)
	}
	source := codexStackResidentBriefingSourceV0{
		RunStore:  stack.Ports.RunStore,
		TaskStore: stack.Ports.DirectorTaskStore,
	}
	openVote, err := source.BuildResidentDirectorBriefingV0(ctx, orquestacionnucleoapp.ResidentDirectorBriefingBuildRequestV0{
		RunRef:     run.RunID,
		StepNumber: 2,
	})
	if err != nil {
		t.Fatalf("BuildResidentDirectorBriefingV0 open vote: %v", err)
	}
	if openVote.NextAction == nil || openVote.NextAction.Kind != codexStackResidentActionKindOpenCouncilVoteV0 {
		t.Fatalf("openVote=%+v", openVote)
	}
	if _, err := handler.ExecuteDirectorBriefingExternalActionV0(ctx, orquestacionnucleoapp.DirectorBriefingExternalActionRequestV0{
		Briefing:      openVote,
		Action:        *openVote.NextAction,
		OccurredAt:    "2026-06-08T12:40:00Z",
		CorrelationID: "corr-" + run.RunID + "-open-vote",
		EvidenceRefs:  []string{"evidence-ref-" + run.RunID + "-open-vote-test"},
	}); err != nil {
		t.Fatalf("open vote phase: %v", err)
	}
	voteRun := codexStackResidentCouncilRunFromStoreForTestV0(t, ctx, stack, run.RunID)
	voteRun = codexStackResidentCouncilMarkRolesDeliveredForTestV0(voteRun, tasks, "v")
	if err := stack.Stores.RunStore.SaveRunV0(ctx, voteRun); err != nil {
		t.Fatalf("SaveRunV0 votes delivered: %v", err)
	}
	briefing, err := source.BuildResidentDirectorBriefingV0(ctx, orquestacionnucleoapp.ResidentDirectorBriefingBuildRequestV0{
		RunRef:     run.RunID,
		StepNumber: 3,
	})
	if err != nil {
		t.Fatalf("BuildResidentDirectorBriefingV0: %v", err)
	}
	if briefing.NextAction != nil && briefing.NextAction.Kind == codexStackResidentActionKindAcceptCouncilDecisionV0 {
		t.Fatalf("briefing propone accept sin VoteSource: %+v", briefing)
	}
}

func TestCodexStackResidentCouncilHydrateVotesV0NormalizaMetadataDurableV0(t *testing.T) {
	ctx := context.Background()
	stack := mustBuildCodexStackForTestV0(t, newFakeCodexStackRuntimeV0())
	run := codexStackResidentCouncilRunForTestV0(
		"run-resident-council-hydrate-001",
		orquestacoreworkflow.OrchestrationPhasePlanificacionMicrotareasV0,
		[]string{"contract:function:decision-council:v0"},
	)
	if err := stack.Stores.RunStore.SaveRunV0(ctx, run); err != nil {
		t.Fatalf("SaveRunV0: %v", err)
	}
	handler := codexStackResidentExternalActionHandlerV0{Ports: stack.Ports}
	if _, err := handler.ExecuteDirectorBriefingExternalActionV0(
		ctx,
		codexStackResidentCouncilExternalActionRequestForTestV0(run.RunID),
	); err != nil {
		t.Fatalf("materialize council: %v", err)
	}
	materialized := codexStackResidentCouncilRunFromStoreForTestV0(t, ctx, stack, run.RunID)
	tasks := codexStackResidentCouncilTasksForTestV0(t, ctx, stack, materialized)
	plan, err := codexStackResidentCouncilPlanFromRunV0(materialized, []string{"evidence-ref-hydrate-test"})
	if err != nil {
		t.Fatalf("codexStackResidentCouncilPlanFromRunV0: %v", err)
	}
	optionRefs := codexStackResidentCouncilOptionRefsV0(tasks)
	voteTasks := codexStackResidentCouncilTasksByRoleV0(tasks, "v")
	rawVotes := make([]orquestadecisioncouncil.CouncilVoteV0, 0, len(voteTasks))
	for _, task := range voteTasks {
		rawVotes = append(rawVotes, orquestadecisioncouncil.CouncilVoteV0{
			TaskRef:      task.TaskID,
			VoteRef:      "vote-ref-external-" + task.TaskID,
			VoterRef:     "agent-ref-spoof",
			FamilyRef:    "family-ref-spoof",
			OptionRef:    optionRefs[0],
			Position:     orquestadecisioncouncil.CouncilVoteApproveV0,
			EvidenceRefs: []string{"evidence-ref-external-" + task.TaskID},
		})
	}
	hydrated := codexStackResidentCouncilHydrateVotesV0(plan, voteTasks, rawVotes)
	if len(hydrated) != len(voteTasks) {
		t.Fatalf("hydrated=%d want %d", len(hydrated), len(voteTasks))
	}
	for index, vote := range hydrated {
		task := voteTasks[index]
		assignment, ok := codexStackResidentCouncilAssignmentForTaskV0(plan, task)
		if !ok {
			t.Fatalf("assignment missing for %s", task.TaskID)
		}
		if vote.TaskRef != task.TaskID ||
			vote.VoteRef == task.TaskID ||
			vote.VoterRef != assignment.AgentRef ||
			vote.FamilyRef != assignment.FamilyRef ||
			codexStackStringInSetForTestV0(vote.EvidenceRefs, task.TaskID) {
			t.Fatalf("vote=%+v assignment=%+v task=%s", vote, assignment, task.TaskID)
		}
	}
}

func TestCodexStackResidentCouncilHandlerV0NoMaterializaSinContratoFuncionalV0(t *testing.T) {
	ctx := context.Background()
	stack := mustBuildCodexStackForTestV0(t, newFakeCodexStackRuntimeV0())
	run := codexStackResidentCouncilRunForTestV0(
		"run-resident-council-handler-no-contract-001",
		orquestacoreworkflow.OrchestrationPhasePlanificacionMicrotareasV0,
		nil,
	)
	if err := stack.Stores.RunStore.SaveRunV0(ctx, run); err != nil {
		t.Fatalf("SaveRunV0: %v", err)
	}
	handler := codexStackResidentExternalActionHandlerV0{Ports: stack.Ports}

	result, err := handler.ExecuteDirectorBriefingExternalActionV0(
		ctx,
		codexStackResidentCouncilExternalActionRequestForTestV0(run.RunID),
	)
	if err != nil {
		t.Fatalf("ExecuteDirectorBriefingExternalActionV0: %v", err)
	}
	latest := codexStackResidentCouncilRunFromStoreForTestV0(t, ctx, stack, run.RunID)
	if result.Status != orquestacionnucleoapp.DirectorBriefingExecutionStatusExternalPendingV0 ||
		len(codexStackResidentCouncilTaskRefsFromRunForTestV0(latest)) != 0 {
		t.Fatalf("result=%+v run=%+v", result, latest)
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

func codexStackResidentCouncilRunForTestV0(
	runRef string,
	phaseID orquestacoreworkflow.OrchestrationPhaseIDV0,
	functionContracts []string,
) orquestacoreworkflow.OrchestrationRunV0 {
	phases := orquestacoreworkflow.OrchestrationPhaseCatalogV0()
	for index := range phases {
		if phases[index].ID == phaseID {
			phases[index].Status = orquestacoreworkflow.OrchestrationPhaseStatusActiveV0
		}
	}
	return orquestacoreworkflow.OrchestrationRunV0{
		SchemaVersion:     orquestacoreworkflow.OrchestrationRunSchemaVersionV0,
		RunID:             runRef,
		ProjectRef:        "project-ref-resident-council",
		AppSpecRef:        "app-spec-ref-resident-council",
		Status:            orquestacoreworkflow.OrchestrationRunStatusActiveV0,
		CurrentPhase:      phaseID,
		Phases:            phases,
		Brainstorms:       []string{"brainstorm-ref-resident-council-" + runRef},
		Votes:             []string{"vote-ref-resident-council-" + runRef},
		FunctionContracts: append([]string(nil), functionContracts...),
	}
}

func codexStackResidentCouncilExternalActionRequestForTestV0(
	runRef string,
) orquestacionnucleoapp.DirectorBriefingExternalActionRequestV0 {
	return orquestacionnucleoapp.DirectorBriefingExternalActionRequestV0{
		Briefing: orquestadirectorsupervisor.DirectorSupervisorBriefingV0{
			SchemaVersion: orquestadirectorsupervisor.DirectorSupervisorBriefingSchemaV0,
			RunRef:        runRef,
		},
		Action: orquestadirectorsupervisor.DirectorSupervisorRecommendedActionV0{
			ActionRef:        "director-action-" + runRef + "-materialize-council",
			Kind:             codexStackResidentActionKindMaterializeDecisionCouncilV0,
			RunRef:           runRef,
			ReasonCode:       codexStackResidentCouncilReasonV0,
			SafeToApply:      true,
			RequiresDirector: false,
			SourceAction:     orquestadirectorsupervisor.DirectorSupervisorActionContinueV0,
		},
		OccurredAt:    "2026-06-08T12:00:00Z",
		CorrelationID: "corr-" + runRef,
		EvidenceRefs:  []string{"evidence-ref-" + runRef + "-test"},
	}
}

func codexStackResidentCouncilRunFromStoreForTestV0(
	t *testing.T,
	ctx context.Context,
	stack StackV0,
	runRef string,
) orquestacoreworkflow.OrchestrationRunV0 {
	t.Helper()
	run, err := stack.Ports.RunStore.LoadRunV0(ctx, runRef)
	if err != nil {
		t.Fatalf("LoadRunV0: %v", err)
	}
	return run
}

func codexStackResidentCouncilTaskRefsFromRunForTestV0(
	run orquestacoreworkflow.OrchestrationRunV0,
) []string {
	refs := []string{}
	for _, taskRef := range run.Tasks {
		if strings.HasPrefix(taskRef, codexStackResidentCouncilTaskPrefixV0) {
			refs = append(refs, taskRef)
		}
	}
	return refs
}

func codexStackResidentCouncilCandidateTaskRefsForTestV0(
	t *testing.T,
	ctx context.Context,
	stack StackV0,
	run orquestacoreworkflow.OrchestrationRunV0,
) []string {
	t.Helper()
	candidates, err := (orquestacionnucleoapp.WorkflowTaskCandidateProviderV0{
		TaskStore: stack.Ports.DirectorTaskStore,
	}).BuildSchedulerCandidatesV0(ctx, orquestacionnucleoapp.SchedulerCandidateRequestV0{
		Run:           run,
		OccurredAt:    "2026-06-08T12:05:00Z",
		CorrelationID: "corr-" + run.RunID + "-candidates",
	})
	if err != nil {
		t.Fatalf("BuildSchedulerCandidatesV0: %v", err)
	}
	refs := []string{}
	for _, candidate := range candidates.WorkCandidates {
		if candidate.AgentCandidate == nil {
			continue
		}
		refs = append(refs, candidate.AgentCandidate.Payload.TaskRef)
	}
	return compactStringsV0(refs)
}

func codexStackResidentCouncilTasksForTestV0(
	t *testing.T,
	ctx context.Context,
	stack StackV0,
	run orquestacoreworkflow.OrchestrationRunV0,
) []orquestacoreworkflow.WorkflowTaskV0 {
	t.Helper()
	tasks, err := stack.Ports.DirectorTaskStore.LoadWorkflowTasksV0(
		ctx,
		run.RunID,
		codexStackResidentCouncilTaskRefsFromRunForTestV0(run),
	)
	if err != nil {
		t.Fatalf("LoadWorkflowTasksV0: %v", err)
	}
	return tasks
}

func codexStackResidentCouncilMarkRolesDeliveredForTestV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	tasks []orquestacoreworkflow.WorkflowTaskV0,
	roles ...string,
) orquestacoreworkflow.OrchestrationRunV0 {
	roleSet := map[string]bool{}
	for _, role := range roles {
		roleSet[strings.TrimSpace(role)] = true
	}
	for _, task := range tasks {
		if roleSet[codexStackResidentCouncilTaskRoleV0(task)] {
			run.DeliveredTasks = append(run.DeliveredTasks, task.TaskID)
		}
	}
	run.DeliveredTasks = compactStringsV0(run.DeliveredTasks)
	return run
}

func codexStackResidentCouncilHasTaskRefWithPrefixForTestV0(
	refs []string,
	prefix string,
) bool {
	for _, ref := range refs {
		if strings.HasPrefix(ref, prefix) {
			return true
		}
	}
	return false
}

type fakeDecisionCouncilVoteSourceForTestV0 struct{}

func (fakeDecisionCouncilVoteSourceForTestV0) BuildDecisionCouncilVotesV0(
	ctx context.Context,
	request DecisionCouncilVoteBuildRequestV0,
) (DecisionCouncilVoteBuildResultV0, error) {
	if err := ctx.Err(); err != nil {
		return DecisionCouncilVoteBuildResultV0{}, err
	}
	optionRef := ""
	if len(request.OptionRefs) > 0 {
		optionRef = request.OptionRefs[0]
	}
	votes := make([]orquestadecisioncouncil.CouncilVoteV0, 0, len(request.VoteTasks))
	for _, task := range request.VoteTasks {
		votes = append(votes, orquestadecisioncouncil.CouncilVoteV0{
			TaskRef:      task.TaskID,
			VoteRef:      "vote-ref-source-" + task.TaskID,
			VoterRef:     "agent-ref-source-spoof",
			FamilyRef:    "family-ref-source-spoof",
			OptionRef:    optionRef,
			Position:     orquestadecisioncouncil.CouncilVoteApproveV0,
			EvidenceRefs: []string{"evidence-ref-vote-" + task.TaskID},
		})
	}
	return DecisionCouncilVoteBuildResultV0{
		Votes:        votes,
		EvidenceRefs: []string{"evidence-ref-fake-structured-votes"},
	}, nil
}
