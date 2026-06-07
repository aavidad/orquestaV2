package orquestahttpgateway

import "net/http"

const (
	RouteNuevaAppV0                       = "/nueva-app"
	RouteOpsDashboardV0                   = "/ops"
	RouteAppChangePageV0                  = "/app-change"
	RouteDirectorStatsPageV0              = "/director-stats"
	RouteRunControlPageV0                 = "/run-control"
	RouteRunQueuePageV0                   = "/run-queue"
	RouteAppSpecV0                        = "/api/v0/apps/spec"
	RouteAppDirectorV0                    = "/api/v0/apps/director"
	RouteAppChangeV0                      = "/api/v0/apps/"
	RouteDirectorStatsV0                  = "/api/v0/director/stats"
	RouteRunControlV0                     = "/api/v0/runs/control"
	RouteRuntimeModelsV0                  = "/api/v0/runtime/models"
	RouteRunQueuePriorityV0               = "/api/v0/runs/queue/priority"
	RouteRunSupervisorV0                  = "/api/v0/runs/supervise"
	RouteOpsAgentRuntimeDetailV0          = "/api/v0/ops/agent-runtime-detail"
	RouteOperationalStatusV0              = "/api/v0/operational-status/query"
	RouteFunctionContractListV0           = "/api/v0/core/function-contracts/list"
	RouteFunctionContractViewV0           = "/api/v0/core/function-contracts/view"
	RouteServerShutdownV0                 = "/api/v0/server/shutdown"
	RouteHumanDirectorWorkReviewPlanV0    = "/api/v0/director/human-work/review-plan"
	RouteAutoprogrammingValidateRequestV0 = "/api/v0/autoprogramming/validate-request"
	RouteAutoprogrammingSelfImprovementV0 = "/api/v0/autoprogramming/self-improvement"
	RouteAutoprogrammingPrepareRunV0      = "/api/v0/autoprogramming/prepare-run"
	RouteAutoprogrammingStatusV0          = "/api/v0/autoprogramming/status"
	RouteAutoprogrammingSuperviseV0       = "/api/v0/autoprogramming/supervise"
	RouteGovernanceCatalogQueryV0         = "/api/v0/governance/catalog/query"
	RouteDomainWorkV0                     = "/api/v0/domain-work"
	RouteExternalWorkRunV0                = "/api/v0/external-work/run"
)

type RouteHandlersV0 struct {
	NuevaApp                       http.Handler
	OpsDashboard                   http.Handler
	AppChangePage                  http.Handler
	DirectorStatsPage              http.Handler
	RunControlPage                 http.Handler
	RunQueuePage                   http.Handler
	AppSpec                        http.Handler
	AppDirector                    http.Handler
	AppChange                      http.Handler
	DirectorStats                  http.Handler
	RunControl                     http.Handler
	RuntimeModels                  http.Handler
	RunQueuePriority               http.Handler
	RunSupervisor                  http.Handler
	OpsAgentRuntimeDetail          http.Handler
	OperationalStatus              http.Handler
	FunctionContractList           http.Handler
	FunctionContractView           http.Handler
	ServerShutdown                 http.Handler
	HumanDirectorWorkReviewPlan    http.Handler
	AutoprogrammingValidateRequest http.Handler
	AutoprogrammingSelfImprovement http.Handler
	AutoprogrammingPrepareRun      http.Handler
	AutoprogrammingStatus          http.Handler
	AutoprogrammingSupervise       http.Handler
	GovernanceCatalogQuery         http.Handler
	DomainWork                     http.Handler
	ExternalWorkRun                http.Handler
}

func NewAppGatewayMuxV0(handlers RouteHandlersV0) http.Handler {
	mux := http.NewServeMux()
	if issue := validateGatewayRouteManifestPublicV0(); issue.Code != "" {
		panic(issue.Code)
	}
	if issue := validateGatewayRouteRegistrationsV0(gatewayRouteRegistrationsV0(handlers)); issue.Code != "" {
		panic(issue.Code)
	}

	for _, registration := range gatewayRouteRegistrationsV0(handlers) {
		handleIfPresent(mux, registration.route, registration.handler)
	}

	return NewControlPlaneHTTPHeadersV0(mux)
}

func handleIfPresent(mux *http.ServeMux, route string, handler http.Handler) {
	if handler == nil {
		return
	}

	mux.Handle(route, handler)
}

