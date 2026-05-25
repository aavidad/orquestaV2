package orquestaappcodexstack

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"

	orquestaautoprogramming "orquesta/modulos/orquesta-autoprogramming"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectoroperativo "orquesta/modulos/orquesta-director-operativo"
	orquestamcp "orquesta/modulos/orquesta-mcp"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
)

func TestCodexStackAutoprogrammingPrepareRunAPIV0CierraConPlanStateYTestsRealesV0(t *testing.T) {
	ctx := context.Background()
	cfg := codexStackRequiredTestLocalConfigV0(t)
	writeCodexStackRequiredTestTinyGoModuleV0(t, cfg.ProjectWorkDir)
	goCommand := codexStackRequiredTestGoCommandV0(t)
	outputDir := filepath.Join(t.TempDir(), "required-test-output")
	runtime := newFakeCodexStackRuntimeV0().withDeliveryBodyForTargetV0(
		"README.md",
		"# Entrega autoprogramming\n\nCambio acotado para probar cierre causal.\n",
	)
	evidenceStore := orquestacionnucleoapp.NewInMemoryRequiredTestEvidenceStoreV0()
	stack := codexStackRealRequiredTestRunnerStackV0(t, cfg, runtime, evidenceStore, goCommand, outputDir)

	prepared := postAutoprogrammingPrepareRunStackV0(t, stack, orquestamcp.MCPAutoprogrammingPrepareRunToolInputV0{
		RequestID:              "request-autoprogramming-closure-flow-001",
		CorrelationID:          "corr-autoprogramming-closure-flow-001",
		OccurredAt:             "2026-05-23T18:00:00Z",
		RequestedBy:            "orquesta-app-stack-test",
		AutoprogrammingRequest: autoprogrammingClosureRequestForTestV0(),
		MaxBursts:              8,
		MaxStepsPerBurst:       8,
		MaxDispatchesPerWait:   4,
		MaxCommands:            16,
		MaxOutboxPerCycle:      8,
	})
	if !prepared.Accepted ||
		prepared.Continue == nil ||
		prepared.Continue.OperationalDirectorPlanRef == "" ||
		len(prepared.WorkflowTaskRefs) != 1 ||
		len(prepared.WaitAgentRefs) != 1 {
		t.Fatalf("prepared=%+v", prepared)
	}

	var last orquestamcp.MCPRunSupervisorToolResultV0
	for cycle := 1; cycle <= 16; cycle++ {
		last = postRunSupervisorStackV0(t, stack, orquestamcp.MCPRunSupervisorToolInputV0{
			RequestID:                  fmt.Sprintf("request-autoprogramming-closure-supervisor-%03d", cycle),
			CorrelationID:              fmt.Sprintf("corr-autoprogramming-closure-supervisor-%03d", cycle),
			RunRef:                     prepared.RunRef,
			OperationalDirectorPlanRef: prepared.Continue.OperationalDirectorPlanRef,
			MaxTicks:                   1,
			MaxRunsPerTick:             1,
			MaxExecutions:              1,
			MaxBursts:                  12,
			MaxStepsPerBurst:           12,
			MaxDispatchesPerWait:       6,
			MaxCommands:                32,
			MaxOutboxPerCycle:          16,
			MaxDecisionCycles:          8,
			MaxExternalWaits:           1,
			AllowRepeatedRuns:          true,
		})
		if last.Estado != orquestamcp.MCPRunSupervisorEstadoOKV0 {
			t.Fatalf("supervisor cycle=%d result=%+v", cycle, last)
		}
		run := mustLoadCodexStackRunForTestV0(t, stack, prepared.RunRef)
		state := autoprogrammingClosurePlanStateForTestV0(t, stack, prepared.RunRef, prepared.Continue.OperationalDirectorPlanRef)
		if run.Status == orquestacoreworkflow.OrchestrationRunStatusClosedV0 &&
			state.Status == orquestacionnucleoapp.OperationalDirectorPlanStateClosedV0 {
			break
		}
	}

	run := mustLoadCodexStackRunForTestV0(t, stack, prepared.RunRef)
	state := autoprogrammingClosurePlanStateForTestV0(t, stack, prepared.RunRef, prepared.Continue.OperationalDirectorPlanRef)
	replanStep := codexStackRequiredTestPlanStepV0(t, state, "step-replan-or-close")
	if run.Status != orquestacoreworkflow.OrchestrationRunStatusClosedV0 ||
		state.Status != orquestacionnucleoapp.OperationalDirectorPlanStateClosedV0 ||
		len(run.ClosedTasks) != 1 ||
		len(run.Validations) != 1 ||
		len(run.Closures) != 1 ||
		replanStep.Status != orquestadirectoroperativo.OperationalDirectorStepClosedV0 ||
		len(replanStep.RequiredTestEvidenceRefs) != 1 {
		t.Fatalf("no cerro autoprogramming: run=%+v state=%+v replan=%+v last=%+v", run, state, replanStep, last)
	}
	evidence, err := evidenceStore.LoadRequiredTestEvidenceV0(ctx, prepared.RunRef, replanStep.RequiredTestEvidenceRefs)
	if err != nil {
		t.Fatalf("LoadRequiredTestEvidenceV0: %v", err)
	}
	if len(evidence) != 1 ||
		evidence[0].Status != orquestacionnucleoapp.RequiredTestEvidenceStatusPassedV0 ||
		evidence[0].TaskRef != prepared.WorkflowTaskRefs[0] ||
		evidence[0].TestCommand != "go test ./..." {
		t.Fatalf("evidence invalida: %+v prepared=%+v", evidence, prepared)
	}
	if runtime.launchCountV0() != 1 ||
		!codexStackStringInSetForTestV0(run.DeliveredTasks, prepared.WorkflowTaskRefs[0]) ||
		!codexStackStringInSetForTestV0(run.DeliveredAgents, prepared.WaitAgentRefs[0]) {
		t.Fatalf("entrega/agente inesperado: launches=%d delivered_tasks=%v delivered_agents=%v prepared=%+v",
			runtime.launchCountV0(),
			run.DeliveredTasks,
			run.DeliveredAgents,
			prepared,
		)
	}
}

