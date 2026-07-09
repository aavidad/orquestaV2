package orquestahttpgateway

import "net/http"

const (
	RouteHomeV0                              = "/"
	RouteNuevaAppV0                          = "/nueva-app"
	RouteNuevaAppGuideV0                     = "/nueva-app/guia"
	RouteOpsDashboardV0                      = "/ops"
	RouteOpsKanbanV0                         = "/ops/kanban"
	RouteAutoprogrammingPageV0               = "/autoprogramming"
	RouteAppChangePageV0                     = "/app-change"
	RouteDirectorStatsPageV0                 = "/director-stats"
	RouteRunControlPageV0                    = "/run-control"
	RouteRunQueuePageV0                      = "/run-queue"
	RouteAppSpecV0                           = "/api/v0/apps/spec"
	RouteAppDirectorV0                       = "/api/v0/apps/director"
	RouteAppDirectorPreviewV0                = "/api/v0/apps/director/preview"
	RouteAppDirectorGoalObserveV0            = "/api/v0/apps/director/goal/observe"
	RouteAppIntakeGuidedTurnV0               = "/api/v0/apps/intake/guided-turn"
	RouteAppIntakeWizardBotV0                = "/api/v0/apps/intake/wizard-bot"
	RouteAppChangeV0                         = "/api/v0/apps/"
	RouteDirectorStatsV0                     = "/api/v0/director/stats"
	RouteRunControlV0                        = "/api/v0/runs/control"
	RouteRuntimeModelsV0                     = "/api/v0/runtime/models"
	RouteRunQueuePriorityV0                  = "/api/v0/runs/queue/priority"
	RouteQueueGlobalStatusV0                 = "/api/v0/queue/global-status"
	RouteRunSupervisorV0                     = "/api/v0/runs/supervise"
	RouteOpsAgentRuntimeDetailV0             = "/api/v0/ops/agent-runtime-detail"
	RouteOperationalStatusV0                 = "/api/v0/operational-status/query"
	RouteFunctionContractListV0              = "/api/v0/core/function-contracts/list"
	RouteFunctionContractViewV0              = "/api/v0/core/function-contracts/view"
	RouteServerShutdownV0                    = "/api/v0/server/shutdown"
	RouteHumanDirectorWorkReviewPlanV0       = "/api/v0/director/human-work/review-plan"
	RouteAutoprogrammingValidateRequestV0    = "/api/v0/autoprogramming/validate-request"
	RouteAutoprogrammingSelfImprovementV0    = "/api/v0/autoprogramming/self-improvement"
	RouteAutoprogrammingPrepareRunV0         = "/api/v0/autoprogramming/prepare-run"
	RouteAutoprogrammingObserveGoalV0        = "/api/v0/autoprogramming/goal/observe"
	RouteAutoprogrammingObserveActiveGoalsV0 = "/api/v0/autoprogramming/goals/observe-active"
	RouteAutoprogrammingStatusV0             = "/api/v0/autoprogramming/status"
	RouteAutoprogrammingSuperviseV0          = "/api/v0/autoprogramming/supervise"
	RouteGovernanceCatalogQueryV0            = "/api/v0/governance/catalog/query"
	RouteDomainWorkV0                        = "/api/v0/domain-work"
	RouteDomainWorkStatusV0                  = "/api/v0/domain-work/status"
	RouteExternalWorkDryRunV0                = "/api/v0/external-work/dry-run"
	RouteExternalWorkObserveV0               = "/api/v0/external-work/observe"
	RouteExternalWorkRunV0                   = "/api/v0/external-work/run"
	RouteCodebaseQueryV0                     = "/api/v0/codebase/query"
	RouteCodebaseStatusV0                    = "/api/v0/codebase/status"
	RouteOperatorTelegramUpdateV0            = "/api/v0/operator/telegram/update"
)

