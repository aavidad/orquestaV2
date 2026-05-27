package orquestaappgateway

import (
	"net/http"

	orquestafactoryhttp "orquesta/modulos/orquesta-factory-http"
	orquestagovernance "orquesta/modulos/orquesta-governance"
	orquestahttpgateway "orquesta/modulos/orquesta-http-gateway"
	orquestamcp "orquesta/modulos/orquesta-mcp"
	orquestaweb "orquesta/modulos/orquesta-web"
)

func NewHTTPHandlerV0(config ConfigV0) http.Handler {
	handler := orquestahttpgateway.NewAppGatewayMuxV0(NewRouteHandlersV0(config))
	if config.AppVCS == nil {
		secured := orquestahttpgateway.NewControlPlaneHTTPHeadersV0(
			orquestahttpgateway.NewBrowserMutationIntentGuardV0(config.BrowserMutationIntent, handler),
		)
		return ObserveWebHTMLRenderErrorsV0(secured, config.WebHTMLRenderObserver)
	}
	mux := http.NewServeMux()
	mux.Handle("/", handler)
	mux.Handle(orquestamcp.MCPAppVCSHTTPPathV0, orquestamcp.NewMCPAppVCSHTTPHandlerV0(config.AppVCS))
	secured := orquestahttpgateway.NewControlPlaneHTTPHeadersV0(
		orquestahttpgateway.NewBrowserMutationIntentGuardV0(config.BrowserMutationIntent, mux),
	)
	return ObserveWebHTMLRenderErrorsV0(secured, config.WebHTMLRenderObserver)
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
		OpsDashboard:                   orquestaweb.NewOpsDashboardWebEndpointV0(),
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
		OpsAgentRuntimeDetail:          apiHandlers.OpsAgentRuntimeDetail,
		OperationalStatus:              apiHandlers.OperationalStatus,
		FunctionContractList:           apiHandlers.FunctionContractList,
		FunctionContractView:           apiHandlers.FunctionContractView,
		ServerShutdown:                 apiHandlers.ServerShutdown,
		HumanDirectorWorkReviewPlan:    apiHandlers.HumanDirectorWorkReviewPlan,
		AutoprogrammingValidateRequest: apiHandlers.AutoprogrammingValidateRequest,
		AutoprogrammingSelfImprovement: apiHandlers.AutoprogrammingSelfImprovement,
		AutoprogrammingPrepareRun:      apiHandlers.AutoprogrammingPrepareRun,
		AutoprogrammingStatus:          apiHandlers.AutoprogrammingStatus,
		AutoprogrammingSupervise:       apiHandlers.AutoprogrammingSupervise,
		GovernanceCatalogQuery:         apiHandlers.GovernanceCatalogQuery,
		DomainWork:                     apiHandlers.DomainWork,
		ExternalWorkRun:                apiHandlers.ExternalWorkRun,
	}
}

func NewAPIRouteHandlersV0(config ConfigV0) orquestahttpgateway.RouteHandlersV0 {
	autoprogrammingStatus := orquestamcp.MCPAutoprogrammingStatusToolExecutorV0{
		Queue: config.RunQueuePriority,
		Stats: config.DirectorStats,
	}
	return orquestahttpgateway.RouteHandlersV0{
		AppSpec:                        orquestafactoryhttp.NewAppSpecHTTPHandlerV0(config.Clock),
		AppDirector:                    orquestamcp.NewMCPArrancarDirectorAppHTTPHandlerV0(config.ArrancarDirector),
		AppChange:                      orquestamcp.NewMCPRequestAppChangeHTTPHandlerV0(config.RequestAppChange),
		DirectorStats:                  orquestamcp.NewMCPDirectorStatsHTTPHandlerV0(config.DirectorStats),
		RunControl:                     orquestamcp.NewMCPRunControlHTTPHandlerV0(config.RunControl),
		RunQueuePriority:               orquestamcp.NewMCPRunQueuePriorityHTTPHandlerV0(config.RunQueuePriority),
		RunSupervisor:                  orquestamcp.NewMCPRunSupervisorHTTPHandlerV0(config.RunSupervisor),
		OpsAgentRuntimeDetail:          config.OpsAgentRuntimeDetail,
		OperationalStatus:              orquestamcp.NewMCPOperationalStatusHTTPHandlerV0(config.OperationalStatus),
		FunctionContractList:           orquestamcp.NewMCPFunctionContractListHTTPHandlerV0(config.FunctionContracts),
		FunctionContractView:           orquestamcp.NewMCPFunctionContractViewHTTPHandlerV0(config.FunctionContracts),
		ServerShutdown:                 orquestamcp.NewMCPServerShutdownHTTPHandlerV0(config.ServerShutdown),
		HumanDirectorWorkReviewPlan:    orquestamcp.NewMCPHumanDirectorWorkReviewPlanHTTPHandlerWithOperatorQueryV0(config.OperatorQuery),
		AutoprogrammingValidateRequest: orquestamcp.NewMCPAutoprogrammingValidateRequestHTTPHandlerV0(),
		AutoprogrammingSelfImprovement: orquestamcp.NewMCPAutoprogrammingSelfImprovementHTTPHandlerWithPrepareRunV0(config.AutoprogrammingPrepareRun),
		AutoprogrammingPrepareRun:      orquestamcp.NewMCPAutoprogrammingPrepareRunHTTPHandlerV0(config.AutoprogrammingPrepareRun),
		AutoprogrammingStatus:          orquestamcp.NewMCPAutoprogrammingStatusHTTPHandlerV0(autoprogrammingStatus),
		AutoprogrammingSupervise:       orquestamcp.NewMCPAutoprogrammingSuperviseHTTPHandlerV0(config.RunSupervisor),
		GovernanceCatalogQuery:         orquestagovernance.GovernanceCatalogQueryHTTPHandlerV0(config.GovernanceCatalog),
		DomainWork:                     orquestamcp.NewMCPDomainWorkHTTPHandlerV0(config.DomainWork),
		ExternalWorkRun:                orquestamcp.NewMCPExternalWorkRunHTTPHandlerV0(config.ExternalWorkRun),
	}
}