func TestCodexStackAutoprogrammingSupervisorGlobalCierraDerivandoPlanStateV0(t *testing.T) {
	cfg := codexStackRequiredTestLocalConfigV0(t)
	writeCodexStackRequiredTestTinyGoModuleV0(t, cfg.ProjectWorkDir)
	goCommand := codexStackRequiredTestGoCommandV0(t)
	outputDir := filepath.Join(t.TempDir(), "required-test-output")
	runtime := newFakeCodexStackRuntimeV0().withDeliveryBodyForTargetV0(
		"README.md",
		"# Entrega autoprogramming\n\nCambio acotado desde cola global.\n",
	)
	evidenceStore := orquestacionnucleoapp.NewInMemoryRequiredTestEvidenceStoreV0()
	stack := codexStackRealRequiredTestRunnerStackV0(t, cfg, runtime, evidenceStore, goCommand, outputDir)

	prepared := postAutoprogrammingPrepareRunStackV0(t, stack, orquestamcp.MCPAutoprogrammingPrepareRunToolInputV0{
		RequestID:              "request-autoprogramming-global-closure-flow-001",
		CorrelationID:          "corr-autoprogramming-global-closure-flow-001",
		OccurredAt:             "2026-05-23T18:30:00Z",
		RequestedBy:            "orquesta-app-stack-test",
		AutoprogrammingRequest: autoprogrammingClosureRequestForTestV0(),
		MaxBursts:              8,
		MaxStepsPerBurst:       8,
		MaxDispatchesPerWait:   4,
		MaxCommands:            16,
		MaxOutboxPerCycle:      8,
	})
	if !prepared.Accepted || prepared.RunRef == "" || prepared.Continue == nil ||
		prepared.Continue.OperationalDirectorPlanRef == "" {
		t.Fatalf("prepared=%+v", prepared)
	}
	planRef := prepared.Continue.OperationalDirectorPlanRef

	for cycle := 1; cycle <= 16; cycle++ {
		result := postRunSupervisorStackV0(t, stack, orquestamcp.MCPRunSupervisorToolInputV0{
			RequestID:            fmt.Sprintf("request-autoprogramming-global-closure-supervisor-%03d", cycle),
			CorrelationID:        fmt.Sprintf("corr-autoprogramming-global-closure-supervisor-%03d", cycle),
			QueueRef:             DefaultRunQueueRefV0,
			MaxTicks:             1,
			MaxRunsPerTick:       1,
			MaxExecutions:        1,
			MaxBursts:            12,
			MaxStepsPerBurst:     12,
			MaxDispatchesPerWait: 6,
			MaxCommands:          32,
			MaxOutboxPerCycle:    16,
			MaxDecisionCycles:    8,
			MaxExternalWaits:     1,
			AllowRepeatedRuns:    true,
		})
		if result.Estado != orquestamcp.MCPRunSupervisorEstadoOKV0 {
			t.Fatalf("supervisor cycle=%d result=%+v", cycle, result)
		}
		run := mustLoadCodexStackRunForTestV0(t, stack, prepared.RunRef)
		state := autoprogrammingClosurePlanStateForTestV0(t, stack, prepared.RunRef, planRef)
		if run.Status == orquestacoreworkflow.OrchestrationRunStatusClosedV0 &&
			state.Status == orquestacionnucleoapp.OperationalDirectorPlanStateClosedV0 {
			break
		}
	}

	run := mustLoadCodexStackRunForTestV0(t, stack, prepared.RunRef)
	state := autoprogrammingClosurePlanStateForTestV0(t, stack, prepared.RunRef, planRef)
	if run.Status != orquestacoreworkflow.OrchestrationRunStatusClosedV0 ||
		state.Status != orquestacionnucleoapp.OperationalDirectorPlanStateClosedV0 ||
		len(run.ClosedTasks) != 1 ||
		len(run.Validations) != 1 ||
		len(run.Closures) != 1 {
		t.Fatalf("global no cerro autoprogramming: run=%+v state=%+v", run, state)
	}
	if runtime.launchCountV0() != 1 {
		t.Fatalf("launches=%d", runtime.launchCountV0())
	}
}