type gatewayRouteRegistrationV0 struct {
	ref     string
	route   string
	handler http.Handler
}

func gatewayRouteRegistrationsV0(handlers RouteHandlersV0) []gatewayRouteRegistrationV0 {
	return []gatewayRouteRegistrationV0{
		{ref: RouteRefNuevaAppV0, route: RouteNuevaAppV0, handler: handlers.NuevaApp},
		{ref: RouteRefOpsDashboardV0, route: RouteOpsDashboardV0, handler: handlers.OpsDashboard},
		{ref: RouteRefAppChangePageV0, route: RouteAppChangePageV0, handler: handlers.AppChangePage},
		{ref: RouteRefDirectorStatsPageV0, route: RouteDirectorStatsPageV0, handler: handlers.DirectorStatsPage},
		{ref: RouteRefRunControlPageV0, route: RouteRunControlPageV0, handler: handlers.RunControlPage},
		{ref: RouteRefRunQueuePageV0, route: RouteRunQueuePageV0, handler: handlers.RunQueuePage},
		{ref: RouteRefAppSpecV0, route: RouteAppSpecV0, handler: handlers.AppSpec},
		{ref: RouteRefAppDirectorV0, route: RouteAppDirectorV0, handler: handlers.AppDirector},
		{ref: RouteRefAppChangeV0, route: RouteAppChangeV0, handler: handlers.AppChange},
		{ref: RouteRefDirectorStatsV0, route: RouteDirectorStatsV0, handler: handlers.DirectorStats},
		{ref: RouteRefRunControlV0, route: RouteRunControlV0, handler: handlers.RunControl},
		{ref: RouteRefRuntimeModelsV0, route: RouteRuntimeModelsV0, handler: handlers.RuntimeModels},
		{ref: RouteRefRunQueuePriorityV0, route: RouteRunQueuePriorityV0, handler: handlers.RunQueuePriority},
		{ref: RouteRefRunSupervisorV0, route: RouteRunSupervisorV0, handler: handlers.RunSupervisor},
		{ref: RouteRefOpsAgentRuntimeDetailV0, route: RouteOpsAgentRuntimeDetailV0, handler: handlers.OpsAgentRuntimeDetail},
		{ref: RouteRefOperationalStatusV0, route: RouteOperationalStatusV0, handler: handlers.OperationalStatus},
		{ref: RouteRefFunctionContractListV0, route: RouteFunctionContractListV0, handler: handlers.FunctionContractList},
		{ref: RouteRefFunctionContractViewV0, route: RouteFunctionContractViewV0, handler: handlers.FunctionContractView},
		{ref: RouteRefServerShutdownV0, route: RouteServerShutdownV0, handler: handlers.ServerShutdown},
		{ref: RouteRefHumanDirectorWorkReviewPlanV0, route: RouteHumanDirectorWorkReviewPlanV0, handler: handlers.HumanDirectorWorkReviewPlan},
		{ref: RouteRefAutoprogrammingValidateRequestV0, route: RouteAutoprogrammingValidateRequestV0, handler: handlers.AutoprogrammingValidateRequest},
		{ref: RouteRefAutoprogrammingSelfImprovementV0, route: RouteAutoprogrammingSelfImprovementV0, handler: handlers.AutoprogrammingSelfImprovement},
		{ref: RouteRefAutoprogrammingPrepareRunV0, route: RouteAutoprogrammingPrepareRunV0, handler: handlers.AutoprogrammingPrepareRun},
		{ref: RouteRefAutoprogrammingStatusV0, route: RouteAutoprogrammingStatusV0, handler: handlers.AutoprogrammingStatus},
		{ref: RouteRefAutoprogrammingSuperviseV0, route: RouteAutoprogrammingSuperviseV0, handler: handlers.AutoprogrammingSupervise},
		{ref: RouteRefGovernanceCatalogQueryV0, route: RouteGovernanceCatalogQueryV0, handler: handlers.GovernanceCatalogQuery},
		{ref: RouteRefDomainWorkV0, route: RouteDomainWorkV0, handler: handlers.DomainWork},
		{ref: RouteRefExternalWorkRunV0, route: RouteExternalWorkRunV0, handler: handlers.ExternalWorkRun},
	}
}
