package orquestaappcodexstack

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectoragent "orquesta/modulos/orquesta-director-agent"
	orquestadirectoragentworkflow "orquesta/modulos/orquesta-director-agent-workflow"
	orquestafactory "orquesta/modulos/orquesta-factory"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
)

func TestProjectPlanBootstrapDirectorDecisionSourceV0EmiteBootstrapDesdePlanSinMicrotareas(t *testing.T) {
	run := projectBacklogRunForTestV0()
	run.CurrentPhase = orquestacoreworkflow.OrchestrationPhaseBrainstormingArquitecturaV0
	run.Tasks = nil
	run.FunctionContracts = nil
	run.PhaseArtifacts = []string{"artifact-ref-docs-plan#phase:brainstorming_arquitectura#agent:director"}
	projectDir := filepath.Join(t.TempDir(), "taskboard-lite")
	writeProjectPlanBootstrapPlanForTestV0(t, projectDir)
	source := ProjectPlanBootstrapDirectorDecisionSourceV0{ProjectWorkDir: projectDir}

	decisions, err := source.ListDirectorAgentDecisionsV0(
		context.Background(),
		orquestadirectoragentworkflow.DirectorAgentDecisionSourceRequestV0{
			Run:           run,
			RequestKind:   orquestafactory.RequestKindCrearAppCompletaV0,
			ExecutionMode: orquestafactory.ExecutionModeNormalV0,
		},
	)

	if err != nil {
		t.Fatalf("ListDirectorAgentDecisionsV0: %v", err)
	}
	if len(decisions) != 7 {
		t.Fatalf("decisiones=%d: %+v", len(decisions), decisions)
	}
	for _, decision := range decisions {
		if issues := orquestadirectoragent.ValidateDirectorAgentDecisionV0(decision); len(issues) != 0 {
			t.Fatalf("decision invalida=%+v issues=%+v", decision, issues)
		}
	}
	task, ok := codexStackMicrotaskForPolicyTestV0(decisions, "task-taskboard-lite-bootstrap-vertical")
	if !ok {
		t.Fatalf("bootstrap no emitido: %+v", decisions)
	}
	for _, want := range []string{"go.mod", "cmd/server", "internal", "web", "i18n", "docs", "README.md"} {
		if !codexStackStringInSetForTestV0(task.WriteSet, want) {
			t.Fatalf("write_set=%v missing=%s", task.WriteSet, want)
		}
	}
	if !codexStackStringInSetForTestV0(task.RequiredTests, "go test ./...") ||
		len(task.FunctionContractRefs) != 1 ||
		task.FunctionContractRefs[0].ContractRef != "contract:function:taskboard-lite:bootstrap:v0" {
		t.Fatalf("task bootstrap incompleta=%+v", task)
	}
}

func TestProjectPlanBootstrapDirectorDecisionSourceV0NoEmiteConAgentePendiente(t *testing.T) {
	run := projectBacklogRunForTestV0()
	run.CurrentPhase = orquestacoreworkflow.OrchestrationPhaseBrainstormingArquitecturaV0
	run.Tasks = nil
	run.FunctionContracts = nil
	run.StartedAgents = []string{"agent-ref-director"}
	run.PhaseArtifacts = nil
	projectDir := filepath.Join(t.TempDir(), "taskboard-lite")
	writeProjectPlanBootstrapPlanForTestV0(t, projectDir)
	source := ProjectPlanBootstrapDirectorDecisionSourceV0{ProjectWorkDir: projectDir}

	decisions, err := source.ListDirectorAgentDecisionsV0(
		context.Background(),
		orquestadirectoragentworkflow.DirectorAgentDecisionSourceRequestV0{
			Run:         run,
			RequestKind: orquestafactory.RequestKindCrearAppCompletaV0,
		},
	)

	if err != nil {
		t.Fatalf("ListDirectorAgentDecisionsV0: %v", err)
	}
	if len(decisions) != 0 {
		t.Fatalf("decisiones inesperadas=%+v", decisions)
	}
}

func TestProjectPlanBootstrapDirectorDecisionSourceV0NoEmiteEnRecoveryTecnico(t *testing.T) {
	run := projectBacklogRunForTestV0()
	run.CurrentPhase = orquestacoreworkflow.OrchestrationPhaseBrainstormingArquitecturaV0
	run.Tasks = nil
	run.FunctionContracts = nil
	run.PhaseArtifacts = []string{"artifact-ref-docs-plan#phase:brainstorming_arquitectura#agent:director"}
	projectDir := filepath.Join(t.TempDir(), "taskboard-lite")
	writeProjectPlanBootstrapPlanForTestV0(t, projectDir)
	source := ProjectPlanBootstrapDirectorDecisionSourceV0{ProjectWorkDir: projectDir}

	decisions, err := source.ListDirectorAgentDecisionsV0(
		context.Background(),
		orquestadirectoragentworkflow.DirectorAgentDecisionSourceRequestV0{
			Run:         run,
			RequestKind: orquestafactory.RequestKindCrearAppCompletaV0,
			RequestedBy: codexStackDecisionSourceRecoveryRequestedByV0,
		},
	)

	if err != nil {
		t.Fatalf("ListDirectorAgentDecisionsV0: %v", err)
	}
	if len(decisions) != 0 {
		t.Fatalf("decisiones inesperadas=%+v", decisions)
	}
}

