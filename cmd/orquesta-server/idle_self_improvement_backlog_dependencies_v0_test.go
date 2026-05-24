package main

import (
	"os"
	"path/filepath"
	"testing"

	orquestaserver "orquesta/modulos/orquesta-server"
)

func TestIdleSelfImprovementBacklogPlannerV0RespetaDependenciasNoCompletadasV0(t *testing.T) {
	projectDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(projectDir, "docs"), 0o700); err != nil {
		t.Fatalf("mkdir docs: %v", err)
	}
	content := `# Backlog

## T01 base-runtime

Objetivo: crear base runtime.

## T02 connector-claude

Objetivo: conectar Claude sobre la base runtime.

Dependencias:

- t01-base-runtime

## T03 director-cleanup

Objetivo: mejorar director.
`
	if err := os.WriteFile(filepath.Join(projectDir, idleSelfImprovementBacklogDocRelV0), []byte(content), 0o600); err != nil {
		t.Fatalf("write backlog: %v", err)
	}

	result, err := (idleSelfImprovementBacklogPlannerV0{ProjectWorkDir: projectDir}).PlanV0(
		orquestaserver.IdleSelfImprovementPlanRequestV0{
			MaxRequests: 3,
			BaseRequest: orquestaserver.IdleSelfImprovementRequestV0{
				RequestRef:    "request-ref-base",
				CorrelationID: "corr-request-ref-base",
				WriteSet:      []string{"cmd/orquesta-server"},
				RequiredTests: []string{"go test -count=1 ./cmd/orquesta-server"},
			},
		},
	)
	if err != nil {
		t.Fatalf("PlanV0: %v", err)
	}
	for _, request := range result.Requests {
		if request.SuggestedArea == "t02-connector-claude" {
			t.Fatalf("no debe planificar dependiente antes de completar dependencia: %+v", result.Requests)
		}
	}
}

func TestIdleSelfImprovementBacklogPlannerV0PlanificaDependienteSiDependenciaCompletadaV0(t *testing.T) {
	projectDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(projectDir, "docs"), 0o700); err != nil {
		t.Fatalf("mkdir docs: %v", err)
	}
	content := `# Backlog

## T01 base-runtime

Objetivo: crear base runtime.

Estado: completada.

## T02 connector-claude

Objetivo: conectar Claude sobre la base runtime.

Dependencias: t01-base-runtime

Entrada:

- contrato runtime neutral completado por T01.

Salida:

- conector Claude cableado al runtime neutral.
`
	if err := os.WriteFile(filepath.Join(projectDir, idleSelfImprovementBacklogDocRelV0), []byte(content), 0o600); err != nil {
		t.Fatalf("write backlog: %v", err)
	}

	result, err := (idleSelfImprovementBacklogPlannerV0{ProjectWorkDir: projectDir}).PlanV0(
		orquestaserver.IdleSelfImprovementPlanRequestV0{
			MaxRequests: 1,
			BaseRequest: orquestaserver.IdleSelfImprovementRequestV0{
				RequestRef:    "request-ref-base",
				CorrelationID: "corr-request-ref-base",
				WriteSet:      []string{"cmd/orquesta-server"},
				RequiredTests: []string{"go test -count=1 ./cmd/orquesta-server"},
			},
		},
	)
	if err != nil {
		t.Fatalf("PlanV0: %v", err)
	}
	if len(result.Requests) != 1 || result.Requests[0].SuggestedArea != "t02-connector-claude" {
		t.Fatalf("requests=%+v", result.Requests)
	}
	contextRefs := result.Requests[0].ContextRefs
	for _, want := range []string{
		"backlog_dependency:t01-base-runtime",
		"backlog_input:contrato runtime neutral completado por T01.",
		"backlog_output:conector Claude cableado al runtime neutral.",
	} {
		if !containsStringForTestV0(contextRefs, want) {
			t.Fatalf("context_refs no contiene %q: %+v", want, contextRefs)
		}
	}
}

func TestIdleSelfImprovementBacklogPlannerV0PlanificaDependienteConContratoAunqueDependenciaNoCompletadaV0(t *testing.T) {
	projectDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(projectDir, "docs"), 0o700); err != nil {
		t.Fatalf("mkdir docs: %v", err)
	}
	content := `# Backlog

## T01 runtime-contract

Objetivo: estabilizar contrato runtime neutral.

## T02 claude-adapter

Objetivo: crear adaptador Claude contra contrato declarado.

Dependencias: t01-runtime-contract

Entrada:

- puerto AgentRuntimeV0 con Launch/Observe documentados.

Salida:

- adaptador Claude opt-in sin imports en nucleo.
`
	if err := os.WriteFile(filepath.Join(projectDir, idleSelfImprovementBacklogDocRelV0), []byte(content), 0o600); err != nil {
		t.Fatalf("write backlog: %v", err)
	}

	result, err := (idleSelfImprovementBacklogPlannerV0{ProjectWorkDir: projectDir}).PlanV0(
		orquestaserver.IdleSelfImprovementPlanRequestV0{
			MaxRequests: 2,
			BaseRequest: orquestaserver.IdleSelfImprovementRequestV0{
				RequestRef:    "request-ref-base",
				CorrelationID: "corr-request-ref-base",
				WriteSet:      []string{"cmd/orquesta-server"},
				RequiredTests: []string{"go test -count=1 ./cmd/orquesta-server"},
			},
		},
	)
	if err != nil {
		t.Fatalf("PlanV0: %v", err)
	}
	var dependent orquestaserver.IdleSelfImprovementRequestV0
	for _, request := range result.Requests {
		if request.SuggestedArea == "t02-claude-adapter" {
			dependent = request
		}
	}
	if dependent.RequestRef == "" {
		t.Fatalf("no planifico dependiente con contrato suficiente: %+v", result.Requests)
	}
	for _, want := range []string{
		"backlog_dependency:t01-runtime-contract",
		"backlog_dependency_contract_ready:t01-runtime-contract",
		"backlog_input:puerto AgentRuntimeV0 con Launch/Observe documentados.",
		"backlog_output:adaptador Claude opt-in sin imports en nucleo.",
	} {
		if !containsStringForTestV0(dependent.ContextRefs, want) {
			t.Fatalf("context_refs no contiene %q: %+v", want, dependent.ContextRefs)
		}
	}
	if !containsStringForTestV0(dependent.AcceptanceCriteria, "si alguna dependencia sigue abierta, programar solo contra las entradas/salidas declaradas y dejar el ajuste final como tarea separada si cambia el contrato") {
		t.Fatalf("acceptance_criteria=%+v", dependent.AcceptanceCriteria)
	}
}
