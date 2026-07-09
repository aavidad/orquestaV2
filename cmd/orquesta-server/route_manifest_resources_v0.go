package main

import (
	"net/http"

	orquestahttpgateway "orquesta/modulos/orquesta-http-gateway"
	orquestaserver "orquesta/modulos/orquesta-server"
)

func serverRouteManifestResourcesV0() []orquestaserver.ServerRouteResourceV0 {
	routes := []orquestaserver.ServerRouteResourceV0{
		{
			Ref:             "route-ref-health-v0",
			Pattern:         "/health",
			Kind:            orquestahttpgateway.RouteManifestKindExactV0,
			Owner:           "orquesta-server",
			Methods:         []string{http.MethodGet},
			SecurityProfile: orquestahttpgateway.RouteSecurityControlPlaneReadV0,
			Mounted:         true,
		},
		{
			Ref:             "route-ref-healthz-v0",
			Pattern:         "/healthz",
			Kind:            orquestahttpgateway.RouteManifestKindExactV0,
			Owner:           "orquesta-server",
			Methods:         []string{http.MethodGet},
			SecurityProfile: orquestahttpgateway.RouteSecurityControlPlaneReadV0,
			Mounted:         true,
		},
		{
			Ref:             "route-ref-server-readiness-v0",
			Pattern:         orquestaserver.ServerReadinessEndpointV0,
			Kind:            orquestahttpgateway.RouteManifestKindExactV0,
			Owner:           "orquesta-server",
			Methods:         []string{http.MethodGet},
			SecurityProfile: orquestahttpgateway.RouteSecurityControlPlaneReadV0,
			Mounted:         true,
		},
		{
			Ref:             "route-ref-server-status-v0",
			Pattern:         orquestaserver.ServerStatusEndpointV0,
			Kind:            orquestahttpgateway.RouteManifestKindExactV0,
			Owner:           "orquesta-server",
			Methods:         []string{http.MethodGet},
			SecurityProfile: orquestahttpgateway.RouteSecurityControlPlaneReadV0,
			Mounted:         true,
		},
		{
			Ref:             "route-ref-server-status-legacy-v0",
			Pattern:         orquestaserver.ServerStatusLegacyEndpointV0,
			Kind:            orquestahttpgateway.RouteManifestKindExactV0,
			Owner:           "orquesta-server",
			Methods:         []string{http.MethodGet},
			SecurityProfile: orquestahttpgateway.RouteSecurityControlPlaneReadV0,
			Mounted:         true,
		},
		{
			Ref:             "route-ref-server-resources-v0",
			Pattern:         orquestaserver.ServerResourcesEndpointV0,
			Kind:            orquestahttpgateway.RouteManifestKindExactV0,
			Owner:           "orquesta-server",
			Methods:         []string{http.MethodGet},
			SecurityProfile: orquestahttpgateway.RouteSecurityControlPlaneReadV0,
			Mounted:         true,
		},
		{
			Ref:             "route-ref-server-routes-v0",
			Pattern:         orquestaserver.ServerRoutesEndpointV0,
			Kind:            orquestahttpgateway.RouteManifestKindExactV0,
			Owner:           "orquesta-server",
			Methods:         []string{http.MethodGet},
			SecurityProfile: orquestahttpgateway.RouteSecurityControlPlaneReadV0,
			Mounted:         true,
		},
	}
	for _, entry := range orquestahttpgateway.PublicRouteManifestV0() {
		routes = append(routes, serverRouteResourceFromHTTPGatewayV0(entry))
	}
	return routes
}

func serverRouteResourceFromHTTPGatewayV0(
	entry orquestahttpgateway.RouteManifestEntryV0,
) orquestaserver.ServerRouteResourceV0 {
	return orquestaserver.ServerRouteResourceV0{
		Ref:               entry.Ref,
		Pattern:           entry.Pattern,
		Kind:              entry.Kind,
		Owner:             entry.Owner,
		Methods:           append([]string{}, entry.Methods...),
		SecurityProfile:   entry.SecurityProfile,
		ContractRefs:      append([]string{}, entry.ContractRefs...),
		Mounted:           true,
		ShadowsPrefixRefs: append([]string{}, entry.ShadowsPrefixRefs...),
	}
}