type RouteHandlersV0 struct {
	Home                              http.Handler
	NuevaApp                          http.Handler
	NuevaAppGuide                     http.Handler
	OpsDashboard                      http.Handler
	OpsKanban                         http.Handler
	AutoprogrammingPage               http.Handler
	AppChangePage                     http.Handler
	DirectorStatsPage                 http.Handler
	RunControlPage                    http.Handler
	RunQueuePage                      http.Handler
	AppSpec                           http.Handler
	AppDirector                       http.Handler
	AppDirectorPreview                http.Handler
	AppDirectorGoalObserve            http.Handler
	AppIntakeGuidedTurn               http.Handler
	AppIntakeWizardBot                http.Handler
	AppChange                         http.Handler
	DirectorStats                     http.Handler
	RunControl                        http.Handler
	RuntimeModels                     http.Handler
	RunQueuePriority                  http.Handler
	QueueGlobalStatus                 http.Handler
	RunSupervisor                     http.Handler
	OpsAgentRuntimeDetail             http.Handler
	OperationalStatus                 http.Handler
	FunctionContractList              http.Handler
	FunctionContractView              http.Handler
	ServerShutdown                    http.Handler
	HumanDirectorWorkReviewPlan       http.Handler
	AutoprogrammingValidateRequest    http.Handler
	AutoprogrammingSelfImprovement    http.Handler
	AutoprogrammingPrepareRun         http.Handler
	AutoprogrammingObserveGoal        http.Handler
	AutoprogrammingObserveActiveGoals http.Handler
	AutoprogrammingStatus             http.Handler
	AutoprogrammingSupervise          http.Handler
	GovernanceCatalogQuery            http.Handler
	DomainWork                        http.Handler
	DomainWorkStatus                  http.Handler
	ExternalWorkDryRun                http.Handler
	ExternalWorkObserve               http.Handler
	ExternalWorkRun                   http.Handler
	CodebaseQuery                     http.Handler
	CodebaseStatus                    http.Handler
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
		{ref: RouteRefHomeV0, route: RouteHomeV0, handler: handlers.Home},
		{ref: RouteRefNuevaAppV0, route: RouteNuevaAppV0, handler: handlers.NuevaApp},
		{ref: RouteRefNuevaAppGuideV0, route: RouteNuevaAppGuideV0, handler: handlers.NuevaAppGuide},
		{ref: RouteRefOpsDashboardV0, route: RouteOpsDashboardV0, handler: handlers.OpsDashboard},
		{ref: RouteRefOpsKanbanV0, route: RouteOpsKanbanV0, handler: handlers.OpsKanban},
		{ref: RouteRefAutoprogrammingPageV0, route: RouteAutoprogrammingPageV0, handler: handlers.AutoprogrammingPage},
		{ref: RouteRefAppChangePageV0, route: RouteAppChangePageV0, handler: handlers.AppChangePage},
		{ref: RouteRefDirectorStatsPageV0, route: RouteDirectorStatsPageV0, handler: handlers.DirectorStatsPage},
		{ref: RouteRefRunControlPageV0, route: RouteRunControlPageV0, handler: handlers.RunControlPage},
		{ref: RouteRefRunQueuePageV0, route: RouteRunQueuePageV0, handler: handlers.RunQueuePage},
		{ref: RouteRefAppSpecV0, route: RouteAppSpecV0, handler: handlers.AppSpec},
		{ref: RouteRefAppDirectorV0, route: RouteAppDirectorV0, handler: handlers.AppDirector},
		{ref: RouteRefAppDirectorPreviewV0, route: RouteAppDirectorPreviewV0, handler: handlers.AppDirectorPreview},
		{ref: RouteRefAppDirectorGoalObserveV0, route: RouteAppDirectorGoalObserveV0, handler: handlers.AppDirectorGoalObserve},
		{ref: RouteRefAppIntakeGuidedTurnV0, route: RouteAppIntakeGuidedTurnV0, handler: handlers.AppIntakeGuidedTurn},
		{ref: RouteRefAppIntakeWizardBotV0, route: RouteAppIntakeWizardBotV0, handler: handlers.AppIntakeWizardBot},
		{ref: RouteRefAppChangeV0, route: RouteAppChangeV0, handler: handlers.AppChange},
		{ref: RouteRefDirectorStatsV0, route: RouteDirectorStatsV0, handler: handlers.DirectorStats},
		{ref: RouteRefRunControlV0, route: RouteRunControlV0, handler: handlers.RunControl},
		{ref: RouteRefRuntimeModelsV0, route: RouteRuntimeModelsV0, handler: handlers.RuntimeModels},
		{ref: RouteRefRunQueuePriorityV0, route: RouteRunQueuePriorityV0, handler: handlers.RunQueuePriority},
		{ref: RouteRefQueueGlobalStatusV0, route: RouteQueueGlobalStatusV0, handler: handlers.QueueGlobalStatus},
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
		{ref: RouteRefAutoprogrammingObserveGoalV0, route: RouteAutoprogrammingObserveGoalV0, handler: handlers.AutoprogrammingObserveGoal},
		{ref: RouteRefAutoprogrammingObserveActiveGoalsV0, route: RouteAutoprogrammingObserveActiveGoalsV0, handler: handlers.AutoprogrammingObserveActiveGoals},
		{ref: RouteRefAutoprogrammingStatusV0, route: RouteAutoprogrammingStatusV0, handler: handlers.AutoprogrammingStatus},
		{ref: RouteRefAutoprogrammingSuperviseV0, route: RouteAutoprogrammingSuperviseV0, handler: handlers.AutoprogrammingSupervise},
		{ref: RouteRefGovernanceCatalogQueryV0, route: RouteGovernanceCatalogQueryV0, handler: handlers.GovernanceCatalogQuery},
		{ref: RouteRefDomainWorkStatusV0, route: RouteDomainWorkStatusV0, handler: handlers.DomainWorkStatus},
		{ref: RouteRefDomainWorkV0, route: RouteDomainWorkV0, handler: handlers.DomainWork},
		{ref: RouteRefExternalWorkDryRunV0, route: RouteExternalWorkDryRunV0, handler: handlers.ExternalWorkDryRun},
		{ref: RouteRefExternalWorkObserveV0, route: RouteExternalWorkObserveV0, handler: handlers.ExternalWorkObserve},
		{ref: RouteRefExternalWorkRunV0, route: RouteExternalWorkRunV0, handler: handlers.ExternalWorkRun},
		{ref: RouteRefCodebaseQueryV0, route: RouteCodebaseQueryV0, handler: handlers.CodebaseQuery},
		{ref: RouteRefCodebaseStatusV0, route: RouteCodebaseStatusV0, handler: handlers.CodebaseStatus},
	}
}
