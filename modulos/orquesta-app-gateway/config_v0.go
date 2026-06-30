package orquestaappgateway

import (
	"net/http"
	"time"

	orquestacore "orquesta/modulos/orquesta-core"
	orquestaexternalworkrun "orquesta/modulos/orquesta-external-work-run"
	orquestafactoryhttp "orquesta/modulos/orquesta-factory-http"
	orquestagoal "orquesta/modulos/orquesta-goal"
	orquestagovernance "orquesta/modulos/orquesta-governance"
	orquestahttpgateway "orquesta/modulos/orquesta-http-gateway"
	orquestamcp "orquesta/modulos/orquesta-mcp"
	orquestaobservability "orquesta/modulos/orquesta-observability"
	operator "orquesta/modulos/orquesta-operator-mcp"
	orquestaruntime "orquesta/modulos/orquesta-runtime"
	orquestaweb "orquesta/modulos/orquesta-web"
)

const InternalBaseURLV0 = "http://orquesta.internal"

type ConfigV0 struct {
	Clock                                       orquestafactoryhttp.AppSpecHTTPClockV0
	ArrancarDirector                            orquestamcp.MCPTransportArrancarDirectorAppExecutorV0
	PreviewDirector                             orquestamcp.MCPTransportPreviewDirectorAppExecutorV0
	ObserveDirectorGoal                         orquestamcp.MCPTransportObserveAppDirectorGoalExecutorV0
	RequestAppChange                            orquestamcp.MCPTransportRequestAppChangeExecutorV0
	DirectorLimits                              orquestaweb.WebArrancarDirectorAppLimitsV0
	DirectorStats                               orquestamcp.MCPTransportDirectorStatsExecutorV0
	RunControl                                  orquestamcp.MCPTransportRunControlExecutorV0
	RuntimeModels                               orquestaruntime.RuntimeModelManagerPortV0
	AppIntakeAssistant                          orquestaweb.WebNuevaAppIntakeAssistantPortV0
	RunQueuePriority                            orquestamcp.MCPTransportRunQueuePriorityExecutorV0
	RunSupervisor                               orquestamcp.MCPTransportRunSupervisorExecutorV0
	OpsAgentRuntimeDetail                       http.Handler
	OperationalStatus                           orquestaobservability.OperationalStatusQuerySourceV0
	FunctionContracts                           orquestacore.FunctionContractReadIndexPortV0
	AutoprogrammingPrepareRun                   orquestamcp.MCPTransportAutoprogrammingPrepareRunExecutorV0
	AutoprogrammingObserveGoal                  orquestamcp.MCPTransportAutoprogrammingObserveGoalExecutorV0
	AutoprogrammingObserveActiveGoals           orquestamcp.MCPTransportAutoprogrammingObserveActiveGoalsExecutorV0
	AutoprogrammingGoalStates                   orquestagoal.GoalWorkStateStorePortV0
	AllowLegacyAutoprogrammingSupervisorActions bool
	GovernanceCatalog                           orquestagovernance.GovernanceCatalogProviderV0
	OperatorQuery                               operator.OperatorMCPDirectedQueryPortV0
	ServerShutdown                              orquestamcp.MCPTransportServerShutdownExecutorV0
	DomainWork                                  orquestamcp.MCPDomainWorkExecutorPortV0
	DomainWorkStatus                            orquestamcp.MCPTransportAutoprogrammingStatusExecutorV0
	ExternalWorkDryRun                          orquestamcp.MCPTransportExternalWorkDryRunExecutorV0
	ExternalWorkDryRunConfig                    orquestaexternalworkrun.StartExternalWorkRunConfigV0
	ExternalWorkRun                             orquestamcp.MCPTransportExternalWorkRunExecutorV0
	CodebaseQuery                               orquestamcp.MCPTransportCodebaseQueryExecutorV0
	CodebaseStatus                              orquestamcp.MCPTransportCodebaseStatusExecutorV0
	AppVCS                                      orquestamcp.MCPAppVCSExecutorPortV0
	WebHTMLRenderObserver                       orquestaobservability.WebHTMLRenderObserverV0
	BrowserMutationIntent                       orquestahttpgateway.BrowserMutationIntentGuardConfigV0
	HTTPClient                                  *http.Client
	Timeout                                     time.Duration
}
