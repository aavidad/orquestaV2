package orquestaappcodexstack

import (
	"context"
	"strings"
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
		RunStore:  stack.Ports.RunStore,
		TaskStore: stack.Ports.DirectorTaskStore,
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