func TestProjectPlanBootstrapDirectorDecisionSourceV0NoEmitePorPalabrasSueltas(t *testing.T) {
	projectDir := filepath.Join(t.TempDir(), "taskboard-lite")
	path := filepath.Join(projectDir, "docs", "plan_tareas.md")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	body := `
# Notas

Este documento menciona backlog, microtarea, implementar, bootstrap, go.mod, api,
web e i18n, pero no publica un backlog ejecutable con tarea T01 y validacion.
`
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	run := projectBacklogRunForTestV0()
	run.CurrentPhase = orquestacoreworkflow.OrchestrationPhaseBrainstormingArquitecturaV0
	run.Tasks = nil
	run.FunctionContracts = nil
	run.PhaseArtifacts = []string{"artifact-ref-docs-plan#phase:brainstorming_arquitectura#agent:director"}
	source := ProjectPlanBootstrapDirectorDecisionSourceV0{ProjectWorkDir: projectDir}

	decisions, err := source.ListDirectorAgentDecisionsV0(
		context.Background(),
		orquestadirectoragentworkflow.DirectorAgentDecisionSourceRequestV0{
			Run:         run,
			RequestKind: orquestafactory.RequestKindCrearAppCompletaV0,
		},
	)

	if err != nil {
		t.Fatalf("ListDirectorAgentDecisionsV0: %v", err)
	}
	if len(decisions) != 0 {
		t.Fatalf("decisiones inesperadas=%+v", decisions)
	}
}

func TestDrainRunV0RescataPlanInicialSinDirectorDecisionsYArrancaProgramacion(t *testing.T) {
	runtime := newFakeCodexStackRuntimeV0().
		withDeliveryBodyForTargetV0("docs/plan_tareas.md", projectPlanBootstrapPlanBodyForTestV0()).
		withDeliveryBodyForTargetV0("docs/plan_microtareas.md", projectPlanBootstrapPlanBodyForTestV0())
	stack := mustBuildCodexStackForTestV0(t, runtime)

	director := postDirectorAPIV0(t, stack)
	if runtime.launchCountV0() != 4 {
		t.Fatalf("launches iniciales=%d want=4", runtime.launchCountV0())
	}
	if _, err := stack.DrainRunV0(context.Background(), DrainRunRequestV0{
		RunRef:               director.RunRef,
		CorrelationID:        "corr-stack-plan-bootstrap-rescue-001",
		MaxBursts:            16,
		MaxStepsPerBurst:     12,
		MaxDispatchesPerWait: 8,
		MaxCommands:          20,
		MaxOutboxPerCycle:    8,
		MaxExternalWaits:     4,
	}); err != nil {
		t.Fatalf("DrainRunV0: %v", err)
	}
	run := mustLoadCodexStackRunForTestV0(t, stack, director.RunRef)
	taskRef := codexStackBootstrapTaskRefForRunV0(run)
	if taskRef == "" {
		t.Fatalf("bootstrap no materializado: tasks=%v phase=%s artifacts=%v", run.Tasks, run.CurrentPhase, run.PhaseArtifacts)
	}
	agentRef := orquestacionnucleoapp.WorkflowTaskAgentRequestRefV0(taskRef)
	if run.CurrentPhase != orquestacoreworkflow.OrchestrationPhaseProgramacionV0 ||
		!codexStackStringInSetForTestV0(run.StartedAgents, agentRef) ||
		runtime.launchCountV0() < 5 {
		t.Fatalf("programacion no arrancada: phase=%s task=%s started=%v launches=%d",
			run.CurrentPhase,
			taskRef,
			run.StartedAgents,
			runtime.launchCountV0(),
		)
	}
}

func writeProjectPlanBootstrapPlanForTestV0(t *testing.T, projectDir string) {
	t.Helper()
	path := filepath.Join(projectDir, "docs", "plan_tareas.md")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(path, []byte(projectPlanBootstrapPlanBodyForTestV0()), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
}

func projectPlanBootstrapPlanBodyForTestV0() string {
	return `
# Plan

## Backlog ejecutable

### T01 bootstrap vertical

- Implementar app Go con go.mod, cmd/server, API, web e i18n.
- Validacion: go test ./...
`
}

func codexStackBootstrapTaskRefForRunV0(run orquestacoreworkflow.OrchestrationRunV0) string {
	for _, taskRef := range run.Tasks {
		if projectBacklogSlugV0(taskRef) != "" &&
			strings.Contains(taskRef, "bootstrap-vertical") {
			return taskRef
		}
	}
	return ""
}
