package orquestaappgateway

import (
	"net/http"

	orquestafactory "orquesta/modulos/orquesta-factory"
	orquestahttpgateway "orquesta/modulos/orquesta-http-gateway"
	orquestamcp "orquesta/modulos/orquesta-mcp"
	orquestaweb "orquesta/modulos/orquesta-web"
)

func NewHTTPHandlerV0(config ConfigV0) http.Handler {
	return orquestahttpgateway.NewAppGatewayMuxV0(NewRouteHandlersV0(config))
}

func NewRouteHandlersV0(config ConfigV0) orquestahttpgateway.RouteHandlersV0 {
	apiHandlers := NewAPIRouteHandlersV0(config)
	apiMux := orquestahttpgateway.NewAppGatewayMuxV0(apiHandlers)
	client := httpClientForWebV0(config, apiMux)

	nuevaEndpoint := orquestaweb.NewNuevaAppWebEndpointV0(
		newSpecClientV0(config, client),
	)
	nuevaEndpoint.DirectorClient = newDirectorClientV0(config, client)
	changeEndpoint := orquestaweb.NewAppChangeWebEndpointV0(newAppChangeClientV0(config, client))

	return orquestahttpgateway.RouteHandlersV0{
		NuevaApp:                       orquestaweb.NuevaAppHTMLHandlerV0{Endpoint: nuevaEndpoint},
		AppChangePage:                  changeEndpoint,
		DirectorStatsPage:              orquestaweb.NewDirectorStatsWebEndpointV0(newStatsClientV0(config, client)),
		RunControlPage:                 orquestaweb.NewRunControlWebEndpointV0(newRunControlClientV0(config, client)),
		RunQueuePage:                   orquestaweb.NewRunQueueWebEndpointV0(newRunQueueClientV0(config, client)),
		AppSpec:                        apiHandlers.AppSpec,
		AppDirector:                    apiHandlers.AppDirector,
		AppChange:                      apiHandlers.AppChange,
		DirectorStats:                  apiHandlers.DirectorStats,
		RunControl:                     apiHandlers.RunControl,
		RunQueuePriority:               apiHandlers.RunQueuePriority,
		RunSupervisor:                  apiHandlers.RunSupervisor,
		ServerShutdown:                 apiHandlers.ServerShutdown,
		AutoprogrammingValidateRequest: apiHandlers.AutoprogrammingValidateRequest,
		AutoprogrammingPrepareRun:      apiHandlers.AutoprogrammingPrepareRun,
		DomainWork:                     apiHandlers.DomainWork,
		ExternalWorkRun:                apiHandlers.ExternalWorkRun,
	}
}

func NewAPIRouteHandlersV0(config ConfigV0) orquestahttpgateway.RouteHandlersV0 {
	return orquestahttpgateway.RouteHandlersV0{
		AppSpec:                        orquestafactory.NewAppSpecHTTPHandlerV0(config.Clock),
		AppDirector:                    orquestamcp.NewMCPArrancarDirectorAppHTTPHandlerV0(config.ArrancarDirector),
		AppChange:                      orquestamcp.NewMCPRequestAppChangeHTTPHandlerV0(config.RequestAppChange),
		DirectorStats:                  orquestamcp.NewMCPDirectorStatsHTTPHandlerV0(config.DirectorStats),
		RunControl:                     orquestamcp.NewMCPRunControlHTTPHandlerV0(config.RunControl),
		RunQueuePriority:               orquestamcp.NewMCPRunQueuePriorityHTTPHandlerV0(config.RunQueuePriority),
		RunSupervisor:                  orquestamcp.NewMCPRunSupervisorHTTPHandlerV0(config.RunSupervisor),
		ServerShutdown:                 orquestamcp.NewMCPServerShutdownHTTPHandlerV0(config.ServerShutdown),
		AutoprogrammingValidateRequest: orquestamcp.NewMCPAutoprogrammingValidateRequestHTTPHandlerV0(),
		AutoprogrammingPrepareRun:      orquestamcp.NewMCPAutoprogrammingPrepareRunHTTPHandlerV0(config.AutoprogrammingPrepareRun),
		DomainWork:                     orquestamcp.NewMCPDomainWorkHTTPHandlerV0(config.DomainWork),
		ExternalWorkRun:                orquestamcp.NewMCPExternalWorkRunHTTPHandlerV0(config.ExternalWorkRun),
	}
}
