package orquestaappgateway

import (
	"net/http"
	"time"

	orquestafactoryhttp "orquesta/modulos/orquesta-factory-http"
	orquestamcp "orquesta/modulos/orquesta-mcp"
	operator "orquesta/modulos/orquesta-operator-mcp"
	orquestaweb "orquesta/modulos/orquesta-web"
)

const InternalBaseURLV0 = "http://orquesta.internal"

type ConfigV0 struct {
	Clock                     orquestafactoryhttp.AppSpecHTTPClockV0
	ArrancarDirector          orquestamcp.MCPTransportArrancarDirectorAppExecutorV0
	RequestAppChange          orquestamcp.MCPTransportRequestAppChangeExecutorV0
	DirectorLimits            orquestaweb.WebArrancarDirectorAppLimitsV0
	DirectorStats             orquestamcp.MCPTransportDirectorStatsExecutorV0
	RunControl                orquestamcp.MCPTransportRunControlExecutorV0
	RunQueuePriority          orquestamcp.MCPTransportRunQueuePriorityExecutorV0
	RunSupervisor             orquestamcp.MCPTransportRunSupervisorExecutorV0
	AutoprogrammingPrepareRun orquestamcp.MCPTransportAutoprogrammingPrepareRunExecutorV0
	OperatorQuery             operator.OperatorMCPDirectedQueryPortV0
	ServerShutdown            orquestamcp.MCPTransportServerShutdownExecutorV0
	DomainWork                orquestamcp.MCPDomainWorkExecutorPortV0
	ExternalWorkRun           orquestamcp.MCPTransportExternalWorkRunExecutorV0
	HTTPClient                *http.Client
	Timeout                   time.Duration
}