func autoprogrammingClosureRequestForTestV0() orquestaautoprogramming.AutoprogrammingRequestV0 {
	return orquestaautoprogramming.AutoprogrammingRequestV0{
		RequestRef:       "run-autoprogramming-closure-flow-001",
		ProjectRef:       "project-ref-autoprogramming-closure-flow-001",
		WorktreeRef:      "worktree-ref-autoprogramming-closure-flow-001",
		WorktreeIsolated: true,
		BranchRef:        "branch-ref-autoprogramming-closure-flow-001",
		Tasks: []orquestaautoprogramming.AutoprogrammingTaskGroupCandidateV0{{
			TaskRef:            "source-task-ref-autoprogramming-closure-flow-001",
			Area:               "docs",
			Objective:          "Actualizar documentacion ligera sin tocar el modulo Go de prueba.",
			AcceptanceCriteria: []string{"README actualizado y go test ./... pasado"},
		}},
		WriteSet:      []string{"README.md"},
		RequiredTests: []string{"go test ./..."},
	}
}

func autoprogrammingClosurePlanStateForTestV0(
	t *testing.T,
	stack StackV0,
	runRef string,
	planRef string,
) orquestacionnucleoapp.OperationalDirectorPlanStateV0 {
	t.Helper()
	state, err := stack.Stores.OperationalPlanStateStore.LoadOperationalDirectorPlanStateV0(
		context.Background(),
		runRef,
		planRef,
	)
	if err != nil {
		t.Fatalf("LoadOperationalDirectorPlanStateV0: %v", err)
	}
	return state
}
