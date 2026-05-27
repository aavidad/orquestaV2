package main

import (
	"strings"
	"testing"

	orquestamcp "orquesta/modulos/orquesta-mcp"
)

func TestMCPRealTransportV0RechazaResourceDuplicadoPorNombreSinMutacionParcial(t *testing.T) {
	registry := newMCPRealRegistryForCollisionTestV0()
	first := resourceEnvelopeForCollisionTestV0("orquesta.contracts.shared.v0", "orquesta://contracts/shared/v0")
	if err := registry.RegisterResourceV0(first); err != nil {
		t.Fatalf("seed: %v", err)
	}

	err := registry.RegisterResourceV0(
		resourceEnvelopeForCollisionTestV0("orquesta.contracts.shared.v0", "orquesta://contracts/other/v0"),
	)
	assertMCPRealRegistrationCollisionTestV0(t, err, mcpDuplicateResourceV0, "orquesta.contracts.shared.v0")
	assertMCPRealResourceCatalogUnchangedTestV0(t, registry, first)
	if _, ok := registry.resourcesByURI["orquesta://contracts/other/v0"]; ok {
		t.Fatalf("catalogo mutado tras error: %+v", registry.resourcesByURI)
	}
}

func TestMCPRealTransportV0RechazaResourceDuplicadoPorURISinMutacionParcial(t *testing.T) {
	registry := newMCPRealRegistryForCollisionTestV0()
	first := resourceEnvelopeForCollisionTestV0("orquesta.contracts.shared.v0", "orquesta://contracts/shared/v0")
	if err := registry.RegisterResourceV0(first); err != nil {
		t.Fatalf("seed: %v", err)
	}

	err := registry.RegisterResourceV0(
		resourceEnvelopeForCollisionTestV0("orquesta.contracts.other.v0", "orquesta://contracts/shared/v0"),
	)
	assertMCPRealRegistrationCollisionTestV0(t, err, mcpDuplicateResourceV0, "orquesta://contracts/shared/v0")
	assertMCPRealResourceCatalogUnchangedTestV0(t, registry, first)
	if _, ok := registry.resourcesByName["orquesta.contracts.other.v0"]; ok {
		t.Fatalf("catalogo mutado tras error: %+v", registry.resourcesByName)
	}
}

func TestMCPRealTransportV0RechazaToolDuplicadoPorNombreSinPayloadPrivado(t *testing.T) {
	registry := newMCPRealRegistryForCollisionTestV0()
	first := toolEnvelopeForCollisionTestV0("orquesta.status.v0", "orquesta://contracts/operator/v0")
	if err := registry.RegisterToolV0(first); err != nil {
		t.Fatalf("seed: %v", err)
	}

	err := registry.RegisterToolV0(
		toolEnvelopeForCollisionTestV0("orquesta.status.v0", "https://localhost/private?token=secret"),
	)
	assertMCPRealRegistrationCollisionTestV0(t, err, mcpDuplicateToolV0, "redacted")
	if len(registry.tools) != 1 || registry.tools[first.Name].ResourceURI != first.ResourceURI {
		t.Fatalf("catalogo mutado tras error: %+v", registry.tools)
	}
	if strings.Contains(err.Error(), "localhost") || strings.Contains(err.Error(), "secret") {
		t.Fatalf("error publico filtra detalle privado: %s", err.Error())
	}
}

func TestMCPRealTransportV0ListadosConservanOrdenEstable(t *testing.T) {
	registry := newMCPRealRegistryForCollisionTestV0()
	for _, resource := range []orquestamcp.MCPTransportResourceEnvelopeV0{
		resourceEnvelopeForCollisionTestV0("zeta.resource.v0", "orquesta://contracts/zeta/v0"),
		resourceEnvelopeForCollisionTestV0("alpha.resource.v0", "orquesta://contracts/alpha/v0"),
	} {
		if err := registry.RegisterResourceV0(resource); err != nil {
			t.Fatalf("resource: %v", err)
		}
	}
	for _, tool := range []orquestamcp.MCPTransportToolEnvelopeV0{
		toolEnvelopeForCollisionTestV0("zeta.tool.v0", "orquesta://contracts/zeta/v0"),
		toolEnvelopeForCollisionTestV0("alpha.tool.v0", "orquesta://contracts/alpha/v0"),
	} {
		if err := registry.RegisterToolV0(tool); err != nil {
			t.Fatalf("tool: %v", err)
		}
	}

	resources := registry.resourceListV0().Resources
	tools := registry.toolListV0().Tools
	if resources[0].Name != "alpha.resource.v0" || resources[1].Name != "zeta.resource.v0" {
		t.Fatalf("resources fuera de orden: %+v", resources)
	}
	if tools[0].Name != "alpha.tool.v0" || tools[1].Name != "zeta.tool.v0" {
		t.Fatalf("tools fuera de orden: %+v", tools)
	}
}

func newMCPRealRegistryForCollisionTestV0() *mcpRealTransportRegistryV0 {
	return &mcpRealTransportRegistryV0{
		resourcesByName: map[string]orquestamcp.MCPTransportResourceEnvelopeV0{},
		resourcesByURI:  map[string]orquestamcp.MCPTransportResourceEnvelopeV0{},
		tools:           map[string]orquestamcp.MCPTransportToolEnvelopeV0{},
	}
}

func resourceEnvelopeForCollisionTestV0(name string, uri string) orquestamcp.MCPTransportResourceEnvelopeV0 {
	return orquestamcp.MCPTransportResourceEnvelopeV0{Name: name, URI: uri, ContentType: "application/json"}
}

func toolEnvelopeForCollisionTestV0(name string, uri string) orquestamcp.MCPTransportToolEnvelopeV0 {
	return orquestamcp.MCPTransportToolEnvelopeV0{Name: name, ResourceURI: uri}
}

func assertMCPRealResourceCatalogUnchangedTestV0(
	t *testing.T,
	registry *mcpRealTransportRegistryV0,
	want orquestamcp.MCPTransportResourceEnvelopeV0,
) {
	t.Helper()
	if len(registry.resourcesByName) != 1 ||
		len(registry.resourcesByURI) != 1 ||
		registry.resourcesByName[want.Name].URI != want.URI ||
		registry.resourcesByURI[want.URI].Name != want.Name {
		t.Fatalf("catalogo mutado: byName=%+v byURI=%+v", registry.resourcesByName, registry.resourcesByURI)
	}
}

func assertMCPRealRegistrationCollisionTestV0(t *testing.T, err error, code string, expected string) {
	t.Helper()
	if err == nil {
		t.Fatalf("error esperado")
	}
	text := err.Error()
	if !strings.Contains(text, code) || !strings.Contains(text, expected) {
		t.Fatalf("error=%q code=%q expected=%q", text, code, expected)
	}
}
