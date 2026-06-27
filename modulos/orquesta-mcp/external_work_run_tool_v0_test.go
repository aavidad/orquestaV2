package orquestamcp

import (
	"context"
	"strings"
	"testing"

	orquestaappchange "orquesta/modulos/orquesta-app-change"
	orquestaexternalworkrun "orquesta/modulos/orquesta-external-work-run"
)

func TestMCPExternalWorkRunToolExecutorV0RechazaEntradaAmbigua(t *testing.T) {
	result, err := MCPExternalWorkRunToolExecutorV0{}.Execute(
		context.Background(),
		MCPExternalWorkRunToolInputV0{
			RequestID: "req-ambigua",
			ExternalWorkRunRequest: orquestaexternalworkrun.StartExternalWorkRunRequestV0{
				AppChangeRequest: orquestaappchange.AppChangeRequestV0{
					ChangeRef: "nested-change",
				},
			},
			AppChangeRequest: orquestaappchange.AppChangeRequestV0{
				ChangeRef: "top-change",
			},
		},
	)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if result.Estado != MCPExternalWorkRunEstadoErrorV0 ||
		len(result.Errores) != 1 ||
		result.Errores[0].Code != MCPExternalWorkRunInputAmbiguousV0 {
		t.Fatalf("result=%+v", result)
	}
}

func TestMCPExternalWorkRunDescriptorV0DeclaraLegacyExplicito(t *testing.T) {
	descriptor := MCPExternalWorkRunDescriptorV0()

	if !mcpExternalWorkRunStringContainsTestV0(descriptor.Output, "route_policy") ||
		!mcpExternalWorkRunStringContainsTestV0(descriptor.Output, "director_execution_mode") {
		t.Fatalf("output no declara politica legacy: %s", descriptor.Output)
	}
	for _, want := range []string{
		"goal-first es la ruta normal para trabajo externo nuevo",
		"sin backend Goal completo la composicion goal-first devuelve error operativo y no degrada a legacy",
		"legacy solo en composicion de compatibilidad opt-in",
		"con backend Goal completo la composicion puede devolver route_policy=goal_first y goal_ref",
		"la ruta legacy opt-in crea run operativo y encola para loop historico por puertos inyectados",
	} {
		if !mcpExternalWorkRunStringInSetTestV0(descriptor.Invariantes, want) {
			t.Fatalf("invariante %q no encontrada: %+v", want, descriptor.Invariantes)
		}
	}
}

func TestNewMCPExternalWorkRunResultV0MarcaLegacyDirectorLoop(t *testing.T) {
	result := newMCPExternalWorkRunResultV0(orquestaexternalworkrun.StartExternalWorkRunResultV0{
		Status:    orquestaexternalworkrun.ExternalWorkRunStatusAcceptedV0,
		RequestID: "req-external-result-legacy-001",
		RunRef:    "run-external-result-legacy-001",
	})

	if result.Estado != MCPExternalWorkRunEstadoOKV0 ||
		result.RoutePolicy != MCPExternalWorkRunRoutePolicyLegacyDirectorLoopV0 ||
		result.DirectorExecutionMode != MCPExternalWorkRunDirectorExecutionModeLegacyLoopV0 {
		t.Fatalf("result=%+v", result)
	}
	if !mcpExternalWorkRunStringInSetTestV0(result.NextActions, MCPExternalWorkRunNextActionSuperviseLegacyRunV0) ||
		!mcpExternalWorkRunStringInSetTestV0(result.NextActions, MCPExternalWorkRunNextActionMigrateGoalFirstV0) {
		t.Fatalf("next_actions=%+v", result.NextActions)
	}
}

func mcpExternalWorkRunStringInSetTestV0(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func mcpExternalWorkRunStringContainsTestV0(value string, want string) bool {
	return strings.Contains(value, want)
}
