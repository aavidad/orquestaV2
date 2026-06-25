package orquestaappgateway

import (
	"net/http"
	"time"

	orquestacore "orquesta/modulos/orquesta-core"
	orquestafactoryhttp "orquesta/modulos/orquesta-factory-http"
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
	Clock                     orquestafactoryhttp.AppSpecHTTPClockV0
	ArrancarDirector          orquestamcp.MCPTransportArrancarDirectorAppExecutorV0
	ObserveDirectorGoal       orquestamcp.MCPTransportObserveAppDirectorGoalExecutorV0
	RequestAppChange          orquestamcp.MCPTransportRequestAppChangeExecutorV0
	DirectorLimits            orquestaweb.WebArrancarDirectorAppLimitsV0
	DirectorStats             orquestamcp.MCPTransportDirectorStatsExecutorV0
	RunControl                orquestamcp.MCPTransportRunControlExecutorV0
	RuntimeModels             orquestaruntime.RuntimeModelManagerPortV0
	RunQueuePriority          orquestamcp.MCPTransportRunQueuePriorityExecutorV0
	RunSupervisor             orquestamcp.MCPTransportRunSupervisorExecutorV0
	OpsAgentRuntimeDetail     http.Handler
	OperationalStatus         orquestaobservability.OperationalStatusQuerySourceV0
	FunctionContracts         orquestacore.FunctionContractReadIndexPortV0
	AutoprogrammingPrepareRun orquestamcp.MCPTransportAutoprogrammingPrepareRunExecutorV0
	GovernanceCatalog         orquestagovernance.GovernanceCatalogProviderV0
	OperatorQuery             operator.OperatorMCPDirectedQueryPortV0
	ServerShutdown            orquestamcp.MCPTransportServerShutdownExecutorV0
	DomainWork                orquestamcp.MCPDomainWorkExecutorPortV0
	ExternalWorkRun           orquestamcp.MCPTransportExternalWorkRunExecutorV0
	AppVCS                    orquestamcp.MCPAppVCSExecutorPortV0
	WebHTMLRenderObserver     orquestaobservability.WebHTMLRenderObserverV0
	BrowserMutationIntent     orquestahttpgateway.BrowserMutationIntentGuardConfigV0
	HTTPClient                *http.Client
	Timeout                   time.Duration
}
