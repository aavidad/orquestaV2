package orquestaappgateway

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"

	orquestadomainwork "orquesta/modulos/orquesta-domain-work"
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
	nuevaEndpoint.DirectorPreviewClient = newDirectorPreviewClientV0(config, client)
	changeEndpoint := orquestaweb.NewAppChangeWebEndpointV0(newAppChangeClientV0(config, client))

	return orquestahttpgateway.RouteHandlersV0{
		Home:                              orquestaweb.NewHomeWebEndpointV0(),
		NuevaApp:                          orquestaweb.NuevaAppHTMLHandlerV0{Endpoint: nuevaEndpoint},
		NuevaAppGuide:                     orquestaweb.NewNuevaAppGuideWebEndpointV0(),
		OpsDashboard:                      orquestaweb.NewOpsDashboardWebEndpointV0(),
		OpsKanban:                         orquestaweb.NewOpsKanbanWebEndpointV0(),
		AutoprogrammingPage:               orquestaweb.NewAutoprogrammingWebEndpointV0(),
		AppChangePage:                     changeEndpoint,
		DirectorStatsPage:                 orquestaweb.NewDirectorStatsWebEndpointV0(newStatsClientV0(config, client)),
		RunControlPage:                    orquestaweb.NewRunControlWebEndpointV0(newRunControlClientV0(config, client)),
		RunQueuePage:                      orquestaweb.NewRunQueueWebEndpointV0(newRunQueueClientV0(config, client)),
		AppSpec:                           apiHandlers.AppSpec,
		AppDirector:                       apiHandlers.AppDirector,
		AppDirectorPreview:                apiHandlers.AppDirectorPreview,
		AppDirectorGoalObserve:            apiHandlers.AppDirectorGoalObserve,
		AppIntakeGuidedTurn:               apiHandlers.AppIntakeGuidedTurn,
		AppChange:                         apiHandlers.AppChange,
		DirectorStats:                     apiHandlers.DirectorStats,
		RunControl:                        apiHandlers.RunControl,
		RuntimeModels:                     apiHandlers.RuntimeModels,
		RunQueuePriority:                  apiHandlers.RunQueuePriority,
		QueueGlobalStatus:                 apiHandlers.QueueGlobalStatus,
		RunSupervisor:                     apiHandlers.RunSupervisor,
		OpsAgentRuntimeDetail:             apiHandlers.OpsAgentRuntimeDetail,
		OperationalStatus:                 apiHandlers.OperationalStatus,
		FunctionContractList:              apiHandlers.FunctionContractList,
		FunctionContractView:              apiHandlers.FunctionContractView,
		ServerShutdown:                    apiHandlers.ServerShutdown,
		HumanDirectorWorkReviewPlan:       apiHandlers.HumanDirectorWorkReviewPlan,
		AutoprogrammingValidateRequest:    apiHandlers.AutoprogrammingValidateRequest,
		AutoprogrammingSelfImprovement:    apiHandlers.AutoprogrammingSelfImprovement,
		AutoprogrammingPrepareRun:         apiHandlers.AutoprogrammingPrepareRun,
		AutoprogrammingObserveGoal:        apiHandlers.AutoprogrammingObserveGoal,
		AutoprogrammingObserveActiveGoals: apiHandlers.AutoprogrammingObserveActiveGoals,
		AutoprogrammingStatus:             apiHandlers.AutoprogrammingStatus,
		AutoprogrammingSupervise:          apiHandlers.AutoprogrammingSupervise,
		GovernanceCatalogQuery:            apiHandlers.GovernanceCatalogQuery,
		DomainWork:                        apiHandlers.DomainWork,
		DomainWorkStatus:                  apiHandlers.DomainWorkStatus,
		ExternalWorkDryRun:                apiHandlers.ExternalWorkDryRun,
		ExternalWorkObserve:               apiHandlers.ExternalWorkObserve,
		ExternalWorkRun:                   apiHandlers.ExternalWorkRun,
		CodebaseQuery:                     apiHandlers.CodebaseQuery,
		CodebaseStatus:                    apiHandlers.CodebaseStatus,
	}
}

