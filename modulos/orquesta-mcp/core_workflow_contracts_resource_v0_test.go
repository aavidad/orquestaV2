package orquestamcp

import (
	"encoding/json"
	"strings"
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
)

func TestMCPCoreWorkflowContractsDescriptorV0Compacto(t *testing.T) {
	descriptor := MCPCoreWorkflowContractsDescriptorV0()
	if descriptor.Name != MCPCoreWorkflowContractsResourceNameV0 ||
		descriptor.Version != MCPCoreWorkflowContractsResourceVersionV0 ||
		descriptor.URI != MCPCoreWorkflowContractsResourceURIV0 {
		t.Fatalf("descriptor identidad: %+v", descriptor)
	}
	if descriptor.ContentType != MCPCoreWorkflowContractsContentTypeV0 {
		t.Fatalf("content type inesperado: %+v", descriptor)
	}
	if descriptor.SummaryKey == "" || strings.Contains(descriptor.SummaryKey, " ") {
		t.Fatalf("summary_key debe ser clave compacta: %+v", descriptor)
	}
}

func TestNewMCPCoreWorkflowContractsResourceV0CompactoTrasNCW025(t *testing.T) {
	resource := NewMCPCoreWorkflowContractsResourceV0()
	if resource.URI != MCPCoreWorkflowContractsResourceURIV0 ||
		resource.Version != "v0" ||
		resource.Owner != "orquesta-core-workflow" {
		t.Fatalf("resource identidad: %+v", resource)
	}
	if resource.StateShape.SchemaVersion != "orchestration_run.v0" {
		t.Fatalf("schema de estado inesperado: %+v", resource.StateShape)
	}
	if len(resource.Phases) != 10 || len(resource.Commands) != 32 || len(resource.Events) != 32 {
		t.Fatalf("catalogo incompleto: phases=%d commands=%d events=%d", len(resource.Phases), len(resource.Commands), len(resource.Events))
	}
	assertTransitionNamesMCPTestV0(t, resource.Commands, orquestacoreworkflow.SupportedOrchestrationCommandTypesV0())
	assertTransitionNamesMCPTestV0(t, resource.Events, orquestacoreworkflow.SupportedOrchestrationEventTypesV0())
	if !containsStringMCPTestV0(resource.Phases, "cierre") ||
		!containsTransitionMCPTestV0(resource.Commands, "CloseRun") ||
		!containsTransitionMCPTestV0(resource.Events, "RunClosed") ||
		!containsStringMCPTestV0(resource.StateShape.Refs, "quality_gates") ||
		!containsStringMCPTestV0(resource.StateShape.Refs, "phase_artifacts") ||
		!containsTransitionMCPTestV0(resource.Commands, orquestacoreworkflow.OrchestrationCommandRecordQualityGateV0) ||
		!containsTransitionMCPTestV0(resource.Events, orquestacoreworkflow.OrchestrationEventQualityGateRecordedV0) ||
		!containsTransitionMCPTestV0(resource.Commands, orquestacoreworkflow.OrchestrationCommandRegisterPhaseArtifactV0) ||
		!containsTransitionMCPTestV0(resource.Events, orquestacoreworkflow.OrchestrationEventPhaseArtifactRegisteredV0) {
		t.Fatalf("NCW-025 no reflejado: %+v", resource)
	}
	if len(resource.PublicErrors) < 10 || len(resource.Guardrails) == 0 || len(resource.CanonicalRefs) < 3 {
		t.Fatalf("metadata incompleta: %+v", resource)
	}

	payload, err := json.Marshal(resource)
	if err != nil {
		t.Fatalf("marshal resource: %v", err)
	}
	text := strings.ToLower(string(payload))
	for _, forbidden := range []string{
		"password",
		"bearer ",
		"select *",
		"insert into",
		"/home/",
		"oauth",
		"transcript",
		"filesystem productivo",
	} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("resource contiene detalle no compacto %q: %s", forbidden, text)
		}
	}
	if len(text) > 10000 {
		t.Fatalf("resource demasiado largo: %d bytes", len(text))
	}
}

func TestMCPCoreWorkflowTransitionByNameV0NormalizaEntradas(t *testing.T) {
	cases := map[string]string{
		"CloseRun":          "command",
		"command/close_run": "CloseRun",
		"mcp.core_workflow.command.close_run.summary.v0": "CloseRun",
		"RunClosed":                 "event",
		"event/run-closed":          "RunClosed",
		"record quality gate":       "RecordQualityGate",
		"register phase artifact":   "RegisterPhaseArtifact",
		"PhaseArtifactRegistered":   "event",
		"QualityGateRecorded":       "event",
		"register final validation": "RegisterFinalValidation",
	}

	for input, wantOneOf := range cases {
		got, ok := MCPCoreWorkflowTransitionByNameV0(input)
		if !ok {
			t.Fatalf("no encontro transicion para %q", input)
		}
		if got.Kind != wantOneOf && got.Name != wantOneOf {
			t.Fatalf("transicion para %q: %+v", input, got)
		}
	}

	if _, ok := MCPCoreWorkflowTransitionByNameV0("runtime real"); ok {
		t.Fatalf("entrada inexistente no debe encontrarse")
	}
}

func assertTransitionNamesMCPTestV0(t *testing.T, guides []MCPCoreWorkflowTransitionGuideV0, want []string) {
	t.Helper()
	for _, name := range want {
		if !containsTransitionMCPTestV0(guides, name) {
			t.Fatalf("catalogo MCP no incluye transicion soportada por core: %q", name)
		}
	}
}

func containsTransitionMCPTestV0(values []MCPCoreWorkflowTransitionGuideV0, name string) bool {
	for _, value := range values {
		if value.Name == name {
			return true
		}
	}
	return false
}

func containsStringMCPTestV0(values []string, needle string) bool {
	for _, value := range values {
		if value == needle {
			return true
		}
	}
	return false
}
