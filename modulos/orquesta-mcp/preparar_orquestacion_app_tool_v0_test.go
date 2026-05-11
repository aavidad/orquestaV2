package orquestamcp

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestafactory "orquesta/modulos/orquesta-factory"
)

func TestMCPPrepararOrquestacionAppToolV0PreparaPlanGrandeCompacto(t *testing.T) {
	result := ExecuteMCPPrepararOrquestacionAppToolV0(MCPPrepararOrquestacionAppToolInputV0{
		RequestID:     "request-ref-mcp-prepare-001",
		CorrelationID: "corr-mcp-prepare-001",
		AppSpec:       validMCPPrepareLargeAppSpecForTestV0(t),
	})

	if result.Estado != MCPPrepararOrquestacionAppEstadoOKV0 ||
		result.PhaseID != string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0) {
		t.Fatalf("result=%+v", result)
	}
	if result.Plan.Units != 11 || result.Progress.TotalUnits != 11 {
		t.Fatalf("plan=%+v progress=%+v", result.Plan, result.Progress)
	}
	if !mcpPrepareStringInSetTestV0(result.Progress.ReadyTaskRefs, "task-agenda-bootstrap") ||
		!mcpPrepareStringInSetTestV0(result.Progress.BlockedTaskRefs, "task-agenda-architecture") {
		t.Fatalf("progress=%+v", result.Progress)
	}
	if !mcpPrepareHasUnitTestV0(result.Plan.UnitRefs, "task-agenda-persistence-port") {
		t.Fatalf("units=%+v", result.Plan.UnitRefs)
	}
	assertMCPPrepareNoProviderDBTestV0(t, result)
}

func TestMCPPrepararOrquestacionAppToolV0DevuelveErrorPublico(t *testing.T) {
	spec := validMCPPrepareLargeAppSpecForTestV0(t)
	spec.Validation.Estado = "provisional"

	result := ExecuteMCPPrepararOrquestacionAppToolV0(MCPPrepararOrquestacionAppToolInputV0{
		RequestID: "request-ref-mcp-prepare-invalid-001",
		AppSpec:   spec,
	})

	if result.Estado != MCPPrepararOrquestacionAppEstadoErrorV0 || len(result.Errores) != 1 {
		t.Fatalf("result=%+v", result)
	}
	if result.Errores[0].Field != "app_spec.validation.estado" {
		t.Fatalf("errores=%+v", result.Errores)
	}
}

func TestMCPTransportV0PrepararOrquestacionAppSeEjecutaSinPuerto(t *testing.T) {
	transport := newFakeMCPTransportV0()
	if err := RegisterMCPTransportV0(transport, MCPTransportBindingsV0{}); err != nil {
		t.Fatalf("register transport: %v", err)
	}

	output, err := transport.CallToolV0(
		context.Background(),
		MCPPrepararOrquestacionAppToolNameV0,
		MCPPrepararOrquestacionAppToolInputV0{AppSpec: validMCPPrepareLargeAppSpecForTestV0(t)},
	)
	if err != nil {
		t.Fatalf("call prepare tool: %v", err)
	}
	var result MCPPrepararOrquestacionAppToolResultV0
	if err := json.Unmarshal(output, &result); err != nil {
		t.Fatalf("decode result: %v", err)
	}
	if result.Estado != MCPPrepararOrquestacionAppEstadoOKV0 ||
		result.Plan.Units != 11 ||
		len(result.Progress.ReadyTaskRefs) != 1 {
		t.Fatalf("result=%+v", result)
	}
}

func validMCPPrepareLargeAppSpecForTestV0(t *testing.T) orquestafactory.AppSpecV0 {
	t.Helper()
	observability := true
	spec, issues := orquestafactory.SolicitarNuevaAppV0(orquestafactory.AppSpecRequestV0{
		SchemaVersion: orquestafactory.AppSpecRequestSchemaV0,
		RequestID:     "request-ref-mcp-prepare-spec-001",
		Source:        "orquesta-web",
		Locale:        "es-ES",
		Nombre:        "Agenda",
		Objetivo:      "Gestionar contactos y citas desde una API y una web.",
		TipoApp:       "mixed",
		PreferenciasTecnicas: orquestafactory.PreferenciasTecnicasV0{
			Lenguaje:     "go",
			Arquitectura: "hexagonal",
		},
		Datos: orquestafactory.DatosRequestV0{
			DBRequired:         true,
			NecesidadFuncional: "Guardar contactos y citas mediante un puerto de persistencia.",
		},
		Calidad: orquestafactory.CalidadRequestV0{
			Pruebas:        "alta",
			Accesibilidad:  "basica",
			Observabilidad: &observability,
		},
	}, time.Date(2026, 5, 9, 23, 30, 0, 0, time.UTC))
	if len(issues) != 0 {
		t.Fatalf("SolicitarNuevaAppV0 issues: %+v", issues)
	}
	return spec
}

func mcpPrepareHasUnitTestV0(units []MCPAppPlanUnitMCPV0, taskRef string) bool {
	for _, unit := range units {
		if unit.TaskRef == taskRef {
			return true
		}
	}
	return false
}

func mcpPrepareStringInSetTestV0(values []string, expected string) bool {
	for _, value := range values {
		if value == expected {
			return true
		}
	}
	return false
}

func assertMCPPrepareNoProviderDBTestV0(
	t *testing.T,
	result MCPPrepararOrquestacionAppToolResultV0,
) {
	t.Helper()
	payload, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("marshal result: %v", err)
	}
	text := strings.ToLower(string(payload))
	for _, forbidden := range []string{"sqlite", "postgres", "mysql", "mongodb", "redis"} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("result contiene proveedor prohibido %q: %s", forbidden, payload)
		}
	}
}