func NewAPIRouteHandlersV0(config ConfigV0) orquestahttpgateway.RouteHandlersV0 {
	autoprogrammingStatus := orquestamcp.MCPAutoprogrammingStatusToolExecutorV0{
		Queue:                        config.RunQueuePriority,
		Stats:                        config.DirectorStats,
		EstadoVivoSource:             config.AutoprogrammingEstadoVivoSource,
		GoalStateStore:               config.AutoprogrammingGoalStates,
		StatusDiagnostics:            config.AutoprogrammingStatusDiagnostics,
		AllowLegacySupervisorActions: config.AllowLegacyAutoprogrammingSupervisorActions,
	}
	return orquestahttpgateway.RouteHandlersV0{
		AppSpec:                           orquestafactoryhttp.NewAppSpecHTTPHandlerV0(config.Clock),
		AppDirector:                       orquestamcp.NewMCPArrancarDirectorAppHTTPHandlerV0(config.ArrancarDirector),
		AppDirectorPreview:                orquestamcp.NewMCPPreviewDirectorAppHTTPHandlerV0(config.PreviewDirector),
		AppDirectorGoalObserve:            orquestamcp.NewMCPObserveAppDirectorGoalHTTPHandlerV0(config.ObserveDirectorGoal),
		AppIntakeGuidedTurn:               orquestaweb.NewNuevaAppIntakeGuidedHTTPHandlerWithAssistantV0(config.AppIntakeAssistant),
		AppChange:                         orquestamcp.NewMCPRequestAppChangeHTTPHandlerV0(config.RequestAppChange),
		DirectorStats:                     orquestamcp.NewMCPDirectorStatsHTTPHandlerV0(config.DirectorStats),
		RunControl:                        orquestamcp.NewMCPRunControlHTTPHandlerV0(config.RunControl),
		RuntimeModels:                     orquestamcp.NewMCPRuntimeModelsHTTPHandlerV0(config.RuntimeModels),
		RunQueuePriority:                  orquestamcp.NewMCPRunQueuePriorityHTTPHandlerV0(config.RunQueuePriority),
		QueueGlobalStatus:                 orquestamcp.NewMCPQueueGlobalStatusHTTPHandlerV0(autoprogrammingStatus),
		RunSupervisor:                     orquestamcp.NewMCPRunSupervisorHTTPHandlerV0(config.RunSupervisor),
		OpsAgentRuntimeDetail:             config.OpsAgentRuntimeDetail,
		OperationalStatus:                 orquestamcp.NewMCPOperationalStatusHTTPHandlerV0(config.OperationalStatus),
		FunctionContractList:              orquestamcp.NewMCPFunctionContractListHTTPHandlerV0(config.FunctionContracts),
		FunctionContractView:              orquestamcp.NewMCPFunctionContractViewHTTPHandlerV0(config.FunctionContracts),
		ServerShutdown:                    orquestamcp.NewMCPServerShutdownHTTPHandlerV0(config.ServerShutdown),
		HumanDirectorWorkReviewPlan:       orquestamcp.NewMCPHumanDirectorWorkReviewPlanHTTPHandlerWithOperatorQueryV0(config.OperatorQuery),
		AutoprogrammingValidateRequest:    orquestamcp.NewMCPAutoprogrammingValidateRequestHTTPHandlerV0(),
		AutoprogrammingSelfImprovement:    orquestamcp.NewMCPAutoprogrammingSelfImprovementHTTPHandlerWithPrepareRunV0(config.AutoprogrammingPrepareRun),
		AutoprogrammingPrepareRun:         orquestamcp.NewMCPAutoprogrammingPrepareRunHTTPHandlerV0(config.AutoprogrammingPrepareRun),
		AutoprogrammingObserveGoal:        orquestamcp.NewMCPAutoprogrammingObserveGoalHTTPHandlerV0(config.AutoprogrammingObserveGoal),
		AutoprogrammingObserveActiveGoals: orquestamcp.NewMCPAutoprogrammingObserveActiveGoalsHTTPHandlerV0(config.AutoprogrammingObserveActiveGoals),
		AutoprogrammingStatus:             orquestamcp.NewMCPAutoprogrammingStatusHTTPHandlerV0(autoprogrammingStatus),
		AutoprogrammingSupervise:          orquestamcp.NewMCPAutoprogrammingSuperviseHTTPHandlerV0(config.RunSupervisor),
		GovernanceCatalogQuery:            orquestagovernance.GovernanceCatalogQueryHTTPHandlerV0(config.GovernanceCatalog),
		DomainWork:                        orquestamcp.NewMCPDomainWorkHTTPHandlerV0(config.DomainWork),
		DomainWorkStatus:                  orquestamcp.NewMCPDomainWorkStatusHTTPHandlerWithRecordsV0(firstNonNilAutoprogrammingStatusExecutorV0(config.DomainWorkStatus, autoprogrammingStatus), firstNonNilDomainWorkRecordSourceV0(config.DomainWorkRecords, config.DomainWork)),
		ExternalWorkDryRun:                newExternalWorkDryRunHTTPHandlerV0(config),
		ExternalWorkObserve:               newExternalWorkObserveAliasHTTPHandlerV0(orquestamcp.NewMCPObserveAppDirectorGoalHTTPHandlerV0(config.ObserveDirectorGoal)),
		ExternalWorkRun:                   orquestamcp.NewMCPExternalWorkRunHTTPHandlerWithResponseTimeoutV0(config.ExternalWorkRun, config.Timeout),
		CodebaseQuery:                     orquestamcp.NewMCPCodebaseQueryHTTPHandlerV0(config.CodebaseQuery),
		CodebaseStatus:                    orquestamcp.NewMCPCodebaseStatusHTTPHandlerV0(config.CodebaseStatus),
	}
}

