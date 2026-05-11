package orquestaweb

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirector "orquesta/modulos/orquesta-director"
)

func TestWebBootstrapProyectoViewModelV0CompactaResultadoDirector(t *testing.T) {
	vm := NewWebBootstrapProyectoViewModelV0(bootstrapResultForWebV0())

	if vm.SchemaVersion != WebBootstrapProyectoSchemaV0 ||
		vm.Estado != WebBootstrapProyectoEstadoRegistrado ||
		vm.ProjectRef != "projectref_123" ||
		vm.AppSpecRef != "appspecref_456" ||
		vm.Registro.FasesIniciales != 2 ||
		vm.Registro.Microtareas != 7 ||
		vm.StartRun.CommandType != orquestacoreworkflow.OrchestrationCommandStartRunV0 ||
		len(vm.Workflow.Events) != 1 ||
		vm.Workflow.Events[0].EventType != orquestacoreworkflow.OrchestrationEventRunStartedV0 {
		t.Fatalf("vm=%+v", vm)
	}
	if len(vm.Warnings) != 1 || vm.Warnings[0] != "warning publico" {
		t.Fatalf("warnings=%+v", vm.Warnings)
	}
}

func TestWebBootstrapProyectoViewModelV0NoExponePayloadNiEntradaCompleta(t *testing.T) {
	result := bootstrapResultForWebV0()
	result.StartRunCommand.Payload = json.RawMessage(`{"secret":"no mostrar","app":"Agenda completa"}`)
	result.WorkflowResult.Events[0].Payload = json.RawMessage(`{"token":"no mostrar"}`)

	raw, err := json.Marshal(NewWebBootstrapProyectoViewModelV0(result))
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	serialized := strings.ToLower(string(raw))
	for _, forbidden := range []string{"payload", "secret", "token", "agenda completa"} {
		if strings.Contains(serialized, forbidden) {
			t.Fatalf("viewmodel expone %q: %s", forbidden, serialized)
		}
	}
}

func TestBootstrapProductivosNoImportanFactoryCoreNiAdaptadores(t *testing.T) {
	assertProductFilesAvoidImportsV0(t, "bootstrap_*_v0.go", []string{
		"orquesta/modulos/orquesta-factory",
		"orquesta/modulos/orquesta-core\"",
		"orquesta/modulos/orquesta-mcp",
		"orquesta/modulos/orquesta-cli",
		"database/sql",
		"net/http",
	})
}

func assertProductFilesAvoidImportsV0(t *testing.T, pattern string, forbidden []string) {
	t.Helper()
	files, err := filepath.Glob(pattern)
	if err != nil {
		t.Fatalf("glob %q: %v", pattern, err)
	}
	for _, file := range files {
		if strings.HasSuffix(file, "_test.go") {
			continue
		}
		raw, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("read %s: %v", file, err)
		}
		content := string(raw)
		for _, fragment := range forbidden {
			if strings.Contains(content, fragment) {
				t.Fatalf("%s contiene import/adaptador prohibido %q", file, fragment)
			}
		}
	}
}

func bootstrapResultForWebV0() orquestadirector.BootstrapProyectoDesdeAppSpecResultV0 {
	return orquestadirector.BootstrapProyectoDesdeAppSpecResultV0{
		RegistroAceptado: orquestadirector.BootstrapRegistroAceptadoV0{
			RegistroID:       "reg-1",
			ProjectRef:       "projectref_123",
			AppSpecRef:       "appspecref_456",
			Estado:           "aceptado",
			EventosDominio:   3,
			Warnings:         []string{" warning publico ", "warning publico"},
			FasesIniciales:   2,
			Microtareas:      7,
			Contratos:        4,
			RequestID:        "req-web-1",
			CorrelationID:    "corr-web-1",
			BootstrapVersion: "v0",
		},
		ProjectRef: "projectref_123",
		AppSpecRef: "appspecref_456",
		StartRunCommand: orquestacoreworkflow.OrchestrationCommandV0{
			CommandID:      "cmd-1",
			CommandType:    orquestacoreworkflow.OrchestrationCommandStartRunV0,
			RunID:          "runref_789",
			PayloadVersion: orquestacoreworkflow.OrchestrationCommandPayloadVersionV0,
			OccurredAt:     "2026-05-04T12:10:00Z",
		},
		WorkflowResult: orquestacoreworkflow.OrchestrationCommandResultV0{
			Events: []orquestacoreworkflow.OrchestrationEventV0{{
				EventID:    "evt-1",
				EventType:  orquestacoreworkflow.OrchestrationEventRunStartedV0,
				RunID:      "runref_789",
				Sequence:   1,
				OccurredAt: "2026-05-04T12:10:00Z",
			}},
		},
	}
}
