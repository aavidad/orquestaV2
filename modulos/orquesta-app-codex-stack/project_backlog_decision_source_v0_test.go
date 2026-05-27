package orquestaappcodexstack

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectoragentworkflow "orquesta/modulos/orquesta-director-agent-workflow"
	orquestafactory "orquesta/modulos/orquesta-factory"
)

func TestProjectBacklogDirectorDecisionSourceV0EmiteSiguienteOlaConDependenciasSatisfechas(t *testing.T) {
	projectDir := filepath.Join(t.TempDir(), "resigrx")
	writeProjectBacklogForTestV0(t, projectDir, `
# Backlog

## RX-000 bootstrap-repo

Objetivo: crear base.

Salida:

- go.mod

Validacion:

- go test ./...

## RX-001 config-i18n-observability

Objetivo: implementar configuracion e i18n.

Dependencias: RX-000.

Salida:

- paquete internal/platform/config

Validacion:

- tests de config

## RX-002 openapi-contract

Objetivo: publicar OpenAPI vivo.

Dependencias: RX-000.

Salida:

- docs/openapi.yaml

Validacion:

- OpenAPI parseable

## RX-003 tenancy-identity-base

Objetivo: implementar tenant.

Dependencias: RX-001, RX-002.
`)
	run := projectBacklogRunForTestV0()
	run.Tasks = []string{"task-resigrx-rx000-bootstrap-vertical-mvp"}
	run.DeliveredTasks = []string{"task-resigrx-rx000-bootstrap-vertical-mvp"}
	source := ProjectBacklogDirectorDecisionSourceV0{ProjectWorkDir: projectDir}

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
	if len(decisions) != 2 {
		t.Fatalf("decisiones=%d: %+v", len(decisions), decisions)
	}
	rx001, ok := codexStackMicrotaskForPolicyTestV0(decisions, "task-resigrx-rx001-config-i18n-observability")
	if !ok {
		t.Fatalf("no se emitio RX-001: %+v", decisions)
	}
	if len(rx001.DependsOn) != 1 || rx001.DependsOn[0] != "task-resigrx-rx000-bootstrap-vertical-mvp" {
		t.Fatalf("depends_on RX-001=%v", rx001.DependsOn)
	}
	if len(rx001.FunctionContractRefs) != 1 ||
		rx001.FunctionContractRefs[0].ContractRef != "contract-ref-resigrx-app-v0" ||
		rx001.MaxChildAgents != 6 {
		t.Fatalf("metadata RX-001=%+v", rx001)
	}
	if _, ok := codexStackMicrotaskForPolicyTestV0(decisions, "task-resigrx-rx003-tenancy-identity-base"); ok {
		t.Fatalf("RX-003 no debe salir hasta completar RX-001 y RX-002")
	}
}

func TestProjectBacklogDirectorDecisionSourceV0NoEmiteConAgentePendiente(t *testing.T) {
	projectDir := filepath.Join(t.TempDir(), "resigrx")
	writeProjectBacklogForTestV0(t, projectDir, `
# Backlog

## RX-001 config-i18n-observability

Objetivo: implementar configuracion.

Dependencias: RX-000.
`)
	run := projectBacklogRunForTestV0()
	run.Tasks = []string{"task-resigrx-rx000-bootstrap"}
	run.DeliveredTasks = []string{"task-resigrx-rx000-bootstrap"}
	run.StartedAgents = []string{"agent-ref-live-001"}
	source := ProjectBacklogDirectorDecisionSourceV0{ProjectWorkDir: projectDir}

	decisions, err := source.ListDirectorAgentDecisionsV0(
		context.Background(),
		orquestadirectoragentworkflow.DirectorAgentDecisionSourceRequestV0{Run: run},
	)

	if err != nil {
		t.Fatalf("ListDirectorAgentDecisionsV0: %v", err)
	}
	if len(decisions) != 0 {
		t.Fatalf("decisiones inesperadas=%+v", decisions)
	}
}

func TestCompositeDirectorDecisionSourceV0AceptaBacklogGoIncremental(t *testing.T) {
	projectDir := filepath.Join(t.TempDir(), "resigrx")
	writeProjectBacklogForTestV0(t, projectDir, `
# Backlog

## RX-001 config-i18n-observability

Objetivo: implementar configuracion e i18n para ResiGRX.

Dependencias: RX-000.

Salida:

- paquete internal/platform/config
`)
	run := projectBacklogRunForTestV0()
	run.Tasks = []string{"task-resigrx-rx000-bootstrap-vertical-mvp"}
	run.DeliveredTasks = []string{"task-resigrx-rx000-bootstrap-vertical-mvp"}
	source := compositeDirectorDecisionSourceV0{
		Sources: []orquestadirectoragentworkflow.DirectorAgentDecisionSourcePortV0{
			ProjectBacklogDirectorDecisionSourceV0{ProjectWorkDir: projectDir},
		},
	}

	decisions, err := source.ListDirectorAgentDecisionsV0(
		context.Background(),
		orquestadirectoragentworkflow.DirectorAgentDecisionSourceRequestV0{
			Run:            run,
			RequestKind:    orquestafactory.RequestKindCrearAppCompletaV0,
			ExecutionMode:  orquestafactory.ExecutionModeNormalV0,
			ObjectiveHints: []string{"Crear app completa ResiGRX"},
		},
	)

	if err != nil {
		t.Fatalf("ListDirectorAgentDecisionsV0: %v", err)
	}
	if _, ok := codexStackMicrotaskForPolicyTestV0(decisions, "task-resigrx-rx001-config-i18n-observability"); !ok {
		t.Fatalf("no se emitio RX-001: %+v", decisions)
	}
}

func TestProjectBacklogDependencyIDsV0ExpandeRangoFinal(t *testing.T) {
	got := projectBacklogDependencyIDsV0("RX-000 a RX-003")
	want := []string{"RX000", "RX001", "RX002", "RX003"}
	if len(got) != len(want) {
		t.Fatalf("deps=%v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("deps=%v", got)
		}
	}
}

func writeProjectBacklogForTestV0(t *testing.T, projectDir string, content string) {
	t.Helper()
	inputDir := filepath.Join(projectDir, "orquesta_input")
	if err := os.MkdirAll(inputDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(filepath.Join(inputDir, "backlog_orquesta_resigrx.md"), []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
}

func projectBacklogRunForTestV0() orquestacoreworkflow.OrchestrationRunV0 {
	phases := orquestacoreworkflow.OrchestrationPhaseCatalogV0()
	for i := range phases {
		if phases[i].ID == orquestacoreworkflow.OrchestrationPhaseProgramacionV0 {
			phases[i].Status = orquestacoreworkflow.OrchestrationPhaseStatusActiveV0
		}
	}
	return orquestacoreworkflow.OrchestrationRunV0{
		SchemaVersion:     orquestacoreworkflow.OrchestrationRunSchemaVersionV0,
		RunID:             "run-ref-resigrx-001",
		ProjectRef:        "project-ref-resigrx",
		AppSpecRef:        "app-spec-ref-resigrx",
		Status:            orquestacoreworkflow.OrchestrationRunStatusActiveV0,
		CurrentPhase:      orquestacoreworkflow.OrchestrationPhaseProgramacionV0,
		Phases:            phases,
		FunctionContracts: []string{"contract-ref-resigrx-app-v0"},
	}
}