func newExternalWorkObserveAliasHTTPHandlerV0(canonical http.Handler) http.Handler {
	if canonical == nil {
		return nil
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		aliased := r.Clone(r.Context())
		urlCopy := *r.URL
		urlCopy.Path = orquestamcp.MCPObserveAppDirectorGoalHTTPPathV0
		aliased.URL = &urlCopy
		if r.Method == http.MethodGet {
			input := orquestamcp.MCPObserveAppDirectorGoalToolInputV0{
				RequestID:     r.URL.Query().Get("request_id"),
				CorrelationID: r.URL.Query().Get("correlation_id"),
				RunRef:        r.URL.Query().Get("run_ref"),
				OccurredAt:    r.URL.Query().Get("occurred_at"),
				RequestedBy:   r.URL.Query().Get("requested_by"),
			}
			body, err := json.Marshal(input)
			if err == nil {
				aliased.Method = http.MethodPost
				aliased.Body = http.NoBody
				aliased.GetBody = nil
				aliased.ContentLength = int64(len(body))
				aliased.Body = io.NopCloser(bytes.NewReader(body))
				aliased.Header = r.Header.Clone()
				aliased.Header.Set("Content-Type", "application/json")
			}
		}
		canonical.ServeHTTP(w, aliased)
	})
}

func firstNonNilAutoprogrammingStatusExecutorV0(
	primary orquestamcp.MCPTransportAutoprogrammingStatusExecutorV0,
	fallback orquestamcp.MCPTransportAutoprogrammingStatusExecutorV0,
) orquestamcp.MCPTransportAutoprogrammingStatusExecutorV0 {
	if primary != nil {
		return primary
	}
	return fallback
}

func firstNonNilDomainWorkRecordSourceV0(
	primary orquestadomainwork.DomainWorkJobRecordSourcePortV0,
	fallback orquestamcp.MCPDomainWorkExecutorPortV0,
) orquestadomainwork.DomainWorkJobRecordSourcePortV0 {
	if primary != nil {
		return primary
	}
	source, _ := fallback.(orquestadomainwork.DomainWorkJobRecordSourcePortV0)
	return source
}

func newExternalWorkDryRunHTTPHandlerV0(config ConfigV0) http.Handler {
	if config.ExternalWorkDryRun != nil {
		return orquestamcp.NewMCPExternalWorkDryRunHTTPHandlerV0(config.ExternalWorkDryRun)
	}
	return orquestamcp.NewMCPExternalWorkDryRunHTTPHandlerWithConfigV0(config.ExternalWorkDryRunConfig)
}
