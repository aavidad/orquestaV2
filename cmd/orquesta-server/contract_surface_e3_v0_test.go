package main

import (
	"testing"

	orquestahttpgateway "orquesta/modulos/orquesta-http-gateway"
	orquestamcp "orquesta/modulos/orquesta-mcp"
	channel "orquesta/modulos/orquesta-operator-director-channel"
	orquestaserver "orquesta/modulos/orquesta-server"
)

type e3ContractSurfaceForTestV0 struct {
	ContractRef string
	MCPToolName string
	RouteRefs   []string
	OptInRoutes []string
}

func TestServerE3ContractSurfaceCatalogV0CubreHTTPDiscoveryYMCP(t *testing.T) {
	manifestByRef := e3RouteManifestByRefForTestV0()
	resourcesByPattern := e3ServerRoutesByPatternForTestV0(serverRouteManifestResourcesV0())

	for _, surface := range e3ContractSurfaceCatalogForTestV0() {
		for _, routeRef := range surface.RouteRefs {
			entry, ok := manifestByRef[routeRef]
			if !ok {
				t.Fatalf("contrato %s sin route_ref %s en manifest", surface.ContractRef, routeRef)
			}
			if !e3ContainsStringForTestV0(entry.ContractRefs, surface.ContractRef) {
				t.Fatalf("route_ref %s no declara contrato %s: %+v", routeRef, surface.ContractRef, entry.ContractRefs)
			}
			resource := resourcesByPattern[entry.Pattern]
			if resource.Pattern == "" {
				t.Fatalf("contrato %s route_ref %s sin discovery de servidor para %s", surface.ContractRef, routeRef, entry.Pattern)
			}
			if !e3ContainsStringForTestV0(resource.ContractRefs, surface.ContractRef) {
				t.Fatalf("discovery %s no expone contrato %s: %+v", entry.Pattern, surface.ContractRef, resource.ContractRefs)
			}
			if !e3ContainsStringForTestV0(surface.OptInRoutes, routeRef) && !resource.Mounted {
				t.Fatalf("discovery %s del contrato %s no montado sin declararse opt-in", entry.Pattern, surface.ContractRef)
			}
		}
		if surface.MCPToolName == "" {
			continue
		}
		fields, ok := orquestamcp.MCPTransportToolInputFieldsV0(surface.MCPToolName)
		if !ok || len(fields) == 0 {
			t.Fatalf("contrato %s tool %s sin DTO MCP canonico", surface.ContractRef, surface.MCPToolName)
		}
	}
}

func TestServerE3ContractSurfaceCatalogV0InventariaContratosHTTP(t *testing.T) {
	known := map[string]bool{}
	for _, surface := range e3ContractSurfaceCatalogForTestV0() {
		known[surface.ContractRef] = true
	}
	for _, entry := range orquestahttpgateway.PublicRouteManifestV0() {
		for _, contractRef := range entry.ContractRefs {
			if !known[contractRef] {
				t.Fatalf("contrato HTTP no inventariado en E3: %s route_ref=%s", contractRef, entry.Ref)
			}
		}
	}
}

func e3ContractSurfaceCatalogForTestV0() []e3ContractSurfaceForTestV0 {
	return []e3ContractSurfaceForTestV0{
		{
			ContractRef: orquestahttpgateway.RouteContractNuevaAppSolicitarV0,
			MCPToolName: orquestamcp.MCPNuevaAppToolNameV0,
			RouteRefs: []string{
				orquestahttpgateway.RouteRefNuevaAppV0,
				orquestahttpgateway.RouteRefAppSpecV0,
			},
		},
		{
			ContractRef: orquestahttpgateway.RouteContractNuevaAppWizardV0,
			MCPToolName: orquestamcp.MCPNuevaAppWizardToolNameV0,
			RouteRefs: []string{
				orquestahttpgateway.RouteRefNuevaAppV0,
				orquestahttpgateway.RouteRefAppIntakeGuidedTurnV0,
			},
		},
		{
			ContractRef: orquestahttpgateway.RouteContractNuevaAppWizardBotV0,
			MCPToolName: orquestamcp.MCPNuevaAppWizardBotToolNameV0,
			RouteRefs: []string{
				orquestahttpgateway.RouteRefNuevaAppV0,
				orquestahttpgateway.RouteRefAppIntakeWizardBotV0,
			},
		},
		{
			ContractRef: orquestahttpgateway.RouteContractAutoprogrammingPrepareRunV0,
			MCPToolName: orquestamcp.MCPAutoprogrammingPrepareRunToolNameV0,
			RouteRefs: []string{
				orquestahttpgateway.RouteRefAutoprogrammingPageV0,
				orquestahttpgateway.RouteRefAutoprogrammingPrepareRunV0,
			},
		},
		{
			ContractRef: orquestahttpgateway.RouteContractAutoprogrammingStatusV0,
			MCPToolName: orquestamcp.MCPAutoprogrammingStatusToolNameV0,
			RouteRefs: []string{
				orquestahttpgateway.RouteRefAutoprogrammingPageV0,
				orquestahttpgateway.RouteRefAutoprogrammingStatusV0,
			},
		},
		{
			ContractRef: orquestahttpgateway.RouteContractOperatorDirectorReviewPlanV0,
			MCPToolName: orquestamcp.MCPHumanDirectorWorkReviewPlanToolNameV0,
			RouteRefs: []string{
				orquestahttpgateway.RouteRefHumanDirectorWorkReviewPlanV0,
			},
		},
		{
			ContractRef: "operator_director.message.v0",
			MCPToolName: channel.OperatorDirectorMessageToolNameV0,
		},
		{
			ContractRef: orquestahttpgateway.RouteContractOperatorTelegramUpdateV0,
			RouteRefs: []string{
				orquestahttpgateway.RouteRefOperatorTelegramUpdateV0,
			},
			OptInRoutes: []string{
				orquestahttpgateway.RouteRefOperatorTelegramUpdateV0,
			},
		},
	}
}

func e3RouteManifestByRefForTestV0() map[string]orquestahttpgateway.RouteManifestEntryV0 {
	out := map[string]orquestahttpgateway.RouteManifestEntryV0{}
	for _, entry := range orquestahttpgateway.PublicRouteManifestV0() {
		out[entry.Ref] = entry
	}
	return out
}

func e3ServerRoutesByPatternForTestV0(
	routes []orquestaserver.ServerRouteResourceV0,
) map[string]orquestaserver.ServerRouteResourceV0 {
	out := map[string]orquestaserver.ServerRouteResourceV0{}
	for _, route := range routes {
		out[route.Pattern] = route
	}
	return out
}

func e3ContainsStringForTestV0(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
