package main

import (
	"testing"

	orquestahttpgateway "orquesta/modulos/orquesta-http-gateway"
	orquestamcp "orquesta/modulos/orquesta-mcp"
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
