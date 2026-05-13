package orquestaappgateway

import (
	"net/http"
	"time"

	orquestafactory "orquesta/modulos/orquesta-factory"
	orquestamcp "orquesta/modulos/orquesta-mcp"
	orquestaweb "orquesta/modulos/orquesta-web"
)

const InternalBaseURLV0 = "http://orquesta.internal"

type ConfigV0 struct {
	Clock            orquestafactory.AppSpecHTTPClockV0
	ArrancarDirector orquestamcp.MCPTransportArrancarDirectorAppExecutorV0
	RequestAppChange orquestamcp.MCPTransportRequestAppChangeExecutorV0
	DirectorLimits   orquestaweb.WebArrancarDirectorAppLimitsV0
	DirectorStats    orquestamcp.MCPTransportDirectorStatsExecutorV0
	RunControl       orquestamcp.MCPTransportRunControlExecutorV0
	RunQueuePriority orquestamcp.MCPTransportRunQueuePriorityExecutorV0
	ServerShutdown   orquestamcp.MCPTransportServerShutdownExecutorV0
	DomainWork       orquestamcp.MCPDomainWorkExecutorPortV0
	HTTPClient       *http.Client
	Timeout          time.Duration
}
