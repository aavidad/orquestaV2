package orquestamcp

import (
	"encoding/json"
	"strings"
	"testing"

	orquestacore "orquesta/modulos/orquesta-core"
	orquestaobservability "orquesta/modulos/orquesta-observability"
	operator "orquesta/modulos/orquesta-operator-mcp"
)

func TestMCPTransportResourcesV0DeclaranFuenteDescriptorVigente(t *testing.T) {
	resources := MCPTransportResourcesV0()
	if len(resources) == 0 {
		t.Fatal("resources vacios")
	}
	for _, resource := range resources {
		if resource.OutputBudget.MaxBytes <= 0 ||
			resource.OutputBudget.Overflow != MCPTransportResourcePayloadBlockedV0 ||
			resource.OutputBudget.Freshness == "" {
			t.Fatalf("%s output budget incompleto: %+v", resource.Name, resource.OutputBudget)
		}
		if !ValidateMCPResourceDescriptorSourceV0(resource.DescriptorSource) {
			t.Fatalf("%s descriptor_source invalido: %+v", resource.Name, resource.DescriptorSource)
		}
		if resource.DescriptorSource.Freshness != resource.OutputBudget.Freshness {
			t.Fatalf("%s freshness divergente: %+v/%+v", resource.Name, resource.DescriptorSource, resource.OutputBudget)
		}
		payload, err := json.Marshal(resource.DescriptorSource)
		if err != nil {
			t.Fatalf("%s marshal descriptor_source: %v", resource.Name, err)
		}
		assertNoSensitiveDescriptorSourceMCPTestV0(t, string(payload))
	}
}

func TestMCPResourceDescriptorsV0MantienenParidadConOwners(t *testing.T) {
	operational := NewMCPOperationalStatusResourceV0()
	if !containsDescriptorStringMCPTestV0(operational.AllowedScopes, orquestaobservability.OperationalStatusScopeSistemaV0) ||
		!containsDescriptorStringMCPTestV0(operational.PublicErrors, orquestaobservability.ErrProyeccionNoDisponibleV0) {
		t.Fatalf("operational status sin constantes owner: %+v", operational)
	}

	timeline := NewMCPWorkspaceTimelineResourceV0()
	if !containsDescriptorStringMCPTestV0(timeline.AllowedSources, orquestaobservability.WorkspaceTimelineSourceDirectorStatsV0) ||
		!containsDescriptorStringMCPTestV0(timeline.PublicErrors, orquestaobservability.ErrWorkspaceTimelineNoDisponibleV0) {
		t.Fatalf("workspace timeline sin constantes owner: %+v", timeline)
	}

	functionContracts := NewMCPFunctionContractResourceV0()
	if !containsDescriptorStringMCPTestV0(functionContracts.PublicErrors, orquestacore.FunctionContractQueryErrEvidenciaInsuficienteV0) ||
		!containsDescriptorStringMCPTestV0(functionContracts.PublicErrors, orquestacore.FunctionContractQueryErrNoEncontradoV0) {
		t.Fatalf("function contracts sin errores owner: %+v", functionContracts.PublicErrors)
	}

	operatorResource := NewMCPOperatorOperationsResourceV0()
	operatorCapabilities := operator.NewOperatorMCPCapabilitiesV0()
	if len(operatorResource.Tools) != len(operatorCapabilities.Tools) ||
		!containsDescriptorStringMCPTestV0(operatorResource.PublicErrors, operator.ErrOperatorMCPPortUnavailableV0) {
		t.Fatalf("operator resource sin paridad owner: %+v", operatorResource)
	}
}

func assertNoSensitiveDescriptorSourceMCPTestV0(t *testing.T, payload string) {
	t.Helper()
	text := strings.ToLower(payload)
	for _, forbidden := range []string{"/home/", "home=", "token", "password", "oauth", "provider", "model", "prompt", "transcript", "completion", "dsn", "sql"} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("descriptor_source contiene %q: %s", forbidden, text)
		}
	}
}

func containsDescriptorStringMCPTestV0(items []string, want string) bool {
	for _, item := range items {
		if item == want {
			return true
		}
	}
	return false
}
