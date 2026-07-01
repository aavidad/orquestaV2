package main

import (
	"net/http"
	"testing"

	orquestahttpgateway "orquesta/modulos/orquesta-http-gateway"
	orquestamcp "orquesta/modulos/orquesta-mcp"
	orquestaserver "orquesta/modulos/orquesta-server"
)

func TestServerOverlayRoutesEstanEnManifestV0(t *testing.T) {
	entries := orquestahttpgateway.PublicRouteManifestV0()
	byRef := map[string]orquestahttpgateway.RouteManifestEntryV0{}
	for _, entry := range entries {
		byRef[entry.Ref] = entry
	}

	assertRouteManifestPatternV0(t, byRef, orquestahttpgateway.RouteRefMCPJSONRPCV0, mcpRealHTTPPathV0)
	assertRouteManifestPatternV0(
		t,
		byRef,
		orquestahttpgateway.RouteRefWorkspaceTimelineV0,
		orquestamcp.MCPWorkspaceTimelineEndpointV0,
	)
}

func assertRouteManifestPatternV0(
	t *testing.T,
	entries map[string]orquestahttpgateway.RouteManifestEntryV0,
	ref string,
	want string,
) {
	t.Helper()
	if entries[ref].Pattern != want {
		t.Fatalf("%s pattern=%q want %q", ref, entries[ref].Pattern, want)
	}
}

func TestServerResourcesRouteManifestIncluyeDiscoveryOPESV0(t *testing.T) {
	routes := serverRouteManifestResourcesV0()
	byPattern := map[string]orquestaserver.ServerRouteResourceV0{}
	for _, route := range routes {
		byPattern[route.Pattern] = route
	}

	for _, path := range []string{
		orquestaserver.ServerReadinessEndpointV0,
		orquestaserver.ServerResourcesEndpointV0,
		orquestaserver.ServerRoutesEndpointV0,
		"/health",
		"/healthz",
		orquestahttpgateway.RouteDomainWorkV0,
		orquestahttpgateway.RouteDomainWorkStatusV0,
		orquestahttpgateway.RouteExternalWorkRunV0,
		orquestahttpgateway.RouteRunSupervisorV0,
		orquestahttpgateway.RouteAutoprogrammingStatusV0,
	} {
		route := byPattern[path]
		if route.Pattern == "" || !route.Mounted {
			t.Fatalf("ruta discovery no montada %s en %+v", path, routes)
		}
		if len(route.Methods) == 0 {
			t.Fatalf("ruta discovery sin metodos %s: %+v", path, route)
		}
	}
	if !serverRouteHasMethodForTestV0(byPattern[orquestahttpgateway.RouteDomainWorkV0], http.MethodPost) ||
		!serverRouteHasMethodForTestV0(byPattern[orquestahttpgateway.RouteDomainWorkStatusV0], http.MethodGet) ||
		!serverRouteHasMethodForTestV0(byPattern[orquestahttpgateway.RouteExternalWorkRunV0], http.MethodPost) ||
		!serverRouteHasMethodForTestV0(byPattern["/health"], http.MethodGet) ||
		!serverRouteHasMethodForTestV0(byPattern[orquestaserver.ServerReadinessEndpointV0], http.MethodGet) {
		t.Fatalf("metodos discovery invalidos: %+v", byPattern)
	}
}

func serverRouteHasMethodForTestV0(route orquestaserver.ServerRouteResourceV0, method string) bool {
	for _, got := range route.Methods {
		if got == method {
			return true
		}
	}
	return false
}
